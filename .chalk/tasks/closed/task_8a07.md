---
id: task_8a07
title: Web: Visual polish pass (You-hub duplication, key-value rows, image frames, dark palette, FAB labels)
type: task
status: closed
priority: 3
labels: [web,design-review]
blocked_by: []
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-24T22:59:33Z
updated_at: 2026-06-28T23:07:07Z
---
Lower-priority polish from the apps/web design review:

- /you hub duplicates the sidebar sub-menu in the main pane on desktop (same 7 items twice). Redirect /you -> /you/account on desktop widths, or render a distinct landing pane.
- Key-value rows (Feeds, About, You) use justify-content: space-between across the full column, leaving a wide void between label and right-side action/value. Cap inner row width or move the value closer.
- Empty/failed article image frames are near-invisible in light mode (--color-card #fbfaf9 ~= --color-background #fdfcfc). Add a faint border or placeholder glyph. (Verify on a real browser first — many blank thumbnails in review were external-CDN images unreachable through the screenshot proxy.)
- Dark palette: --color-card #1c1c1e (cool) clashes with the warm dark background hsl(40,21%,9%); nudge the card warm.
- Reader FAB: 6 icon-only unlabeled actions — add tooltips (instant-on-subsequent-hover) or labels.
- Consider a confirm step for Unsubscribe (one click loses a feed).

From design review of apps/web. Priority 5 of 5.
