import Foundation
import OSLog
import Synchronization

/// Everything that belongs to one signed-in user (decision 10): the database, the article store,
/// the outbox and mutations on them, a `ReadService` bound to the user, `Sync`, the Feed cache,
/// and the work started on them. `SessionModel` creates one at sign-in and closes it at
/// sign-out, which erases the user's on-device data.
nonisolated final class UserSession: Sendable {
    let user: User
    let database: AppDatabase
    let articles: ArticleStore
    let read: ReadService
    let outbox: Outbox
    let mutations: ArticleMutations
    let sync: Sync
    let feedCache: FeedCache

    private struct Work {
        var closed = false
        var next = 0
        var tasks: [Int: Task<Void, Never>] = [:]
    }

    private let work = Mutex(Work())
    private let logger = Logger(subsystem: "com.cairnapp.cairnreader", category: "session")

    /// `isOnline` is the app's `Connectivity`; prefetch skips its run offline.
    init(
        user: User, auth: AuthService, isOnline: @escaping @Sendable () async -> Bool, database: AppDatabase? = nil
    ) {
        self.user = user
        let database = database ?? AppDatabase(url: AppDatabase.url(forUser: user.id))
        self.database = database
        let articles = ArticleStore(database: database)
        self.articles = articles
        let read = ReadService(auth: auth, userID: user.id)
        self.read = read
        let outbox = Outbox(database: database, read: read)
        self.outbox = outbox
        let sync = Sync(
            outbox: outbox, articles: articles, read: read,
            prefetch: ArticlePrefetch(articles: articles, read: read, isOnline: isOnline))
        self.sync = sync
        mutations = ArticleMutations(database: database) { sync.requestDrain() }
        feedCache = FeedCache(url: FeedCache.url(forUser: user.id))
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

    /// Cancels the session's work and `Sync` (with the outbox drain), waits for them, then erases
    /// the Feed cache and the database.
    func close() async {
        let tasks = work.withLock { work in
            work.closed = true
            defer { work.tasks = [:] }
            return Array(work.tasks.values)
        }
        for task in tasks { task.cancel() }
        // Stop `Sync` first: session work awaiting a queued operation cannot be cancelled
        // through its own task, only by cancelling the operation.
        await sync.stop()
        for task in tasks { await task.value }
        feedCache.clear()
        do {
            try database.erase()
        } catch {
            logger.error("Closing the database at sign-out failed: \(String(describing: error))")
        }
    }
}
