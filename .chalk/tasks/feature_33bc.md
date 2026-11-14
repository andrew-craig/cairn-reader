---
id: feature_33bc
title: [iOS native 10] Article list model, ArticleRow and the Reads tab
type: feature
status: open
priority: 1
labels: [ios,ui]
blocked_by: [task_4e17,feature_0b9a]
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-14T02:35:53Z
---
## Goal
The reusable list pieces (paging model, row) and the offline-first Reads tab on top of them.

## Source
`apps/mobile/src/hooks/useCursorArticleList.ts`, `components/ArticleListScreen.tsx`, `components/common/{ArticleRow,SwipeAction,HeaderSearch,ScreenHeader,TopBlurGradient}.tsx`, `screens/ReadsScreen.tsx`; tests `useCursorArticleList.test.ts`, `ArticleListScreen.test.tsx`, `ReadsScreen.test.tsx`, `ReadsScreen.storeFailure.test.tsx`.

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

## Plan
- [ ] `ArticleListModel` (`@Observable`): cursor pagination with load-more when the last row appears, refresh, debounced live search (`searchUserContents`) that replaces the list while a query is held and restores plus quietly refreshes the pre-search list when cleared, an `exclude` id set, and stale-response protection (a slow earlier response must not overwrite a newer one).
- [ ] `ArticleRow`: title, source/domain, relative date, reading time, thumbnail via `AsyncImage`, optional `pending` note. A plain view used as `List` row content. The title is muted (`textSecondary`) for a partially read article, using the same rule as `isPartiallyRead` in `@cairn/shared` (added in #452); port that predicate with its tests.
- [ ] `ArticleListView`: generic list taking a model and row swipe configuration; reused by Feed and Bookmarks.
- [ ] Reads tab: render `ArticleStore.listRecent` first, then fetch `list=reads`; on fetch failure keep the stored rows and show "Showing cached data". Every list sync (including pull-to-refresh) calls `ArticleStore.upsertMany` then `SyncTrigger.run`.
- [ ] Queued saves (`ArticleStore.listPendingSaves`) listed above the server page until it includes them, with the "Waiting to save — will sync when online" note. Keep the snapshot rule: the merged set is taken before each first-page fetch; the pending indicator is re-read after every `SyncTrigger.run`.
- [ ] Trailing full-swipe archives through `ArticleMutations.archive`; if it rejects definitively the row returns and an alert is shown.
- [ ] Row tap pushes the reader route with the article, the current list and index (needed for "next article").

## Verify
- `useCursorArticleList.test.ts` ported against `ArticleListModel`; Reads view-model tests for store-first render, cached banner, store failure, pending saves merge.
- Owner: scroll through paging, search, pull to refresh, swipe to archive, airplane mode shows cached list.
