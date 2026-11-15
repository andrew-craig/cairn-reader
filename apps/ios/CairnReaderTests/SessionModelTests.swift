import Foundation
import Testing
@testable import CairnReader

// Port of apps/mobile/src/contexts/AuthContext.test.tsx.

/// Sessions whose databases live in temporary files, so a test can check the file is erased.
@MainActor
private final class SessionFactory {
    private(set) var urls: [URL] = []

    func make(_ user: User, _ auth: AuthService) -> UserSession {
        let url = try! temporaryDatabaseURL()
        urls.append(url)
        return UserSession(user: user, auth: auth, database: AppDatabase(url: url))
    }
}

@MainActor
private func makeModel(
    _ server: StubServer, session: StoredSession?, sessions: SessionFactory = SessionFactory()
) -> SessionModel {
    let auth = AuthService(
        client: HTTPClient(
            server: ServerConfig(defaults: UserDefaults(suiteName: UUID().uuidString)!), session: server.session()),
        storage: InMemorySessionStorage(session), deviceID: { "device-1" }, sleep: { _ in })
    return SessionModel(auth: auth, makeSession: sessions.make)
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
        let model = SessionModel(auth: auth, makeSession: SessionFactory().make)
        await model.start()

        _ = try await auth.loginWithEmail(email: "a@b.com", password: "pw")
        await model.login()

        #expect(model.user?.email == "a@b.com")
    }

    /// After a device account is upgraded, `login()` shows its email and keeps the session (and
    /// its data): the user id is unchanged.
    @Test func loginPicksUpAnUpgradedAccountInTheSameSession() async throws {
        let model = makeModel(
            StubServer(replies: [.json(200, AuthFixtures.userBody(email: "a@b.com"))]), session: AuthFixtures.valid)
        await model.start()
        let session = try #require(model.session)

        _ = try await model.auth.upgradeAccount(email: "a@b.com", password: "password1")
        await model.login()

        #expect(model.user?.email == "a@b.com")
        #expect(model.session === session)
    }

    @Test func startCreatesASessionForTheRestoredUser() async {
        let model = makeModel(StubServer(replies: []), session: AuthFixtures.valid)
        await model.start()
        #expect(model.session?.user == AuthFixtures.user)
    }

    @Test func logoutClosesTheSessionAndErasesItsDatabase() async throws {
        let sessions = SessionFactory()
        let model = makeModel(StubServer(replies: [.json(200, "{}")]), session: AuthFixtures.valid, sessions: sessions)
        await model.start()
        try await model.session?.articles.upsertMany([storedArticle("a1")])
        let url = try #require(sessions.urls.first)
        #expect(FileManager.default.fileExists(atPath: url.path()))

        await model.logout()

        #expect(model.session == nil)
        #expect(!FileManager.default.fileExists(atPath: url.path()))
    }

    /// A refresh rejected during a background request signs the UI out too and erases the data.
    @Test func followsSignOutByTheService() async throws {
        let server = StubServer(replies: [.json(401, "{}"), .json(401, AuthFixtures.errorBody("invalid refresh token"))])
        let auth = AuthService(
            client: HTTPClient(
                server: ServerConfig(defaults: UserDefaults(suiteName: UUID().uuidString)!), session: server.session()),
            storage: InMemorySessionStorage(AuthFixtures.valid))
        let sessions = SessionFactory()
        let model = SessionModel(auth: auth, makeSession: sessions.make)
        await model.start()
        #expect(model.isAuthenticated)
        try await model.session?.articles.upsertMany([storedArticle("a1")])
        let url = try #require(sessions.urls.first)

        await #expect(throws: AuthError.sessionExpired) { try await auth.authorizedRequest(HTTPRequest(.get, "/x")) }

        let deadline = ContinuousClock.now + .seconds(2)
        while FileManager.default.fileExists(atPath: url.path()) && ContinuousClock.now < deadline {
            try await Task.sleep(for: .milliseconds(10))
        }
        #expect(!model.isAuthenticated)
        #expect(!FileManager.default.fileExists(atPath: url.path()))
    }
}
