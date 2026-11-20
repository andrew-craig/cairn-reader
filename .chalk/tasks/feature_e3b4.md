---
id: feature_e3b4
title: Mobile: Smart URL detection (auto-determine article vs feed)
type: feature
status: open
priority: 4
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-04-07T11:53:36Z
updated_at: 2026-04-07T11:53:36Z
---

## Platforms
Both. Check first: both add-link flows already run a debounced `detectURL` and pick article or feed (iOS `AddLinkModel`, Android `AddLinkModal`); close this if that covers it. (Set at the iOS native cutover, chore_c364: iOS ships from `apps/ios`, Android from `apps/mobile`.)
