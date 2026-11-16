---
id: feature_ab01
title: [iOS native] Use the app's own icons instead of SF Symbols
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
Switch from the iOS system icons to the app's own icons.
This overrides the epic's "SF Symbols" mapping (decision 7: platform look wins) for icons; system containers stay.
## Source
- SVGs: `apps/mobile/assets/icons/*.svg` and the RN icon components `apps/mobile/src/components/icons/*` (Feed, Read, Profile, Bookmark, Archive, Return, ArrowDown, Bug, ChevronRight); web `apps/web/src/components/icons/*` (Add, Search, Close, …).
## Plan
- [x] Add the icons to the asset catalog as template SVG symbol/vector images (preserve vector data), one set with outline/solid variants where RN has both.
- [x] Replace `systemImage:` uses with `image:` / `Image(_:)` across `MainTabs`, `ReaderView` toolbar, list swipe actions (`ReadsView`, `FeedView`) and empty states. List any SF Symbol kept (no app icon exists, e.g. share) with a reason.
## Verify
- iOS App Checks green; owner checks tabs, reader toolbar, swipe actions in light and dark.
## Review
- `apps/ios/CairnReader/Resources/Icons.xcassets`: template-rendered SVG imagesets with Preserve Vector Data, 24x24 pt (as the RN/web components): `feed`/`feed-fill`, `read`/`read-fill`, `profile`/`profile-fill`, `bookmark`/`bookmark-fill`, `archive`, `arrow-down` (RN components, converted faithfully to plain SVG with RN stroke widths), `search`, `close` (web Ionicons, for the header search follow-up). Return, Bug, ChevronRight and Add are not added: nothing uses them yet.
- `MainTabs`: Feed, Reads and You tabs use `feed`, `read`, `profile`, the `-fill` variant while selected (bug_eb92).
- Reader bottom toolbar (images only, layout untouched): Next Article `arrow-down` (RN `next-article`), Favorite `bookmark`/`bookmark-fill`, Save to Reads `read`, Archive `archive`.
- Swipes: Reads Archive `archive`, Feed Save to Reads `read` (as RN). Empty states: Reads `read`, Feed `feed`, Bookmarks `bookmark`.
- Buttons and Labels use the generated `ImageResource` symbols (`.archive`, `.bookmarkFill`, ...); `Tab` and `ContentUnavailableView` take only a name string.
### SF Symbols kept
- `wifi.slash` (Reader "Not available offline", Reads/Feed/Bookmarks load errors): no app icon for offline/error.
- `ShareLink` default share symbol in the reader: no app share icon.
- `dot.radiowaves.up.forward` (RSS) and `envelope` (Newsletters) in `Route.swift`: placeholder screens, no app icon.
### Verification
- `xcodebuild test` (iPhone 17 simulator, iOS 26.2; the iPhone 16 runtime is not installed): TEST SUCCEEDED, no warnings.
- `assetutil` on the built `Assets.car`: all 12 icons compiled as template images (24 pt, @3x 72 px) plus vector representations.
- Not checked visually: the simulator stops at login (no backend), so tabs, toolbar and swipes need the owner's check in light and dark.
