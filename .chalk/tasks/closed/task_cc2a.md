---
id: task_cc2a
title: Web: Reads header icon actions with slide-out search
type: task
status: closed
priority: 2
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-10-24T20:37:09Z
updated_at: 2026-10-24T20:37:16Z
---


## Plan
- [x] Port mobile's Ionicons `search-outline` / `add-outline` to web icon components
- [x] Replace the Reads/Feed header search bar + "Add link" button with right-aligned icon buttons
- [x] Search icon reveals an inline field (clip-path slide out leftwards from the icon); results replace the list; Escape / icon closes
- [x] Remove the obsolete SearchModal overlay and its CSS; update web requirements doc
- [x] Tests: plus icon opens Add link; search toggle, focus, results, close

## Review
Type-check, lint and vitest (57 tests) pass. Verified in Chromium against a mocked API (desktop + 390px).
