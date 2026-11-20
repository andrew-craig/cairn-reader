import Foundation
import Testing
@testable import CairnReader

// Port of apps/mobile/src/components/AddLinkModal.test.tsx against `AddLinkModel`, plus the
// debounce, superseded detection, Find feed outcomes and error wording.

/// A response the test answers later (or never).
@MainActor
private final class Pending<Value: Sendable> {
    private var continuation: CheckedContinuation<Value, any Error>?
    var isWaiting: Bool { continuation != nil }

    func wait() async throws -> Value {
        try await withCheckedThrowingContinuation { continuation = $0 }
    }

    func resolve(_ value: Value) {
        continuation?.resume(returning: value)
        continuation = nil
    }
}

private struct Failure: Error {}

private func detection(_ url: String, _ type: URLType, title: String? = nil) -> DetectURLResponse {
    DetectURLResponse(url: url, type: type, title: title)
}

private func feedResponse(_ list: ContentList) -> AddURLResponse {
    .feed(
        feedId: "f1",
        subscription: FeedSubscription(
            id: "s1", userId: "u", feedId: "f1", feedUrl: "https://example.com/rss", title: "Example", list: list,
            subscribedAt: Date(timeIntervalSince1970: 0)))
}

private func pageResponse() -> AddURLResponse {
    // Fixture JSON: a decoding failure is a broken test, not a case to handle.
    let page = try! HTTPClient.decoder.decode(
        Page<UserContentResponse>.self, from: SyncFixtures.readsPage(["a"]).utf8Data)
    return .page(page.data[0])
}

/// The requests a model makes, answered by `detect`, `discover` and `add`.
@MainActor
private final class Source {
    private(set) var detected: [String] = []
    private(set) var discovered: [String] = []
    private(set) var added: [AddURLRequest] = []
    var detect: (String) async throws -> DetectURLResponse = { detection($0, .page) }
    var discover: (String) async throws -> [DiscoveredFeed] = { _ in [] }
    var add: (AddURLRequest) async throws -> AddURLResponse = { _ in pageResponse() }

    func model(_ mode: AddLinkModel.Mode = .link) -> AddLinkModel {
        AddLinkModel(
            mode: mode,
            detect: { url in
                self.detected.append(url)
                return try await self.detect(url)
            },
            discover: { url in
                self.discovered.append(url)
                return try await self.discover(url)
            },
            add: { request in
                self.added.append(request)
                return try await self.add(request)
            },
            debounce: .milliseconds(20))
    }
}

/// Sets the URL and waits for its detection to finish.
@MainActor
private func enter(_ url: String, in model: AddLinkModel) async {
    model.url = url
    await eventually { !model.isDetecting }
}

@MainActor
struct AddLinkModelTests {
    // MARK: Detection

    @Test func detectionIsDebouncedToTheLastEdit() async {
        let source = Source()
        let model = source.model()

        model.url = "https://example.com/a"
        model.url = "https://example.com/ab"
        model.url = "https://example.com/abc"
        #expect(model.isDetecting)
        #expect(model.addTitle == "Detecting…")
        await eventually { !model.isDetecting }

        #expect(source.detected == ["https://example.com/abc"])
        #expect(model.summary == "Article")
    }

    @Test func aSupersededDetectionIsNotApplied() async {
        let source = Source()
        let pending = Pending<DetectURLResponse>()
        source.detect = { url in url.hasSuffix("/rss") ? try await pending.wait() : detection(url, .page) }
        let model = source.model()

        model.url = "https://example.com/rss"
        await eventually { pending.isWaiting }
        await enter("https://example.com/post", in: model)
        pending.resolve(detection("https://example.com/rss", .feed))
        try? await Task.sleep(for: .milliseconds(50))

        #expect(!model.isFeed)
        #expect(model.detection?.url == "https://example.com/post")
    }

    @Test func aURLWithoutASchemeIsDetectedAsHTTPS() async {
        let source = Source()
        let model = source.model()
        await enter("  example.com/post ", in: model)
        #expect(source.detected == ["https://example.com/post"])
    }

    @Test func aFailedDetectionIsUnknownAndStillAdds() async {
        let source = Source()
        source.detect = { _ in throw Failure() }
        let model = source.model()
        await enter("https://example.com", in: model)

        #expect(model.detection?.type == .unknown)
        #expect(model.canAdd)
        await model.addURL()
        #expect(source.added.first?.type == .unknown)
        #expect(source.added.first?.list == nil)
    }

    // MARK: Dismissal

    /// Jest: "closes on request while a detection is in flight".
    @Test func closingIsAllowedWhileADetectionIsInFlight() async {
        let source = Source()
        let pending = Pending<DetectURLResponse>()
        source.detect = { _ in try await pending.wait() }
        let model = source.model()

        model.url = "https://example.com"
        await eventually { pending.isWaiting }
        #expect(model.canDismiss)

        model.close()
        #expect(!model.isDetecting)
        pending.resolve(detection("https://example.com", .feed))
        try? await Task.sleep(for: .milliseconds(50))
        #expect(model.detection == nil)
    }

    /// Jest: "still blocks dismissal while a real submit is in flight".
    @Test func closingIsBlockedWhileAnAddIsInFlight() async {
        let source = Source()
        let pending = Pending<AddURLResponse>()
        source.add = { _ in try await pending.wait() }
        let model = source.model()
        await enter("https://example.com", in: model)

        let adding = Task { await model.addURL() }
        await eventually { pending.isWaiting }
        #expect(model.isAdding)
        #expect(!model.canDismiss)
        #expect(model.addTitle == "Adding…")

        pending.resolve(pageResponse())
        await adding.value
        #expect(model.canDismiss)
    }

    /// Jest: "does not fire the Multiple feeds Alert when discoverFeed resolves after close".
    @Test func aDiscoveryThatLandsAfterCloseOffersNoChoices() async {
        let source = Source()
        let pending = Pending<[DiscoveredFeed]>()
        source.discover = { _ in try await pending.wait() }
        let model = source.model()
        await enter("https://example.com", in: model)

        let finding = Task { await model.findFeed() }
        await eventually { pending.isWaiting }
        #expect(model.canDismiss)
        model.close()

        pending.resolve([
            DiscoveredFeed(url: "https://example.com/feed1", title: "Feed 1"),
            DiscoveredFeed(url: "https://example.com/feed2", title: "Feed 2"),
        ])
        await finding.value
        #expect(model.feedChoices == nil)
        #expect(model.url == "https://example.com")
    }

    // MARK: Feed/Reads choice (epic 6e4d)

    @Test func aNewFeedSubscriptionDefaultsToReads() async {
        let source = Source()
        source.detect = { detection($0, .feed, title: "Example") }
        source.add = { _ in feedResponse(.reads) }
        let model = source.model()
        await enter("https://example.com/rss", in: model)
        #expect(model.addTitle == "Add Feed")

        await model.addURL()

        #expect(source.added.first?.type == .feed)
        #expect(source.added.first?.list == .reads)
        #expect(source.added.first?.title == "Example")
        #expect(model.successMessage == "Subscribed to Example. New items will appear in Reads.")
    }

    @Test func subscribesToFeedWhenTheUserPicksFeedAndSaysWhereItemsLand() async {
        let source = Source()
        source.detect = { detection($0, .feed, title: "Example") }
        source.add = { _ in feedResponse(.feed) }
        let model = source.model()
        await enter("https://example.com/rss", in: model)

        model.list = .feed
        await model.addURL()

        #expect(source.added.first?.type == .feed)
        #expect(source.added.first?.list == .feed)
        #expect(model.successMessage == "Subscribed to Example. New items will appear in Feed.")
    }

    @Test func aStaleFeedDetectionIsDroppedAndAddIsIgnoredUntilTheNewURLIsDetected() async {
        let source = Source()
        let pending = Pending<DetectURLResponse>()
        source.detect = { url in url.hasSuffix("/rss") ? detection(url, .feed, title: "Example") : try await pending.wait() }
        let model = source.model()
        await enter("https://example.com/rss", in: model)
        #expect(model.isFeed)

        // Edit to a different URL: the feed result and its Feed/Reads choice go away.
        model.url = "https://example.com/post"
        #expect(!model.isFeed)
        #expect(model.addTitle != "Add Feed")

        // Adding now would drop type and list for lack of a detection, so it is ignored.
        await model.submit()
        #expect(source.added.isEmpty)

        // Once the new URL is detected (as a page), Add goes out without a list.
        await eventually { pending.isWaiting }
        pending.resolve(detection("https://example.com/post", .page))
        await eventually { !model.isDetecting }
        await model.submit()
        #expect(source.added.count == 1)
        #expect(source.added.first?.type == .page)
        #expect(source.added.first?.list == nil)
    }

    @Test func doesNotStayDetectingWhenTheURLBecomesInvalidMidDetection() async {
        let source = Source()
        let pending = Pending<DetectURLResponse>()
        source.detect = { _ in try await pending.wait() }
        let model = source.model()
        model.url = "https://example.com/post"
        await eventually { pending.isWaiting }

        // The URL becomes invalid while that detection is still in flight.
        model.url = "not a url"
        #expect(!model.isDetecting)
        await model.submit()

        #expect(model.error == "Please enter a valid URL")
        #expect(source.added.isEmpty)
    }

    @Test func aPageOffersNoChoiceAndSendsNoList() async {
        let source = Source()
        let model = source.model()
        await enter("https://example.com/post", in: model)

        #expect(!model.isFeed)
        #expect(model.addTitle == "Add")
        await model.addURL()

        #expect(source.added.first?.list == nil)
        #expect(model.successMessage == "Article added to reading list")
    }

    // MARK: Feed mode

    /// Jest: "explains both destinations when a feed is detected". The explanation is the sheet's
    /// section footer; what the model owns is that the choice shows, defaulting to Reads.
    @Test func feedModeOffersTheChoiceForADetectedFeed() async {
        let source = Source()
        source.detect = { detection($0, .feed, title: "Example") }
        let model = source.model(.feed)
        await enter("https://example.com/rss", in: model)

        #expect(model.isFeed)
        #expect(model.showsAdd)
        #expect(!model.showsFindFeed)
        #expect(model.list == .reads)
        #expect(model.addTitle == "Add Feed")
    }

    @Test func feedModeNeverSavesAPageAndReturnRunsFindFeed() async {
        let source = Source()
        source.discover = { _ in [DiscoveredFeed(url: "https://example.com/rss", title: "Example")] }
        let model = source.model(.feed)
        await enter("https://example.com", in: model)

        #expect(!model.showsAdd)
        #expect(model.showsFindFeed)
        await model.addURL()
        #expect(source.added.isEmpty)

        await model.submit()
        #expect(source.discovered == ["https://example.com"])
        #expect(source.added.isEmpty)
    }

    @Test func feedModeRunsFindFeedOnReturnWhileDetectionIsInFlight() async {
        let source = Source()
        let pending = Pending<DetectURLResponse>()
        source.detect = { _ in try await pending.wait() }
        let model = source.model(.feed)
        model.url = "https://example.com"
        await eventually { pending.isWaiting }

        #expect(!model.showsAdd)
        await model.submit()
        #expect(source.discovered == ["https://example.com"])
    }

    // MARK: Find feed

    @Test func oneFoundFeedReplacesTheURLAndIsDetectedAsAFeed() async {
        let source = Source()
        source.detect = { url in url.hasSuffix("/rss") ? detection(url, .feed, title: "Example") : detection(url, .page) }
        source.discover = { _ in [DiscoveredFeed(url: "https://example.com/rss", title: "Example")] }
        let model = source.model()
        await enter("https://example.com", in: model)

        await model.findFeed()
        #expect(model.url == "https://example.com/rss")
        await eventually { !model.isDetecting }
        #expect(model.isFeed)
        #expect(model.summary == "RSS feed: Example")
    }

    @Test func aDiscoveryForAURLSinceEditedIsDropped() async {
        let source = Source()
        let pending = Pending<[DiscoveredFeed]>()
        source.discover = { _ in try await pending.wait() }
        let model = source.model()
        await enter("https://example.com", in: model)

        let finding = Task { await model.findFeed() }
        await eventually { pending.isWaiting }
        model.url = "https://example.org"

        pending.resolve([DiscoveredFeed(url: "https://example.com/rss", title: "Example")])
        await finding.value
        #expect(model.url == "https://example.org")
        #expect(model.feedChoices == nil)
        #expect(model.error == nil)
        #expect(!model.isDiscovering)
    }

    @Test func severalFoundFeedsAreOfferedToPickFrom() async {
        let source = Source()
        let feeds = [
            DiscoveredFeed(url: "https://example.com/feed1", title: "Feed 1"),
            DiscoveredFeed(url: "https://example.com/feed2", title: ""),
        ]
        source.discover = { _ in feeds }
        let model = source.model()
        await enter("https://example.com", in: model)

        await model.findFeed()
        #expect(model.feedChoices == feeds)
        #expect(model.url == "https://example.com")
    }

    @Test func noFoundFeedIsAnError() async {
        let source = Source()
        let model = source.model()
        await enter("https://example.com", in: model)

        await model.findFeed()
        #expect(model.error == "No RSS feed found for this site")
        #expect(!model.isDiscovering)

        // Editing the URL clears it.
        model.url = "https://example.org"
        #expect(model.error == nil)
    }

    // MARK: Errors

    @Test(arguments: [
        ("Already subscribed to this feed", "Already subscribed to this feed"),
        ("user is already subscribed to this feed", "Already subscribed to this feed"),
        ("Failed to subscribe to feed", "Failed to subscribe to feed"),
        ("Failed to add content to user: boom", "Failed to add article"),
        ("Invalid feed URL", "Invalid feed URL"),
    ])
    func aFailedAddShowsTheRNWording(server: String, shown: String) async {
        let source = Source()
        source.add = { _ in throw HttpError(status: 409, message: server) }
        let model = source.model()
        await enter("https://example.com", in: model)

        await model.addURL()
        #expect(model.error == shown)
        #expect(model.successMessage == nil)
        #expect(!model.isAdding)
    }

    @Test func anUnreachableServerShowsTheNetworkMessage() async {
        let source = Source()
        source.add = { _ in throw NetworkError.unreachable }
        let model = source.model()
        await enter("https://example.com", in: model)

        await model.addURL()
        #expect(model.error == "Unable to reach the server. Please try again later.")
    }

    @Test func anInvalidURLIsNotSentToFindFeed() async {
        let source = Source()
        let model = source.model()
        model.url = "not a url"

        await model.findFeed()
        #expect(model.error == "Please enter a valid URL")
        #expect(source.discovered.isEmpty)
        #expect(source.detected.isEmpty)
    }
}
