import SwiftUI

/// The Reads tab: the store's rows (or search results) in `ArticleListView`, with search,
/// pull-to-refresh, full-swipe archive, the cached-data banner and the add-link sheet.
struct ReadsView: View {
    private let session: UserSession
    @State private var model: ReadsModel
    @State private var isAddingLink = false

    init(session: UserSession) {
        self.session = session
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
                Label("Archive", image: .archive)
            }
            .tint(Color(.success))
        }
        .overlay { emptyState }
        .headerSearch(query: $search.query, prompt: "Search your reads")
        .refreshable { await model.refresh() }
        .navigationTitle("Reads")
        .toolbarTitleDisplayMode(.inlineLarge)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("Add link", systemImage: "plus") { isAddingLink = true }
            }
        }
        .sheet(isPresented: $isAddingLink) {
            AddLinkSheet(read: session.read, mode: .link) {
                Task { await model.refresh() }
            }
        }
        .navigationDestination(for: Route.self) { RouteDestination($0, nextArticle: model.article(after:)) }
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
                EmptyState(title: "No saved articles yet", image: .read)
            }
        }
    }
}
