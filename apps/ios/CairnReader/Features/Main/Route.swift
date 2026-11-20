import SwiftUI

/// Every destination a tab's `NavigationStack` can push, typed in one place. The article carries
/// the `Article` value only, never a list or an index.
enum Route: Hashable {
    case article(Article)
    case bookmarks
    case account
    case about
    case rss
    case newsletters
}

/// The screen for each route, shared by every tab's stack.
///
/// A screen whose rows the reader can step through (Next article) declares the stack's
/// destinations with `nextArticle`, its model's `article(after:)`. Only the destinations declared
/// closest to the stack's root are used, so this works for a tab's root screen.
struct RouteDestination: View {
    @Environment(SessionModel.self) private var sessionModel
    @Environment(\.userSession) private var session
    let route: Route
    let nextArticle: (@MainActor (String) -> Article?)?

    init(_ route: Route) {
        self.route = route
        nextArticle = nil
    }

    init(_ route: Route, nextArticle: @escaping @MainActor (String) -> Article?) {
        self.route = route
        self.nextArticle = nextArticle
    }

    var body: some View {
        switch route {
        case let .article(article):
            if let session {
                ReaderView(article: article, session: session, nextArticle: nextArticle)
            }
        case .bookmarks:
            if let session { BookmarksView(session: session) }
        case .account:
            AccountView(sessionModel: sessionModel)
        case .about:
            AboutView()
        case .rss:
            if let session { SubscriptionListView(session: session, kind: .rss) }
        case .newsletters:
            if let session { SubscriptionListView(session: session, kind: .newsletters) }
        }
    }
}
