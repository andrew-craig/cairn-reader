import Network
import Observation

/// Whether the device has a network path, for the UI. Watches `NWPathMonitor` for the life of
/// the app.
@Observable
final class Connectivity {
    private(set) var isOnline = true

    @ObservationIgnored private var monitoring: Task<Void, Never>?

    init() {
        let monitor = NWPathMonitor()
        monitoring = Task { [weak self] in
            for await path in monitor {
                self?.isOnline = path.status == .satisfied
            }
        }
    }

    deinit {
        monitoring?.cancel()
    }
}
