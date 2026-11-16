---
id: bug_f421
title: [iOS native] Offline banner: orange text, no orange background
type: bug
status: open
priority: 2
labels: [ios,ui]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-15T21:51:10Z
updated_at: 2026-11-15T21:51:10Z
---
## Feedback (owner, TestFlight testing of task 10)
The "You're offline" message is too prominent.
## Change
`apps/ios/CairnReader/App/OfflineBanner.swift`: remove the `.warning` background and draw the text in `Color(.warning)` (orange) instead of `Color(.background)`. Keep the text "You're offline" and its current position (above everything, under the status bar).
- [x] `.background(Color(.warning), …)` removed; text drawn in `Color(.warning)`. Text, font, padding and placement (the root's top `safeAreaInset`) unchanged.
## Verify
- Owner on device: airplane mode shows orange "You're offline" text with no filled bar, in light and dark.
## Review
- Changed: `apps/ios/CairnReader/App/OfflineBanner.swift` only (`apps/ios/CLAUDE.md` does not describe the banner's colours).
- To check on device: with no background, the strip is clear, so a list scrolled under the navigation bar may show through behind the text.
- Verified: `xcodebuild test` on the iPhone 17 Pro simulator (iOS 26.5) builds with warnings as errors and the suite passes (same unrelated `HTTPClientTests.plusInQueryIsPercentEncoded` issue as bug_6ed5).
