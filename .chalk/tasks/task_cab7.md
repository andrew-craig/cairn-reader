---
id: task_cab7
title: Mobile: keep auth tokens when the server is unreachable
type: task
status: open
priority: 2
labels: [mobile,offline,auth]
blocked_by: []
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-05T23:46:28Z
updated_at: 2026-09-05T23:46:28Z
---
Prerequisite for feature_90a5 (the offline half of task_47c1). doRefreshAccessToken (apps/mobile/src/services/auth.ts:358-433) calls clearTokens() on ANY error, so opening the app offline within the 5-minute expiry buffer wipes tokens and logs the user out. Distinguish 'server rejected the refresh' (4xx: clear tokens, real logout) from 'could not reach the server' (network/timeout/5xx: keep tokens, throw). The thrown network error must NOT contain 'Session expired' or 'Not authenticated': utils/retry.ts:26-31 treats those as non-retryable and ExploreScreen.tsx:201-209 logs out on them. If task_47c1's shared auth consolidation lands first, this collapses into consuming that module. Verify: unit test that an offline refresh keeps tokens and throws a retryable error; 401 refresh still clears tokens and notifies listeners.
