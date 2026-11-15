import Foundation
@testable import CairnReader

/// A database file in a fresh temporary directory.
nonisolated func temporaryDatabaseURL() throws -> URL {
    let directory = FileManager.default.temporaryDirectory.appending(path: UUID().uuidString, directoryHint: .isDirectory)
    try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
    return directory.appending(path: "store.sqlite")
}

nonisolated func storedArticle(_ id: String) -> Article {
    Article(
        id: id, url: "https://example.com/\(id)", title: id, isRead: false, isFavorite: false, list: .reads,
        addedAt: Date(timeIntervalSince1970: 1000))
}
