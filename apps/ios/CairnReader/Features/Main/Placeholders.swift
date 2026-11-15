import SwiftUI

// Stand-ins until the screens land; each is replaced by its own feature task.

struct FeedPlaceholder: View {
    var body: some View {
        ContentUnavailableView("Feed", systemImage: "newspaper")
            .navigationTitle("Feed")
            .navigationDestination(for: Route.self, destination: RoutePlaceholder.init)
    }
}

struct ReadsPlaceholder: View {
    var body: some View {
        ContentUnavailableView("Reads", systemImage: "book")
            .navigationTitle("Reads")
            .navigationDestination(for: Route.self, destination: RoutePlaceholder.init)
    }
}

struct YouPlaceholder: View {
    @Environment(SessionModel.self) private var sessionModel

    var body: some View {
        List {
            NavigationLink("Bookmarks", value: Route.bookmarks)
            NavigationLink("Account", value: Route.account)
            NavigationLink("RSS", value: Route.rss)
            NavigationLink("Newsletters", value: Route.newsletters)
            NavigationLink("About", value: Route.about)
            // Temporary, for checking sign-out until Account has its own.
            Button("Sign out", role: .destructive) {
                Task { await sessionModel.logout() }
            }
        }
        .navigationTitle("You")
        .navigationDestination(for: Route.self, destination: RoutePlaceholder.init)
    }
}

struct RoutePlaceholder: View {
    let route: Route

    init(_ route: Route) {
        self.route = route
    }

    var body: some View {
        switch route {
        case let .article(article):
            ContentUnavailableView(article.title, systemImage: "doc.text")
        case .bookmarks:
            ContentUnavailableView("Bookmarks", systemImage: "bookmark")
        case .account:
            ContentUnavailableView("Account", systemImage: "person")
        case .about:
            ContentUnavailableView("About", systemImage: "info.circle")
        case .rss:
            ContentUnavailableView("RSS", systemImage: "dot.radiowaves.up.forward")
        case .newsletters:
            ContentUnavailableView("Newsletters", systemImage: "envelope")
        }
    }
}
