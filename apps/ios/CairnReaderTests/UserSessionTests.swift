import Foundation
import Synchronization
import Testing
@testable import CairnReader

struct UserSessionTests {
    @Test func closeCancelsAndAwaitsWorkBeforeErasingTheDatabase() async throws {
        let url = try temporaryDatabaseURL()
        let session = UserSession(user: AuthFixtures.user, database: AppDatabase(url: url))
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
        let session = UserSession(user: AuthFixtures.user, database: .inMemory())
        await session.close()
        let ran = Mutex(false)

        session.start { ran.withLock { $0 = true } }
        try await Task.sleep(for: .milliseconds(50))

        #expect(!ran.withLock { $0 })
    }
}
