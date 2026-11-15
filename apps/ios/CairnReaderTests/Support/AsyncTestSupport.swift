import Foundation
import Testing

/// Waits until `condition` holds, checking every few milliseconds; records an issue if it does
/// not within `timeout`. For state a model changes from its own tasks (debounced search, the
/// store observation).
@MainActor
func eventually(
    timeout: Duration = .seconds(2), sourceLocation: SourceLocation = #_sourceLocation, _ condition: () -> Bool
) async {
    let deadline = ContinuousClock.now + timeout
    while !condition() {
        guard ContinuousClock.now < deadline else {
            Issue.record("Condition not met within \(timeout)", sourceLocation: sourceLocation)
            return
        }
        try? await Task.sleep(for: .milliseconds(5))
    }
}
