import OSLog
import SwiftUI

extension EnvironmentValues {
    /// Set by the composition root.
    @Entry var systemService: SystemService?
}

/// The app's and the server's versions.
struct AboutView: View {
    @Environment(\.systemService) private var systemService
    @State private var serverVersion = "Loading…"

    var body: some View {
        Form {
            LabeledContent("App version", value: Self.appVersion)
            LabeledContent("Server version", value: serverVersion)
        }
        .navigationTitle("About")
        .toolbarTitleDisplayMode(.inline)
        .task { serverVersion = await Self.serverVersionText(systemService) }
    }

    static var appVersion: String {
        Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? ""
    }

    /// The server's version, "Unknown" when it reports none (the cloud backend may not), or
    /// "Unavailable" when it could not be read.
    static func serverVersionText(_ service: SystemService?) async -> String {
        guard let service else { return "Unavailable" }
        do {
            return try await service.serverVersion() ?? "Unknown"
        } catch {
            Logger(subsystem: "com.cairnapp.cairnreader", category: "about")
                .error("Fetching the server version failed: \(error.localizedDescription)")
            return "Unavailable"
        }
    }
}
