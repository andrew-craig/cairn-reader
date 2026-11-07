---
id: task_fab2
title: Add oxblood brand color for the web wordmark
type: task
status: closed
priority: 2
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-11-07T06:20:09Z
updated_at: 2026-11-07T06:20:23Z
---


Add an oxblood `brand` token to the shared palette and use it for the "Cairn" wordmark at the top left of the web app.
Logo SVG and mobile usage come later.

## Plan
- [x] Add `brand` to `design/tokens/colors.json`: light `#6E2A22`, dark `#C94C3E`
- [x] Regenerate web/mobile color files
- [x] `.sidebar__brand` uses `var(--color-brand)`

## Review
Contrast: light 10.0:1 on background; dark 3.9:1 on background (3.5:1 on card). Passes WCAG AA for large text
(the wordmark), not for body text, so the token description limits it to large text and graphics.
