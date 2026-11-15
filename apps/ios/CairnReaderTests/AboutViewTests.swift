import Foundation
import Testing
@testable import CairnReader

// The server version text of apps/mobile/src/screens/AboutScreen.tsx.

@MainActor
struct AboutViewTests {
    private func makeSystem(_ reply: StubServer.Reply) -> SystemService {
        let config = ServerConfig(defaults: UserDefaults(suiteName: UUID().uuidString)!)
        return SystemService(client: HTTPClient(server: config, session: StubServer(replies: [reply]).session()))
    }

    @Test func showsTheReportedVersion() async {
        let system = makeSystem(.json(200, #"{ "status": "ok", "version": "1.2.3" }"#))
        #expect(await AboutView.serverVersionText(system) == "1.2.3")
    }

    @Test func noReportedVersionIsUnknown() async {
        let system = makeSystem(.json(200, #"{ "status": "ok" }"#))
        #expect(await AboutView.serverVersionText(system) == "Unknown")
    }

    @Test func failureIsUnavailable() async {
        let system = makeSystem(.failure(.cannotConnectToHost))
        #expect(await AboutView.serverVersionText(system) == "Unavailable")
    }

    @Test func appVersionComesFromTheBundle() {
        #expect(!AboutView.appVersion.isEmpty)
    }
}
