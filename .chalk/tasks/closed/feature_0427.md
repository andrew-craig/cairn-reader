---
id: feature_0427
title: Mobile: swipe to archive in Reads list
type: feature
status: closed
priority: 2
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-10-31T02:44:43Z
updated_at: 2026-10-31T02:47:42Z
---


Swipe right-to-left on a Reads row to archive. Light green background with a white archive icon behind the row; releasing past the icon's full width archives, and the icon switches from outline to solid at that point. Reuse ArchiveIcon (outline from the quick action bar) and add the solid variant to it.

## Plan
- [x] Add `filled` variant to `ArchiveIcon` (mirrors `BookmarkIcon`)
- [x] `SwipeToArchive` wrapper on gesture-handler `Swipeable` (already a dependency; no reanimated in the project)
- [x] Mount `GestureHandlerRootView` in `App.tsx`; gesture-handler jest setup
- [x] Wire into `ReadsScreen` via `ArticleListScreen.renderItem`; archive through `ArticleMutations.archive` (offline outbox)
- [x] Tests: release past threshold archives, short release springs back

## Review
- Threshold = `Spacing.md + 24` (icon fully uncovered). Action panel spans the row, so a confirmed swipe carries the row off-screen, then the list collapses the gap with `LayoutAnimation`.
- Background uses theme `success` green.
- Verified with jest, tsc, eslint. Not verified on a device/simulator (no simulator in the cloud environment).
