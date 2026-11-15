import Foundation
import GRDB
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
            user: AuthFixtures.user, auth: auth, isOnline: { true },
            database: url.map { AppDatabase(url: $0) } ?? .inMemory())
    }

    /// A session for `user` with its database in the App Group, as at sign-in; its server is
    /// unreachable.
    private func signIn(_ user: User) -> UserSession {
        let auth = AuthService(
            client: HTTPClient(
                server: ServerConfig(defaults: UserDefaults(suiteName: UUID().uuidString)!),
                session: StubServer(replies: []).session()),
            storage: InMemorySessionStorage(AuthFixtures.valid), deviceID: { "device-1" }, sleep: { _ in })
        return UserSession(user: user, auth: auth, isOnline: { true })
    }

    private func user() -> User {
        User(
            id: UUID().uuidString, email: nil, emailVerified: false, createdAt: Date(timeIntervalSince1970: 0),
            updatedAt: Date(timeIntervalSince1970: 0), lastLoginAt: nil)
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

    @Test func closeDuringADrainLeavesNoRequestOrWriteAfterwards() async throws {
        let entered = AsyncStream<Void>.makeStream()
        let harness = SyncHarness(
            StubServer { _ in
                entered.continuation.yield()
                return SyncFixtures.ok
            })
        for index in 0..<20 {
            try harness.enqueue("a\(index)", .isFavorite, UpdateUserContentRequest(isFavorite: true))
        }
        harness.sync.requestDrain()
        var iterator = entered.stream.makeAsyncIterator()
        await iterator.next()

        await harness.session.close()
        let sent = harness.server.requests.count
        try await Task.sleep(for: .milliseconds(100))

        #expect(harness.server.requests.count == sent)
        #expect(throws: (any Error).self) { try harness.session.database.writer() }
    }

    @Test func closeDuringAPageFetchLeavesNoRequestOrWriteAfterwards() async throws {
        let entered = AsyncStream<Void>.makeStream()
        let harness = SyncHarness(
            online: false,
            StubServer { _ in
                entered.continuation.yield()
                Thread.sleep(forTimeInterval: 0.1)
                return .json(200, SyncFixtures.readsPage(["a1"]))
            })
        let fetch = Task { try await harness.sync.syncReads(after: nil) }
        var iterator = entered.stream.makeAsyncIterator()
        await iterator.next()

        await harness.session.close()
        let sent = harness.server.requests.count
        _ = await fetch.result
        try await Task.sleep(for: .milliseconds(100))

        #expect(harness.server.requests.count == sent)
        #expect(throws: (any Error).self) { try harness.session.database.writer() }
    }

    @Test func signingInAfterCloseSeesAnEmptyStoreOutboxAndFeedCache() async throws {
        let first = user()
        let session = signIn(first)
        try await session.articles.upsertMany([storedArticle("a1")])
        try await session.mutations.setFavorite("a1", isFavorite: true)
        try session.feedCache.save([storedArticle("f1")])

        await session.close()

        // Another account, then the same account again.
        for next in [signIn(user()), signIn(first)] {
            #expect(try await next.articles.getById("a1") == nil)
            let rows = try await next.database.writer().read { db in
                try Int.fetchOne(db, sql: "SELECT COUNT(*) FROM outbox")
            }
            #expect(rows == 0)
            #expect(next.feedCache.load().isEmpty)
            await next.close()
        }
    }
}
