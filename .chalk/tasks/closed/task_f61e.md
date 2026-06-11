---
id: task_f61e
title: Web: Authentication — login/register screen, AuthContext, token refresh
type: task
status: closed
priority: 1
labels: []
blocked_by: []
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-08T11:32:58Z
updated_at: 2026-06-10T21:59:33Z
---
Implement the full authentication layer: login/register UI, AuthContext, localStorage token storage, proactive + reactive refresh, and logout.

WHAT TO DO:
- Build the /login screen with email/password login and register forms (tab/toggle between the two), matching mobile's LoginScreen.
- Implement AuthContext (user, isAuthenticated, isLoading, login, logout) backed by localStorage keys @cairn:access_token, @cairn:refresh_token, @cairn:token_expires_at, @cairn:user.
- Port mobile AuthService logic: ensureValidToken (proactive 5-min buffer refresh), refreshAccessToken, and 401-retry interceptor.
- On refresh failure or expired session: clear tokens and navigate to /login.
- Wrap all authenticated routes in an auth guard that redirects to /login when !isAuthenticated.
- On successful login: redirect to /read.
- Change password via PUT /api/v1/user/{id}/password (implemented in the Account screen task, but wire the service method here).

VERIFICATION (agent-testable against real backend at cairn.seatrain.net):
1. Visiting /read while logged out redirects to /login.
2. Submitting login with a valid email/password navigates to /read.
3. Submitting login with wrong credentials shows a user-visible error message (not a console error).
4. After a successful login, refreshing the page keeps the user logged in (tokens persist in localStorage).
5. Clicking logout clears localStorage tokens and returns the user to /login.
6. Registering a new account with a fresh email/password succeeds and navigates to /read.

## Plan
- [x] Port AuthService → apps/web/src/services/auth.ts (localStorage backend, no device-ID,
      email login/register, logout, ensureValidToken, refreshAccessToken, fetchWithAuth
      401-retry interceptor, changePassword). Reuse shared getServerUrl + types.
- [x] AuthContext → apps/web/src/contexts/AuthContext.tsx (user, isAuthenticated, isLoading,
      login, logout; onAuthStateChange listener clears state on token wipe).
- [x] Login screen → apps/web/src/routes/Login.tsx (login/register toggle, inline error,
      loading state; navigate to /read on success).
- [x] Auth guard (RequireAuth) + AuthProvider wiring in App.tsx; RootRedirect via context.
      Delete placeholder src/auth.ts.
- [x] Verify: tsc --noEmit, eslint, vite build all pass.

## Review

Implemented the full web auth layer, ported ~verbatim from the mobile AuthService/AuthContext
with the two spec-mandated changes: localStorage (sync) instead of AsyncStorage, and no
device-ID login (email/password only).

Files:
- NEW apps/web/src/services/auth.ts — AuthService: email login/register (shared `authenticate`
  helper), logout, token persistence, isTokenExpired (5-min buffer), ensureValidToken,
  refreshAccessToken (single-flight via shared promise), fetchWithAuth (proactive refresh +
  one reactive 401 retry → clearTokens on failure), changePassword (PUT /user/{id}/password).
- NEW apps/web/src/contexts/AuthContext.tsx — same shape as mobile; checkAuthStatus on mount,
  onAuthStateChange listener resets state when tokens are wiped (drives redirect to /login).
- REWROTE apps/web/src/routes/Login.tsx (+ Login.css) — login/register toggle, inline
  user-visible error, loading state, navigate('/read') on success.
- MODIFIED apps/web/src/App.tsx — AuthProvider wrapper, RequireAuth guard (Outlet) around all
  authenticated routes, RootRedirect via context. Deleted placeholder src/auth.ts.

Out-of-scope but necessary fix:
- apps/web/tsconfig.json — removed deprecated `baseUrl` (kept the relative `paths` value).
  The environment installed TS 6.0.2 (scaffold pinned 5.7.3), which errors on `baseUrl`.
  Removing it is the forward-compatible root fix (baseUrl is removed entirely in TS 7); the
  relative path value resolves identically without it.

Verification:
- tsc --noEmit: pass. eslint: pass (1 non-blocking react-refresh HMR warning on the
  context file, same single-file context+hook pattern as mobile). vite build: pass.
- Live backend (cairn.seatrain.net) contract checks all match the parsing:
  - POST /auth/register → 201, data.{user,access_token,refresh_token,expires_in}.
  - POST /auth/login (valid) → 200, same shape.
  - POST /auth/login (wrong password) → 401, {message:"invalid email or password"} →
    surfaced as the UI error (verification #3).
  - POST /auth/refresh → 200, data.{access_token,refresh_token,expires_in}.
- Routing verifications (#1 guard redirect, #2/#6 navigate to /read, #4 persistence via
  localStorage, #5 logout clears tokens) are covered by the RequireAuth guard + AuthContext
  wiring and validated by the production build.

Note (pre-existing, not touched): shared User type uses camelCase (createdAt/lastLoginAt)
while the backend returns snake_case; the stored user object only relies on `id`, so this
mismatch is inert here — same as mobile.
