import Foundation
import Synchronization
import Testing
@testable import CairnReader

// Re-expression of apps/mobile/src/screens/ReadsScreen.test.tsx against `ReadsModel` with an
// in-memory store (decision_adb1: the screen observes the store instead of merging the store,
// the server page and queued saves). ReadsScreen.storeFailure.test.tsx does not port.

/// A server for one user's Reads: list pages by cursor, search results, PATCHes and details,
/// all failing as unreachable while offline.
private final class ReadsServer: Sendable {
    let online = Mutex(true)
    /// Page JSON by cursor; "" is the first page.
    let pages = Mutex<[String: String]>([:])
    let results = Mutex(SyncFixtures.readsPage([]))

    func stub() -> StubServer {
        StubServer { [self] request in
            guard online.withLock({ $0 }) else { return .failure(.notConnectedToInternet) }
            switch (request.httpMethod, request.url?.lastPathComponent) {
            case ("PATCH"?, _):
                return SyncFixtures.ok
            case ("GET"?, "user-1"?):
                let cursor = request.url
                    .flatMap { URLComponents(url: $0, resolvingAgainstBaseURL: false) }?
                    .queryItems?.first { $0.name == "cursor" }?.value ?? ""
                return .json(200, pages.withLock { $0[cursor] } ?? SyncFixtures.readsPage([]))
            case ("GET"?, "search"?):
                return .json(200, results.withLock { $0 })
            case ("GET"?, let id?):
                return .json(200, SyncFixtures.detail(id))
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

private func article(_ id: String, addedAt: TimeInterval) -> Article {
    var article = storedArticle(id)
    article.addedAt = Date(timeIntervalSince1970: addedAt)
    return article
}

@MainActor
private func ids(_ model: ReadsModel) -> [String] {
    model.articles.map(\.id)
}

/// Runs the model's store observation as `ReadsView`'s `.task(id: window)` does: restarted
/// whenever the window changes. Cancel the returned task at the end of the test.
@MainActor
private func follow(_ model: ReadsModel) -> Task<Void, Never> {
    Task {
        while !Task.isCancelled {
            let window = model.window
            let observing = Task { await model.observe(window) }
            while model.window == window, !Task.isCancelled {
                try? await Task.sleep(for: .milliseconds(5))
            }
            observing.cancel()
        }
    }
}

@MainActor
struct ReadsModelTests {
    fileprivate let server = ReadsServer()
    let harness: SyncHarness
    fileprivate let clock = Clock()
    let model: ReadsModel

    init() {
        harness = SyncHarness(server.stub())
        model = ReadsModel(session: harness.session) { [clock] in clock.now }
    }

    private var listRequests: Int {
        harness.requests.count(where: { $0 == "GET user-1" })
    }

    /// "renders stored articles before the network call resolves": the newest 100 stored rows
    /// show before (and without) a sync.
    @Test func storedRowsShowBeforeTheFirstSync() async throws {
        try await harness.session.articles.upsertMany((0...100).map { article("a\($0)", addedAt: 1000 + Double($0)) })
        #expect(model.isLoading)

        let following = follow(model)
        defer { following.cancel() }

        await eventually { ids(model).count == ReadsModel.storedLimit }
        #expect(ids(model).first == "a100")
        #expect(!ids(model).contains("a0"))
        #expect(!model.isLoading)
        #expect(harness.requests.isEmpty)
    }

    @Test func aFailedSyncKeepsTheRowsAndShowsTheCachedBanner() async throws {
        try await harness.session.articles.upsertMany([storedArticle("a1")])
        server.online.withLock { $0 = false }
        let following = follow(model)
        defer { following.cancel() }
        await eventually { ids(model) == ["a1"] }

        await model.appear()

        #expect(ids(model) == ["a1"])
        #expect(model.notice == "Showing cached data — pull to refresh")
        #expect(!model.showsError)
    }

    @Test func aFailedSyncWithNoRowsIsTheErrorState() async {
        server.online.withLock { $0 = false }
        let following = follow(model)
        defer { following.cancel() }

        await model.appear()
        await eventually { !model.isLoading }

        #expect(model.showsError)
        #expect(model.notice == nil)

        // Retry.
        server.online.withLock { $0 = true }
        server.pages.withLock { $0[""] = SyncFixtures.readsPage(["a1"]) }
        await model.refresh()
        await eventually { ids(model) == ["a1"] }
        #expect(!model.showsError)
    }

    /// The jest "queued saves" cases: a queued save is listed with the pending note, and keeps
    /// its row but loses the note once the outbox drains.
    @Test func thePendingNoteFollowsTheOutbox() async throws {
        server.online.withLock { $0 = false }
        let following = follow(model)
        defer { following.cancel() }

        var feedItem = storedArticle("queued")
        feedItem.list = .feed
        try await harness.session.mutations.saveToReads(feedItem)

        await eventually { model.pendingIDs == ["queued"] }
        #expect(ids(model) == ["queued"])

        server.online.withLock { $0 = true }
        await harness.sync.drain()

        await eventually { model.pendingIDs.isEmpty }
        #expect(ids(model) == ["queued"])
    }

    @Test func anArticleArchivedElsewhereDisappearsAfterASync() async {
        server.pages.withLock { $0[""] = SyncFixtures.readsPage(["a1", "a2"]) }
        let following = follow(model)
        defer { following.cancel() }
        await model.appear()
        await eventually { ids(model) == ["a1", "a2"] }

        server.pages.withLock { $0[""] = SyncFixtures.readsPage(["a1"]) }
        await model.refresh()

        await eventually { ids(model) == ["a1"] }
    }

    @Test func theWindowGrowsWithPaging() async throws {
        let first = (0..<20).map { "p\($0)" }
        server.pages.withLock {
            $0[""] = SyncFixtures.readsPage(first, hasMore: true, cursor: "c1")
            $0["c1"] = SyncFixtures.readsPage(["p20"], skip: 20)
        }
        // Older than every page: shown until the first sync narrows the window.
        try await harness.session.articles.upsertMany([storedArticle("old")])
        let following = follow(model)
        defer { following.cancel() }
        await eventually { ids(model) == ["old"] }

        await model.appear()
        guard case .addedSince(.some(_)) = model.window else {
            Issue.record("expected the window to end at the first page")
            return
        }
        await eventually { ids(model) == first }

        await model.loadMore()
        #expect(model.window == .addedSince(nil))
        await eventually { ids(model) == first + ["p20"] }
        #expect(listRequests == 2)

        // After the last page there is nothing more to load.
        await model.loadMore()
        #expect(listRequests == 2)
    }

    @Test func loadMoreDoesNothingBeforeTheFirstSync() async {
        await model.loadMore()
        #expect(harness.requests.isEmpty)
    }

    @Test func appearingSyncsAgainOnlyAfterTheTTL() async {
        await model.appear()
        await model.appear()
        #expect(listRequests == 1)

        clock.now += ReadsModel.refetchInterval + 1
        await model.appear()
        #expect(listRequests == 2)
    }

    @Test func archivingRemovesTheRowThroughTheOutbox() async throws {
        server.pages.withLock { $0[""] = SyncFixtures.readsPage(["a1", "a2"]) }
        let following = follow(model)
        defer { following.cancel() }
        await model.appear()
        await eventually { ids(model) == ["a1", "a2"] }

        let first = try #require(model.articles.first)
        await model.archive(first)

        await eventually { ids(model) == ["a2"] }
        await eventually { harness.requests.contains("PATCH a1") }
    }

    @Test func searchResultsReplaceTheStoredRowsWhileAQueryIsHeld() async throws {
        try await harness.session.articles.upsertMany([storedArticle("a1")])
        server.results.withLock { $0 = SyncFixtures.readsPage(["s1"]) }
        let following = follow(model)
        defer { following.cancel() }
        await eventually { ids(model) == ["a1"] }

        model.search.query = "term"
        await eventually { ids(model) == ["s1"] }
        let search = try #require(server.requests.last?.url?.query())
        #expect(search.contains("list=reads"))

        model.search.query = ""
        #expect(ids(model) == ["a1"])
    }

    @Test func articleAfterAnswersTheNextRow() async throws {
        try await harness.session.articles.upsertMany([article("a1", addedAt: 2), article("a2", addedAt: 1)])
        let following = follow(model)
        defer { following.cancel() }
        await eventually { ids(model) == ["a1", "a2"] }

        #expect(model.article(after: "a1")?.id == "a2")
        #expect(model.article(after: "a2") == nil)
    }
}
