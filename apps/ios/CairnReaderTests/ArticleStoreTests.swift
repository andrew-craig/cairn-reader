import Foundation
import GRDB
import Synchronization
import Testing
@testable import CairnReader

// Port of apps/mobile/src/services/articleStore.test.ts against a real in-memory database.
// Not ported: the v1 -> v2 migration cases (there is no RN migration ladder here; the schema is
// created in one migration, epic_bc48 decision 4). Outbox rows are inserted directly until the
// outbox itself is ported (task_387d).

private func makeArticle(
    _ id: String = "a1", title: String = "Article One", isRead: Bool = false, isFavorite: Bool = false,
    addedAt: TimeInterval = 1000, contentHash: String? = nil
) -> Article {
    Article(
        id: id, url: "https://example.com/\(id)", title: title, contentHash: contentHash, isRead: isRead,
        isFavorite: isFavorite, list: .reads, addedAt: Date(timeIntervalSince1970: addedAt))
}

private func enqueue(
    _ database: AppDatabase, _ articleID: String, _ field: String, _ payload: String, createdAt: Int = 0
) async throws {
    try await database.queue().write { db in
        try db.execute(
            sql: "INSERT INTO outbox (article_id, field, payload, created_at) VALUES (?, ?, ?, ?)",
            arguments: [articleID, field, payload, createdAt])
    }
}

struct ArticleStoreTests {
    let database = AppDatabase.inMemory()
    var store: ArticleStore { ArticleStore(database: database) }

    // MARK: upsertMany

    @Test func insertsNewArticles() async throws {
        try await store.upsertMany([makeArticle("a1"), makeArticle("a2")])

        #expect(await store.listRecent(limit: 10).count == 2)
    }

    @Test func updatesAnExistingRowInsteadOfDuplicatingIt() async throws {
        try await store.upsertMany([makeArticle("a1", title: "Old Title")])
        try await store.upsertMany([makeArticle("a1", title: "New Title")])

        let recent = await store.listRecent(limit: 10)
        #expect(recent.map(\.title) == ["New Title"])
    }

    @Test func doesNotDeleteRowsAbsentFromTheIncomingPage() async throws {
        try await store.upsertMany([makeArticle("a1")])
        try await store.upsertMany([makeArticle("a2")])

        #expect(await store.listRecent(limit: 10).map(\.id).sorted() == ["a1", "a2"])
    }

    @Test func roundTripsEveryField() async throws {
        var article = makeArticle("a1", isRead: true, isFavorite: true, contentHash: "h1")
        article.description = "Description"
        article.content = "<p>Body</p>"
        article.imageUrl = "https://example.com/image.jpg"
        article.author = "Author"
        article.publishedDate = Date(timeIntervalSince1970: 500.5)
        article.readingTime = 7
        article.readAt = Date(timeIntervalSince1970: 2000.25)
        article.scrollPosition = 0.42

        try await store.upsertMany([article])

        #expect(await store.getById("a1") == article)
    }

    // MARK: listRecent

    @Test func listRecentOrdersByAddedAtDescendingAndRespectsTheLimit() async throws {
        try await store.upsertMany([
            makeArticle("old", addedAt: 100),
            makeArticle("newest", addedAt: 300),
            makeArticle("middle", addedAt: 200),
        ])

        #expect(await store.listRecent(limit: 2).map(\.id) == ["newest", "middle"])
    }

    // MARK: listFavorites

    @Test func listFavoritesReturnsOnlyFavorites() async throws {
        try await store.upsertMany([
            makeArticle("fav", isFavorite: true, addedAt: 100),
            makeArticle("not-fav", isFavorite: false, addedAt: 200),
        ])

        #expect(await store.listFavorites().map(\.id) == ["fav"])
    }

    // MARK: listPendingSaves

    @Test func listPendingSavesReturnsQueuedSavesMostRecentlyQueuedFirst() async throws {
        try await store.upsertMany([
            makeArticle("first"), makeArticle("second"), makeArticle("not-queued"), makeArticle("favorited"),
        ])
        try await enqueue(database, "first", "list", #"{"list":"reads"}"#, createdAt: 100)
        try await enqueue(database, "second", "list", #"{"list":"reads"}"#, createdAt: 200)
        try await enqueue(database, "favorited", "is_favorite", #"{"is_favorite":true}"#, createdAt: 300)
        // Queued but no longer stored (e.g. archived since): not listed.
        try await enqueue(database, "gone", "list", #"{"list":"reads"}"#, createdAt: 400)

        #expect(await store.listPendingSaves().map(\.id) == ["second", "first"])
    }

    // MARK: body round-tripping

    @Test func contentIsNilWhenNoBodyHasBeenSaved() async throws {
        try await store.upsertMany([makeArticle("a1")])

        #expect(await store.getById("a1")?.content == nil)
    }

    @Test func roundTripsASavedBodyThroughGetById() async throws {
        try await store.upsertMany([makeArticle("a1")])
        try await store.saveBody(id: "a1", body: "<p>Hello</p>")

        #expect(await store.getById("a1")?.content == "<p>Hello</p>")
    }

    @Test func preservesAnExistingBodyWhenALaterListSyncHasNoContent() async throws {
        try await store.upsertMany([makeArticle("a1")])
        try await store.saveBody(id: "a1", body: "<p>Hello</p>")

        // A list-page refresh never carries the body.
        try await store.upsertMany([makeArticle("a1", title: "Refreshed")])

        let stored = await store.getById("a1")
        #expect(stored?.title == "Refreshed")
        #expect(stored?.content == "<p>Hello</p>")
    }

    // MARK: content_hash body invalidation

    @Test func keepsTheBodyWhenARefreshedRowCarriesTheSameContentHash() async throws {
        try await store.upsertMany([makeArticle("a1", contentHash: "h1")])
        try await store.saveBody(id: "a1", body: "<p>Hello</p>")

        try await store.upsertMany([makeArticle("a1", title: "Refreshed", contentHash: "h1")])

        let stored = await store.getById("a1")
        #expect(stored?.title == "Refreshed")
        #expect(stored?.content == "<p>Hello</p>")
        #expect(stored?.contentHash == "h1")
    }

    @Test func clearsTheBodyWhenARefreshedRowCarriesADifferentContentHash() async throws {
        try await store.upsertMany([makeArticle("a1", contentHash: "h1")])
        try await store.saveBody(id: "a1", body: "<p>Hello</p>")

        try await store.upsertMany([makeArticle("a1", contentHash: "h2")])

        let stored = await store.getById("a1")
        #expect(stored?.content == nil)
        #expect(stored?.contentHash == "h2")
    }

    @Test func keepsTheBodyWhenARefreshedRowCarriesNoContentHash() async throws {
        try await store.upsertMany([makeArticle("a1", contentHash: "h1")])
        try await store.saveBody(id: "a1", body: "<p>Hello</p>")

        try await store.upsertMany([makeArticle("a1", contentHash: nil)])

        let stored = await store.getById("a1")
        #expect(stored?.content == "<p>Hello</p>")
        #expect(stored?.contentHash == "h1")
    }

    // MARK: listPrefetchCandidates

    @Test func prefetchCandidatesAreUnreadWithNoCachedBodyNewestFirst() async throws {
        try await store.upsertMany([
            makeArticle("no-body", addedAt: 100),
            makeArticle("newer-no-body", addedAt: 300),
            makeArticle("has-body", addedAt: 200),
            makeArticle("read", isRead: true, addedAt: 400),
        ])
        try await store.saveBody(id: "has-body", body: "<p>already cached</p>")

        #expect(await store.listPrefetchCandidates(limit: 10).map(\.id) == ["newer-no-body", "no-body"])
    }

    @Test func prefetchCandidatesRespectTheLimit() async throws {
        try await store.upsertMany([
            makeArticle("a", addedAt: 100), makeArticle("b", addedAt: 200), makeArticle("c", addedAt: 300),
        ])

        #expect(await store.listPrefetchCandidates(limit: 2).map(\.id) == ["c", "b"])
    }

    // MARK: evictBodiesOutsideCap

    @Test func evictionClearsBodiesOutsideTheCapKeepingTheMostRecent() async throws {
        try await store.upsertMany([makeArticle("old", addedAt: 100), makeArticle("newer", addedAt: 200)])
        try await store.saveBody(id: "old", body: "<p>old</p>")
        try await store.saveBody(id: "newer", body: "<p>newer</p>")

        try await store.evictBodiesOutsideCap(1)

        #expect(await store.getById("old")?.content == nil)
        #expect(await store.getById("newer")?.content == "<p>newer</p>")
    }

    @Test func evictionTouchesOnlyTheBody() async throws {
        try await store.upsertMany([makeArticle("old", title: "Old", addedAt: 100)])
        try await store.saveBody(id: "old", body: "<p>old</p>")

        try await store.evictBodiesOutsideCap(0)

        let old = await store.getById("old")
        #expect(old?.title == "Old")
        #expect(old?.content == nil)
    }

    // MARK: updateUserState

    @Test func updatesUserStateFieldsIndependently() async throws {
        try await store.upsertMany([makeArticle("a1")])

        try await store.updateUserState(id: "a1", isFavorite: true)
        var stored = await store.getById("a1")
        #expect(stored?.isFavorite == true)
        #expect(stored?.isRead == false)

        try await store.updateUserState(id: "a1", isRead: true, readAt: Date(timeIntervalSince1970: 500))
        stored = await store.getById("a1")
        #expect(stored?.isRead == true)
        #expect(stored?.readAt == Date(timeIntervalSince1970: 500))
        #expect(stored?.isFavorite == true)

        try await store.updateUserState(id: "a1", scrollPosition: 0.42)
        stored = await store.getById("a1")
        #expect(stored?.scrollPosition == 0.42)
    }

    @Test func updateUserStateDoesNothingWhenTheIdIsNotFound() async throws {
        try await store.updateUserState(id: "missing", isRead: true)

        #expect(await store.getById("missing") == nil)
    }

    // MARK: remove

    @Test func removeDeletesASingleArticle() async throws {
        try await store.upsertMany([makeArticle("a1"), makeArticle("a2")])

        try await store.remove(id: "a1")

        #expect(await store.listRecent(limit: 10).map(\.id) == ["a2"])
    }

    // MARK: clear

    @Test func clearRemovesAllStoredArticles() async throws {
        try await store.upsertMany([makeArticle("a1"), makeArticle("a2")])

        try await store.clear()

        #expect(await store.listRecent(limit: 10).isEmpty)
    }

    /// A queued write must never replay against a different account.
    @Test func clearAlsoClearsQueuedOutboxRows() async throws {
        try await store.upsertMany([makeArticle("a1", isFavorite: true)])
        try await enqueue(database, "a1", "is_favorite", #"{"is_favorite":true}"#)

        try await store.clear()

        // If the outbox row had survived, the guard would freeze isFavorite on the second upsert.
        try await store.upsertMany([makeArticle("a1", isFavorite: true)])
        try await store.upsertMany([makeArticle("a1", isFavorite: false)])

        #expect(await store.getById("a1")?.isFavorite == false)
    }

    // MARK: outbox guard on upsertMany

    @Test func preservesAPendingOfflineChangeWhenAListSyncArrivesBeforeTheDrain() async throws {
        try await store.upsertMany([makeArticle("a1", title: "Original", isFavorite: false)])
        // The mutation writes the store first, then queues the write after a NetworkError.
        try await store.updateUserState(id: "a1", isFavorite: true)
        try await enqueue(database, "a1", "is_favorite", #"{"is_favorite":true}"#)

        // A list sync arrives before the outbox drains, carrying the server's stale value.
        try await store.upsertMany([makeArticle("a1", title: "Refreshed", isFavorite: false)])

        let stored = await store.getById("a1")
        #expect(stored?.isFavorite == true)
        #expect(stored?.title == "Refreshed")
    }

    @Test func freezesEveryUserStateColumnWhileAnyOutboxRowExists() async throws {
        try await store.upsertMany([makeArticle("a1")])
        try await store.updateUserState(
            id: "a1", isRead: true, isFavorite: true, scrollPosition: 0.5, readAt: Date(timeIntervalSince1970: 500))
        try await enqueue(database, "a1", "status", #"{"status":"completed"}"#)

        try await store.upsertMany([makeArticle("a1")])

        let stored = await store.getById("a1")
        #expect(stored?.isRead == true)
        #expect(stored?.isFavorite == true)
        #expect(stored?.scrollPosition == 0.5)
        #expect(stored?.readAt == Date(timeIntervalSince1970: 500))
    }

    @Test func doesNotResurrectAnArticleWithAPendingArchive() async throws {
        try await store.upsertMany([makeArticle("a1")])
        try await store.remove(id: "a1")
        try await enqueue(database, "a1", "status", #"{"status":"archived"}"#)

        // The next list sync still sees the article server-side.
        try await store.upsertMany([makeArticle("a1")])

        #expect(await store.getById("a1") == nil)
    }

    @Test func resumesNormalSyncingOnceTheOutboxRowIsGone() async throws {
        try await store.upsertMany([makeArticle("a1", isFavorite: true)])
        try await enqueue(database, "a1", "is_favorite", #"{"is_favorite":true}"#)

        // The drain succeeded and removed its row.
        try await database.queue().write { db in
            try db.execute(sql: "DELETE FROM outbox WHERE article_id = 'a1'")
        }

        try await store.upsertMany([makeArticle("a1", isFavorite: false)])

        #expect(await store.getById("a1")?.isFavorite == false)
    }
}

/// Reads degrade to "nothing cached"; writes still throw; a failed open is retried.
struct ArticleStoreFailureTests {
    struct OpenFailure: Error {}

    final class Attempts: Sendable {
        private let count = Mutex(0)

        func next() -> Int {
            count.withLock { count in
                count += 1
                return count
            }
        }
    }

    @Test func readsNeverThrow() async {
        let store = ArticleStore(database: AppDatabase { throw OpenFailure() })

        #expect(await store.listRecent(limit: 10).isEmpty)
        #expect(await store.listFavorites().isEmpty)
        #expect(await store.listPendingSaves().isEmpty)
        #expect(await store.listPrefetchCandidates(limit: 10).isEmpty)
        #expect(await store.getById("a1") == nil)
    }

    @Test func writesThrow() async {
        let store = ArticleStore(database: AppDatabase { throw OpenFailure() })

        await #expect(throws: OpenFailure.self) { try await store.upsertMany([makeArticle("write-fails")]) }
    }

    @Test func retriesAfterAFailedOpen() async throws {
        let attempts = Attempts()
        let store = ArticleStore(
            database: AppDatabase {
                if attempts.next() == 1 { throw OpenFailure() }
                return try DatabaseQueue()
            })

        await #expect(throws: OpenFailure.self) { try await store.upsertMany([makeArticle("retry-1")]) }
        try await store.upsertMany([makeArticle("retry-1")])

        #expect(await store.listRecent(limit: 10).map(\.id) == ["retry-1"])
    }
}
