import Foundation
import GRDB
import os

/// The on-device store of Reads articles: metadata, user state, and an opportunistically cached
/// body diffed by `content_hash`. A read-through cache for the Reads tab's first render and for
/// offline reading, not a query engine: lists still paginate against the network.
///
/// Sync is upsert-only; rows leave by `remove` (archive) or `clear` (sign-out). Reads never throw:
/// a store failure degrades to "nothing cached" (logged), so a caller's network refresh always
/// runs. Writes throw.
nonisolated final class ArticleStore: Sendable {
    private static let logger = Logger(subsystem: "com.cairnapp.cairnreader", category: "store")

    private let database: AppDatabase

    init(database: AppDatabase = .shared) {
        self.database = database
    }

    // MARK: - Writes

    /// Upserts a page of articles from the server. Never deletes existing rows.
    func upsertMany(_ articles: [Article]) async throws {
        guard !articles.isEmpty else { return }
        try await database.queue().write { db in
            for article in articles {
                try db.execute(sql: Self.upsertSQL, arguments: Self.arguments(for: article))
            }
        }
    }

    /// Caches a freshly fetched body (cleaned HTML).
    func saveBody(id: String, body: String) async throws {
        try await database.queue().write { db in
            try db.execute(sql: "UPDATE articles SET body = :body WHERE id = :id", arguments: ["id": id, "body": body])
        }
    }

    /// Drops cached bodies for rows outside the `limit` most recently added, so the store never
    /// keeps more bodies than the prefetch cap.
    func evictBodiesOutsideCap(_ limit: Int) async throws {
        try await database.queue().write { db in
            try db.execute(
                sql: """
                    UPDATE articles SET body = NULL WHERE id NOT IN (
                      SELECT id FROM articles ORDER BY added_at DESC LIMIT :limit
                    )
                    """,
                arguments: ["limit": limit])
        }
    }

    /// Sets the given user-state fields; nil leaves a field unchanged. Does nothing for an
    /// unknown id.
    func updateUserState(
        id: String, isRead: Bool? = nil, isFavorite: Bool? = nil, scrollPosition: Double? = nil,
        readAt: Date? = nil
    ) async throws {
        var sets: [String] = []
        var arguments: StatementArguments = ["id": id]
        if let isRead {
            sets.append("is_read = :is_read")
            arguments += ["is_read": isRead]
        }
        if let isFavorite {
            sets.append("is_favorite = :is_favorite")
            arguments += ["is_favorite": isFavorite]
        }
        if let scrollPosition {
            sets.append("scroll_position = :scroll_position")
            arguments += ["scroll_position": scrollPosition]
        }
        if let readAt {
            sets.append("read_at = :read_at")
            arguments += ["read_at": readAt.timeIntervalSince1970]
        }
        guard !sets.isEmpty else { return }

        let sql = "UPDATE articles SET \(sets.joined(separator: ", ")) WHERE id = :id"
        try await database.queue().write { [arguments] db in
            try db.execute(sql: sql, arguments: arguments)
        }
    }

    /// Removes one article, e.g. on archive.
    func remove(id: String) async throws {
        try await database.queue().write { db in
            try db.execute(sql: "DELETE FROM articles WHERE id = :id", arguments: ["id": id])
        }
    }

    /// Drops every stored article and every queued outbox row, e.g. on sign-out: a queued write
    /// must never replay against a different account.
    func clear() async throws {
        try await database.queue().write { db in
            try db.execute(sql: "DELETE FROM articles")
            try db.execute(sql: "DELETE FROM outbox")
        }
    }

    // MARK: - Reads (never throw)

    /// The most recently added articles, for the Reads tab's first render.
    func listRecent(limit: Int) async -> [Article] {
        await fetch("recent articles", "SELECT * FROM articles ORDER BY added_at DESC LIMIT :limit", ["limit": limit])
    }

    /// Stored favorites, for the Bookmarks screen's first render.
    func listFavorites() async -> [Article] {
        await fetch("favorite articles", "SELECT * FROM articles WHERE is_favorite = 1 ORDER BY added_at DESC")
    }

    /// Stored articles with a queued Save to Reads (an outbox `list` row), most recently queued
    /// first: the Reads tab keeps them listed until the server's list includes them.
    func listPendingSaves() async -> [Article] {
        await fetch(
            "pending saves",
            """
            SELECT articles.* FROM articles
            JOIN outbox ON outbox.article_id = articles.id AND outbox.field = 'list'
            ORDER BY outbox.created_at DESC
            """)
    }

    /// Unread articles with no cached body, newest first, for prefetch. `upsertMany` already
    /// clears a body whose `content_hash` changed, so this is the full selection.
    func listPrefetchCandidates(limit: Int) async -> [Article] {
        await fetch(
            "prefetch candidates",
            "SELECT * FROM articles WHERE is_read = 0 AND body IS NULL ORDER BY added_at DESC LIMIT :limit",
            ["limit": limit])
    }

    /// The stored article, or nil when it is not stored or the store failed.
    func getById(_ id: String) async -> Article? {
        await fetch("article by id", "SELECT * FROM articles WHERE id = :id", ["id": id]).first
    }

    private func fetch(_ what: String, _ sql: String, _ arguments: StatementArguments = []) async -> [Article] {
        do {
            return try await database.queue().read { db in
                try Row.fetchAll(db, sql: sql, arguments: arguments).map(Self.article(from:))
            }
        } catch {
            Self.logger.error("Error loading \(what, privacy: .public): \(String(describing: error))")
            return []
        }
    }

    // MARK: - Mapping

    // A list page upserts its rows; existing rows are updated, never bulk-deleted. Ported from
    // the RN store's UPSERT_SQL, with its three guards:
    // - A queued archive (a `status` outbox row with `{"status":"archived"}`) skips the row
    //   entirely: the WHERE on the SELECT yields no row, so ON CONFLICT never fires. Otherwise the
    //   server still listing the article would resurrect it before the queued PATCH lands.
    // - Any other outbox row for the article freezes the user-state columns (is_read,
    //   is_favorite, read_at, scroll_position) at their stored value: those are what a queued
    //   write is waiting to change, and the server has not seen the new value yet. Keyed on
    //   article_id alone, not per field: simpler, and the over-freezing clears once the outbox
    //   drains.
    // - `body` survives a refresh whose `content_hash` is absent or unchanged (list pages never
    //   carry a body), and is dropped when the hash differs: a stale body must not be served
    //   offline. `content_hash` keeps the stored value when the incoming row has none.
    private static let upsertSQL = """
        INSERT INTO articles (
          id, url, title, description, image_url, author, published_date,
          reading_time, is_read, is_favorite, added_at, read_at,
          scroll_position, body, content_hash
        )
        SELECT
          :id, :url, :title, :description, :image_url, :author, :published_date,
          :reading_time, :is_read, :is_favorite, :added_at, :read_at,
          :scroll_position, :body, :content_hash
        WHERE NOT EXISTS (
          SELECT 1 FROM outbox
          WHERE article_id = :id AND field = 'status' AND json_extract(payload, '$.status') = 'archived'
        )
        ON CONFLICT(id) DO UPDATE SET
          url = excluded.url,
          title = excluded.title,
          description = excluded.description,
          image_url = excluded.image_url,
          author = excluded.author,
          published_date = excluded.published_date,
          reading_time = excluded.reading_time,
          is_read = CASE
            WHEN EXISTS (SELECT 1 FROM outbox WHERE article_id = articles.id) THEN articles.is_read
            ELSE excluded.is_read
          END,
          is_favorite = CASE
            WHEN EXISTS (SELECT 1 FROM outbox WHERE article_id = articles.id) THEN articles.is_favorite
            ELSE excluded.is_favorite
          END,
          added_at = excluded.added_at,
          read_at = CASE
            WHEN EXISTS (SELECT 1 FROM outbox WHERE article_id = articles.id) THEN articles.read_at
            ELSE excluded.read_at
          END,
          scroll_position = CASE
            WHEN EXISTS (SELECT 1 FROM outbox WHERE article_id = articles.id) THEN articles.scroll_position
            ELSE excluded.scroll_position
          END,
          content_hash = COALESCE(excluded.content_hash, articles.content_hash),
          body = CASE
            WHEN excluded.content_hash IS NULL THEN COALESCE(excluded.body, articles.body)
            WHEN excluded.content_hash = articles.content_hash THEN COALESCE(excluded.body, articles.body)
            ELSE NULL
          END
        """

    private static func arguments(for article: Article) -> StatementArguments {
        [
            "id": article.id,
            "url": article.url,
            "title": article.title,
            "description": article.description,
            "image_url": article.imageUrl,
            "author": article.author,
            "published_date": article.publishedDate?.timeIntervalSince1970,
            "reading_time": article.readingTime,
            "is_read": article.isRead,
            "is_favorite": article.isFavorite,
            "added_at": article.addedAt.timeIntervalSince1970,
            "read_at": article.readAt?.timeIntervalSince1970,
            "scroll_position": article.scrollPosition,
            "body": article.content,
            "content_hash": article.contentHash,
        ]
    }

    private static func article(from row: Row) -> Article {
        Article(
            id: row["id"],
            url: row["url"],
            title: row["title"],
            description: row["description"],
            content: row["body"],
            contentHash: row["content_hash"],
            imageUrl: row["image_url"],
            author: row["author"],
            publishedDate: (row["published_date"] as Double?).map(Date.init(timeIntervalSince1970:)),
            readingTime: row["reading_time"],
            isRead: row["is_read"],
            isFavorite: row["is_favorite"],
            list: .reads,  // the store only ever holds Reads
            addedAt: Date(timeIntervalSince1970: row["added_at"]),
            readAt: (row["read_at"] as Double?).map(Date.init(timeIntervalSince1970:)),
            scrollPosition: row["scroll_position"])
    }
}
