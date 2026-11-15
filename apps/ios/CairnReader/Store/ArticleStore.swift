import Foundation
import GRDB

/// One stored Reads article as the Reads list shows it.
nonisolated struct ReadsEntry: Identifiable, Hashable, Sendable {
    var article: Article
    /// A Save to Reads is queued (an outbox `list` row) and the server has not confirmed it yet.
    var pendingSave: Bool
    var id: String { article.id }
}

/// The on-device store of Reads articles: metadata, user state, and an opportunistically cached
/// body diffed by `content_hash`. The source of truth for the Reads list (decision 8): network
/// pages are written in with `syncPage`, and screens follow the store through its observations.
///
/// Every method throws a store failure: a broken database breaks Reads (there is no network-only
/// fallback list).
nonisolated final class ArticleStore: Sendable {
    private let database: AppDatabase

    init(database: AppDatabase) {
        self.database = database
    }

    // MARK: - Writes

    /// Upserts articles from the server. Never deletes existing rows.
    func upsertMany(_ articles: [Article]) async throws {
        guard !articles.isEmpty else { return }
        try await database.writer().write { db in
            try Self.upsert(articles, db)
        }
    }

    /// Writes one Reads page from the server, newest first, in one transaction: upserts it, then
    /// deletes the stored rows strictly inside the page's `added_at` window that the page did not
    /// return (archived or moved elsewhere) and that have no outbox row.
    ///
    /// The window runs from `previousPageLast` (the previous page's last `added_at`; nil for the
    /// first page) down to this page's last `added_at` (unbounded when `hasMore` is false). Rows
    /// on a boundary are kept: a row with the same `added_at` as a page's last may be on the
    /// neighbouring page.
    func syncPage(_ articles: [Article], olderThan previousPageLast: Date?, hasMore: Bool) async throws {
        let pageLast = articles.map(\.addedAt).min()
        // A page with more to come but no rows has no lower bound: delete nothing.
        let deletes = !(hasMore && pageLast == nil)
        let ids = String(decoding: try JSONEncoder().encode(articles.map(\.id)), as: UTF8.self)
        let arguments: StatementArguments = [
            "upper": previousPageLast?.timeIntervalSince1970,
            "lower": hasMore ? pageLast?.timeIntervalSince1970 : nil,
            "ids": ids,
        ]
        try await database.writer().write { db in
            try Self.upsert(articles, db)
            guard deletes else { return }
            try db.execute(
                sql: """
                    DELETE FROM articles
                    WHERE (:upper IS NULL OR added_at < :upper)
                      AND (:lower IS NULL OR added_at > :lower)
                      AND id NOT IN (SELECT value FROM json_each(:ids))
                      AND NOT EXISTS (SELECT 1 FROM outbox WHERE article_id = articles.id)
                    """,
                arguments: arguments)
        }
    }

    /// Caches a freshly fetched body (cleaned HTML).
    func saveBody(id: String, body: String) async throws {
        try await database.writer().write { db in
            try db.execute(sql: "UPDATE articles SET body = :body WHERE id = :id", arguments: ["id": id, "body": body])
        }
    }

    /// Drops cached bodies for rows outside the `limit` most recently added, so the store never
    /// keeps more bodies than the prefetch cap.
    func evictBodiesOutsideCap(_ limit: Int) async throws {
        try await database.writer().write { db in
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
        try await database.writer().write { db in
            try Self.updateUserState(
                db, id: id, isRead: isRead, isFavorite: isFavorite, scrollPosition: scrollPosition, readAt: readAt)
        }
    }

    /// Removes one article, e.g. on archive.
    func remove(id: String) async throws {
        try await database.writer().write { db in try Self.remove(db, id: id) }
    }

    // The statics are the store writes `ArticleMutations` joins with an outbox upsert in one transaction.

    static func updateUserState(
        _ db: Database, id: String, isRead: Bool? = nil, isFavorite: Bool? = nil, scrollPosition: Double? = nil,
        readAt: Date? = nil
    ) throws {
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
        try db.execute(sql: "UPDATE articles SET \(sets.joined(separator: ", ")) WHERE id = :id", arguments: arguments)
    }

    static func remove(_ db: Database, id: String) throws {
        try db.execute(sql: "DELETE FROM articles WHERE id = :id", arguments: ["id": id])
    }

    // MARK: - Reads

    /// Unread articles with no cached body, newest first, for prefetch. `upsertMany` already
    /// clears a body whose `content_hash` changed, so this is the full selection.
    func listPrefetchCandidates(limit: Int) async throws -> [Article] {
        try await database.writer().read { db in
            try Row.fetchAll(
                db, sql: "SELECT * FROM articles WHERE is_read = 0 AND body IS NULL ORDER BY added_at DESC LIMIT :limit",
                arguments: ["limit": limit]
            ).map(Self.article(from:))
        }
    }

    /// The stored article, or nil when it is not stored.
    func getById(_ id: String) async throws -> Article? {
        try await database.writer().read { db in
            try Self.fetchArticle(id, db)
        }
    }

    // MARK: - Observations

    /// The stored articles added at or after `addedSince`, newest first, at most `limit` of them
    /// (nil: no bound), each with its pending-save flag; emits again after every change to
    /// `articles` or `outbox`.
    func observeReads(limit: Int? = nil, addedSince: Date? = nil) throws -> AsyncValueObservation<[ReadsEntry]> {
        ValueObservation.tracking { db in
            try Row.fetchAll(
                db,
                sql: """
                    SELECT articles.*, EXISTS (
                      SELECT 1 FROM outbox WHERE outbox.article_id = articles.id AND outbox.field = 'list'
                    ) AS pending_save
                    FROM articles
                    WHERE (:since IS NULL OR added_at >= :since)
                    ORDER BY added_at DESC LIMIT COALESCE(:limit, -1)
                    """,
                arguments: ["limit": limit, "since": addedSince?.timeIntervalSince1970]
            ).map { ReadsEntry(article: Self.article(from: $0), pendingSave: $0["pending_save"]) }
        }
        .values(in: try database.writer())
    }

    /// One stored article, for the reader; nil while it is not stored.
    func observeArticle(id: String) throws -> AsyncValueObservation<Article?> {
        ValueObservation.tracking { db in try Self.fetchArticle(id, db) }
            .values(in: try database.writer())
    }

    /// The ids of every stored article: the Feed leaves out items already in Reads.
    func observeStoredIDs() throws -> AsyncValueObservation<Set<String>> {
        ValueObservation.tracking { db in try Set(String.fetchAll(db, sql: "SELECT id FROM articles")) }
            .values(in: try database.writer())
    }

    private static func fetchArticle(_ id: String, _ db: Database) throws -> Article? {
        try Row.fetchOne(db, sql: "SELECT * FROM articles WHERE id = :id", arguments: ["id": id]).map(article(from:))
    }

    static func upsert(_ articles: [Article], _ db: Database) throws {
        for article in articles {
            try db.execute(sql: upsertSQL, arguments: arguments(for: article))
        }
    }

    // MARK: - Mapping

    // A list page upserts its rows; existing rows are updated (`syncPage` deletes separately). Ported from
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
