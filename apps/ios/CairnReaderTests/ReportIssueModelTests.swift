import Foundation
import Synchronization
import Testing
@testable import CairnReader

// Port of apps/mobile/src/components/ReportIssueModal.test.tsx against `ReportIssueModel`. The
// "Report sent" toast and closing the sheet are the view's (`send()` answering true).

private let reportBody = """
    {
      "data": {
        "content_id": "c-1", "categories": ["title"], "comment": "earlier note",
        "status": "open", "created_at": "2025-01-16T12:00:00Z", "updated_at": "2025-01-16T12:00:00Z"
      }
    }
    """
private let noReport = StubServer.Reply.json(404, #"{ "error": "not_found" }"#)

/// The feedback endpoint: GET answers `existing`, PUT answers `submit`, and a PUT waits while
/// `holdSubmit` is set.
private final class FeedbackServer: Sendable {
    let existing = Mutex(noReport)
    let submit = Mutex(StubServer.Reply.json(200, reportBody))
    let holdSubmit = Mutex(false)

    func stub() -> StubServer {
        StubServer { [self] request in
            if request.httpMethod == "PUT" {
                while holdSubmit.withLock({ $0 }) { Thread.sleep(forTimeInterval: 0.005) }
                return submit.withLock { $0 }
            }
            return existing.withLock { $0 }
        }
    }
}

@MainActor
private func makeModel(_ server: StubServer) -> ReportIssueModel {
    let config = ServerConfig(defaults: UserDefaults(suiteName: UUID().uuidString)!)
    let auth = AuthService(
        client: HTTPClient(server: config, session: server.session()),
        storage: InMemorySessionStorage(AuthFixtures.valid), deviceID: { "device-1" }, sleep: { _ in })
    return ReportIssueModel(read: ReadService(auth: auth, userID: AuthFixtures.user.id), contentID: "c-1")
}

private let feedbackPath = "/api/v1/content/user/user-1/c-1/feedback"

@MainActor
struct ReportIssueModelTests {
    @Test func sendIsDisabledUntilAReasonIsSelected() async {
        let server = FeedbackServer().stub()
        let model = makeModel(server)
        await model.load()

        #expect(server.paths == [feedbackPath])
        #expect(!model.canSend)
        model.toggle(.images)
        #expect(model.selected == [.images])
        #expect(model.canSend)
        model.toggle(.images)
        #expect(model.selected.isEmpty)
        #expect(!model.canSend)
    }

    @Test func prefillsFromTheExistingReportEachTimeItOpens() async {
        let feedback = FeedbackServer()
        feedback.existing.withLock { $0 = .json(200, reportBody) }
        let server = feedback.stub()

        let first = makeModel(server)
        await first.load()
        #expect(first.selected == [.title])
        #expect(first.comment == "earlier note")
        #expect(!first.selected.contains(.images))

        // Closed without sending, then opened again: the sheet builds a new model.
        first.toggle(.title)
        first.comment = ""
        let second = makeModel(server)
        await second.load()
        #expect(second.selected == [.title])
        #expect(second.comment == "earlier note")
        #expect(server.paths == [feedbackPath, feedbackPath])
    }

    @Test func aFailedLoadLeavesTheFormEmpty() async {
        let feedback = FeedbackServer()
        feedback.existing.withLock { $0 = .json(500, AuthFixtures.errorBody("boom")) }
        let model = makeModel(feedback.stub())

        await model.load()

        #expect(model.selected.isEmpty)
        #expect(model.comment.isEmpty)
        #expect(!model.isLoading)
        #expect(model.error == nil)
    }

    @Test func sendsCategoriesInListOrderWithTheTrimmedComment() async throws {
        let server = FeedbackServer().stub()
        let model = makeModel(server)
        await model.load()

        // Tapped out of order: categories are sent in list order.
        model.toggle(.title)
        model.toggle(.images)
        model.comment = "  see top \n"
        #expect(await model.send())

        let request = try #require(server.requests.last)
        #expect(request.httpMethod == "PUT")
        #expect(request.url?.path() == feedbackPath)
        #expect(request.jsonBody["categories"] as? [String] == ["images", "title"])
        #expect(request.jsonBody["comment"] as? String == "see top")
        #expect(!model.isSending)
    }

    @Test func aBlankCommentIsNotSent() async throws {
        let server = FeedbackServer().stub()
        let model = makeModel(server)
        await model.load()

        model.toggle(.other)
        model.comment = "   "
        #expect(await model.send())

        let body = try #require(server.requests.last).jsonBody
        #expect(body["categories"] as? [String] == ["other"])
        #expect(body["comment"] == nil)
    }

    @Test func sendIsDisabledWhileSending() async {
        let feedback = FeedbackServer()
        feedback.holdSubmit.withLock { $0 = true }
        let server = feedback.stub()
        let model = makeModel(server)
        await model.load()
        model.toggle(.images)

        let sending = Task { await model.send() }
        await eventually { model.isSending }
        #expect(!model.canSend)
        #expect(await model.send() == false)

        feedback.holdSubmit.withLock { $0 = false }
        #expect(await sending.value)
        #expect(server.requests.filter { $0.httpMethod == "PUT" }.count == 1)
        #expect(model.canSend)
    }

    @Test func limitsTheDetailsTo2000Characters() {
        let model = makeModel(FeedbackServer().stub())

        model.comment = String(repeating: "a", count: 2001)

        #expect(ReportIssueModel.commentLimit == 2000)
        #expect(model.comment.count == 2000)
    }

    @Test func showsAnInlineErrorOnFailureAndKeepsTheSelection() async {
        let feedback = FeedbackServer()
        feedback.submit.withLock { $0 = .json(500, AuthFixtures.errorBody("Network down")) }
        let model = makeModel(feedback.stub())
        await model.load()

        model.toggle(.images)
        #expect(await model.send() == false)

        #expect(model.error == "Network down")
        #expect(model.selected == [.images])
        #expect(model.canSend)

        // Changing the reasons clears the error.
        model.toggle(.title)
        #expect(model.error == nil)
    }
}
