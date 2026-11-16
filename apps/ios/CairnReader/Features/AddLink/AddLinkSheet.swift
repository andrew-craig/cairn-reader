import SwiftUI

/// The add-link sheet: a URL to save to Reads or a feed to subscribe to. Present it with
/// `.sheet { AddLinkSheet(read:mode:onAdded:) }`; `onAdded` runs when something was added, before
/// the success alert, so the presenting list can refresh. Online only.
struct AddLinkSheet: View {
    private let onAdded: () -> Void

    @State private var model: AddLinkModel
    @Environment(Connectivity.self) private var connectivity
    @Environment(\.dismiss) private var dismiss

    init(read: ReadService, mode: AddLinkModel.Mode, onAdded: @escaping () -> Void) {
        _model = State(initialValue: AddLinkModel(read: read, mode: mode))
        self.onAdded = onAdded
    }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    HStack {
                        TextField(model.mode == .feed ? "Feed or website URL" : "Add link", text: $model.url)
                            .keyboardType(.URL)
                            .textContentType(.URL)
                            .textInputAutocapitalization(.never)
                            .autocorrectionDisabled()
                            .submitLabel(.done)
                            .onSubmit { Task { await model.submit() } }
                        // The payload can arrive off the main thread, so the closure is not
                        // main-actor isolated and hops there itself.
                        PasteButton(payloadType: String.self) { @Sendable [model] strings in
                            guard let text = strings.first else { return }
                            Task { @MainActor in model.url = text }
                        }
                        .labelStyle(.iconOnly)
                    }
                } footer: {
                    urlFooter
                }
                if model.isFeed {
                    Section {
                        Picker("Where should new items go?", selection: $model.list) {
                            Text("Feed").tag(ContentList.feed)
                            Text("Reads").tag(ContentList.reads)
                        }
                        .pickerStyle(.inline)
                        .labelsHidden()
                    } header: {
                        Text("Where should new items go?")
                    } footer: {
                        Text(
                            """
                            Feed: Skim in an endless scroll. Items are deleted after 30 days unless you favorite them.
                            Reads: Read or triage each item. Items are kept and available offline.
                            You can change this later on the RSS page.
                            """)
                    }
                }
                if model.showsFindFeed {
                    Section {
                        Button {
                            Task { await model.findFeed() }
                        } label: {
                            HStack {
                                Text("Find feed")
                                Spacer()
                                if model.isDiscovering { ProgressView() }
                            }
                        }
                        .disabled(!model.canFindFeed)
                    }
                }
            }
            .disabled(!connectivity.isOnline || model.isAdding)
            .navigationTitle(model.mode == .feed ? "Add Feed" : "Add Link")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(role: .cancel) { dismiss() }
                        .disabled(!model.canDismiss)
                }
                if model.showsAdd {
                    ToolbarItem(placement: .confirmationAction) {
                        Button(model.addTitle) { Task { await model.addURL() } }
                            .disabled(!model.canAdd || !connectivity.isOnline)
                    }
                }
            }
            .confirmationDialog(
                "Multiple feeds found",
                isPresented: Binding(get: { model.feedChoices != nil }, set: { if !$0 { model.feedChoices = nil } }),
                titleVisibility: .visible, presenting: model.feedChoices
            ) { feeds in
                ForEach(feeds, id: \.url) { feed in
                    Button(feed.title.isEmpty ? feed.url : feed.title) { model.url = feed.url }
                }
            } message: { _ in
                Text("Select a feed to use:")
            }
            .alert("Success", isPresented: .constant(model.successMessage != nil)) {
                Button("OK") { dismiss() }
            } message: {
                Text(model.successMessage ?? "")
            }
        }
        .interactiveDismissDisabled(!model.canDismiss)
        .onChange(of: model.successMessage) { _, message in
            if message != nil { onAdded() }
        }
        .onDisappear { model.close() }
    }

    @ViewBuilder
    private var urlFooter: some View {
        if !connectivity.isOnline {
            Text("You're offline. Adding a link needs a connection.")
        } else if let error = model.error {
            Text(error).foregroundStyle(Color(.error))
        } else if let summary = model.summary {
            Text(summary)
        }
    }
}
