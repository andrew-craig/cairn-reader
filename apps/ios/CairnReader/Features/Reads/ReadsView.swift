import SwiftUI

/// The Reads tab: the store's rows (or search results) in `ArticleListView`, with search,
/// pull-to-refresh, full-swipe archive and the cached-data banner.
struct ReadsView: View {
    @State private var model: ReadsModel

    init(session: UserSession) {
        _model = State(initialValue: ReadsModel(session: session))
    }

    var body: some View {
        @Bindable var search = model.search
        ArticleListView(
            articles: model.articles, pending: model.pendingIDs, notice: model.notice,
            isLoadingMore: model.isSyncingMore, onLastRowAppear: { Task { await model.loadMore() } }
        ) { article in
            Button {
                Task { await model.archive(article) }
            } label: {
                Label("Archive", systemImage: "archivebox")
            }
            .tint(Color(.success))
        }
        .overlay { emptyState }
        .searchable(text: $search.query, prompt: "Search your reads")
        .refreshable { await model.refresh() }
        .navigationTitle("Reads")
        .navigationDestination(for: Route.self, destination: RouteDestination.init)
        .task(id: model.window) { await model.observe(model.window) }
        // Not `.task`: leaving the tab mid-sync would cancel the wait and read as a failure.
        .onAppear {
            Task { await model.appear() }
        }
        .alert(
            "Error", isPresented: Binding(get: { search.alertMessage != nil }, set: { if !$0 { search.alertMessage = nil } })
        ) {
            Button("OK") {}
        } message: {
            Text(search.alertMessage ?? "")
        }
    }

    @ViewBuilder
    private var emptyState: some View {
        if model.articles.isEmpty {
            if model.search.isShowingResults {
                ContentUnavailableView.search(text: model.search.query)
            } else if model.isLoading {
                ProgressView()
            } else if model.showsError {
                ContentUnavailableView {
                    Label("Couldn't load your reading list", systemImage: "wifi.slash")
                } description: {
                    Text("Check your connection and try again.")
                } actions: {
                    Button("Retry") {
                        Task { await model.refresh() }
                    }
                }
            } else {
                ContentUnavailableView("No saved articles yet", systemImage: "book")
            }
        }
    }
}
