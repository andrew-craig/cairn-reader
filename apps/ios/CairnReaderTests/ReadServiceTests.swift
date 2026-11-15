import Foundation
import Testing
@testable import CairnReader

// Port of apps/mobile/src/services/read.test.ts and system.test.ts: method, path, query and body
// of each call against a stub server. The transformToArticle cases live in ModelTests (Article).

private typealias Reply = StubServer.Reply

private func makeRead(_ server: StubServer, session: StoredSession? = AuthFixtures.valid) -> ReadService {
    let config = ServerConfig(defaults: UserDefaults(suiteName: UUID().uuidString)!)
    let auth = AuthService(
        client: HTTPClient(server: config, session: server.session()), storage: InMemorySessionStorage(session),
        deviceID: { "device-1" }, sleep: { _ in })
    return ReadService(auth: auth, userID: AuthFixtures.user.id)
}

private func query(_ request: URLRequest?) -> [String: String] {
    guard let url = request?.url, let items = URLComponents(url: url, resolvingAgainstBaseURL: false)?.queryItems
    else { return [:] }
    return Dictionary(uniqueKeysWithValues: items.map { ($0.name, $0.value ?? "") })
}

private let emptyPage = #"{ "data": [], "pagination": { "has_more": false, "limit": 20 } }"#

struct ReadServiceContentTests {
    @Test func listSendsFiltersAndPaging() async throws {
        let server = StubServer(replies: [.json(200, Fixtures.listPage)])
        let page = try await makeRead(server).listUserContents(
            list: .feed, status: .unread, isFavorite: true, limit: 20, cursor: "abc+/=")

        let request = server.requests.first
        #expect(request?.httpMethod == "GET")
        #expect(server.paths == ["/api/v1/content/user/user-1"])
        #expect(
            query(request) == ["list": "feed", "status": "unread", "is_favorite": "true", "limit": "20", "cursor": "abc+/="])
        #expect(request?.bearer == "Bearer access-1")
        #expect(page.data.count == 2)
        #expect(page.pagination.cursor == "next-cursor")
    }

    @Test func listOmitsUnsetFilters() async throws {
        let server = StubServer(replies: [.json(200, emptyPage)])
        let page = try await makeRead(server).listUserContents()

        #expect(server.requests.first?.url?.query() == nil)
        #expect(page.pagination.cursor == nil)
        #expect(!page.pagination.hasMore)
    }

    @Test func listRetriesServerErrors() async throws {
        let server = StubServer(replies: [.json(503, AuthFixtures.errorBody("busy")), .json(200, emptyPage)])
        _ = try await makeRead(server).listUserContents()
        #expect(server.paths.count == 2)
    }

    @Test func countScopesToList() async throws {
        let server = StubServer(replies: [.json(200, #"{ "data": { "count": 3 } }"#)])
        let count = try await makeRead(server).countUserContents(list: .feed, isFavorite: false)

        #expect(count == 3)
        #expect(server.paths == ["/api/v1/content/user/user-1/count"])
        #expect(query(server.requests.first) == ["list": "feed", "is_favorite": "false"])
    }

    @Test func searchScopesToList() async throws {
        let server = StubServer(replies: [.json(200, emptyPage)])
        _ = try await makeRead(server).searchUserContents("go lang", list: .reads, cursor: "c1")

        #expect(server.paths == ["/api/v1/content/user/user-1/search"])
        #expect(query(server.requests.first) == ["q": "go lang", "list": "reads", "cursor": "c1"])
    }

    @Test func getContentByIdReturnsBody() async throws {
        let server = StubServer(replies: [.json(200, Fixtures.detail)])
        let item = try await makeRead(server).getContentById("content-1")

        #expect(server.requests.first?.httpMethod == "GET")
        #expect(server.paths == ["/api/v1/content/user/user-1/content-1"])
        #expect(item.content?.cleanedHtml == "<p>Body</p>")
    }

    @Test func updateSendsOnlySetFields() async throws {
        let server = StubServer(replies: [.json(200, Fixtures.detail)])
        _ = try await makeRead(server).updateUserContent(
            "content-1", UpdateUserContentRequest(scrollPosition: 0.5, isFavorite: true))

        let request = server.requests.first
        #expect(request?.httpMethod == "PATCH")
        #expect(server.paths == ["/api/v1/content/user/user-1/content-1"])
        #expect(request?.jsonBody.count == 2)
        #expect(request?.jsonBody["scroll_position"] as? Double == 0.5)
        #expect(request?.jsonBody["is_favorite"] as? Bool == true)
    }

    @Test func moveToReadsPatchesList() async throws {
        let server = StubServer(replies: [.json(200, Fixtures.detail)])
        _ = try await makeRead(server).moveToReads("content-1")

        let request = server.requests.first
        #expect(request?.httpMethod == "PATCH")
        #expect(server.paths == ["/api/v1/content/user/user-1/content-1"])
        #expect(request?.jsonBody as? [String: String] == ["list": "reads"])
    }

    @Test func addURLForwardsListForFeed() async throws {
        let server = StubServer(replies: [.json(200, Fixtures.addFeed)])
        let result = try await makeRead(server).addURL(
            AddURLRequest(url: "https://example.com/rss", type: .feed, list: .feed))

        let request = server.requests.first
        #expect(request?.httpMethod == "POST")
        #expect(server.paths == ["/api/v1/content/user/user-1"])
        #expect(request?.jsonBody as? [String: String] == ["url": "https://example.com/rss", "type": "feed", "list": "feed"])
        guard case let .feed(feedId, _) = result else {
            Issue.record("expected a feed")
            return
        }
        #expect(feedId == "feed-1")
    }

    @Test func addURLSavesPage() async throws {
        let server = StubServer(replies: [.json(200, Fixtures.addPage)])
        let result = try await makeRead(server).addURL(AddURLRequest(url: "https://example.com/a"))

        #expect(server.requests.first?.jsonBody as? [String: String] == ["url": "https://example.com/a"])
        guard case .page = result else {
            Issue.record("expected a page")
            return
        }
    }
}

struct ReadServiceErrorTests {
    @Test func serverErrorPassesThrough() async {
        let server = StubServer(replies: [.json(404, Fixtures.error404)])
        await #expect(throws: HttpError(status: 404, message: "content not found")) {
            try await makeRead(server).getContentById("content-1")
        }
    }

    @Test func unreachablePassesThrough() async {
        let server = StubServer(replies: [.failure(.notConnectedToInternet)])
        await #expect(throws: NetworkError.unreachable) {
            try await makeRead(server).moveToReads("content-1")
        }
    }

    @Test func noSessionIsSessionExpired() async {
        let server = StubServer(replies: [])
        await #expect(throws: AuthError.sessionExpired) {
            try await makeRead(server, session: nil).setSourceList(.rss, key: "feed-1", list: .feed)
        }
        #expect(server.requests.isEmpty)
    }
}

struct ReadServiceAddingTests {
    @Test func detectPostsURLWithTimeout() async throws {
        let body = #"{ "data": { "url": "https://example.com/rss", "type": "feed", "title": "Blog" } }"#
        let server = StubServer(replies: [.json(200, body)])
        let result = try await makeRead(server).detectURL("https://example.com/rss")

        let request = server.requests.first
        #expect(request?.httpMethod == "POST")
        #expect(server.paths == ["/api/v1/content/detect"])
        #expect(request?.jsonBody as? [String: String] == ["url": "https://example.com/rss"])
        #expect(request?.timeoutInterval == ReadService.detectTimeout)
        #expect(result.type == .feed)
        #expect(result.title == "Blog")
    }

    @Test(arguments: [Reply.json(500, AuthFixtures.errorBody("boom")), .failure(.timedOut)])
    func detectFailureIsUnknown(reply: StubServer.Reply) async throws {
        let result = try await makeRead(StubServer(replies: [reply])).detectURL("https://example.com/x")

        #expect(result.url == "https://example.com/x")
        #expect(result.type == .unknown)
        #expect(result.title == nil)
    }

    @Test func discoverPostsURLWithTimeout() async throws {
        let body = #"{ "data": { "feeds": [{ "url": "https://example.com/feed.xml", "title": "" }] } }"#
        let server = StubServer(replies: [.json(200, body)])
        let feeds = try await makeRead(server).discoverFeed("https://example.com")

        let request = server.requests.first
        #expect(server.paths == ["/api/v1/content/discover-feed"])
        #expect(request?.jsonBody as? [String: String] == ["url": "https://example.com"])
        #expect(request?.timeoutInterval == ReadService.discoverTimeout)
        #expect(feeds == [DiscoveredFeed(url: "https://example.com/feed.xml", title: "")])
    }

    @Test(arguments: [Reply.json(400, AuthFixtures.errorBody("bad url")), .failure(.timedOut)])
    func discoverFailureIsEmpty(reply: StubServer.Reply) async throws {
        #expect(try await makeRead(StubServer(replies: [reply])).discoverFeed("https://example.com").isEmpty)
    }
}

struct ReadServiceSubscriptionTests {
    @Test func listAllSubscriptions() async throws {
        let server = StubServer(replies: [.json(200, Fixtures.subscriptions)])
        let result = try await makeRead(server).listAllSubscriptions()

        #expect(server.requests.first?.httpMethod == "GET")
        #expect(server.paths == ["/api/v1/content/user/user-1/subscriptions"])
        #expect(result.subscriptions.map(\.id) == ["sub-1", "sender-1"])
        #expect(result.failedSources == [.email])
    }

    @Test func unsubscribeDeletesFeed() async throws {
        let server = StubServer(replies: [.json(200, #"{ "data": { "success": true, "message": "unsubscribed" } }"#)])
        try await makeRead(server).unsubscribeFromRSSFeed("feed-1")

        #expect(server.requests.first?.httpMethod == "DELETE")
        #expect(server.paths == ["/api/v1/content/user/user-1/subscriptions/rss/feed-1"])
    }

    @Test(arguments: [(SubscriptionType.rss, "feed-1", ContentList.feed), (.email, "sender-1", .reads)])
    func setSourceListPutsList(type: SubscriptionType, key: String, list: ContentList) async throws {
        let body = #"{ "data": { "type": "\#(type.rawValue)", "key": "\#(key)", "list": "\#(list.rawValue)" } }"#
        let server = StubServer(replies: [.json(200, body)])
        let result = try await makeRead(server).setSourceList(type, key: key, list: list)

        let request = server.requests.first
        #expect(request?.httpMethod == "PUT")
        #expect(server.paths == ["/api/v1/content/user/user-1/subscriptions/\(type.rawValue)/\(key)/list"])
        #expect(request?.jsonBody as? [String: String] == ["list": list.rawValue])
        #expect(result.list == list)
    }

    @Test func setSourceListThrowsServerMessage() async {
        let server = StubServer(replies: [.json(404, AuthFixtures.errorBody("not subscribed"))])
        await #expect(throws: HttpError(status: 404, message: "not subscribed")) {
            try await makeRead(server).setSourceList(.rss, key: "feed-1", list: .feed)
        }
    }

    @Test func getOrCreateEmailAddress() async throws {
        let server = StubServer(replies: [.json(200, #"{ "data": { "email_address": "abc@in.example.com" } }"#)])
        let address = try await makeRead(server).getOrCreateEmailAddress()

        #expect(server.requests.first?.httpMethod == "POST")
        #expect(server.paths == ["/api/v1/source/email/user/user-1/address"])
        #expect(address == "abc@in.example.com")
    }
}

struct ReadServiceFeedbackTests {
    @Test func getUnwrapsReport() async throws {
        let server = StubServer(replies: [.json(200, Fixtures.feedback)])
        let report = try await makeRead(server).getExtractionFeedback("content-1")

        #expect(server.requests.first?.httpMethod == "GET")
        #expect(server.paths == ["/api/v1/content/user/user-1/content-1/feedback"])
        #expect(report?.categories == [.bodyMissingContent, .leadImage])
    }

    @Test func notFoundIsNoReport() async throws {
        let server = StubServer(replies: [.json(404, #"{ "error": "not_found" }"#)])
        #expect(try await makeRead(server).getExtractionFeedback("content-1") == nil)
    }

    @Test func otherGetFailuresThrow() async {
        let server = StubServer(replies: [.json(500, AuthFixtures.errorBody("boom"))])
        await #expect(throws: HttpError(status: 500, message: "boom")) {
            try await makeRead(server).getExtractionFeedback("content-1")
        }
    }

    @Test func submitPutsCategoriesAndComment() async throws {
        let server = StubServer(replies: [.json(200, Fixtures.feedback)])
        let report = try await makeRead(server).submitExtractionFeedback(
            "content-1", ExtractionFeedbackRequest(categories: [.images], comment: "broken"))

        let request = server.requests.first
        #expect(request?.httpMethod == "PUT")
        #expect(server.paths == ["/api/v1/content/user/user-1/content-1/feedback"])
        #expect(request?.jsonBody["categories"] as? [String] == ["images"])
        #expect(request?.jsonBody["comment"] as? String == "broken")
        #expect(report.contentId == "content-1")
    }

    @Test func submitThrowsServerMessage() async {
        let server = StubServer(replies: [.json(400, AuthFixtures.errorBody("bad category"))])
        await #expect(throws: HttpError(status: 400, message: "bad category")) {
            try await makeRead(server).submitExtractionFeedback("content-1", ExtractionFeedbackRequest(categories: [.images]))
        }
    }
}

struct SystemServiceTests {
    private func makeSystem(_ server: StubServer) -> SystemService {
        let config = ServerConfig(defaults: UserDefaults(suiteName: UUID().uuidString)!)
        return SystemService(client: HTTPClient(server: config, session: server.session()))
    }

    @Test func returnsReportedVersion() async throws {
        let server = StubServer(replies: [.json(200, #"{ "status": "ok", "version": "1.2.3" }"#)])
        #expect(try await makeSystem(server).serverVersion() == "1.2.3")
        #expect(server.paths == ["/health/live"])
        #expect(server.requests.first?.bearer == nil)
    }

    @Test func noVersionIsNil() async throws {
        let server = StubServer(replies: [.json(200, #"{ "status": "ok" }"#)])
        #expect(try await makeSystem(server).serverVersion() == nil)
    }

    @Test func unreachableIsNetworkError() async {
        let server = StubServer(replies: [.failure(.cannotConnectToHost)])
        await #expect(throws: NetworkError.unreachable) { try await makeSystem(server).serverVersion() }
    }
}
