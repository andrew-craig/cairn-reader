import Foundation

/// An item in the user's Feed or Reads, as the app shows it.
nonisolated struct Article: Identifiable, Hashable, Sendable {
    /// The content ID (not the user-content ID).
    let id: String
    var url: String
    var title: String
    var description: String?
    /// Cleaned HTML body; nil until the detail has been fetched.
    var content: String?
    /// Used to tell whether a cached body is stale.
    var contentHash: String?
    var imageUrl: String?
    var author: String?
    var publishedDate: Date?
    /// Minutes, at 200 words per minute.
    var readingTime: Int?
    var isRead: Bool
    var isFavorite: Bool
    /// Which list the item lives in; the reader's actions differ for Feed items.
    var list: ContentList
    var addedAt: Date
    var readAt: Date?
    /// Fraction of the article scrolled (0–1), nil when not started.
    var scrollPosition: Double?
}

extension Article {
    /// Builds the app's article from a list, search or detail response. List and search
    /// responses carry no body, so `content` stays nil until the detail is fetched.
    nonisolated init(_ userContent: UserContentResponse) {
        let isRead = userContent.status == .completed
        let scrollPosition = userContent.scrollPosition.flatMap { $0 > 0 ? $0 : nil }

        guard let content = userContent.content else {
            self.init(
                id: userContent.contentId, url: "", title: "Unknown Article", description: "",
                isRead: isRead, isFavorite: userContent.isFavorite, list: userContent.list,
                addedAt: userContent.addedAt, scrollPosition: scrollPosition)
            return
        }

        self.init(
            id: content.id,
            url: content.originalUrl,
            title: content.title,
            description: content.description,
            content: content.cleanedHtml,
            contentHash: content.contentHash,
            imageUrl: content.imageUrls?.first,
            author: content.author,
            publishedDate: content.publishedAt,
            readingTime: content.wordCount.flatMap { $0 > 0 ? Int((Double($0) / 200).rounded(.up)) : nil },
            isRead: isRead,
            isFavorite: userContent.isFavorite,
            list: userContent.list,
            addedAt: userContent.addedAt,
            readAt: isRead ? userContent.updatedAt : nil,
            scrollPosition: scrollPosition)
    }
}
