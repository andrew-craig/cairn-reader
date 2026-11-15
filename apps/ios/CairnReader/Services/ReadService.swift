import Foundation

/// The read service's user endpoints (content, subscriptions, extraction feedback) and the email
/// address endpoint, sent through `AuthService` for one user (decision 10: bound at init, owned by
/// that user's `UserSession`). Errors pass through unchanged: `NetworkError`
/// when the server could not be reached, `HttpError` for its answer, `AuthError.sessionExpired`
/// when there is no session.
nonisolated struct ReadService: Sendable {
    /// Idle timeouts for detection and discovery (discovery matches the backend's own timeout).
    static let detectTimeout: TimeInterval = 10
    static let discoverTimeout: TimeInterval = 15

    let auth: AuthService
    let userID: String

    // MARK: Contents

    /// One page of the user's items, newest first. Pass the previous page's `cursor` for the next.
    func listUserContents(
        list: ContentList? = nil, status: ContentStatus? = nil, isFavorite: Bool? = nil, limit: Int? = nil,
        cursor: String? = nil
    ) async throws -> Page<UserContentResponse> {
        let query = Self.filters(list: list, status: status, isFavorite: isFavorite) + Self.paging(limit, cursor)
        let request = HTTPRequest(.get, userPath(), query: query)
        return try await auth.authorizedRequestWithRetry(request, as: Page<UserContentResponse>.self)
    }

    /// How many items match the filters, without paging through them.
    func countUserContents(list: ContentList? = nil, status: ContentStatus? = nil, isFavorite: Bool? = nil)
        async throws -> Int
    {
        let path = userPath("count")
        let request = HTTPRequest(.get, path, query: Self.filters(list: list, status: status, isFavorite: isFavorite))
        return try await auth.authorizedRequestWithRetry(request, as: Envelope<ContentCountResponse>.self).data.count
    }

    func searchUserContents(_ text: String, list: ContentList? = nil, limit: Int? = nil, cursor: String? = nil)
        async throws -> Page<UserContentResponse>
    {
        let query = [URLQueryItem(name: "q", value: text)] + Self.filters(list: list) + Self.paging(limit, cursor)
        let request = HTTPRequest(.get, userPath("search"), query: query)
        return try await auth.authorizedRequestWithRetry(request, as: Page<UserContentResponse>.self)
    }

    /// The item with the article body (`content.cleanedHtml`).
    func getContentById(_ contentID: String) async throws -> UserContentResponse {
        let request = HTTPRequest(.get, userPath(contentID))
        return try await auth.authorizedRequest(request, as: Envelope<UserContentResponse>.self).data
    }

    /// Saves a page or subscribes to a feed. Also how a link tapped in an article is saved.
    func addURL(_ body: AddURLRequest) async throws -> AddURLResponse {
        let request = HTTPRequest(.post, userPath(), body: body)
        return try await auth.authorizedRequest(request, as: Envelope<AddURLResponse>.self).data
    }

    func updateUserContent(_ contentID: String, _ update: UpdateUserContentRequest) async throws
        -> UserContentResponse
    {
        let request = HTTPRequest(.patch, userPath(contentID), body: update)
        return try await auth.authorizedRequest(request, as: Envelope<UserContentResponse>.self).data
    }

    /// Saves a Feed item to Reads. The backend re-stamps `addedAt`, so it lands at the top.
    func moveToReads(_ contentID: String) async throws -> UserContentResponse {
        try await updateUserContent(contentID, UpdateUserContentRequest(list: .reads))
    }

    // MARK: Adding

    /// Whether `url` is a feed or a page. Never throws but for cancellation: a failure or a
    /// timeout answers `.unknown`, and the caller asks the user.
    func detectURL(_ url: String) async throws -> DetectURLResponse {
        let request = HTTPRequest(.post, "/api/v1/content/detect", body: URLBody(url: url), timeout: Self.detectTimeout)
        do {
            return try await auth.authorizedRequest(request, as: Envelope<DetectURLResponse>.self).data
        } catch is CancellationError {
            throw CancellationError()
        } catch {
            return DetectURLResponse(url: url, type: .unknown, title: nil)
        }
    }

    /// Feeds advertised by the page at `url`. Never throws but for cancellation: a failure or a
    /// timeout answers no feeds.
    func discoverFeed(_ url: String) async throws -> [DiscoveredFeed] {
        let request = HTTPRequest(
            .post, "/api/v1/content/discover-feed", body: URLBody(url: url), timeout: Self.discoverTimeout)
        do {
            return try await auth.authorizedRequest(request, as: Envelope<DiscoverFeedResponse>.self).data.feeds
        } catch is CancellationError {
            throw CancellationError()
        } catch {
            return []
        }
    }

    // MARK: Subscriptions

    /// RSS and email subscriptions together, each with the list its new items go to.
    func listAllSubscriptions() async throws -> UnifiedSubscriptionsResponse {
        let request = HTTPRequest(.get, userPath("subscriptions"))
        return try await auth.authorizedRequestWithRetry(request, as: Envelope<UnifiedSubscriptionsResponse>.self).data
    }

    /// Stops future deliveries from the feed; items already in the user's lists stay.
    func unsubscribeFromRSSFeed(_ feedID: String) async throws {
        let request = HTTPRequest(.delete, userPath("subscriptions/rss/\(feedID)"))
        try await auth.authorizedRequest(request)
    }

    /// Chooses where a source's new items land. `key` is the feed ID for `.rss` and the
    /// subscription (sender) ID for `.email`. Items already delivered stay where they are.
    func setSourceList(_ type: SubscriptionType, key: String, list: ContentList) async throws
        -> SetSourceListResponse
    {
        let path = userPath("subscriptions/\(type.rawValue)/\(key)/list")
        let request = HTTPRequest(.put, path, body: SetSourceListRequest(list: list))
        return try await auth.authorizedRequest(request, as: Envelope<SetSourceListResponse>.self).data
    }

    /// The user's newsletter address, created on first request.
    func getOrCreateEmailAddress() async throws -> String {
        let request = HTTPRequest(.post, "/api/v1/source/email/user/\(userID)/address")
        return try await auth.authorizedRequest(request, as: Envelope<EmailAddressResponse>.self).data.emailAddress
    }

    // MARK: Extraction feedback

    /// The user's report on the item, or nil when there is none.
    func getExtractionFeedback(_ contentID: String) async throws -> ExtractionFeedbackResponse? {
        let request = HTTPRequest(.get, userPath("\(contentID)/feedback"))
        do {
            return try await auth.authorizedRequest(request, as: Envelope<ExtractionFeedbackResponse>.self).data
        } catch let error as HttpError where error.status == 404 {
            return nil
        }
    }

    /// Reports a bad extraction; resubmitting replaces the earlier report. Online only.
    func submitExtractionFeedback(_ contentID: String, _ feedback: ExtractionFeedbackRequest) async throws
        -> ExtractionFeedbackResponse
    {
        let request = HTTPRequest(.put, userPath("\(contentID)/feedback"), body: feedback)
        return try await auth.authorizedRequest(request, as: Envelope<ExtractionFeedbackResponse>.self).data
    }

    // MARK: Private

    private struct URLBody: Encodable, Sendable {
        let url: String
    }

    /// `/api/v1/content/user/{userID}`, followed by `suffix` when given.
    private func userPath(_ suffix: String? = nil) -> String {
        let base = "/api/v1/content/user/\(userID)"
        return suffix.map { "\(base)/\($0)" } ?? base
    }

    private static func filters(list: ContentList?, status: ContentStatus? = nil, isFavorite: Bool? = nil)
        -> [URLQueryItem]
    {
        var query: [URLQueryItem] = []
        if let status { query.append(URLQueryItem(name: "status", value: status.rawValue)) }
        if let list { query.append(URLQueryItem(name: "list", value: list.rawValue)) }
        if let isFavorite { query.append(URLQueryItem(name: "is_favorite", value: String(isFavorite))) }
        return query
    }

    private static func paging(_ limit: Int?, _ cursor: String?) -> [URLQueryItem] {
        var query: [URLQueryItem] = []
        if let limit { query.append(URLQueryItem(name: "limit", value: String(limit))) }
        if let cursor { query.append(URLQueryItem(name: "cursor", value: cursor)) }
        return query
    }
}
