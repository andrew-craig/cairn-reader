import SwiftUI

/// One article as list row content: title, details line, optional pending note, thumbnail.
/// The title is bold and in the text colour while unread, muted once started, and muted and
/// regular weight once read (the React Native `ArticleRow`).
struct ArticleRow: View {
    let article: Article
    /// A Save to Reads is still queued.
    var pending = false

    var body: some View {
        HStack(spacing: 16) {
            VStack(alignment: .leading, spacing: 4) {
                Text(article.title)
                    .font(article.isRead ? Typography.body : Typography.bodyBold)
                    .foregroundStyle(article.isRead || article.isPartiallyRead ? Color(.textSecondary) : Color(.text))
                    .lineLimit(2)
                Text(Self.details(for: article))
                    .font(Typography.subheadline)
                    .foregroundStyle(Color(.textSecondary))
                    .lineLimit(1)
                if pending {
                    Text("Waiting to save — will sync when online")
                        .font(Typography.caption)
                        .foregroundStyle(Color(.info))
                        .lineLimit(1)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)

            if let url = article.imageUrl.flatMap({ URL(string: $0) }) {
                AsyncImage(url: url) { image in
                    image.resizable().scaledToFill()
                } placeholder: {
                    Color(.hover)
                }
                .frame(width: 48, height: 48)
                .clipShape(.rect(cornerRadius: 6))
                .accessibilityHidden(true)
            }
        }
        .padding(.vertical, 8)
    }

    /// "Author · 3 days ago · 5 min": the author (else the URL's domain), the published date and
    /// the reading time, each left out when unknown.
    static func details(for article: Article, now: Date = .now) -> String {
        let source = article.author.flatMap { $0.isEmpty ? nil : $0 } ?? domain(of: article.url)
        let date = article.publishedDate?.relativeDayLabel(now: now)
        let readingTime = article.readingTime.map { "\($0) min" }
        return [source, date, readingTime].compactMap { $0 }.joined(separator: " · ")
    }

    /// The URL's host without a leading "www."; nil when it has none.
    static func domain(of url: String) -> String? {
        guard let host = URL(string: url)?.host(), !host.isEmpty else { return nil }
        return host.hasPrefix("www.") ? String(host.dropFirst(4)) : host
    }
}
