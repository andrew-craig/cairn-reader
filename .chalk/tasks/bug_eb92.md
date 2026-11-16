---
id: bug_eb92
title: [iOS native] Feed and Reads tab icons are wrong
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
The Feed and Reads icons are incorrect.
## Cause
`apps/ios/CairnReader/Features/Main/MainTabs.swift` uses SF Symbols `newspaper` (Feed) and `book` (Reads). The RN tab bar (`apps/mobile/src/components/common/CustomTabBar.tsx`) uses `FeedIcon` and `ReadIcon` (`apps/mobile/src/components/icons/`), outline when unselected and filled when selected. The same symbols are reused for the Feed "Save to Reads" swipe and the empty states.
## Change
Use the app's Feed and Read icons in the tabs, with the selected/unselected (filled/outline) variants. Overlaps with the "app's own icons" task: do this as part of it if both are picked up together.
## Verify
- Owner on device: tab icons match the Expo/web app, selected state filled.
## Review
Done with feature_ab01: `MainTabs` picks `feed`/`read`/`profile` or their `-fill` variant from the `selection` binding (`icon(_:for:)`), so the selected tab shows the filled icon as RN `CustomTabBar` does (You uses Profile there too). Tests green; the visual check is the owner's (simulator stops at login).
