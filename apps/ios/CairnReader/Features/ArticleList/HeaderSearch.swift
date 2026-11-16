import SwiftUI

extension View {
    /// The list header search of Reads, Feed and Bookmarks, bound to the list's `query`.
    func headerSearch(query: Binding<String>, prompt: String) -> some View {
        modifier(HeaderSearch(query: query, prompt: prompt))
    }
}

/// Ported from the React Native `HeaderSearch` (which mirrors the web list header): a round
/// search button in the navigation bar opens a pill field. A bar item cannot host it (it keeps
/// its first width, and a custom view holding an image is shown as that image), so the field is an
/// inset beneath the navigation bar, and the bar button hides while it is open. Not a
/// `safeAreaBar`: its content never reports focus to `@FocusState`, so the ring stayed unfocused.
private struct HeaderSearch: ViewModifier {
    @Binding var query: String
    let prompt: String

    @State private var isOpen = false

    func body(content: Content) -> some View {
        content
            .toolbar {
                if !isOpen {
                    ToolbarItem(placement: .topBarTrailing) {
                        Button("Search", image: .search) { isOpen = true }
                    }
                }
            }
            .safeAreaInset(edge: .top) {
                if isOpen {
                    Pill(query: $query, prompt: prompt) { isOpen = false }
                        .padding(.horizontal)
                        .padding(.bottom, 8)
                }
            }
    }
}

/// The field, timed as `HeaderSearch.tsx`: the circle rings, then stretches leftwards, the icon
/// riding its leading edge, and the focused field and a close button (where the icon was) fade
/// in; closing clears `query` and plays it in reverse, faster. Reduce Motion snaps the size and
/// fades the contents.
private struct Pill: View {
    @Binding var query: String
    let prompt: String
    let onClosed: () -> Void

    @State private var ringed = false
    @State private var expanded = false
    @State private var showsContents = false
    @FocusState private var isFocused: Bool
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    private static let size: CGFloat = 44

    var body: some View {
        ZStack {
            Image(.search)
                .foregroundStyle(Color(.textSecondary))
                .frame(width: Self.size, height: Self.size)
                .frame(maxWidth: .infinity, alignment: .leading)
                .accessibilityHidden(true)

            if showsContents {
                TextField(prompt, text: $query)
                    .focused($isFocused)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .submitLabel(.search)
                    .foregroundStyle(Color(.text))
                    .padding(.horizontal, Self.size)
                    .onAppear { isFocused = true }
                    .transition(.opacity)

                Button("Close search", image: .close, action: close)
                    .labelStyle(.iconOnly)
                    .foregroundStyle(Color(.text))
                    .frame(width: Self.size, height: Self.size)
                    .contentShape(Rectangle())
                    .frame(maxWidth: .infinity, alignment: .trailing)
                    .transition(.opacity.combined(with: .scale(scale: 0.6)))
            }
        }
        .frame(maxWidth: expanded ? .infinity : Self.size, minHeight: Self.size, maxHeight: Self.size)
        .glassEffect(in: Capsule())
        .overlay {
            Capsule()
                .strokeBorder(Color(isFocused ? .primary : .textSecondary))
                .opacity(ringed ? 1 : 0)
        }
        .frame(maxWidth: .infinity, alignment: .trailing)
        .onAppear(perform: open)
    }

    private func open() {
        if reduceMotion {
            ringed = true
            expanded = true
            withAnimation(.easeOut(duration: 0.15)) { showsContents = true }
        } else {
            withAnimation(.easeOut(duration: 0.12)) { ringed = true }
            withAnimation(.timingCurve(0.77, 0, 0.175, 1, duration: 0.26).delay(0.08)) { expanded = true }
            withAnimation(.easeOut(duration: 0.2).delay(0.24)) { showsContents = true }
        }
    }

    private func close() {
        query = ""
        isFocused = false
        if reduceMotion {
            ringed = false
            expanded = false
            withAnimation(.easeOut(duration: 0.15)) { showsContents = false } completion: { onClosed() }
        } else {
            withAnimation(.easeOut(duration: 0.1)) { showsContents = false }
            withAnimation(.timingCurve(0.23, 1, 0.32, 1, duration: 0.22).delay(0.04)) { expanded = false }
            withAnimation(.easeOut(duration: 0.12).delay(0.2)) { ringed = false } completion: { onClosed() }
        }
    }
}

#Preview {
    @Previewable @State var query = ""
    NavigationStack {
        List(1...20, id: \.self) { Text("Article \($0)") }
            .navigationTitle("Reads")
            .headerSearch(query: $query, prompt: "Search your reads")
    }
}
