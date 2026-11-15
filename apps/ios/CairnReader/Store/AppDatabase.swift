import Foundation
import GRDB
import Synchronization

/// The single on-device database shared by `ArticleStore` and the outbox: one file and one
/// migrator, so the two tables can reference each other (the outbox guards in `ArticleStore`'s
/// upsert) without either reaching into the other's internals.
///
/// Opened lazily on first use. A failed open is not cached: the next call tries again, so a
/// transient failure (the OS briefly denying disk access) cannot disable the store for the rest
/// of the run.
nonisolated final class AppDatabase: Sendable {
    /// The app's database file in Application Support. The RN app's `cairnreader.db` is never
    /// opened: on-device RN data is not migrated (epic_bc48 decision 4).
    static let shared = AppDatabase {
        let directory = try FileManager.default.url(
            for: .applicationSupportDirectory, in: .userDomainMask, appropriateFor: nil, create: true)
        return try DatabaseQueue(path: directory.appendingPathComponent("store.sqlite").path)
    }

    /// A private in-memory database, for tests.
    static func inMemory() -> AppDatabase {
        AppDatabase { try DatabaseQueue() }
    }

    private let open: @Sendable () throws -> DatabaseQueue
    private let opened = Mutex<DatabaseQueue?>(nil)

    /// - Parameter open: opens the (unmigrated) database; `queue()` migrates it.
    init(open: @escaping @Sendable () throws -> DatabaseQueue) {
        self.open = open
    }

    /// The open, migrated database.
    func queue() throws -> DatabaseQueue {
        try opened.withLock { cached in
            if let cached { return cached }
            let queue = try open()
            try Self.migrator.migrate(queue)
            cached = queue
            return queue
        }
    }

    /// One migration creating the final schema. Later schema changes add migrations here.
    private static var migrator: DatabaseMigrator {
        var migrator = DatabaseMigrator()
        migrator.registerMigration("v1") { db in
            // Dates are seconds since 1970. `scroll_position` is the fraction scrolled (0–1),
            // the same value the server stores under that name.
            try db.execute(
                sql: """
                    CREATE TABLE articles (
                      id TEXT PRIMARY KEY,
                      url TEXT NOT NULL,
                      title TEXT NOT NULL,
                      description TEXT,
                      image_url TEXT,
                      author TEXT,
                      published_date REAL,
                      reading_time INTEGER,
                      is_read INTEGER NOT NULL DEFAULT 0,
                      is_favorite INTEGER NOT NULL DEFAULT 0,
                      added_at REAL NOT NULL,
                      read_at REAL,
                      scroll_position REAL,
                      body TEXT,
                      content_hash TEXT
                    );
                    """)
            // The offline mutation outbox. Keyed on (article_id, field) so a repeated write to
            // the same field coalesces into one row. Keeps the implicit rowid (no WITHOUT ROWID)
            // so a drain can break created_at ties deterministically.
            try db.execute(
                sql: """
                    CREATE TABLE outbox (
                      article_id TEXT NOT NULL,
                      field TEXT NOT NULL,
                      payload TEXT NOT NULL,
                      created_at INTEGER NOT NULL,
                      attempts INTEGER NOT NULL DEFAULT 0,
                      PRIMARY KEY (article_id, field)
                    );
                    """)
        }
        return migrator
    }
}
