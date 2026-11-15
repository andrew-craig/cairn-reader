import Foundation
import GRDB
import Testing
@testable import CairnReader

struct AppDatabaseTests {
    @Test func opensAFileDatabaseInWALMode() async throws {
        let url = try temporaryDatabaseURL()
        let database = AppDatabase(url: url)

        try await ArticleStore(database: database).upsertMany([storedArticle("a1")])

        let mode = try await database.writer().read { db in try String.fetchOne(db, sql: "PRAGMA journal_mode") }
        #expect(mode == "wal")
        #expect(FileManager.default.fileExists(atPath: url.path()))
    }

    @Test func recreatesADatabaseThatCannotBeOpened() async throws {
        let url = try temporaryDatabaseURL()
        try Data("not a database, just some bytes long enough to look like a header".utf8).write(to: url)
        let store = ArticleStore(database: AppDatabase(url: url))

        try await store.upsertMany([storedArticle("a1")])

        #expect(try await store.getById("a1")?.id == "a1")
    }

    @Test func eraseDeletesTheFilesAndClosesTheDatabase() async throws {
        let url = try temporaryDatabaseURL()
        let database = AppDatabase(url: url)
        try await ArticleStore(database: database).upsertMany([storedArticle("a1")])

        try database.erase()

        for suffix in ["", "-wal", "-shm"] {
            #expect(!FileManager.default.fileExists(atPath: url.path() + suffix))
        }
        #expect(throws: (any Error).self) { try database.writer() }
    }

    @Test func eraseDeletesAFileThatWasNeverOpened() throws {
        let url = try temporaryDatabaseURL()
        try Data("left over".utf8).write(to: url)

        try AppDatabase(url: url).erase()

        #expect(!FileManager.default.fileExists(atPath: url.path()))
    }
}
