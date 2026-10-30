---
id: task_abc0
title: Mobile: morphing header search pill with live search (port of web)
type: task
status: closed
priority: 2
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-10-28T07:53:17Z
updated_at: 2026-10-28T08:01:51Z
---


## Plan
- [x] `HeaderSearch` (common): round search button morphs into a pill (48px, 42px buttons inset 3px so the ring sits outside them), same staged timings/easings as web via RN Animated; reduced motion snaps size and fades
- [x] `ScreenHeader` `search` prop: pill grows leftwards over the title (to 3px past its start; Back button stays usable)
- [x] `useCursorArticleList`: owns open state + query; 300ms debounced live search; clearing restores the pre-search list at once and refreshes quietly; a reset load during search updates the restore list instead of replacing results
- [x] `ArticleListScreen`: header rendered as an element (no remount while typing); no spinner/error takeover while search is open; "Results for" banner removed
- [x] Feed, Reads, Bookmarks switched over; `SearchModal` deleted; docs updated
- [x] Verify: jest (310 pass), tsc, lint (no new warnings), frames from a react-native-web harness

## Review
User chose live search as you type (like web) over submit-to-search. Frames rendered with react-native-web match web's timing; the open pill extends 3px past the title start so its rounded end hides the title's first glyph.
