import Foundation
import Observation
import OSLog

/// A network article list (Feed, Bookmarks) and the search of every tab: cursor paging, refresh,
/// and a debounced live search whose results replace the list while a query is held. Clearing the
/// query shows the list again at once and refreshes it quietly. Ported from the React Native
/// `useCursorArticleList`.
///
/// Responses are applied only while still current: a reset supersedes every earlier page
/// request, and a search response counts only if no later search started and the query did not
/// change since.
@Observable
final class ArticleListModel {
    /// Page size of every list, as in the React Native app.
    static let pageSize = 20
    /// Live search starts this long after the last keystroke.
    static let searchDebounce = Duration.milliseconds(300)

    /// One page of the list; nil for the first.
    typealias FetchPage = @MainActor (_ cursor: String?) async throws -> Page<UserContentResponse>
    typealias Search = @MainActor (_ query: String) async throws -> Page<UserContentResponse>

    /// The search field's text.
    var query = "" {
        didSet { queryChanged(from: oldValue) }
    }

    /// Ids left out of the list and the search results, applied when read, so a row hides (and
    /// comes back) as the set changes.
    var exclude: Set<String> = []

    /// True until the first reset has finished.
    private(set) var isLoading: Bool
    private(set) var isLoadingMore = false
    private(set) var hasMore = false
    /// The last reset failed; the list is whatever it was before.
    private(set) var resetFailed = false
    /// A failed search or load-more, for an alert; set to nil to dismiss.
    var alertMessage: String?

    /// The list's pages.
    private var items: [Article] = []
    /// Results for the held query; nil while none are showing.
    private var results: [Article]?

    @ObservationIgnored private let fetchPage: FetchPage?
    @ObservationIgnored private let search: Search
    @ObservationIgnored private let debounce: Duration
    @ObservationIgnored private var cursor: String?
    /// Bumped by each reset: a page response from an earlier generation is dropped.
    @ObservationIgnored private var generation = 0
    @ObservationIgnored private var isResetting = false
    /// Bumped by each search and query change: an earlier search response is dropped.
    @ObservationIgnored private var searchGeneration = 0
    @ObservationIgnored private var debounceTask: Task<Void, Never>?
    @ObservationIgnored private let logger = Logger(subsystem: "com.cairnapp.cairnreader", category: "list")

    /// `fetchPage` nil is a search-only model (Reads, whose list is the store).
    init(fetchPage: FetchPage?, search: @escaping Search, debounce: Duration = ArticleListModel.searchDebounce) {
        self.fetchPage = fetchPage
        self.search = search
        self.debounce = debounce
        isLoading = fetchPage != nil
    }

    /// Searches `list` (nil: every list) through `read`.
    convenience init(read: ReadService, list: ContentList?, fetchPage: FetchPage?) {
        self.init(fetchPage: fetchPage) { query in
            try await read.searchUserContents(query, list: list, limit: ArticleListModel.pageSize)
        }
    }

    /// The trimmed query, nil when blank.
    var searchQuery: String? { Self.trimmed(query) }

    /// Whether search results replace the list.
    var isShowingResults: Bool { results != nil }

    /// What the list shows: the results for the held query, or the list's pages.
    var articles: [Article] {
        (results ?? items).filter { !exclude.contains($0.id) }
    }

    /// The row after `id` in what the list shows, for the reader's next article.
    func article(after id: String) -> Article? {
        articles.article(after: id)
    }

    /// Reloads the first page, or reruns the search while a query is held. What appearing,
    /// pull-to-refresh and Retry call.
    func refresh() async {
        if let query = searchQuery {
            debounceTask?.cancel()
            await runSearch(query)
        } else {
            await reset()
        }
    }

    /// Appends the next page; does nothing while searching, while a page is loading or after the
    /// last page. What the last row appearing calls.
    func loadMore() async {
        guard let fetchPage, let cursor, hasMore, !isLoadingMore, !isResetting, searchQuery == nil else { return }
        let generation = self.generation
        isLoadingMore = true
        defer { isLoadingMore = false }
        do {
            let page = try await fetchPage(cursor)
            guard generation == self.generation else { return }
            items += page.data.map { Article($0) }
            apply(page.pagination)
        } catch {
            guard generation == self.generation, !(error is CancellationError) else { return }
            logger.error("Loading more articles failed: \(String(describing: error))")
            alertMessage = "Failed to load articles. Please try again."
        }
    }

    /// Takes a row out of the list and the results, e.g. after an archive from search results.
    func remove(_ id: String) {
        items.removeAll { $0.id == id }
        results?.removeAll { $0.id == id }
    }

    // MARK: - Private

    /// Loads the first page. The cursor is kept until the page lands, so a failed reset leaves
    /// load-more continuing the list on screen.
    private func reset() async {
        guard let fetchPage else { return }
        generation += 1
        let generation = self.generation
        isResetting = true
        defer {
            if generation == self.generation {
                isResetting = false
                isLoading = false
            }
        }
        do {
            let page = try await fetchPage(nil)
            guard generation == self.generation else { return }
            items = page.data.map { Article($0) }
            apply(page.pagination)
            resetFailed = false
        } catch {
            guard generation == self.generation, !(error is CancellationError) else { return }
            logger.error("Loading articles failed: \(String(describing: error))")
            resetFailed = true
        }
    }

    private func apply(_ pagination: Page<UserContentResponse>.Pagination) {
        cursor = pagination.cursor
        hasMore = pagination.hasMore
    }

    private func queryChanged(from oldValue: String) {
        guard Self.trimmed(oldValue) != searchQuery else { return }
        debounceTask?.cancel()
        searchGeneration += 1
        guard let query = searchQuery else {
            if results != nil {
                results = nil
                Task { await reset() }
            }
            return
        }
        debounceTask = Task { [debounce] in
            try? await Task.sleep(for: debounce)
            guard !Task.isCancelled else { return }
            await runSearch(query)
        }
    }

    private func runSearch(_ query: String) async {
        searchGeneration += 1
        let generation = searchGeneration
        do {
            let page = try await search(query)
            guard generation == searchGeneration else { return }
            results = page.data.map { Article($0) }
        } catch {
            guard generation == searchGeneration, !(error is CancellationError) else { return }
            logger.error("Searching articles failed: \(String(describing: error))")
            alertMessage = "Failed to search articles. Please try again."
        }
    }

    private static func trimmed(_ text: String) -> String? {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }
}

extension [Article] {
    /// The article after `id`; nil when `id` is last or not listed.
    func article(after id: String) -> Article? {
        guard let index = firstIndex(where: { $0.id == id }), index + 1 < endIndex else { return nil }
        return self[index + 1]
    }
}
