import Foundation
import GRDB
import Observation
import OSLog

/// The Feed tab: online-only `list=feed` pages in an `ArticleListModel`, never written to the
/// store and never prefetched. Stored ids are left out of the pages and the search results: the
/// store holds only Reads, so a stored id has been saved. That one rule hides a row
/// the moment it is saved (from the swipe or the reader), keeps it hidden while the save is
/// queued, and brings it back if the save is rejected and its store row deleted. The last first
/// page, after that exclusion, is kept in `FeedCache` and shown with a stale banner when a
/// refresh fails.
@Observable
final class FeedModel {
    /// Minimum time between refreshes triggered by the tab appearing.
    static let refetchInterval: TimeInterval = 30

    let list: ArticleListModel
    /// The list is the cached page: no page has landed since it was primed.
    private(set) var isShowingCache = false

    @ObservationIgnored private let session: UserSession
    @ObservationIgnored private let now: () -> Date
    @ObservationIgnored private var lastFetchedAt: Date?
    @ObservationIgnored private let logger = Logger(subsystem: "com.cairnapp.cairnreader", category: "feed")

    init(session: UserSession, now: @escaping () -> Date = Date.init) {
        self.session = session
        self.now = now
        let read = session.read
        list = ArticleListModel(read: read, list: .feed) { cursor in
            try await read.listUserContents(list: .feed, limit: ArticleListModel.pageSize, cursor: cursor)
        }
        list.onReset = { [weak self] in self?.pageLoaded() }
    }

    // MARK: - State

    /// The banner over the cached page, or over the last page after a failed refresh.
    var notice: String? {
        (isShowingCache || list.resetFailed) && !list.articles.isEmpty && !list.isShowingResults
            ? "Showing cached data — pull to refresh" : nil
    }

    /// The error state with Retry: a failed refresh with nothing to show.
    var showsError: Bool {
        list.resetFailed && list.articles.isEmpty && !list.isShowingResults
    }

    /// The row after `id` in what the list shows, for the reader's next article.
    func article(after id: String) -> Article? {
        list.article(after: id)
    }

    // MARK: - Actions

    /// Follows the stored ids into the list's `exclude` until cancelled; the view runs it in
    /// `.task`. A change rewrites the cache, so a saved row leaves the cached page too.
    func observeSaved() async {
        do {
            for try await ids in try session.articles.observeStoredIDs() where ids != list.exclude {
                list.exclude = ids
                if !list.isLoading { saveCache() }
            }
        } catch is CancellationError {
        } catch {
            logger.error("Observing stored ids failed: \(String(describing: error))")
        }
    }

    /// Refreshes unless the last page is under `refetchInterval` old or a query is held. With
    /// nothing listed yet, the cached page shows first, so a failed refresh is not a blank tab.
    func appear() async {
        if let lastFetchedAt, now().timeIntervalSince(lastFetchedAt) <= Self.refetchInterval { return }
        guard list.searchQuery == nil else { return }
        if list.articles.isEmpty {
            let cached = session.feedCache.load()
            if !cached.isEmpty {
                list.prime(cached)
                isShowingCache = true
            }
        }
        await list.refresh()
    }

    /// Stores the item as a Reads item and queues the move (`ArticleMutations.saveToReads`); the
    /// row leaves through `exclude`. A refusal is the root's alert (`Outbox.rejections`).
    func saveToReads(_ article: Article) async {
        do {
            try await session.mutations.saveToReads(article)
        } catch {
            logger.error("Saving to Reads failed: \(String(describing: error))")
        }
    }

    // MARK: - Private

    private func pageLoaded() {
        lastFetchedAt = now()
        isShowingCache = false
        saveCache()
    }

    /// Writes the page as listed (after exclusion); not while search results replace it.
    private func saveCache() {
        guard !list.isShowingResults else { return }
        do {
            try session.feedCache.save(Array(list.articles.prefix(ArticleListModel.pageSize)))
        } catch {
            logger.error("Caching the Feed failed: \(String(describing: error))")
        }
    }
}
