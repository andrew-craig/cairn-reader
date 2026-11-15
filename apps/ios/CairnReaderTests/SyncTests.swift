import Dispatch
import Foundation
import GRDB
import Synchronization
import Testing
@testable import CairnReader

// Re-expression of apps/mobile/src/services/syncTrigger.test.ts against `Sync` (decision_adb1:
// the trigger's fixed consumer order becomes `Sync.run`, its single-flight guard becomes the
// operation queue), plus the Reads page sync and the queueing rules.

/// Answers PATCHes with `patch`, the Reads list with `page`, and a detail with its body.
private func server(
    patch: StubServer.Reply = SyncFixtures.ok, page: String = SyncFixtures.readsPage([])
) -> StubServer {
    StubServer { request in
        switch (request.httpMethod, request.url?.lastPathComponent) {
        case ("PATCH"?, _): patch
        case ("GET"?, "user-1"?): .json(200, page)
        case ("GET"?, let id?): .json(200, SyncFixtures.detail(id))
        default: .failure(.badServerResponse)
        }
    }
}

private let favoriteTrue = UpdateUserContentRequest(isFavorite: true)
private let serverError = StubServer.Reply.json(500, AuthFixtures.errorBody("Server error"))

struct SyncRunTests {
    @Test func runPrefetchesBodies() async throws {
        let harness = SyncHarness(server())
        try await harness.session.articles.upsertMany([storedArticle("a1")])

        await harness.sync.run()

        #expect(try await harness.session.articles.getById("a1")?.content == SyncFixtures.body("a1"))
    }

    @Test func runDrainsBeforePrefetching() async throws {
        let harness = SyncHarness(server())
        try await harness.session.articles.upsertMany([storedArticle("a1")])
        try harness.enqueue("a1", .isFavorite, favoriteTrue)

        await harness.sync.run()

        #expect(harness.requests == ["PATCH a1", "GET a1"])
        #expect(try harness.outboxCount() == 0)
    }

    /// The jest suite's `runConsumersIsolated` and "does not reject" cases: a failed drain does
    /// not stop the prefetch (and `run` cannot throw).
    @Test func aFailedDrainDoesNotStopThePrefetch() async throws {
        let harness = SyncHarness(server(patch: serverError))
        try await harness.session.articles.upsertMany([storedArticle("a1")])
        try harness.enqueue("a1", .isFavorite, favoriteTrue)

        await harness.sync.run()

        #expect(harness.requests == ["PATCH a1", "GET a1"])
        #expect(try harness.outboxCount() == 1)
        #expect(try await harness.session.articles.getById("a1")?.content == SyncFixtures.body("a1"))
    }

    /// "does not stack concurrent runs": a run requested while one runs joins it.
    @Test func aRunRequestedDuringARunJoinsIt() async throws {
        let entered = AsyncStream<Void>.makeStream()
        let gate = DispatchSemaphore(value: 0)
        let harness = SyncHarness(
            StubServer { request in
                entered.continuation.yield()
                gate.wait()
                return .json(200, SyncFixtures.detail(request.url?.lastPathComponent ?? ""))
            })
        try await harness.session.articles.upsertMany([storedArticle("a1")])

        let first = Task { await harness.sync.run() }
        var iterator = entered.stream.makeAsyncIterator()
        await iterator.next()
        let second = Task { await harness.sync.run() }
        try await Task.sleep(for: .milliseconds(100))
        gate.signal()
        await first.value
        await second.value

        #expect(harness.requests == ["GET a1"])
    }

    @Test func aNewRunStartsOnceThePreviousFinished() async throws {
        let harness = SyncHarness(online: false, server(patch: serverError))
        try harness.enqueue("a1", .isFavorite, favoriteTrue)

        await harness.sync.run()
        await harness.sync.run()

        #expect(harness.requests == ["PATCH a1", "PATCH a1"])
    }
}

struct SyncReadsTests {
    @Test func aFirstPageSyncDrainsFetchesWritesAndPrefetchesInThatOrder() async throws {
        let harness = SyncHarness(server(page: SyncFixtures.readsPage(["a1", "a2"])))
        try harness.enqueue("a0", .isFavorite, favoriteTrue)

        let next = try await harness.sync.syncReads(after: nil)

        #expect(next == nil)
        #expect(Array(harness.requests.prefix(2)) == ["PATCH a0", "GET user-1"])
        #expect(Set(harness.requests.dropFirst(2)) == ["GET a1", "GET a2"])
        #expect(try await harness.session.articles.getById("a2")?.content == SyncFixtures.body("a2"))
    }

    @Test func aLaterPageUsesTheCursorWithoutDrainingOrPrefetching() async throws {
        let harness = SyncHarness(server(page: SyncFixtures.readsPage(["a3"], hasMore: true, cursor: "c2")))
        try harness.enqueue("a0", .isFavorite, favoriteTrue)
        let previous = ReadsCursor(cursor: "c1", last: Date(timeIntervalSince1970: 2_000_000_000))

        let next = try await harness.sync.syncReads(after: previous)

        #expect(harness.requests == ["GET user-1"])
        #expect(harness.server.requests.first?.url?.query()?.contains("cursor=c1") == true)
        #expect(harness.server.requests.first?.url?.query()?.contains("list=reads") == true)
        let last = try #require(ISO8601DateFormatter().date(from: "2025-01-15T10:00:59Z"))
        #expect(next == ReadsCursor(cursor: "c2", last: last))
        #expect(try await harness.session.articles.getById("a3") != nil)
        #expect(try harness.outboxCount() == 1)
    }

    @Test func aFailedFetchThrows() async throws {
        let harness = SyncHarness(online: false, StubServer(replies: []))

        await #expect(throws: NetworkError.self) { try await harness.sync.syncReads(after: nil) }
    }

    /// A Save to Reads made while a page is in flight: the page does not have the article yet.
    /// Its drain runs after the page's `syncPage`, which keeps the article for its outbox row; a
    /// drain in between would remove the row and let the page delete the article.
    @Test func aDrainRequestedDuringAPageFetchRunsAfterThatPagesWrite() async throws {
        let box = Mutex<SyncHarness?>(nil)
        let harness = SyncHarness(
            online: false,
            StubServer { request in
                switch (request.httpMethod, request.url?.lastPathComponent) {
                case ("GET"?, "user-1"?):
                    let harness = box.withLock { $0 }!
                    try! harness.session.database.writer().write { db in
                        try ArticleStore.upsert([storedArticle("saved")], db)
                        try Outbox.enqueue(db, articleID: "saved", field: .list, UpdateUserContentRequest(list: .reads))
                    }
                    harness.sync.requestDrain()
                    return .json(200, SyncFixtures.readsPage(["a1"]))
                case ("PATCH"?, _):
                    return SyncFixtures.ok
                default:
                    return .failure(.badServerResponse)
                }
            })
        box.withLock { $0 = harness }

        _ = try await harness.sync.syncReads(after: nil)
        await harness.sync.drain()

        #expect(harness.requests == ["GET user-1", "PATCH saved"])
        #expect(try await harness.session.articles.getById("saved") != nil)
        #expect(try harness.outboxCount() == 0)
    }

    @Test func aDroppedRowIsFollowedByAFirstPageSync() async throws {
        let harness = SyncHarness(
            online: false,
            server(patch: .json(422, AuthFixtures.errorBody("nope")), page: SyncFixtures.readsPage(["a1"])))
        try await harness.session.articles.upsertMany([storedArticle("a1")])

        try await harness.session.mutations.setFavorite("a1", isFavorite: true)
        await harness.sync.drain()

        #expect(harness.requests == ["PATCH a1", "GET user-1"])
        #expect(try await harness.session.articles.getById("a1")?.isFavorite == false)
    }

    @Test func requestsAfterStopDoNothing() async throws {
        let harness = SyncHarness(server())
        try await harness.session.articles.upsertMany([storedArticle("a1")])
        try harness.enqueue("a1", .isFavorite, favoriteTrue)
        await harness.sync.stop()

        harness.sync.requestDrain()
        await harness.sync.drain()
        await harness.sync.run()
        await #expect(throws: CancellationError.self) { try await harness.sync.syncReads(after: nil) }

        #expect(harness.requests.isEmpty)
    }
}
