import Foundation
import Testing

/// Waits until `condition` holds, checking every few milliseconds; records an issue if it does
/// not within `timeout`. For state a model changes from its own tasks (debounced search, the
/// store observation). The timeout is generous because the whole suite runs in parallel on the
/// main actor, so a model's task can wait seconds for its turn on a loaded CI runner; a condition
/// that holds returns at once.
@MainActor
func eventually(
    timeout: Duration = .seconds(10), sourceLocation: SourceLocation = #_sourceLocation, _ condition: () -> Bool
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
