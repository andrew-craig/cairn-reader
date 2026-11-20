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

/// Body of `POST /auth/login` and `POST /auth/register`.
nonisolated struct EmailCredentials: Encodable, Sendable {
    let email: String
    let password: String
}

/// Body of `POST /auth/login/mobile` and `POST /auth/register/mobile`. The field keeps its Expo
/// name on the wire (`expo_device_id`); the iOS value is `identifierForVendor`.
nonisolated struct DeviceCredentials: Encodable, Sendable {
    let expoDeviceId: String
}

/// Body of `POST /auth/refresh` and `POST /auth/logout`.
nonisolated struct RefreshTokenRequest: Encodable, Sendable {
    let refreshToken: String
}

/// Body of `PUT /user/{id}/password`.
nonisolated struct ChangePasswordRequest: Encodable, Sendable {
    let currentPassword: String
    let newPassword: String
}
