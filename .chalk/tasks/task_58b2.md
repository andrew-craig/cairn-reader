---
id: task_58b2
title: Web: App shell + sidebar navigation
type: task
status: open
priority: 1
labels: []
blocked_by: []
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-08T11:33:09Z
updated_at: 2026-06-11T12:16:27Z
---
Build the persistent sidebar layout that replaces mobile's bottom tab bar. This is the chrome inside which all authenticated screens render.

WHAT TO DO:
- Persistent left sidebar on desktop (≥1024px) with nav items: Read, Explore, You.
- You item is expandable: clicking it shows sub-items Account, Feeds, Newsletters, Bookmarks, Votes, and Log out inline in the sidebar.
- Sidebar shows counts next to You sub-items where relevant (fetched via Promise.allSettled — vote stats, subscription counts by type, bookmarks total). Refresh counts when the You section is expanded.
- Active route highlighted in the sidebar.
- The sidebar collapses/hides at narrower viewports (implemented fully in the responsive layout task, but the sidebar must not break at any width).
- Search bar spanning the width of the list area, plus action buttons (Add link, etc.) to its right — visible on the /read route.

VERIFICATION (agent-testable):
1. After login, the sidebar is visible with Read, Explore, and You items.
2. Clicking Read navigates to /read; clicking Explore navigates to /explore. Active item is visually highlighted.
3. Clicking You expands the sub-menu in the sidebar showing Account, Feeds, Newsletters, Bookmarks, Votes, Log out.
4. Clicking Account in the sub-menu navigates to /you/account; sub-menu stays open.
5. Count badges appear next to sub-items (e.g. Feeds shows subscription count); counts refresh each time the You section is expanded.
6. Clicking Log out from the sidebar sub-menu clears the session and redirects to /login.
