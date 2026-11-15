import SwiftUI

/// The profile hub: the account name, the counted lists, then Account and About.
struct YouView: View {
    @Environment(SessionModel.self) private var sessionModel
    @State private var model: YouModel

    init(read: ReadService) {
        _model = State(initialValue: YouModel(read: read))
    }

    var body: some View {
        List {
            Section {
                NavigationLink(value: Route.bookmarks) {
                    LabeledContent("Bookmarks") { count(model.bookmarksCount) }
                }
                NavigationLink(value: Route.rss) {
                    LabeledContent("RSS") { count(model.rssCount) }
                }
                NavigationLink(value: Route.newsletters) {
                    LabeledContent("Newsletters") { count(model.newslettersCount) }
                }
            } header: {
                Text(sessionModel.user?.email ?? "Anonymous user")
                    .textCase(nil)
            } footer: {
                if let message = model.errorMessage {
                    Text(message).foregroundStyle(Color(.error))
                }
            }

            Section {
                NavigationLink("Account", value: Route.account)
                NavigationLink("About", value: Route.about)
            }
        }
        .listStyle(.insetGrouped)
        .navigationTitle("You")
        .navigationDestination(for: Route.self, destination: RouteDestination.init)
        // Not `.task`: leaving the tab mid-fetch would cancel it and show a failure.
        .onAppear {
            Task { await model.refreshIfStale() }
        }
    }

    @ViewBuilder
    private func count(_ value: Int?) -> some View {
        if let value {
            Text(value, format: .number)
        }
    }
}
