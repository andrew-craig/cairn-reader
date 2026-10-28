---
id: task_f9d3
title: Web: keep search pill ring outside the search/close button fill
type: task
status: closed
priority: 2
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-10-28T06:38:08Z
updated_at: 2026-10-28T06:39:14Z
---


## Plan
- [x] Grow the search pill 3px past the 42px search/close buttons on every side (spilling into the header gap) so its ring sits outside their hover/press fill
- [x] Drop the close button's padding/background-clip workaround, now redundant
- [x] Verify in Chromium (hovered close button clears the ring) and run ArticleList tests

## Review
Ring now sits 2px clear of the button fill. Header layout unchanged: the slot is still 42px; the pill overflows it by 3px top, bottom and right, and its open left edge stays flush with the content column.
