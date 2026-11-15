import Foundation
import GRDB
import Synchronization
import Testing
@testable import CairnReader

// Re-expression of apps/mobile/src/screens/ReadArticleDetailScreen.test.tsx and
// ReadArticleDetailScreen.mutations.test.tsx against `ReaderModel` with an in-memory store. The
// optimistic-rollback and backend-failure alert cases do not port (decision_adb1: mutations never
// see the backend's answer; a refusal is the root's alert and the store follows the re-sync).

/// The reader's server: details (optionally held until released), PATCHes and the Reads first
/// page, all unreachable while offline.
private final class ReaderServer: Sendable {
    let online = Mutex(true)
    /// While true, a detail request waits.
    let holdDetail = Mutex(false)
    let patch = Mutex(SyncFixtures.ok)
    let page = Mutex(SyncFixtures.readsPage([]))

    func stub() -> StubServer {
        StubServer { [self] request in
            while holdDetail.withLock({ $0 }), request.httpMethod == "GET" {
                Thread.sleep(forTimeInterval: 0.005)
            }
            guard online.withLock({ $0 }) else { return .failure(.notConnectedToInternet) }
            switch (request.httpMethod, request.url?.lastPathComponent) {
            case ("PATCH"?, _):
                return patch.withLock { $0 }
            case ("GET"?, "user-1"?):
                return .json(200, page.withLock { $0 })
            case ("GET"?, let id?):
                return .json(200, SyncFixtures.detail(id))
            default:
                return .failure(.badServerResponse)
            }
        }
    }
}

private func article(
    _ id: String = "a1", list: ContentList = .reads, isRead: Bool = false, content: String? = nil,
    contentHash: String? = nil, scrollPosition: Double? = nil
) -> Article {
    var article = storedArticle(id)
    article.list = list
    article.isRead = isRead
    article.content = content
    article.contentHash = contentHash
    article.scrollPosition = scrollPosition
    return article
}

@MainActor
private func scroll(_ offset: Double, isRestoring: Bool = false) -> ReaderModel.Scroll {
    ReaderModel.Scroll(offset: offset, viewport: 100, contentHeight: 1000, isRestoring: isRestoring)
}

/// Runs `operation` as the view's `.task` does. Cancel the returned task at the end of the test.
@MainActor
private func run(_ operation: @escaping @MainActor () async -> Void) -> Task<Void, Never> {
    Task { await operation() }
}

@MainActor
struct ReaderModelTests {
    fileprivate let server = ReaderServer()
    let harness: SyncHarness

    init() {
        harness = SyncHarness(server.stub())
    }

    private func makeModel(_ article: Article, saveInterval: Duration = ReaderModel.saveInterval) -> ReaderModel {
        ReaderModel(article: article, session: harness.session, saveInterval: saveInterval)
    }

    private func store(_ articles: Article...) async throws {
        try await harness.session.articles.upsertMany(articles)
    }

    /// The queued outbox payloads by field.
    private func outbox() throws -> [String: String] {
        try harness.session.database.writer().read { db in
            var payloads: [String: String] = [:]
            for row in try Row.fetchAll(db, sql: "SELECT field, payload FROM outbox") {
                let field: String = row["field"]
                let payload: String = row["payload"]
                payloads[field] = payload
            }
            return payloads
        }
    }

    private var detailRequests: Int {
        harness.requests.count(where: { $0 == "GET a1" })
    }

    // MARK: - Body

    @Test func aStoredBodyShowsWithoutWaitingForTheFetch() async throws {
        try await store(article(content: "<p>Cached</p>"))
        server.holdDetail.withLock { $0 = true }
        defer { server.holdDetail.withLock { $0 = false } }
        let model = makeModel(article())
        let following = run { await model.follow() }
        defer { following.cancel() }
        let loading = run { await model.loadBody(isOnline: true) }
        defer { loading.cancel() }

        await eventually { model.article.content == "<p>Cached</p>" && detailRequests == 1 }
        #expect(model.display == .article)
    }

    @Test func aStoredBodyWithTheRoutesHashIsNotFetched() async throws {
        try await store(article(content: "<p>Current</p>", contentHash: "hash-1"))
        let model = makeModel(article(contentHash: "hash-1"))
        let following = run { await model.follow() }
        defer { following.cancel() }

        await model.loadBody(isOnline: true)

        await eventually { model.article.content == "<p>Current</p>" }
        #expect(harness.requests.isEmpty)
    }

    @Test func aStoredBodyWithAnotherHashIsShownAndRefetched() async throws {
        try await store(article(content: "<p>Stale</p>", contentHash: "hash-1"))
        server.holdDetail.withLock { $0 = true }
        defer { server.holdDetail.withLock { $0 = false } }
        let model = makeModel(article(contentHash: "hash-2"))
        let following = run { await model.follow() }
        defer { following.cancel() }
        let loading = run { await model.loadBody(isOnline: true) }

        await eventually { model.article.content == "<p>Stale</p>" && detailRequests == 1 }

        server.holdDetail.withLock { $0 = false }
        await loading.value
        #expect(model.article.content == SyncFixtures.body("a1"))
        #expect(try await harness.session.articles.getById("a1")?.content == SyncFixtures.body("a1"))
    }

    @Test func offlineWithNoStoredBodyIsUnavailable() async {
        let model = makeModel(article(list: .feed))

        await model.loadBody(isOnline: false)

        #expect(model.display == .unavailableOffline)
        #expect(harness.requests.isEmpty)
    }

    @Test func offlineWithAStoredBodyShowsIt() async throws {
        try await store(article(content: "<p>Cached</p>"))
        let model = makeModel(article())
        let following = run { await model.follow() }
        defer { following.cancel() }

        await model.loadBody(isOnline: false)

        await eventually { model.article.content == "<p>Cached</p>" }
        #expect(model.display == .article)
        #expect(harness.requests.isEmpty)
    }

    /// "shows the spinner, never a blank body, while re-fetching after reconnect" and "renders
    /// content once connectivity returns and the fetch resolves".
    @Test func reconnectingRetriesWithTheSpinner() async {
        let model = makeModel(article(list: .feed))
        await model.loadBody(isOnline: false)
        #expect(model.display == .unavailableOffline)

        server.holdDetail.withLock { $0 = true }
        defer { server.holdDetail.withLock { $0 = false } }
        let loading = run { await model.loadBody(isOnline: true) }
        await eventually { detailRequests == 1 }
        #expect(model.display == .loading)

        server.holdDetail.withLock { $0 = false }
        await loading.value
        #expect(model.display == .article)
        #expect(model.article.content == SyncFixtures.body("a1"))
    }

    // MARK: - Following the store

    @Test func aStoredArticleShowsAFavouriteChangedElsewhere() async throws {
        try await store(article())
        let model = makeModel(article())
        let following = run { await model.follow() }
        defer { following.cancel() }

        try await harness.session.articles.updateUserState(id: "a1", isFavorite: true)

        await eventually { model.article.isFavorite }
    }

    @Test func aRejectedFavouriteShowsTheServersValueAfterTheResync() async throws {
        server.patch.withLock { $0 = .json(422, AuthFixtures.errorBody("nope")) }
        server.page.withLock { $0 = SyncFixtures.readsPage(["a1"]) }
        try await store(article())
        let model = makeModel(article())
        let following = run { await model.follow() }
        defer { following.cancel() }

        await model.toggleFavorite()

        await eventually { harness.requests.contains("GET user-1") }
        await eventually { !model.article.isFavorite }
        #expect(harness.requests.contains("PATCH a1"))
        #expect(try outbox().isEmpty)
    }

    @Test func anUnstoredArticleKeepsItsOwnEdits() async throws {
        server.online.withLock { $0 = false }
        let model = makeModel(article(list: .feed))
        let following = run { await model.follow() }
        defer { following.cancel() }

        await model.toggleFavorite()

        #expect(model.article.isFavorite)
        #expect(try outbox()["is_favorite"] == #"{"is_favorite":true}"#)
    }

    // MARK: - Toolbar

    @Test func archiveQueuesTheArchiveAndPops() async throws {
        server.online.withLock { $0 = false }
        try await store(article())
        let model = makeModel(article())

        await model.archive()

        #expect(model.isFinished)
        #expect(try await harness.session.articles.getById("a1") == nil)
        #expect(try outbox()["status"] == #"{"status":"archived"}"#)
    }

    @Test func saveToReadsQueuesTheMoveAndPops() async throws {
        server.online.withLock { $0 = false }
        let model = makeModel(article(list: .feed))

        await model.saveToReads()

        #expect(model.isFinished)
        #expect(try await harness.session.articles.getById("a1") != nil)
        #expect(try outbox() == ["list": #"{"list":"reads"}"#])
    }

    @Test func aFailedSaveToReadsKeepsTheReaderOpenAndCanBeRetried() async throws {
        server.online.withLock { $0 = false }
        let model = makeModel(article(list: .feed))
        try await harness.session.database.writer().write { db in
            try db.execute(sql: "CREATE TRIGGER fail BEFORE INSERT ON outbox BEGIN SELECT RAISE(ABORT, 'full'); END")
        }

        await model.saveToReads()

        #expect(!model.isFinished)
        #expect(model.alert?.title == "Error")
        #expect(model.alert?.message == "Failed to save to Reads")

        try await harness.session.database.writer().write { db in try db.execute(sql: "DROP TRIGGER fail") }
        await model.saveToReads()

        #expect(model.isFinished)
        #expect(try outbox() == ["list": #"{"list":"reads"}"#])
    }

    // MARK: - Progress

    @Test func progressIsNotTrackedForFeedItems() async throws {
        server.online.withLock { $0 = false }
        let model = makeModel(article(list: .feed))

        await model.markReadingIfNeeded()
        model.scrolled(scroll(900))
        await model.leave()

        #expect(!model.tracksProgress)
        try await Task.sleep(for: .milliseconds(50))
        #expect(try outbox().isEmpty)
    }

    @Test func openingAnUnreadReadsItemMarksItReadingOnce() async throws {
        server.online.withLock { $0 = false }
        try await store(article())
        let model = makeModel(article())

        await model.markReadingIfNeeded()
        #expect(try outbox()["status"] == #"{"status":"reading"}"#)

        model.scrolled(scroll(900))
        await eventually { (try? outbox()["status"]) == #"{"status":"completed"}"# }
        // Appearing again (another tab and back) does not undo the completion.
        await model.markReadingIfNeeded()
        #expect(try outbox()["status"] == #"{"status":"completed"}"#)
    }

    @Test func aReadItemIsNotMarkedReading() async throws {
        server.online.withLock { $0 = false }
        let model = makeModel(article(isRead: true))

        await model.markReadingIfNeeded()

        #expect(try outbox().isEmpty)
    }

    @Test func completionFiresOnceAt95Percent() async throws {
        server.online.withLock { $0 = false }
        try await store(article())
        let model = makeModel(article())

        model.scrolled(scroll(840))  // (840 + 100) / 1000 = 0.94
        try await Task.sleep(for: .milliseconds(50))
        #expect(try await harness.session.articles.getById("a1")?.isRead == false)

        model.scrolled(scroll(850))
        await eventually { (try? outbox()["status"]) == #"{"status":"completed"}"# }
        let readAt = try await harness.session.articles.getById("a1")?.readAt
        #expect(readAt != nil)

        try await Task.sleep(for: .milliseconds(20))
        model.scrolled(scroll(900))
        try await Task.sleep(for: .milliseconds(50))
        #expect(try await harness.session.articles.getById("a1")?.readAt == readAt)
    }

    /// The rule "do not persist the pre-restore offset": readings taken while restoring neither
    /// save nor complete, and leaving without a confirmed position saves nothing.
    @Test func thePreRestoreOffsetIsNeverSaved() async throws {
        server.online.withLock { $0 = false }
        try await store(article(scrollPosition: 0.5))
        let model = makeModel(article(scrollPosition: 0.5))

        model.scrolled(scroll(0, isRestoring: true))
        model.scrolled(scroll(900, isRestoring: true))
        await model.leave()

        try await Task.sleep(for: .milliseconds(50))
        #expect(try outbox().isEmpty)
        #expect(try await harness.session.articles.getById("a1")?.scrollPosition == 0.5)
        #expect(model.restoreFraction == 0.5)
    }

    @Test func scrollSavesAreThrottledWithATrailingSave() async throws {
        server.online.withLock { $0 = false }
        try await store(article())
        let model = makeModel(article(), saveInterval: .milliseconds(300))

        model.scrolled(scroll(100))
        await eventually { (try? outbox()["scroll_position"]) == #"{"scroll_position":0.1}"# }

        model.scrolled(scroll(200))
        model.scrolled(scroll(300))
        try await Task.sleep(for: .milliseconds(50))
        #expect(try outbox()["scroll_position"] == #"{"scroll_position":0.1}"#)

        await eventually { (try? outbox()["scroll_position"]) == #"{"scroll_position":0.3}"# }
        #expect(try await harness.session.articles.getById("a1")?.scrollPosition == 0.3)
    }

    @Test func leavingSavesTheLastPosition() async throws {
        server.online.withLock { $0 = false }
        try await store(article())
        let model = makeModel(article(), saveInterval: .seconds(60))

        model.scrolled(scroll(100))
        model.scrolled(scroll(400))
        await model.leave()

        #expect(try await harness.session.articles.getById("a1")?.scrollPosition == 0.4)
    }
}

/// The restore-reachability rule, ported from ArticleContent.test.tsx.
struct ScrollTargetReachabilityTests {
    @Test func unreachableWhileTheContentIsTooShort() {
        // Target 640, largest offset 800 - 600 = 200.
        #expect(!ArticleWebView.isScrollTargetReachable(contentHeight: 800, viewport: 600, fraction: 0.8))
    }

    @Test func reachableOnceTheContentHasGrown() {
        // Target 1300, largest offset 2600 - 600 = 2000.
        #expect(ArticleWebView.isScrollTargetReachable(contentHeight: 2600, viewport: 600, fraction: 0.5))
    }

    @Test func unreachableBeforeTheViewportIsMeasured() {
        #expect(!ArticleWebView.isScrollTargetReachable(contentHeight: 5000, viewport: 0, fraction: 0.5))
    }

    @Test func theExactLargestOffsetIsReachable() {
        #expect(ArticleWebView.isScrollTargetReachable(contentHeight: 1200, viewport: 600, fraction: 0.5))
        #expect(!ArticleWebView.isScrollTargetReachable(contentHeight: 1200, viewport: 600, fraction: 0.51))
    }

    @Test func zeroIsReachable() {
        #expect(ArticleWebView.isScrollTargetReachable(contentHeight: 2000, viewport: 600, fraction: 0))
    }
}

/// The page's header and body; the byline cases ported from ArticleContent.test.tsx.
@MainActor
struct ReaderTemplateTests {
    let now = Date(timeIntervalSince1970: 1_700_000_000)

    @Test func theBylineShowsThePublishedDateNotTheDateAdded() {
        var article = storedArticle("a1")
        article.addedAt = now
        article.publishedDate = now.addingTimeInterval(-10 * 86_400)
        let byline = ReaderTemplate.byline(for: article, now: now)
        #expect(byline == "example.com | \(article.publishedDate!.relativeDayLabel(now: now))")
        #expect(!byline.contains("Today"))
    }

    @Test func theBylineOmitsAMissingDate() {
        var article = storedArticle("a1")
        article.author = "Ada"
        article.readingTime = 5
        #expect(ReaderTemplate.byline(for: article, now: now) == "Ada | 5 min read")
    }

    @Test func anEmailWithoutAnAuthorIsFromEmail() {
        var article = storedArticle("a1")
        article.url = "email://abc"
        #expect(ReaderTemplate.byline(for: article, now: now) == "Email")
    }

    @Test func thePageHoldsTheEscapedHeaderAndTheBody() {
        var article = storedArticle("a1")
        article.title = "<b>Fish</b> & {{body}}"
        article.content = "<p>Body</p>"
        let html = ReaderTemplate.html(for: article, baseSize: 17, now: now)
        #expect(html.contains("&lt;b&gt;Fish&lt;/b&gt; &amp; &#123;&#123;body}}"))
        #expect(html.contains("<article>\n<p>Body</p>\n</article>"))
        #expect(html.contains("--base-size: 17.0px"))
        #expect(html.contains("<a href=\"https://example.com/a1\">"))
        #expect(!html.contains("{{"))
    }

    @Test func withoutABodyThePageShowsTheDescription() {
        var article = storedArticle("a1")
        article.description = "Summary"
        #expect(ReaderTemplate.html(for: article, baseSize: 16, now: now).contains("<p>Summary</p>"))
    }

    @Test func theStylesheetAndFontsAreBundled() {
        for name in ["reader.css", "AtkinsonHyperlegibleNext_400Regular.ttf", "Inter_700Bold.ttf"] {
            #expect(Bundle.main.url(forResource: name, withExtension: nil) != nil, "\(name)")
        }
    }
}
