import SwiftUI

/// The profile hub: the account name, the counted lists, then Account and About.
struct YouView: View {
    @Environment(SessionModel.self) private var sessionModel
    /// Owned by `MainTabs`, which follows the outbox's favourite changes into it.
    let model: YouModel

    init(model: YouModel) {
        self.model = model
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
        .toolbarTitleDisplayMode(.inlineLarge)
        .navigationDestination(for: Route.self, destination: RouteDestination.init)
        // Not `.task`: leaving the tab mid-fetch would cancel it and show a failure.
        .onAppear {
            model.isVisible = true
            Task { await model.refreshIfStale() }
        }
        .onDisappear { model.isVisible = false }
    }

    @ViewBuilder
    private func count(_ value: Int?) -> some View {
        if let value {
            Text(value, format: .number)
        }
    }
}
