---
id: task_2a40
title: Audit 1: API contract consistency & stability
type: task
status: open
priority: 1
labels: [audit,api]
blocked_by: []
parent: epic_9c21
remote_task_url: null
created_at: 2026-06-06T05:06:15Z
updated_at: 2026-06-06T05:06:15Z
---
MOST TIME-SENSITIVE: these contracts freeze when the public beta starts. Audit the HTTP API surface across users/read/explore (incl. email ingest) for consistency and long-term stability.

Examine:
- OpenAPI specs: services/users/api/openapi.yaml, services/read/api/openapi.yaml, services/read/email/api/openapi.yaml, services/explore/api/openapi.yaml
- Routers: services/users/internal/handlers/router.go, services/read/content/internal/api/router.go, services/read/email/internal/api/router.go, explore routers
- pkg/api, pkg/middleware

Findings to confirm/resolve:
1. Error response shape is inconsistent: users returns {error}, read/explore return {error,message,details}, email wraps everything in {data,meta}. Pick ONE canonical envelope and document it.
2. Health check format inconsistent (email returns plain text 'OK'; others JSON). Standardise.
3. Pagination: read/explore use limit/offset; users has none. Confirm offset-based is acceptable for beta or move to cursor where lists can grow large.
4. [VERIFIED — see findings below] PATH MISMATCH: mobile client calls /api/v1/content/user/{userId} while OpenAPI documents /api/v1/users/{user_id}/contents. Confirmed real; the spec is stale, not the client.
5. Versioning: confirm every public endpoint is under /api/v1.
6. Verify OpenAPI specs actually match implemented routes (drift check).

Deliverable: a 'API contract' findings section listing each inconsistency, the proposed canonical convention, and which are beta-blocking (contract-affecting) vs cosmetic.

---

## VERIFIED FINDINGS

### F-4: Read service OpenAPI spec does not match the implemented routes (BETA-BLOCKING)
Verified 2026-06-06. The mobile client and the running server agree; the OpenAPI spec — the artifact a beta freeze locks — describes paths that do not exist.

Ground truth = route registration in services/read/content/internal/api/router.go:79-113. Everything is mounted under `/api/v1/content` (singular), user routes nested as `/user/{user_id}`.

| Real path (router + mobile client + read CLAUDE.md) | OpenAPI spec (services/read/api/openapi.yaml) |
|---|---|
| GET/POST /api/v1/content/user/{user_id}            | /api/v1/users/{user_id}/contents (L483)        |
| GET /api/v1/content/user/{user_id}/search          | /api/v1/users/{user_id}/contents/search (L602) |
| PATCH/DELETE /api/v1/content/user/{user_id}/{content_id} | /api/v1/users/{user_id}/contents/{content_id} (L661) |
| POST /api/v1/content ; GET/PUT /api/v1/content/{content_id} | /api/v1/contents ; /api/v1/contents/{id} (L291,L332) |
| POST /api/v1/content/check-duplicate               | /api/v1/contents/check-duplicates (L448)       |

Spec diverges on three independent axes: resource segment `content` (real) vs `contents` (spec); user nesting `/content/user/{id}` (real) vs `/users/{id}/contents` (spec); `check-duplicate` (real) vs `check-duplicates` (spec).

Evidence: apps/mobile/src/services/read.ts:96,144,183,218,250 call /api/v1/content/user/...; services/read/CLAUDE.md documents the singular /content/user form; services/read/api/openapi.yaml:483-754 documents the /users/.../contents form.

Drift went undetected because:
- Handler doc-comments contradict each other: user_content_handler.go:83 says /api/v1/users/:user_id/contents (wrong) but :202 says /api/v1/content/user/:user_id (right).
- Unit tests use the spec-style wrong paths and still pass (user_content_handler_test.go builds its own chi route context instead of routing through the production router); only integration_test.go:105 uses the correct /api/v1/content/user/%s. The test suite does NOT guard against this drift.

Recommendation (decision needed): client already ships the singular form, so the cheapest correct pre-freeze move is to make the SPEC match the implementation (singular /api/v1/content/user/..., check-duplicate), then fix the contradictory handler comments and re-point the unit tests through the real router so drift is caught going forward. Changing the server to the REST-conventional /users/{id}/contents form would break the shipped client and is the more expensive path. Must land before signups open — both forms can't be supported cheaply post-freeze.
