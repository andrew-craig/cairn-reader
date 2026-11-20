---
id: feature_d632
title: Mobile: Add browser extension for sharing URLs to Cairn
type: feature
status: open
priority: 4
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-04-07T11:53:35Z
updated_at: 2026-04-07T11:53:35Z
---

## Platforms
Both, as separate native pieces. iOS: a share extension in `apps/ios` (the App Group `group.com.cairnapp.cairnreader` exists for this, decision 11). Android: a share intent filter, which needs an Expo config plugin or a committed `android/` project in `apps/mobile`. (Set at the iOS native cutover, chore_c364: iOS ships from `apps/ios`, Android from `apps/mobile`.)
