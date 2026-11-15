import Foundation
import Testing
@testable import CairnReader

// Port of the sign-in cases of apps/mobile/src/screens/LoginScreen.test.tsx.

@MainActor
private struct Harness {
    let model: LoginModel
    let server: StubServer
    let storage: InMemorySessionStorage
    let config: ServerConfig
    let signedIn: SignInCounter

    init(_ server: StubServer, session: StoredSession? = nil) {
        self.server = server
        config = ServerConfig(defaults: UserDefaults(suiteName: UUID().uuidString)!)
        storage = InMemorySessionStorage(session)
        let auth = AuthService(
            client: HTTPClient(server: config, session: server.session()), storage: storage,
            deviceID: { "device-1" }, sleep: { _ in })
        let signedIn = SignInCounter()
        self.signedIn = signedIn
        model = LoginModel(auth: auth, server: config) { signedIn.count += 1 }
    }
}

@MainActor
private final class SignInCounter {
    var count = 0
}

private let authOK = StubServer.Reply.json(200, AuthFixtures.authBody())
private let unauthorized = StubServer.Reply.json(401, AuthFixtures.errorBody("device not found"))

@MainActor
struct LoginModelTests {
    @Test func deviceLoginSignsIn() async {
        let h = Harness(StubServer(replies: [authOK]))
        await h.model.continueWithDevice()
        #expect(h.server.paths == ["/api/v1/auth/login/mobile"])
        #expect(h.signedIn.count == 1)
        #expect(h.model.errorMessage == nil)
        #expect(!h.model.isWorking)
    }

    @Test func deviceLoginFallsBackToRegisterOn401() async {
        let h = Harness(StubServer(replies: [unauthorized, authOK]))
        await h.model.continueWithDevice()
        #expect(h.server.paths == ["/api/v1/auth/login/mobile", "/api/v1/auth/register/mobile"])
        #expect(h.signedIn.count == 1)
    }

    @Test(arguments: [
        StubServer.Reply.json(500, AuthFixtures.errorBody("boom")),
        StubServer.Reply.json(429, AuthFixtures.errorBody("slow down")),
        StubServer.Reply.failure(.notConnectedToInternet),
    ])
    func deviceLoginDoesNotRegisterOtherFailures(reply: StubServer.Reply) async {
        let h = Harness(StubServer(replies: [reply, authOK]))
        await h.model.continueWithDevice()
        #expect(h.server.paths == ["/api/v1/auth/login/mobile"])
        #expect(h.signedIn.count == 0)
        #expect(h.model.errorMessage != nil)
    }

    @Test func registerFailureAfterFallbackIsShown() async {
        let h = Harness(StubServer(replies: [unauthorized, .json(409, AuthFixtures.errorBody("conflict"))]))
        await h.model.continueWithDevice()
        #expect(h.model.errorMessage == "conflict")
        #expect(h.signedIn.count == 0)
    }

    @Test func emailSignInSendsTrimmedEmail() async {
        let h = Harness(StubServer(replies: [authOK]))
        h.model.email = "  me@example.com "
        h.model.password = "secret"
        await h.model.signIn()
        #expect(h.server.paths == ["/api/v1/auth/login"])
        #expect(h.server.requests[0].jsonBody["email"] as? String == "me@example.com")
        #expect(h.signedIn.count == 1)
        // The next person to see the form does not find this account's details in it.
        #expect(h.model.email.isEmpty && h.model.password.isEmpty)
    }

    @Test func createAccountRegisters() async {
        let h = Harness(StubServer(replies: [authOK]))
        h.model.email = "me@example.com"
        h.model.password = "secret"
        await h.model.createAccount()
        #expect(h.server.paths == ["/api/v1/auth/register"])
        #expect(h.signedIn.count == 1)
    }

    @Test func emailActionsNeedBothFields() async {
        let h = Harness(StubServer(replies: []))
        h.model.email = "me@example.com"
        await h.model.signIn()
        #expect(h.model.errorMessage == "Please enter both email and password")
        h.model.email = ""
        h.model.password = "secret"
        await h.model.createAccount()
        #expect(h.model.errorMessage == "Please enter both email and password")
        #expect(h.server.requests.isEmpty)
    }

    @Test func rejectedLoginShowsServerMessageAndKeepsFields() async {
        let h = Harness(StubServer(replies: [.json(401, AuthFixtures.errorBody("Invalid credentials"))]))
        h.model.email = "me@example.com"
        h.model.password = "wrong"
        await h.model.signIn()
        #expect(h.model.errorMessage == "Invalid credentials")
        #expect(h.model.password == "wrong")
        #expect(h.signedIn.count == 0)
    }

    @Test func nextActionClearsTheError() async {
        let h = Harness(StubServer(replies: [.json(500, AuthFixtures.errorBody("boom")), authOK]))
        await h.model.continueWithDevice()
        #expect(h.model.errorMessage == "boom")
        await h.model.continueWithDevice()
        #expect(h.model.errorMessage == nil)
    }

    @Test func serverChangeClearsTheSession() async {
        let h = Harness(StubServer(replies: []), session: AuthFixtures.valid)
        h.model.serverURL = "selfhost.example.com/"
        await h.model.saveServer()
        #expect(h.config.url.absoluteString == "https://selfhost.example.com")
        #expect(h.model.serverURL == "https://selfhost.example.com")
        #expect(h.storage.session == nil)
    }

    @Test func unchangedServerKeepsTheSession() async {
        let h = Harness(StubServer(replies: []), session: AuthFixtures.valid)
        await h.model.saveServer()
        #expect(h.storage.session != nil)
    }

    @Test func emptyServerFieldMeansTheDefault() async {
        let h = Harness(StubServer(replies: []))
        h.model.serverURL = "https://selfhost.example.com"
        await h.model.saveServer()
        h.model.serverURL = ""
        await h.model.saveServer()
        #expect(h.config.url == ServerConfig.defaultURL)
        #expect(h.model.serverURL == ServerConfig.defaultURL.absoluteString)
    }
}
