import Foundation
import Observation

/// Saves a link to Reads or subscribes to a feed. Ported from the React Native `AddLinkModal`.
///
/// Each URL change detects (debounced) whether the URL is a feed or a page, superseding any
/// detection still running; until the current URL is detected, Add is ignored, so a stale result
/// never sends the wrong type or list. A detected feed offers where its new items go (default
/// Reads); a page always lands in Reads. In `.feed` mode a page can only go through Find feed.
@Observable
final class AddLinkModel {
    /// `.link` saves a page or subscribes to a feed (Reads); `.feed` subscribes only (RSS).
    nonisolated enum Mode: Sendable {
        case link, feed
    }

    /// Detection starts this long after the last edit of the URL.
    static let detectDebounce = Duration.milliseconds(300)

    typealias Detect = @MainActor (_ url: String) async throws -> DetectURLResponse
    typealias Discover = @MainActor (_ url: String) async throws -> [DiscoveredFeed]
    typealias Add = @MainActor (_ request: AddURLRequest) async throws -> AddURLResponse

    let mode: Mode

    /// The URL field's text.
    var url = "" {
        didSet { if url != oldValue { urlChanged() } }
    }

    /// Where a subscribed feed's new items land. Reads is the backend default.
    var list = ContentList.reads
    /// The current URL's detection; nil until it is detected.
    private(set) var detection: DetectURLResponse?
    /// The current URL is waiting for, or in, detection.
    private(set) var isDetecting = false
    private(set) var isDiscovering = false
    private(set) var isAdding = false
    /// Shown under the URL field; cleared by editing it.
    private(set) var error: String?
    /// Find feed found more than one; the user picks one. Set to nil to dismiss.
    var feedChoices: [DiscoveredFeed]?
    /// What was added; set once, when the add succeeds.
    private(set) var successMessage: String?

    @ObservationIgnored private let detect: Detect
    @ObservationIgnored private let discover: Discover
    @ObservationIgnored private let add: Add
    @ObservationIgnored private let debounce: Duration
    @ObservationIgnored private var detectTask: Task<Void, Never>?
    /// The sheet has gone: a late discovery is not applied.
    @ObservationIgnored private var isClosed = false

    init(
        mode: Mode, detect: @escaping Detect, discover: @escaping Discover, add: @escaping Add,
        debounce: Duration = AddLinkModel.detectDebounce
    ) {
        self.mode = mode
        self.detect = detect
        self.discover = discover
        self.add = add
        self.debounce = debounce
    }

    convenience init(read: ReadService, mode: Mode) {
        self.init(
            mode: mode, detect: { try await read.detectURL($0) }, discover: { try await read.discoverFeed($0) },
            add: { try await read.addURL($0) })
    }

    // MARK: - State

    var isFeed: Bool { detection?.type == .feed }

    /// Whether Add is offered: in `.feed` mode only once a feed is detected.
    var showsAdd: Bool { mode == .link || isFeed }

    /// Whether Find feed is offered: for anything not detected as a feed.
    var showsFindFeed: Bool { !isFeed }

    var canAdd: Bool { !isAdding && !isDetecting && !isBlank }
    var canFindFeed: Bool { !isAdding && !isDiscovering && !isBlank }

    /// Only a submit in flight blocks closing; detection and discovery are background work.
    var canDismiss: Bool { !isAdding }

    var addTitle: String {
        if isAdding { return "Adding…" }
        if isDetecting { return "Detecting…" }
        return isFeed ? "Add Feed" : "Add"
    }

    /// What the URL was detected as, when known.
    var summary: String? {
        switch detection?.type {
        case .feed: detection?.title.map { "RSS feed: \($0)" } ?? "RSS feed"
        case .page: detection?.title.map { "Article: \($0)" } ?? "Article"
        case .unknown, nil: nil
        }
    }

    // MARK: - Actions

    /// The return key: Add where it is offered, else Find feed.
    func submit() async {
        if showsAdd {
            await addURL()
        } else {
            await findFeed()
        }
    }

    /// Saves the page or subscribes to the feed. Ignored while the URL is being detected.
    func addURL() async {
        guard showsAdd, !isDetecting, !isAdding else { return }
        guard let normalized = Self.normalized(url) else {
            error = "Please enter a valid URL"
            return
        }
        error = nil
        isAdding = true
        defer { isAdding = false }
        let request = AddURLRequest(
            url: normalized, type: detection?.type, title: detection?.title,
            // Only feeds have a destination to choose; pages always land in Reads.
            list: isFeed ? list : nil)
        do {
            successMessage = Self.successMessage(for: try await add(request))
        } catch {
            self.error = Self.message(for: error)
        }
    }

    /// Looks for feeds advertised by the page: one replaces the URL (which is then detected as a
    /// feed), several are offered in `feedChoices`.
    func findFeed() async {
        guard !isDiscovering, !isAdding else { return }
        guard let normalized = Self.normalized(url) else {
            error = "Please enter a valid URL"
            return
        }
        error = nil
        isDiscovering = true
        // A failure answers no feeds (`ReadService.discoverFeed`).
        let feeds = (try? await discover(normalized)) ?? []
        guard !isClosed else { return }
        isDiscovering = false
        // An edit during the discovery makes its result stale.
        guard Self.normalized(url) == normalized else { return }
        switch feeds.count {
        case 0: error = "No RSS feed found for this site"
        case 1: url = feeds[0].url
        default: feedChoices = feeds
        }
    }

    /// The sheet has gone: abandons detection and any discovery still running.
    func close() {
        isClosed = true
        detectTask?.cancel()
        isDetecting = false
        isDiscovering = false
    }

    // MARK: - Private

    private var isBlank: Bool { url.trimmingCharacters(in: .whitespaces).isEmpty }

    private func urlChanged() {
        error = nil
        detectTask?.cancel()
        // The previous URL's result must not describe this one.
        detection = nil
        guard let normalized = Self.normalized(url) else {
            isDetecting = false
            return
        }
        isDetecting = true
        detectTask = Task { [debounce, detect] in
            try? await Task.sleep(for: debounce)
            guard !Task.isCancelled else { return }
            // A failure answers `.unknown`: the user can still add.
            let result = (try? await detect(normalized)) ?? DetectURLResponse(url: normalized, type: .unknown, title: nil)
            guard !Task.isCancelled else { return }
            detection = result
            isDetecting = false
        }
    }

    /// The trimmed URL with `https://` added when it has no scheme; nil when it is not a URL.
    static func normalized(_ text: String) -> String? {
        let trimmed = text.trimmingCharacters(in: .whitespaces)
        guard !trimmed.isEmpty else { return nil }
        let candidate = trimmed.hasPrefix("http://") || trimmed.hasPrefix("https://") ? trimmed : "https://\(trimmed)"
        guard let url = URL(string: candidate, encodingInvalidCharacters: false), url.host() != nil else { return nil }
        return candidate
    }

    private static func successMessage(for response: AddURLResponse) -> String {
        switch response {
        case let .feed(_, subscription):
            "Subscribed to \(subscription.title). New items will appear in \(subscription.list == .feed ? "Feed" : "Reads")."
        case .page:
            "Article added to reading list"
        }
    }

    /// The React Native wording for a failed add.
    private static func message(for error: any Error) -> String {
        let message = error.localizedDescription
        if message.contains("already subscribed") { return "Already subscribed to this feed" }
        if message.contains("Failed to subscribe") { return "Failed to subscribe to feed" }
        if message.contains("Failed to add") { return "Failed to add article" }
        return message
    }
}
