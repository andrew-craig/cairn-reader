import Foundation
import GRDB

/// The reader's and the lists' edits (decision 9). Each is one transaction, the store change plus
/// the outbox row, then a request to drain that the caller does not wait for. Nothing is sent
/// from here, so a throw means the database failed, never that the backend said no: a refusal
/// comes back later through `Outbox.rejections`.
nonisolated final class ArticleMutations: Sendable {
    private let database: AppDatabase
    private let outbox: Outbox

    init(database: AppDatabase, outbox: Outbox) {
        self.database = database
        self.outbox = outbox
    }

    func markCompleted(_ id: String, readAt: Date) async throws {
        try await apply(id, .status, UpdateUserContentRequest(status: .completed)) { db in
            try ArticleStore.updateUserState(db, id: id, isRead: true, readAt: readAt)
        }
    }

    /// Queues only: the store has no "reading" state.
    func markReading(_ id: String) async throws {
        try await apply(id, .status, UpdateUserContentRequest(status: .reading)) { _ in }
    }

    func saveScrollPosition(_ id: String, fraction: Double) async throws {
        try await apply(id, .scrollPosition, UpdateUserContentRequest(scrollPosition: fraction)) { db in
            try ArticleStore.updateUserState(db, id: id, scrollPosition: fraction)
        }
    }

    func setFavorite(_ id: String, isFavorite: Bool) async throws {
        try await apply(id, .isFavorite, UpdateUserContentRequest(isFavorite: isFavorite)) { db in
            try ArticleStore.updateUserState(db, id: id, isFavorite: isFavorite)
        }
    }

    func archive(_ id: String) async throws {
        try await apply(id, .status, UpdateUserContentRequest(status: .archived)) { db in
            try ArticleStore.remove(db, id: id)
        }
    }

    /// Stores a Feed item as a Reads item, added now as the server will stamp it (a body is kept
    /// only if the item carries one), and queues the move.
    func saveToReads(_ article: Article) async throws {
        var saved = article
        saved.list = .reads
        saved.addedAt = Date()
        try await apply(article.id, .list, UpdateUserContentRequest(list: .reads)) { [saved] db in
            try ArticleStore.upsert([saved], db)
        }
    }

    private func apply(
        _ id: String, _ field: Outbox.Field, _ request: UpdateUserContentRequest,
        store: @Sendable (Database) throws -> Void
    ) async throws {
        try await database.writer().write { db in
            try store(db)
            try Outbox.enqueue(db, articleID: id, field: field, request)
        }
        outbox.requestDrain()
    }
}
