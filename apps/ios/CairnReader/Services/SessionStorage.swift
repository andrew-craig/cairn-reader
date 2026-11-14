import Foundation
import OSLog
import Security

/// The signed-in session: the token pair, when the access token expires, and the user.
nonisolated struct StoredSession: Codable, Sendable, Equatable {
    var accessToken: String
    var refreshToken: String
    var expiresAt: Date
    var user: User
}

/// Persists the session across launches.
nonisolated protocol SessionStorage: Sendable {
    /// The saved session, or nil when there is none or it cannot be read.
    func load() -> StoredSession?
    func save(_ session: StoredSession)
    func clear()
}

/// Keeps the session as one Keychain generic-password item, so the token pair, expiry and user
/// are always written together.
///
/// `AfterFirstUnlock` keeps the item readable while the device is locked, so a sync that starts
/// in the background (or as the app returns to the foreground) still has a session.
nonisolated struct KeychainSessionStorage: SessionStorage {
    private let service: String
    private static let account = "session"
    private static let logger = Logger(subsystem: "com.cairnapp.cairnreader", category: "auth")

    init(service: String = "com.cairnapp.cairnreader.auth") {
        self.service = service
    }

    private var query: [String: Any] {
        [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: Self.account,
        ]
    }

    func load() -> StoredSession? {
        var query = query
        query[kSecReturnData as String] = true
        query[kSecMatchLimit as String] = kSecMatchLimitOne
        var result: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &result)
        guard status == errSecSuccess, let data = result as? Data else {
            if status != errSecItemNotFound {
                Self.logger.error("Reading the session from the Keychain failed: \(status)")
            }
            return nil
        }
        do {
            return try JSONDecoder().decode(StoredSession.self, from: data)
        } catch {
            Self.logger.error("The saved session could not be decoded")
            return nil
        }
    }

    func save(_ session: StoredSession) {
        guard let data = try? JSONEncoder().encode(session) else {
            Self.logger.error("Encoding the session failed")
            return
        }
        let attributes: [String: Any] = [
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlock,
        ]
        var status = SecItemUpdate(query as CFDictionary, attributes as CFDictionary)
        if status == errSecItemNotFound {
            status = SecItemAdd(query.merging(attributes) { $1 } as CFDictionary, nil)
        }
        if status != errSecSuccess {
            Self.logger.error("Saving the session to the Keychain failed: \(status)")
        }
    }

    func clear() {
        let status = SecItemDelete(query as CFDictionary)
        if status != errSecSuccess && status != errSecItemNotFound {
            Self.logger.error("Deleting the session from the Keychain failed: \(status)")
        }
    }
}
