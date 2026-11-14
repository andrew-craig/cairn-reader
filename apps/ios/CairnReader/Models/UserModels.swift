import Foundation

// User service wire types (services/users/api/openapi.yaml).

nonisolated struct User: Codable, Sendable, Hashable {
    let id: String
    /// Omitted for device-only accounts.
    let email: String?
    let emailVerified: Bool
    let createdAt: Date
    let updatedAt: Date
    let lastLoginAt: Date?
}

/// Returned by login, register and token refresh.
nonisolated struct AuthResponse: Codable, Sendable {
    let user: User
    let accessToken: String
    /// Opaque, not a JWT.
    let refreshToken: String
    /// Access token lifetime in seconds.
    let expiresIn: Int
}
