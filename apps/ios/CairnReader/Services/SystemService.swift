import Foundation

/// Backend metadata that is not tied to a session.
nonisolated struct SystemService: Sendable {
    let client: HTTPClient

    /// The version the server reports from `/health/live`, or nil when it reports none (a
    /// deployment that predates version embedding).
    func serverVersion() async throws -> String? {
        try await client.send(HTTPRequest(.get, "/health/live"), as: Health.self).version
    }

    private struct Health: Decodable {
        let version: String?
    }
}
