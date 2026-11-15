import Foundation
import Observation
import OSLog

/// Bookmarks, pushed from You: the user's favourites from every list (`is_favorite=true` pages)
/// in an `ArticleListModel`, searching every list as the React Native screen does. Stored
/// favourites show while nothing is listed yet, so a failed refresh is not a blank screen. It
/// never writes the store: `Sync` is the only writer of Reads pages (decision 8), and favourited
/// Feed items do not belong there.
@Observable
final class BookmarksModel {
    let list: ArticleListModel

    @ObservationIgnored private let articles: ArticleStore
    @ObservationIgnored private let logger = Logger(subsystem: "com.cairnapp.cairnreader", category: "bookmarks")

    init(session: UserSession) {
        articles = session.articles
        let read = session.read
        list = ArticleListModel(read: read, list: nil) { cursor in
            try await read.listUserContents(isFavorite: true, limit: ArticleListModel.pageSize, cursor: cursor)
        }
    }

    /// The banner over stored favourites or the last page after a failed refresh.
    var notice: String? {
        list.resetFailed && !list.articles.isEmpty && !list.isShowingResults
            ? "Showing cached data — pull to refresh" : nil
    }

    /// The error state with Retry: a failed refresh with nothing to show.
    var showsError: Bool {
        list.resetFailed && list.articles.isEmpty && !list.isShowingResults
    }

    /// The row after `id` in what the list shows, for the reader's next article.
    func article(after id: String) -> Article? {
        list.article(after: id)
    }

    /// Refreshes on every appearance (so a favourite removed in the reader is gone on return)
    /// unless a query is held; with nothing listed yet, the stored favourites show first.
    func appear() async {
        guard list.searchQuery == nil else { return }
        if list.articles.isEmpty {
            do {
                let stored = try await articles.listFavorites()
                if !stored.isEmpty, list.articles.isEmpty { list.prime(stored) }
            } catch {
                logger.error("Loading stored favourites failed: \(String(describing: error))")
            }
        }
        await list.refresh()
    }
}
