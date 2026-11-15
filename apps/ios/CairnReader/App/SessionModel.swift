import Foundation
import Observation
import OSLog

/// Whether someone is signed in, for the UI. Mirrors `AuthService`, including sign-outs the
/// service makes itself (a rejected refresh during a background request, a server change), and
/// owns the signed-in user's `UserSession`.
@Observable
final class SessionModel {
    /// True until `start()` has checked the saved session.
    private(set) var isLoading = true
    /// The signed-in user's session; nil when signed out.
    private(set) var session: UserSession?
    /// The signed-in user as last read from `AuthService`; can be newer than `session.user` (an
    /// upgraded device account has an email).
    private(set) var user: User?
    var isAuthenticated: Bool { session != nil }

    @ObservationIgnored let auth: AuthService
    @ObservationIgnored private let makeSession: (User, AuthService) -> UserSession
    @ObservationIgnored private let logger = Logger(subsystem: "com.cairnapp.cairnreader", category: "auth")

    init(
        auth: AuthService,
        makeSession: @escaping (User, AuthService) -> UserSession = { UserSession(user: $0, auth: $1) }
    ) {
        self.auth = auth
        self.makeSession = makeSession
        Task { [weak self, signOuts = auth.signOuts] in
            for await _ in signOuts {
                guard let self else { return }
                // Re-read rather than assume nil: a sign-in may already have followed.
                await setUser(await self.auth.user)
            }
        }
    }

    /// Restores the saved session at launch, refreshing the token if it is about to expire.
    /// When the server cannot be reached the saved session is kept: offline is not signed out.
    func start() async {
        let user: User?
        do {
            user = try await auth.validatedUser()
        } catch is NetworkError {
            user = await auth.user
        } catch {
            logger.error("Checking the saved session failed: \(error.localizedDescription)")
            user = nil
        }
        await setUser(user)
        isLoading = false
    }

    /// Picks up the session after signing in through `AuthService`, or the user's new details
    /// after an account change (upgrading a device account).
    func login() async {
        await setUser(await auth.user)
    }

    /// Signs out and deletes the user's on-device data.
    func logout() async {
        await auth.logout()
        await setUser(nil)
    }

    /// Records the user, and replaces the session when the signed-in user changed, closing the
    /// old one (which erases its data) after the UI has stopped showing it.
    private func setUser(_ user: User?) async {
        self.user = user
        guard user?.id != session?.user.id else { return }
        let old = session
        session = user.map { makeSession($0, auth) }
        await old?.close()
    }
}
