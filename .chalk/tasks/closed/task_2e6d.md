---
id: task_2e6d
title: Content Service: Add JWT integration tests (full request flow)
type: task
status: closed
priority: 2
labels: []
blocked_by: []
parent: epic_8461
remote_task_url: null
created_at: 2026-04-07T21:37:09Z
updated_at: 2026-04-22T12:56:07Z
---

## Context

From `docs/CONTENT_SERVICE_JWT_AUTH.md` Phase 5.3. No integration tests exist for the JWT auth flow end-to-end. The unit tests mock auth context injection; integration tests should exercise the full middleware → handler pipeline with real (or test) JWT tokens.

## Tasks

- [ ] Add integration tests (tagged `// +build integration`) in `services/read/content/integration_test.go` covering:
  - Full request to a protected endpoint with a valid JWT → 200
  - Full request with missing `Authorization` header → 401
  - Full request with expired JWT → 401
  - Full request with a valid JWT but wrong user ID in URL → 403
- [ ] Generate test RSA key pair in test setup (no Vault dependency needed for unit-level integration tests)
- [ ] Verify Vault integration works if `VAULT_ADDR` is set (can be skipped in CI without Vault)
