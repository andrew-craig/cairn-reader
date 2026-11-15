import Foundation
import Testing
@testable import CairnReader

// Port of apps/mobile/src/hooks/useCursorArticleList.test.ts against `ArticleListModel`.

private func page(_ ids: [String], cursor: String? = nil, hasMore: Bool = false) -> Page<UserContentResponse> {
    // Fixture JSON: a decoding failure is a broken test, not a case to handle.
    try! HTTPClient.decoder.decode(
        Page<UserContentResponse>.self,
        from: SyncFixtures.readsPage(ids, hasMore: hasMore, cursor: cursor).utf8Data)
}

private struct Offline: Error {}

/// A response the test answers later.
@MainActor
private final class Pending {
    private var continuation: CheckedContinuation<Page<UserContentResponse>, any Error>?
    var isWaiting: Bool { continuation != nil }

    func wait() async throws -> Page<UserContentResponse> {
        try await withCheckedThrowingContinuation { continuation = $0 }
    }

    func resolve(_ page: Page<UserContentResponse>) {
        continuation?.resume(returning: page)
        continuation = nil
    }
}

/// The page and search requests a model makes, answered by `pages` and `results`.
@MainActor
private final class Source {
    private(set) var cursors: [String?] = []
    private(set) var queries: [String] = []
    var pages: (String?) async throws -> Page<UserContentResponse> = { _ in page([]) }
    var results: (String) async throws -> Page<UserContentResponse> = { _ in page([]) }

    func model(paged: Bool = true) -> ArticleListModel {
        let fetchPage: ArticleListModel.FetchPage = { cursor in
            self.cursors.append(cursor)
            return try await self.pages(cursor)
        }
        return ArticleListModel(
            fetchPage: paged ? fetchPage : nil,
            search: { query in
                self.queries.append(query)
                return try await self.results(query)
            },
            debounce: .milliseconds(20))
    }
}

@MainActor
private func ids(_ model: ArticleListModel) -> [String] {
    model.articles.map(\.id)
}

@MainActor
struct ArticleListModelTests {
    @Test func resetsThenAppendsAcrossPagesViaTheCursor() async {
        let source = Source()
        source.pages = { cursor in cursor == nil ? page(["a", "b"], cursor: "cursor-1", hasMore: true) : page(["c"]) }
        let model = source.model()
        #expect(model.isLoading)

        await model.refresh()
        #expect(ids(model) == ["a", "b"])
        #expect(!model.isLoading)

        await model.loadMore()
        #expect(ids(model) == ["a", "b", "c"])
        #expect(source.cursors == [nil, "cursor-1"])
        #expect(!model.hasMore)
    }

    /// The jest case "calls onResetLoaded … and onLoadError on failure": a failed reset is
    /// `resetFailed` (the screen's banner or error state), a failed load-more is an alert.
    @Test func reportsAFailedResetAndAFailedLoadMore() async {
        let source = Source()
        source.pages = { _ in throw Offline() }
        let model = source.model()

        await model.refresh()
        #expect(model.resetFailed)
        #expect(!model.isLoading)

        source.pages = { _ in page(["a"], cursor: "c1", hasMore: true) }
        await model.refresh()
        #expect(!model.resetFailed)

        source.pages = { _ in throw Offline() }
        await model.loadMore()
        #expect(model.alertMessage == "Failed to load articles. Please try again.")
        #expect(ids(model) == ["a"])
    }

    @Test func loadMoreIsANoOpWhileLoadingMoreOrAfterTheLastPage() async {
        let source = Source()
        source.pages = { _ in page(["a"], cursor: "c1", hasMore: true) }
        let model = source.model()
        await model.refresh()

        let pending = Pending()
        source.pages = { _ in try await pending.wait() }
        let first = Task { await model.loadMore() }
        await eventually { pending.isWaiting }
        await model.loadMore()
        #expect(source.cursors == [nil, "c1"])

        pending.resolve(page(["b"]))
        await first.value
        await model.loadMore()
        #expect(source.cursors == [nil, "c1"])
        #expect(ids(model) == ["a", "b"])
    }

    @Test func liveSearchDebouncesTheQueryAndReplacesTheList() async {
        let source = Source()
        source.results = { _ in page(["s1", "s2"]) }
        let model = source.model()
        await model.refresh()

        model.query = "te"
        model.query = "term "

        #expect(model.searchQuery == "term")
        await eventually { ids(model) == ["s1", "s2"] }
        #expect(source.queries == ["term"])
        #expect(model.isShowingResults)
    }

    @Test func scopesSearchToTheListItWasGiven() async throws {
        let server = StubServer { _ in .json(200, SyncFixtures.readsPage(["s1"])) }
        let auth = AuthService(
            client: HTTPClient(
                server: ServerConfig(defaults: UserDefaults(suiteName: UUID().uuidString)!), session: server.session()),
            storage: InMemorySessionStorage(AuthFixtures.valid), deviceID: { "device-1" }, sleep: { _ in })
        let model = ArticleListModel(read: ReadService(auth: auth, userID: "user-1"), list: .feed, fetchPage: nil)

        model.query = "term"

        await eventually { ids(model) == ["s1"] }
        let url = try #require(server.requests.first?.url)
        #expect(url.path() == "/api/v1/content/user/user-1/search")
        let query = Set(url.query()?.split(separator: "&").map(String.init) ?? [])
        #expect(query.isSuperset(of: ["q=term", "list=feed", "limit=20"]))
    }

    @Test func leavesExcludedIDsOutOfPagesAndSearchResults() async {
        let source = Source()
        source.pages = { _ in page(["a", "queued"]) }
        source.results = { _ in page(["s1", "queued"]) }
        let model = source.model()
        model.exclude = ["queued"]

        await model.refresh()
        #expect(ids(model) == ["a"])

        model.query = "term"
        await eventually { ids(model) == ["s1"] }

        // Applied when read: a row comes back once its id leaves the set.
        model.exclude = []
        #expect(ids(model) == ["s1", "queued"])
    }

    /// The jest case "clears the refreshing indicator after a pull-to-refresh during search":
    /// the spinner is `.refreshable`'s, so the refresh must rerun the search and return.
    @Test func aRefreshDuringSearchRerunsTheSearch() async {
        let source = Source()
        source.results = { _ in page(["s1"]) }
        let model = source.model()
        model.query = "term"
        await eventually { ids(model) == ["s1"] }

        source.results = { _ in page(["s2"]) }
        await model.refresh()

        #expect(ids(model) == ["s2"])
        #expect(source.queries == ["term", "term"])
    }

    @Test func clearingSearchRestoresTheListAtOnceThenRefreshesItWithoutASpinner() async {
        let source = Source()
        source.pages = { _ in page(["a", "b"], cursor: "c1", hasMore: true) }
        let model = source.model()
        await model.refresh()

        model.query = "nothing"
        await eventually { model.isShowingResults }
        #expect(ids(model) == [])

        source.pages = { _ in page(["a", "b", "new"], cursor: "c2", hasMore: true) }
        model.query = ""

        #expect(model.searchQuery == nil)
        #expect(ids(model) == ["a", "b"])
        #expect(!model.isLoading)
        await eventually { ids(model) == ["a", "b", "new"] }
        #expect(source.cursors == [nil, nil])
    }

    @Test func aListReloadDuringSearchUpdatesWhatSearchRestoresNotTheResults() async {
        let source = Source()
        source.pages = { _ in page(["a"]) }
        source.results = { _ in page(["s1"]) }
        let model = source.model()
        await model.refresh()

        // A reset that lands while results show.
        let pending = Pending()
        source.pages = { _ in try await pending.wait() }
        let reload = Task { await model.refresh() }
        await eventually { pending.isWaiting }
        model.query = "term"
        await eventually { ids(model) == ["s1"] }
        pending.resolve(page(["a", "fresh"]))
        await reload.value
        #expect(ids(model) == ["s1"])

        source.pages = { _ in try await Pending().wait() }
        model.query = ""
        #expect(ids(model) == ["a", "fresh"])
    }

    @Test func dropsARefreshSearchResponseThatLandsAfterSearchCloses() async {
        let source = Source()
        source.pages = { _ in page(["a", "b"], cursor: "c1", hasMore: true) }
        source.results = { _ in page(["s1"]) }
        let model = source.model()
        await model.refresh()
        model.query = "term"
        await eventually { ids(model) == ["s1"] }

        let pending = Pending()
        source.results = { _ in try await pending.wait() }
        let refresh = Task { await model.refresh() }
        await eventually { pending.isWaiting }
        model.query = ""
        #expect(ids(model) == ["a", "b"])

        pending.resolve(page(["late"]))
        await refresh.value
        #expect(ids(model) == ["a", "b"])
        #expect(!model.isShowingResults)
    }

    @Test func aRefreshInsideTheDebounceStillRestoresTheList() async {
        let source = Source()
        source.pages = { _ in page(["a", "b"], cursor: "c1", hasMore: true) }
        source.results = { _ in page(["s1"]) }
        let model = source.model()
        await model.refresh()

        model.query = "term"
        await model.refresh()
        #expect(ids(model) == ["s1"])
        // Past the debounce.
        try? await Task.sleep(for: .milliseconds(50))

        source.pages = { _ in try await Pending().wait() }
        model.query = ""
        #expect(ids(model) == ["a", "b"])
    }

    @Test func holdsLoadMoreWhileTheQuietReloadAfterSearchIsInFlight() async {
        let source = Source()
        source.pages = { _ in page(["a", "b"], cursor: "c1", hasMore: true) }
        source.results = { _ in page(["s1"]) }
        let model = source.model()
        await model.refresh()
        model.query = "term"
        await eventually { ids(model) == ["s1"] }

        let pending = Pending()
        source.pages = { _ in try await pending.wait() }
        model.query = ""
        await eventually { pending.isWaiting }

        await model.loadMore()
        #expect(source.cursors == [nil, nil])

        pending.resolve(page(["a", "b", "c"], cursor: "c2", hasMore: true))
        await eventually { ids(model) == ["a", "b", "c"] }
        source.pages = { _ in page([]) }
        await model.loadMore()
        #expect(source.cursors == [nil, nil, "c2"])
    }

    /// The jest case "a stale refresh response leaves the spinner for the newer search": the
    /// earlier query's response does not replace the newer query's results.
    @Test func aSearchResponseForAnEarlierQueryIsDropped() async {
        let source = Source()
        source.results = { _ in page(["s1"]) }
        let model = source.model()
        model.query = "old"
        await eventually { ids(model) == ["s1"] }

        let pending = Pending()
        source.results = { _ in try await pending.wait() }
        let refresh = Task { await model.refresh() }
        await eventually { pending.isWaiting }
        source.results = { _ in page(["new"]) }
        model.query = "new"
        await eventually { ids(model) == ["new"] }

        pending.resolve(page(["stale"]))
        await refresh.value
        #expect(ids(model) == ["new"])
    }

    @Test func aFailedQuietReloadLeavesLoadMoreContinuingTheRestoredList() async {
        let source = Source()
        source.pages = { _ in page(["a", "b"], cursor: "c1", hasMore: true) }
        source.results = { _ in page(["s1"]) }
        let model = source.model()
        await model.refresh()
        model.query = "term"
        await eventually { ids(model) == ["s1"] }

        source.pages = { _ in throw Offline() }
        model.query = ""
        await eventually { model.resetFailed }
        #expect(ids(model) == ["a", "b"])

        source.pages = { _ in page(["c"]) }
        await model.loadMore()
        #expect(ids(model) == ["a", "b", "c"])
        #expect(source.cursors.last == "c1")
    }

    @Test func aSearchFailureIsAnAlertAndKeepsTheList() async {
        let source = Source()
        source.pages = { _ in page(["a"]) }
        source.results = { _ in throw Offline() }
        let model = source.model()
        await model.refresh()

        model.query = "term"

        await eventually { model.alertMessage == "Failed to search articles. Please try again." }
        #expect(ids(model) == ["a"])
    }

    // Beyond the jest suite.

    @Test func aSlowEarlierResetDoesNotOverwriteANewerOne() async {
        let source = Source()
        let pending = Pending()
        source.pages = { _ in try await pending.wait() }
        let model = source.model()
        let slow = Task { await model.refresh() }
        await eventually { pending.isWaiting }

        source.pages = { _ in page(["new"]) }
        await model.refresh()
        pending.resolve(page(["old"]))
        await slow.value

        #expect(ids(model) == ["new"])
    }

    @Test func aResetDropsALoadMoreStillInFlight() async {
        let source = Source()
        source.pages = { _ in page(["a"], cursor: "c1", hasMore: true) }
        let model = source.model()
        await model.refresh()

        let pending = Pending()
        source.pages = { _ in try await pending.wait() }
        let more = Task { await model.loadMore() }
        await eventually { pending.isWaiting }
        source.pages = { _ in page(["fresh"]) }
        await model.refresh()
        pending.resolve(page(["stale"]))
        await more.value

        #expect(ids(model) == ["fresh"])
    }

    @Test func aSearchOnlyModelRestoresNothingAndFetchesNoPage() async {
        let source = Source()
        source.results = { _ in page(["s1"]) }
        let model = source.model(paged: false)
        #expect(!model.isLoading)

        model.query = "term"
        await eventually { ids(model) == ["s1"] }
        model.query = ""

        #expect(ids(model) == [])
        #expect(!model.isShowingResults)
        await model.loadMore()
        #expect(source.cursors == [])
    }

    @Test func articleAfterAnswersTheNextShownRow() async {
        let source = Source()
        source.pages = { _ in page(["a", "b", "c"]) }
        let model = source.model()
        await model.refresh()
        model.exclude = ["b"]

        #expect(model.article(after: "a")?.id == "c")
        #expect(model.article(after: "c") == nil)
        #expect(model.article(after: "missing") == nil)
    }
}
