import Foundation
import Testing
import UIKit
@testable import CairnReader

// Port of apps/shared/src/services/auth.test.ts and apps/mobile/src/services/{auth,authFetch,
// authNetworkErrors}.test.ts. Log-content assertions are not ported: nothing in AuthService
// interpolates token material or response bodies into a log message.

private typealias Reply = StubServer.Reply

private func makeAuth(
    _ server: StubServer, storage: InMemorySessionStorage = InMemorySessionStorage(AuthFixtures.valid),
    deviceID: String? = "device-1", sleeper: SleepRecorder = SleepRecorder()
) -> AuthService {
    let config = ServerConfig(defaults: UserDefaults(suiteName: UUID().uuidString)!)
    return AuthService(
        client: HTTPClient(server: config, session: server.session()), storage: storage,
        deviceID: { deviceID }, sleep: { sleeper.sleep($0) })
}

private let anyRequest = HTTPRequest(.get, "/api/v1/x")

/// Refresh outcomes, driven through a request with an expired token.
struct AuthRefreshTests {
    @Test(arguments: [URLError.Code.notConnectedToInternet, .timedOut])
    func unreachableKeepsSessionAndThrowsNetworkError(code: URLError.Code) async {
        let storage = InMemorySessionStorage(AuthFixtures.expired)
        let auth = makeAuth(StubServer(replies: [.failure(code)]), storage: storage)

        await #expect(throws: NetworkError.self) { try await auth.authorizedRequest(anyRequest) }

        #expect(storage.session == AuthFixtures.expired)
        #expect(await auth.user != nil)
    }

    @Test(arguments: [
        "<html>Please log in to the WiFi</html>",
        #"{ "data": { "access_token": "a" } }"#,
    ])
    func unreadableSuccessBodyKeepsSession(body: String) async {
        let storage = InMemorySessionStorage(AuthFixtures.expired)
        let auth = makeAuth(StubServer(replies: [.json(200, body)]), storage: storage)

        await #expect(throws: NetworkError.undecodableBody) { try await auth.authorizedRequest(anyRequest) }

        #expect(storage.session == AuthFixtures.expired)
    }

    @Test(arguments: [401, 403])
    func rejectionClearsSession(status: Int) async {
        let storage = InMemorySessionStorage(AuthFixtures.expired)
        let server = StubServer(replies: [.json(status, AuthFixtures.errorBody("invalid refresh token"))])
        let auth = makeAuth(server, storage: storage)

        await #expect(throws: AuthError.sessionExpired) { try await auth.authorizedRequest(anyRequest) }

        #expect(storage.session == nil)
        #expect(await auth.user == nil)
        #expect(server.paths == ["/api/v1/auth/refresh"])
    }

    /// Rate limits and server errors are not a verdict on the refresh token.
    @Test(arguments: [400, 404, 429, 500, 503])
    func otherStatusesKeepSession(status: Int) async {
        let storage = InMemorySessionStorage(AuthFixtures.expired)
        let auth = makeAuth(StubServer(replies: [.json(status, AuthFixtures.errorBody("nope"))]), storage: storage)

        await #expect(throws: NetworkError.self) { try await auth.authorizedRequest(anyRequest) }

        #expect(storage.session == AuthFixtures.expired)
    }

    @Test func successStoresRotatedPairAndSendsRequestWithIt() async throws {
        let storage = InMemorySessionStorage(AuthFixtures.expired)
        let server = StubServer(replies: [.json(200, AuthFixtures.authBody()), .json(200, "{}")])
        let auth = makeAuth(server, storage: storage)

        try await auth.authorizedRequest(anyRequest)

        #expect(server.requests[0].jsonBody["refresh_token"] as? String == "refresh-1")
        #expect(server.requests[1].bearer == "Bearer access-2")
        #expect(storage.session?.accessToken == "access-2")
        #expect(storage.session?.refreshToken == "refresh-2")
        #expect(try #require(storage.session?.expiresAt).timeIntervalSinceNow > 3500)
    }

    @Test func validTokenIsNotRefreshed() async throws {
        let server = StubServer(replies: [.json(200, "{}")])
        try await makeAuth(server).authorizedRequest(anyRequest)
        #expect(server.paths == ["/api/v1/x"])
    }

    @Test func concurrentRequestsWithExpiredTokenShareOneRefresh() async throws {
        let server = StubServer { request in
            request.url?.path() == "/api/v1/auth/refresh" ? .json(200, AuthFixtures.authBody()) : .json(200, "{}")
        }
        let auth = makeAuth(server, storage: InMemorySessionStorage(AuthFixtures.expired))

        try await withThrowingTaskGroup { group in
            for _ in 0..<5 { group.addTask { try await auth.authorizedRequest(anyRequest) } }
            try await group.waitForAll()
        }

        #expect(server.paths.filter { $0 == "/api/v1/auth/refresh" }.count == 1)
        #expect(server.requests.filter { $0.url?.path() == "/api/v1/x" }.allSatisfy { $0.bearer == "Bearer access-2" })
    }
}

struct AuthorizedRequestTests {
    @Test func noSessionThrowsWithoutARequest() async {
        let server = StubServer(replies: [])
        let auth = makeAuth(server, storage: InMemorySessionStorage())

        await #expect(throws: AuthError.sessionExpired) { try await auth.authorizedRequest(anyRequest) }
        #expect(server.requests.isEmpty)
    }

    @Test func attachesBearerOverCallerAuthorizationAndKeepsOtherHeaders() async throws {
        let server = StubServer(replies: [.json(200, "{}")])
        let request = HTTPRequest(.get, "/api/v1/x", headers: ["X-Trace": "abc", "Authorization": "Bearer attacker"])

        try await makeAuth(server).authorizedRequest(request)

        #expect(server.requests[0].bearer == "Bearer access-1")
        #expect(server.requests[0].value(forHTTPHeaderField: "X-Trace") == "abc")
    }

    @Test func decodesTheResponse() async throws {
        let server = StubServer(replies: [.json(200, AuthFixtures.authBody())])
        let response = try await makeAuth(server).authorizedRequest(anyRequest, as: Envelope<AuthResponse>.self)
        #expect(response.data.accessToken == "access-2")
    }

    @Test func unauthorizedRefreshesOnceAndRetriesWithRotatedToken() async throws {
        let server = StubServer(replies: [.json(401, "{}"), .json(200, AuthFixtures.authBody()), .json(200, "{}")])

        try await makeAuth(server).authorizedRequest(anyRequest)

        #expect(server.paths == ["/api/v1/x", "/api/v1/auth/refresh", "/api/v1/x"])
        #expect(server.requests[2].bearer == "Bearer access-2")
    }

    @Test func unauthorizedThenRejectedRefreshClearsSession() async {
        let storage = InMemorySessionStorage(AuthFixtures.valid)
        let server = StubServer(replies: [.json(401, "{}"), .json(401, AuthFixtures.errorBody("invalid refresh token"))])
        let auth = makeAuth(server, storage: storage)

        await #expect(throws: AuthError.sessionExpired) { try await auth.authorizedRequest(anyRequest) }
        #expect(storage.session == nil)
    }

    @Test func unauthorizedThenUnreachableRefreshKeepsSession() async {
        let storage = InMemorySessionStorage(AuthFixtures.valid)
        let server = StubServer(replies: [.json(401, "{}"), .failure(.notConnectedToInternet)])
        let auth = makeAuth(server, storage: storage)

        await #expect(throws: NetworkError.unreachable) { try await auth.authorizedRequest(anyRequest) }
        #expect(storage.session == AuthFixtures.valid)
    }

    /// A rotated token that is rejected too is a dead session, not an ordinary 401.
    @Test func secondUnauthorizedAfterRefreshClearsSession() async {
        let storage = InMemorySessionStorage(AuthFixtures.valid)
        let server = StubServer(replies: [.json(401, "{}"), .json(200, AuthFixtures.authBody()), .json(401, "{}")])
        let auth = makeAuth(server, storage: storage)

        await #expect(throws: AuthError.sessionExpired) { try await auth.authorizedRequest(anyRequest) }
        #expect(storage.session == nil)
    }

    @Test func retryDroppingAfterRefreshKeepsRotatedSession() async {
        let storage = InMemorySessionStorage(AuthFixtures.valid)
        let server = StubServer(replies: [
            .json(401, "{}"), .json(200, AuthFixtures.authBody()), .failure(.networkConnectionLost),
        ])
        let auth = makeAuth(server, storage: storage)

        await #expect(throws: NetworkError.unreachable) { try await auth.authorizedRequest(anyRequest) }
        #expect(storage.session?.accessToken == "access-2")
    }

    @Test func unreachableWithValidTokenIsNetworkError() async {
        let auth = makeAuth(StubServer(replies: [.failure(.cannotConnectToHost)]))
        await #expect(throws: NetworkError.unreachable) { try await auth.authorizedRequest(anyRequest) }
    }

    @Test func otherStatusIsHttpErrorAndKeepsSession() async {
        let storage = InMemorySessionStorage(AuthFixtures.valid)
        let auth = makeAuth(StubServer(replies: [.json(404, AuthFixtures.errorBody("not found"))]), storage: storage)

        await #expect(throws: HttpError(status: 404, message: "not found")) {
            try await auth.authorizedRequest(anyRequest)
        }
        #expect(storage.session == AuthFixtures.valid)
    }

    @Test func concurrentUnauthorizedResponsesShareOneRefresh() async throws {
        let server = StubServer { request in
            switch (request.url?.path(), request.bearer) {
            case ("/api/v1/auth/refresh", _): .json(200, AuthFixtures.authBody())
            case (_, "Bearer access-1"): .json(401, "{}")
            default: .json(200, "{}")
            }
        }
        let auth = makeAuth(server)

        try await withThrowingTaskGroup { group in
            for _ in 0..<5 { group.addTask { try await auth.authorizedRequest(anyRequest) } }
            try await group.waitForAll()
        }

        #expect(server.paths.filter { $0 == "/api/v1/auth/refresh" }.count == 1)
    }
}

struct AuthRetryTests {
    @Test func clientErrorIsNotRetried() async {
        let server = StubServer(replies: [.json(400, AuthFixtures.errorBody("bad"))])
        let sleeper = SleepRecorder()

        await #expect(throws: HttpError(status: 400, message: "bad")) {
            try await makeAuth(server, sleeper: sleeper).authorizedRequestWithRetry(anyRequest)
        }
        #expect(server.requests.count == 1)
        #expect(sleeper.delays.isEmpty)
    }

    @Test func serverErrorIsRetriedThreeTimesWithBackoff() async {
        let server = StubServer { _ in .json(500, "{}") }
        let sleeper = SleepRecorder()

        await #expect(throws: HttpError(status: 500, message: nil)) {
            try await makeAuth(server, sleeper: sleeper).authorizedRequestWithRetry(anyRequest)
        }
        #expect(server.requests.count == 4)
        #expect(sleeper.delays == [.seconds(1), .seconds(2), .seconds(4)])
    }

    @Test func networkErrorIsRetriedUntilSuccess() async throws {
        let server = StubServer(replies: [.failure(.notConnectedToInternet), .json(200, AuthFixtures.authBody())])

        let response = try await makeAuth(server).authorizedRequestWithRetry(anyRequest, as: Envelope<AuthResponse>.self)

        #expect(response.data.user.id == "user-1")
        #expect(server.requests.count == 2)
    }

    @Test func expiredSessionIsNotRetried() async {
        let server = StubServer(replies: [])
        let sleeper = SleepRecorder()
        let auth = makeAuth(server, storage: InMemorySessionStorage(), sleeper: sleeper)

        await #expect(throws: AuthError.sessionExpired) { try await auth.authorizedRequestWithRetry(anyRequest) }
        #expect(sleeper.delays.isEmpty)
    }
}

struct AuthLoginTests {
    typealias EntryPoint = @Sendable (AuthService) async throws -> User

    static let entryPoints: [(String, EntryPoint)] = [
        ("/api/v1/auth/login", { try await $0.loginWithEmail(email: "a@b.com", password: "pw") }),
        ("/api/v1/auth/register", { try await $0.registerWithEmail(email: "a@b.com", password: "pw") }),
        ("/api/v1/auth/login/mobile", { try await $0.loginWithDevice() }),
        ("/api/v1/auth/register/mobile", { try await $0.registerWithDevice() }),
    ]

    @Test(arguments: entryPoints)
    func successStoresSession(path: String, call: EntryPoint) async throws {
        let storage = InMemorySessionStorage()
        let server = StubServer(replies: [.json(200, AuthFixtures.authBody())])

        let user = try await call(makeAuth(server, storage: storage))

        #expect(user.id == "user-1")
        #expect(server.paths == [path])
        #expect(storage.session?.accessToken == "access-2")
        #expect(storage.session?.user == user)
    }

    @Test func emailLoginSendsCredentials() async throws {
        let server = StubServer(replies: [.json(200, AuthFixtures.authBody())])
        _ = try await makeAuth(server).loginWithEmail(email: "a@b.com", password: "pw")
        #expect(server.requests[0].jsonBody as? [String: String] == ["email": "a@b.com", "password": "pw"])
    }

    /// Callers tell a rejected credential (4xx) from a broken server (5xx) by status, not text.
    @Test(arguments: [401, 500])
    func failureIsHttpErrorWithStatusAndMessage(status: Int) async {
        for (_, call) in Self.entryPoints {
            let server = StubServer(replies: [.json(status, AuthFixtures.errorBody("server says no"))])
            await #expect(throws: HttpError(status: status, message: "server says no")) {
                try await call(makeAuth(server))
            }
        }
    }

    /// A captive portal answering 200 with a login page did not reach the API.
    @Test(arguments: entryPoints)
    func captivePortalIsNetworkError(path _: String, call: EntryPoint) async {
        let server = StubServer(replies: [.json(200, "<html>Please log in to the WiFi</html>")])
        await #expect(throws: NetworkError.undecodableBody) { try await call(makeAuth(server)) }
    }

    @Test(arguments: entryPoints)
    func unreachableIsNetworkError(path _: String, call: EntryPoint) async {
        let server = StubServer(replies: [.failure(.notConnectedToInternet)])
        await #expect(throws: NetworkError.unreachable) { try await call(makeAuth(server)) }
    }

    /// The user service matches device accounts by exact string, and the React Native app sent
    /// `identifierForVendor.uuidString` as is (upper case). Uses the real provider.
    @Test func deviceLoginSendsIdentifierForVendorUnmodified() async throws {
        let expected = try #require(await UIDevice.current.identifierForVendor?.uuidString)
        let server = StubServer(replies: [.json(200, AuthFixtures.authBody())])
        let auth = AuthService(
            client: HTTPClient(
                server: ServerConfig(defaults: UserDefaults(suiteName: UUID().uuidString)!), session: server.session()),
            storage: InMemorySessionStorage())

        _ = try await auth.loginWithDevice()

        #expect(server.requests[0].jsonBody as? [String: String] == ["expo_device_id": expected])
    }

    @Test func missingDeviceIDThrowsWithoutARequest() async {
        let server = StubServer(replies: [])
        await #expect(throws: AuthError.deviceIDUnavailable) {
            try await makeAuth(server, deviceID: nil).loginWithDevice()
        }
        #expect(server.requests.isEmpty)
    }
}

struct AuthSessionTests {
    @Test func logoutRevokesRefreshTokenAndClears() async {
        let storage = InMemorySessionStorage(AuthFixtures.valid)
        let server = StubServer(replies: [.json(200, "{}")])
        let auth = makeAuth(server, storage: storage)

        await auth.logout()

        #expect(server.paths == ["/api/v1/auth/logout"])
        #expect(server.requests[0].bearer == "Bearer access-1")
        #expect(server.requests[0].jsonBody["refresh_token"] as? String == "refresh-1")
        #expect(storage.session == nil)
        #expect(await auth.user == nil)
    }

    @Test func logoutClearsWhenServerIsUnreachable() async {
        let storage = InMemorySessionStorage(AuthFixtures.valid)
        let auth = makeAuth(StubServer(replies: [.failure(.notConnectedToInternet)]), storage: storage)

        await auth.logout()

        #expect(storage.session == nil)
    }

    @Test func logoutSignalsSignOut() async {
        let auth = makeAuth(StubServer(replies: [.json(200, "{}")]))
        var iterator = auth.signOuts.makeAsyncIterator()

        await auth.logout()

        #expect(await iterator.next() != nil)
    }

    @Test func restoresSavedSession() async throws {
        let storage = InMemorySessionStorage(AuthFixtures.valid)
        let server = StubServer(replies: [.json(200, "{}")])
        let auth = makeAuth(server, storage: storage)

        #expect(await auth.user == AuthFixtures.user)
        try await auth.authorizedRequest(anyRequest)
        #expect(server.requests[0].bearer == "Bearer access-1")
    }

    @Test func validatedUserRefreshesAnExpiredToken() async throws {
        let server = StubServer(replies: [.json(200, AuthFixtures.authBody())])
        let auth = makeAuth(server, storage: InMemorySessionStorage(AuthFixtures.expired))

        #expect(try await auth.validatedUser()?.id == "user-1")
        #expect(server.paths == ["/api/v1/auth/refresh"])
    }

    @Test func validatedUserIsNilWhenRefreshIsRejected() async throws {
        let server = StubServer(replies: [.json(401, AuthFixtures.errorBody("invalid refresh token"))])
        let auth = makeAuth(server, storage: InMemorySessionStorage(AuthFixtures.expired))

        #expect(try await auth.validatedUser() == nil)
    }

    @Test func validatedUserThrowsWhenRefreshIsUnreachable() async {
        let auth = makeAuth(
            StubServer(replies: [.failure(.notConnectedToInternet)]),
            storage: InMemorySessionStorage(AuthFixtures.expired))

        await #expect(throws: NetworkError.unreachable) { try await auth.validatedUser() }
    }

    @Test func changingServerClearsSession() async {
        let storage = InMemorySessionStorage(AuthFixtures.valid)
        let auth = makeAuth(StubServer(replies: []), storage: storage)

        #expect(await auth.setServerURL(ServerConfig.defaultURL.absoluteString) == false)
        #expect(storage.session == AuthFixtures.valid)

        #expect(await auth.setServerURL("cairn.example.com"))
        #expect(storage.session == nil)
        #expect(await auth.user == nil)
    }

    @Test func changePasswordAcceptsEmptyNoContent() async throws {
        let server = StubServer(replies: [.json(204, "")])

        try await makeAuth(server).changePassword(current: "old", new: "new")

        #expect(server.requests[0].httpMethod == "PUT")
        #expect(server.paths == ["/api/v1/user/user-1/password"])
        #expect(server.requests[0].jsonBody as? [String: String] == ["current_password": "old", "new_password": "new"])
    }

    @Test func changePasswordReportsServerMessage() async {
        let server = StubServer(replies: [.json(400, AuthFixtures.errorBody("wrong password"))])
        await #expect(throws: HttpError(status: 400, message: "wrong password")) {
            try await makeAuth(server).changePassword(current: "old", new: "new")
        }
    }

    @Test func changePasswordUnreachableIsNetworkError() async {
        let server = StubServer(replies: [.failure(.notConnectedToInternet)])
        await #expect(throws: NetworkError.unreachable) {
            try await makeAuth(server).changePassword(current: "old", new: "new")
        }
    }

    @Test func upgradeAccountSavesUpgradedUser() async throws {
        let storage = InMemorySessionStorage(AuthFixtures.valid)
        let upgraded = """
            { "data": { "id": "user-1", "email": "a@b.com", "email_verified": false,
              "created_at": "2020-01-01T00:00:00Z", "updated_at": "2020-01-02T00:00:00Z" } }
            """
        let server = StubServer(replies: [.json(200, upgraded)])
        let auth = makeAuth(server, storage: storage)

        let user = try await auth.upgradeAccount(email: "a@b.com", password: "password1")

        #expect(user.email == "a@b.com")
        #expect(server.paths == ["/api/v1/user/user-1/upgrade"])
        #expect(server.requests[0].jsonBody as? [String: String] == ["email": "a@b.com", "password": "password1"])
        #expect(storage.session?.user.email == "a@b.com")
        #expect(storage.session?.accessToken == "access-1")
    }

    @Test func upgradeAccountUnreachableIsNetworkError() async {
        let server = StubServer(replies: [.failure(.notConnectedToInternet)])
        await #expect(throws: NetworkError.unreachable) {
            try await makeAuth(server).upgradeAccount(email: "a@b.com", password: "password1")
        }
    }
}

struct KeychainSessionStorageTests {
    @Test func roundTrips() throws {
        let storage = KeychainSessionStorage(service: "test-\(UUID().uuidString)")
        defer { storage.clear() }
        #expect(storage.load() == nil)

        storage.save(AuthFixtures.valid)
        #expect(storage.load() == AuthFixtures.valid)

        var rotated = AuthFixtures.valid
        rotated.accessToken = "access-2"
        storage.save(rotated)
        #expect(storage.load() == rotated)

        storage.clear()
        #expect(storage.load() == nil)
    }
}
