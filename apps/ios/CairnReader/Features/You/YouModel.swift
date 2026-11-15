import Foundation
import Observation
import OSLog

/// The counts on the You tab. Subscriptions and bookmarks are fetched concurrently and fail
/// independently, so a broken endpoint does not hide the other counts.
@Observable
final class YouModel {
    /// Minimum time between fetches triggered by the tab appearing.
    static let refetchInterval: TimeInterval = 30

    /// Nil until first loaded; a failed fetch keeps the last value.
    private(set) var rssCount: Int?
    private(set) var newslettersCount: Int?
    private(set) var bookmarksCount: Int?
    /// The first failure of the last fetch (subscriptions before bookmarks).
    private(set) var errorMessage: String?

    @ObservationIgnored private let read: ReadService
    @ObservationIgnored private let now: () -> Date
    @ObservationIgnored private var lastFetchedAt: Date?
    @ObservationIgnored private var isLoading = false
    @ObservationIgnored private let logger = Logger(subsystem: "com.cairnapp.cairnreader", category: "you")

    init(read: ReadService, now: @escaping () -> Date = Date.init) {
        self.read = read
        self.now = now
    }

    /// Fetches the counts unless the last fully successful fetch is under `refetchInterval` old.
    func refreshIfStale() async {
        guard !isLoading else { return }
        if let lastFetchedAt, now().timeIntervalSince(lastFetchedAt) <= Self.refetchInterval { return }
        isLoading = true
        defer { isLoading = false }

        let read = read
        async let subscriptions = read.listAllSubscriptions()
        async let bookmarks = read.countUserContents(isFavorite: true)
        var failure: String?

        do {
            let all = try await subscriptions.subscriptions
            rssCount = all.count(where: { $0.type != .email })
            newslettersCount = all.count(where: { $0.type == .email })
        } catch {
            logger.error("Fetching subscriptions failed: \(error.localizedDescription)")
            failure = "Failed to load subscription data"
        }
        do {
            bookmarksCount = try await bookmarks
        } catch {
            logger.error("Fetching bookmarks failed: \(error.localizedDescription)")
            failure = failure ?? "Failed to load bookmark data"
        }

        errorMessage = failure
        // Only a fully successful fetch starts the clock, so a failure is retried on next appear.
        if failure == nil {
            lastFetchedAt = now()
        }
    }
}
