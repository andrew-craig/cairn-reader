import SwiftUI

/// The sheet behind the login screen's "Powered by" link: point the app at a self-hosted Cairn
/// server, or reset to the default. The field is a draft, so Cancel leaves the saved server alone.
struct ServerSheet: View {
    @Environment(\.dismiss) private var dismiss
    let model: LoginModel
    @State private var draft: String

    init(model: LoginModel) {
        self.model = model
        _draft = State(initialValue: model.serverURL)
    }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("https://your-server.example.com", text: $draft)
                        .keyboardType(.URL)
                        .textContentType(.URL)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                        .submitLabel(.done)
                        .onSubmit(save)
                } footer: {
                    Text("Point the app at a self-hosted Cairn server. Leave empty for the default.")
                }

                Section {
                    Button("Reset to default") {
                        Task {
                            await model.resetServer()
                            dismiss()
                        }
                    }
                    .disabled(model.isDefaultServer)
                } footer: {
                    Text("Default: \(ServerConfig.defaultURL.absoluteString)")
                }
            }
            .navigationTitle("Server")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save", action: save)
                }
            }
        }
    }

    private func save() {
        Task {
            model.serverURL = draft
            await model.saveServer()
            dismiss()
        }
    }
}
