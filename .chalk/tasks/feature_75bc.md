---
id: feature_75bc
title: [iOS native 11] Feed tab and Bookmarks
type: feature
status: open
priority: 2
labels: [ios,ui]
blocked_by: [feature_33bc]
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-14T02:35:53Z
---
## Goal
The online-only skim list and the favourites list, both on `ArticleListView`.

## Source
`apps/mobile/src/screens/{FeedScreen,BookmarksScreen}.tsx`; tests `FeedScreen.test.tsx`, `BookmarksScreen.test.tsx`.

## Plan
- [ ] Feed: `list=feed`. Pages are never written to `ArticleStore` and never prefetched. Keep the last first page in `FeedCache`; when a refresh fails show the cached page with a stale banner.
- [ ] Leading full-swipe "Save to Reads" (`info` colour) through `ArticleMutations.saveToReads`. The row leaves the Feed and `FeedCache` once the server has moved it or the move is queued; only a definitive rejection alerts and puts the row back. Bump the Reads invalidation flag on success.
- [ ] Fetched pages and search results exclude ids with a queued save (`Outbox.pendingSaveIds` into the model's `exclude`).
- [ ] Bookmarks: favourites filter, search, pushed from You. No swipe action unless the RN screen has one (check and match).

## Verify
- Ported tests: offline save removes the row and queues; definitive rejection restores it; pending ids excluded from pages and search; stale banner on failed refresh.
