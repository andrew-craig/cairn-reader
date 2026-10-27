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
- [x] Frame-by-frame check in Chromium (mocked API, 390px and 1100px, Web Animations seeked per frame)

## Review
Open: ring 120ms → pill width 42px→100% (260ms ease-in-out, +80ms) carrying the icon left → input fades (+240ms) → ✕ scales/unblurs in (+280ms). Close is faster in reverse (fades 100ms, width 220ms ease-out, ring fades last). Reduced motion: width snaps, contents fade. Type-check, lint (one pre-existing warning) and vitest (58) pass. Not yet reviewed visually: Playwright's ARM64 Chromium download stalls on the Pi.

Visual check (Chromium, mocked API): found the icon pushed half out of the round pill for the first ~150ms of opening. Focusing the input (84px wide due to its 42px side padding) scrolled the clipped 42px pill sideways by 22px; fixed with `focus({ preventScroll: true })`. After the fix every frame of open and close matches the sequence above; focus returns to the search button on close, the query clears, and reduced motion transitions only box-shadow and opacity. PR review nits: reduced-motion overrides now outrank the open-state rules by specificity, not import order; the inert-toggle a11y tradeoff is documented inline.
