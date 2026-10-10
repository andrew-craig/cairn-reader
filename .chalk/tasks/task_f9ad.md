---
id: task_f9ad
title: Mobile: Feed and Reads tabs
type: task
status: in_progress
priority: 1
labels: [mobile]
blocked_by: [task_66b2]
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-08T11:32:05Z
updated_at: 2026-10-10T08:28:37Z
---
- [ ] Shared types: list on UserContentResponse + UnifiedSubscription; ReadService.listUserContents({list}), setSourceList, moveToReads
- [ ] Tabs: Feed | Reads | You
- [ ] Reads = current ReadScreen with list='reads'; offline store, prefetch and outbox scoped to Reads only
- [ ] Feed = ArticleListScreen + useCursorArticleList({list:'feed'}); opens in the existing reader; 'Save to Reads' action instead of status triage; no unread counts; online-only with a stale cache
- [ ] AddLinkModal: 'Add Feed' asks Feed or Reads before subscribing
- [ ] Feeds and Newsletters stay separate screens; both get a per-row Feed/Reads toggle (applies to new items only — say so in the UI), via the shared SubscriptionListScreen and ReadService.setSourceList
- [ ] Rename the Feeds screen and You menu entry to RSS so "Feed" only means the list (Newsletters keeps its name); unsubscribe stays RSS-only
- Done when: typecheck, lint, tests pass; manual run shows routed items landing in the right tab

## Plan (drafted 2026-10-10)

Branch: `feat/mobile-feed-reads-tabs` off `origin/main` (checkout is currently a detached HEAD).
Shape: build bottom-up (types → service → shared list plumbing → screens → nav), verify each layer with tests before the next.

### 1. Shared types + ReadService → verify: `read.test.ts` (mobile) passes
- [ ] `apps/shared/src/types/read.ts`: `ContentList = 'feed' | 'reads'`; add `list` to the content summary/detail response types, to `UnifiedSubscription`, to `ListContentsParams` and `UpdateUserContentRequest`, and to `AddURLRequest`. Mirror the fields in `services/read/content/api/openapi.yaml`; the web task reuses these.
- [ ] `ReadService.listUserContents` sends `list`. `searchUserContents` and `countUserContents` get the same param if the backend filter supports it (openapi says Search/Count do).
- [ ] New `ReadService.setSourceList(type, key, list)` → `PUT /content/user/{id}/subscriptions/{type}/{key}/list`. Mirror the error handling of `unsubscribeFromRSSFeed`.
- [ ] New `ReadService.moveToReads(contentId)` = `updateUserContent(id, { list: 'reads' })`. Online-only, not routed through the outbox.
- [ ] `addURL` forwards `list` for feed adds.

### 2. List plumbing → verify: existing ReadScreen/Bookmarks tests still green
- [ ] `useCursorArticleList` already takes `fetchPage`, so Feed needs no change. Check that `search` isn't hard-coded to Reads. `search` calls `searchUserContents` without a list, so a Feed search would return Reads items. Add an optional `list` option to the hook and pass it through.
- [ ] `ReadScreen` passes `list: 'reads'` in `fetchPage`. Also scope `ArticleStore.listRecent` and `upsertMany` so Feed items never enter the offline store, prefetch or outbox. This is the riskiest part: `ArticleStore.listRecent` has no list filter today, so the stale-cache render would mix lists if Feed items were stored. Decision: Feed never writes to `ArticleStore`; its stale cache is in-memory only (see Open questions).

### 3. Feed screen → verify: new `FeedScreen.test.tsx`
- [ ] `FeedScreen`: `ArticleListScreen` + `useCursorArticleList({ list: 'feed' })`, pull-to-refresh, infinite scroll, no unread counts, a stale banner when the fetch fails after data was shown, and an offline empty state ("Feed needs a connection").
- [ ] Reader: Feed items open `ReadArticleDetailScreen` (it needs the article body fetched online, since it's not in the store). Add a route param `source: 'feed' | 'reads'`.
  - Feed: swap the Archive action for "Save to Reads" (`ReadService.moveToReads`, then remove from the Feed list via `onArchived`-style callback). Favorite still works (and exempts the item from the 30-day retention, `task_d0f5`).
  - Reads: unchanged.
- [ ] Do not put Feed items through `ArticleMutations` (outbox). Check how `ReadArticleDetailScreen` handles an article missing from the store/offline.

### 4. Subscription management → verify: `SubscriptionListScreen` tests + manual
- [ ] `SourceRow` gets a Feed/Reads toggle (segmented control) wired to `ReadService.setSourceList`. Optimistic update with rollback and an Alert on failure.
- [ ] Helper text under the toggle or in the screen footer: "Applies to new items only."
- [ ] RSS unsubscribe stays RSS-only (unchanged).
- [ ] Email sender key: confirm `UnifiedSubscription` exposes `sender_id` (the PUT key); if not, add it in the backend response, which would be a small `read` service follow-up.

### 5. AddLinkModal → verify: AddLinkModal test
- [ ] When `detectionResult.type === 'feed'`, "Add Feed" first asks Feed or Reads (default Reads, matching the backend default). Pass `list` to `addURL`. Update the success alert to say where it was routed.

### 6. Navigation + renames → verify: `RootNavigator.test.tsx`, typecheck, lint
- [ ] `MainTabParamList`: `Feed | Reads | You`. `TabNavigator` registers `FeedScreen` and `ReadsScreen`. Rename `ReadScreen` → `ReadsScreen` (file + tests + `screens/index.ts`) with `title="Reads"`.
- [ ] `CustomTabBar`: icon for Feed. Check `components/icons` for a suitable one or add an SVG.
- [ ] Rename `Feeds` → `RSS`: route name, `FeedsScreen` → `RssScreen`, title, You menu entry and nav types. Newsletters keeps its name.
- [ ] Update `apps/mobile/CLAUDE.md` project-structure listing only for the renamed/added files (full docs pass is `task_082f`).

### 7. Verify end to end
- [ ] `npm run typecheck`, `npm run lint`, `npm test` in `apps/mobile` (and `apps/shared` if it has its own).
- [ ] Manual run against a remote backend: subscribe to an RSS feed as Feed → new items appear in Feed, not Reads; toggle to Reads → only new items move; Save to Reads moves an item; offline Reads still opens; Feed offline shows the stale/offline state.

### Decisions (owner, 2026-10-10)
1. **Feed offline:** show the last loaded Feed page from an AsyncStorage snapshot (separate from `ArticleStore`) with the "Showing cached data" banner; opening an item offline shows the reader's "Not available offline" state. (Recommended option A, pending owner confirmation after explanation.)
2. **Save to Reads:** newly saved content lands at the top of Reads. Needs a backend change, tracked as `task_66b2` (moving to reads re-stamps `added_at`). The 'Save to Reads' action depends on it.
3. **Favorites:** kept in the Feed reader; favorited items are exempt from retention (`task_d0f5`).
