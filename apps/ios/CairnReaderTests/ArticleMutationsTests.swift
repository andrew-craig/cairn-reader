import Foundation
import GRDB
import Testing
@testable import CairnReader

// Re-expression of apps/mobile/src/services/articleMutations.test.ts: a mutation
// is one transaction (store change plus outbox row) and never sends or throws for the backend.
// The jest cases about enqueuing on a retryable error or surfacing a 4xx become the drain's
// outcomes (OutboxTests) and "mutations do not throw for the backend" below.

/// A harness whose server is unreachable, so queued rows stay put.
private func offline() -> OutboxHarness {
    OutboxHarness(always: OutboxHarness.unreachable)
}

private func stored(_ harness: OutboxHarness, _ id: String = "a1") async throws {
    try await harness.store.upsertMany([storedArticle(id)])
}

struct ArticleMutationsTests {
    @Test func markCompletedStoresReadAndQueuesStatus() async throws {
        let harness = offline()
        try await stored(harness)

        try await harness.mutations.markCompleted("a1", readAt: Date(timeIntervalSince1970: 500))

        let article = try await harness.store.getById("a1")
        #expect(article?.isRead == true)
        #expect(article?.readAt == Date(timeIntervalSince1970: 500))
        #expect(try harness.rows().map(\.payload) == [#"{"status":"completed"}"#])
    }

    @Test func markReadingQueuesStatusWithoutTouchingTheStore() async throws {
        let harness = offline()
        try await stored(harness)

        try await harness.mutations.markReading("a1")

        #expect(try await harness.store.getById("a1") == storedArticle("a1"))
        #expect(try harness.rows().map(\.payload) == [#"{"status":"reading"}"#])
    }

    @Test func completedReplacesReadingInTheSameRow() async throws {
        let harness = offline()
        try await stored(harness)

        try await harness.mutations.markReading("a1")
        try await harness.mutations.markCompleted("a1", readAt: Date(timeIntervalSince1970: 500))

        #expect(try harness.rows().map(\.payload) == [#"{"status":"completed"}"#])
    }

    @Test func saveScrollPositionStoresAndQueuesUnderScrollPosition() async throws {
        let harness = offline()
        try await stored(harness)

        try await harness.mutations.saveScrollPosition("a1", fraction: 0.42)

        #expect(try await harness.store.getById("a1")?.scrollPosition == 0.42)
        let rows = try harness.rows()
        #expect(rows.map(\.field) == ["scroll_position"])
        #expect(rows.map(\.payload) == [#"{"scroll_position":0.42}"#])
    }

    @Test func setFavoriteStoresAndQueues() async throws {
        let harness = offline()
        try await stored(harness)

        try await harness.mutations.setFavorite("a1", isFavorite: true)

        #expect(try await harness.store.getById("a1")?.isFavorite == true)
        #expect(try harness.rows().map(\.field) == ["is_favorite"])
        #expect(try harness.rows().map(\.payload) == [#"{"is_favorite":true}"#])
    }

    @Test func archiveRemovesFromTheStoreAndQueuesArchived() async throws {
        let harness = offline()
        try await stored(harness)

        try await harness.mutations.archive("a1")

        #expect(try await harness.store.getById("a1") == nil)
        #expect(try harness.rows().map(\.payload) == [#"{"status":"archived"}"#])
    }

    @Test func saveToReadsStoresAsReadsItemAddedNowAndQueuesList() async throws {
        let harness = offline()
        var feedItem = storedArticle("a1")
        feedItem.list = .feed
        feedItem.content = "<p>Body</p>"
        let before = Date()

        try await harness.mutations.saveToReads(feedItem)

        let article = try #require(try await harness.store.getById("a1"))
        #expect(article.list == .reads)
        // The store keeps `added_at` as seconds since 1970, so the round trip can shift it by a
        // fraction of a microsecond either way.
        #expect(article.addedAt.timeIntervalSince(before) > -0.001)
        #expect(article.addedAt.timeIntervalSinceNow < 0.001)
        #expect(article.content == "<p>Body</p>")
        #expect(try harness.rows().map(\.field) == ["list"])
        #expect(try harness.rows().map(\.payload) == [#"{"list":"reads"}"#])
    }

    @Test func saveToReadsWithoutABodyStoresNone() async throws {
        let harness = offline()

        try await harness.mutations.saveToReads(storedArticle("a1"))

        #expect(try await harness.store.getById("a1")?.content == nil)
    }
}

struct ArticleMutationsAtomicityTests {
    /// Makes every outbox insert fail, as a full disk or a broken row would.
    private func failingOutbox() throws -> OutboxHarness {
        let harness = offline()
        try harness.database.writer().write { db in
            try db.execute(sql: "CREATE TRIGGER fail BEFORE INSERT ON outbox BEGIN SELECT RAISE(ABORT, 'full'); END")
        }
        return harness
    }

    @Test func aFailedOutboxWriteLeavesNoStoreChange() async throws {
        let harness = try failingOutbox()
        try await stored(harness, "a1")
        try await stored(harness, "a2")

        await #expect(throws: DatabaseError.self) {
            try await harness.mutations.markCompleted("a1", readAt: Date(timeIntervalSince1970: 500))
        }
        await #expect(throws: DatabaseError.self) {
            try await harness.mutations.setFavorite("a1", isFavorite: true)
        }
        await #expect(throws: DatabaseError.self) { try await harness.mutations.saveScrollPosition("a1", fraction: 0.5) }
        await #expect(throws: DatabaseError.self) { try await harness.mutations.archive("a2") }
        await #expect(throws: DatabaseError.self) { try await harness.mutations.saveToReads(storedArticle("a3")) }

        #expect(try await harness.store.getById("a1") == storedArticle("a1"))
        #expect(try await harness.store.getById("a2") == storedArticle("a2"))
        #expect(try await harness.store.getById("a3") == nil)
        #expect(try harness.rows().isEmpty)
    }

    @Test func aFailedStoreWriteLeavesNoOutboxRow() async throws {
        let harness = offline()
        try await harness.database.writer().write { db in
            try db.execute(sql: "CREATE TRIGGER fail BEFORE UPDATE ON articles BEGIN SELECT RAISE(ABORT, 'full'); END")
        }
        try await stored(harness)

        await #expect(throws: DatabaseError.self) { try await harness.mutations.setFavorite("a1", isFavorite: true) }

        #expect(try harness.rows().isEmpty)
    }
}

struct ArticleMutationsSendingTests {
    @Test func aMutationSendsNothingItself() async throws {
        let harness = OutboxHarness(always: OutboxHarness.ok)  // its mutations never start a drain
        try await stored(harness)

        try await harness.mutations.setFavorite("a1", isFavorite: true)
        try await harness.mutations.archive("a1")

        #expect(harness.server.requests.isEmpty)
        #expect(try harness.rows().count == 2)
    }

    @Test func aMutationRequestsADrainThatSendsTheQueuedEdit() async throws {
        let harness = OutboxHarness(always: OutboxHarness.ok)
        try await stored(harness)

        try await harness.mutations.setFavorite("a1", isFavorite: true)
        #expect(harness.drainRequests.count == 1)
        await harness.outbox.drain()

        #expect(harness.patches.map(\.id) == ["a1"])
        #expect(harness.patches.first?.body as? [String: Bool] == ["is_favorite": true])
        #expect(try harness.rows().isEmpty)
    }

    @Test func aMutationDoesNotThrowForTheBackendsAnswer() async throws {
        let harness = OutboxHarness(always: .json(422, AuthFixtures.errorBody("nope")))
        try await stored(harness)

        try await harness.mutations.markCompleted("a1", readAt: Date(timeIntervalSince1970: 500))
        await harness.outbox.drain()

        #expect(try harness.rows().isEmpty)
        // The store keeps the local state until the next Reads sync restores the server's.
        #expect(try await harness.store.getById("a1")?.isRead == true)
    }
}
