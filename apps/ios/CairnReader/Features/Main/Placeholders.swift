import SwiftUI

// Stand-ins until the screens land; each is replaced by its own feature task.

struct FeedPlaceholder: View {
    var body: some View {
        ContentUnavailableView("Feed", systemImage: "newspaper")
            .navigationTitle("Feed")
            .navigationDestination(for: Route.self, destination: RouteDestination.init)
    }
}

struct ReadsPlaceholder: View {
    var body: some View {
        ContentUnavailableView("Reads", systemImage: "book")
            .navigationTitle("Reads")
            .navigationDestination(for: Route.self, destination: RouteDestination.init)
    }
}
