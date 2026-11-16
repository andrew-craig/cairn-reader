import SwiftUI

extension EnvironmentValues {
    /// The signed-in user's session; set by `MainTabs`.
    @Entry var userSession: UserSession?
}

enum MainTab: Hashable {
    case feed, reads, you
}

/// Feed, Reads and You, each in its own `NavigationStack`. Created per signed-in user.
struct MainTabs: View {
    let session: UserSession

    @State private var selection = MainTab.reads
    @State private var rejectionMessage: String?
    @State private var you: YouModel

    init(session: UserSession) {
        self.session = session
        _you = State(initialValue: YouModel(read: session.read))
    }

    var body: some View {
        TabView(selection: $selection) {
            Tab("Feed", image: icon("feed", for: .feed), value: MainTab.feed) {
                NavigationStack { FeedView(session: session) }
            }
            Tab("Reads", image: icon("read", for: .reads), value: MainTab.reads) {
                NavigationStack { ReadsView(session: session) }
            }
            Tab("You", image: icon("profile", for: .you), value: MainTab.you) {
                NavigationStack { YouView(model: you) }
            }
        }
        .environment(\.userSession, session)
        .task {
            for await rejection in session.outbox.rejections {
                rejectionMessage = Self.message(for: rejection)
            }
        }
        // Here, not in `YouView`: a favourite sent while You is hidden must still mark it stale.
        .task { await you.followFavoriteChanges(session.outbox.favoriteChanges) }
        .alert(
            "Error", isPresented: Binding(get: { rejectionMessage != nil }, set: { if !$0 { rejectionMessage = nil } })
        ) {
            Button("OK") {}
        } message: {
            Text(rejectionMessage ?? "")
        }
    }

    /// A tab's icon: the filled variant while the tab is selected, the outline otherwise (as the
    /// React Native tab bar).
    private func icon(_ name: String, for tab: MainTab) -> String {
        selection == tab ? "\(name)-fill" : name
    }

    /// What the user is told when the server refuses an archive or a Save to Reads (the
    /// wording of the React Native app).
    static func message(for rejection: Outbox.Rejection) -> String {
        rejection.field == .list ? "Failed to save to Reads" : "Failed to archive article"
    }
}
