import Foundation
import Synchronization
import Testing
@testable import CairnReader

// Re-expression of apps/mobile/src/screens/FeedScreen.test.tsx against `FeedModel` with an
// in-memory store (decision_adb1: a saved row is hidden by the store's ids, not by
// `Outbox.pendingSaveIds` and a put-back on rejection).

/// A server for one user's Feed: `list=feed` pages, an empty Reads page for the sync after a
/// dropped outbox row, search results and PATCHes, all failing as unreachable while offline.
private final class FeedServer: Sendable {
    let online = Mutex(true)
    let feed = Mutex(SyncFixtures.readsPage([], list: .feed))
    let results = Mutex(SyncFixtures.readsPage([], list: .feed))
    let patch = Mutex(SyncFixtures.ok)

    func stub() -> StubServer {
        StubServer { [self] request in
            guard online.withLock({ $0 }) else { return .failure(.notConnectedToInternet) }
            switch (request.httpMethod, request.url?.lastPathComponent) {
            case ("PATCH"?, _):
                return patch.withLock { $0 }
            case ("GET"?, "search"?):
                return .json(200, results.withLock { $0 })
            case ("GET"?, "user-1"?):
                let isFeed = request.url?.query()?.contains("list=feed") == true
                return .json(200, isFeed ? feed.withLock { $0 } : SyncFixtures.readsPage([]))
            default:
                return .failure(.badServerResponse)
            }
        }
    }
}

@MainActor
private final class Clock {
    var now = Date(timeIntervalSince1970: 1_700_000_000)
}

private func feedArticle(_ id: String) -> Article {
    var article = storedArticle(id)
    article.list = .feed
    return article
}

@MainActor
private func ids(_ model: FeedModel) -> [String] {
    model.list.articles.map(\.id)
}

@MainActor
struct FeedModelTests {
    fileprivate let server = FeedServer()
    let harness: SyncHarness
    fileprivate let clock = Clock()
    let model: FeedModel

    init() {
        harness = SyncHarness(server.stub())
        model = FeedModel(session: harness.session) { [clock] in clock.now }
    }

    private var feedRequests: [URLRequest] {
        harness.server.requests.filter { $0.url?.query()?.contains("list=feed") == true }
    }

    private var cachedIDs: [String] {
        harness.session.feedCache.load().map(\.id)
    }

    /// Runs the stored-id observation as `FeedView`'s `.task` does; cancel it at the end.
    private func observing() -> Task<Void, Never> {
        Task { await model.observeSaved() }
    }

    /// "fetches only the feed list" and "does not write fetched items to the offline Reads
    /// store": nothing is stored and nothing is prefetched.
    @Test func fetchesOnlyTheFeedListAndNeverStoresIt() async throws {
        server.feed.withLock { $0 = SyncFixtures.readsPage(["a", "b"], list: .feed) }

        await model.appear()

        #expect(ids(model) == ["a", "b"])
        #expect(feedRequests.count == 1)
        #expect(harness.requests == ["GET user-1"])
        #expect(try await harness.session.articles.getById("a") == nil)
    }

    @Test func cachesTheFetchedPage() async {
        server.feed.withLock { $0 = SyncFixtures.readsPage(["a"], list: .feed) }

        await model.appear()

        #expect(cachedIDs == ["a"])
        #expect(model.notice == nil)
    }

    @Test func showsTheCachedPageWithAStaleBannerWhenTheRefreshFails() async throws {
        try harness.session.feedCache.save([feedArticle("old")])
        server.online.withLock { $0 = false }

        await model.appear()

        #expect(ids(model) == ["old"])
        #expect(model.notice == "Showing cached data — pull to refresh")
        #expect(!model.showsError)
    }

    @Test func showsTheErrorStateWithRetryWhenNothingIsCached() async {
        server.online.withLock { $0 = false }

        await model.appear()

        #expect(model.showsError)
        #expect(model.notice == nil)

        // Retry.
        server.online.withLock { $0 = true }
        server.feed.withLock { $0 = SyncFixtures.readsPage(["a"], list: .feed) }
        await model.list.refresh()
        #expect(ids(model) == ["a"])
        #expect(!model.showsError)
    }

    @Test func anEmptyFeedIsNeitherLoadingNorAnError() async {
        await model.appear()

        #expect(model.list.articles.isEmpty)
        #expect(!model.list.isLoading)
        #expect(!model.showsError)
    }

    /// "offline: queues the save and drops the row from the Feed and its cache".
    @Test func anOfflineSaveHidesTheRowAndQueuesIt() async throws {
        server.feed.withLock { $0 = SyncFixtures.readsPage(["a", "b"], list: .feed) }
        let observation = observing()
        defer { observation.cancel() }
        await model.appear()
        server.online.withLock { $0 = false }

        let first = try #require(model.list.articles.first)
        await model.saveToReads(first)

        await eventually { ids(model) == ["b"] }
        await eventually { cachedIDs == ["b"] }
        #expect(try harness.outboxCount() == 1)
        #expect(try await harness.session.articles.getById("a")?.list == .reads)
    }

    /// "keeps the row … when Save to Reads is rejected": the drain drops the `list` row and
    /// deletes the store row, so the row is listed again.
    @Test func aRejectedSaveShowsTheRowAgain() async throws {
        server.feed.withLock { $0 = SyncFixtures.readsPage(["a", "b"], list: .feed) }
        server.patch.withLock { $0 = .json(404, AuthFixtures.errorBody("gone")) }
        let observation = observing()
        defer { observation.cancel() }
        await model.appear()

        let first = try #require(model.list.articles.first)
        await model.saveToReads(first)
        await harness.sync.drain()

        #expect(try harness.outboxCount() == 0)
        #expect(try await harness.session.articles.getById("a") == nil)
        await eventually { ids(model) == ["a", "b"] }
    }

    /// "leaves out items with a queued Save to Reads": every stored id is left out of the pages,
    /// the cached page and the search results.
    @Test func storedIDsAreLeftOutOfPagesAndSearch() async throws {
        try await harness.session.articles.upsertMany([storedArticle("a")])
        server.feed.withLock { $0 = SyncFixtures.readsPage(["a", "b"], list: .feed) }
        server.results.withLock { $0 = SyncFixtures.readsPage(["a", "c"], list: .feed) }
        let observation = observing()
        defer { observation.cancel() }
        await eventually { model.list.exclude == ["a"] }

        await model.appear()
        #expect(ids(model) == ["b"])
        #expect(cachedIDs == ["b"])

        model.list.query = "term"
        await eventually { ids(model) == ["c"] }
    }

    /// "does not overwrite the cached Feed page with search results when swiping during search".
    @Test func savingDuringSearchLeavesTheCachedPage() async throws {
        server.feed.withLock { $0 = SyncFixtures.readsPage(["a", "b"], list: .feed) }
        server.results.withLock { $0 = SyncFixtures.readsPage(["b"], list: .feed) }
        let observation = observing()
        defer { observation.cancel() }
        await model.appear()
        #expect(cachedIDs == ["a", "b"])

        model.list.query = "b"
        await eventually { ids(model) == ["b"] }
        let result = try #require(model.list.articles.first)
        await model.saveToReads(result)

        await eventually { ids(model).isEmpty }
        #expect(cachedIDs == ["a", "b"])
    }

    /// The quiet refresh after a search is cleared caches its page too (`onReset`).
    @Test func clearingASearchRecachesTheRefreshedPage() async {
        server.feed.withLock { $0 = SyncFixtures.readsPage(["a"], list: .feed) }
        server.results.withLock { $0 = SyncFixtures.readsPage(["s"], list: .feed) }
        await model.appear()
        model.list.query = "s"
        await eventually { ids(model) == ["s"] }

        server.feed.withLock { $0 = SyncFixtures.readsPage(["c"], list: .feed) }
        model.list.query = ""

        await eventually { cachedIDs == ["c"] }
        #expect(ids(model) == ["c"])
    }

    @Test func appearingRefreshesAgainOnlyAfterTheTTL() async {
        await model.appear()
        await model.appear()
        #expect(feedRequests.count == 1)

        clock.now += FeedModel.refetchInterval + 1
        await model.appear()
        #expect(feedRequests.count == 2)
    }

    @Test func articleAfterAnswersTheNextRow() async {
        server.feed.withLock { $0 = SyncFixtures.readsPage(["a", "b"], list: .feed) }
        await model.appear()

        #expect(model.article(after: "a")?.id == "b")
        #expect(model.article(after: "b") == nil)
    }
}
