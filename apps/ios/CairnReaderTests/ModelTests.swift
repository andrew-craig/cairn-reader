import Foundation
import Testing
@testable import CairnReader

struct ModelDecodingTests {
    func decode<T: Decodable>(_ json: String, as _: T.Type = T.self) throws -> T {
        try HTTPClient.decoder.decode(T.self, from: json.utf8Data)
    }

    @Test func listPage() throws {
        let page = try decode(Fixtures.listPage, as: Page<UserContentResponse>.self)
        #expect(page.data.count == 2)
        #expect(page.pagination.cursor == "next-cursor")
        #expect(page.pagination.hasMore)
        #expect(page.pagination.limit == 20)

        let item = page.data[0]
        #expect(item.contentId == "content-1")
        #expect(item.status == .unread)
        #expect(item.list == .reads)
        #expect(item.scrollPosition == nil)
        #expect(item.addedAt == Date(timeIntervalSince1970: 1_736_935_200))
        #expect(item.content?.cleanedHtml == nil)
        #expect(item.content?.imageUrls?.count == 2)
        #expect(page.data[1].scrollPosition == 0.4)
    }

    @Test func fractionalSecondDates() throws {
        let item = try decode(Fixtures.userContent(), as: UserContentResponse.self)
        #expect(item.updatedAt == Date(timeIntervalSince1970: 1_737_027_000.5))
        let created = try #require(item.content?.createdAt.timeIntervalSince1970)
        #expect(abs(created - 1_736_931_600.123456) < 0.001)
    }

    @Test func detail() throws {
        let detail = try decode(Fixtures.detail, as: Envelope<UserContentResponse>.self).data
        #expect(detail.content?.cleanedHtml == "<p>Body</p>")
    }

    @Test func subscriptions() throws {
        let response = try decode(Fixtures.subscriptions, as: Envelope<UnifiedSubscriptionsResponse>.self).data
        #expect(response.totalCount == 2)
        #expect(response.failedSources == [.email])
        let rss = response.subscriptions[0]
        #expect(rss.type == .rss)
        #expect(rss.list == .feed)
        #expect(rss.rssData?.feedId == "feed-1")
        #expect(rss.emailData == nil)
        let email = response.subscriptions[1]
        #expect(email.emailData?.emailAddress == "news@example.com")
        #expect(email.description == "Weekly")
    }

    @Test func detectAndDiscover() throws {
        let detected = try decode(
            #"{"url": "https://example.com/feed.xml", "type": "feed", "title": null}"#, as: DetectURLResponse.self)
        #expect(detected.type == .feed)
        #expect(detected.title == nil)

        let discovered = try decode(
            #"{"feeds": [{"url": "https://example.com/feed.xml", "title": ""}]}"#, as: DiscoverFeedResponse.self)
        #expect(discovered.feeds == [DiscoveredFeed(url: "https://example.com/feed.xml", title: "")])
    }

    @Test func addURL() throws {
        guard case let .feed(feedId, subscription) = try decode(Fixtures.addFeed, as: Envelope<AddURLResponse>.self).data
        else { Issue.record("expected a feed"); return }
        #expect(feedId == "feed-1")
        #expect(subscription.list == .feed)

        guard case let .page(content) = try decode(Fixtures.addPage, as: Envelope<AddURLResponse>.self).data
        else { Issue.record("expected a page"); return }
        #expect(content.contentId == "content-1")
    }

    @Test func extractionFeedback() throws {
        let response = try decode(Fixtures.feedback, as: Envelope<ExtractionFeedbackResponse>.self).data
        #expect(response.categories == [.bodyMissingContent, .leadImage])
        #expect(response.comment == nil)
        #expect(response.status == .open)
    }

    @Test func authResponse() throws {
        let response = try decode(Fixtures.auth, as: Envelope<AuthResponse>.self).data
        #expect(response.expiresIn == 3600)
        #expect(response.refreshToken.hasPrefix("k3J9"))
        #expect(response.user.email == nil)
        #expect(response.user.lastLoginAt == nil)
        #expect(!response.user.emailVerified)
    }

    @Test func requestsEncodeSnakeCase() throws {
        let feedback = ExtractionFeedbackRequest(categories: [.bodyExtraContent], comment: "Footer kept")
        let json = try #require(String(data: HTTPClient.encoder.encode(feedback), encoding: .utf8))
        #expect(json.contains(#""body_extra_content""#))

        let add = AddURLRequest(url: "https://example.com/feed.xml", type: .feed, list: .feed)
        let object = try #require(
            JSONSerialization.jsonObject(with: HTTPClient.encoder.encode(add)) as? [String: String])
        #expect(object == ["url": "https://example.com/feed.xml", "type": "feed", "list": "feed"])
    }
}

struct ArticleTests {
    func article(_ json: String) throws -> Article {
        Article(try HTTPClient.decoder.decode(UserContentResponse.self, from: json.utf8Data))
    }

    @Test func summary() throws {
        let article = try article(Fixtures.userContent())
        #expect(article.id == "content-1")
        #expect(article.url == "https://example.com/article")
        #expect(article.title == "Test Article")
        #expect(article.description == "A test description")
        #expect(article.author == "John Doe")
        #expect(article.imageUrl == "https://example.com/image.jpg")
        #expect(article.publishedDate == Date(timeIntervalSince1970: 1_736_496_000))
        #expect(article.contentHash == "abc123")
        #expect(article.content == nil)
        #expect(article.readingTime == 5)
        #expect(!article.isRead)
        #expect(article.readAt == nil)
        #expect(!article.isFavorite)
        #expect(article.list == .reads)
        #expect(article.addedAt == Date(timeIntervalSince1970: 1_736_935_200))
        #expect(article.scrollPosition == nil)
    }

    @Test func detailCarriesBody() throws {
        let detail = try HTTPClient.decoder.decode(Envelope<UserContentResponse>.self, from: Fixtures.detail.utf8Data)
        #expect(Article(detail.data).content == "<p>Body</p>")
    }

    @Test func completedIsRead() throws {
        let article = try article(Fixtures.userContent(status: "completed", extra: #", "scroll_position": 0.4"#))
        #expect(article.isRead)
        #expect(article.readAt == Date(timeIntervalSince1970: 1_737_027_000.5))
        #expect(article.scrollPosition == 0.4)
    }

    @Test(arguments: [(1000, 5), (1001, 6), (1, 1), (0, nil)] as [(Int, Int?)])
    func readingTime(words: Int, minutes: Int?) throws {
        let content = Fixtures.summary.replacingOccurrences(of: "\"word_count\": 1000", with: "\"word_count\": \(words)")
        #expect(try article(Fixtures.userContent(content: content)).readingTime == minutes)
    }

    @Test func zeroScrollPositionIsNotStarted() throws {
        #expect(try article(Fixtures.userContent(extra: #", "scroll_position": 0"#)).scrollPosition == nil)
    }

    @Test func missingContentFallsBack() throws {
        let article = try article(Fixtures.userContent(status: "completed", content: nil))
        #expect(article.id == "content-1")
        #expect(article.url == "")
        #expect(article.title == "Unknown Article")
        #expect(article.description == "")
        #expect(article.isRead)
        #expect(article.readAt == nil)
        #expect(article.readingTime == nil)
    }
}
