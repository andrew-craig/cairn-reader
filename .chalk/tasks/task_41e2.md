---
id: task_41e2
title: [Audit/Tier 1] Key rotation never reaches the users service's own validator — it signs tokens its middleware cannot verify
type: task
status: open
priority: 1
labels: [quality,security,audit,ops]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T10:13:13Z
updated_at: 2026-08-17T10:13:13Z
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
