import Foundation
import OSLog
import Synchronization

/// Everything that belongs to one signed-in user (decision 10): the database and the article
/// store, and the work started on them. `SessionModel` creates one at sign-in and closes it at
/// sign-out, which erases the user's on-device data.
nonisolated final class UserSession: Sendable {
    let user: User
    let database: AppDatabase
    let articles: ArticleStore

    private struct Work {
        var closed = false
        var next = 0
        var tasks: [Int: Task<Void, Never>] = [:]
    }

    private let work = Mutex(Work())
    private let logger = Logger(subsystem: "com.cairnapp.cairnreader", category: "session")

    init(user: User, database: AppDatabase? = nil) {
        self.user = user
        self.database = database ?? AppDatabase(url: AppDatabase.url(forUser: user.id))
        articles = ArticleStore(database: self.database)
    }

    /// Runs `operation` as work of this session, cancelled and awaited by `close()`. Does nothing
    /// once the session is closed.
    func start(_ operation: @escaping @Sendable () async -> Void) {
        work.withLock { work in
            guard !work.closed else { return }
            let key = work.next
            work.next += 1
            work.tasks[key] = Task { [weak self] in
                await operation()
                _ = self?.work.withLock { $0.tasks.removeValue(forKey: key) }
            }
        }
    }

    /// Cancels the session's work, waits for it, then erases the database.
    func close() async {
        let tasks = work.withLock { work in
            work.closed = true
            defer { work.tasks = [:] }
            return Array(work.tasks.values)
        }
        for task in tasks { task.cancel() }
        for task in tasks { await task.value }
        do {
            try database.erase()
        } catch {
            logger.error("Closing the database at sign-out failed: \(String(describing: error))")
        }
    }
}
