---
id: task_e1d3
title: Web: Responsive layout — tablet and mobile breakpoints
type: task
status: closed
priority: 2
labels: []
blocked_by: []
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-08T11:34:42Z
updated_at: 2026-06-14T03:29:38Z
---
Make the app usable at tablet (≥768px) and mobile (<768px) widths, where the sidebar collapses.

WHAT TO DO:
- At tablet widths (768–1023px): collapse sidebar to icon-only or a narrower rail; single-column list layout.
- At mobile widths (<768px): hide the sidebar entirely and show a bottom nav bar (or hamburger menu) replicating the mobile app's tab bar (Read, Explore, You). The You destination at mobile widths shows the You sub-menu as a full page/modal.
- All features (reading, voting, search, add-link, etc.) remain fully functional at all breakpoints.
- Article reading column remains centered and max-width constrained at all widths.

VERIFICATION (agent-testable):
1. At 1280px viewport width: sidebar is fully visible with labels; reading list shows in the main area.
2. At 768px viewport width: layout adapts (sidebar icon-only or narrower); reading list is single-column and usable.
3. At 375px viewport width: sidebar is hidden; a bottom nav bar or equivalent shows Read, Explore, You tabs.
4. At 375px, navigating to You shows the sub-menu destinations (Account, Feeds, etc.) as a page/modal.
5. All auth, read, and explore flows complete successfully at 375px width.
