---
id: bug_3e3d
title: [iOS native] You tab: Bookmarks count does not update after bookmarking an article
type: bug
status: open
priority: 2
labels: [ios,ui]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-15T22:01:13Z
updated_at: 2026-11-15T22:01:13Z
---
## Feedback (owner, TestFlight testing of task 11)
The Bookmarks count on the You tab does not update when an article is bookmarked.
## Cause (from reading the code, not yet reproduced in a test)
`apps/ios/CairnReader/Features/You/YouModel.swift`: `bookmarksCount` is a server count (`countUserContents(isFavorite: true)`) fetched by `refreshIfStale()` from `YouView.onAppear`, and skipped when the last successful fetch is under 30 s old. Two things keep it stale:
1. **The 30 s skip.** Nothing marks the count stale when a favourite changes, so returning to You within 30 s shows the old number.
2. **The outbox.** `ArticleMutations.setFavorite` writes the store and queues an outbox row (decision 9). Until `Outbox.drain()` sends it, a fetch still returns the old server count. A fetch that runs right after bookmarking, or while offline, is still wrong.
RN had the same 30 s skip, but there the PATCH went out at once, so a later fetch was correct.
## Plan
- [ ] Mark `YouModel` stale when a favourite is set or cleared (a signal from `ArticleMutations.setFavorite`, or the outbox passing on changes to favourites), so the next time You appears it fetches again.
- [ ] Make sure that fetch runs after the queued favourite has been sent (refetch once the drain finishes, or ask for a drain first), so the server count includes the change. Pick the smallest design that respects decision 9 (writes only through the outbox) and decision 8 (no second copy of article state). Don't adjust the count on the device.
- [ ] Unit test: favourite an article, the drain sends it, You appears within 30 s, and the count shows the new value. Also: offline, then online again.
## Verify
- iOS App Checks green.
- Owner on device: bookmark or unbookmark from the reader, go to You, and the count matches. Also after going offline and coming back online.
