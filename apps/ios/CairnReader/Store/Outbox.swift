import Foundation
import GRDB
import OSLog
import Synchronization

/// The queue of edits not yet sent to the backend, and the only thing that sends them (decision
/// 9). `ArticleMutations` writes a row in the same transaction as the store change; `drain`
/// replays the rows. Rows are keyed on `(article_id, field)`, so a repeated edit replaces the
/// payload and keeps the row's place in the queue. Payloads are `UpdateUserContentRequest`
/// fragments in the server's field names, replayed as they are.
nonisolated final class Outbox: Sendable {
    /// The server's PATCH field names; an archive is a `status` row, a Save to Reads a `list` row.
    enum Field: String, DatabaseValueConvertible, Sendable {
        case status
        case isFavorite = "is_favorite"
        case scrollPosition = "scroll_position"
        case list
    }

    /// An archive or a Save to Reads the server definitively refused. The row is gone and the
    /// next Reads sync restores the server's state; the signed-in root tells the user.
    struct Rejection: Equatable, Sendable {
        let articleID: String
        let field: Field
    }

    /// Dropped archives and saves. One consumer.
    let rejections: AsyncStream<Rejection>

    private struct Row: Sendable {
        let articleID: String
        let field: Field
        let payload: String
    }

    private struct State {
        var running: Task<Bool, Never>?
        var rerun = false
        var stopped = false
    }

    private static let logger = Logger(subsystem: "com.cairnapp.cairnreader", category: "outbox")

    private let database: AppDatabase
    private let read: ReadService
    private let continuation: AsyncStream<Rejection>.Continuation
    private let state = Mutex(State())

    init(database: AppDatabase, read: ReadService) {
        self.database = database
        self.read = read
        (rejections, continuation) = AsyncStream.makeStream()
    }

    /// True for a transient failure: the server could not be reached, 401 or 5xx. Anything else
    /// from the server is a definitive refusal; an error that is neither is not recognised.
    static func isRetryable(_ error: any Error) -> Bool {
        switch error {
        case is NetworkError: true
        case let error as HttpError: error.status == 401 || error.status >= 500
        default: false
        }
    }

    // MARK: - Writing rows

    /// Queues `request` for `article` inside the caller's transaction. A row for the same
    /// `(article, field)` keeps its `created_at`, so it keeps its place in the queue.
    static func enqueue(
        _ db: Database, articleID: String, field: Field, _ request: UpdateUserContentRequest
    ) throws {
        try db.execute(
            sql: """
                INSERT INTO outbox (article_id, field, payload, created_at) VALUES (?, ?, ?, ?)
                ON CONFLICT(article_id, field) DO UPDATE SET payload = excluded.payload
                """,
            arguments: [articleID, field, try payload(request), Int(Date().timeIntervalSince1970 * 1000)])
    }

    private static func payload(_ request: UpdateUserContentRequest) throws -> String {
        let encoder = JSONEncoder()
        encoder.keyEncodingStrategy = .convertToSnakeCase
        encoder.outputFormatting = .sortedKeys
        return String(decoding: try encoder.encode(request), as: UTF8.self)
    }

    private static func request(from payload: String) throws -> UpdateUserContentRequest {
        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        return try decoder.decode(UpdateUserContentRequest.self, from: Data(payload.utf8))
    }

    // MARK: - Draining

    /// Drains and returns when the queue is empty or a row cannot be sent; true when a row was
    /// dropped, so the store may hold an edit the server refused. Joins a drain already running,
    /// which then runs again so edits made before this call are included. Does nothing once
    /// stopped. Drains are requested through `Sync`.
    @discardableResult
    func drain() async -> Bool {
        guard let task = start() else { return false }
        return await task.value
    }

    /// Cancels the drain, waits for it, and ends `rejections`.
    func stop() async {
        let task = state.withLock { state in
            state.stopped = true
            return state.running
        }
        task?.cancel()
        await task?.value
        continuation.finish()
    }

    /// One drain at a time: a request made while one runs sets `rerun`.
    private func start() -> Task<Bool, Never>? {
        state.withLock { state in
            guard !state.stopped else { return nil }
            if let running = state.running {
                state.rerun = true
                return running
            }
            let task = Task { await self.run() }
            state.running = task
            return task
        }
    }

    private func run() async -> Bool {
        var dropped = false
        repeat {
            if await pass() { dropped = true }
        } while state.withLock { state in
            if state.rerun && !state.stopped {
                state.rerun = false
                return true
            }
            state.running = nil
            return false
        }
        return dropped
    }

    /// Replays the rows in `created_at` order (rowid breaks ties), stopping at the first that
    /// cannot succeed so a later row never reaches the server ahead of an earlier one. No backoff
    /// and no drop-after-N: the next request tries again. True when a row was dropped.
    private func pass() async -> Bool {
        var dropped = false
        do {
            let rows = try await database.writer().read { db in
                try GRDB.Row.fetchAll(
                    db, sql: "SELECT article_id, field, payload FROM outbox ORDER BY created_at, rowid"
                ).map { Row(articleID: $0["article_id"], field: $0["field"], payload: $0["payload"]) }
            }
            for row in rows {
                if Task.isCancelled { break }
                switch try await send(row) {
                case .sent: break
                case .dropped: dropped = true
                case .halted: return dropped
                }
            }
        } catch {
            Self.logger.error("Draining the outbox failed: \(String(describing: error))")
        }
        return dropped
    }

    private enum Outcome {
        /// Sent; the drain continues.
        case sent
        /// The server definitively refused it; the drain continues.
        case dropped
        /// It cannot succeed now; the drain stops.
        case halted
    }

    /// Sends one row. A refused `list` row also deletes the article, undoing the optimistic Save
    /// to Reads.
    private func send(_ row: Row) async throws -> Outcome {
        let request = try Self.request(from: row.payload)
        do {
            _ = try await read.updateUserContent(row.articleID, request)
        } catch let error as HttpError where !Self.isRetryable(error) {
            Self.logger.error("Dropping \(row.field.rawValue) edit of \(row.articleID): \(error.status)")
            try await remove(row, deletingArticle: row.field == .list)
            if row.field == .list || request.status == .archived {
                continuation.yield(Rejection(articleID: row.articleID, field: row.field))
            }
            return .dropped
        } catch {
            Self.logger.info("Outbox halted at \(row.field.rawValue) edit of \(row.articleID): \(String(describing: error))")
            return .halted
        }
        try await remove(row, deletingArticle: false)
        return .sent
    }

    /// Deletes the row only if it still holds the payload that was sent: an edit made during the
    /// request replaced it and goes out on the next pass.
    private func remove(_ row: Row, deletingArticle: Bool) async throws {
        try await database.writer().write { db in
            try db.execute(
                sql: "DELETE FROM outbox WHERE article_id = ? AND field = ? AND payload = ?",
                arguments: [row.articleID, row.field, row.payload])
            if deletingArticle {
                try ArticleStore.remove(db, id: row.articleID)
            }
        }
    }
}
