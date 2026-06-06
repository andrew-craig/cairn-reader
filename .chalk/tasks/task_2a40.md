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
4. PATH MISMATCH RISK: mobile client appears to call /api/v1/content/user/{userId} while OpenAPI documents /api/v1/users/{user_id}/contents. Verify which is real; reconcile spec vs implementation vs client. This is the kind of thing that must be fixed before freeze.
5. Versioning: confirm every public endpoint is under /api/v1.
6. Verify OpenAPI specs actually match implemented routes (drift check).

Deliverable: a 'API contract' findings section listing each inconsistency, the proposed canonical convention, and which are beta-blocking (contract-affecting) vs cosmetic.
