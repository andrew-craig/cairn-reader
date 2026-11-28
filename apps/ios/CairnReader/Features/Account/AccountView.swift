import SwiftUI

/// Saving a device account or changing an email account's password, and logging out.
struct AccountView: View {
    let sessionModel: SessionModel
    @State private var model: AccountModel
    @State private var isConfirmingLogout = false

    init(sessionModel: SessionModel) {
        self.sessionModel = sessionModel
        _model = State(initialValue: AccountModel(auth: sessionModel.auth) { await sessionModel.login() })
    }

    var body: some View {
        Form {
            Section {
                LabeledContent("Account", value: sessionModel.user?.email ?? "Anonymous")
            }

            if sessionModel.user?.email == nil {
                Section("Save account") {
                    TextField("Email", text: $model.email)
                        .textContentType(.username)
                        .keyboardType(.emailAddress)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                    SecureField("Password", text: $model.password)
                        .textContentType(.newPassword)
                    Button("Save") {
                        Task { await model.saveAccount() }
                    }
                }
            } else {
                Section("Change password") {
                    SecureField("Current password", text: $model.currentPassword)
                        .textContentType(.password)
                    SecureField("New password", text: $model.newPassword)
                        .textContentType(.newPassword)
                    SecureField("Confirm new password", text: $model.confirmPassword)
                        .textContentType(.newPassword)
                    Button("Change password") {
                        Task { await model.changePassword() }
                    }
                }
            }

            Section {
                Button("Log out", role: .destructive) {
                    isConfirmingLogout = true
                }
                .alert("Log out?", isPresented: $isConfirmingLogout) {
                    Button("Cancel", role: .cancel) {}
                    Button("Log out", role: .destructive) {
                        Task { await sessionModel.logout() }
                    }
                }
            }
        }
        .disabled(model.isWorking)
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("Account")
        .toolbarTitleDisplayMode(.inline)
        .alert(
            model.alert?.title ?? "",
            isPresented: Binding(get: { model.alert != nil }, set: { if !$0 { model.alert = nil } }),
            presenting: model.alert
        ) { _ in
            Button("OK") {}
        } message: { alert in
            Text(alert.message)
        }
    }
}
