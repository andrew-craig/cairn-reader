---
id: feature_753f
title: Add proactive duplicate detection in AddLinkModal
type: feature
status: open
priority: 4
labels: []
blocked_by: []
parent: null
created_at: 2026-03-21T04:20:37Z
updated_at: 2026-03-21T04:20:37Z
---
Currently duplicates are only detected after the user attempts to add a URL. Add proactive detection while typing.

## Platforms
Both. iOS: `AddLinkModel` in `apps/ios`. Android: `AddLinkModal` in `apps/mobile`. (Set at the iOS native cutover, chore_c364: iOS ships from `apps/ios`, Android from `apps/mobile`.)
