---
id: task_f1ce
title: Content Service: Document POST /api/v1/content/user/bulk in OpenAPI spec
type: task
status: open
priority: 2
labels: []
blocked_by: []
parent: epic_8461
remote_task_url: null
created_at: 2026-04-07T21:37:09Z
updated_at: 2026-04-07T21:37:09Z
---

## Context

From `docs/CONTENT_SERVICE_JWT_AUTH.md` Phase 6. The router registers `POST /api/v1/content/user/bulk` with `RequireAuth` middleware (in `services/read/content/internal/api/router.go` line 108), but this endpoint is absent from `services/read/content/api/openapi.yaml`. Only the internal variant (`/api/v1/internal/content/user/bulk`) is documented.

## Tasks

- [ ] Add `POST /api/v1/content/user/bulk` path to `services/read/content/api/openapi.yaml`:
  - Add `security: - BearerAuth: []`
  - Reference `BulkAddToUsersRequest` schema for request body
  - Reference `BulkAddToUsersResponse` schema for 200 response
  - Document 401 (missing/invalid token) and 403 (user adding to another user's account) error responses
