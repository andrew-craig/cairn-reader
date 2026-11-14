import Foundation
import OSLog
import UIKit

/// The session is gone (none was saved, or the server rejected it): the user has to sign in.
/// Never thrown for an unreachable server, which is a `NetworkError`.
nonisolated enum AuthError: Error, Equatable, LocalizedError {
    case sessionExpired
    case deviceIDUnavailable

    var errorDescription: String? {
        switch self {
        case .sessionExpired: "Session expired. Please log in again."
        case .deviceIDUnavailable: "Failed to get device ID."
        }
    }
}

/// Owns the session: signs in and out, keeps the tokens in `SessionStorage`, and sends
/// authorized requests, refreshing the access token when it is about to expire or is rejected.
///
/// A failed refresh signs the user out only when the server rejected the refresh token (401 or
/// 403). Anything else — no connection, a 429 or 5xx, an unreadable body — keeps the session and
/// surfaces as `NetworkError`, so being offline never looks like "please log in again".
actor AuthService {
    /// Refresh the access token when it expires within this many seconds.
    static let refreshLeeway: TimeInterval = 5 * 60
    /// Retries after the first attempt in the `…WithRetry` requests; the delay doubles from 1s.
    static let maxRetries = 3

    /// Yields whenever the session is cleared, including by the service itself (a rejected
    /// refresh, a server change). One consumer: `SessionModel`.
    nonisolated let signOuts: AsyncStream<Void>
    private let signOutContinuation: AsyncStream<Void>.Continuation

    private let client: HTTPClient
    private let storage: any SessionStorage
    private let deviceID: @Sendable () async -> String?
    private let sleep: @Sendable (Duration) async throws -> Void
    private let logger = Logger(subsystem: "com.cairnapp.cairnreader", category: "auth")

    private var session: StoredSession?
    private var refreshTask: Task<Void, any Error>?

    init(
        client: HTTPClient = HTTPClient(),
        storage: any SessionStorage = KeychainSessionStorage(),
        deviceID: @escaping @Sendable () async -> String? = AuthService.vendorID,
        sleep: @escaping @Sendable (Duration) async throws -> Void = { try await Task.sleep(for: $0) }
    ) {
        self.client = client
        self.storage = storage
        self.deviceID = deviceID
        self.sleep = sleep
        (signOuts, signOutContinuation) = AsyncStream.makeStream()
        session = storage.load()
    }

    /// The device id the backend knows device accounts by. Sent exactly as the system formats
    /// it: the user service matches it as a string, and it is what the React Native app sent.
    static func vendorID() async -> String? {
        await MainActor.run { UIDevice.current.identifierForVendor?.uuidString }
    }

    var user: User? { session?.user }

    // MARK: Sign in and out

    func loginWithEmail(email: String, password: String) async throws -> User {
        try await authenticate("/api/v1/auth/login", EmailCredentials(email: email, password: password))
    }

    func registerWithEmail(email: String, password: String) async throws -> User {
        try await authenticate("/api/v1/auth/register", EmailCredentials(email: email, password: password))
    }

    func loginWithDevice() async throws -> User {
        try await authenticate("/api/v1/auth/login/mobile", DeviceCredentials(expoDeviceId: try await deviceIdentifier()))
    }

    func registerWithDevice() async throws -> User {
        try await authenticate(
            "/api/v1/auth/register/mobile", DeviceCredentials(expoDeviceId: try await deviceIdentifier()))
    }

    /// Revokes the refresh token on the server when it can be reached, and clears the session
    /// either way.
    func logout() async {
        if let session {
            let request = HTTPRequest(
                .post, "/api/v1/auth/logout", body: RefreshTokenRequest(refreshToken: session.refreshToken),
                headers: ["Authorization": "Bearer \(session.accessToken)"])
            do {
                try await client.send(request)
            } catch {
                logger.error("Logout request failed: \(error.localizedDescription)")
            }
        }
        clearSession()
    }

    /// Adds an email and password to a device account.
    func upgradeAccount(email: String, password: String) async throws -> User {
        guard let userID = session?.user.id else { throw AuthError.sessionExpired }
        let request = HTTPRequest(
            .post, "/api/v1/user/\(userID)/upgrade", body: EmailCredentials(email: email, password: password))
        let user = try await authorizedRequest(request, as: Envelope<User>.self).data
        if var session, session.user.id == user.id {
            session.user = user
            save(session)
        }
        return user
    }

    func changePassword(current: String, new: String) async throws {
        guard let userID = session?.user.id else { throw AuthError.sessionExpired }
        try await authorizedRequest(
            HTTPRequest(
                .put, "/api/v1/user/\(userID)/password",
                body: ChangePasswordRequest(currentPassword: current, newPassword: new)))
    }

    /// Saves the server URL (see `ServerConfig.setURL`). Tokens belong to the server that issued
    /// them, so a change clears the session.
    @discardableResult
    func setServerURL(_ input: String) -> Bool {
        let changed = client.server.setURL(input)
        if changed { clearSession() }
        return changed
    }

    // MARK: Authorized requests

    /// The signed-in user once the access token is known to be valid (refreshed if needed), or
    /// nil when there is no session or the server rejected it. Throws `NetworkError` when the
    /// refresh could not reach the server.
    func validatedUser() async throws -> User? {
        do {
            _ = try await validAccessToken()
            return session?.user
        } catch AuthError.sessionExpired {
            return nil
        }
    }

    /// Sends `request` with the access token. A 401 refreshes the token and retries once; a
    /// second 401, or a rejected refresh, clears the session and throws `AuthError.sessionExpired`.
    func authorizedRequest<Response: Decodable & Sendable>(
        _ request: HTTPRequest, as _: Response.Type = Response.self
    ) async throws -> Response {
        try await authorized(request) { client, request in try await client.send(request, as: Response.self) }
    }

    func authorizedRequest(_ request: HTTPRequest) async throws {
        try await authorized(request) { client, request in try await client.send(request) }
    }

    /// `authorizedRequest`, retrying a `NetworkError` or 5xx up to `maxRetries` times with
    /// exponential backoff (1s, 2s, 4s). Other errors, including 4xx, are thrown at once.
    func authorizedRequestWithRetry<Response: Decodable & Sendable>(
        _ request: HTTPRequest, as _: Response.Type = Response.self
    ) async throws -> Response {
        try await withRetry { try await self.authorizedRequest(request, as: Response.self) }
    }

    func authorizedRequestWithRetry(_ request: HTTPRequest) async throws {
        try await withRetry { try await self.authorizedRequest(request) }
    }

    static func isRetryable(_ error: any Error) -> Bool {
        switch error {
        case is NetworkError: true
        case let error as HttpError: error.status >= 500
        default: false
        }
    }

    // MARK: Private

    private func authenticate(_ path: String, _ body: some Encodable & Sendable) async throws -> User {
        let response = try await client.send(HTTPRequest(.post, path, body: body), as: Envelope<AuthResponse>.self)
        store(response.data)
        return response.data.user
    }

    private func deviceIdentifier() async throws -> String {
        guard let id = await deviceID() else { throw AuthError.deviceIDUnavailable }
        return id
    }

    private func authorized<Response: Sendable>(
        _ request: HTTPRequest, _ perform: @Sendable (HTTPClient, HTTPRequest) async throws -> Response
    ) async throws -> Response {
        let token = try await validAccessToken()
        do {
            return try await perform(client, request.bearer(token))
        } catch let error as HttpError where error.status == 401 {
            // Concurrent requests rejected with the same token share one refresh.
            if session?.accessToken == token {
                try await refresh()
            }
            guard let token = session?.accessToken else { throw AuthError.sessionExpired }
            do {
                return try await perform(client, request.bearer(token))
            } catch let error as HttpError where error.status == 401 {
                logger.error("Request rejected again after refresh; signing out")
                clearSession()
                throw AuthError.sessionExpired
            }
        }
    }

    private func validAccessToken() async throws -> String {
        guard let session else { throw AuthError.sessionExpired }
        if session.expiresAt.timeIntervalSinceNow > Self.refreshLeeway {
            return session.accessToken
        }
        try await refresh()
        guard let session = self.session else { throw AuthError.sessionExpired }
        return session.accessToken
    }

    /// Single flight: callers that arrive while a refresh is running await that one, so the
    /// refresh token is never sent twice.
    private func refresh() async throws {
        if let refreshTask {
            return try await refreshTask.value
        }
        let task = Task {
            defer { refreshTask = nil }
            try await performRefresh()
        }
        refreshTask = task
        try await task.value
    }

    private func performRefresh() async throws {
        guard let refreshToken = session?.refreshToken else { throw AuthError.sessionExpired }
        let request = HTTPRequest(.post, "/api/v1/auth/refresh", body: RefreshTokenRequest(refreshToken: refreshToken))
        let response: AuthResponse
        do {
            response = try await client.send(request, as: Envelope<AuthResponse>.self).data
        } catch let error as HttpError where error.status == 401 || error.status == 403 {
            logger.error("Refresh rejected with \(error.status); signing out")
            clearSession()
            throw AuthError.sessionExpired
        } catch let error as HttpError {
            // 400, 404, 429 (auth endpoints are rate limited), 5xx: not a verdict on the token.
            logger.error("Refresh failed with \(error.status); keeping the session")
            throw NetworkError.unreachable
        }
        // Signed out (or in as someone else) while the refresh was in flight.
        guard session?.refreshToken == refreshToken else { throw AuthError.sessionExpired }
        store(response)
    }

    private func store(_ response: AuthResponse) {
        save(
            StoredSession(
                accessToken: response.accessToken, refreshToken: response.refreshToken,
                expiresAt: Date(timeIntervalSinceNow: TimeInterval(response.expiresIn)), user: response.user))
    }

    private func save(_ session: StoredSession) {
        self.session = session
        storage.save(session)
    }

    private func clearSession() {
        session = nil
        storage.clear()
        signOutContinuation.yield()
    }

    private func withRetry<Response: Sendable>(_ operation: () async throws -> Response) async throws -> Response {
        var attempt = 0
        while true {
            do {
                return try await operation()
            } catch where attempt < Self.maxRetries && Self.isRetryable(error) {
                try await sleep(.seconds(1 << attempt))
                attempt += 1
            }
        }
    }
}

extension HTTPRequest {
    /// The request with `Authorization` set to `token`, replacing any the caller set.
    nonisolated fileprivate func bearer(_ token: String) -> HTTPRequest {
        var request = self
        request.headers["Authorization"] = "Bearer \(token)"
        return request
    }
}
