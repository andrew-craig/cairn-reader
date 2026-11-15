import SwiftUI

/// Every destination a tab's `NavigationStack` can push, typed in one place. The article carries
/// the `Article` value only, never a list or an index (decision 8).
enum Route: Hashable {
    case article(Article)
    case bookmarks
    case account
    case about
    case rss
    case newsletters
}

/// The screen for each route, shared by every tab's stack. Routes whose screen has not landed yet
/// show a stand-in; each is replaced by its own feature task.
struct RouteDestination: View {
    @Environment(SessionModel.self) private var sessionModel
    @Environment(\.userSession) private var session
    let route: Route

    init(_ route: Route) {
        self.route = route
    }

    var body: some View {
        switch route {
        case let .article(article):
            ContentUnavailableView(article.title, systemImage: "doc.text")
        case .bookmarks:
            if let session { BookmarksView(session: session) }
        case .account:
            AccountView(sessionModel: sessionModel)
        case .about:
            AboutView()
        case .rss:
            ContentUnavailableView("RSS", systemImage: "dot.radiowaves.up.forward")
        case .newsletters:
            ContentUnavailableView("Newsletters", systemImage: "envelope")
        }
    }
}
