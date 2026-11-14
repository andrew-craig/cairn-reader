import Foundation

// Read service wire types (services/read/content/api/openapi.yaml). Keys are snake_case on the
// wire and converted by HTTPClient's coders, so `cleaned_html` is `cleanedHtml` here.

/// Which list an item lives in: the skimmable Feed, or Reads (read or triage).
nonisolated enum ContentList: String, Codable, Sendable {
    case feed
    case reads
}

nonisolated enum ContentStatus: String, Codable, Sendable {
    case unread
    case reading
    case completed
    case archived
}

/// An article. List and search results leave out `cleanedHtml`; the detail endpoint includes it.
nonisolated struct ContentResponse: Codable, Sendable {
    let id: String
    let contentHash: String
    let cleanedHtml: String?
    let originalUrl: String
    let canonicalUrl: String?
    let title: String
    let author: String?
    let publishedAt: Date?
    let description: String?
    let imageUrls: [String]?
    let wordCount: Int?
    let sourceType: String
    let createdAt: Date
    let updatedAt: Date
}

/// One item in the user's lists, with the article it points at.
nonisolated struct UserContentResponse: Codable, Sendable {
    let id: String
    let userId: String
    let contentId: String
    let status: ContentStatus
    let list: ContentList
    /// Fraction of the article scrolled (0–1); omitted when zero.
    let scrollPosition: Double?
    let isFavorite: Bool
    let addedAt: Date
    let updatedAt: Date
    let content: ContentResponse?
}

nonisolated enum URLType: String, Codable, Sendable {
    case feed
    case page
    case unknown
}

nonisolated struct DetectURLResponse: Codable, Sendable {
    /// The URL that was checked (may differ from the request if redirected).
    let url: String
    let type: URLType
    let title: String?
}

nonisolated struct DiscoveredFeed: Codable, Sendable, Hashable {
    let url: String
    /// May be empty.
    let title: String
}

nonisolated struct DiscoverFeedResponse: Codable, Sendable {
    let feeds: [DiscoveredFeed]
}

nonisolated struct AddURLRequest: Codable, Sendable {
    let url: String
    var type: URLType?
    var title: String?
    /// Feeds only: where the feed's new items land (backend default: reads).
    var list: ContentList?
}

nonisolated struct FeedSubscription: Codable, Sendable {
    let id: String
    let userId: String
    let feedId: String
    let feedUrl: String
    let title: String
    let list: ContentList?
    let subscribedAt: Date
}

/// The result of adding a URL: a feed subscription or a saved page, told apart by `type`.
nonisolated enum AddURLResponse: Decodable, Sendable {
    case feed(feedId: String, subscription: FeedSubscription)
    case page(UserContentResponse)

    private enum CodingKeys: String, CodingKey {
        case type, feedId, subscription, content
    }

    init(from decoder: any Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        switch try container.decode(String.self, forKey: .type) {
        case "feed":
            self = .feed(
                feedId: try container.decode(String.self, forKey: .feedId),
                subscription: try container.decode(FeedSubscription.self, forKey: .subscription)
            )
        case "page":
            self = .page(try container.decode(UserContentResponse.self, forKey: .content))
        case let other:
            throw DecodingError.dataCorruptedError(
                forKey: .type, in: container, debugDescription: "Unknown add-URL type \(other)")
        }
    }
}

/// Source types that can be routed to a list (the `{type}` in the list endpoint).
nonisolated enum SubscriptionType: String, Codable, Sendable {
    case rss
    case email
}

nonisolated struct RSSSubscriptionData: Codable, Sendable, Hashable {
    let feedId: String
    let feedUrl: String
    let siteUrl: String?
    let pollingTier: String?
    let lastFetchedAt: String?
}

nonisolated struct EmailSubscriptionData: Codable, Sendable, Hashable {
    let emailAddress: String
    let filterRules: String?
}

nonisolated struct UnifiedSubscription: Codable, Sendable, Identifiable, Hashable {
    let id: String
    let type: SubscriptionType
    let title: String
    let description: String?
    /// Passed through from the source service as a string.
    let subscribedAt: String
    /// Where this source's new items are delivered.
    let list: ContentList
    let rssData: RSSSubscriptionData?
    let emailData: EmailSubscriptionData?
}

nonisolated struct UnifiedSubscriptionsResponse: Codable, Sendable {
    let subscriptions: [UnifiedSubscription]
    let totalCount: Int
    /// Sources that failed to load; their subscriptions are missing from the list.
    let failedSources: [SubscriptionType]
}

nonisolated struct SetSourceListResponse: Codable, Sendable {
    let type: SubscriptionType
    let key: String
    let list: ContentList
}

nonisolated enum ExtractionFeedbackCategory: String, Codable, Sendable, CaseIterable {
    case bodyMissingContent = "body_missing_content"
    case bodyExtraContent = "body_extra_content"
    case images
    case formatting
    case links
    case title
    case author
    case description
    case leadImage = "lead_image"
    case other
}

nonisolated struct ExtractionFeedbackRequest: Codable, Sendable {
    let categories: [ExtractionFeedbackCategory]
    var comment: String?
}

nonisolated struct ExtractionFeedbackResponse: Codable, Sendable {
    enum Status: String, Codable, Sendable {
        case open
        case resolved
    }

    let contentId: String
    let categories: [ExtractionFeedbackCategory]
    let comment: String?
    let status: Status
    let createdAt: Date
    let updatedAt: Date
}
