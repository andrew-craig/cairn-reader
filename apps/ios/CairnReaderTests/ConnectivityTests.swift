import Network
import Testing
@testable import CairnReader

// Port of apps/mobile/src/utils/network.test.ts and hooks/useNetworkStatus.test.ts.

@MainActor
struct ConnectivityTests {
    @Test func aSatisfiedPathIsOnline() {
        #expect(Connectivity.isOnline(.satisfied))
    }

    @Test func anUnsatisfiedPathIsOffline() {
        #expect(!Connectivity.isOnline(.unsatisfied))
    }

    /// Before the monitor reports a path the state is unknown, which counts as online.
    @Test func anUnknownStateIsOnline() {
        #expect(Connectivity().isOnline)
    }
}
