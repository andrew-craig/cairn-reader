---
id: task_9df8
title: [Explore auth] explore article-inject + fetch/sync trigger endpoints unauthenticated
type: task
status: open
priority: 1
labels: [quality,security,wave1,auth]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:43:57Z
updated_at: 2026-08-09T06:43:57Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** Theme 1 (explore) | **Wave 1** | **Recipe:** R1 (strategy §2.5) | **Test level:** httptest against the real router constructor
**Touches:** services/explore/recommender router, services/explore/fetcher router, both openapi.yaml files

## Problem
`POST /api/v1/explore/article` and the fetcher's `/fetch` / `/sync` trigger endpoints have no service-to-service auth. Anyone who can reach them can inject articles into the shared explore catalog or hammer the fetch triggers. Same root cause as C1/C2 and the read/fetcher IDOR: "internal" is a comment, not an enforced boundary.

## What to do
1. Failing test first: httptest against the real router constructors, no credentials.
2. Apply `RequireInternalAPIKey` at the route group. Reference: `/api/v1/internal/*` in read/content's router.
3. Update the internal callers to send the key; update both `openapi.yaml` files and the service CLAUDE.md route tables.
4. Add the router-inventory allowlist test (R1 step 5) to both routers.

## Done when
- Both routers reject uncredentialed requests to the trigger/inject endpoints, proven by test.

