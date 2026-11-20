import Network
import Observation

/// Whether the device has a network path, for the UI and for `Sync`. Watches `NWPathMonitor`
/// for the life of the app. Online until the monitor reports a path: an unknown state never
/// blocks anything.
@Observable
final class Connectivity {
    private(set) var isOnline = true

    @ObservationIgnored private var monitoring: Task<Void, Never>?

    init() {
        let monitor = NWPathMonitor()
        monitoring = Task { [weak self] in
            for await path in monitor {
                self?.isOnline = Connectivity.isOnline(path.status)
            }
        }
    }

    deinit {
        monitoring?.cancel()
    }

    /// Whether a path with `status` counts as online.
    nonisolated static func isOnline(_ status: NWPath.Status) -> Bool {
        status == .satisfied
    }
}
