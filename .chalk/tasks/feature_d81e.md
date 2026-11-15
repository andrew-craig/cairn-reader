---
id: feature_d81e
title: [iOS native] Search: port the web/Expo header search instead of always-visible .searchable
type: feature
status: open
priority: 2
labels: [ios,ui]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-15T21:51:10Z
updated_at: 2026-11-15T21:51:10Z
---
## Feedback (owner, TestFlight testing)
Implement the same search bar behaviour as web/Expo. Don't use the native always-shown search.
This overrides the epic's `HeaderSearch` → `.searchable` mapping; the search view is added to the custom view budget.
## Source
`apps/mobile/src/components/common/HeaderSearch.tsx` (mirrors the web list header): a round search button in the header; tapping it morphs into a pill field (ring, then stretches leftwards, icon slides to the leading edge, close button appears where the icon was); closing reverses, faster; reduced motion snaps. Closing clears the query.
## Plan
- [ ] Hidden by default: a search toolbar button on Reads, Feed and Bookmarks; no search field in the navigation bar until opened.
- [ ] Opening shows the pill field with focus and a close button, following HeaderSearch's animation (respect Reduce Motion); closing clears `query`, which restores the list (existing `ArticleListModel` behaviour).
- [ ] Remove `.searchable` from `ReadsView`, `FeedView`, `BookmarksView`; keep binding to the existing model `query`.
## Verify
- iOS App Checks green (models unchanged).
- Owner on device: search is hidden until tapped, opens/closes like the Expo app, clearing restores the list, on all three lists.
