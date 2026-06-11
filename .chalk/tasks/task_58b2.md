---
id: task_58b2
title: Web: App shell + sidebar navigation
type: task
status: in_progress
priority: 1
labels: []
blocked_by: []
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-08T11:33:09Z
updated_at: 2026-06-11T22:15:22Z
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

---

## Review (implementation)

### Changes
- `apps/web/src/components/AppLayout.tsx` + `AppLayout.css` — authenticated shell:
  persistent left sidebar + main content `<Outlet />`. Search bar + "Add link"
  quick-action row shown only on `/read` (search/add modals land in later tasks).
- `apps/web/src/components/Sidebar.tsx` — Read / Explore / You nav. You is an
  expandable toggle revealing Account, Feeds, Newsletters, Bookmarks, Votes, Log out.
  Counts fetched via `Promise.allSettled` (independent failure tolerance) and
  refreshed every time the You section is (re-)expanded. Active route highlighted
  via react-router `NavLink` `.active`; the You toggle self-highlights on `/you*`.
- `apps/web/src/services/read.ts`, `explore.ts` — minimal web ports of
  `ReadService.listUserContents` / `listAllSubscriptions` and
  `ExploreService.getUserVoteStats`, built on the web `AuthService.fetchWithAuth`
  (proactive refresh + 401 retry). Later tasks extend these.
- `apps/web/src/utils/helpers.ts` — `pluralize` (ported from mobile).
- `apps/web/src/App.tsx` — authenticated routes now nest inside `<AppLayout />`.

### Verification — live against cairn.seatrain.net (Chromium / Playwright, 11/11)
A throwaway Playwright harness drove the built app (vite preview) against the real
backend; all spec checks passed, then the harness was removed (no browser dep left
in the repo). Results:
1. Login → redirect to `/read`; sidebar shows Read/Explore/You. ✓
2. Read↔Explore navigation + active highlight; `/read` toolbar shows, hidden elsewhere. ✓
3. You expands → Account, Feeds, Newsletters, Bookmarks, Votes, Log out. ✓
4. Account → `/you/account`, sub-menu stays open + active. ✓
5. Feeds badge shows seeded subscription count (**2**); re-expanding You re-hits the
   vote-stats endpoint (counts refresh). ✓
6. Log out clears `localStorage` tokens + redirects to `/login`. ✓

Also verified: CORS preflight on the API returns `access-control-allow-origin: *`
(browser origin works); `tsc --noEmit`, `eslint`, and `vite build` all pass.

### Notes / known limitations
- **Bookmarks badge shows 0** even with a saved page. The content-list endpoint's
  `pagination` has no `total` field, so `total_count` resolves to 0 — this matches
  mobile's identical transform (`pagination.total || 0`), i.e. a shared backend
  limitation, not a regression introduced here.
- Responsive collapse is only stubbed (sidebar narrows, never breaks); the full
  collapsed/mobile treatment is the separate responsive-layout task.

### Reusable test account (seatrain.net)
- email: `cairn.web.test@seatrain.net` / password: `CairnWebTest!2026`
- Seeded: 2 RSS feed subscriptions (Hacker News, The Verge) + 1 saved page (example.com).
