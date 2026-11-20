---
id: feature_a849
title: Enhanced article reader with adjustable fonts
type: feature
status: open
priority: 3
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-04-08T09:02:33Z
updated_at: 2026-04-08T09:02:33Z
---

## Platforms
Both, implemented differently. iOS: the reader's HTML template in the web view (`apps/ios`), alongside Dynamic Type. Android: `ArticleContent` (`react-native-render-html`) in `apps/mobile`. (Set at the iOS native cutover, chore_c364: iOS ships from `apps/ios`, Android from `apps/mobile`.)
