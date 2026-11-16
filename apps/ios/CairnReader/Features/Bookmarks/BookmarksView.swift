import SwiftUI

/// Bookmarks, pushed from You: the favourites (or search results) in `ArticleListView`, with
/// search and pull-to-refresh, and no swipe action (as in the React Native screen).
struct BookmarksView: View {
    @State private var model: BookmarksModel

    init(session: UserSession) {
        _model = State(initialValue: BookmarksModel(session: session))
    }

    var body: some View {
        @Bindable var list = model.list
        ArticleListView(
            articles: list.articles, notice: model.notice, isLoadingMore: list.isLoadingMore,
            onLastRowAppear: { Task { await list.loadMore() } }
        ) { _ in
            EmptyView()
        }
        .overlay { emptyState }
        .searchable(text: $list.query, prompt: "Search your bookmarks")
        .refreshable { await list.refresh() }
        .navigationTitle("Bookmarks")
        // Not `.task`: popping back mid-refresh would cancel it and read as a failure.
        .onAppear {
            Task { await model.appear() }
        }
        .alert(
            "Error", isPresented: Binding(get: { list.alertMessage != nil }, set: { if !$0 { list.alertMessage = nil } })
        ) {
            Button("OK") {}
        } message: {
            Text(list.alertMessage ?? "")
        }
    }

    @ViewBuilder
    private var emptyState: some View {
        if model.list.articles.isEmpty {
            if model.list.isShowingResults {
                ContentUnavailableView.search(text: model.list.query)
            } else if model.list.isLoading {
                ProgressView()
            } else if model.showsError {
                ContentUnavailableView {
                    Label("Couldn't load your bookmarks", systemImage: "wifi.slash")
                } description: {
                    Text("Check your connection and try again.")
                } actions: {
                    Button("Retry") {
                        Task { await model.list.refresh() }
                    }
                }
            } else {
                ContentUnavailableView("No bookmarked articles yet", image: "bookmark")
            }
        }
    }
}
