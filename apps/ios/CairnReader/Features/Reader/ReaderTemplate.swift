import UIKit

/// The reader's page: the bundled `reader.html` and `reader.css` (decision 2) with the article's
/// header and `cleaned_html` filled in. The colour tokens are written as CSS variables from the
/// colour assets (light, and dark under `prefers-color-scheme`), so the page has no colour of
/// its own; the base size is the user's Dynamic Type body size.
enum ReaderTemplate {
    /// The bundle's `reader.html`; synchronized-group resources land at the bundle's top level.
    private static let template: String = {
        // The template is part of the app; CairnReaderTests fails if it is missing.
        let url = Bundle.main.url(forResource: "reader", withExtension: "html")!
        return try! String(contentsOf: url, encoding: .utf8)
    }()

    /// The tokens the stylesheet uses, by CSS variable name.
    private static let colors: [(name: String, color: ColorResource)] = [
        ("text", .text), ("text-secondary", .textSecondary), ("primary", .primary), ("border", .border),
        ("card", .card), ("background", .background),
    ]

    /// The page for `article`. `baseSize` is the body text size in points.
    static func html(for article: Article, baseSize: Double, now: Date = .now) -> String {
        let variables = """
            :root { --base-size: \(baseSize)px; \(colorVariables(.light)) }
            @media (prefers-color-scheme: dark) { :root { \(colorVariables(.dark)) } }
            """
        let title = escape(article.title)
        let body = article.content ?? article.description.map { "<p>\(escape($0))</p>" } ?? ""
        return
            template
            .replacingOccurrences(of: "{{variables}}", with: variables)
            .replacingOccurrences(of: "{{title}}", with: webURL(article.url) != nil ? "<a href=\"\(escape(article.url))\">\(title)</a>" : title)
            .replacingOccurrences(of: "{{byline}}", with: escape(byline(for: article, now: now)))
            .replacingOccurrences(of: "{{body}}", with: body)
    }

    /// "Author | 1 week ago | 5 min read": the author (else the domain; "Email" for an email
    /// without one), the published date (not the date added) and the reading time, each left out
    /// when unknown.
    static func byline(for article: Article, now: Date = .now) -> String {
        let author = article.author.flatMap { $0.isEmpty ? nil : $0 }
        let source = article.url.hasPrefix("email://") ? author ?? "Email" : author ?? ArticleRow.domain(of: article.url)
        let date = article.publishedDate?.relativeDayLabel(now: now)
        let readingTime = article.readingTime.map { "\($0) min read" }
        return [source, date, readingTime].compactMap { $0 }.joined(separator: " | ")
    }

    /// `string` as an http(s) URL, else nil.
    static func webURL(_ string: String) -> URL? {
        guard let url = URL(string: string), url.scheme == "https" || url.scheme == "http" else { return nil }
        return url
    }

    private static func colorVariables(_ style: UIUserInterfaceStyle) -> String {
        let traits = UITraitCollection(userInterfaceStyle: style)
        return colors.map { "--\($0.name): \(hex(UIColor(resource: $0.color).resolvedColor(with: traits)));" }
            .joined(separator: " ")
    }

    /// '#RRGGBBAA'.
    private static func hex(_ color: UIColor) -> String {
        var r: CGFloat = 0, g: CGFloat = 0, b: CGFloat = 0, a: CGFloat = 0
        color.getRed(&r, green: &g, blue: &b, alpha: &a)
        let bytes = [r, g, b, a].map { Int(($0 * 255).rounded()) }
        return String(format: "#%02X%02X%02X%02X", bytes[0], bytes[1], bytes[2], bytes[3])
    }

    private static func escape(_ text: String) -> String {
        text.replacingOccurrences(of: "&", with: "&amp;")
            .replacingOccurrences(of: "<", with: "&lt;")
            .replacingOccurrences(of: ">", with: "&gt;")
            .replacingOccurrences(of: "\"", with: "&quot;")
            // So no escaped text can contain a later placeholder; the body goes in last.
            .replacingOccurrences(of: "{", with: "&#123;")
    }
}
