import Foundation
import OSLog

/// Downloads bodies (cleaned HTML) for the newest unread Reads articles that lack one, so they
/// open offline. Run by `Sync`, never by a screen; Feed items are never prefetched.
nonisolated struct ArticlePrefetch: Sendable {
    /// Cap on both the selection and the bodies kept: the 100 most recently added articles.
    static let limit = 100
    /// Requests in flight at once.
    static let concurrency = 3

    private static let logger = Logger(subsystem: "com.cairnapp.cairnreader", category: "prefetch")

    private let articles: ArticleStore
    private let isOnline: @Sendable () async -> Bool
    private let fetchBody: @Sendable (String) async throws -> (body: String, contentHash: String?)?

    init(
        articles: ArticleStore, isOnline: @escaping @Sendable () async -> Bool,
        fetchBody: @escaping @Sendable (String) async throws -> (body: String, contentHash: String?)?
    ) {
        self.articles = articles
        self.isOnline = isOnline
        self.fetchBody = fetchBody
    }

    /// Fetches bodies through `read`'s content detail.
    init(articles: ArticleStore, read: ReadService, isOnline: @escaping @Sendable () async -> Bool) {
        self.init(articles: articles, isOnline: isOnline) { id in
            let detail = Article(try await read.getContentById(id))
            return detail.content.map { ($0, detail.contentHash) }
        }
    }

    /// Does nothing offline. Fetches the candidates' bodies newest first; the first `NetworkError`
    /// (the connection went away) ends the batch, any other failure skips that article. Bodies
    /// outside the cap are evicted however the batch ended. Throws only for a store failure.
    func run() async throws {
        guard await isOnline() else { return }
        let ids = try await articles.listPrefetchCandidates(limit: Self.limit).map(\.id)
        await fetchBodies(ids)
        try await articles.evictBodiesOutsideCap(Self.limit)
    }

    /// Keeps `concurrency` fetches in flight. Once one finds the connection gone no further one
    /// starts; those already in flight finish.
    private func fetchBodies(_ ids: [String]) async {
        await withTaskGroup(of: Bool.self) { group in
            var remaining = ids[...]
            for _ in 0..<Self.concurrency {
                guard let id = remaining.popFirst() else { break }
                group.addTask { await prefetch(id) }
            }
            for await connectionLost in group {
                if connectionLost { remaining.removeAll() }
                if let id = remaining.popFirst() {
                    group.addTask { await prefetch(id) }
                }
            }
        }
    }

    /// Fetches and stores one body. True when the connection is gone or the run was cancelled,
    /// which ends the batch.
    private func prefetch(_ id: String) async -> Bool {
        do {
            if let fetched = try await fetchBody(id) {
                try await articles.saveBody(id: id, body: fetched.body, contentHash: fetched.contentHash)
            }
            return false
        } catch is NetworkError {
            return true
        } catch is CancellationError {
            return true
        } catch {
            Self.logger.error("Prefetching the body of \(id) failed: \(String(describing: error))")
            return false
        }
    }
}
