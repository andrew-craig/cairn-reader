import SwiftUI

/// The plain article `List` of Reads, Feed and Bookmarks: an optional notice row, one
/// `ArticleRow` per article pushing `Route.article`, full-swipe row actions on one edge, and
/// paging when the last row appears. Empty, loading and error states are the screen's
/// (`ContentUnavailableView` overlays), as are `.searchable` and `.refreshable`.
struct ArticleListView<RowActions: View>: View {
    let articles: [Article]
    /// Ids whose row shows the pending-save note.
    var pending: Set<String> = []
    /// A line above the rows, e.g. that the list is cached.
    var notice: String?
    var isLoadingMore = false
    var swipeEdge = HorizontalEdge.trailing
    let onLastRowAppear: () -> Void
    @ViewBuilder let rowActions: (Article) -> RowActions

    var body: some View {
        List {
            if let notice {
                Text(notice)
                    .font(Typography.caption)
                    .foregroundStyle(Color(.textSecondary))
                    .listRowSeparator(.hidden)
            }
            ForEach(articles) { article in
                NavigationLink(value: Route.article(article)) {
                    ArticleRow(article: article, pending: pending.contains(article.id))
                }
                .swipeActions(edge: swipeEdge, allowsFullSwipe: true) {
                    rowActions(article)
                }
                .onAppear {
                    if article.id == articles.last?.id { onLastRowAppear() }
                }
            }
            if isLoadingMore {
                ProgressView()
                    .frame(maxWidth: .infinity)
                    .listRowSeparator(.hidden)
            }
        }
        .listStyle(.plain)
    }
}
