---
id: task_4e17
title: [iOS native 08] Connectivity, Sync (drain, Reads page sync, prefetch), FeedCache
type: task
status: open
priority: 2
labels: [ios,offline]
blocked_by: [task_387d]
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-15T00:01:49Z
---
## Goal
The background machinery around the store: know when the device is offline, sync Reads pages into the store, drain and prefetch at the right moments, and keep the last Feed page for the stale banner. All of it owned by `UserSession` (decision 10).

## Source
`apps/mobile/src/utils/network.ts`, `hooks/{useNetworkStatus,useSyncTrigger}.ts`, `services/{syncTrigger,articlePrefetch,feedCache}.ts`, `components/common/SyncTriggerEffect.tsx`, and their tests. `readsInvalidation.ts` is not ported (decision 8).

## Plan
- [ ] `Connectivity` (`@Observable`) over `NWPathMonitor`; unknown counts as online. App-wide, not per session.
- [ ] `ArticlePrefetch.run`: bodies for unread/reading Reads articles that lack one, 3 concurrent workers (task group), newest first, capped at the 100 most recent, evicts bodies outside the cap, skips offline, aborts the batch on the first `NetworkError`.
- [ ] `Sync` (owned by `UserSession`), running one operation at a time. Actor isolation alone is not enough because actor methods interleave at their awaits; the operations must queue.
  - `drain()`: `Outbox.drain`. What `ArticleMutations` requests.
  - `syncReads(cursor:)`: fetch a `list=reads` page and hand it to `ArticleStore.syncPage`. A first-page sync is drain, fetch, `syncPage`, prefetch, in that order. No drain may run between a page's fetch and its `syncPage`: that is what keeps a just-drained save from being deleted by a response fetched before the server had it.
  - `run()`: drain then prefetch, each isolated from the other's failure. Fired on offline-to-online and on `scenePhase` background-to-active transitions (transitions only, not every state read).
  - When the drain drops a row, the next operation is a first-page `syncReads`, which puts the server's state back in the store.
- [ ] `FeedCache`: last Feed page as a JSON file in Caches, owned and erased by `UserSession`.
- [ ] `UserSession.close()` cancels `Sync` and waits for it before erasing, so nothing writes to the store or sends a request after sign-out.

## Verify
- Ported tests for prefetch (cap, eviction, abort) and feed cache; `syncTrigger.test.ts` re-expressed against `Sync` (isolation, transition-only firing).
- New tests: operations do not interleave (a drain requested during a page fetch runs after that page's `syncPage`); a dropped row triggers a first-page sync; `close()` during a drain and during a page fetch leaves no request and no write afterwards; signing in as a second user after `close()` sees an empty store and outbox.
