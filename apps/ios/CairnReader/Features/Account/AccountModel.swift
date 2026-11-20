import Foundation
import Observation

/// The Account form: saving a device account with an email and password, or changing an email
/// account's password. Outcomes are shown as an alert, with the React Native app's wording.
@Observable
final class AccountModel {
    nonisolated struct Alert: Equatable {
        let title: String
        let message: String

        static func error(_ message: String) -> Alert { Alert(title: "Error", message: message) }
        static func success(_ message: String) -> Alert { Alert(title: "Success", message: message) }
    }

    static let minimumPasswordLength = 8

    // Save account (device accounts).
    var email = ""
    var password = ""
    // Change password (email accounts).
    var currentPassword = ""
    var newPassword = ""
    var confirmPassword = ""

    private(set) var isWorking = false
    /// The alert to show; cleared when dismissed.
    var alert: Alert?

    @ObservationIgnored private let auth: AuthService
    @ObservationIgnored private let onAccountChanged: () async -> Void

    /// `onAccountChanged` runs after a successful upgrade, so the session picks up the email.
    init(auth: AuthService, onAccountChanged: @escaping () async -> Void) {
        self.auth = auth
        self.onAccountChanged = onAccountChanged
    }

    func saveAccount() async {
        let email = email.trimmingCharacters(in: .whitespacesAndNewlines)
        if let message = Self.validateSave(email: email, password: password) {
            alert = .error(message)
            return
        }
        await perform {
            _ = try await auth.upgradeAccount(email: email, password: password)
        } success: {
            await onAccountChanged()
            self.email = ""
            password = ""
            return "Your account has been saved"
        }
    }

    func changePassword() async {
        if let message = Self.validateChange(current: currentPassword, new: newPassword, confirm: confirmPassword) {
            alert = .error(message)
            return
        }
        await perform {
            try await auth.changePassword(current: currentPassword, new: newPassword)
        } success: {
            currentPassword = ""
            newPassword = ""
            confirmPassword = ""
            return "Your password has been changed"
        }
    }

    private static func validateSave(email: String, password: String) -> String? {
        if email.isEmpty { return "Please enter an email address" }
        if isBlank(password) { return "Please enter a password" }
        if password.count < minimumPasswordLength { return "Password must be at least 8 characters" }
        return nil
    }

    private static func validateChange(current: String, new: String, confirm: String) -> String? {
        if isBlank(current) { return "Please enter your current password" }
        if isBlank(new) { return "Please enter a new password" }
        if new.count < minimumPasswordLength { return "New password must be at least 8 characters" }
        if new != confirm { return "Passwords do not match" }
        return nil
    }

    private static func isBlank(_ text: String) -> Bool {
        text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    /// Runs `request` once at a time; on success runs `success` and shows its message, on failure
    /// shows the error's message and keeps the fields.
    private func perform(_ request: () async throws -> Void, success: () async -> String) async {
        guard !isWorking else { return }
        isWorking = true
        defer { isWorking = false }
        do {
            try await request()
        } catch {
            alert = .error(error.localizedDescription)
            return
        }
        let message = await success()
        alert = .success(message)
    }
}
