import SwiftUI

@main
struct CairnReaderApp: App {
    var body: some Scene {
        WindowGroup {
            RootView()
        }
    }
}

/// Placeholder until the tab structure (Feed, Reads, You) lands.
struct RootView: View {
    var body: some View {
        ContentUnavailableView("Cairn Reader", systemImage: "books.vertical")
    }
}
