import Foundation

/// The backend the app talks to: `defaultURL` unless the user saved another server, which is
/// kept in `UserDefaults`.
nonisolated struct ServerConfig: Sendable {
    static let defaultURL = URL(string: "https://cairn.seatrain.net")!
    private static let key = "serverURL"

    // UserDefaults is documented as thread-safe but not marked Sendable.
    nonisolated(unsafe) private let defaults: UserDefaults

    init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
    }

    var url: URL {
        defaults.string(forKey: Self.key).flatMap(URL.init(string:)) ?? Self.defaultURL
    }

    /// Saves `input` (normalized: trimmed, trailing slashes dropped, `https://` added when there
    /// is no scheme; empty means the default). Returns whether the server changed.
    ///
    /// Tokens belong to the server that issued them, so a `true` result means the caller must
    /// clear the session. Settings change the server through `AuthService`, which does that.
    func setURL(_ input: String) -> Bool {
        let newURL = Self.normalize(input)
        let changed = newURL != url
        if newURL == Self.defaultURL {
            defaults.removeObject(forKey: Self.key)
        } else {
            defaults.set(newURL.absoluteString, forKey: Self.key)
        }
        return changed
    }

    static func normalize(_ input: String) -> URL {
        var text = input.trimmingCharacters(in: .whitespacesAndNewlines)
        while text.hasSuffix("/") { text.removeLast() }
        if text.range(of: "^https?://", options: [.regularExpression, .caseInsensitive]) == nil {
            // Drop a partial scheme such as "https:" or "http:/".
            let host = text.replacingOccurrences(
                of: "^https?:/*", with: "", options: [.regularExpression, .caseInsensitive])
            text = host.isEmpty ? "" : "https://\(host)"
        }
        return text.isEmpty ? defaultURL : URL(string: text) ?? defaultURL
    }
}
