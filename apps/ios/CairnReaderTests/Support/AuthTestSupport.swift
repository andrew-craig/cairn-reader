import Foundation
import Synchronization
@testable import CairnReader

nonisolated final class InMemorySessionStorage: SessionStorage {
    private let stored: Mutex<StoredSession?>

    init(_ session: StoredSession? = nil) {
        stored = Mutex(session)
    }

    var session: StoredSession? { stored.withLock { $0 } }

    func load() -> StoredSession? { session }
    func save(_ session: StoredSession) { stored.withLock { $0 = session } }
    func clear() { stored.withLock { $0 = nil } }
}

/// A fake server: answers each request through `respond` and records what it was sent.
nonisolated final class StubServer: Sendable {
    enum Reply: Sendable {
        case json(Int, String)
        case failure(URLError.Code)
    }

    private let log = Mutex<[URLRequest]>([])
    private let respond: @Sendable (URLRequest) -> Reply

    init(_ respond: @escaping @Sendable (URLRequest) -> Reply) {
        self.respond = respond
    }

    /// Answers requests in order with `replies`. A request beyond the last gets a transport failure
    /// (`NetworkError.unreachable`), so assert on `requests` or `paths` to catch extra requests.
    convenience init(replies: [Reply]) {
        let queue = Mutex(replies)
        self.init { request in
            queue.withLock { $0.isEmpty ? .failure(.badServerResponse) : $0.removeFirst() }
        }
    }

    var requests: [URLRequest] { log.withLock { $0 } }
    var paths: [String] { requests.map { $0.url?.path() ?? "" } }

    func session() -> URLSession {
        StubURLProtocol.session { request in
            self.log.withLock { $0.append(request) }
            switch self.respond(request) {
            case let .json(status, body): return (status, body.utf8Data)
            case let .failure(code): throw URLError(code)
            }
        }
    }
}

nonisolated final class SleepRecorder: Sendable {
    private let log = Mutex<[Duration]>([])
    var delays: [Duration] { log.withLock { $0 } }
    func sleep(_ duration: Duration) { log.withLock { $0.append(duration) } }
}

nonisolated enum AuthFixtures {
    static let user = User(
        id: "user-1", email: nil, emailVerified: false, createdAt: Date(timeIntervalSince1970: 1_577_836_800),
        updatedAt: Date(timeIntervalSince1970: 1_577_836_800), lastLoginAt: nil)

    /// A saved session comfortably inside its validity window.
    static let valid = StoredSession(
        accessToken: "access-1", refreshToken: "refresh-1", expiresAt: Date(timeIntervalSinceNow: 3600), user: user)

    /// A saved session whose access token is already past the refresh leeway.
    static let expired = StoredSession(
        accessToken: "access-1", refreshToken: "refresh-1", expiresAt: Date(timeIntervalSinceNow: -1), user: user)

    static func authBody(access: String = "access-2", refresh: String = "refresh-2", email: String? = nil) -> String {
        """
        {
          "data": {
            "user": {
              "id": "user-1", \(email.map { "\"email\": \"\($0)\"," } ?? "") "email_verified": false,
              "created_at": "2020-01-01T00:00:00Z", "updated_at": "2020-01-01T00:00:00Z"
            },
            "access_token": "\(access)", "refresh_token": "\(refresh)", "expires_in": 3600
          }
        }
        """
    }

    static func errorBody(_ message: String) -> String {
        #"{ "error": "error", "message": "\#(message)" }"#
    }
}

extension URLRequest {
    nonisolated var bearer: String? { value(forHTTPHeaderField: "Authorization") }

    nonisolated var jsonBody: [String: Any] {
        (try? JSONSerialization.jsonObject(with: httpBody ?? Data())) as? [String: Any] ?? [:]
    }
}
