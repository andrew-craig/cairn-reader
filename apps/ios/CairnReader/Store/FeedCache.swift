import Foundation
import OSLog

/// The last Feed page, shown with a stale banner when a refresh fails. Feed is online-only, so
/// this is a snapshot file, not the store: Feed items reach the store and the outbox only
/// through Save to Reads. One JSON file per user in the App Group's Caches, owned by
/// `UserSession` and erased at sign-out.
nonisolated struct FeedCache: Sendable {
    private static let logger = Logger(subsystem: "com.cairnapp.cairnreader", category: "store")

    let url: URL

    static func url(forUser userID: String) -> URL {
        AppGroup.containerURL.appending(path: "Library/Caches/feed-\(userID).json")
    }

    func save(_ articles: [Article]) throws {
        try FileManager.default.createDirectory(at: url.deletingLastPathComponent(), withIntermediateDirectories: true)
        try JSONEncoder().encode(articles).write(to: url, options: .atomic)
    }

    /// The saved page; empty when nothing is saved or the file cannot be read.
    func load() -> [Article] {
        guard let data = try? Data(contentsOf: url) else { return [] }
        do {
            return try JSONDecoder().decode([Article].self, from: data)
        } catch {
            Self.logger.error("Loading the cached Feed failed: \(String(describing: error))")
            return []
        }
    }

    func clear() {
        try? FileManager.default.removeItem(at: url)
    }
}
