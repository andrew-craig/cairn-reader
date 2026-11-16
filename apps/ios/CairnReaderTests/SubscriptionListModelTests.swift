import Foundation
import Synchronization
import Testing
@testable import CairnReader

// Port of apps/mobile/src/components/SubscriptionListScreen.test.tsx and
// apps/mobile/src/screens/RssScreen.test.tsx against `SubscriptionListModel`, plus the unsubscribe
// flow, a failed load and the newsletter address.

private let goBlog = """
    { "id": "sub-rss", "type": "rss", "title": "Go Blog", "subscribed_at": "2025-01-01T00:00:00Z", "list": "reads",
      "rss_data": { "feed_id": "feed-1", "feed_url": "https://go.dev/feed" } }
    """
private let goBlogInFeed = goBlog.replacingOccurrences(of: #""list": "reads""#, with: #""list": "feed""#)
private let rustBlog = """
    { "id": "sub-rust", "type": "rss", "title": "Rust Blog", "subscribed_at": "2025-01-01T00:00:00Z",
      "list": "reads", "rss_data": { "feed_id": "feed-2", "feed_url": "https://blog.rust-lang.org/feed.xml" } }
    """
private let weeklyLetter = """
    { "id": "sender-1", "type": "email", "title": "Weekly Letter", "subscribed_at": "2025-01-01T00:00:00Z",
      "list": "reads", "email_data": { "email_address": "letter@example.com" } }
    """
/// An RSS row without its feed: the list endpoint cannot route it.
private let unroutable = """
    { "id": "sub-x", "type": "rss", "title": "Some Account", "subscribed_at": "2025-01-01T00:00:00Z", "list": "reads" }
    """

private func subscriptions(_ items: [String]) -> StubServer.Reply {
    .json(
        200,
        #"{ "data": { "subscriptions": [\#(items.joined(separator: ","))], "total_count": \#(items.count), "failed_sources": [] } }"#
    )
}

private let ok = StubServer.Reply.json(200, #"{ "data": { "type": "rss", "key": "k", "list": "feed" } }"#)
private let serverError = StubServer.Reply.json(500, AuthFixtures.errorBody("boom"))

/// Answers the list with `list`, and every other request with `other`.
private func makeServer(
    _ list: StubServer.Reply, other: @escaping @Sendable (URLRequest) -> StubServer.Reply = { _ in ok }
) -> StubServer {
    StubServer { request in
        request.httpMethod == "GET" && request.url?.path().hasSuffix("/subscriptions") == true ? list : other(request)
    }
}

@MainActor
private func makeRead(_ server: StubServer) -> ReadService {
    let config = ServerConfig(defaults: UserDefaults(suiteName: UUID().uuidString)!)
    let auth = AuthService(
        client: HTTPClient(server: config, session: server.session()),
        storage: InMemorySessionStorage(AuthFixtures.valid), deviceID: { "device-1" }, sleep: { _ in })
    return ReadService(auth: auth, userID: AuthFixtures.user.id)
}

@MainActor
private func loaded(_ server: StubServer, _ kind: SubscriptionListModel.Kind = .rss) async -> SubscriptionListModel {
    let model = SubscriptionListModel(read: makeRead(server), kind: kind)
    await model.load()
    return model
}

private extension StubServer {
    var writes: [URLRequest] { requests.filter { $0.httpMethod != "GET" } }
}

@MainActor
struct SubscriptionListModelTests {
    // MARK: SubscriptionListScreen.test.tsx

    /// Jest: "shows each source's current list and says the choice only affects new items".
    @Test func showsEachSourcesListAndTheRoutingNote() async {
        let model = await loaded(makeServer(subscriptions([goBlogInFeed, rustBlog])))

        #expect(model.subscriptions.map(\.list) == [.feed, .reads])
        #expect(model.isRoutable)
        #expect(!model.isLoading)
    }

    /// The RSS and Newsletters screens' filters and subtitles.
    @Test func filtersByType() async {
        let server = makeServer(subscriptions([goBlog, weeklyLetter]))

        let rss = await loaded(server, .rss)
        #expect(rss.subscriptions.map(\.title) == ["Go Blog"])
        #expect(rss.subtitle(for: rss.subscriptions[0]) == "https://go.dev/feed")

        let newsletters = await loaded(server, .newsletters)
        #expect(newsletters.subscriptions.map(\.title) == ["Weekly Letter"])
        #expect(newsletters.subtitle(for: newsletters.subscriptions[0]) == "letter@example.com")
    }

    /// Jest: "routes an RSS feed by feed_id and reflects the change immediately".
    @Test func routesAnRSSFeedByFeedIDAndReflectsTheChangeAtOnce() async {
        let gate = DispatchSemaphore(value: 0)
        let server = makeServer(subscriptions([goBlog])) { _ in
            gate.wait()
            return ok
        }
        let model = await loaded(server)

        let change = Task { await model.setList(.feed, for: model.subscriptions[0]) }
        await eventually { !server.writes.isEmpty }
        #expect(model.subscriptions[0].list == .feed)
        gate.signal()
        await change.value

        #expect(model.subscriptions[0].list == .feed)
        #expect(server.writes.map { $0.url?.path() } == ["/api/v1/content/user/user-1/subscriptions/rss/feed-1/list"])
        #expect(server.writes[0].httpMethod == "PUT")
        #expect(server.writes[0].jsonBody["list"] as? String == "feed")
        #expect(model.errorMessage == nil)
    }

    /// Jest: "routes an email sender by its subscription id".
    @Test func routesAnEmailSenderBySubscriptionID() async {
        let server = makeServer(subscriptions([weeklyLetter]))
        let model = await loaded(server, .newsletters)

        await model.setList(.feed, for: model.subscriptions[0])

        #expect(
            server.writes.map { $0.url?.path() } == ["/api/v1/content/user/user-1/subscriptions/email/sender-1/list"])
        #expect(server.writes[0].jsonBody["list"] as? String == "feed")
    }

    /// Jest: "does nothing when the already-selected list is pressed".
    @Test func pickingTheCurrentListDoesNothing() async {
        let server = makeServer(subscriptions([goBlog]))
        let model = await loaded(server)

        await model.setList(.reads, for: model.subscriptions[0])

        #expect(server.writes.isEmpty)
    }

    /// Jest: "rolls back and alerts when the update fails".
    @Test func rollsBackAndAlertsWhenTheUpdateFails() async {
        let server = makeServer(subscriptions([goBlog])) { _ in serverError }
        let model = await loaded(server)

        await model.setList(.feed, for: model.subscriptions[0])

        #expect(model.subscriptions[0].list == .reads)
        #expect(model.errorMessage == "Failed to update. Please try again.")
    }

    /// Jest: "offers no toggle for source types the backend cannot route". The app has no social
    /// type (the backend returns none), so the unroutable source is an RSS row without its feed.
    @Test func offersNoChoiceForASourceThatCannotBeRouted() async {
        let server = makeServer(subscriptions([unroutable]))
        let model = await loaded(server)

        #expect(SubscriptionListModel.route(for: model.subscriptions[0]) == nil)
        #expect(!model.isRoutable)
        await model.setList(.feed, for: model.subscriptions[0])
        #expect(server.writes.isEmpty)
    }

    // MARK: RssScreen.test.tsx

    /// Jest: "subscribes to the chosen list and shows the new feed without leaving the page": the
    /// add-link sheet in feed mode on the same `ReadService`, then its `onAdded` reload.
    @Test func addingAFeedSubscribesToTheChosenListAndReloads() async {
        let listed = Mutex(0)
        let server = StubServer { request in
            switch (request.httpMethod, request.url?.path()) {
            case ("GET"?, "/api/v1/content/user/user-1/subscriptions"?):
                let count = listed.withLock {
                    $0 += 1
                    return $0
                }
                return count == 1 ? subscriptions([]) : subscriptions([goBlogInFeed])
            case (_, "/api/v1/content/detect"?):
                return .json(200, #"{ "data": { "url": "https://go.dev/feed", "type": "feed", "title": "Go Blog" } }"#)
            default:
                return .json(200, Fixtures.addFeed)
            }
        }
        let read = makeRead(server)
        let model = SubscriptionListModel(read: read, kind: .rss)
        await model.load()
        #expect(model.subscriptions.isEmpty)

        let sheet = AddLinkModel(read: read, mode: .feed)
        sheet.url = "https://go.dev/feed"
        await eventually { sheet.isFeed && !sheet.isDetecting }
        sheet.list = .feed
        await sheet.addURL()
        #expect(sheet.successMessage != nil)
        await model.load()

        let add = server.requests.first { $0.httpMethod == "POST" && $0.url?.path() == "/api/v1/content/user/user-1" }
        #expect(add?.jsonBody["url"] as? String == "https://go.dev/feed")
        #expect(add?.jsonBody["type"] as? String == "feed")
        #expect(add?.jsonBody["list"] as? String == "feed")
        #expect(model.subscriptions.map(\.title) == ["Go Blog"])
        #expect(listed.withLock { $0 } == 2)
    }

    // MARK: Beyond the jest suites

    @Test func unsubscribeRemovesTheFeed() async {
        let server = makeServer(subscriptions([goBlog, rustBlog]))
        let model = await loaded(server)

        await model.unsubscribe(model.subscriptions[0])

        #expect(model.subscriptions.map(\.title) == ["Rust Blog"])
        #expect(server.writes.map { $0.httpMethod } == ["DELETE"])
        #expect(server.writes.map { $0.url?.path() } == ["/api/v1/content/user/user-1/subscriptions/rss/feed-1"])
        #expect(model.errorMessage == nil)
    }

    @Test func aFailedUnsubscribePutsTheFeedBackAndAlerts() async {
        let server = makeServer(subscriptions([goBlog, rustBlog])) { _ in serverError }
        let model = await loaded(server)

        await model.unsubscribe(model.subscriptions[0])

        #expect(model.subscriptions.map(\.title) == ["Go Blog", "Rust Blog"])
        #expect(model.errorMessage == "Failed to unsubscribe. Please try again.")
    }

    /// Only RSS feeds offer the unsubscribe action.
    @Test func newslettersCannotBeUnsubscribed() async {
        let server = makeServer(subscriptions([goBlog, weeklyLetter]))
        let rss = await loaded(server, .rss)
        let newsletters = await loaded(server, .newsletters)

        #expect(SubscriptionListModel.canUnsubscribe(rss.subscriptions[0]))
        #expect(!SubscriptionListModel.canUnsubscribe(newsletters.subscriptions[0]))
        await newsletters.unsubscribe(newsletters.subscriptions[0])
        #expect(server.writes.isEmpty)
        #expect(newsletters.subscriptions.count == 1)
    }

    @Test func aFailedLoadAlertsAndShowsTheEmptyList() async {
        let model = await loaded(makeServer(serverError))

        #expect(model.subscriptions.isEmpty)
        #expect(!model.isLoading)
        #expect(model.errorMessage == "Failed to load subscriptions. Please try again.")
    }

    @Test func loadsTheNewsletterAddressOnce() async {
        let server = StubServer(replies: [.json(200, #"{ "data": { "email_address": "abc@in.example.com" } }"#)])
        let model = SubscriptionListModel(read: makeRead(server), kind: .newsletters)

        await model.loadEmailAddress()
        await model.loadEmailAddress()

        #expect(model.emailAddress == "abc@in.example.com")
        #expect(model.emailAddressError == nil)
        #expect(server.paths == ["/api/v1/source/email/user/user-1/address"])
    }

    @Test func aFailedNewsletterAddressSaysSo() async {
        let model = SubscriptionListModel(read: makeRead(StubServer(replies: [serverError])), kind: .newsletters)

        await model.loadEmailAddress()

        #expect(model.emailAddress == nil)
        #expect(model.emailAddressError == "Unable to load address")
    }
}
