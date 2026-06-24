---
id: task_ab73
title: Web: Add :active press feedback and gate hover states for touch
type: task
status: open
priority: 2
labels: [web,design-review,motion]
blocked_by: []
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-24T22:59:32Z
updated_at: 2026-06-24T22:59:32Z
---
No pressable element gives press feedback anywhere in apps/web — zero :active states across all buttons, nav items, article rows, FAB icons, modal actions. Per Emil design-eng philosophy every pressable element should confirm it heard the user.

Add: transition: transform 160ms ease-out; + :active { transform: scale(0.97); } to .login-submit, .read__add-link, .read__refresh, .add-link-modal__btn, .reader__action, .sidebar__item, .article-row__button, .search-modal__close.

Also gate all :hover rules behind @media (hover: hover) and (pointer: fine) — touch devices fire :hover on tap and it sticks (.article-row__button:hover, .sidebar__item:hover, .you-page__link:hover, modal item hovers).

From design review of apps/web. Priority 2 of 5.
