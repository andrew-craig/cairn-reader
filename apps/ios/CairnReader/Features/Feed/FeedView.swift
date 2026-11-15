import SwiftUI

/// The Feed tab: `list=feed` pages (or search results) in `ArticleListView`, with search,
/// pull-to-refresh, full-swipe Save to Reads and the cached-data banner.
struct FeedView: View {
    @State private var model: FeedModel

    init(session: UserSession) {
        _model = State(initialValue: FeedModel(session: session))
    }

    var body: some View {
        @Bindable var list = model.list
        ArticleListView(
            articles: list.articles, notice: model.notice, isLoadingMore: list.isLoadingMore, swipeEdge: .leading,
            onLastRowAppear: { Task { await list.loadMore() } }
        ) { article in
            Button {
                Task { await model.saveToReads(article) }
            } label: {
                Label("Save to Reads", systemImage: "book")
            }
            .tint(Color(.info))
        }
        .overlay { emptyState }
        .searchable(text: $list.query, prompt: "Search your feed")
        .refreshable { await list.refresh() }
        .navigationTitle("Feed")
        .navigationDestination(for: Route.self, destination: RouteDestination.init)
        .task { await model.observeSaved() }
        // Not `.task`: leaving the tab mid-refresh would cancel it and read as a failure.
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
                    Label("Couldn't load your Feed", systemImage: "wifi.slash")
                } description: {
                    Text("Check your connection and try again.")
                } actions: {
                    Button("Retry") {
                        Task { await model.list.refresh() }
                    }
                }
            } else {
                ContentUnavailableView("Your Feed is empty", systemImage: "newspaper")
            }
        }
    }
}
