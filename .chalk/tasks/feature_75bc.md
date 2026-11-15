---
id: feature_75bc
title: [iOS native 11] Feed tab and Bookmarks
type: feature
status: in_progress
priority: 2
labels: [ios,ui]
blocked_by: [feature_33bc]
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-15T08:17:38Z
---
## Goal
The online-only skim list and the favourites list, both on `ArticleListView` and `ArticleListModel`.

## Source
`apps/mobile/src/screens/{FeedScreen,BookmarksScreen}.tsx`; tests `FeedScreen.test.tsx`, `BookmarksScreen.test.tsx`.

## Plan
- [x] Feed: `list=feed`. Pages are never written to `ArticleStore` and never prefetched. Keep the last first page in `FeedCache`; when a refresh fails show the cached page with a stale banner.
- [x] The model's `exclude` set is `ArticleStore.observeStoredIDs()`: the store only holds Reads, so an id in it has been saved (decision 8). This one rule hides a row the moment it is saved from the swipe or from the reader, keeps it hidden in later pages and search results while the save is queued, and brings it back if the save is rejected and the store row is deleted. No `pendingSaveIds`, no invalidation flag, no put-back code.
- [x] Leading full-swipe "Save to Reads" (`info` colour) calls `ArticleMutations.saveToReads`.
- [x] `FeedCache` is written from the list after exclusion, so a saved row is not in the cached page.
- [x] Bookmarks: favourites filter, search, pushed from You. No swipe action unless the RN screen has one (check and match).

## Verify
- `FeedScreen.test.tsx` re-expressed: an offline save hides the row and queues it; a rejected save (drain drops the `list` row) shows it again; stored ids are excluded from pages and search; stale banner on failed refresh; the cached page omits saved rows.
- `BookmarksScreen.test.tsx` ported.

## Review
Implemented on `task11`; not built locally (Linux, no Swift toolchain). To be verified by the iOS App Checks workflow.

What changed:
- `Features/Feed/FeedModel.swift`, `FeedView.swift` (new; `Features/Main/Placeholders.swift` removed, `MainTabs` shows `FeedView`): `FeedModel` wraps an `ArticleListModel` of `list=feed` pages (search `list=feed`); nothing reaches the store or prefetch. `observeSaved()` (the view's `.task`) assigns `ArticleStore.observeStoredIDs()` to `exclude`; a change also rewrites `FeedCache`, so a row saved from the swipe or the reader leaves the cached page. `appear()` (30 s TTL, skipped while a query is held) primes the list from `FeedCache` when nothing is listed, then refreshes. `notice` ("Showing cached data — pull to refresh") while the cached page shows or after a failed refresh; `showsError` (Retry) when that leaves nothing. `saveToReads` calls `ArticleMutations.saveToReads` (leading full swipe, `info` tint); a database failure is logged, a refusal is the root's alert. `article(after:)` delegates to the list.
- `Features/ArticleList/ArticleListModel.swift`: `onReset` (called after every first page that lands, including the quiet refresh after a search is cleared) and `prime(_:)` (sets the rows, ends `isLoading`). The cache is written from `onReset` and on an `exclude` change, from `articles` (after exclusion), never while results show.
- `Features/Bookmarks/BookmarksModel.swift`, `BookmarksView.swift` (new; `RouteDestination` shows it for `.bookmarks`, reading `\.userSession`): `is_favorite=true` pages of every list, search of every list (the RN hook passes no `list` for Bookmarks), refresh on every appearance (the RN focus effect), stored favourites prime the list while nothing is listed. No swipe action (the RN screen has none). Never writes the store.
- `Store/ArticleStore.swift`: `listFavorites()`.
- `App/UserSession.swift`: optional `feedCache:` (like `database:`), so tests do not share the App Group's `feed-user-1.json`.

Tests: `FeedModelTests` (feed list only, nothing stored or prefetched; cache written; stale banner over the cached page on a failed refresh; error state and Retry; empty Feed; an offline save hides the row, queues it and drops it from the cache; a rejected save (404: the drain drops the `list` row and the store row) shows the row again; stored ids left out of pages, cache and search; a save during search leaves the cached page; clearing a search re-caches the refreshed page; TTL; `article(after:)`), `BookmarksModelTests` (the three jest cases, the error state, search of every list, refresh on every appearance). `SyncFixtures.readsPage` takes `list:`; `SyncHarness` uses a temporary `FeedCache`.

Dropped or re-expressed jest cases:
- FeedScreen "saves a row to Reads on swipe, then drops it" (online): the screen has no online path; a save is always store + outbox and the row leaves through `exclude` (the offline case). `invalidateReads()` has no counterpart (Reads observes the store).
- "keeps the row and rejects (so it springs back)": the row is not awaited or sprung back; it hides at once and returns when the drain drops the row. The alert is the root's (`Outbox.rejections`), not re-tested here.
- "opens an item in the reader, and drops it … when the reader reports it left the Feed": no callback; a save from the reader hides the row through the same `exclude`. Opening is `NavigationLink(value: Route.article)` (view layout, not unit-tested).
- "leaves out items with a queued Save to Reads": every stored id now, not `Outbox.pendingSaveIds`.
- "does not cache search results when search opens while a swipe save is in flight": nothing awaits the save; covered by "a save during search leaves the cached page" (the cache is written only while no results show).
- BookmarksScreen "keeps favorited Feed items out of the offline store": re-expressed as "lists favourites from every list without storing them". Bookmarks never writes the store (`Sync` is the only writer of Reads pages, decision 8), so Reads favourites are no longer upserted from Bookmarks either.
- BookmarksScreen re-primed from the store on every focus, replacing a loaded list; native primes only while nothing is listed.

Notes:
- The Feed's empty state can flash when the whole cached page is excluded while the first refresh runs.
- A rejected save's row returns to the list and, on that `exclude` change, to the cache too, matching the server.
- Timing-based tests poll with `eventually`; `rendersStoredFavoritesWhileTheNetworkCallIsPending` holds the stub handler with a polling sleep.
