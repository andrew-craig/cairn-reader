import Foundation
import Testing
@testable import CairnReader

// The validation rules and messages of apps/mobile/src/screens/AccountScreen.tsx (which has no
// test of its own).

@MainActor
private final class ChangeCounter {
    var count = 0
}

@MainActor
private struct Harness {
    let model: AccountModel
    let server: StubServer
    let changes: ChangeCounter

    init(_ server: StubServer) {
        self.server = server
        let config = ServerConfig(defaults: UserDefaults(suiteName: UUID().uuidString)!)
        let auth = AuthService(
            client: HTTPClient(server: config, session: server.session()),
            storage: InMemorySessionStorage(AuthFixtures.valid), deviceID: { "device-1" }, sleep: { _ in })
        let changes = ChangeCounter()
        self.changes = changes
        model = AccountModel(auth: auth) { changes.count += 1 }
    }
}

@MainActor
struct AccountModelTests {
    // MARK: Save account

    @Test(arguments: [
        ("", "password1", "Please enter an email address"),
        ("   ", "password1", "Please enter an email address"),
        ("me@example.com", "", "Please enter a password"),
        ("me@example.com", "        ", "Please enter a password"),
        ("me@example.com", "short", "Password must be at least 8 characters"),
    ])
    func saveValidation(email: String, password: String, message: String) async {
        let h = Harness(StubServer(replies: []))
        h.model.email = email
        h.model.password = password
        await h.model.saveAccount()
        #expect(h.model.alert == .error(message))
        #expect(h.server.requests.isEmpty)
    }

    @Test func saveUpgradesTheAccount() async {
        let h = Harness(StubServer(replies: [.json(200, AuthFixtures.userBody(email: "me@example.com"))]))
        h.model.email = " me@example.com "
        h.model.password = "password1"

        await h.model.saveAccount()

        #expect(h.server.paths == ["/api/v1/user/user-1/upgrade"])
        #expect(h.server.requests.first?.httpMethod == "POST")
        #expect(h.server.requests.first?.jsonBody["email"] as? String == "me@example.com")
        #expect(h.server.requests.first?.jsonBody["password"] as? String == "password1")
        #expect(h.model.alert == .success("Your account has been saved"))
        #expect(h.changes.count == 1)
        #expect(h.model.email.isEmpty && h.model.password.isEmpty)
        #expect(!h.model.isWorking)
    }

    @Test func saveFailureShowsTheServerMessageAndKeepsFields() async {
        let h = Harness(StubServer(replies: [.json(409, AuthFixtures.errorBody("email already registered"))]))
        h.model.email = "me@example.com"
        h.model.password = "password1"

        await h.model.saveAccount()

        #expect(h.model.alert == .error("email already registered"))
        #expect(h.changes.count == 0)
        #expect(h.model.email == "me@example.com")
        #expect(h.model.password == "password1")
    }

    @Test func saveWhileOfflineSaysTheServerIsUnreachable() async {
        let h = Harness(StubServer(replies: [.failure(.notConnectedToInternet)]))
        h.model.email = "me@example.com"
        h.model.password = "password1"
        await h.model.saveAccount()
        #expect(h.model.alert == .error("Unable to reach the server. Please try again later."))
    }

    // MARK: Change password

    @Test(arguments: [
        ("", "newpassword", "newpassword", "Please enter your current password"),
        ("  ", "newpassword", "newpassword", "Please enter your current password"),
        ("oldpassword", "", "", "Please enter a new password"),
        ("oldpassword", "short", "short", "New password must be at least 8 characters"),
        ("oldpassword", "newpassword", "newpassword2", "Passwords do not match"),
    ])
    func changePasswordValidation(current: String, new: String, confirm: String, message: String) async {
        let h = Harness(StubServer(replies: []))
        h.model.currentPassword = current
        h.model.newPassword = new
        h.model.confirmPassword = confirm
        await h.model.changePassword()
        #expect(h.model.alert == .error(message))
        #expect(h.server.requests.isEmpty)
    }

    @Test func changePasswordSendsBothPasswords() async {
        let h = Harness(StubServer(replies: [.json(200, "{}")]))
        h.model.currentPassword = "oldpassword"
        h.model.newPassword = "newpassword"
        h.model.confirmPassword = "newpassword"

        await h.model.changePassword()

        #expect(h.server.paths == ["/api/v1/user/user-1/password"])
        #expect(h.server.requests.first?.httpMethod == "PUT")
        #expect(h.server.requests.first?.jsonBody["current_password"] as? String == "oldpassword")
        #expect(h.server.requests.first?.jsonBody["new_password"] as? String == "newpassword")
        #expect(h.model.alert == .success("Your password has been changed"))
        #expect(h.model.currentPassword.isEmpty && h.model.newPassword.isEmpty && h.model.confirmPassword.isEmpty)
        #expect(h.changes.count == 0)
    }

    @Test func changePasswordFailureShowsTheServerMessageAndKeepsFields() async {
        let h = Harness(StubServer(replies: [.json(400, AuthFixtures.errorBody("password too weak"))]))
        h.model.currentPassword = "oldpassword"
        h.model.newPassword = "newpassword"
        h.model.confirmPassword = "newpassword"

        await h.model.changePassword()

        #expect(h.model.alert == .error("password too weak"))
        #expect(h.model.newPassword == "newpassword")
    }
}
