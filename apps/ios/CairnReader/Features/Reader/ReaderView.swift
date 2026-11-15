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
            article: article, session: session, next: nextArticle.map { $0(article.id) },
            onNext: nextArticle == nil ? nil : { article = $0 }
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
    let next: Article?
    let onNext: (@MainActor (Article) -> Void)?

    init(article: Article, session: UserSession, next: Article?, onNext: (@MainActor (Article) -> Void)?) {
        _model = State(initialValue: ReaderModel(article: article, session: session))
        self.next = next
        self.onNext = onNext
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
                onLink: { linkTapped($0) }
            )
            .ignoresSafeArea()
        }
    }

    @ToolbarContentBuilder
    private var toolbar: some ToolbarContent {
        ToolbarItemGroup(placement: .bottomBar) {
            if let onNext {
                Button("Next Article", systemImage: "forward") {
                    if let next { onNext(next) }
                }
                .disabled(next == nil)
                Spacer()
            }
            let isFavorite = model.article.isFavorite
            Button(isFavorite ? "Remove from Favorites" : "Favorite", systemImage: isFavorite ? "bookmark.fill" : "bookmark") {
                Task { await model.toggleFavorite() }
            }
            Spacer()
            if model.article.list == .feed {
                Button("Save to Reads", systemImage: "tray.and.arrow.down") {
                    Task { await model.saveToReads() }
                }
            } else {
                Button("Archive", systemImage: "archivebox") {
                    Task { await model.archive() }
                }
            }
            if let url = ReaderTemplate.webURL(model.article.url) {
                Spacer()
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
    private func linkTapped(_ url: URL) {
        if url == ReaderTemplate.webURL(model.article.url) {
            openURL(url)
        } else {
            link = url
        }
    }
}
