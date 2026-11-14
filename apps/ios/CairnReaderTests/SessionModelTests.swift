import Foundation
import Synchronization
import Testing
@testable import CairnReader

// Port of apps/mobile/src/contexts/AuthContext.test.tsx.

private nonisolated final class RecordingStore: UserDataStore {
    struct Failure: Error {}
    private let cleared = Mutex(false)
    private let fails: Bool

    init(fails: Bool = false) { self.fails = fails }

    var wasCleared: Bool { cleared.withLock { $0 } }

    func clear() async throws {
        cleared.withLock { $0 = true }
        if fails { throw Failure() }
    }
}

@MainActor
private func makeModel(
    _ server: StubServer, session: StoredSession?, userData: [any UserDataStore] = []
) -> SessionModel {
    let auth = AuthService(
        client: HTTPClient(
            server: ServerConfig(defaults: UserDefaults(suiteName: UUID().uuidString)!), session: server.session()),
        storage: InMemorySessionStorage(session), deviceID: { "device-1" }, sleep: { _ in })
    return SessionModel(auth: auth, userData: userData)
}

@MainActor
struct SessionModelTests {
    @Test func startRestoresValidSession() async {
        let model = makeModel(StubServer(replies: []), session: AuthFixtures.valid)
        #expect(model.isLoading)

        await model.start()

        #expect(!model.isLoading)
        #expect(model.user == AuthFixtures.user)
        #expect(model.isAuthenticated)
    }

    @Test func startWithoutSessionIsSignedOut() async {
        let model = makeModel(StubServer(replies: []), session: nil)
        await model.start()
        #expect(!model.isLoading)
        #expect(!model.isAuthenticated)
    }

    /// Offline at launch with an expiring token: stay signed in with the saved session.
    @Test func startKeepsSessionWhenRefreshIsUnreachable() async {
        let model = makeModel(StubServer(replies: [.failure(.notConnectedToInternet)]), session: AuthFixtures.expired)
        await model.start()
        #expect(model.user == AuthFixtures.user)
    }

    @Test func startSignsOutWhenRefreshIsRejected() async {
        let server = StubServer(replies: [.json(401, AuthFixtures.errorBody("invalid refresh token"))])
        let model = makeModel(server, session: AuthFixtures.expired)
        await model.start()
        #expect(model.user == nil)
    }

    @Test func loginPicksUpTheNewSession() async throws {
        let auth = AuthService(
            client: HTTPClient(
                server: ServerConfig(defaults: UserDefaults(suiteName: UUID().uuidString)!),
                session: StubServer(replies: [.json(200, AuthFixtures.authBody(email: "a@b.com"))]).session()),
            storage: InMemorySessionStorage())
        let model = SessionModel(auth: auth)
        await model.start()

        _ = try await auth.loginWithEmail(email: "a@b.com", password: "pw")
        await model.login()

        #expect(model.user?.email == "a@b.com")
    }

    @Test func logoutClearsUserData() async {
        let stores = [RecordingStore(), RecordingStore()]
        let model = makeModel(StubServer(replies: [.json(200, "{}")]), session: AuthFixtures.valid, userData: stores)
        await model.start()

        await model.logout()

        #expect(model.user == nil)
        #expect(stores.allSatisfy { $0.wasCleared })
    }

    @Test func logoutSucceedsWhenAStoreFailsToClear() async {
        let stores = [RecordingStore(fails: true), RecordingStore()]
        let model = makeModel(StubServer(replies: [.json(200, "{}")]), session: AuthFixtures.valid, userData: stores)
        await model.start()

        await model.logout()

        #expect(model.user == nil)
        #expect(stores[1].wasCleared)
    }

    /// A refresh rejected during a background request signs the UI out too.
    @Test func followsSignOutByTheService() async throws {
        let server = StubServer(replies: [.json(401, "{}"), .json(401, AuthFixtures.errorBody("invalid refresh token"))])
        let auth = AuthService(
            client: HTTPClient(
                server: ServerConfig(defaults: UserDefaults(suiteName: UUID().uuidString)!), session: server.session()),
            storage: InMemorySessionStorage(AuthFixtures.valid))
        let model = SessionModel(auth: auth)
        await model.start()
        #expect(model.isAuthenticated)

        await #expect(throws: AuthError.sessionExpired) { try await auth.authorizedRequest(HTTPRequest(.get, "/x")) }

        let deadline = ContinuousClock.now + .seconds(2)
        while model.isAuthenticated && ContinuousClock.now < deadline {
            try await Task.sleep(for: .milliseconds(10))
        }
        #expect(!model.isAuthenticated)
    }
}
