---
id: feature_33bc
title: [iOS native 10] Article list model, ArticleRow and the Reads tab
type: feature
status: in_progress
priority: 1
labels: [ios,ui]
blocked_by: [task_4e17]
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-15T08:04:08Z
---
## Goal
The reusable list pieces (network paging model, row, list view) and the Reads tab, which shows the store and nothing else (decision 8).

## Source
`apps/mobile/src/hooks/useCursorArticleList.ts`, `components/ArticleListScreen.tsx`, `components/common/{ArticleRow,SwipeAction,HeaderSearch,ScreenHeader,TopBlurGradient}.tsx`, `screens/ReadsScreen.tsx`; tests `useCursorArticleList.test.ts`, `ArticleListScreen.test.tsx`, `ReadsScreen.test.tsx`.

## Native mapping
| RN | Native |
|----|--------|
| `FlatList` + manual paddings | `List` (plain style) |
| `HeaderSearch` pill morph | `.searchable` |
| `SwipeAction` (gesture-handler) | `.swipeActions(edge:allowsFullSwipe:)` |
| pull-to-refresh control | `.refreshable` |
| `ScreenHeader`, `TopBlurGradient` | `navigationTitle` + toolbar |
| empty / error text blocks | `ContentUnavailableView` |
| add button in header | toolbar `Button` (opens the add-link sheet once that task lands) |
| screen state merged from store, server page and queued saves | an observation of the store |

## Plan
- [x] `ArticleListModel` (`@Observable`), for network lists (Feed, Bookmarks, and search in every tab): cursor pagination with load-more when the last row appears, refresh, debounced live search (`searchUserContents`) that replaces the list while a query is held and restores plus quietly refreshes the pre-search list when cleared, an `exclude` id set, stale-response protection (a slow earlier response must not overwrite a newer one), and `article(after:)` for the reader's "next article".
- [x] `ArticleRow`: title, source/domain, relative date, reading time, thumbnail via `AsyncImage`, optional `pending` note. A plain view used as `List` row content. The title is muted (`textSecondary`) for a partially read article, using the same rule as `isPartiallyRead` in `@cairn/shared` (added in #452); port that predicate with its tests.
- [x] `ArticleListView`: generic list taking rows, paging callbacks and row swipe configuration; reused by Reads, Feed and Bookmarks.
- [x] `ReadsModel` (`@Observable`): rows come from `ArticleStore.observeReads`. The window is the newest 100 stored rows until the first page has synced, then everything down to the oldest synced page's last `addedAt`. Appearing, pull-to-refresh and returning after the 30 s TTL call `Sync.syncReads(cursor: nil)`; the last row appearing syncs the next page. It keeps no copy of the articles. Also provides `article(after:)`.
- [x] A failed sync leaves the rows as they are and shows "Showing cached data"; with no rows it is the error state with Retry.
- [x] Rows with `pendingSave` show "Waiting to save — will sync when online". Their position needs no special handling: `saveToReads` stamps `addedAt` now, so they sort to the top.
- [x] Search uses `ArticleListModel` (`list=reads`) and replaces the stored rows while a query is held.
- [x] Trailing full-swipe calls `ArticleMutations.archive`. The row leaves because the store row is gone; there is no put-back or alert code in the view (a rejection is handled at the root, task 09).
- [x] Row tap pushes the article route with the `Article`. The reader gets "next article" from the model that presented it.

## Verify
- `useCursorArticleList.test.ts` ported against `ArticleListModel`.
- `ReadsScreen.test.tsx` re-expressed against `ReadsModel` with an in-memory store: stored rows before the first sync, cached banner on a failed sync, pending note follows the outbox, an article archived elsewhere disappears after a sync, window grows with paging. `ReadsScreen.storeFailure.test.tsx` does not port (decision_adb1).
- Owner: scroll through paging, search, pull to refresh, swipe to archive, airplane mode shows cached list.

## Review
Implemented on `claude/fervent-brown-x3gbf5`; not built locally (Linux, no Swift toolchain). To be verified by the iOS App Checks workflow.

What changed:
- `Features/ArticleList/ArticleListModel.swift` (new): `@Observable` network list and search. `refresh()` (first page, or reruns the search while a query is held), `loadMore()` (guards: cursor and `hasMore`, not while loading more, during a reset or while searching), `query` (bound to `.searchable`; a `didSet` debounces 300 ms, then searches), `exclude` (applied when `articles` is read, so a row hides and comes back as the set changes), `isLoading` (until the first reset finishes), `resetFailed` (the screen's stale banner or error state), `alertMessage` (failed search or load-more, RN wording), `remove(_:)`, `article(after:)`. The pages and the search results are held separately, so the RN pre-search snapshot is gone: clearing the query drops the results (the list shows at once) and starts a quiet reset that keeps the cursor until its page lands. Stale responses: a reset bumps a generation that drops earlier page responses (including a load-more in flight); every search and query change bumps a search generation that drops earlier search responses. Built from closures (`FetchPage`, `Search`); `init(read:list:fetchPage:)` searches through `ReadService.searchUserContents(list:limit: 20)`; `fetchPage: nil` is a search-only model (Reads).
- `Features/ArticleList/ArticleRow.swift` (new): title (bold/text unread, bold/`textSecondary` started, regular/`textSecondary` read), details line "author (else domain) · relative published date · N min", pending note in `info`, 48 pt `AsyncImage` thumbnail. `Models/Article.swift`: `Article.isPartiallyRead`, the `@cairn/shared` rule over `scrollPosition` (the store's value is the local one, so RN's separate `scrollFraction` has no counterpart).
- `Features/ArticleList/ArticleListView.swift` (new): plain `List` with an optional notice row, `NavigationLink(value: Route.article(article))` rows, `.swipeActions(edge:allowsFullSwipe: true)` from a row-actions builder, load-more on the last row's `onAppear`, a progress footer. Empty/loading/error stay with the screen (overlay), as do `.searchable` and `.refreshable`.
- `Features/Reads/ReadsModel.swift`, `ReadsView.swift` (new; `ReadsPlaceholder` removed, `MainTabs` shows `ReadsView`): the window (`newest(100)` until the first page syncs, then `addedSince(cursor.last)`, `addedSince(nil)` after the last page) is followed by `observe(window)` in `.task(id: window)`. `appear()` (30 s TTL, skipped while a query is held), `refresh()` (pull-to-refresh and Retry; reruns the search while a query is held), `loadMore()` (next page through `Sync.syncReads(after:)`; nothing before the first sync or after the last page); a first-page sync bumps a generation so a load-more started before it is not applied. A failed sync keeps the rows and shows "Showing cached data — pull to refresh", or the error state with Retry when there are none; a store observation failure takes the same path. Swipe archive: `ArticleMutations.archive` (and `search.remove` for a search result; not `exclude`, which would hide a refused archive from search for the session); no put-back or alert.
- `Store/ArticleStore.swift`: `observeReads(limit:addedSince:)` (both optional) replaces `observeReads(limit:)`, for the synced window.

Tests: `ArticleListModelTests` (useCursorArticleList.test.ts ported, plus slow-reset, reset-drops-load-more, search-only model and `article(after:)`), `ReadsModelTests` (stored rows before the first sync, cached banner, error state and Retry, pending note follows the outbox, archived elsewhere disappears after a sync, window grows with paging, TTL, archive through the outbox, search replaces the rows, `article(after:)`), `ArticleRowTests` (readProgress.test.ts ported, details line), `ArticleStoreTests` window case. New `Support/AsyncTestSupport.swift` (`eventually`) and a `skip:` parameter on `SyncFixtures.readsPage` for later pages.

Dropped or re-expressed jest cases:
- useCursorArticleList "calls onResetLoaded … onLoadError": no callbacks; a failed reset is `resetFailed`, a failed load-more `alertMessage`. Feed (task 11) caches after `await refresh()` when `!resetFailed`.
- "clears the refreshing indicator after a pull-to-refresh during search" and "a stale refresh response leaves the spinner for the newer search": the spinner is `.refreshable`'s and lasts as long as `refresh()`; re-expressed as "a refresh during search reruns the search" and "a search response for an earlier query is dropped".
- "a list reload during search updates what search restores": there is no direct reset during search (a refresh reruns the search), so it is a reset that was in flight when the results landed.
- "a pull-to-refresh inside the debounce": a refresh now cancels the pending debounce rather than letting it search again; the restore assertion is kept.
- readProgress "prefers the local scrollFraction over the server scrollPosition": no separate local field in the native `Article`.
- ReadsScreen "triggers SyncTrigger.run() after upsertMany" and "renders stored articles before the network call resolves" (network half): the order is `Sync.syncReads`'s (covered by SyncTests); the stored-rows half is kept. The queued-save cases become the pending-note test: there is no merge of queued saves above the page, the store row sorts to the top by `addedAt`.
- ReadsScreen.storeFailure.test.tsx does not port (decision_adb1). ArticleListScreen.test.tsx (spinner/error/empty rendering) is view layout, not ported; the states it covers are `ReadsModel.isLoading`/`showsError`/`notice`.

Notes and follow-ups:
- No add-link toolbar button: the sheet is task 14, and a button that opens nothing would be dead. Task 14 adds it to `ReadsView` (`.toolbar`) and refreshes on success.
- The article route already existed (`Route.article(Article)`, placeholder in `RouteDestination`); rows push it. Task 12 replaces the placeholder and wires "next article" from the presenting model's `article(after:)`.
- An archive from the reader of a search result leaves the result row until the search reruns (only `ReadsModel.archive` calls `search.remove`); store rows need nothing.
- A first-page sync (appear after the TTL, pull-to-refresh) narrows the window back to the first page, as the RN reset did.
- Timing-based tests: `ReadsModelTests` and the search tests poll with `eventually` (2 s timeout); `aRefreshInsideTheDebounceStillRestoresTheList` sleeps 50 ms past a 20 ms debounce.
- CI: iOS App Checks green on PR #468 (3451253). Close after merge and on-device check.
