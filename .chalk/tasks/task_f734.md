---
id: task_f734
title: Web: Project scaffold — Vite + React + TypeScript + routing
type: task
status: open
priority: 1
labels: []
blocked_by: []
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-08T11:32:37Z
updated_at: 2026-06-09T22:28:19Z
---
Bootstrap apps/web as a Vite + React + TypeScript SPA with React Router and the full route graph from the requirements doc.

WHAT TO DO:
- Create apps/web with Vite react-ts template.
- Add React Router; wire routes: /login, /read, /read/:id, /explore, /explore/:id, /you, /you/account, /you/feeds, /you/newsletters, /you/bookmarks, /you/votes.
- Each route renders a stub component (just a heading) — content filled in later tasks.
- Add top-level redirect: / → /read (authenticated) or /login (unauthenticated).
- Configure tsc strict mode; ensure 'npm run type-check' (tsc --noEmit), 'npm run lint', and 'npm run build' all pass with zero errors.
- Add apps/web to the workspace (package.json workspaces or equivalent).
- Port/copy types from apps/mobile/types/ (article.ts, read.ts, auth.ts) — either from shared package or directly, per decision_9b2d.
- Port config/api.ts replacing AsyncStorage with localStorage.

VERIFICATION (agent-testable):
1. cd apps/web && npm run dev starts without errors; app is accessible at localhost:5173.
2. Navigating to /login, /read, /explore, /you each renders the stub for that route (no 404, no blank screen).
3. npm run build completes without errors and produces dist/.
4. npm run type-check passes with zero TypeScript errors.
5. npm run lint passes with zero errors.
