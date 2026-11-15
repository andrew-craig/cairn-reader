import Foundation
import GRDB
@testable import CairnReader

/// A signed-in user's session (store, outbox, mutations, `Sync`) against a fake server.
nonisolated struct SyncHarness {
    let server: StubServer
    let session: UserSession
    var sync: Sync { session.sync }

    init(online: Bool = true, _ server: StubServer) {
        self.server = server
        let auth = AuthService(
            client: HTTPClient(
                server: ServerConfig(defaults: UserDefaults(suiteName: UUID().uuidString)!), session: server.session()),
            storage: InMemorySessionStorage(AuthFixtures.valid), deviceID: { "device-1" }, sleep: { _ in })
        session = UserSession(user: AuthFixtures.user, auth: auth, isOnline: { online }, database: AppDatabase.inMemory())
    }

    /// Each request as "METHOD last-path-component", e.g. "GET user-1" for a list page,
    /// "GET a1" for a detail, "PATCH a1".
    var requests: [String] {
        server.requests.map { "\($0.httpMethod ?? "") \($0.url?.lastPathComponent ?? "")" }
    }

    func enqueue(_ articleID: String, _ field: Outbox.Field, _ request: UpdateUserContentRequest) throws {
        try session.database.writer().write { db in try Outbox.enqueue(db, articleID: articleID, field: field, request) }
    }

    func outboxCount() throws -> Int {
        try session.database.writer().read { db in try Int.fetchOne(db, sql: "SELECT COUNT(*) FROM outbox") ?? 0 }
    }
}

nonisolated enum SyncFixtures {
    /// A Reads list page of `ids`, newest first, one second apart, starting `skip` seconds below
    /// the newest time (pass the earlier pages' row count for a later page).
    static func readsPage(
        _ ids: [String], hasMore: Bool = false, cursor: String? = nil, isFavorite: Bool = false, skip: Int = 0
    ) -> String {
        let items = ids.enumerated().map { index, id in
            userContent(id, addedAt: String(format: "2025-01-15T10:00:%02dZ", 59 - skip - index), isFavorite: isFavorite)
        }
        let cursor = cursor.map { "\"\($0)\"" } ?? "null"
        return """
            { "data": [\(items.joined(separator: ","))],
              "pagination": { "cursor": \(cursor), "has_more": \(hasMore), "limit": 20 } }
            """
    }

    /// The detail of `id`, with a body.
    static func detail(_ id: String) -> String {
        let item = userContent(id, addedAt: "2025-01-15T10:00:00Z", body: body(id))
        return "{ \"data\": \(item) }"
    }

    /// The body `detail(id)` carries.
    static func body(_ id: String) -> String {
        "<p>Body \(id)</p>"
    }

    static let ok = StubServer.Reply.json(200, detail("any"))

    private static func userContent(_ id: String, addedAt: String, isFavorite: Bool = false, body: String? = nil)
        -> String
    {
        """
        {
          "id": "uc-\(id)", "user_id": "user-1", "content_id": "\(id)", "status": "unread", "list": "reads",
          "is_favorite": \(isFavorite), "added_at": "\(addedAt)", "updated_at": "\(addedAt)",
          "content": {
            "id": "\(id)", "content_hash": "hash-\(id)", "original_url": "https://example.com/\(id)",
            "title": "\(id)", "source_type": "rss", \(body.map { "\"cleaned_html\": \"\($0)\"," } ?? "")
            "created_at": "\(addedAt)", "updated_at": "\(addedAt)"
          }
        }
        """
    }
}
