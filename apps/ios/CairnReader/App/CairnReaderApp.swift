import SwiftUI

@main
struct CairnReaderApp: App {
    @Environment(\.scenePhase) private var scenePhase

    init() {
        Typography.applyBarAppearance()
    }

    var body: some Scene {
        WindowGroup {
            RootView()
                .font(Typography.body)
        }
        .onChange(of: scenePhase) { _, phase in
            AppDatabase.setSuspended(phase == .background)
        }
    }
}

/// Placeholder until the tab structure (Feed, Reads, You) lands.
struct RootView: View {
    var body: some View {
        ContentUnavailableView("Cairn Reader", systemImage: "books.vertical")
    }
}
