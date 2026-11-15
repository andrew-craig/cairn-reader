import Foundation
import GRDB
@testable import CairnReader

/// A user's database, outbox and mutations against a fake server.
nonisolated struct OutboxHarness {
    struct StoredRow: Equatable {
        let articleID: String
        let field: String
        let payload: String
        let createdAt: Int
    }

    let database = AppDatabase.inMemory()
    let server: StubServer
    let store: ArticleStore
    let outbox: Outbox
    let mutations: ArticleMutations

    init(session: StoredSession? = AuthFixtures.valid, _ server: StubServer) {
        self.server = server
        let auth = AuthService(
            client: HTTPClient(
                server: ServerConfig(defaults: UserDefaults(suiteName: UUID().uuidString)!), session: server.session()),
            storage: InMemorySessionStorage(session), deviceID: { "device-1" }, sleep: { _ in })
        store = ArticleStore(database: database)
        outbox = Outbox(database: database, read: ReadService(auth: auth, userID: AuthFixtures.user.id))
        mutations = ArticleMutations(database: database, outbox: outbox)
    }

    /// A server that answers every request with `reply`.
    init(session: StoredSession? = AuthFixtures.valid, always reply: StubServer.Reply) {
        self.init(session: session, StubServer { _ in reply })
    }

    static let ok = StubServer.Reply.json(200, #"{ "data": \#(Fixtures.userContent()) }"#)
    static let unreachable = StubServer.Reply.failure(.notConnectedToInternet)

    func rows() throws -> [StoredRow] {
        try database.writer().read { db in
            try Row.fetchAll(db, sql: "SELECT * FROM outbox ORDER BY created_at, rowid").map {
                StoredRow(articleID: $0["article_id"], field: $0["field"], payload: $0["payload"], createdAt: $0["created_at"])
            }
        }
    }

    /// Inserts a row with a chosen `created_at` (`Outbox.enqueue` always uses the clock).
    func seed(_ articleID: String, _ field: String, _ payload: String, createdAt: Int) throws {
        try database.writer().write { db in
            try db.execute(
                sql: "INSERT INTO outbox (article_id, field, payload, created_at) VALUES (?, ?, ?, ?)",
                arguments: [articleID, field, payload, createdAt])
        }
    }

    func enqueue(_ articleID: String, _ field: Outbox.Field, _ request: UpdateUserContentRequest) throws {
        try database.writer().write { db in try Outbox.enqueue(db, articleID: articleID, field: field, request) }
    }

    /// The PATCHes the server received, as (article ID, body).
    var patches: [(id: String, body: [String: Any])] {
        server.requests.filter { $0.httpMethod == "PATCH" }.map { ($0.url?.lastPathComponent ?? "", $0.jsonBody) }
    }

    /// The rejections announced so far; stops the outbox, which ends the stream.
    func finishedRejections() async -> [Outbox.Rejection] {
        await outbox.stop()
        var rejections: [Outbox.Rejection] = []
        for await rejection in outbox.rejections { rejections.append(rejection) }
        return rejections
    }
}
