import Foundation
import Observation

/// The Login form's state and actions. Signing in goes through `AuthService`, then
/// `onSignedIn` tells the `SessionModel`.
@Observable
final class LoginModel {
    var email = ""
    var password = ""
    var serverURL: String
    private(set) var isWorking = false
    /// The last failure, shown in the form footer; cleared by the next action.
    private(set) var errorMessage: String?

    @ObservationIgnored private let auth: AuthService
    @ObservationIgnored private let server: ServerConfig
    @ObservationIgnored private let onSignedIn: () async -> Void

    init(auth: AuthService, server: ServerConfig, onSignedIn: @escaping () async -> Void) {
        self.auth = auth
        self.server = server
        self.onSignedIn = onSignedIn
        serverURL = server.url.absoluteString
    }

    /// Signs in with this device's id. The backend answers 401 for a device it has not seen, and
    /// only then is the device registered: any other failure is not a reason to create an account.
    func continueWithDevice() async {
        await perform {
            do {
                return try await auth.loginWithDevice()
            } catch let error as HttpError where error.status == 401 {
                return try await auth.registerWithDevice()
            }
        }
    }

    func signIn() async {
        guard let (email, password) = credentials() else { return }
        await perform { try await auth.loginWithEmail(email: email, password: password) }
    }

    func createAccount() async {
        guard let (email, password) = credentials() else { return }
        await perform { try await auth.registerWithEmail(email: email, password: password) }
    }

    /// Saves the server URL. Tokens belong to the server that issued them, so `AuthService`
    /// clears the session when the server changes. An empty field means the default server.
    func saveServer() async {
        errorMessage = nil
        await auth.setServerURL(serverURL)
        serverURL = server.url.absoluteString
    }

    var isDefaultServer: Bool { server.url == ServerConfig.defaultURL }

    /// Goes back to the default server (clearing the session if that changes the server).
    func resetServer() async {
        serverURL = ""
        await saveServer()
    }

    private func credentials() -> (email: String, password: String)? {
        let email = email.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !email.isEmpty, !password.isEmpty else {
            errorMessage = "Please enter both email and password"
            return nil
        }
        return (email, password)
    }

    private func perform(_ signIn: () async throws -> User) async {
        guard !isWorking else { return }
        isWorking = true
        errorMessage = nil
        defer { isWorking = false }
        do {
            _ = try await signIn()
        } catch {
            errorMessage = error.localizedDescription
            return
        }
        email = ""
        password = ""
        await onSignedIn()
    }
}
