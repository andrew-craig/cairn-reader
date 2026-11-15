---
id: task_4e17
title: [iOS native 08] Connectivity, Sync (drain, Reads page sync, prefetch), FeedCache
type: task
status: open
priority: 2
labels: [ios,offline]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-15T03:40:23Z
---
## Goal
The background machinery around the store: know when the device is offline, sync Reads pages into the store, drain and prefetch at the right moments, and keep the last Feed page for the stale banner. All of it owned by `UserSession` (decision 10).

## Source
`apps/mobile/src/utils/network.ts`, `hooks/{useNetworkStatus,useSyncTrigger}.ts`, `services/{syncTrigger,articlePrefetch,feedCache}.ts`, `components/common/SyncTriggerEffect.tsx`, and their tests. `readsInvalidation.ts` is not ported (decision 8).

## Plan
- [x] `Connectivity` (`@Observable`) over `NWPathMonitor`; unknown counts as online. App-wide, not per session.
- [x] `ArticlePrefetch.run`: bodies for unread/reading Reads articles that lack one, 3 concurrent workers (task group), newest first, capped at the 100 most recent, evicts bodies outside the cap, skips offline, aborts the batch on the first `NetworkError`.
- [x] `Sync` (owned by `UserSession`), running one operation at a time. Actor isolation alone is not enough because actor methods interleave at their awaits; the operations must queue.
  - `drain()`: `Outbox.drain`. What `ArticleMutations` requests.
  - `syncReads(cursor:)`: fetch a `list=reads` page and hand it to `ArticleStore.syncPage`. A first-page sync is drain, fetch, `syncPage`, prefetch, in that order. No drain may run between a page's fetch and its `syncPage`: that is what keeps a just-drained save from being deleted by a response fetched before the server had it.
  - `run()`: drain then prefetch, each isolated from the other's failure. Fired on offline-to-online and on `scenePhase` background-to-active transitions (transitions only, not every state read).
  - When the drain drops a row, the next operation is a first-page `syncReads`, which puts the server's state back in the store.
- [x] `FeedCache`: last Feed page as a JSON file in Caches, owned and erased by `UserSession`.
- [x] `UserSession.close()` cancels `Sync` and waits for it before erasing, so nothing writes to the store or sends a request after sign-out.

### Implementation steps
- [x] `Connectivity`: already app-wide, `NWPathMonitor`, online until the first path; extract the status mapping so the network.ts/useNetworkStatus cases can be tested.
- [x] `Outbox.drain` reports whether it dropped a row; `requestDrain` goes (drains are requested through `Sync`).
- [x] `ArticleMutations` takes a `requestDrain` closure (`Sync.requestDrain` in `UserSession`).
- [x] `ArticlePrefetch` (`Services/`), `Sync` (`Services/`), `FeedCache` (`Store/`); `Article` becomes `Codable` for the cache file.
- [x] `UserSession` owns `Sync`, `FeedCache`; takes `isOnline` (from `Connectivity`); `close()` stops `Sync` (which stops the outbox), clears the feed cache, erases the database.
- [x] Triggers in `RootView`: `onChange` of `connectivity.isOnline` (offline to online) and `scenePhase` (to active), through `UserSession.start`.
- [x] Tests: ConnectivityTests, ArticlePrefetchTests, FeedCacheTests, SyncTests, UserSessionTests (close cases, second user), RootViewTests (trigger predicates); adjust Outbox/ArticleMutations/SessionModel tests.
- [x] `apps/ios/CLAUDE.md`, review section, commit.

## Verify
- Ported tests for prefetch (cap, eviction, abort) and feed cache; `syncTrigger.test.ts` re-expressed against `Sync` (isolation, transition-only firing).
- New tests: operations do not interleave (a drain requested during a page fetch runs after that page's `syncPage`); a dropped row triggers a first-page sync; `close()` during a drain and during a page fetch leaves no request and no write afterwards; signing in as a second user after `close()` sees an empty store and outbox.

## Review
Implemented on the worktree branch; not built locally (Linux, no Swift toolchain). **CI verification (iOS App Checks) is pending**; the task stays open until it passes.

What changed:
- `Services/Connectivity.swift`: already app-wide and online until the first path; only the status mapping was extracted (`Connectivity.isOnline(_:)`) so the network.ts/useNetworkStatus cases are testable.
- `Services/Sync.swift` (new): a serial queue (a chain of tasks behind a `Mutex`), not an actor. `requestDrain`/`drain` (a new request joins a drain queued but not started), `syncReads(after: ReadsCursor?)` (first page: drain, fetch, `syncPage`, prefetch; later pages: fetch, `syncPage`), `run` (drain then prefetch, isolated; joins a pending run), `stop` (cancels and awaits every operation and stops the outbox). A drain that dropped a row is followed, inside the same operation, by a first-page sync.
- `Store/Outbox.swift`: `drain()` returns whether it dropped a row; `requestDrain()` removed (drains are requested through `Sync`). `Store/ArticleMutations.swift` takes a `requestDrain` closure (`Sync.requestDrain` in `UserSession`).
- `Services/ArticlePrefetch.swift` (new): task group keeping 3 fetches in flight, newest first; offline skip, abort on the first `NetworkError` (or cancellation), eviction afterwards. Bodies come through an injected `fetchBody` (the convenience init uses `ReadService.getContentById`).
- `Store/FeedCache.swift` (new): `feed-<user id>.json` under the App Group's `Library/Caches`; `Article` is now `Codable` for it.
- `App/UserSession.swift`: owns `Sync` and `FeedCache`, takes `isOnline`; `close()` = cancel work, `sync.stop()`, clear the feed cache, erase the database. `SessionModel.makeSession` has no default any more; `CairnReaderApp` builds sessions with `isOnline` from `Connectivity`.
- `RootView`: `onChange` of `connectivity.isOnline` (offline to online) and `scenePhase` (to active) starts `Sync.run` through `UserSession.start`. Features/Main untouched.

Tests: ConnectivityTests, ArticlePrefetchTests, FeedCacheTests, SyncTests (run and reads suites), new UserSessionTests cases (close during a drain, close during a page fetch, sign-in after close sees an empty store, outbox and feed cache), RootViewTests trigger predicates; OutboxTests/ArticleMutationsTests adjusted for the removed `requestDrain`.

Dropped or re-expressed jest cases:
- articlePrefetch.test.ts "is single-flight" and its "allows a new run" twin: there is no in-flight guard; prefetch only runs inside `Sync`'s queue. The "new run" case is kept as a plain repeat run.
- syncTrigger.test.ts "does not stack concurrent runs": re-expressed as "a run requested during a run joins it" (the second call waits for the joined run instead of returning at once). `runConsumersIsolated` and "does not reject" become "a failed drain does not stop the prefetch" (`run` cannot throw).
- useSyncTrigger.test.ts: re-expressed as `RootView.runsSync` predicate tests; "re-render with unchanged state" and "online at mount" rely on `onChange` not firing for the initial or an unchanged value (not unit-tested). Like the RN app, inactive-to-active also fires (iOS goes background, inactive, active).
- useNetworkStatus/network.test.ts: "unknown" is the state before the monitor's first path.
- OutboxTests `aPayloadReplacedMidRequestIsNotLost` no longer relies on a mid-drain request (it drains twice).

Risks and follow-ups:
- A queued prefetch (up to 100 bodies) delays a mutation's drain until it finishes, since operations do not interleave.
- `Outbox`'s own one-drain-at-a-time/rerun guard is now redundant with `Sync`'s queue (only tests call `drain` concurrently); left as is.
- Timing-based tests: `aRunRequestedDuringARunJoinsIt` (100 ms for the second run to join), `neverStartsACandidateBeyondTheFirstWaveOnceAborted` (100 ms delay of a2/a3), and the close tests (blocking/sleeping stub handlers).

