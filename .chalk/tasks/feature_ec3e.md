---
id: feature_ec3e
title: Add URL preview UI in AddLinkModal
type: feature
status: open
priority: 3
labels: []
blocked_by: []
parent: null
created_at: 2026-03-21T04:20:37Z
updated_at: 2026-03-21T04:20:37Z
---
The AddLinkModal detects URL type and title but doesn't show a preview. Show detected title/image so users can confirm before saving.

## Platforms
Both. iOS: `AddLinkSheet` in `apps/ios`. Android: `AddLinkModal` in `apps/mobile`. (Set at the iOS native cutover, chore_c364: iOS ships from `apps/ios`, Android from `apps/mobile`.)
