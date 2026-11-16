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
## Verify
- Owner on device: airplane mode shows orange "You're offline" text with no filled bar, in light and dark.
