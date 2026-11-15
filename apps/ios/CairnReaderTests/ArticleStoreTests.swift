import Foundation
import GRDB
import Testing
@testable import CairnReader

// Port of apps/mobile/src/services/articleStore.test.ts against a real in-memory database.
// Not ported: the v1 -> v2 migration cases (there is no RN migration ladder here; the schema is
// created in one migration, epic_bc48 decision 4). Outbox rows are inserted directly until the
// outbox itself is ported (task_387d). The list methods and `clear` were replaced by the
// observations and `UserSession.close()` (task_288e), so their cases moved there.

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
    try await database.writer().write { db in
        try db.execute(
            sql: "INSERT INTO outbox (article_id, field, payload, created_at) VALUES (?, ?, ?, ?)",
            arguments: [articleID, field, payload, createdAt])
    }
}

/// Every stored id, newest first.
private func storedIDs(_ database: AppDatabase) async throws -> [String] {
    try await database.writer().read { db in
        try String.fetchAll(db, sql: "SELECT id FROM articles ORDER BY added_at DESC, id")
    }
}

struct ArticleStoreTests {
    let database = AppDatabase.inMemory()
    var store: ArticleStore { ArticleStore(database: database) }

    // MARK: upsertMany

    @Test func insertsNewArticles() async throws {
        try await store.upsertMany([makeArticle("a1"), makeArticle("a2")])

        #expect(try await storedIDs(database).count == 2)
    }

    @Test func updatesAnExistingRowInsteadOfDuplicatingIt() async throws {
        try await store.upsertMany([makeArticle("a1", title: "Old Title")])
        try await store.upsertMany([makeArticle("a1", title: "New Title")])

        #expect(try await storedIDs(database) == ["a1"])
        #expect(try await store.getById("a1")?.title == "New Title")
    }

    @Test func doesNotDeleteRowsAbsentFromTheIncomingPage() async throws {
        try await store.upsertMany([makeArticle("a1")])
        try await store.upsertMany([makeArticle("a2")])

        #expect(try await storedIDs(database).sorted() == ["a1", "a2"])
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

        #expect(try await store.getById("a1") == article)
    }

    // MARK: body round-tripping

    @Test func contentIsNilWhenNoBodyHasBeenSaved() async throws {
        try await store.upsertMany([makeArticle("a1")])

        #expect(try await store.getById("a1")?.content == nil)
    }

    @Test func roundTripsASavedBodyThroughGetById() async throws {
        try await store.upsertMany([makeArticle("a1")])
        try await store.saveBody(id: "a1", body: "<p>Hello</p>")

        #expect(try await store.getById("a1")?.content == "<p>Hello</p>")
    }

    @Test func preservesAnExistingBodyWhenALaterListSyncHasNoContent() async throws {
        try await store.upsertMany([makeArticle("a1")])
        try await store.saveBody(id: "a1", body: "<p>Hello</p>")

        // A list-page refresh never carries the body.
        try await store.upsertMany([makeArticle("a1", title: "Refreshed")])

        let stored = try await store.getById("a1")
        #expect(stored?.title == "Refreshed")
        #expect(stored?.content == "<p>Hello</p>")
    }

    // MARK: content_hash body invalidation

    @Test func keepsTheBodyWhenARefreshedRowCarriesTheSameContentHash() async throws {
        try await store.upsertMany([makeArticle("a1", contentHash: "h1")])
        try await store.saveBody(id: "a1", body: "<p>Hello</p>")

        try await store.upsertMany([makeArticle("a1", title: "Refreshed", contentHash: "h1")])

        let stored = try await store.getById("a1")
        #expect(stored?.title == "Refreshed")
        #expect(stored?.content == "<p>Hello</p>")
        #expect(stored?.contentHash == "h1")
    }

    @Test func clearsTheBodyWhenARefreshedRowCarriesADifferentContentHash() async throws {
        try await store.upsertMany([makeArticle("a1", contentHash: "h1")])
        try await store.saveBody(id: "a1", body: "<p>Hello</p>")

        try await store.upsertMany([makeArticle("a1", contentHash: "h2")])

        let stored = try await store.getById("a1")
        #expect(stored?.content == nil)
        #expect(stored?.contentHash == "h2")
    }

    @Test func keepsTheBodyWhenARefreshedRowCarriesNoContentHash() async throws {
        try await store.upsertMany([makeArticle("a1", contentHash: "h1")])
        try await store.saveBody(id: "a1", body: "<p>Hello</p>")

        try await store.upsertMany([makeArticle("a1", contentHash: nil)])

        let stored = try await store.getById("a1")
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

        #expect(try await store.listPrefetchCandidates(limit: 10).map(\.id) == ["newer-no-body", "no-body"])
    }

    @Test func prefetchCandidatesRespectTheLimit() async throws {
        try await store.upsertMany([
            makeArticle("a", addedAt: 100), makeArticle("b", addedAt: 200), makeArticle("c", addedAt: 300),
        ])

        #expect(try await store.listPrefetchCandidates(limit: 2).map(\.id) == ["c", "b"])
    }

    // MARK: evictBodiesOutsideCap

    @Test func evictionClearsBodiesOutsideTheCapKeepingTheMostRecent() async throws {
        try await store.upsertMany([makeArticle("old", addedAt: 100), makeArticle("newer", addedAt: 200)])
        try await store.saveBody(id: "old", body: "<p>old</p>")
        try await store.saveBody(id: "newer", body: "<p>newer</p>")

        try await store.evictBodiesOutsideCap(1)

        #expect(try await store.getById("old")?.content == nil)
        #expect(try await store.getById("newer")?.content == "<p>newer</p>")
    }

    @Test func evictionTouchesOnlyTheBody() async throws {
        try await store.upsertMany([makeArticle("old", title: "Old", addedAt: 100)])
        try await store.saveBody(id: "old", body: "<p>old</p>")

        try await store.evictBodiesOutsideCap(0)

        let old = try await store.getById("old")
        #expect(old?.title == "Old")
        #expect(old?.content == nil)
    }

    // MARK: updateUserState

    @Test func updatesUserStateFieldsIndependently() async throws {
        try await store.upsertMany([makeArticle("a1")])

        try await store.updateUserState(id: "a1", isFavorite: true)
        var stored = try await store.getById("a1")
        #expect(stored?.isFavorite == true)
        #expect(stored?.isRead == false)

        try await store.updateUserState(id: "a1", isRead: true, readAt: Date(timeIntervalSince1970: 500))
        stored = try await store.getById("a1")
        #expect(stored?.isRead == true)
        #expect(stored?.readAt == Date(timeIntervalSince1970: 500))
        #expect(stored?.isFavorite == true)

        try await store.updateUserState(id: "a1", scrollPosition: 0.42)
        stored = try await store.getById("a1")
        #expect(stored?.scrollPosition == 0.42)
    }

    @Test func updateUserStateDoesNothingWhenTheIdIsNotFound() async throws {
        try await store.updateUserState(id: "missing", isRead: true)

        #expect(try await store.getById("missing") == nil)
    }

    // MARK: remove

    @Test func removeDeletesASingleArticle() async throws {
        try await store.upsertMany([makeArticle("a1"), makeArticle("a2")])

        try await store.remove(id: "a1")

        #expect(try await storedIDs(database) == ["a2"])
    }

    // MARK: outbox guard on upsertMany

    @Test func preservesAPendingOfflineChangeWhenAListSyncArrivesBeforeTheDrain() async throws {
        try await store.upsertMany([makeArticle("a1", title: "Original", isFavorite: false)])
        // The mutation writes the store first, then queues the write after a NetworkError.
        try await store.updateUserState(id: "a1", isFavorite: true)
        try await enqueue(database, "a1", "is_favorite", #"{"is_favorite":true}"#)

        // A list sync arrives before the outbox drains, carrying the server's stale value.
        try await store.upsertMany([makeArticle("a1", title: "Refreshed", isFavorite: false)])

        let stored = try await store.getById("a1")
        #expect(stored?.isFavorite == true)
        #expect(stored?.title == "Refreshed")
    }

    @Test func freezesEveryUserStateColumnWhileAnyOutboxRowExists() async throws {
        try await store.upsertMany([makeArticle("a1")])
        try await store.updateUserState(
            id: "a1", isRead: true, isFavorite: true, scrollPosition: 0.5, readAt: Date(timeIntervalSince1970: 500))
        try await enqueue(database, "a1", "status", #"{"status":"completed"}"#)

        try await store.upsertMany([makeArticle("a1")])

        let stored = try await store.getById("a1")
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

        #expect(try await store.getById("a1") == nil)
    }

    @Test func resumesNormalSyncingOnceTheOutboxRowIsGone() async throws {
        try await store.upsertMany([makeArticle("a1", isFavorite: true)])
        try await enqueue(database, "a1", "is_favorite", #"{"is_favorite":true}"#)

        // The drain succeeded and removed its row.
        try await database.writer().write { db in
            try db.execute(sql: "DELETE FROM outbox WHERE article_id = 'a1'")
        }

        try await store.upsertMany([makeArticle("a1", isFavorite: false)])

        #expect(try await store.getById("a1")?.isFavorite == false)
    }

    // MARK: syncPage

    /// Stores a1 (300), a2 (200), a3 (100), a4 (50).
    private func storeFour() async throws {
        try await store.upsertMany([
            makeArticle("a1", addedAt: 300), makeArticle("a2", addedAt: 200),
            makeArticle("a3", addedAt: 100), makeArticle("a4", addedAt: 50),
        ])
    }

    @Test func syncPageDeletesARowTheServerDropped() async throws {
        try await storeFour()

        // First page: a1 and a3; a2 was archived elsewhere.
        try await store.syncPage(
            [makeArticle("a1", addedAt: 300), makeArticle("a3", addedAt: 100)], olderThan: nil, hasMore: true)

        #expect(try await storedIDs(database) == ["a1", "a3", "a4"])
    }

    @Test func syncPageKeepsADroppedRowWithAnOutboxRow() async throws {
        try await storeFour()
        try await enqueue(database, "a2", "list", #"{"list":"reads"}"#)

        try await store.syncPage(
            [makeArticle("a1", addedAt: 300), makeArticle("a3", addedAt: 100)], olderThan: nil, hasMore: true)

        #expect(try await storedIDs(database) == ["a1", "a2", "a3", "a4"])
    }

    @Test func syncPageKeepsRowsOutsideTheWindowAndOnItsBoundaries() async throws {
        try await storeFour()
        try await store.upsertMany([makeArticle("upper-tie", addedAt: 300), makeArticle("lower-tie", addedAt: 100)])

        // A middle page: newer than a3 (100), older than the previous page's last (300).
        try await store.syncPage(
            [makeArticle("a2", addedAt: 200), makeArticle("a3", addedAt: 100)], olderThan: Date(timeIntervalSince1970: 300),
            hasMore: true)

        #expect(try await storedIDs(database) == ["a1", "upper-tie", "a2", "a3", "lower-tie", "a4"])
    }

    @Test func syncPageWindowIsUnboundedBelowOnTheLastPage() async throws {
        try await storeFour()

        try await store.syncPage([makeArticle("a2", addedAt: 200)], olderThan: Date(timeIntervalSince1970: 300), hasMore: false)

        #expect(try await storedIDs(database) == ["a1", "a2"])
    }

    @Test func syncPageWithAnEmptyLastPageDeletesEverythingWithoutAnOutboxRow() async throws {
        try await storeFour()
        try await enqueue(database, "a4", "is_favorite", #"{"is_favorite":true}"#)

        try await store.syncPage([], olderThan: nil, hasMore: false)

        #expect(try await storedIDs(database) == ["a4"])
    }

    @Test func syncPageAppliesTheUpsertGuards() async throws {
        try await store.upsertMany([makeArticle("a1", isFavorite: true, addedAt: 300)])
        try await enqueue(database, "a1", "is_favorite", #"{"is_favorite":true}"#)
        try await enqueue(database, "gone", "status", #"{"status":"archived"}"#)

        try await store.syncPage(
            [makeArticle("a1", isFavorite: false, addedAt: 300), makeArticle("gone", addedAt: 200)], olderThan: nil,
            hasMore: false)

        #expect(try await storedIDs(database) == ["a1"])
        #expect(try await store.getById("a1")?.isFavorite == true)
    }

    // MARK: observations

    @Test func observeReadsListsNewestFirstWithinTheLimitAndFlagsPendingSaves() async throws {
        try await storeFour()
        try await enqueue(database, "a2", "list", #"{"list":"reads"}"#)
        try await enqueue(database, "a1", "is_favorite", #"{"is_favorite":true}"#)

        var reads = try store.observeReads(limit: 3).makeAsyncIterator()
        let entries = try await reads.next()

        #expect(entries?.map(\.id) == ["a1", "a2", "a3"])
        #expect(entries?.map(\.pendingSave) == [false, true, false])
    }

    @Test func observeReadsEmitsAfterUpsertRemoveAndAnOutboxChange() async throws {
        var reads = try store.observeReads(limit: 10).makeAsyncIterator()
        #expect(try await reads.next() == [])

        try await store.upsertMany([makeArticle("a1")])
        #expect(try await reads.next()?.map(\.id) == ["a1"])

        try await enqueue(database, "a1", "list", #"{"list":"reads"}"#)
        #expect(try await reads.next()?.map(\.pendingSave) == [true])

        try await store.remove(id: "a1")
        #expect(try await reads.next() == [])
    }

    @Test func observeArticleFollowsOneArticle() async throws {
        var article = try store.observeArticle(id: "a1").makeAsyncIterator()
        #expect(try await article.next() == .some(nil))

        try await store.upsertMany([makeArticle("a1")])
        #expect(try await article.next()??.id == "a1")

        try await store.updateUserState(id: "a1", isFavorite: true)
        #expect(try await article.next()??.isFavorite == true)
    }

    @Test func observeStoredIDsFollowsInsertsAndDeletes() async throws {
        var ids = try store.observeStoredIDs().makeAsyncIterator()
        #expect(try await ids.next() == [])

        try await store.upsertMany([makeArticle("a1"), makeArticle("a2")])
        #expect(try await ids.next() == ["a1", "a2"])

        try await store.remove(id: "a1")
        #expect(try await ids.next() == ["a2"])
    }
}
