---
id: task_f734
title: Web: Project scaffold — Vite + React + TypeScript + routing
type: task
status: closed
priority: 1
labels: []
blocked_by: []
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-08T11:32:37Z
updated_at: 2026-06-09T22:47:21Z
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

## Review (2026-06-09)

Scaffolded the desktop web app per the requirements doc and decision_9b2d ADR.

### What was built
- **Root npm workspace** (`/package.json`) covering `apps/web` + `apps/shared` only.
  `apps/mobile` is deliberately left out of the workspace so the shipping Expo
  build is untouched — verified `npm prefix` from `apps/mobile` still resolves to
  itself and `npm ci` there still installs from its own lockfile.
- **`apps/shared`** (ADR Decision 1 — extract shared package): ported the
  framework-agnostic `types/` verbatim (`article.ts`, `auth.ts`, `read.ts`; index
  excludes RN-specific `navigation.ts`) and `config/api.ts` parameterized over a
  `StorageAdapter` interface (the AsyncStorage→localStorage swap point).
- **`apps/web`**: Vite + React 19 + TS (strict) SPA. React Router with the full
  route graph (`/login`, `/read`, `/read/:id`, `/explore`, `/explore/:id`, `/you`,
  `/you/account|feeds|newsletters|bookmarks|votes`), each a stub heading. Root
  redirect `/` → `/read` (authenticated) | `/login` (otherwise); catch-all routes
  to the same redirect. localStorage `StorageAdapter` wired into shared config via
  `configureStorage()` in `main.tsx` (ADR Decision 2 — localStorage token storage).

### Scope boundaries (deferred to their own tasks)
- Real auth (`AuthContext`, token refresh, login form) — task_f61e. Scaffold uses a
  minimal token-presence check only for the root redirect.
- Design system / theme tokens / fonts — task_07ff.
- Service logic (auth/read/explore) added to `apps/shared` by the feature tasks.
- Migrating `apps/mobile` onto `apps/shared` is a follow-up (kept out to avoid
  disrupting the shipping build in a scaffold change).

### Verification
1. `npm run dev` starts cleanly; SPA serves 200 at `/`, `/read`, `/you/account`. ✓
2. All 11 route components + router compile and bundle; deep links serve 200 (no
   server 404). ✓
3. `npm run build` produces `dist/`. ✓
4. `npm run type-check` (tsc --noEmit) — zero errors. ✓
5. `npm run lint` (eslint) — zero errors. ✓

Note on backend testing: this scaffold makes no API calls yet, so there is nothing
to exercise against cairn.seatrain.net at this stage (confirmed healthy; CORS done
in task_1318). Backend/login testing becomes relevant from the auth task (f61e)
onward — login details not needed yet.
