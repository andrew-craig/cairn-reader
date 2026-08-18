---
id: bug_f53a
title: [users JWT] Access tokens are deterministic within the same second — rotation doesn't rotate
type: bug
status: closed
priority: 1
labels: [quality,security,auth-service]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-18T09:29:26Z
updated_at: 2026-08-18T11:40:05Z
---
**Discovered while enabling the users service's DB-backed test suite in CI (task_1958).** Not in the original CODE_QUALITY_REVIEW.md ledger - found by actually running these tests against real Postgres for the first time.

## Problem
services/users/internal/auth/jwt.go `GenerateToken` builds RS256 JWT claims from `time.Now()` truncated to seconds (`iat`/`nbf`/`exp`) plus fixed `iss`/`aud`/`sub`/`user_id` - there is no `jti` or any other per-call-unique claim. `jwt.SigningMethodRS256` uses PKCS#1v1.5 padding, which is **deterministic**: signing the same claim set with the same key twice produces byte-identical output.

Consequence: two tokens minted for the same user within the same wall-clock second are **byte-for-byte identical**. This is not theoretical - it reproduces deterministically (5/5 local runs) in:
- `TestAuthService_Login/success` (register then login) - services/users/internal/services/auth_service_test.go:333
- `TestAuthService_RefreshAccessToken/success` (register then refresh) - services/users/internal/services/auth_service_test.go:508

Both assert `assert.NotEqual(t, <old access token>, <new access token>)` and both fail because register+login/refresh complete inside the same second in practice, not just in tests. Refresh-token rotation still works (refresh tokens are random bytes, unaffected) but the **access token** "rotation" is a no-op whenever two tokens are issued in the same second - which undermines the security rationale for rotation (limiting exposure window of a leaked access token).

## What to do
Add a per-token-unique claim (`jti`, e.g. a new `uuid.NewString()` per call) to `Claims` in jwt.go's `GenerateToken`, matching the pattern already used for refresh tokens (which are already unique per issuance). Re-run the two tests above to confirm they pass without weakening the assertion.

## Verify
```
cd services/users
TEST_DB_HOST=localhost TEST_DB_PORT=5432 TEST_DB_USER=postgres TEST_DB_PASSWORD=postgres TEST_DB_NAME=cairn_test \
  go test -race -run 'TestAuthService_(Login|RefreshAccessToken)' -count=5 -v ./internal/services/...
```
All 5 counts should pass (this is what deterministically failed 5/5 before the fix).

## Related
- Currently a known, tracked-separately red test in CI (`test-users` job) after task_1958 lands - `TestAuthService_RefreshAccessToken/success` fails until this is fixed. `TestAuthService_Login/success` passes/fails depending on second-boundary timing luck, same root cause.

## Review

Fixed in the same PR as task_1958 (PR #335), after a reviewer flagged that shipping the CI-enablement PR with this left red would leave `test-users` perpetually failing on `main` and mask future regressions in that job.

Added `ID: uuid.NewString()` to the `jwt.RegisteredClaims` built in `GenerateToken` (`internal/auth/jwt.go`) - `uuid` was already imported. Verified via the exact repro command above: `TestAuthService_Login` and `TestAuthService_RefreshAccessToken` both pass 5/5 with `-count=5` (previously `TestAuthService_RefreshAccessToken/success` failed 5/5 deterministically). Full `go test -race -p 1 ./internal/...` suite is green. `gofmt`, `go vet`, `golangci-lint run` all clean.
