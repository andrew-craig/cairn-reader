---
id: bug_3e3d
title: [iOS native] You tab: Bookmarks count does not update after bookmarking an article
type: bug
status: closed
priority: 2
labels: [ios,ui]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-15T22:01:13Z
updated_at: 2026-11-16T07:22:58Z
---
## Feedback (owner, TestFlight testing of task 11)
The Bookmarks count on the You tab does not update when an article is bookmarked.
## Cause (from reading the code, not yet reproduced in a test)
`apps/ios/CairnReader/Features/You/YouModel.swift`: `bookmarksCount` is a server count (`countUserContents(isFavorite: true)`) fetched by `refreshIfStale()` from `YouView.onAppear`, and skipped when the last successful fetch is under 30 s old. Two things keep it stale:
1. **The 30 s skip.** Nothing marks the count stale when a favourite changes, so returning to You within 30 s shows the old number.
2. **The outbox.** `ArticleMutations.setFavorite` writes the store and queues an outbox row (decision 9). Until `Outbox.drain()` sends it, a fetch still returns the old server count. A fetch that runs right after bookmarking, or while offline, is still wrong.
RN had the same 30 s skip, but there the PATCH went out at once, so a later fetch was correct.
## Plan
- [x] Mark `YouModel` stale when a favourite is set or cleared (a signal from `ArticleMutations.setFavorite`, or the outbox passing on changes to favourites), so the next time You appears it fetches again.
- [x] Make sure that fetch runs after the queued favourite has been sent (refetch once the drain finishes, or ask for a drain first), so the server count includes the change. Pick the smallest design that respects decision 9 (writes only through the outbox) and decision 8 (no second copy of article state). Don't adjust the count on the device.
- [x] Unit test: favourite an article, the drain sends it, You appears within 30 s, and the count shows the new value. Also: offline, then online again.
## Verify
- iOS App Checks green.
- Owner on device: bookmark or unbookmark from the reader, go to You, and the count matches. Also after going offline and coming back online.
## Review
**Design.** The signal is the outbox's own: `Outbox.send` yields on a new `Outbox.favoriteChanges` (`AsyncStream<Void>`, buffering only the newest, one consumer, finished by `stop()`), right after the server accepted an `is_favorite` row. That is the same mechanism as `Outbox.rejections`, and it fires only once the edit is on the server, so a fetch after it includes the change (offline, nothing is sent and nothing fires until the drain after reconnecting sends it). No count is adjusted on the device and no article state is copied (decisions 8 and 9 unchanged: `ArticleMutations` still only writes the store and the queue row; the drain is still the only sender).
- `YouModel.followFavoriteChanges(_:)` clears `lastFetchedAt` on each signal, so the next appearance refetches even within 30 s, and refetches at once while `isVisible`. A counter of signals keeps a fetch that overlapped a signal from starting the clock (its count may predate the send).
- `MainTabs` now owns the `YouModel` (`@State`, built in a new `init(session:)`) and runs `followFavoriteChanges(session.outbox.favoriteChanges)` in a TabView-level `.task`, next to the rejections loop, so a favourite sent while You is hidden (reader pushed from Bookmarks or another tab) is not missed; a `.task` in `YouView` would be cancelled whenever You disappears. `YouView` takes the model and sets `isVisible` in `onAppear`/`onDisappear`.
- Known limit: a signal arriving while You is visible and a fetch is already in flight does not start a second fetch; the clock is not started, so the next appearance refetches.

**Files.** `Store/Outbox.swift`, `Features/You/YouModel.swift`, `Features/You/YouView.swift`, `Features/Main/MainTabs.swift`, `apps/ios/CLAUDE.md` (Writes and You notes).

**Tests** (`CairnReaderTests/YouModelTests.swift`, new suite `YouModelFavoriteTests` over `SyncHarness` with a server whose count rises on an accepted favourite PATCH): favourite, drain sends it, You appears 1 s later and shows the new count; favourite offline (count unchanged, row queued), back online, `Sync.run` sends it, You appears and shows the new count; a favourite sent while You is visible refetches without an appearance; a non-favourite edit (scroll position) leaves the 30 s interval in force. Not built locally (Linux); verify on iOS App Checks.
