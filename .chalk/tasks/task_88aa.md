---
id: task_88aa
title: [P2-C1] Downstream services never refresh the JWT public key → scheduled auth outage
type: task
status: in_progress
priority: 0
labels: [quality,security,wave1,ops]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:41:40Z
updated_at: 2026-08-09T06:55:57Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** P2-C1 | **Wave 1** | **Recipe:** R6 (strategy §2.5) | **Test level:** unit with a fake key source
**Touches:** services/read/content/cmd/content/main.go, explore-recommender + email-ingest mains, the shared Vault client

## Problem
The users service rotates its JWT signing key every `JWT_KEY_ROTATION_INTERVAL` (default 24h, `services/users/internal/config/config.go:123`, `KeyRotationManager`). content, explore-recommender and email-ingest each call `vaultClient.GetPublicKey()` **exactly once at startup** and never refresh (`services/read/content/cmd/content/main.go:100-117` and equivalents). After the first rotation every access token fails signature verification on those three services until they are manually restarted. Timer-triggered outage, no bad input required — highest-impact operational finding in the review.

## What to do
1. Add periodic public-key re-fetch (interval much smaller than the rotation interval) plus a re-fetch-once on signature-validation failure, to handle out-of-band rotation.
2. Apply it in all three downstream services; put the logic in one place rather than three copies.

## Done when
- Test with a fake key source: rotate the key, assert validation recovers without a process restart.
- Test fails before the fix.

## Validation (2026-08-09, pre-implementation)

**Verdict: STILL VALID, in full.** No commit since 2026-07-05 touches any of the relevant files (`git log --since=2026-07-05` over `services/users/internal/auth/vault.go`, `services/users/internal/config/config.go`, the three downstream `main.go`s, and `pkg/auth/{validator,vault,middleware}.go` returns nothing). The bug is exactly as described.

### Q1 — Users service rotates the key on a timer, and it's wired in (not dead code)
Confirmed live, not dead code.
- `services/users/internal/config/config.go:123` — `KeyRotationInterval: env.GetDuration("JWT_KEY_ROTATION_INTERVAL", 24*time.Hour)`.
- `services/users/internal/auth/vault.go:279-421` — `KeyRotationManager` struct + `Start`/`rotateKeysLoop`/`rotateKeys`. `rotateKeysLoop` (line 347) runs a `time.NewTicker(krm.rotationInterval)` loop calling `rotateKeys()`, which calls `vaultClient.RefreshJWTKeys(...)` and **replaces `currentKeyPair` wholesale** (line 398-402) — no previous key retained.
- Wired into the real startup path at `services/users/cmd/user-service/main.go:169-188`: `auth.NewKeyRotationManager(...)` is constructed with `RotationInterval: cfg.JWT.KeyRotationInterval`, and its `OnRotation` callback calls `jwtManager.UpdateKeys(...)` — then `keyRotationManager.Start(ctx)` is called unconditionally at line 187. Default 24h rotation is live in the real binary, not just in tests.

### Q2 — Do the three downstream services fetch the key exactly once? Yes, all three, no refresh of any kind.
- **content**: `services/read/content/cmd/content/main.go:100-123`. `vaultClient, err := auth.NewVaultClient(vaultCfg)` → `publicKey, err := vaultClient.GetPublicKey(cfg.Vault.PublicKeyPath)` (line 113) → `jwtValidator := auth.NewValidator(publicKey)` (line 122), passed once into the router. No goroutine, no ticker, no retry-on-401 anywhere in the file.
- **explore-recommender**: `services/explore/recommender/cmd/explore_recommender/main.go:87-111`. Same shape: `vaultClient.GetPublicKey(cfg.Vault.PublicKeyPath)` (line 103) → `validator := auth.NewValidator(publicKey)` (line 110), passed once into `api.NewServer(...)`.
- **email-ingest**: `services/read/email/cmd/email_ingest/main.go:95-130`. Same shape: `vaultClient.GetPublicKey(cfg.Auth.JWTPublicKeyPath)` (line 110) → `jwtAuth := middleware.NewJWTAuth(publicKey)` (line 130), where `NewJWTAuth` (`services/read/email/internal/api/middleware/auth.go:16-22`) just wraps `auth.NewValidator`+`auth.NewMiddleware` around the static key.
- No cache TTL, no periodic re-fetch, no re-fetch-on-validation-failure, no JWKS-style multi-key lookup in any of the three, and none in the shared `pkg/auth` package they all depend on (see Q3).

### Q3 — Nothing masks the outage
- `pkg/auth.VaultClient.GetPublicKey` (`pkg/auth/vault.go:89-107`) does a synchronous, uncached Vault read every call — it is simply never called again after startup by the three services, not silently refreshed underneath them.
- `pkg/auth.Validator` (`pkg/auth/validator.go`) holds exactly one `*rsa.PublicKey` (line 39) protected by a mutex, with an `UpdatePublicKey` method (line 227) that swaps it atomically — but nothing in content/explore-recommender/email-ingest ever calls it. `ValidateToken` (line 121) checks the token's `kid` header against the validator's current `keyID` but **only logs a warning on mismatch** (line 144) — it still attempts verification with the single stored key, so there is no fallback/multi-key lookup and no grace window.
- The rotation manager itself keeps no overlap: `rotateKeys()` (`services/users/internal/auth/vault.go:391-416`) replaces `currentKeyPair` in one atomic swap with no dual-valid-key period.
- No scheduled process restarts exist anywhere in the compose files that could accidentally paper over this (`infrastructure/docker/dev/docker-compose.yml` and `infrastructure/docker/prod/docker-compose.yml` only use `restart: unless-stopped`, i.e. crash-restart, not a timer).
- Aside (does not change the verdict, scoping note only): the self-host combined binary (`cmd/selfhost/main.go:35-60`) is a separate deploy path that loads the JWT key from a local file via `auth.FileKeyProvider`, not Vault, and never starts a `KeyRotationManager` — so self-host mode has no key rotation at all and is not exposed to this specific bug. The bug is specific to the multi-container Vault-backed deployment (the one `services/*/cmd/*/main.go` files target), which is what P2-C1 describes.

### Q4 — No fix has landed
`git log --since=2026-07-05 --oneline -- services/users/internal/auth/vault.go services/users/internal/config/config.go services/read/content/cmd/content/main.go services/explore/recommender/cmd/explore_recommender/main.go services/read/email/cmd/email_ingest/main.go pkg/auth/validator.go pkg/auth/vault.go pkg/auth/middleware.go` returns **no commits**. Only doc/styling/mobile commits landed since the review (#301-#306); the ledger's Appendix note ("essentially none of it has been remediated") holds for this finding specifically.

### Q5 — Verdict: STILL VALID (not partial, not stale)

### Corrected file:line references for the implementer
- Rotation source of truth: `services/users/internal/auth/vault.go:279-421` (`KeyRotationManager`), interval default at `services/users/internal/config/config.go:123`, wiring at `services/users/cmd/user-service/main.go:169-188`.
- content: `services/read/content/cmd/content/main.go:100-123` (Vault client at 100, `GetPublicKey` at 113, `NewValidator` at 122).
- explore-recommender: `services/explore/recommender/cmd/explore_recommender/main.go:87-111` (Vault client at 87, `GetPublicKey` at 103, `NewValidator` at 110).
- email-ingest: `services/read/email/cmd/email_ingest/main.go:95-130` (Vault client at 95, `GetPublicKey` at 110, `NewJWTAuth` at 130) — plus its wrapper `services/read/email/internal/api/middleware/auth.go:16-29`. Note there is also a second, non-Vault entrypoint at `services/read/email/selfhost/email.go` and `cmd/selfhost/main.go` — out of scope for this finding (see Q3 aside).

### Shape of the fix / complications for the implementer
- The mechanism to fix this already exists and doesn't require new plumbing: `pkg/auth.Validator.UpdatePublicKey(*rsa.PublicKey)` (`pkg/auth/validator.go:227-235`) is a thread-safe in-place swap, and `auth.Middleware` (`pkg/auth/middleware.go:43-45`) holds a pointer to that same `*Validator`. So a background goroutine holding the `*Validator` (or the `*auth.Middleware`, which embeds it) plus the `*auth.VaultClient` can poll `vaultClient.GetPublicKey(path)` on an interval and call `validator.UpdatePublicKey(newKey)` — no router or handler changes needed in any of the three services.
- The three services do **not** share one Vault client instance or one main.go — each constructs its own `auth.VaultClient` and `auth.Validator` independently (content and explore-recommender via `auth.NewValidator` directly; email-ingest via its `middleware.NewJWTAuth` wrapper around the same). Per the task's "put the logic in one place" instruction and recipe R6, the right home is a new small helper in `pkg/auth` (e.g. a `KeyRefresher`/poller type taking a `*VaultClient`, a path, and a `*Validator`, started with one `Start(ctx, interval)` call) that each of the three `main.go` files calls after constructing its validator — rather than three copies of a ticker loop.
- Re-fetch-once-on-signature-failure (the second half of the "Done when" spec) needs a hook in `Validator.ValidateToken` or its caller: today `ErrInvalidSignature` is returned but nothing observes it to trigger an out-of-band refresh. This will likely mean either exposing a callback/error channel from the validator, or having `auth.Middleware.RequireAuth` trigger a one-shot refresh via the same `KeyRefresher` when it sees `ErrInvalidSignature`, with de-duplication so concurrent 401s don't cause a refresh storm.
- No overlap/grace period exists on the rotation-manager side, so a refresh interval that is not comfortably smaller than `JWT_KEY_ROTATION_INTERVAL` (default 24h) still leaves a window; the task's own guidance ("interval much smaller than the rotation interval") is correct and necessary, not just nice-to-have, given the confirmed no-grace-period behavior from Q3.

## Implementation plan (2026-08-09)

Validation approved; implementing per the shape identified above. Branch: `fix/p2-c1-jwt-public-key-refresh`.

- [ ] `pkg/auth/validator.go`: add an `onSignatureFailure func()` hook field on `Validator`, `SetOnSignatureFailure`, and refactor `ValidateToken` to retry once via the hook when it hits `ErrInvalidSignature` (extract the existing body into `parseAndValidate`). No change to any other error path or to any exported signature.
- [ ] `pkg/auth/key_refresher.go` (new): `PublicKeyFetcher` interface (matches `*VaultClient.GetPublicKey`'s signature so `*VaultClient` satisfies it with no adapter), `KeyRefresher` type wrapping a fetcher + key path + `*Validator`. `NewKeyRefresher` wires itself as the validator's on-signature-failure hook (debounced, to stop crafted-bad-signature spam from hammering Vault). `Start(ctx, interval)` polls on a ticker and pushes `UpdatePublicKey`.
- [ ] `pkg/auth/key_refresher_test.go` (new): fake `PublicKeyFetcher`; reuse the existing `generateTestKeys`/`generateTestToken` helpers in `validator_test.go` (same package). Cases: (1) on-demand recovery — rotate the fake fetcher's key, validate a token signed with the new key with *no* `Start()` call, assert it succeeds on the very first call (proves "no restart needed"); (2) periodic poll recovery with a short interval; (3) debounce — two rapid signature failures trigger only one fetch; (4) pre-rotation tokens still validate (no regression to the happy path). Prove case (1) fails without the fix by temporarily no-op'ing the refresh call, run, confirm the expected failure, then restore the real implementation and confirm it passes.
- [ ] Add `JWT_PUBLIC_KEY_REFRESH_INTERVAL` to each of the three services' config (default well under the 24h rotation interval, value + justification in the PR): `services/read/content/internal/config/config.go` (`VaultConfig`), `services/explore/recommender/internal/config/config.go` (`VaultConfig`), `services/read/email/internal/config/config.go` (`AuthConfig`).
- [ ] Wire `auth.NewKeyRefresher(...).Start(ctx, interval)` into each `main.go` right after its validator/middleware is constructed: `services/read/content/cmd/content/main.go`, `services/explore/recommender/cmd/explore_recommender/main.go`, `services/read/email/cmd/email_ingest/main.go`. No router or handler changes.
- [ ] Verify per module (pkg/auth, services/read, services/explore, services/read/email): `gofmt -l . && go vet ./... && golangci-lint run && go test -race -count=1 ./...`.
- [ ] Commit only the Go files touched + `.chalk/tasks/task_88aa.md` (per coordinator: AGENTS.md, the strategy doc, and other task files are pre-existing uncommitted work, not mine — leave them). Open PR; note in the PR body that the §1.3 ledger row still needs ticking since the strategy doc isn't tracked in this commit.
