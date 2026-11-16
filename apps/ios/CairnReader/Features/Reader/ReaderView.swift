import SwiftUI
import UIKit

/// The reading screen for `Route.article`. Next article replaces the article shown here, a new
/// `ReaderModel` for it, instead of pushing another screen, as the React Native reader replaced
/// its route.
struct ReaderView: View {
    let session: UserSession
    /// The presenting list's row after an id (`article(after:)`); nil when the presenter has no
    /// order to follow (no Next button).
    let nextArticle: (@MainActor (String) -> Article?)?
    @State private var article: Article

    init(article: Article, session: UserSession, nextArticle: (@MainActor (String) -> Article?)?) {
        self.session = session
        self.nextArticle = nextArticle
        _article = State(initialValue: article)
    }

    var body: some View {
        ReaderScreen(
            article: article, session: session, showsNext: nextArticle != nil, next: nextArticle?(article.id),
            current: $article
        )
        .id(article.id)
    }
}

/// One article: the web view (or its loading and offline states), the link dialog and the bottom
/// toolbar.
private struct ReaderScreen: View {
    @Environment(Connectivity.self) private var connectivity
    @Environment(\.dismiss) private var dismiss
    @Environment(\.openURL) private var openURL
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize
    @State private var model: ReaderModel
    /// The link whose dialog is showing.
    @State private var link: URL?
    /// Where the link was tapped, in the web view's coordinates; the dialog points there.
    @State private var linkPoint = CGPoint.zero
    /// Whether the presenter has an order to follow (the Next button shows).
    let showsNext: Bool
    let next: Article?
    /// The article `ReaderView` shows; Next replaces it.
    @Binding var current: Article

    init(article: Article, session: UserSession, showsNext: Bool, next: Article?, current: Binding<Article>) {
        _model = State(initialValue: ReaderModel(article: article, session: session))
        self.showsNext = showsNext
        self.next = next
        _current = current
    }

    var body: some View {
        content
            .navigationBarTitleDisplayMode(.inline)
            .toolbar(.hidden, for: .tabBar)
            .toolbar { toolbar }
            .task { await model.follow() }
            .task { await model.markReadingIfNeeded() }
            .task(id: connectivity.isOnline) { await model.loadBody(isOnline: connectivity.isOnline) }
            .onDisappear {
                Task { await model.leave() }
            }
            .onChange(of: model.isFinished) { _, isFinished in
                if isFinished { dismiss() }
            }
            .sensoryFeedback(.impact(weight: .medium), trigger: link) { _, link in link != nil }
            .sensoryFeedback(.success, trigger: model.savedLinks)
            .alert(
                model.alert?.title ?? "",
                isPresented: Binding(get: { model.alert != nil }, set: { if !$0 { model.alert = nil } })
            ) {
                Button("OK") {}
            } message: {
                Text(model.alert?.message ?? "")
            }
    }

    @ViewBuilder
    private var content: some View {
        switch model.display {
        case .loading:
            ProgressView()
        case .unavailableOffline:
            ContentUnavailableView("Not available offline", systemImage: "wifi.slash")
        case .article:
            ArticleWebView(
                html: ReaderTemplate.html(for: model.article, baseSize: baseSize),
                restoreFraction: { model.restoreFraction },
                onScroll: { model.scrolled($0) },
                onLink: { linkTapped($0, at: $1) }
            )
            .overlay { linkAnchor }
            .ignoresSafeArea()
        }
    }

    /// The link dialog, anchored at the tapped point: on iOS 26 it is a popover pointing at the
    /// view it is attached to. In the web view's overlay, inside the same `ignoresSafeArea`, so the
    /// web view's own coordinates place it.
    private var linkAnchor: some View {
        Color.clear
            .frame(width: 1, height: 1)
            .allowsHitTesting(false)
            .confirmationDialog(
                link.map { ArticleRow.domain(of: $0.absoluteString) ?? $0.absoluteString } ?? "",
                isPresented: Binding(get: { link != nil }, set: { if !$0 { link = nil } }),
                titleVisibility: .visible, presenting: link
            ) { url in
                Button("Save to Reads") {
                    Task { await model.saveLink(url) }
                }
                Button("Open in Browser") { openURL(url) }
                Button("Copy Link") { UIPasteboard.general.url = url }
            }
            // After the dialog, so it anchors to the 1-point frame, not the full-size position box.
            .position(linkPoint)
    }

    @ToolbarContentBuilder
    private var toolbar: some ToolbarContent {
        ToolbarItemGroup(placement: .bottomBar) {
            Spacer()
            if showsNext {
                Button("Next Article", image: .arrowDown) {
                    if let next { current = next }
                }
                .disabled(next == nil)
            }
            let isFavorite = model.article.isFavorite
            Button(isFavorite ? "Remove from Favorites" : "Favorite", image: isFavorite ? .bookmarkFill : .bookmark) {
                Task { await model.toggleFavorite() }
            }
            if model.article.list == .feed {
                Button("Save to Reads", image: .read) {
                    Task { await model.saveToReads() }
                }
            } else {
                Button("Archive", image: .archive) {
                    Task { await model.archive() }
                }
            }
            if let url = ReaderTemplate.webURL(model.article.url) {
                ShareLink(item: url)
            }
        }
    }

    /// The user's Dynamic Type body size for the base of the page's type scale (16 pt at the
    /// default size, as `FontSizes.md`).
    private var baseSize: Double {
        let traits = UITraitCollection(preferredContentSizeCategory: UIContentSizeCategory(dynamicTypeSize))
        return UIFontMetrics(forTextStyle: .body).scaledValue(for: 16, compatibleWith: traits)
    }

    /// The title opens the article in the browser; any other link asks what to do with it.
    private func linkTapped(_ url: URL, at point: CGPoint) {
        if url == ReaderTemplate.webURL(model.article.url) {
            openURL(url)
        } else {
            linkPoint = point
            link = url
        }
    }
}
