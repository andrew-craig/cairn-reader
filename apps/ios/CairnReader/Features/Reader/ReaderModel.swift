import Foundation
import GRDB
import Observation
import OSLog

/// The reader of one article. It starts from the route's `Article`; while the store has a row
/// for the id (every Reads item) it shows that row, followed through `observeArticle`, so
/// favourite, read state and a value re-synced after a rejection are never held twice. An article
/// the store does not hold (Feed, and Bookmarks or search results outside the store) keeps the
/// route's value with the reader's own edits applied. Every edit goes through `ArticleMutations`
/// (decision 9) without waiting for the backend. Ported from the React Native
/// `ReadArticleDetailScreen`.
@Observable
final class ReaderModel {
    /// Fraction of the article read, `(offset + viewport) / contentHeight`, at which it is completed.
    static let completedThreshold = 0.95
    /// Minimum time between scroll position saves while scrolling.
    static let saveInterval = Duration.seconds(1)

    /// What the reader shows.
    nonisolated enum Display: Equatable, Sendable {
        case loading
        case article
        /// Offline with no body stored.
        case unavailableOffline
    }

    /// One reading of the web view's scroll geometry, in points from the top of the content.
    nonisolated struct Scroll: Equatable, Sendable {
        var offset: Double
        var viewport: Double
        var contentHeight: Double
        /// The saved position has not been restored yet and the user has not scrolled: `offset`
        /// is not where the reader is, and must not be saved.
        var isRestoring: Bool
    }

    /// The route's value with the reader's own edits, for an article the store does not hold.
    private var route: Article
    /// The store's row, once the observation found one.
    private var stored: Article?
    /// A body fetched by this reader.
    private var fetched: (content: String, contentHash: String?)?
    private var isLoadingBody: Bool
    private var isOnline = true
    /// Archived or saved to Reads: the screen pops.
    private(set) var isFinished = false
    @ObservationIgnored private var isSavingToReads = false
    /// The alert to show (a link save's outcome or a store failure); set to nil to dismiss.
    var alert: (title: String, message: String)?
    /// Bumped by each saved link, for the success haptic.
    private(set) var savedLinks = 0

    /// Progress is a Reads concern: Feed items are skimmed and triaged by Save to Reads, and
    /// writing progress for them would also queue outbox rows.
    let tracksProgress: Bool
    /// The fraction to restore: the saved one, then the last one confirmed while reading.
    @ObservationIgnored private(set) var restoreFraction: Double
    @ObservationIgnored private var hasScrolled = false
    @ObservationIgnored private var hasMarkedCompleted: Bool
    @ObservationIgnored private var hasMarkedReading = false
    /// The last fraction saved (at first the article's own): an unchanged one is not saved again.
    @ObservationIgnored private var savedFraction: Double
    @ObservationIgnored private var bodyIsCurrent: Bool
    @ObservationIgnored private var loadGeneration = 0
    @ObservationIgnored private var lastSave: ContinuousClock.Instant?
    @ObservationIgnored private var trailingSave: Task<Void, Never>?
    @ObservationIgnored private var leadingSave: Task<Void, Never>?
    @ObservationIgnored private let session: UserSession
    @ObservationIgnored private let saveInterval: Duration
    @ObservationIgnored private let logger = Logger(subsystem: "com.cairnapp.cairnreader", category: "reader")

    init(article: Article, session: UserSession, saveInterval: Duration = ReaderModel.saveInterval) {
        route = article
        self.session = session
        self.saveInterval = saveInterval
        tracksProgress = article.list == .reads
        restoreFraction = article.scrollPosition.flatMap { $0 <= 1 ? $0 : nil } ?? 0
        savedFraction = restoreFraction
        hasMarkedCompleted = article.isRead
        // A route value with a body comes from the store, which drops a body whose hash changed.
        bodyIsCurrent = article.content != nil
        isLoadingBody = article.content == nil
    }

    // MARK: - State

    /// The article as shown: the stored row if there is one, else the route's value; the body
    /// this reader fetched wins over both.
    var article: Article {
        var article = stored ?? route
        if let fetched {
            article.content = fetched.content
            article.contentHash = fetched.contentHash
        }
        return article
    }

    var display: Display {
        if article.content != nil { return .article }
        if isLoadingBody { return .loading }
        // Online with a failed fetch: the template shows the description.
        return isOnline ? .article : .unavailableOffline
    }

    // MARK: - Lifecycle

    /// Follows the store's row until cancelled; the view runs it in `.task`. A row that goes
    /// away (archived) leaves the last value shown.
    func follow() async {
        do {
            for try await article in try session.articles.observeArticle(id: route.id) {
                if let article { stored = article }
            }
        } catch is CancellationError {
        } catch {
            logger.error("Observing the article failed: \(String(describing: error))")
        }
    }

    /// Loads the body unless the one shown is current: a stored body whose hash matches the
    /// route's is, otherwise it is fetched (never offline) and saved to the store. The view runs
    /// it in `.task(id: isOnline)`, so it retries when connectivity returns.
    func loadBody(isOnline: Bool) async {
        self.isOnline = isOnline
        guard !bodyIsCurrent else { return }
        loadGeneration += 1
        let generation = loadGeneration
        isLoadingBody = true
        defer {
            if generation == loadGeneration { isLoadingBody = false }
        }
        let id = route.id
        do {
            let snapshot = try await session.articles.getById(id)
            if snapshot?.content != nil, let hash = snapshot?.contentHash, hash == route.contentHash {
                bodyIsCurrent = true
                return
            }
        } catch is CancellationError {
            return
        } catch {
            logger.error("Reading the stored article failed: \(String(describing: error))")
        }
        guard isOnline else { return }
        do {
            let detail = Article(try await session.read.getContentById(id))
            guard let content = detail.content else { return }
            fetched = (content, detail.contentHash)
            bodyIsCurrent = true
            try await session.articles.saveBody(id: id, body: content, contentHash: detail.contentHash)
        } catch is CancellationError {
            return
        } catch {
            logger.error("Loading the article body failed: \(String(describing: error))")
        }
    }

    /// Marks a Reads article that is not read as being read, once; what opening the reader does.
    func markReadingIfNeeded() async {
        guard tracksProgress, !hasMarkedCompleted, !hasMarkedReading else { return }
        hasMarkedReading = true
        do {
            try await session.mutations.markReading(route.id)
        } catch {
            logger.error("Marking the article reading failed: \(String(describing: error))")
        }
    }

    /// Takes a reading of the scroll geometry. A position confirmed by a scroll (not while the
    /// saved one is still being restored) is saved, at most once per `saveInterval`, and marks
    /// the article completed once `completedThreshold` of it has been read.
    func scrolled(_ scroll: Scroll) {
        guard tracksProgress, !scroll.isRestoring, scroll.contentHeight > 0 else { return }
        let fraction = min(max(scroll.offset / scroll.contentHeight, 0), 1)
        restoreFraction = fraction
        hasScrolled = true
        if !hasMarkedCompleted, (scroll.offset + scroll.viewport) / scroll.contentHeight >= Self.completedThreshold {
            hasMarkedCompleted = true
            let id = route.id
            Task { await mutate("Marking the article completed") { try await $0.markCompleted(id, readAt: Date()) } }
        }
        throttledSave()
    }

    /// Saves the last confirmed position once more on the way out, unless the article left Reads
    /// from here; nothing if it was never scrolled, so the pre-restore offset is never saved.
    /// (Every save is skipped when the fraction is the one last saved.)
    func leave() async {
        // A leading save still queued would land after this one with an older reading: cancel it
        // if it has not started, and wait for it if it has, so this save is the last write.
        let leading = leadingSave
        leadingSave = nil
        leading?.cancel()
        await leading?.value
        trailingSave?.cancel()
        trailingSave = nil
        guard hasScrolled, !isFinished else { return }
        await saveScrollPosition(restoreFraction)
    }

    // MARK: - Actions

    /// Toggles the favourite. A stored article shows the store's value; any other keeps the edit.
    func toggleFavorite() async {
        let isFavorite = !article.isFavorite
        if stored == nil { route.isFavorite = isFavorite }
        let id = route.id
        await mutate("Setting the favourite") { try await $0.setFavorite(id, isFavorite: isFavorite) }
    }

    /// Archives a Reads item and pops at once; the lists follow the store (decision 8).
    func archive() async {
        isFinished = true
        let id = route.id
        await mutate("Archiving") { try await $0.archive(id) }
    }

    /// Saves a Feed item to Reads, then pops. A store failure keeps the reader open; a second tap
    /// while the write runs is ignored.
    func saveToReads() async {
        guard !isFinished, !isSavingToReads else { return }
        isSavingToReads = true
        defer { isSavingToReads = false }
        do {
            try await session.mutations.saveToReads(article)
            isFinished = true
        } catch {
            logger.error("Saving to Reads failed: \(String(describing: error))")
            alert = ("Error", "Failed to save to Reads")
        }
    }

    /// Saves a link tapped in the article to Reads.
    func saveLink(_ url: URL) async {
        do {
            _ = try await session.read.addURL(AddURLRequest(url: url.absoluteString))
            savedLinks += 1
            alert = ("Saved", "Link saved to your reading list.")
        } catch {
            alert = ("Error", "Failed to save link. Please try again.")
        }
    }

    // MARK: - Private

    /// Leading and trailing: saves at once when idle, else once at the end of the interval with
    /// the latest position.
    private func throttledSave() {
        let now = ContinuousClock.now
        if let lastSave, now - lastSave < saveInterval {
            guard trailingSave == nil else { return }
            let delay = saveInterval - (now - lastSave)
            trailingSave = Task { [weak self] in
                try? await Task.sleep(for: delay)
                guard !Task.isCancelled, let self else { return }
                trailingSave = nil
                self.lastSave = .now
                await saveScrollPosition(restoreFraction)
            }
            return
        }
        lastSave = now
        // The leading save writes the reading that triggered it; later readings in the window
        // are the trailing save's.
        let fraction = restoreFraction
        leadingSave = Task { [weak self] in
            guard !Task.isCancelled, let self else { return }
            await saveScrollPosition(fraction)
        }
    }

    private func saveScrollPosition(_ fraction: Double) async {
        let id = route.id
        guard fraction != savedFraction else { return }
        savedFraction = fraction
        await mutate("Saving the scroll position") { try await $0.saveScrollPosition(id, fraction: fraction) }
    }

    private func mutate(_ what: String, _ mutation: (ArticleMutations) async throws -> Void) async {
        do {
            try await mutation(session.mutations)
        } catch {
            logger.error("\(what) failed: \(String(describing: error))")
        }
    }
}
