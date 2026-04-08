---
id: task_15ee
title: Content Service: Add BulkAddToUsers auth tests
type: task
status: closed
priority: 2
labels: []
blocked_by: []
parent: epic_8461
remote_task_url: null
created_at: 2026-04-07T21:37:09Z
updated_at: 2026-04-07T22:45:39Z
---

## Context

From `docs/CONTENT_SERVICE_JWT_AUTH.md` Phase 5.1/5.2. The bulk handler tests in `services/read/content/internal/api/handlers/bulk_handler_test.go` only call `BulkAddToUsersInternal`. The authenticated `BulkAddToUsers` handler (registered at `POST /api/v1/content/user/bulk`) is completely untested.

## Tasks

- [ ] Add tests for `BulkAddToUsers` in `bulk_handler_test.go`:
  - Success: authenticated user adds content to their own account → 200
  - Forbidden: authenticated user tries to add content to a different user's account → 403
  - Missing auth context (internal error path) → 500
- [ ] Use `addAuthContextToRequest` helper (already in `user_content_handler_test.go`) or extract it to a shared test helper
