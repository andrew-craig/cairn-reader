import Foundation

/// The request could not be completed because the server could not be reached, did not answer
/// in time, or sent a success body that could not be read — as opposed to `HttpError`, a
/// definitive answer from the server.
///
/// Keep the two apart: offline logic (the outbox, sign-out on a rejected session) treats
/// "couldn't reach the server" as retryable and never as "the server said no".
nonisolated enum NetworkError: Error, Equatable, LocalizedError {
    case unreachable
    case timeout
    case undecodableBody

    var errorDescription: String? {
        "Unable to reach the server. Please try again later."
    }
}

/// The server answered with a non-2xx status. `status` lets callers tell a definitive rejection
/// (4xx) from a transient one (401, 5xx) without reading the message.
nonisolated struct HttpError: Error, Equatable, LocalizedError {
    let status: Int
    /// The server's `message` (or `error` code) from the error body, when it sent one.
    let message: String?

    var errorDescription: String? {
        message ?? "The server returned an error (\(status))."
    }
}
