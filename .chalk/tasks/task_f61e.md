---
id: task_f61e
title: Web: Authentication — login/register screen, AuthContext, token refresh
type: task
status: open
priority: 1
labels: []
blocked_by: [task_f734]
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-08T11:32:58Z
updated_at: 2026-06-08T11:32:58Z
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
