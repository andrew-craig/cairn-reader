---
id: feature_2920
title: Mobile: Add filter/sort controls to ReadScreen
type: feature
status: open
priority: 3
labels: []
blocked_by: []
parent: null
created_at: 2026-03-21T04:20:36Z
updated_at: 2026-03-21T04:20:36Z
---
ReadScreen has no UI for filtering by status or sorting. The backend API supports these via ListContentsParams but no UI controls exist.

## Platforms
Both. iOS: Reads list in `apps/ios` (`Features/Reads`, store-backed, so filters apply to the store query). Android: `ReadsScreen` in `apps/mobile`. (Set at the iOS native cutover, chore_c364: iOS ships from `apps/ios`, Android from `apps/mobile`.)
