import SwiftUI

/// A list's empty state with one of the app's icons. `ContentUnavailableView(_:image:)` draws an
/// asset at its own 24 pt, where it scales an SF Symbol up to the large title.
struct EmptyState: View {
    let title: LocalizedStringKey
    let image: ImageResource

    @ScaledMetric(relativeTo: .largeTitle) private var size: CGFloat = 48

    var body: some View {
        ContentUnavailableView {
            Label {
                Text(title)
            } icon: {
                Image(image)
                    .resizable()
                    .scaledToFit()
                    .frame(width: size, height: size)
            }
        }
    }
}
