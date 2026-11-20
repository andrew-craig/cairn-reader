import Foundation
import Observation
import OSLog

/// One kind of source (RSS feeds or newsletter senders) with the list each delivers to, the RN
/// `SubscriptionListScreen`. Routing changes and unsubscribes are optimistic: the row changes at
/// once and is put back, with an error, if the server refuses.
@Observable
final class SubscriptionListModel {
    nonisolated enum Kind: Sendable {
        case rss, newsletters
    }

    let kind: Kind
    private(set) var subscriptions: [UnifiedSubscription] = []
    /// True until the first load finishes.
    private(set) var isLoading = true
    /// Shown in an "Error" alert; the view clears it.
    var errorMessage: String?
    /// The newsletter ingest address (newsletters only), or why it could not be loaded.
    private(set) var emailAddress: String?
    private(set) var emailAddressError: String?

    @ObservationIgnored private let read: ReadService
    @ObservationIgnored private let logger = Logger(subsystem: "com.cairnapp.cairnreader", category: "subscriptions")

    init(read: ReadService, kind: Kind) {
        self.read = read
        self.kind = kind
    }

    /// Whether any row offers the Feed/Reads choice (the note about new items goes with it).
    var isRoutable: Bool { subscriptions.contains { Self.route(for: $0) != nil } }

    /// The `{type}`/`{key}` the list endpoint routes by, or nil when the source cannot be routed.
    static func route(for subscription: UnifiedSubscription) -> (type: SubscriptionType, key: String)? {
        switch subscription.type {
        case .rss: subscription.rssData.map { (.rss, $0.feedId) }
        case .email: (.email, subscription.id)
        }
    }

    /// Only RSS feeds can be unsubscribed from.
    static func canUnsubscribe(_ subscription: UnifiedSubscription) -> Bool {
        subscription.type == .rss && subscription.rssData != nil
    }

    func subtitle(for subscription: UnifiedSubscription) -> String? {
        switch kind {
        case .rss: subscription.rssData?.feedUrl ?? subscription.description
        case .newsletters: subscription.emailData?.emailAddress ?? subscription.description
        }
    }

    func load() async {
        defer { isLoading = false }
        do {
            let all = try await read.listAllSubscriptions().subscriptions
            subscriptions = all.filter { kind == .newsletters ? $0.type == .email : $0.type != .email }
        } catch {
            logger.error("Loading subscriptions failed: \(error.localizedDescription)")
            errorMessage = "Failed to load subscriptions. Please try again."
        }
    }

    /// Routes the source's new items to `list`; picking the current list does nothing.
    func setList(_ list: ContentList, for subscription: UnifiedSubscription) async {
        guard let route = Self.route(for: subscription),
            let current = subscriptions.first(where: { $0.id == subscription.id }), current.list != list
        else { return }
        let previous = current.list
        update(subscription.id, list: list)
        do {
            _ = try await read.setSourceList(route.type, key: route.key, list: list)
        } catch {
            logger.error("Updating the source list failed: \(error.localizedDescription)")
            update(subscription.id, list: previous)
            errorMessage = "Failed to update. Please try again."
        }
    }

    /// Removes the feed at once and puts it back where it was if the server refuses.
    func unsubscribe(_ subscription: UnifiedSubscription) async {
        guard let feedID = subscription.rssData?.feedId, subscription.type == .rss,
            let index = subscriptions.firstIndex(where: { $0.id == subscription.id })
        else { return }
        subscriptions.remove(at: index)
        do {
            try await read.unsubscribeFromRSSFeed(feedID)
        } catch {
            logger.error("Unsubscribing failed: \(error.localizedDescription)")
            subscriptions.insert(subscription, at: min(index, subscriptions.count))
            errorMessage = "Failed to unsubscribe. Please try again."
        }
    }

    /// Fetches the newsletter address once; a failure stays until the screen is shown again.
    func loadEmailAddress() async {
        guard emailAddress == nil else { return }
        do {
            emailAddress = try await read.getOrCreateEmailAddress()
            emailAddressError = nil
        } catch {
            logger.error("Loading the newsletter address failed: \(error.localizedDescription)")
            emailAddressError = "Unable to load address"
        }
    }

    private func update(_ id: String, list: ContentList) {
        guard let index = subscriptions.firstIndex(where: { $0.id == id }) else { return }
        subscriptions[index].list = list
    }
}
