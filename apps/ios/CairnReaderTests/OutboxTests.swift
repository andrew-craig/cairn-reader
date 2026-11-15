import Foundation
import GRDB
import Synchronization
import Testing
@testable import CairnReader

// Port of apps/mobile/src/services/outbox.test.ts, minus `attempts` and `pendingSaveIds` (the
// pending flag is the store's observation, tested in ArticleStoreTests), plus the cases for the
// outbox being the only write path (decision 9).

private let favoriteTrue = UpdateUserContentRequest(isFavorite: true)

struct OutboxEnqueueTests {
    @Test func coalescesRepeatedWritesAndPreservesCreatedAt() async throws {
        let harness = OutboxHarness(always: OutboxHarness.ok)
        try harness.enqueue("a1", .scrollPosition, UpdateUserContentRequest(scrollPosition: 0.1))
        let first = try harness.rows()
        try await Task.sleep(for: .milliseconds(5))

        try harness.enqueue("a1", .scrollPosition, UpdateUserContentRequest(scrollPosition: 0.9))

        let rows = try harness.rows()
        #expect(rows.count == 1)
        #expect(rows.first?.payload == #"{"scroll_position":0.9}"#)
        #expect(rows.first?.createdAt == first.first?.createdAt)
    }

    @Test func keepsSeparateRowsForDifferentFieldsOnTheSameArticle() async throws {
        let harness = OutboxHarness(always: OutboxHarness.ok)
        try harness.enqueue("a1", .isFavorite, favoriteTrue)
        try harness.enqueue("a1", .scrollPosition, UpdateUserContentRequest(scrollPosition: 0.5))

        await harness.outbox.drain()

        #expect(harness.patches.count == 2)
    }

    @Test func doesNotDisturbADifferentArticlesRows() async throws {
        let harness = OutboxHarness(always: OutboxHarness.ok)
        try harness.enqueue("a1", .isFavorite, favoriteTrue)
        try harness.enqueue("a2", .status, UpdateUserContentRequest(status: .archived))

        await harness.outbox.drain()

        #expect(harness.patches.map(\.id).sorted() == ["a1", "a2"])
        let a1 = harness.patches.first { $0.id == "a1" }
        let a2 = harness.patches.first { $0.id == "a2" }
        #expect(a1?.body as? [String: Bool] == ["is_favorite": true])
        #expect(a2?.body as? [String: String] == ["status": "archived"])
    }
}

struct OutboxDrainOrderingTests {
    @Test func replaysRowsInCreatedAtOrder() async throws {
        let harness = OutboxHarness(always: OutboxHarness.ok)
        try harness.seed("a2", "is_favorite", #"{"is_favorite":true}"#, createdAt: 200)
        try harness.seed("a1", "is_favorite", #"{"is_favorite":true}"#, createdAt: 100)

        await harness.outbox.drain()

        #expect(harness.patches.map(\.id) == ["a1", "a2"])
    }

    @Test func breaksACreatedAtTieWithInsertionOrder() async throws {
        let harness = OutboxHarness(always: OutboxHarness.ok)
        try harness.seed("first", "is_favorite", #"{"is_favorite":true}"#, createdAt: 500)
        try harness.seed("second", "is_favorite", #"{"is_favorite":true}"#, createdAt: 500)

        await harness.outbox.drain()

        #expect(harness.patches.map(\.id) == ["first", "second"])
    }
}

struct OutboxDrainOutcomeTests {
    @Test func aSuccessfulWriteDeletesTheRow() async throws {
        let harness = OutboxHarness(always: OutboxHarness.ok)
        try harness.enqueue("a1", .isFavorite, favoriteTrue)

        await harness.outbox.drain()

        #expect(try harness.rows().isEmpty)
    }

    @Test func aDefinitiveRejectionDropsTheRowAndContinuesTheBatch() async throws {
        let harness = OutboxHarness(
            StubServer(replies: [.json(422, AuthFixtures.errorBody("nope")), OutboxHarness.ok]))
        try harness.seed("a1", "is_favorite", #"{"is_favorite":true}"#, createdAt: 1)
        try harness.seed("a2", "is_favorite", #"{"is_favorite":true}"#, createdAt: 2)

        await harness.outbox.drain()

        #expect(try harness.rows().isEmpty)
        #expect(harness.patches.count == 2)
    }

    @Test(arguments: [
        StubServer.Reply.failure(.notConnectedToInternet),
        .json(401, AuthFixtures.errorBody("Unauthorized")),
        .json(500, AuthFixtures.errorBody("Server error")),
    ])
    func aTransientFailureKeepsTheRowAndHaltsTheBatch(reply: StubServer.Reply) async throws {
        let harness = OutboxHarness(always: reply)
        try harness.seed("a1", "is_favorite", #"{"is_favorite":true}"#, createdAt: 1)
        try harness.seed("a2", "is_favorite", #"{"is_favorite":true}"#, createdAt: 2)

        await harness.outbox.drain()

        #expect(harness.patches.map(\.id) == ["a1"])
        #expect(try harness.rows().map(\.articleID) == ["a1", "a2"])
    }

    @Test func noSessionKeepsTheRows() async throws {
        let harness = OutboxHarness(session: nil, always: OutboxHarness.ok)
        try harness.enqueue("a1", .isFavorite, favoriteTrue)

        await harness.outbox.drain()

        #expect(harness.server.requests.isEmpty)
        #expect(try harness.rows().count == 1)
    }

    @Test func aSecondDrainRetriesAHaltedRow() async throws {
        let harness = OutboxHarness(StubServer(replies: [.failure(.notConnectedToInternet), OutboxHarness.ok]))
        try harness.enqueue("a1", .isFavorite, favoriteTrue)

        await harness.outbox.drain()
        #expect(try harness.rows().count == 1)
        await harness.outbox.drain()

        #expect(try harness.rows().isEmpty)
    }

    @Test func classifiesErrors() {
        #expect(Outbox.isRetryable(NetworkError.unreachable))
        #expect(Outbox.isRetryable(NetworkError.timeout))
        #expect(Outbox.isRetryable(HttpError(status: 401, message: nil)))
        #expect(Outbox.isRetryable(HttpError(status: 500, message: nil)))
        #expect(Outbox.isRetryable(HttpError(status: 503, message: nil)))
        #expect(!Outbox.isRetryable(HttpError(status: 404, message: nil)))
        #expect(!Outbox.isRetryable(HttpError(status: 422, message: nil)))
        #expect(!Outbox.isRetryable(AuthError.sessionExpired))
    }
}

struct OutboxSaveToReadsTests {
    @Test func replaysAListRowAndKeepsTheStoredArticle() async throws {
        let harness = OutboxHarness(always: OutboxHarness.ok)
        try await harness.store.upsertMany([storedArticle("a1")])
        try harness.enqueue("a1", .list, UpdateUserContentRequest(list: .reads))

        await harness.outbox.drain()

        #expect(harness.patches.first?.body as? [String: String] == ["list": "reads"])
        #expect(try harness.rows().isEmpty)
        #expect(try await harness.store.getById("a1") != nil)
    }

    @Test func aRejectedSaveDeletesTheArticleAndYieldsARejection() async throws {
        let harness = OutboxHarness(always: .json(404, Fixtures.error404))
        try await harness.store.upsertMany([storedArticle("a1")])
        try harness.enqueue("a1", .list, UpdateUserContentRequest(list: .reads))

        await harness.outbox.drain()

        #expect(try harness.rows().isEmpty)
        #expect(try await harness.store.getById("a1") == nil)
        #expect(await harness.finishedRejections() == [Outbox.Rejection(articleID: "a1", field: .list)])
    }
}

struct OutboxRejectionTests {
    @Test func aRejectedArchiveYieldsARejection() async throws {
        let harness = OutboxHarness(always: .json(404, Fixtures.error404))
        try harness.enqueue("a1", .status, UpdateUserContentRequest(status: .archived))

        await harness.outbox.drain()

        #expect(try harness.rows().isEmpty)
        #expect(await harness.finishedRejections() == [Outbox.Rejection(articleID: "a1", field: .status)])
    }

    @Test func otherDroppedEditsYieldNone() async throws {
        let harness = OutboxHarness(always: .json(422, AuthFixtures.errorBody("nope")))
        try harness.enqueue("a1", .status, UpdateUserContentRequest(status: .completed))
        try harness.enqueue("a2", .isFavorite, favoriteTrue)

        await harness.outbox.drain()

        #expect(try harness.rows().isEmpty)
        #expect(await harness.finishedRejections().isEmpty)
    }
}

struct OutboxWritePathTests {
    /// Favourite offline, unfavourite online before the drain: the server must end on `false`,
    /// never see the older `true` after it.
    @Test func favouriteOfflineThenUnfavouriteOnlineEndsFalse() async throws {
        let online = Mutex(false)
        let harness = OutboxHarness(
            StubServer { _ in online.withLock { $0 } ? OutboxHarness.ok : OutboxHarness.unreachable })
        try await harness.store.upsertMany([storedArticle("a1")])

        try await harness.mutations.setFavorite("a1", isFavorite: true)
        await harness.outbox.drain()
        #expect(try harness.rows().count == 1)

        online.withLock { $0 = true }
        try await harness.mutations.setFavorite("a1", isFavorite: false)
        await harness.outbox.drain()

        #expect(try harness.rows().isEmpty)
        let delivered = harness.server.requests.filter { $0.httpMethod == "PATCH" }
        #expect(delivered.last?.jsonBody as? [String: Bool] == ["is_favorite": false])
        #expect(try await harness.store.getById("a1")?.isFavorite == false)
    }

    /// An edit coalesced into a row while its old payload is in flight is sent by the next pass.
    @Test func aPayloadReplacedMidRequestIsNotLost() async throws {
        let first = Mutex(true)
        let harnessBox = Mutex<OutboxHarness?>(nil)
        let harness = OutboxHarness(
            StubServer { _ in
                if first.withLock({ let was = $0; $0 = false; return was }) {
                    // What a mutation does: queue the edit, then ask for a drain.
                    let harness = harnessBox.withLock { $0 }!
                    try! harness.enqueue("a1", .isFavorite, UpdateUserContentRequest(isFavorite: false))
                    harness.outbox.requestDrain()
                }
                return OutboxHarness.ok
            })
        harnessBox.withLock { $0 = harness }
        try harness.enqueue("a1", .isFavorite, favoriteTrue)

        await harness.outbox.drain()

        #expect(harness.patches.map { $0.body as? [String: Bool] } == [["is_favorite": true], ["is_favorite": false]])
        #expect(try harness.rows().isEmpty)
    }

    @Test func requestsAfterStopAreIgnored() async throws {
        let harness = OutboxHarness(always: OutboxHarness.ok)
        try harness.enqueue("a1", .isFavorite, favoriteTrue)
        await harness.outbox.stop()

        harness.outbox.requestDrain()
        await harness.outbox.drain()

        #expect(harness.server.requests.isEmpty)
        #expect(try harness.rows().count == 1)
    }
}
