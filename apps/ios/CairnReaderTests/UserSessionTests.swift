import Foundation
import Synchronization
import Testing
@testable import CairnReader

struct UserSessionTests {
    private func makeSession(url: URL? = nil) -> UserSession {
        let auth = AuthService(
            client: HTTPClient(
                server: ServerConfig(defaults: UserDefaults(suiteName: UUID().uuidString)!),
                session: StubServer(replies: []).session()),
            storage: InMemorySessionStorage(AuthFixtures.valid), deviceID: { "device-1" }, sleep: { _ in })
        return UserSession(
            user: AuthFixtures.user, auth: auth, database: url.map { AppDatabase(url: $0) } ?? .inMemory())
    }

    @Test func closeCancelsAndAwaitsWorkBeforeErasingTheDatabase() async throws {
        let url = try temporaryDatabaseURL()
        let session = makeSession(url: url)
        try await session.articles.upsertMany([storedArticle("a1")])
        let fileExistedWhenWorkEnded = Mutex<Bool?>(nil)
        let started = AsyncStream<Void>.makeStream()

        session.start {
            started.continuation.yield()
            while !Task.isCancelled { await Task.yield() }
            fileExistedWhenWorkEnded.withLock { $0 = FileManager.default.fileExists(atPath: url.path()) }
        }
        var iterator = started.stream.makeAsyncIterator()
        await iterator.next()

        await session.close()

        #expect(fileExistedWhenWorkEnded.withLock { $0 } == true)
        #expect(!FileManager.default.fileExists(atPath: url.path()))
    }

    @Test func startDoesNothingOnceClosed() async throws {
        let session = makeSession()
        await session.close()
        let ran = Mutex(false)

        session.start { ran.withLock { $0 = true } }
        try await Task.sleep(for: .milliseconds(50))

        #expect(!ran.withLock { $0 })
    }
}
