import Foundation

/// The App Group shared with a future share extension. The database, the Keychain
/// session and the saved server URL live here so the extension can reach them.
nonisolated enum AppGroup {
    static let identifier = "group.com.cairnapp.cairnreader"

    /// The group's shared container directory.
    static let containerURL: URL = {
        guard let url = FileManager.default.containerURL(
            forSecurityApplicationGroupIdentifier: identifier)
        else {
            preconditionFailure("The \(identifier) entitlement is missing")
        }
        return url
    }()
}
