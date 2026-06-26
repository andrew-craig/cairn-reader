---
id: task_3a07
title: Web: Animate modal enter/exit and add prefers-reduced-motion
type: task
status: open
priority: 2
labels: [web,design-review,motion]
blocked_by: []
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-24T22:59:33Z
updated_at: 2026-06-24T22:59:33Z
---
SearchModal and AddLinkModal are conditionally mounted ({showSearch && <SearchModal/>}) with no entrance or exit — they snap in/out, reading as a glitch.

Add: backdrop opacity 0->1; card opacity 0->1 + transform scale(0.96)->scale(1) over ~200ms ease-out (cubic-bezier(0.23,1,0.32,1)). Keep transform-origin: center (these are centered modals, not trigger-anchored popovers). Never start from scale(0). Fade the backdrop (optional backdrop-filter: blur(2px) — already used on the FAB). Add an exit transition (@starting-style or brief unmount delay), exit faster than enter (~150ms).

Add @media (prefers-reduced-motion: reduce) across the app: keep opacity fades, drop scale/transform motion. None exists today.

From design review of apps/web. Priority 3 of 5.
