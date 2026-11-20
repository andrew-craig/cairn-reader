import SwiftUI

/// The sheet for reporting a badly extracted article: the reasons, an optional comment, Cancel and
/// Send. Each presentation builds a new view, so a new model prefilled from the earlier report.
struct ReportIssueView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var model: ReportIssueModel
    /// Runs after the report was accepted, just before the sheet closes.
    let onSent: () -> Void

    init(read: ReadService, contentID: String, onSent: @escaping () -> Void) {
        _model = State(initialValue: ReportIssueModel(read: read, contentID: contentID))
        self.onSent = onSent
    }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    ForEach(ReportIssueModel.reasons, id: \.category) { reason in
                        let isSelected = model.selected.contains(reason.category)
                        Button {
                            model.toggle(reason.category)
                        } label: {
                            HStack {
                                Text(reason.label).foregroundStyle(Color(.text))
                                Spacer()
                                if isSelected {
                                    Image(systemName: "checkmark").fontWeight(.semibold)
                                }
                            }
                        }
                        .accessibilityAddTraits(isSelected ? .isSelected : [])
                    }
                }
                .disabled(model.isLoading || model.isSending)

                Section {
                    TextField("Details (optional)", text: $model.comment, axis: .vertical)
                        .lineLimit(4...)
                        .disabled(model.isSending)
                } footer: {
                    Text("\(model.comment.count)/\(ReportIssueModel.commentLimit)")
                        .frame(maxWidth: .infinity, alignment: .trailing)
                }

                if let error = model.error {
                    Section {
                        Text(error).foregroundStyle(Color(.error))
                    }
                }
            }
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle("Report an issue")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                        .disabled(model.isSending)
                }
                ToolbarItem(placement: .confirmationAction) {
                    if model.isSending || model.isLoading {
                        ProgressView()
                    } else {
                        Button("Send") {
                            Task {
                                if await model.send() {
                                    onSent()
                                    dismiss()
                                }
                            }
                        }
                        .disabled(!model.canSend)
                    }
                }
            }
        }
        .interactiveDismissDisabled(model.isSending)
        .task { await model.load() }
    }
}
