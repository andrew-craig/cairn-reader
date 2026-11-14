import Foundation
import Observation
import OSLog

/// On-device data that belongs to the signed-in user and is deleted at sign-out (the article
/// store, the outbox, the feed cache).
nonisolated protocol UserDataStore: Sendable {
    func clear() async throws
}

/// Whether someone is signed in, for the UI. Mirrors `AuthService`, including sign-outs the
/// service makes itself (a rejected refresh during a background request, a server change).
@Observable
final class SessionModel {
    /// True until `start()` has checked the saved session.
    private(set) var isLoading = true
    private(set) var user: User?
    var isAuthenticated: Bool { user != nil }

    @ObservationIgnored private let auth: AuthService
    @ObservationIgnored private let userData: [any UserDataStore]
    @ObservationIgnored private let logger = Logger(subsystem: "com.cairnapp.cairnreader", category: "auth")

    init(auth: AuthService, userData: [any UserDataStore] = []) {
        self.auth = auth
        self.userData = userData
        Task { [weak self, signOuts = auth.signOuts] in
            for await _ in signOuts {
                guard let self else { return }
                // Re-read rather than assume nil: a sign-in may already have followed.
                user = await self.auth.user
            }
        }
    }

    /// Restores the saved session at launch, refreshing the token if it is about to expire.
    /// When the server cannot be reached the saved session is kept: offline is not signed out.
    func start() async {
        do {
            user = try await auth.validatedUser()
        } catch is NetworkError {
            user = await auth.user
        } catch {
            logger.error("Checking the saved session failed: \(error.localizedDescription)")
            user = nil
        }
        isLoading = false
    }

    /// Picks up the session after signing in through `AuthService`.
    func login() async {
        user = await auth.user
    }

    /// Signs out and deletes the user's on-device data. A store that fails to clear does not
    /// keep the user signed in.
    func logout() async {
        await auth.logout()
        for store in userData {
            do {
                try await store.clear()
            } catch {
                logger.error("Clearing \(String(describing: type(of: store))) at sign-out failed: \(error.localizedDescription)")
            }
        }
        user = nil
    }
}
