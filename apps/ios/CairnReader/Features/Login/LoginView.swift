import SwiftUI

struct LoginView: View {
    @Bindable var model: LoginModel

    var body: some View {
        Form {
            Section {
                VStack(spacing: 8) {
                    Text("Cairn")
                        .font(Typography.wordmark)
                        .foregroundStyle(Color(.brand))
                    Text("Read and discover\nwhat you love")
                        .font(Typography.title)
                        .foregroundStyle(Color(.textSecondary))
                        .multilineTextAlignment(.center)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 24)
            }
            .listRowBackground(Color.clear)

            Section {
                Button("Continue with this device") {
                    Task { await model.continueWithDevice() }
                }
            }

            Section {
                TextField("Email", text: $model.email)
                    .textContentType(.username)
                    .keyboardType(.emailAddress)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                SecureField("Password", text: $model.password)
                    .textContentType(.password)
                Button("Sign in") {
                    Task { await model.signIn() }
                }
                Button("Create account") {
                    Task { await model.createAccount() }
                }
            } footer: {
                if let message = model.errorMessage {
                    Text(message).foregroundStyle(Color(.error))
                }
            }

            Section {
                DisclosureGroup("Advanced") {
                    TextField("https://your-server.example.com", text: $model.serverURL)
                        .keyboardType(.URL)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                    Text("Point the app at a self-hosted Cairn server. Leave empty for the default.")
                        .font(Typography.caption)
                        .foregroundStyle(Color(.textSecondary))
                    Button("Save server") {
                        Task { await model.saveServer() }
                    }
                }
            }
        }
        .disabled(model.isWorking)
        .scrollDismissesKeyboard(.interactively)
    }
}
