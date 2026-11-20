import Foundation
import Synchronization
import Testing
@testable import CairnReader

// Port of apps/mobile/src/services/articlePrefetch.test.ts. The single-flight cases are dropped:
// prefetch only runs inside `Sync`, whose queue runs one operation at a time (SyncTests).

/// The ids a prefetch asked bodies for, in order.
private nonisolated final class Attempts: Sendable {
    private let log = Mutex<[String]>([])
    var ids: [String] { log.withLock { $0 } }
    func record(_ id: String) { log.withLock { $0.append(id) } }
}

private func article(_ id: String, addedAt: TimeInterval = 1000, body: String? = nil) -> Article {
    Article(
        id: id, url: "https://example.com/\(id)", title: id, content: body, isRead: false, isFavorite: false,
        list: .reads, addedAt: Date(timeIntervalSince1970: addedAt))
}

private func prefetch(
    _ store: ArticleStore, online: Bool = true, attempts: Attempts = Attempts(),
    fetch: @escaping @Sendable (String) async throws -> String? = { "body-\($0)" }
) -> ArticlePrefetch {
    ArticlePrefetch(articles: store, isOnline: { online }) { id in
        attempts.record(id)
        return try await fetch(id).map { ($0, nil) }
    }
}

/// Stores `ArticlePrefetch.limit + 1` articles with bodies, newer than any candidate, named
/// "cached-0" (the oldest) upwards, so one cached body lies outside the cap.
private func storeBodiesBeyondTheCap(_ store: ArticleStore) async throws {
    try await store.upsertMany(
        (0...ArticlePrefetch.limit).map { article("cached-\($0)", addedAt: 10_000 + TimeInterval($0), body: "cached") })
}

struct ArticlePrefetchTests {
    @Test func neverPrefetchesWhileOffline() async throws {
        let store = ArticleStore(database: .inMemory())
        try await storeBodiesBeyondTheCap(store)
        try await store.upsertMany([article("a1")])
        let attempts = Attempts()

        try await prefetch(store, online: false, attempts: attempts).run()

        #expect(attempts.ids.isEmpty)
        #expect(try await store.getById("cached-0")?.content == "cached")  // nothing evicted
    }

    @Test func downloadsAndStoresABodyForEveryCandidate() async throws {
        let store = ArticleStore(database: .inMemory())
        try await store.upsertMany([article("a1"), article("a2")])

        try await prefetch(store).run()

        #expect(try await store.getById("a1")?.content == "body-a1")
        #expect(try await store.getById("a2")?.content == "body-a2")
    }

    @Test func evictsBodiesOutsideTheCapAfterTheBatch() async throws {
        let store = ArticleStore(database: .inMemory())
        try await storeBodiesBeyondTheCap(store)
        try await store.upsertMany([article("a1")])

        try await prefetch(store).run()

        #expect(try await store.getById("cached-0")?.content == nil)
        #expect(try await store.getById("cached-1")?.content == "cached")
    }

    @Test func skipsACandidateThatFailsWithANonNetworkErrorAndContinues() async throws {
        let store = ArticleStore(database: .inMemory())
        try await store.upsertMany([article("bad", addedAt: 2000), article("good", addedAt: 1000)])

        try await prefetch(store) { id in
            if id == "bad" { throw HttpError(status: 500, message: nil) }
            return "body-\(id)"
        }.run()

        #expect(try await store.getById("good")?.content == "body-good")
        #expect(try await store.getById("bad")?.content == nil)
    }

    @Test func abortsOnTheFirstNetworkErrorAndStillEvicts() async throws {
        let store = ArticleStore(database: .inMemory())
        try await storeBodiesBeyondTheCap(store)
        try await store.upsertMany([article("only")])

        try await prefetch(store) { _ in throw NetworkError.unreachable }.run()

        #expect(try await store.getById("only")?.content == nil)
        #expect(try await store.getById("cached-0")?.content == nil)
    }

    /// Five candidates against three workers: the first wave (a1–a3) starts at once; a1 finds the
    /// connection gone, so a4 and a5 never start, while a2 and a3 finish.
    @Test func neverStartsACandidateBeyondTheFirstWaveOnceAborted() async throws {
        let store = ArticleStore(database: .inMemory())
        try await store.upsertMany((1...5).map { article("a\($0)", addedAt: 1000 - TimeInterval($0)) })
        let attempts = Attempts()

        try await prefetch(store, attempts: attempts) { id in
            if id == "a1" { throw NetworkError.unreachable }
            try await Task.sleep(for: .milliseconds(100))
            return "body-\(id)"
        }.run()

        #expect(Set(attempts.ids) == ["a1", "a2", "a3"])
        #expect(try await store.getById("a1")?.content == nil)
        #expect(try await store.getById("a2")?.content == "body-a2")
        #expect(try await store.getById("a3")?.content == "body-a3")
    }

    @Test func aNewRunStartsOnceThePreviousFinished() async throws {
        let store = ArticleStore(database: .inMemory())
        try await store.upsertMany([article("a1")])
        let attempts = Attempts()
        let failing = prefetch(store, attempts: attempts) { _ in throw HttpError(status: 404, message: nil) }

        try await failing.run()
        try await failing.run()

        #expect(attempts.ids == ["a1", "a1"])
    }
}
