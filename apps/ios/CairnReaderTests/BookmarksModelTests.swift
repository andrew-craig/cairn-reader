import Foundation
import Synchronization
import Testing
@testable import CairnReader

// Port of apps/mobile/src/screens/BookmarksScreen.test.tsx against `BookmarksModel` with an
// in-memory store.

/// A server for one user's favourites; while `held`, a list request waits until it is released.
private final class BookmarksServer: Sendable {
    let online = Mutex(true)
    let held = Mutex(false)
    let page = Mutex(SyncFixtures.readsPage([]))

    func stub() -> StubServer {
        StubServer { [self] request in
            while held.withLock({ $0 }) { Thread.sleep(forTimeInterval: 0.01) }
            guard online.withLock({ $0 }) else { return .failure(.notConnectedToInternet) }
            switch (request.httpMethod, request.url?.lastPathComponent) {
            case ("GET"?, "user-1"?), ("GET"?, "search"?):
                return .json(200, page.withLock { $0 })
            default:
                return .failure(.badServerResponse)
            }
        }
    }
}

private func favorite(_ id: String) -> Article {
    var article = storedArticle(id)
    article.isFavorite = true
    return article
}

@MainActor
private func ids(_ model: BookmarksModel) -> [String] {
    model.list.articles.map(\.id)
}

@MainActor
struct BookmarksModelTests {
    fileprivate let server = BookmarksServer()
    let harness: SyncHarness
    let model: BookmarksModel

    init() {
        harness = SyncHarness(server.stub())
        model = BookmarksModel(session: harness.session)
    }

    @Test func rendersStoredFavoritesWhileTheNetworkCallIsPending() async throws {
        try await harness.session.articles.upsertMany([favorite("fav-1"), storedArticle("plain")])
        server.held.withLock { $0 = true }

        let appearing = Task { await model.appear() }
        await eventually { ids(model) == ["fav-1"] }
        #expect(!model.list.isLoading)

        server.held.withLock { $0 = false }
        await appearing.value
    }

    @Test func doesNotBlankTheListWhenTheNetworkRefreshFails() async throws {
        try await harness.session.articles.upsertMany([favorite("fav-1")])
        server.online.withLock { $0 = false }

        await model.appear()

        #expect(model.list.resetFailed)
        #expect(!model.showsError)
        #expect(ids(model) == ["fav-1"])
    }

    @Test func showsTheErrorStateWhenTheRefreshFailsWithNothingStored() async {
        server.online.withLock { $0 = false }

        await model.appear()

        #expect(model.showsError)
    }

    /// "keeps favorited Feed items out of the offline store", re-expressed: Bookmarks lists
    /// favourites from every list and never writes the store (`Sync` is the only writer of
    /// Reads pages).
    @Test func listsFavoritesFromEveryListWithoutStoringThem() async throws {
        server.page.withLock { $0 = SyncFixtures.readsPage(["feed-1", "feed-2"], isFavorite: true, list: .feed) }

        await model.appear()

        #expect(ids(model) == ["feed-1", "feed-2"])
        let query = try #require(harness.server.requests.last?.url?.query())
        #expect(query.contains("is_favorite=true"))
        #expect(!query.contains("list="))
        #expect(try await harness.session.articles.listFavorites().isEmpty)
        #expect(try await harness.session.articles.getById("feed-1") == nil)
    }

    @Test func searchesEveryList() async throws {
        server.page.withLock { $0 = SyncFixtures.readsPage(["s1"]) }

        model.list.query = "term"

        await eventually { ids(model) == ["s1"] }
        let request = try #require(harness.server.requests.last?.url)
        #expect(request.lastPathComponent == "search")
        #expect(request.query()?.contains("list=") == false)
    }

    @Test func refreshesOnEveryAppearance() async {
        await model.appear()
        await model.appear()

        #expect(harness.requests == ["GET user-1", "GET user-1"])
    }
}
