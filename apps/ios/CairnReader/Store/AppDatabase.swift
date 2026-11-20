import Foundation
import GRDB
import os
import SQLite3
import Synchronization

/// The signed-in user's on-device database, shared by `ArticleStore` and the outbox: one file and
/// one migrator, so the two tables can reference each other (the outbox guards in
/// `ArticleStore`'s upsert) without either reaching into the other's internals. Owned by a
/// `UserSession`, which erases it at sign-out.
///
/// The file lives in the App Group container so a future share extension reaches it (decision
/// 11), and is opened as GRDB's database-sharing guide prescribes: a `DatabasePool` (WAL)
/// created under file coordination, persistent WAL, a busy timeout, and suspension while the app
/// is in the background (`setSuspended`).
///
/// Opened on first use, once. The database is a cache plus the outbox, so a file that cannot be
/// opened or migrated is deleted and recreated; if that fails too, every later call throws the
/// error for the rest of the session.
nonisolated final class AppDatabase: Sendable {
    private static let logger = Logger(subsystem: "com.cairnapp.cairnreader", category: "store")

    /// The database file for a user, in the App Group container. One file per user, so a file
    /// left behind by a sign-out that did not finish is never read for another account. The RN
    /// app's `cairnreader.db` is never opened: on-device RN data is not migrated.
    static func url(forUser userID: String) -> URL {
        AppGroup.containerURL.appending(path: "store-\(userID).sqlite")
    }

    /// A private in-memory database, for tests.
    static func inMemory() -> AppDatabase {
        AppDatabase(url: nil) {
            let queue = try DatabaseQueue()
            try migrator.migrate(queue)
            return queue
        }
    }

    /// Stops (`true`) or resumes (`false`) database access in every open database, so the app
    /// holds no SQLite lock while suspended (iOS kills an app that does, `0xDEAD10CC`). Access
    /// attempted while suspended fails with `SQLITE_INTERRUPT` or `SQLITE_ABORT`.
    static func setSuspended(_ suspended: Bool) {
        NotificationCenter.default.post(
            name: suspended ? Database.suspendNotification : Database.resumeNotification, object: nil)
    }

    private struct Closed: Error {}

    /// The file, nil when in memory.
    private let url: URL?
    private let open: @Sendable () throws -> any DatabaseWriter
    private let opened = Mutex<Result<any DatabaseWriter, any Error>?>(nil)

    convenience init(url: URL) {
        self.init(url: url) { try Self.openRecreatingOnFailure(at: url) }
    }

    private init(url: URL?, open: @escaping @Sendable () throws -> any DatabaseWriter) {
        self.url = url
        self.open = open
    }

    /// The open, migrated database.
    func writer() throws -> any DatabaseWriter {
        try opened.withLock { opened in
            if opened == nil { opened = Result { try open() } }
            return try opened!.get()
        }
    }

    /// Closes the database and deletes its files. Every later call throws.
    func erase() throws {
        let writer = opened.withLock { opened in
            defer { opened = .failure(Closed()) }
            return try? opened?.get()
        }
        defer {
            if let url { Self.deleteFiles(at: url) }
        }
        try writer?.close()
    }

    // MARK: - Opening

    private static func openRecreatingOnFailure(at url: URL) throws -> any DatabaseWriter {
        do {
            return try openShared(at: url)
        } catch {
            logger.error("Opening the database failed, recreating it: \(String(describing: error))")
            deleteFiles(at: url)
            return try openShared(at: url)
        }
    }

    /// The database files: the persistent WAL keeps `-wal` and `-shm` next to the database.
    private static func deleteFiles(at url: URL) {
        for suffix in ["", "-wal", "-shm"] {
            try? FileManager.default.removeItem(at: URL(filePath: url.path() + suffix))
        }
    }

    /// GRDB's `openSharedDatabase(at:)`: concurrent setup by several processes is serialized by
    /// file coordination.
    private static func openShared(at url: URL) throws -> DatabasePool {
        var coordinatorError: NSError?
        var result: Result<DatabasePool, any Error>?
        NSFileCoordinator(filePresenter: nil).coordinate(
            writingItemAt: url, options: .forMerging, error: &coordinatorError
        ) { url in
            result = Result { try openPool(at: url) }
        }
        if let coordinatorError { throw coordinatorError }
        return try result!.get()
    }

    private struct SchemaTooRecent: Error {}

    private static func openPool(at url: URL) throws -> DatabasePool {
        var configuration = Configuration()
        // Write transactions wait for another process's write instead of failing.
        configuration.busyMode = .timeout(5)
        configuration.observesSuspensionNotifications = true
        configuration.prepareDatabase { db in
            // Persistent WAL keeps the -wal and -shm files, which read-only connections from
            // another process need.
            if !db.configuration.readonly {
                var flag: CInt = 1
                let code = withUnsafeMutablePointer(to: &flag) { flag in
                    sqlite3_file_control(db.sqliteConnection, nil, SQLITE_FCNTL_PERSIST_WAL, flag)
                }
                guard code == SQLITE_OK else { throw DatabaseError(resultCode: ResultCode(rawValue: code)) }
            }
        }
        let pool = try DatabasePool(path: url.path(), configuration: configuration)
        try migrator.migrate(pool)
        // Written by a newer build (a downgrade): recreated rather than misread.
        if try pool.read(migrator.hasBeenSuperseded) { throw SchemaTooRecent() }
        return pool
    }

    /// One migration creating the schema. It may be edited in place until the app ships; after
    /// that, schema changes add migrations here.
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
                      PRIMARY KEY (article_id, field)
                    );
                    """)
        }
        return migrator
    }
}
