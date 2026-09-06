---
id: chore_1089
title: Mobile: route every raw fetch through NetworkError conversion, and lint against new ones
type: chore
status: open
priority: 2
labels: [mobile,offline,auth]
blocked_by: [task_c87c]
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-06T07:46:07Z
updated_at: 2026-09-06T07:46:07Z
---
Structural guard for a bug class that has now recurred three times across task_cab7 and task_c87c. Converting a failed fetch into a NetworkError is done per-call-site, so every unwrapped fetch is a latent instance of 'a dropped connection is mistaken for a rejected credential'. Two of the three found so far logged the user out; each was found by review, one at a time, never by a test or a rule. There is currently nothing stopping a fourth. Verified on main: 9 raw fetch call sites remain in apps/mobile/src/services/auth.ts (login/mobile:95, register/mobile:123, login:149, register:175, logout:203, refresh:382 which has its own inline handling from task_cab7, upgrade:569, password:600) plus system.ts:13 (unauthenticated health check). task_c87c wrapped only the two in fetchWithAuth (the primary request and the 401 retry) because those were the two that caused a logout. The rest are lower severity but not harmless: AccountScreen surfaces error.message directly in an Alert, so an offline password change or account upgrade currently shows the user a raw 'Network request failed'. Scope: (1) promote AuthService.fetchOrNetworkError to a shared helper so system.ts can use it too. Likely apps/mobile/src/utils/http.ts, keeping the services-to-utils import direction intact and avoiding a cycle; NetworkError already lives in utils/errors.ts. (2) Route all remaining call sites through it. doRefreshAccessToken already distinguishes network from rejection inline and its 401/403-only rule is settled by task_cab7, so refactor it only if the result is provably identical, otherwise leave it and say why. (3) Add an ESLint rule banning bare fetch under apps/mobile/src/services (no-restricted-globals or no-restricted-syntax) with a message naming the helper, so the next call site fails lint rather than review. Verify: existing suite stays green; a test asserts each newly wrapped path rejects with NetworkError rather than TypeError; a deliberately added bare fetch in a service fails lint. Do not change any 4xx/credential-rejection semantics anywhere. This is an error-type and lint change only.
