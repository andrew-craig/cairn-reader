import SwiftUI

/// "You're offline", shown above everything while the device has no connection. One of the few
/// custom views (epic_bc48): the system has no equivalent status bar.
struct OfflineBanner: View {
    var body: some View {
        Text("You\u{2019}re offline")
            .font(Typography.subheadline)
            .foregroundStyle(Color(.background))
            .frame(maxWidth: .infinity)
            .padding(.vertical, 4)
            .background(Color(.warning), ignoresSafeAreaEdges: .top)
    }
}
