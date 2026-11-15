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
- [ ] `ArticleListModel` (`@Observable`), for network lists (Feed, Bookmarks, and search in every tab): cursor pagination with load-more when the last row appears, refresh, debounced live search (`searchUserContents`) that replaces the list while a query is held and restores plus quietly refreshes the pre-search list when cleared, an `exclude` id set, stale-response protection (a slow earlier response must not overwrite a newer one), and `article(after:)` for the reader's "next article".
- [ ] `ArticleRow`: title, source/domain, relative date, reading time, thumbnail via `AsyncImage`, optional `pending` note. A plain view used as `List` row content. The title is muted (`textSecondary`) for a partially read article, using the same rule as `isPartiallyRead` in `@cairn/shared` (added in #452); port that predicate with its tests.
- [ ] `ArticleListView`: generic list taking rows, paging callbacks and row swipe configuration; reused by Reads, Feed and Bookmarks.
- [ ] `ReadsModel` (`@Observable`): rows come from `ArticleStore.observeReads`. The window is the newest 100 stored rows until the first page has synced, then everything down to the oldest synced page's last `addedAt`. Appearing, pull-to-refresh and returning after the 30 s TTL call `Sync.syncReads(cursor: nil)`; the last row appearing syncs the next page. It keeps no copy of the articles. Also provides `article(after:)`.
- [ ] A failed sync leaves the rows as they are and shows "Showing cached data"; with no rows it is the error state with Retry.
- [ ] Rows with `pendingSave` show "Waiting to save — will sync when online". Their position needs no special handling: `saveToReads` stamps `addedAt` now, so they sort to the top.
- [ ] Search uses `ArticleListModel` (`list=reads`) and replaces the stored rows while a query is held.
- [ ] Trailing full-swipe calls `ArticleMutations.archive`. The row leaves because the store row is gone; there is no put-back or alert code in the view (a rejection is handled at the root, task 09).
- [ ] Row tap pushes the article route with the `Article`. The reader gets "next article" from the model that presented it.

## Verify
- `useCursorArticleList.test.ts` ported against `ArticleListModel`.
- `ReadsScreen.test.tsx` re-expressed against `ReadsModel` with an in-memory store: stored rows before the first sync, cached banner on a failed sync, pending note follows the outbox, an article archived elsewhere disappears after a sync, window grows with paging. `ReadsScreen.storeFailure.test.tsx` does not port (decision_adb1).
- Owner: scroll through paging, search, pull to refresh, swipe to archive, airplane mode shows cached list.
