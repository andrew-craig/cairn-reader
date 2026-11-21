import SwiftUI
import UIKit

/// RSS or Newsletters, pushed from You: each source with a Feed/Reads menu for where its new
/// items go. RSS feeds can be unsubscribed (swipe, then confirm) and added through the add-link
/// sheet in feed mode; Newsletters shows the ingest address from its add button.
struct SubscriptionListView: View {
    private let session: UserSession
    @State private var model: SubscriptionListModel
    @State private var isAddingFeed = false
    @State private var isShowingAddress = false
    @State private var unsubscribing: UnifiedSubscription?

    init(session: UserSession, kind: SubscriptionListModel.Kind) {
        self.session = session
        _model = State(initialValue: SubscriptionListModel(read: session.read, kind: kind))
    }

    var body: some View {
        List {
            Section {
                ForEach(model.subscriptions) { subscription in
                    row(subscription)
                        .swipeActions {
                            if SubscriptionListModel.canUnsubscribe(subscription) {
                                Button("Unsubscribe", systemImage: "trash", role: .destructive) {
                                    unsubscribing = subscription
                                }
                            }
                        }
                }
            } footer: {
                if model.isRoutable {
                    Text(
                        "Choose where each source's new items land. This applies to new items only; items already delivered stay where they are."
                    )
                }
            }
        }
        .overlay { emptyState }
        .refreshable { await model.load() }
        .navigationTitle(model.kind == .rss ? "RSS" : "Newsletters")
        .toolbarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button(model.kind == .rss ? "Add feed" : "Add newsletter", systemImage: "plus") {
                    switch model.kind {
                    case .rss: isAddingFeed = true
                    case .newsletters: isShowingAddress = true
                    }
                }
                .popover(isPresented: $isShowingAddress) {
                    NewsletterAddressView(model: model)
                        .presentationCompactAdaptation(.popover)
                }
            }
        }
        .sheet(isPresented: $isAddingFeed) {
            AddLinkSheet(read: session.read, mode: .feed) { Task { await model.load() } }
        }
        .confirmationDialog(
            "Unsubscribe",
            isPresented: Binding(get: { unsubscribing != nil }, set: { if !$0 { unsubscribing = nil } }),
            titleVisibility: .visible, presenting: unsubscribing
        ) { subscription in
            Button("Unsubscribe", role: .destructive) {
                Task { await model.unsubscribe(subscription) }
            }
            Button("Cancel", role: .cancel) {}
        } message: { subscription in
            Text(
                "Stop receiving new articles from \"\(subscription.title)\"? Articles already in your reading list will be kept."
            )
        }
        .alert(
            "Error",
            isPresented: Binding(get: { model.errorMessage != nil }, set: { if !$0 { model.errorMessage = nil } })
        ) {
            Button("OK") {}
        } message: {
            Text(model.errorMessage ?? "")
        }
        // Not `.task`: popping back mid-load would cancel it and read as a failure.
        .onAppear {
            Task { await model.load() }
            if model.kind == .newsletters {
                Task { await model.loadEmailAddress() }
            }
        }
    }

    @ViewBuilder
    private func row(_ subscription: UnifiedSubscription) -> some View {
        let label = VStack(alignment: .leading) {
            Text(subscription.title)
                .font(Typography.bodyBold)
                .lineLimit(1)
            if let subtitle = model.subtitle(for: subscription) {
                Text(subtitle)
                    .font(Typography.subheadline)
                    .foregroundStyle(Color(.textSecondary))
                    .lineLimit(1)
            }
        }
        if SubscriptionListModel.route(for: subscription) != nil {
            Picker(
                selection: Binding(
                    get: { subscription.list },
                    set: { list in Task { await model.setList(list, for: subscription) } })
            ) {
                Text("Feed").tag(ContentList.feed)
                Text("Reads").tag(ContentList.reads)
            } label: {
                label
            }
            .pickerStyle(.menu)
        } else {
            label
        }
    }

    @ViewBuilder
    private var emptyState: some View {
        if model.subscriptions.isEmpty {
            if model.isLoading {
                ProgressView()
            } else {
                ContentUnavailableView(
                    model.kind == .rss ? "No RSS feeds yet" : "No newsletters yet",
                    systemImage: model.kind == .rss ? "dot.radiowaves.up.forward" : "envelope")
            }
        }
    }
}

/// The newsletter ingest address with a copy button, shown from the Newsletters add button.
private struct NewsletterAddressView: View {
    let model: SubscriptionListModel
    @State private var copied = false

    var body: some View {
        VStack(spacing: 8) {
            Text("Send or forward emails to")
            HStack {
                Text(model.emailAddressError ?? model.emailAddress ?? "Loading…")
                    .font(Typography.bodyBold)
                    .textSelection(.enabled)
                if model.emailAddressError == nil, let address = model.emailAddress {
                    Button(
                        copied ? "Copied" : "Copy address",
                        systemImage: copied ? "checkmark" : "doc.on.doc"
                    ) {
                        UIPasteboard.general.string = address
                        copied = true
                    }
                    .labelStyle(.iconOnly)
                }
            }
            Text("New subscriptions will be automatically added to your reading list.")
                .font(Typography.subheadline)
                .foregroundStyle(Color(.textSecondary))
        }
        .multilineTextAlignment(.center)
        .padding()
        .task(id: copied) {
            guard copied else { return }
            try? await Task.sleep(for: .seconds(2))
            copied = false
        }
    }
}
