import SwiftUI

extension View {
    /// The list header search of Reads, Feed and Bookmarks, bound to the list's `query`.
    func headerSearch(query: Binding<String>, prompt: String) -> some View {
        modifier(HeaderSearch(query: query, prompt: prompt))
    }
}

/// A search button in the navigation bar opens a field in the bar's place, as the system apps
/// do: the bar (title and buttons) hides and the field takes its row. A bar item cannot host it
/// (it keeps its first width, and a custom view holding an image is shown as that image), so the
/// field is an inset at the top of the safe area. Not a `safeAreaBar`: its content never reports
/// focus to `@FocusState`. Closing clears `query` and brings the bar back.
private struct HeaderSearch: ViewModifier {
    @Binding var query: String
    let prompt: String

    @State private var isOpen = false

    func body(content: Content) -> some View {
        content
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Search", image: .search) { isOpen = true }
                }
            }
            .toolbar(isOpen ? .hidden : .automatic, for: .navigationBar)
            .safeAreaInset(edge: .top) {
                if isOpen {
                    Field(query: $query, prompt: prompt) { isOpen = false }
                        .padding(.horizontal)
                        .padding(.vertical, 5)
                }
            }
    }
}

/// A glass capsule holding the search icon and the focused text field, with a glass close circle
/// to its right. Opening, the capsule stretches leftwards from where the search button was and the
/// close circle and field fade in; closing clears `query` and plays it in reverse, faster. Reduce
/// Motion snaps the size and fades the contents.
private struct Field: View {
    @Binding var query: String
    let prompt: String
    let onClosed: () -> Void

    @State private var expanded = false
    @State private var showsContents = false
    @FocusState private var isFocused: Bool
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    private static let size: CGFloat = 44

    var body: some View {
        HStack(spacing: 8) {
            HStack(spacing: 0) {
                Image(.search)
                    .foregroundStyle(Color(.textSecondary))
                    .frame(width: Self.size, height: Self.size)
                    .accessibilityHidden(true)

                if showsContents {
                    TextField(prompt, text: $query)
                        .focused($isFocused)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                        .submitLabel(.search)
                        .foregroundStyle(Color(.text))
                        .padding(.trailing)
                        .onAppear { isFocused = true }
                        .transition(.opacity)
                }
            }
            .frame(maxWidth: expanded ? .infinity : Self.size, minHeight: Self.size, maxHeight: Self.size, alignment: .leading)
            .glassEffect(in: Capsule())
            .frame(maxWidth: .infinity, alignment: .trailing)

            // Out of the layout until shown, so the collapsed capsule sits where the search button was.
            if showsContents {
                Button("Close search", image: .close, action: close)
                    .labelStyle(.iconOnly)
                    .foregroundStyle(Color(.text))
                    .frame(width: Self.size, height: Self.size)
                    .contentShape(Circle())
                    .glassEffect(.regular.interactive(), in: Circle())
                    .transition(.opacity.combined(with: .scale(scale: 0.6)))
            }
        }
        .onAppear(perform: open)
    }

    private func open() {
        if reduceMotion {
            expanded = true
            withAnimation(.easeOut(duration: 0.15)) { showsContents = true }
        } else {
            withAnimation(.timingCurve(0.77, 0, 0.175, 1, duration: 0.26)) { expanded = true }
            withAnimation(.easeOut(duration: 0.2).delay(0.16)) { showsContents = true }
        }
    }

    private func close() {
        query = ""
        isFocused = false
        if reduceMotion {
            expanded = false
            withAnimation(.easeOut(duration: 0.15)) { showsContents = false } completion: { onClosed() }
        } else {
            withAnimation(.easeOut(duration: 0.1)) { showsContents = false }
            withAnimation(.timingCurve(0.23, 1, 0.32, 1, duration: 0.22).delay(0.04)) { expanded = false } completion: {
                onClosed()
            }
        }
    }
}

#Preview {
    @Previewable @State var query = ""
    NavigationStack {
        List(1...20, id: \.self) { Text("Article \($0)") }
            .navigationTitle("Reads")
            .toolbarTitleDisplayMode(.inlineLarge)
            .headerSearch(query: $query, prompt: "Search your reads")
    }
}
