import Foundation

/// Response bodies shaped like the OpenAPI examples and the RN test fixtures.
nonisolated enum Fixtures {
    static let summary = """
        {
          "id": "content-1",
          "content_hash": "abc123",
          "original_url": "https://example.com/article",
          "title": "Test Article",
          "author": "John Doe",
          "published_at": "2025-01-10T08:00:00Z",
          "description": "A test description",
          "image_urls": ["https://example.com/image.jpg", "https://example.com/second.jpg"],
          "word_count": 1000,
          "source_type": "rss",
          "created_at": "2025-01-15T09:00:00.123456Z",
          "updated_at": "2025-01-15T09:00:00Z"
        }
        """

    static func userContent(status: String = "unread", content: String? = summary, extra: String = "") -> String {
        """
        {
          "id": "uc-1",
          "user_id": "user-1",
          "content_id": "content-1",
          "status": "\(status)",
          "list": "reads",
          "is_favorite": false,
          "added_at": "2025-01-15T10:00:00Z",
          "updated_at": "2025-01-16T11:30:00.5Z"\(extra)\(content.map { ",\n\"content\": \($0)" } ?? "")
        }
        """
    }

    static let listPage = """
        {
          "data": [\(userContent()), \(userContent(status: "completed", extra: ", \"scroll_position\": 0.4"))],
          "pagination": { "cursor": "next-cursor", "has_more": true, "limit": 20 },
          "meta": { "timestamp": "2025-01-16T12:00:00Z", "version": "v1" }
        }
        """

    static let detail: String = {
        let content = summary.replacingOccurrences(
            of: "\"title\"", with: "\"cleaned_html\": \"<p>Body</p>\",\n  \"title\"")
        return """
            { "data": \(userContent(content: content)), "meta": { "timestamp": "2025-01-16T12:00:00Z", "version": "v1" } }
            """
    }()

    static let subscriptions = """
        {
          "data": {
            "subscriptions": [
              {
                "id": "sub-1", "type": "rss", "title": "Example Blog", "subscribed_at": "2025-01-01T00:00:00Z",
                "list": "feed",
                "rss_data": { "feed_id": "feed-1", "feed_url": "https://example.com/feed.xml", "polling_tier": "hourly" }
              },
              {
                "id": "sender-1", "type": "email", "title": "Newsletter", "description": "Weekly",
                "subscribed_at": "2025-01-02T00:00:00Z", "list": "reads",
                "email_data": { "email_address": "news@example.com" }
              }
            ],
            "total_count": 2,
            "failed_sources": ["email"]
          }
        }
        """

    static let addFeed = """
        {
          "data": {
            "type": "feed",
            "feed_id": "feed-1",
            "subscription": {
              "id": "sub-1", "user_id": "user-1", "feed_id": "feed-1", "feed_url": "https://example.com/feed.xml",
              "title": "Example Blog", "list": "feed", "subscribed_at": "2025-01-01T00:00:00Z"
            }
          }
        }
        """

    static let addPage = """
        { "data": { "type": "page", "content": \(userContent()) } }
        """

    static let feedback = """
        {
          "data": {
            "content_id": "content-1", "categories": ["body_missing_content", "lead_image"], "comment": null,
            "status": "open", "created_at": "2025-01-16T12:00:00Z", "updated_at": "2025-01-16T12:00:00Z"
          }
        }
        """

    static let auth = """
        {
          "data": {
            "user": {
              "id": "123e4567-e89b-12d3-a456-426614174000", "email_verified": false,
              "created_at": "2025-12-29T10:00:00Z", "updated_at": "2025-12-29T10:00:00Z"
            },
            "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
            "refresh_token": "k3J9x0Qm1vT8bq2l7Zs4dH6yUe5nWcA1oPfRgXiLhMa",
            "expires_in": 3600
          },
          "meta": { "timestamp": "2025-12-29T10:00:00Z", "version": "v1" }
        }
        """

    static let error404 = """
        { "error": "not_found", "message": "content not found", "meta": { "timestamp": "2025-01-16T12:00:00Z", "version": "v1" } }
        """
}

extension String {
    nonisolated var utf8Data: Data { Data(utf8) }
}
