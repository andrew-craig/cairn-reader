---
id: bug_6ed5
title: [iOS native] Reader: group the bottom quick actions on the right
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
## Feedback (owner, TestFlight testing)
The quick actions at the bottom of the article page should be grouped together and sit on the right.
## Change
`apps/ios/CairnReader/Features/Reader/ReaderView.swift` `toolbar`: the bottom-bar `ToolbarItemGroup` currently spreads Next, Favorite, Archive/Save to Reads and Share across the bar with `Spacer()`s. Put one leading `Spacer()` first and drop the others, so the buttons form one group on the trailing side (iOS 26 glass groups them). Match the RN `BottomActionMenu` order.
## Verify
- Owner on device: actions appear as one cluster on the right for Reads items, Feed items, and with/without Next.
