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
- [ ] Feed: `list=feed`. Pages are never written to `ArticleStore` and never prefetched. Keep the last first page in `FeedCache`; when a refresh fails show the cached page with a stale banner.
- [ ] The model's `exclude` set is `ArticleStore.observeStoredIDs()`: the store only holds Reads, so an id in it has been saved (decision 8). This one rule hides a row the moment it is saved from the swipe or from the reader, keeps it hidden in later pages and search results while the save is queued, and brings it back if the save is rejected and the store row is deleted. No `pendingSaveIds`, no invalidation flag, no put-back code.
- [ ] Leading full-swipe "Save to Reads" (`info` colour) calls `ArticleMutations.saveToReads`.
- [ ] `FeedCache` is written from the list after exclusion, so a saved row is not in the cached page.
- [ ] Bookmarks: favourites filter, search, pushed from You. No swipe action unless the RN screen has one (check and match).

## Verify
- `FeedScreen.test.tsx` re-expressed: an offline save hides the row and queues it; a rejected save (drain drops the `list` row) shows it again; stored ids are excluded from pages and search; stale banner on failed refresh; the cached page omits saved rows.
- `BookmarksScreen.test.tsx` ported.
