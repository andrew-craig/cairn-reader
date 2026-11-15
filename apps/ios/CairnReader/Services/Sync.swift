import Foundation
import OSLog
import Synchronization

/// Where the next Reads page starts: the server's cursor, and the `added_at` the page before it
/// ended on (the upper bound of the next page's delete window in `ArticleStore.syncPage`).
nonisolated struct ReadsCursor: Hashable, Sendable {
    let cursor: String
    let last: Date
}

/// The signed-in user's traffic around the store: the outbox drain, Reads page syncs and body
/// prefetch. Owned by `UserSession`.
///
/// Operations run one at a time, in the order requested. Actor isolation would not be enough,
/// because actor methods interleave at their awaits: the operations queue. That is what keeps a
/// drain from running between a page's fetch and its `syncPage`, where it could let the page
/// (fetched before the server had a just-drained Save to Reads) delete the saved article.
nonisolated final class Sync: Sendable {
    /// Reads page size, as in the React Native app.
    static let pageSize = 20

    private struct State {
        var stopped = false
        /// Finishes when the last queued operation has.
        var tail: Task<Void, Never>?
        var next = 0
        /// Cancels each queued or running operation, by key.
        var cancels: [Int: @Sendable () -> Void] = [:]
        /// A drain queued but not started, which a new request joins.
        var queuedDrain: Task<Void, any Error>?
        /// A run queued or running, which a new run joins.
        var pendingRun: Task<Void, any Error>?
    }

    private static let logger = Logger(subsystem: "com.cairnapp.cairnreader", category: "sync")

    private let outbox: Outbox
    private let articles: ArticleStore
    private let read: ReadService
    private let prefetch: ArticlePrefetch
    private let queue = Mutex(State())

    init(outbox: Outbox, articles: ArticleStore, read: ReadService, prefetch: ArticlePrefetch) {
        self.outbox = outbox
        self.articles = articles
        self.read = read
        self.prefetch = prefetch
    }

    // MARK: - Operations

    /// Queues a drain and returns at once. What `ArticleMutations` calls after each edit.
    func requestDrain() {
        _ = queueDrain()
    }

    /// Queues a drain and waits for it.
    func drain() async {
        _ = await queueDrain()?.result
    }

    /// Fetches one `list=reads` page and writes it with `ArticleStore.syncPage`; pass the cursor
    /// this returned for the previous page, nil for the first. A first-page sync drains before
    /// the fetch and prefetches after the write. Returns the next page's cursor, nil on the last
    /// page. Throws the fetch's or the store's error, and `CancellationError` once stopped.
    func syncReads(after cursor: ReadsCursor?) async throws -> ReadsCursor? {
        let task = queue.withLock { state in
            enqueue(&state) { [self] () async throws -> ReadsCursor? in
                if cursor == nil { await outbox.drain() }
                let next = try await syncPage(after: cursor)
                if cursor == nil { await prefetchBodies() }
                return next
            }
        }
        guard let task else { throw CancellationError() }
        return try await task.value
    }

    /// Drains, then prefetches, each regardless of how the other went. Fired when the device
    /// comes back online and when the app comes back to the foreground. Joins a run already
    /// queued or running.
    func run() async {
        let task = queue.withLock { state -> Task<Void, any Error>? in
            if let pending = state.pendingRun { return pending }
            let task = enqueue(&state) { [self] in
                defer { queue.withLock { $0.pendingRun = nil } }
                await drainOutbox()
                await prefetchBodies()
            }
            state.pendingRun = task
            return task
        }
        _ = await task?.result
    }

    /// Cancels every queued and running operation and the outbox drain, and waits for them, so
    /// nothing is sent or written afterwards. Later requests do nothing.
    func stop() async {
        let (cancels, tail) = queue.withLock { state in
            state.stopped = true
            return (Array(state.cancels.values), state.tail)
        }
        for cancel in cancels { cancel() }
        await outbox.stop()
        await tail?.value
    }

    // MARK: - Queue

    private func queueDrain() -> Task<Void, any Error>? {
        queue.withLock { state -> Task<Void, any Error>? in
            if let queued = state.queuedDrain { return queued }
            let task = enqueue(&state) { [self] in
                queue.withLock { $0.queuedDrain = nil }
                await drainOutbox()
            }
            state.queuedDrain = task
            return task
        }
    }

    /// Appends `operation`, to start once every earlier operation has finished. Called with the
    /// lock held; nil once stopped.
    private func enqueue<Success: Sendable>(
        _ state: inout State, _ operation: @escaping @Sendable () async throws -> Success
    ) -> Task<Success, any Error>? {
        guard !state.stopped else { return nil }
        let previous = state.tail
        let key = state.next
        state.next += 1
        let task = Task {
            await previous?.value
            try Task.checkCancellation()
            return try await operation()
        }
        state.cancels[key] = { task.cancel() }
        state.tail = Task { [weak self] in
            _ = await task.result
            _ = self?.queue.withLock { $0.cancels.removeValue(forKey: key) }
        }
        return task
    }

    // MARK: - Steps

    /// Drains; when the drain dropped a row, the store may hold an edit the server refused, so a
    /// first-page sync follows at once to put the server's state back.
    private func drainOutbox() async {
        guard await outbox.drain() else { return }
        do {
            _ = try await syncPage(after: nil)
        } catch {
            Self.logger.error("Syncing Reads after a dropped edit failed: \(String(describing: error))")
        }
    }

    private func syncPage(after cursor: ReadsCursor?) async throws -> ReadsCursor? {
        let page = try await read.listUserContents(list: .reads, limit: Self.pageSize, cursor: cursor?.cursor)
        try Task.checkCancellation()
        let items = page.data.map { Article($0) }
        try await articles.syncPage(items, olderThan: cursor?.last, hasMore: page.pagination.hasMore)
        guard page.pagination.hasMore, let next = page.pagination.cursor, let last = items.map(\.addedAt).min()
        else { return nil }
        return ReadsCursor(cursor: next, last: last)
    }

    private func prefetchBodies() async {
        do {
            try await prefetch.run()
        } catch {
            Self.logger.error("Prefetching bodies failed: \(String(describing: error))")
        }
    }
}
