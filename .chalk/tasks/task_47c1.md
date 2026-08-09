---
id: task_47c1
title: [FE auth layer] Move the duplicated web/mobile auth.ts into apps/shared; fix H12 + offline-clears-tokens
type: task
status: open
priority: 2
labels: [quality,wave4,consolidation,frontend]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:53:56Z
updated_at: 2026-08-09T06:53:56Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** Theme 3 (FE auth) + H12 + network-failure-clears-tokens | **Wave 4** | **Recipe:** R11 step 4 (strategy §2.5)
**Touches:** apps/web/src/services/auth.ts, apps/mobile/src/services/auth.ts, apps/shared, apps/mobile/src/services/read.ts + explore.ts

## Problem
`apps/web/src/services/auth.ts` is a near-verbatim copy of `apps/mobile/src/services/auth.ts` — the whole token-refresh state machine, duplicated. `apps/shared` already demonstrates the right injectable-adapter pattern (the server-URL logic), so the pattern exists; it just was not used here.

Two live bugs sit inside the duplicated code:
- **H12 (mobile):** a second 401 after token refresh is silently swallowed (`services/read.ts:50-72`, `explore.ts:86-108`) — only a *thrown* refresh failure logs the user out, so a re-rejected retry leaves the user stuck in a broken authenticated UI.
- **Both platforms:** any error in `doRefreshAccessToken`, **including plain offline**, calls `clearTokens()`. Opening the app offline near token expiry forces a full re-login instead of a retry.

## What to do
1. Port the web and mobile `auth.ts` tests onto the shared module **first**, using the `apps/shared` injectable-adapter pattern for the platform-specific storage/fetch bits.
2. Fix H12 and the offline-clears-tokens bug **in the shared copy**, so both platforms inherit the fix. Distinguish "refresh rejected by the server" from "refresh failed to reach the server".
3. Repoint web and mobile; delete both copies in the same PR as the last repoint.
4. The refresh-dedup mutex, stale-while-revalidate caching and optimistic-update rollback guards are **correct today** — preserve their behavior, do not redesign them.

## Done when
- One auth/token-refresh implementation lives in `apps/shared`; tests cover the second-401 and offline cases and fail on the old behavior.

