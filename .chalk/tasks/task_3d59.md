---
id: task_3d59
title: Offline mobile: keep auth tokens when server is unreachable (piece 1/7)
type: task
status: open
priority: 3
labels: [mobile,offline,auth]
blocked_by: []
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-05T23:13:20Z
updated_at: 2026-09-05T23:13:20Z
---
Prerequisite for offline reading (feature_90a5). Today doRefreshAccessToken's catch calls clearTokens() on ANY error including a plain network failure (apps/mobile/src/services/auth.ts:358-433), so opening the app offline within the 5-min token-expiry buffer forces a full logout and wipes tokens. AuthContext.checkAuthStatus hits this on every cold start.

Scope:
- In doRefreshAccessToken, distinguish 'server rejected the refresh' (4xx -> clear tokens, real logout) from 'could not reach the server' (network/timeout/5xx -> keep tokens, throw a retryable error).
- Preserve the load-bearing error message strings that apps/mobile/src/utils/retry.ts:19-37 matches on ('session expired', 'not authenticated', etc.).
- Fix ExploreScreen.loadExploreArticles (ExploreScreen.tsx:201-210) which calls logout() on any error whose message matches auth substrings — must not log out on a transient/offline error.

Coordinate with task_47c1 (FE auth-layer consolidation into apps/shared). If task_47c1 lands first, this collapses to consuming the shared module; otherwise do the minimal fix here.

Verify: unit test — offline refresh keeps tokens and throws a retryable error; 401/invalid-refresh still clears tokens and fires the auth-state listener. ExploreScreen test — transient error does not call logout().
