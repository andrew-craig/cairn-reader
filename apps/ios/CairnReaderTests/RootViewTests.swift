import Foundation
import Testing
@testable import CairnReader

@MainActor
struct RootViewTests {
    private func model(_ session: StoredSession?) -> SessionModel {
        let server = ServerConfig(defaults: UserDefaults(suiteName: UUID().uuidString)!)
        let auth = AuthService(
            client: HTTPClient(server: server, session: StubServer(replies: []).session()),
            storage: InMemorySessionStorage(session), deviceID: { "device-1" }, sleep: { _ in })
        return SessionModel(
            auth: auth,
            makeSession: { UserSession(user: $0, auth: $1, database: AppDatabase.inMemory()) })
    }

    @Test func rootFollowsTheSession() async {
        let model = model(AuthFixtures.valid)
        guard case .loading = RootView.Screen(model) else { Issue.record("expected loading"); return }

        await model.start()
        guard case let .signedIn(session) = RootView.Screen(model) else { Issue.record("expected signed in"); return }
        #expect(session.user == AuthFixtures.user)

        await model.logout()
        guard case .signedOut = RootView.Screen(model) else { Issue.record("expected signed out"); return }
    }

    @Test func rootIsSignedOutWithoutASavedSession() async {
        let model = model(nil)
        await model.start()
        guard case .signedOut = RootView.Screen(model) else { Issue.record("expected signed out"); return }
    }

    @Test func rejectionWording() {
        #expect(MainTabs.message(for: .init(articleID: "a", field: .status)) == "Failed to archive article")
        #expect(MainTabs.message(for: .init(articleID: "a", field: .list)) == "Failed to save to Reads")
    }
}
