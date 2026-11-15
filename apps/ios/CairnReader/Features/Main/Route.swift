/// Every destination a tab's `NavigationStack` can push, typed in one place. The article carries
/// the `Article` value only, never a list or an index (decision 8).
enum Route: Hashable {
    case article(Article)
    case bookmarks
    case account
    case about
    case rss
    case newsletters
}
