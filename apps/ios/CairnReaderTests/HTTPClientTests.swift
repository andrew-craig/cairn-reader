import Foundation
import Testing
@testable import CairnReader

struct HTTPClientTests {
    func client(_ handler: @escaping StubURLProtocol.Handler) -> HTTPClient {
        let defaults = UserDefaults(suiteName: UUID().uuidString)!
        return HTTPClient(server: ServerConfig(defaults: defaults), session: StubURLProtocol.session(handler))
    }

    @Test func buildsRequestAndDecodes() async throws {
        let client = client { request in
            #expect(request.httpMethod == "PUT")
            #expect(
                request.url?.absoluteString
                    == "https://cairn.seatrain.net/api/v1/content/user/user-1/content-1/feedback?a=1&b=two%20words")
            #expect(request.value(forHTTPHeaderField: "Authorization") == "Bearer token")
            #expect(request.value(forHTTPHeaderField: "Content-Type") == "application/json")
            let body = try JSONSerialization.jsonObject(with: request.httpBody ?? Data()) as? [String: Any]
            #expect(body?["categories"] as? [String] == ["lead_image"])
            return (200, Fixtures.feedback.utf8Data)
        }
        let request = HTTPRequest(
            .put, "/api/v1/content/user/user-1/content-1/feedback",
            query: [URLQueryItem(name: "a", value: "1"), URLQueryItem(name: "b", value: "two words")],
            body: ExtractionFeedbackRequest(categories: [.leadImage]),
            headers: ["Authorization": "Bearer token"])
        let response = try await client.send(request, as: Envelope<ExtractionFeedbackResponse>.self)
        #expect(response.data.contentId == "content-1")
    }

    @Test func usesSavedServer() async throws {
        let defaults = UserDefaults(suiteName: UUID().uuidString)!
        let server = ServerConfig(defaults: defaults)
        #expect(server.setURL("cairn.example.com/"))
        let client = HTTPClient(server: server, session: StubURLProtocol.session { request in
            #expect(request.url?.absoluteString == "https://cairn.example.com/health/live")
            #expect(request.httpBody == nil)
            return (200, Data())
        })
        try await client.send(HTTPRequest(.get, "/health/live"))
    }

    @Test func timeoutIsNetworkError() async {
        let client = client { _ in throw URLError(.timedOut) }
        await #expect(throws: NetworkError.timeout) {
            try await client.send(HTTPRequest(.get, "/x"))
        }
    }

    @Test(arguments: [URLError.Code.notConnectedToInternet, .cannotConnectToHost, .networkConnectionLost])
    func transportFailureIsUnreachable(code: URLError.Code) async {
        let client = client { _ in throw URLError(code) }
        await #expect(throws: NetworkError.unreachable) {
            try await client.send(HTTPRequest(.get, "/x"))
        }
    }

    @Test func notFoundIsHttpErrorWithMessage() async {
        let client = client { _ in (404, Fixtures.error404.utf8Data) }
        await #expect(throws: HttpError(status: 404, message: "content not found")) {
            try await client.send(HTTPRequest(.get, "/x"), as: Envelope<UserContentResponse>.self)
        }
    }

    @Test func serverErrorWithoutJSONIsHttpError() async {
        let client = client { _ in (500, "<html>Bad Gateway</html>".utf8Data) }
        await #expect(throws: HttpError(status: 500, message: nil)) {
            try await client.send(HTTPRequest(.get, "/x"))
        }
    }

    @Test func badJSONIsNetworkError() async {
        let client = client { _ in (200, "{\"data\": ".utf8Data) }
        await #expect(throws: NetworkError.undecodableBody) {
            try await client.send(HTTPRequest(.get, "/x"), as: Envelope<UserContentResponse>.self)
        }
    }
}

struct ServerConfigTests {
    let defaults = UserDefaults(suiteName: UUID().uuidString)!

    @Test func defaultsToCairn() {
        #expect(ServerConfig(defaults: defaults).url.absoluteString == "https://cairn.seatrain.net")
    }

    @Test(arguments: [
        ("https://cairn.example.com", "https://cairn.example.com"),
        ("  cairn.example.com//  ", "https://cairn.example.com"),
        ("http://localhost:8080/", "http://localhost:8080"),
        ("HTTPS://Cairn.example.com", "HTTPS://Cairn.example.com"),
        ("https:/cairn.example.com", "https://cairn.example.com"),
        ("https://", "https://cairn.seatrain.net"),
        ("", "https://cairn.seatrain.net"),
    ] as [(String, String)])
    func normalizes(input: String, expected: String) {
        #expect(ServerConfig.normalize(input).absoluteString == expected)
    }

    @Test func savingReportsChangesAndPersists() {
        let server = ServerConfig(defaults: defaults)
        #expect(server.setURL("cairn.example.com"))
        #expect(ServerConfig(defaults: defaults).url.absoluteString == "https://cairn.example.com")
        #expect(!server.setURL("https://cairn.example.com/"))

        #expect(server.setURL(""))
        #expect(server.url == ServerConfig.defaultURL)
        #expect(defaults.string(forKey: "serverURL") == nil)
        #expect(!server.setURL(ServerConfig.defaultURL.absoluteString))
    }
}

struct RelativeDateTests {
    let now = Date(timeIntervalSince1970: 1_736_935_200)
    let locale = Locale(identifier: "en_US")

    @Test(arguments: [
        (-3_600.0, "Today"),
        (3_600.0, "Today"),
        (86_400.0, "Yesterday"),
        (3 * 86_400.0, "3 days ago"),
        (14 * 86_400.0, "2 weeks ago"),
        (65 * 86_400.0, "2 months ago"),
        (800 * 86_400.0, "2 years ago"),
    ] as [(Double, String)])
    func labels(secondsAgo: Double, expected: String) {
        #expect(now.addingTimeInterval(-secondsAgo).relativeDayLabel(now: now, locale: locale) == expected)
    }
}
