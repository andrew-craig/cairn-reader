---
id: task_ed6a
title: Web: morph the round search button into a pill search field
type: task
status: open
priority: 2
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-10-25T11:01:40Z
updated_at: 2026-10-25T11:01:40Z
---


## Plan
- [x] One pill element in the header: closed it is the 42px round search button
- [x] Open sequence: border fades in → pill elongates leftward (right edge anchored) carrying the search icon to the leading position → input fades in → ✕ appears in the icon's original spot
- [x] Close: faster reverse; Escape / ✕ close and return focus to the search button
- [x] Reduced motion: no width motion, opacity only
- [x] Tests updated; type-check, lint, vitest
- [ ] Frame-by-frame check in Chromium (blocked: no working headless Chromium on the dev Pi)

## Review
Open: ring 120ms → pill width 42px→100% (260ms ease-in-out, +80ms) carrying the icon left → input fades (+240ms) → ✕ scales/unblurs in (+280ms). Close is faster in reverse (fades 100ms, width 220ms ease-out, ring fades last). Reduced motion: width snaps, contents fade. Type-check, lint (one pre-existing warning) and vitest (58) pass. Not yet reviewed visually: Playwright's ARM64 Chromium download stalls on the Pi.
