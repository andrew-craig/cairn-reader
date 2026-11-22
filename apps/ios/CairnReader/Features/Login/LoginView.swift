import SwiftUI

struct LoginView: View {
    @Bindable var model: LoginModel
    @State private var showsServerSheet = false

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
        }
        .scrollDismissesKeyboard(.interactively)
        .safeAreaInset(edge: .bottom) {
            HStack(spacing: 4) {
                Text("Powered by")
                Button("cairnreader.app") { showsServerSheet = true }
                    .buttonStyle(.plain)
                    .underline()
                    .accessibilityHint("Choose the server")
            }
            .font(Typography.subheadline)
            .foregroundStyle(Color(.textSecondary))
            .padding(.vertical, 8)
        }
        .disabled(model.isWorking)
        .sheet(isPresented: $showsServerSheet) {
            ServerSheet(model: model)
        }
    }
}
