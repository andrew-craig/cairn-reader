---
id: feature_cf80
title: Mobile: Add article status change UI
type: feature
status: open
priority: 3
labels: []
blocked_by: []
parent: null
created_at: 2026-03-21T04:20:37Z
updated_at: 2026-03-21T04:20:37Z
---
ReadScreen and ArticleDetailScreen have no way to change article status (unread/reading/completed/archived). Add status controls.

## Platforms
Both. iOS: reader toolbar and Reads swipe actions in `apps/ios` (through `ArticleMutations`). Android: `ReadArticleDetailScreen` and `ReadsScreen` in `apps/mobile`. (Set at the iOS native cutover, chore_c364: iOS ships from `apps/ios`, Android from `apps/mobile`.)
