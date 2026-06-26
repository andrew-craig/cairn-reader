---
id: chore_6dc8
title: Web: Define shared easing tokens and adopt them
type: chore
status: open
priority: 3
labels: [web,design-review,motion]
blocked_by: []
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-24T22:59:33Z
updated_at: 2026-06-24T22:59:33Z
---
The whole app has only two transitions (FAB color/opacity, sidebar chevron rotate) and both use the default 'ease', which is weak.

Add motion tokens to :root in src/index.css: --ease-out: cubic-bezier(0.23,1,0.32,1); --ease-in-out: cubic-bezier(0.77,0,0.175,1); and adopt them in the existing transitions and any new ones (press feedback, modal animations) for a single cohesive motion vocabulary. Keep UI durations under ~300ms.

From design review of apps/web. Priority 4 of 5.
