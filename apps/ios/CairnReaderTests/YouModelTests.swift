import Foundation
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
