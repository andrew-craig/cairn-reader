---
id: feature_d81e
title: [iOS native] Search: port the web/Expo header search instead of always-visible .searchable
type: feature
status: closed
priority: 2
labels: [ios,ui]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-15T21:51:10Z
updated_at: 2026-11-16T10:52:07Z
---
## Feedback (owner, TestFlight testing)
Implement the same search bar behaviour as web/Expo. Don't use the native always-shown search.
This overrides the epic's `HeaderSearch` → `.searchable` mapping; the search view is added to the custom view budget.
## Source
`apps/mobile/src/components/common/HeaderSearch.tsx` (mirrors the web list header): a round search button in the header; tapping it morphs into a pill field (ring, then stretches leftwards, icon slides to the leading edge, close button appears where the icon was); closing reverses, faster; reduced motion snaps. Closing clears the query.
## Plan
- [x] Hidden by default: a search toolbar button on Reads, Feed and Bookmarks; no search field in the navigation bar until opened.
- [x] Opening shows the pill field with focus and a close button, following HeaderSearch's animation (respect Reduce Motion); closing clears `query`, which restores the list (existing `ArticleListModel` behaviour).
- [x] Remove `.searchable` from `ReadsView`, `FeedView`, `BookmarksView`; keep binding to the existing model `query`.
## Verify
- iOS App Checks green (models unchanged).
- Owner on device: search is hidden until tapped, opens/closes like the Expo app, clearing restores the list, on all three lists.
## Review
- Approach: one shared modifier, `.headerSearch(query:prompt:)` in `Features/ArticleList/HeaderSearch.swift`, replaces `.searchable` on Reads, Feed and Bookmarks, bound to the existing `query` (models unchanged). Closed: a trailing toolbar button with the app's `search` icon. Open: the bar button hides and a pill field appears with the RN timings: ring (120 ms), stretch leftwards from a 44 pt circle at the trailing edge (260 ms after 80 ms, the RN ease-in-out curve), icon on the leading edge, field and close button fade in (close scales from 0.6), field focused; the ring is `primary` while focused. Close clears `query` (the model restores the list) and reverses, faster (100/220/120 ms). Reduce Motion snaps ring and width and fades the contents. Colour tokens only; labels "Search", "Close search" and the prompt on the field.
- Placement: tried the navigation bar first (`ToolbarItem(.topBarTrailing)`, `.principal`, a custom `ToolbarContent`, id-swapped items). On iOS 26 none works: a bar item keeps the width it was first laid out at, `.principal` is hidden under a large title, `@State` in a custom `ToolbarContent` does not stick, and a custom item view containing an image is rendered as that one image (field and close button disappear). So the pill sits in `.safeAreaBar(edge: .top)` beneath the navigation bar (with the bar's scroll-edge effect), and the morph starts from a circle under the bar's trailing edge rather than from the bar button itself.
- Verification: `xcodebuild test` on the iPhone 17 simulator (iOS 26.5) passes, warnings as errors (386 test cases, no issues; `plusInQueryIsPercentEncoded` did not record one this run). Visual check with a throwaway harness (a 20-row list with the modifier, no backend): closed, open (focused, keyboard up, ring, icon leading, close trailing) and closed again after tapping close. The `#Preview` in `HeaderSearch.swift` stays. On-device feel is the owner's.
