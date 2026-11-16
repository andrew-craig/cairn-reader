---
id: bug_6ed5
title: [iOS native] Reader: group the bottom quick actions on the right
type: bug
status: closed
priority: 2
labels: [ios,ui]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-15T21:51:10Z
updated_at: 2026-11-16T10:52:05Z
---
## Feedback (owner, TestFlight testing)
The quick actions at the bottom of the article page should be grouped together and sit on the right.
## Change
`apps/ios/CairnReader/Features/Reader/ReaderView.swift` `toolbar`: the bottom-bar `ToolbarItemGroup` currently spreads Next, Favorite, Archive/Save to Reads and Share across the bar with `Spacer()`s. Put one leading `Spacer()` first and drop the others, so the buttons form one group on the trailing side (iOS 26 glass groups them). Match the RN `BottomActionMenu` order.
- [x] One leading `Spacer()` in the bottom-bar `ToolbarItemGroup`; the three others removed. Order unchanged: Next, Favorite, Archive / Save to Reads, Share (RN `BottomActionMenu` is Back, Next, Favorite, Archive / Save to Reads; Back is the navigation bar's here, Share is iOS-only and stays last).
## Verify
- Owner on device: actions appear as one cluster on the right for Reads items, Feed items, and with/without Next.
## Review
- Changed: `apps/ios/CairnReader/Features/Reader/ReaderView.swift` `toolbar` only (spacers; button images untouched).
- Verified: `xcodebuild test` on the iPhone 17 Pro simulator (iOS 26.5) builds with warnings as errors and the suite passes. One issue recorded outside any test, from `HTTPClientTests.plusInQueryIsPercentEncoded` (query encoding), reproduces on every run and is unrelated (`HTTPClient` is untouched). Layout on device is the owner's check.
