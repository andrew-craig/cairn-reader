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

    var body: some View {
        TabView(selection: $selection) {
            Tab("Feed", systemImage: "newspaper", value: MainTab.feed) {
                NavigationStack { FeedPlaceholder() }
            }
            Tab("Reads", systemImage: "book", value: MainTab.reads) {
                NavigationStack { ReadsView(session: session) }
            }
            Tab("You", systemImage: "person.crop.circle", value: MainTab.you) {
                NavigationStack { YouView(read: session.read) }
            }
        }
        .environment(\.userSession, session)
        .task {
            for await rejection in session.outbox.rejections {
                rejectionMessage = Self.message(for: rejection)
            }
        }
        .alert(
            "Error", isPresented: Binding(get: { rejectionMessage != nil }, set: { if !$0 { rejectionMessage = nil } })
        ) {
            Button("OK") {}
        } message: {
            Text(rejectionMessage ?? "")
        }
    }

    /// What the user is told when the server refuses an archive or a Save to Reads (the
    /// wording of the React Native app).
    static func message(for rejection: Outbox.Rejection) -> String {
        rejection.field == .list ? "Failed to save to Reads" : "Failed to archive article"
    }
}
