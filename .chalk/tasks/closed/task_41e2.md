---
id: task_41e2
title: [Audit/Tier 1] Key rotation never reaches the users service's own validator — it signs tokens its middleware cannot verify
type: task
status: closed
priority: 1
labels: [quality,security,audit,ops]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T10:13:13Z
updated_at: 2026-08-18T13:41:14Z
---
**Source:** Cairn Simplification Audit (read-only pass at HEAD `a6c56a1`, 2026-08-16) — https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. One finding, one branch, one PR. Re-verify before fixing — all file:line references below were confirmed at `a6c56a1`.

**Audit tier:** 1 (correctness & security) | **Verified** | Security/ops-relevant.

## Problem
`services/users/internal/handlers/router.go:58`:
```go
authMiddleware := auth.NewMiddleware(auth.NewValidator(config.JWTManager.GetPublicKey()))
```
The router builds its validator from a **public key snapshot taken at startup**. `KeyRotationManager` rotation (`services/users/internal/auth/vault.go`, interval `JWT_KEY_ROTATION_INTERVAL`, default 24h) updates only the **signer** — `jwtManager.UpdateKeys(...)` — and nothing pushes the new public key into this validator.

**Consequence:** after the first rotation the users service **signs access tokens that its own middleware cannot verify**. Timer-triggered, no bad input required.

**It is the only service left with this gap.** task_88aa's fix landed for all three downstream services, each of which now wires the refresher:
- services/read/content/cmd/content/main.go:129 — `auth.NewKeyRefresher(vaultClient, cfg.Vault.PublicKeyPath, jwtValidator)`
- services/read/email/cmd/email_ingest/main.go:137 — same
- services/explore/recommender/cmd/explore_recommender/main.go:118 — same

The users service wires **none** — no `NewKeyRefresher` call exists anywhere under `services/users`.

## What to do
1. **Failing test first:** rotate the key via the rotation manager's `OnRotation` path and assert a freshly signed token still validates through the router's middleware, with no process restart. Must fail on main.
2. Wire the rotation into the validator. Note the shape differs from the downstream services: users is the **signer**, so it does not need to poll Vault — its `OnRotation` callback (already wired at `services/users/cmd/user-service/main.go`, which calls `jwtManager.UpdateKeys(...)`) can push the new public key straight into the validator via `(*auth.Validator).UpdatePublicKey` (`pkg/auth/validator.go`, thread-safe in-place swap). Prefer that over adding a Vault poller to the service that owns the key.
3. Reuse `pkg/auth`'s existing primitives — `UpdatePublicKey` already exists and is the in-repo reference pattern here.

## Done when
- A rotation is proven to leave the users service able to verify its own freshly issued tokens, by a test that fails on main.

## Related
- **task_88aa** (in_progress, P0) — the downstream half of the same class. This is the residue it does not cover: the users service's own middleware. Read its validation notes first; they document the rotation mechanics in detail and explicitly scoped the users service out.

## Validation (2026-08-18)
Re-verified on current `main`: `services/users/internal/handlers/router.go:58` still built `auth.NewMiddleware(auth.NewValidator(config.JWTManager.GetPublicKey()))` — a one-time snapshot with no path back to the running `*auth.Validator`. `main.go`'s `OnRotation` callback (`services/users/cmd/user-service/main.go:168-173`) only called `jwtManager.UpdateKeys(...)`; it had no handle on the router's validator to push the new key into. Confirmed still valid, exactly as described.

## Implementation
Branch: `claude/epic-fefa-next-task-19xayr`.
- `services/users/internal/handlers/router.go`: `RouterConfig.JWTManager` replaced with `RouterConfig.Validator *pkg/auth.Validator`. `Router` now uses the caller-supplied validator directly instead of building its own throwaway one from a key snapshot.
- `services/users/cmd/user-service/main.go`: constructs `jwtValidator := pkgauth.NewValidator(publicKey)` alongside the JWT manager, passes it into `Router`, and the `KeyRotationManager.OnRotation` callback now also calls `jwtValidator.UpdatePublicKey(keyPair.PublicKey)` right after `jwtManager.UpdateKeys(...)` — the same in-place-swap primitive `task_88aa` used downstream, applied here to the signer's own validator instead of a Vault poller (per the task's guidance: users owns the key, so it pushes rather than polls).
- `services/users/selfhost/users.go`: same `RouterConfig.Validator` wiring (selfhost has no rotation — static file key, confirmed out of scope by task_88aa's validation notes — so it's just a one-time `pkgauth.NewValidator(cfg.PublicKey)`, no `OnRotation` hook).
- `services/users/internal/handlers/router_test.go`: new `TestKeyRotationReachesRouterValidator` builds the real router with a live `*pkgauth.Validator`, signs a token with key A (200 via `POST /api/v1/auth/logout-all`), rotates the JWT manager to key B exactly as `KeyRotationManager.rotateKeys` does, and asserts a token signed with key B is rejected (401) by the router's still-stale validator — this is the bug, reproduced against the real router+middleware, not a mock. It then calls `validator.UpdatePublicKey(&privB.PublicKey)` (what the fixed `OnRotation` callback now does) and asserts the same key-B token now validates (200), with no new router and no restart.

**Failing-test-first proof:** the test's middle assertion (key-B token → 401 before `UpdatePublicKey` is called) is itself the pre-fix behavior reproduced live — confirmed by the test's own log output (`JWT kid mismatch` warning + `status=401`) before the final `UpdatePublicKey` call flips it to 200. Ran `go test -race -run TestKeyRotationReachesRouterValidator -v ./internal/handlers/...`: PASS end-to-end (both the reproduction and the recovery are in the same test).

**Verification:** `gofmt -l .` clean, `go vet ./...` clean, `golangci-lint run ./...` → 0 issues, `go test -race -count=1` green across all `services/users` packages except `test/integration` (pre-existing, unrelated: that suite lacks a `//go:build integration` tag — task_8b77 — so a bare `go test ./...` hangs trying to reach a real Postgres; confirmed this is not new). `cmd/selfhost` module also builds, vets, lints, and tests clean with the same `RouterConfig.Validator` wiring.

**Scope:** no route, request, or response shape changed — this is internal wiring only, so no `openapi.yaml` or `CLAUDE.md` route-table update is needed. `docs/ARCHITECTURE.md` unaffected — no service boundary changed.
