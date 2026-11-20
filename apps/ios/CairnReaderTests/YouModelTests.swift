import Foundation
import Synchronization
import Testing
@testable import CairnReader

// The stats behaviour of apps/mobile/src/screens/YouScreen.tsx (which has no test of its own).

@MainActor
private final class Clock {
    var now = Date(timeIntervalSince1970: 1_700_000_000)
}

private let countOK = StubServer.Reply.json(200, #"{ "data": { "count": 5 } }"#)
private let subscriptionsOK = StubServer.Reply.json(200, Fixtures.subscriptions)
private let notFound = StubServer.Reply.json(404, AuthFixtures.errorBody("not found"))

/// Answers by endpoint, since the two requests run concurrently and arrive in either order.
private func makeServer(subscriptions: StubServer.Reply, count: StubServer.Reply) -> StubServer {
    StubServer { request in
        request.url?.path().hasSuffix("/count") == true ? count : subscriptions
    }
}

@MainActor
private func makeModel(_ server: StubServer, clock: Clock = Clock()) -> YouModel {
    let config = ServerConfig(defaults: UserDefaults(suiteName: UUID().uuidString)!)
    let auth = AuthService(
        client: HTTPClient(server: config, session: server.session()),
        storage: InMemorySessionStorage(AuthFixtures.valid), deviceID: { "device-1" }, sleep: { _ in })
    return YouModel(read: ReadService(auth: auth, userID: AuthFixtures.user.id)) { clock.now }
}

@MainActor
struct YouModelTests {
    @Test func loadsAllCounts() async {
        let server = makeServer(subscriptions: subscriptionsOK, count: countOK)
        let model = makeModel(server)

        await model.refreshIfStale()

        #expect(model.rssCount == 1)
        #expect(model.newslettersCount == 1)
        #expect(model.bookmarksCount == 5)
        #expect(model.errorMessage == nil)
        #expect(
            Set(server.paths)
                == ["/api/v1/content/user/user-1/subscriptions", "/api/v1/content/user/user-1/count"])
        let countRequest = server.requests.first { $0.url?.path().hasSuffix("/count") == true }
        #expect(countRequest?.url?.query()?.contains("is_favorite=true") == true)
    }

    @Test func subscriptionsFailureStillShowsBookmarks() async {
        let model = makeModel(makeServer(subscriptions: notFound, count: countOK))
        await model.refreshIfStale()
        #expect(model.bookmarksCount == 5)
        #expect(model.rssCount == nil)
        #expect(model.newslettersCount == nil)
        #expect(model.errorMessage == "Failed to load subscription data")
    }

    @Test func bookmarksFailureStillShowsSubscriptions() async {
        let model = makeModel(makeServer(subscriptions: subscriptionsOK, count: notFound))
        await model.refreshIfStale()
        #expect(model.rssCount == 1)
        #expect(model.newslettersCount == 1)
        #expect(model.bookmarksCount == nil)
        #expect(model.errorMessage == "Failed to load bookmark data")
    }

    @Test func bothFailingShowsTheSubscriptionsError() async {
        let model = makeModel(makeServer(subscriptions: notFound, count: notFound))
        await model.refreshIfStale()
        #expect(model.errorMessage == "Failed to load subscription data")
    }

    @Test func ttlPreventsRefetch() async {
        let server = makeServer(subscriptions: subscriptionsOK, count: countOK)
        let clock = Clock()
        let model = makeModel(server, clock: clock)

        await model.refreshIfStale()
        #expect(server.requests.count == 2)

        clock.now += YouModel.refetchInterval
        await model.refreshIfStale()
        #expect(server.requests.count == 2)

        clock.now += 1
        await model.refreshIfStale()
        #expect(server.requests.count == 4)
    }

    /// Only a fully successful fetch starts the clock, so a partial failure is retried at once.
    @Test func partialFailureDoesNotStartTheClock() async {
        let server = makeServer(subscriptions: subscriptionsOK, count: notFound)
        let model = makeModel(server)

        await model.refreshIfStale()
        await model.refreshIfStale()

        #expect(server.requests.count == 4)
        #expect(model.rssCount == 1)
    }
}

/// One user's server for the favourite tests: subscriptions, and a bookmarks count that each
/// accepted favourite PATCH raises by one; everything fails as unreachable while offline.
private final class BookmarksServer: Sendable {
    let online = Mutex(true)
    let count = Mutex(5)

    func stub() -> StubServer {
        StubServer { [self] request in
            guard online.withLock({ $0 }) else { return .failure(.notConnectedToInternet) }
            switch (request.httpMethod, request.url?.lastPathComponent) {
            case ("PATCH"?, _):
                if request.jsonBody["is_favorite"] as? Bool == true { count.withLock { $0 += 1 } }
                return SyncFixtures.ok
            case ("GET"?, "count"?):
                return .json(200, #"{ "data": { "count": \#(count.withLock { $0 }) } }"#)
            default:
                return subscriptionsOK
            }
        }
    }
}

/// The Bookmarks count after a favourite edit, through the session's outbox and `Sync`:
/// the edit is only queued, so the count can change once the drain has sent it.
@MainActor
struct YouModelFavoriteTests {
    fileprivate let server = BookmarksServer()
    let harness: SyncHarness
    fileprivate let clock = Clock()
    let model: YouModel

    init() {
        harness = SyncHarness(server.stub())
        model = YouModel(read: harness.session.read) { [clock] in clock.now }
    }

    /// Follows the outbox's favourite changes as `MainTabs`' `.task` does.
    private func followingFavoriteChanges() -> Task<Void, Never> {
        Task { await model.followFavoriteChanges(harness.session.outbox.favoriteChanges) }
    }

    /// Stops `Sync`, which ends `favoriteChanges`, and waits until `following` has taken every
    /// change the drains announced.
    private func settle(_ following: Task<Void, Never>) async {
        await harness.sync.stop()
        await following.value
    }

    @Test func favoriteSentByTheDrainIsCountedWhenYouAppearsWithinTheInterval() async throws {
        let following = followingFavoriteChanges()
        await model.refreshIfStale()
        #expect(model.bookmarksCount == 5)

        try await harness.session.mutations.setFavorite("a1", isFavorite: true)
        await harness.sync.drain()
        #expect(harness.requests.contains("PATCH a1"))
        await settle(following)

        clock.now += 1
        await model.refreshIfStale()
        #expect(model.bookmarksCount == 6)
    }

    @Test func favoriteMadeOfflineIsCountedOnceSentAfterComingBackOnline() async throws {
        let following = followingFavoriteChanges()
        await model.refreshIfStale()

        server.online.withLock { $0 = false }
        try await harness.session.mutations.setFavorite("a1", isFavorite: true)
        await harness.sync.drain()
        clock.now += 1
        await model.refreshIfStale()
        // Not adjusted on the device: the server has not seen the edit.
        #expect(model.bookmarksCount == 5)
        #expect(try harness.outboxCount() == 1)

        server.online.withLock { $0 = true }
        await harness.sync.run()
        #expect(try harness.outboxCount() == 0)
        await settle(following)

        clock.now += 1
        await model.refreshIfStale()
        #expect(model.bookmarksCount == 6)
    }

    @Test func favoriteSentWhileYouIsVisibleRefetchesAtOnce() async throws {
        let following = followingFavoriteChanges()
        defer { following.cancel() }
        await model.refreshIfStale()
        model.isVisible = true

        try await harness.session.mutations.setFavorite("a1", isFavorite: true)
        await harness.sync.drain()

        await eventually { model.bookmarksCount == 6 }
    }

    @Test func otherEditsDoNotMakeTheCountsStale() async throws {
        let following = followingFavoriteChanges()
        await model.refreshIfStale()

        try await harness.session.mutations.saveScrollPosition("a1", fraction: 0.5)
        await harness.sync.drain()
        await settle(following)

        #expect(harness.requests.contains("PATCH a1"))
        server.count.withLock { $0 = 6 }

        clock.now += 1
        await model.refreshIfStale()
        #expect(model.bookmarksCount == 5)
    }
}
