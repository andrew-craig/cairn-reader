---
id: task_4e17
title: [iOS native 08] Connectivity, SyncTrigger, article prefetch and FeedCache
type: task
status: open
priority: 2
labels: [ios,offline]
blocked_by: [task_387d]
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-14T02:35:53Z
---
## Goal
The background machinery around the store: know when the device is offline, drain and prefetch at the right moments, and keep the last Feed page for the stale banner.

## Source
`apps/mobile/src/utils/network.ts`, `hooks/{useNetworkStatus,useSyncTrigger}.ts`, `services/{syncTrigger,articlePrefetch,feedCache,readsInvalidation}.ts`, `components/common/SyncTriggerEffect.tsx`, and their tests.

## Plan
- [ ] `Connectivity` (`@Observable`) over `NWPathMonitor`; unknown counts as online.
- [ ] `ArticlePrefetch.run`: bodies for unread/reading Reads articles that lack one, 3 concurrent workers (task group), newest first, capped at the 100 most recent, evicts bodies outside the cap, skips offline, aborts the batch on the first `NetworkError`, single-flight.
- [ ] `SyncTrigger.run`: `Outbox.drain` then prefetch, each isolated from the other's failure, single-flight.
- [ ] Fire `SyncTrigger.run` on offline-to-online and on `scenePhase` background-to-active transitions (transitions only, not every state read), from the authenticated root only.
- [ ] `FeedCache`: last Feed page as a JSON file in Caches; `clear` on logout.
- [ ] Reads invalidation: a flag/version on the shared Reads list model that Save to Reads bumps (replaces `readsInvalidation.ts`; no global event bus).
- [ ] Wire `SessionModel.logout` to `ArticleStore.clear` and `FeedCache.clear`.

## Verify
- Ported tests for prefetch (cap, eviction, abort, single-flight), sync trigger (isolation, single-flight, transition-only firing) and feed cache.
