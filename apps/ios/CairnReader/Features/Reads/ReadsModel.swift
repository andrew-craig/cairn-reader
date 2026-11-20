import Foundation
import GRDB
import Observation
import OSLog

/// The Reads tab (decision 8): its rows are the store's, followed through
/// `ArticleStore.observeReads`; syncing writes pages into the store through `Sync` and widens the
/// window of stored rows shown. It keeps no list of its own, so a row archived, saved or changed
/// anywhere shows here through the store. Search results (`ArticleListModel`, `list=reads`)
/// replace the rows while a query is held.
@Observable
final class ReadsModel {
    /// Stored rows shown before the first page has synced.
    static let storedLimit = 100
    /// Minimum time between syncs triggered by the tab appearing.
    static let refetchInterval: TimeInterval = 30

    /// Which stored rows are shown.
    nonisolated enum Window: Hashable, Sendable {
        /// The newest `n`: nothing has synced yet.
        case newest(Int)
        /// Every row added at or after the date: the oldest synced page's last `addedAt`. Nil once
        /// the last page has synced: every row.
        case addedSince(Date?)
    }

    let search: ArticleListModel
    private(set) var window = Window.newest(ReadsModel.storedLimit)
    /// The rows in the window, as last observed.
    private(set) var entries: [ReadsEntry] = []
    /// The last sync (or the store) failed; the rows are what the store had.
    private(set) var syncFailed = false
    private(set) var isSyncingMore = false

    private var isObserved = false
    private var isSyncingFirstPage = false

    @ObservationIgnored private let session: UserSession
    @ObservationIgnored private let now: () -> Date
    @ObservationIgnored private var lastSyncedAt: Date?
    /// Where the next page starts; nil before the first sync and after the last page.
    @ObservationIgnored private var cursor: ReadsCursor?
    /// Bumped by each first-page sync: a page sync from an earlier generation is not applied.
    @ObservationIgnored private var generation = 0
    @ObservationIgnored private let logger = Logger(subsystem: "com.cairnapp.cairnreader", category: "reads")

    init(session: UserSession, now: @escaping () -> Date = Date.init) {
        self.session = session
        self.now = now
        search = ArticleListModel(read: session.read, list: .reads, fetchPage: nil)
    }

    // MARK: - State

    /// What the list shows: the search results while they replace the rows, else the rows.
    var articles: [Article] {
        search.isShowingResults ? search.articles : entries.map(\.article)
    }

    /// Ids with a queued Save to Reads.
    var pendingIDs: Set<String> {
        Set(entries.filter(\.pendingSave).map(\.id))
    }

    /// Nothing to show yet: the store has not answered, or it is empty and the first sync runs.
    var isLoading: Bool {
        guard entries.isEmpty else { return false }
        if !isObserved { return true }
        guard case .newest = window else { return false }
        return isSyncingFirstPage
    }

    /// The banner over cached rows after a failed sync.
    var notice: String? {
        syncFailed && !entries.isEmpty && !search.isShowingResults ? "Showing cached data — pull to refresh" : nil
    }

    /// The error state with Retry: a failed sync with no rows to show.
    var showsError: Bool {
        syncFailed && entries.isEmpty && !search.isShowingResults
    }

    /// The row after `id` in what the list shows, for the reader's next article.
    func article(after id: String) -> Article? {
        articles.article(after: id)
    }

    // MARK: - Actions

    /// Follows the store's rows in `window` until cancelled; the view runs it in
    /// `.task(id: window)`, so it restarts when the window changes.
    func observe(_ window: Window) async {
        do {
            let (limit, since): (Int?, Date?) =
                switch window {
                case let .newest(count): (count, nil)
                case let .addedSince(date): (nil, date)
                }
            for try await entries in try session.articles.observeReads(limit: limit, addedSince: since) {
                self.entries = entries
                isObserved = true
            }
        } catch is CancellationError {
        } catch {
            logger.error("Observing Reads failed: \(String(describing: error))")
            isObserved = true
            syncFailed = true
        }
    }

    /// Syncs the first page unless the last sync is under `refetchInterval` old or a query is held.
    func appear() async {
        if let lastSyncedAt, now().timeIntervalSince(lastSyncedAt) <= Self.refetchInterval { return }
        guard search.searchQuery == nil else { return }
        await syncFirstPage()
    }

    /// Pull-to-refresh and Retry: reruns the search while a query is held, else syncs the first page.
    func refresh() async {
        if search.searchQuery != nil {
            await search.refresh()
        } else {
            await syncFirstPage()
        }
    }

    /// Syncs the next page, widening the window. What the last row appearing calls; does nothing
    /// while searching, while a sync runs, before the first sync and after the last page.
    func loadMore() async {
        guard search.searchQuery == nil, !isSyncingMore, !isSyncingFirstPage, let cursor else { return }
        let generation = self.generation
        isSyncingMore = true
        defer { isSyncingMore = false }
        do {
            let next = try await session.sync.syncReads(after: cursor)
            guard generation == self.generation else { return }
            synced(next)
        } catch {
            guard generation == self.generation else { return }
            failed(error)
        }
    }

    /// Archives through the outbox. The row leaves when the store row does; a search result is
    /// taken out of the results. A refusal is the root's alert (`Outbox.rejections`).
    func archive(_ article: Article) async {
        do {
            try await session.mutations.archive(article.id)
            search.remove(article.id)
        } catch {
            logger.error("Archiving failed: \(String(describing: error))")
        }
    }

    // MARK: - Private

    private func syncFirstPage() async {
        generation += 1
        let generation = self.generation
        isSyncingFirstPage = true
        defer {
            if generation == self.generation { isSyncingFirstPage = false }
        }
        do {
            let next = try await session.sync.syncReads(after: nil)
            guard generation == self.generation else { return }
            lastSyncedAt = now()
            synced(next)
        } catch {
            guard generation == self.generation else { return }
            failed(error)
        }
    }

    private func synced(_ next: ReadsCursor?) {
        cursor = next
        window = .addedSince(next?.last)
        syncFailed = false
    }

    private func failed(_ error: any Error) {
        // `Sync` stopped: the session is closing.
        guard !(error is CancellationError) else { return }
        logger.error("Syncing Reads failed: \(String(describing: error))")
        syncFailed = true
    }
}
