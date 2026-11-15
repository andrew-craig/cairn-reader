---
id: feature_ab01
title: [iOS native] Use the app's own icons instead of SF Symbols
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
Switch from the iOS system icons to the app's own icons.
This overrides the epic's "SF Symbols" mapping (decision 7: platform look wins) for icons; system containers stay.
## Source
- SVGs: `apps/mobile/assets/icons/*.svg` and the RN icon components `apps/mobile/src/components/icons/*` (Feed, Read, Profile, Bookmark, Archive, Return, ArrowDown, Bug, ChevronRight); web `apps/web/src/components/icons/*` (Add, Search, Close, …).
## Plan
- [ ] Add the icons to the asset catalog as template SVG symbol/vector images (preserve vector data), one set with outline/solid variants where RN has both.
- [ ] Replace `systemImage:` uses with `image:` / `Image(_:)` across `MainTabs`, `ReaderView` toolbar, list swipe actions (`ReadsView`, `FeedView`) and empty states. List any SF Symbol kept (no app icon exists, e.g. share) with a reason.
## Verify
- iOS App Checks green; owner checks tabs, reader toolbar, swipe actions in light and dark.
