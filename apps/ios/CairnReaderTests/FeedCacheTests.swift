import Foundation
import Testing
@testable import CairnReader

// Port of apps/mobile/src/services/feedCache.test.ts.

private func temporaryCache() -> FeedCache {
    FeedCache(url: FileManager.default.temporaryDirectory.appending(path: "\(UUID().uuidString)/feed.json"))
}

private let article = Article(
    id: "a1", url: "https://example.com/a1", title: "One", description: "A description", isRead: false,
    isFavorite: true, list: .feed, addedAt: Date(timeIntervalSince1970: 1.5), scrollPosition: 0.25)

struct FeedCacheTests {
    @Test func roundTripsASavedPage() throws {
        let cache = temporaryCache()
        try cache.save([article])
        #expect(cache.load() == [article])
    }

    @Test func loadsAnEmptyListWhenNothingIsCached() {
        #expect(temporaryCache().load().isEmpty)
    }

    @Test func loadsAnEmptyListWhenTheCachedValueIsCorrupt() throws {
        let cache = temporaryCache()
        try FileManager.default.createDirectory(
            at: cache.url.deletingLastPathComponent(), withIntermediateDirectories: true)
        try Data("{not json".utf8).write(to: cache.url)
        #expect(cache.load().isEmpty)
    }

    @Test func clearRemovesTheSnapshot() throws {
        let cache = temporaryCache()
        try cache.save([article])
        cache.clear()
        #expect(cache.load().isEmpty)
    }
}
