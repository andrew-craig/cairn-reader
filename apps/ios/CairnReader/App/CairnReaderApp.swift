import SwiftUI

@main
struct CairnReaderApp: App {
    init() {
        Typography.applyBarAppearance()
    }

    var body: some Scene {
        WindowGroup {
            RootView()
                .font(Typography.body)
        }
    }
}

/// Placeholder until the tab structure (Feed, Reads, You) lands.
struct RootView: View {
    var body: some View {
        ContentUnavailableView("Cairn Reader", systemImage: "books.vertical")
    }
}
