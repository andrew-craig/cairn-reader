---
id: feature_fd31
title: Mobile: swipe Feed rows left-to-right to save to Reads
type: feature
status: closed
priority: 2
labels: [mobile]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-11-01T21:30:59Z
updated_at: 2026-11-01T21:33:21Z
---
Generalise SwipeToArchive into a reusable SwipeAction (direction, icon, colour, onAction). Reads: right-to-left archive (green, ArchiveIcon). Feed: left-to-right Save to Reads (blue, ReadIcon), awaited ReadService.moveToReads; row springs back on failure.

- [x] SwipeAction component replacing SwipeToArchive (+ tests for both directions and failure spring-back)
- [x] Add blue theme colour
- [x] ReadsScreen uses SwipeAction
- [x] FeedScreen swipe -> moveToReads, invalidateReads, remove row
- [x] Update apps/mobile/AGENTS.md
- [x] tsc, lint, jest

## Review
- `SwipeToArchive` → generic `SwipeAction` (`direction`, `icon`, theme `color`, `onAction`). Same thresholds, outline→solid icon step and full-row action. New: if `onAction` returns a rejecting promise the row springs back (`swipeable.close()`).
- Reads: `direction="left"`, `ArchiveIcon`, `success` — behaviour unchanged.
- Feed: `direction="right"`, `ReadIcon`, new `info` blue (`#007AFF` / dark `#0A84FF`). Awaits `ReadService.moveToReads` like the reader; on success `invalidateReads()` + remove row (LayoutAnimation, FeedCache updated); on failure alert + rethrow → row springs back.
- Tests: SwipeAction both directions, wrong direction, reject/resolve; FeedScreen swipe success/failure. tsc, eslint, jest (all suites) green.
