# Audit 4 — Security & Auth Hardening

**Workstream:** `task_659b` (epic `epic_9c21`, P2)
**Status:** Findings complete
**Date:** 2026-06-16

> This is the auth/security posture workstream. The primary surface is the User Service
> (`services/users/`), which is the **only service that issues credentials** and the only one
> with password/token logic. All other services are downstream JWT validators. Every
> **BETA-BLOCKING** item must land before public signups open; **FAST-FOLLOW** items are
> safe to fix during beta without breaking clients; **ROADMAP** items are correct at launch
> scale and documented to prevent rediscovery.

## How to read the classification

| Class | Meaning | Why |
|---|---|---|
| **BETA-BLOCKING** | An exploitable gap or missing control that would be unacceptable on a public service | Must land before signups |
| **FAST-FOLLOW** | Real gap, but not immediately exploitable or not outward-facing | Fix during beta |
| **ROADMAP** | Correct at current scale; tracked to prevent future surprises | No action now |

---

## Confirmed-good baseline

Each item below was verified against real code. The leads that said "check these" all hold.

| Control | Evidence |
|---|---|
| bcrypt cost 12 (default, validated 10–14 range) | `services/users/internal/config/config.go:113,150` |
| RS256 stateless JWT (`jwt.SigningMethodRS256`) | `services/users/internal/auth/jwt.go:142` |
| Algorithm enforcement in validator (rejects non-RSA alg) | `services/users/internal/auth/jwt.go:178-180` |
| Refresh-token rotation (old revoked, new issued, family tracked) | `services/users/internal/auth/refresh_token.go:130-211` |
| Reuse detection → full token-family revocation | `services/users/internal/auth/refresh_token.go:155-177` |
| Refresh tokens stored as SHA-256 hash, not plaintext | `services/users/internal/auth/refresh_token.go:62-83` |
| IDOR prevention: `RequireSameUser` middleware (UUID comparison) | `services/users/internal/middleware/authorization.go:15-65` |
| IDOR prevention: service-layer ownership check (`requestingUserID != targetUserID`) | `services/users/internal/handlers/user_handler.go:65,120` |
| Parameterized queries throughout (`QueryRow(ctx, query, $1…)`) | `services/users/internal/database/user_repository.go:81,135,177,210,244` |
| HTML sanitization via bluemonday (canonical shared policy) | `pkg/rss/sanitize/sanitize.go:82-84` |
| Security headers: `X-Frame-Options`, `X-Content-Type-Options`, HSTS, `Referrer-Policy`, `Permissions-Policy` | `pkg/middleware/security.go:56-66` |
| HTTPS enforcement (`RequireHTTPS` middleware on all `/api/v1` routes) | `services/users/internal/handlers/router.go:78` |
| No hardcoded secrets (Vault required; secrets from env only) | `services/users/internal/config/config.go:96-108`, `cmd/user-service/main.go:52-58` |
| IP-based rate limiter on auth endpoints (default 10 req/min per IP) | `services/users/internal/handlers/router.go:67-84`, `pkg/middleware/rate_limit.go:99-121` |
| Password complexity validation (uppercase + lowercase + digit + special) | `services/users/internal/auth/password.go:46-84` |

---

## Summary table

| ID | Finding | Class |
|---|---|---|
| S-1 | No per-account failed-login tracking; IP rate limit only | **BETA-BLOCKING** |
| S-2 | Email validation is `strings.Contains(@)` only — no RFC check, no verification flow | **BETA-BLOCKING** |
| S-3 | No password reset / account recovery flow | **BETA-BLOCKING** |
| S-4 | Rate limiter is in-memory and per-instance — resets on restart and won't hold under horizontal scale | FAST-FOLLOW (cross-ref Audit 6) |
| S-5 | No structured auth audit log (successful logins, failed logins, account changes) | FAST-FOLLOW |
| S-6 | Vault is a hard startup dependency with no offline key fallback | FAST-FOLLOW |
| S-7 | No CAPTCHA / bot protection on public signup endpoints | ROADMAP |

---

## S-1 — No per-account failed-login tracking (BETA-BLOCKING)

### Current state

The only brute-force protection is a token-bucket rate limiter keyed on the **client IP address**
(`pkg/middleware/rate_limit.go:103-104`, key = `getClientIP(r)`). The `RateLimiter` stores state in
a Go `map` guarded by a `sync.RWMutex` (`pkg/middleware/rate_limit.go:12-18`). There is no counter
of consecutive failed logins, no exponential back-off, and no account lockout.

The Login handler returns `ErrInvalidCredentials` on a bad password
(`services/users/internal/services/auth_service.go:188-189`) and logs nothing for the failure at the
service layer (`auth_service.go:167-212` has no `slog` call on the wrong-password path). The handler
emits a 401 with no logging either (`auth_handler.go:183-194`).

A distributed attacker using thousands of IPs — trivially available via residential proxy networks —
faces no obstacle beyond the 10-req/min per-IP cap.

### Why beta-blocking

Credential stuffing and targeted account takeover are the most common attack on a newly-public
service. The IP rate limit is easily bypassed by any off-the-shelf tool; once the attacker has a
valid email (obtainable via `/register` conflict response), they can try every password from a breach
list across thousands of IPs with zero friction.

### Recommended control

1. Add a `failed_login_attempts` column (int, default 0) and `locked_until` column (timestamptz,
   nullable) to the `users` table via a new migration.
2. In `authService.Login` / `LoginMobile`: on `ErrInvalidCredentials`, increment the counter and
   set `locked_until = NOW() + lockout_duration` when the threshold is reached (suggested:
   5 failures → 15-minute lockout, with exponential back-off to 1-hour maximum).
3. At the start of every login, reject immediately if `locked_until > NOW()` (return the same
   generic 401 to avoid revealing lock state to the attacker; log the blocked attempt internally).
4. Reset the counter on successful login.
5. The lockout state lives in the `cairn_users` DB so it is consistent across instances (unlike the
   in-memory rate limiter).

**Effort:** 1–2 days (new migration + ~50 lines in repository + ~30 lines in service).

**Note:** this adds a schema column; coordinate with Audit 1 (no wire-format change to the external
API, so no contract freeze impact).

---

## S-2 — Email validation is `strings.Contains(req.Email, "@")` only (BETA-BLOCKING)

### Current state

Three places in the codebase perform email validation and all use the same one-liner:

| Location | Line |
|---|---|
| `Register` handler | `services/users/internal/handlers/auth_handler.go:81` |
| `UpdateUser` handler | `services/users/internal/handlers/user_handler.go:114` |
| `UpgradeAccount` handler | `services/users/internal/handlers/user_handler.go:177` |

`strings.Contains(req.Email, "@")` accepts `@` (a valid-to-this-check single character), `@@@@`,
`notanemail@`, and other obviously malformed strings. More importantly, the service has **no email
verification flow**: a user can register with `fake@anything.tld`, receive tokens immediately, and
use the service indefinitely without proving they own the address. There is no verification token,
no verification endpoint, and no schema column to track verification status.

### Why beta-blocking

Two separate issues, both beta-blocking:

1. **RFC validation:** accepting `@` as a valid email is a quality-of-service bug that will produce
   confusing support requests and dirty data. Easy to fix and cheap to do before the schema
   stabilises.
2. **Email verification:** a public service that issues accounts without verifying emails will be
   abused for spam, throwaways, and credential stuffing. Password reset (S-3) is also impossible
   without a verified, reachable email. The verification flow requires a new DB column, a new
   endpoint, and a transactional email — these are structural decisions that are far cheaper before
   freeze than after.

### Recommended control

**Immediate (before freeze):** Replace `strings.Contains` with a proper RFC 5322 parser. The
standard library `net/mail.ParseAddress` rejects obviously malformed addresses; or use
`github.com/badoux/checkmail` for DNS-aware validation. Apply in one shared helper so all three
call sites stay in sync.

**Verification flow (before beta opens):**
1. Add `email_verified BOOLEAN NOT NULL DEFAULT FALSE` and
   `email_verification_token VARCHAR(255)` columns.
2. On register/upgrade: generate a signed, time-limited token, store the hash, send a verification
   email. Return tokens immediately (allow unverified use for a grace period, e.g. 7 days) but mark
   the account unverified.
3. Add `POST /api/v1/auth/verify-email` accepting the token, setting `email_verified = TRUE`.
4. Gate password-reset (S-3) on `email_verified`.

**Effort:** RFC validation = 1 hour; full verification flow = 3–4 days.

**Contract impact:** The verification endpoint is a new route (no freeze concern). The `email_verified`
field may appear in the user profile response — coordinate with Audit 1 to lock the user DTO shape.

---

## S-3 — No password reset / account recovery flow (BETA-BLOCKING)

### Current state

There is no `POST /api/v1/auth/forgot-password`, no reset-token table or column, and no
transactional email integration. If a user with an email/password account forgets their password,
the only option is account deletion and re-registration (losing all content). Mobile-only accounts
have a separate recovery problem (reinstall = new account), but that is by design and documented.

The `services/users/CLAUDE.md` lists "Password reset functionality" under **High Priority** future
enhancements, confirming this is a known gap.

### Why beta-blocking

A public service with no account recovery will generate support tickets from day one and will lose
users who are locked out. More critically: without password reset, a compromised account (attacker
changes password via `PUT /api/v1/user/{id}/password`) is permanently inaccessible to the real
owner. The `ChangePassword` handler requires the current password
(`services/users/internal/handlers/user_handler.go:254`), so a compromised account cannot be
reclaimed.

### Recommended control

Standard "forgot password" flow:
1. `POST /api/v1/auth/forgot-password` — accepts `email`; if found and `email_verified` (S-2),
   generate a cryptographically random reset token, store its SHA-256 hash in a
   `password_reset_tokens` table (or a column on `users`) with a short expiry (e.g. 15 minutes),
   and send the raw token via transactional email. Always return 200 regardless of whether the
   email exists (prevents user enumeration).
2. `POST /api/v1/auth/reset-password` — accepts `token` + `new_password`; validates hash, checks
   expiry, updates password, invalidates the token and all refresh tokens for the account.

**Dependencies:** requires S-2 (email verification) and a transactional email provider (SendGrid,
Postmark, SES, etc.).

**Effort:** 2–3 days for the backend logic; additional time for email provider integration.

**Contract impact:** Two new public endpoints — include in Audit 1 spec freeze and OpenAPI spec.

---

## S-4 — Rate limiter is in-memory and per-instance (FAST-FOLLOW; cross-ref Audit 6)

### Current state

`pkg/middleware/rate_limit.go` implements a token-bucket limiter backed by a Go `map[string]*bucket`
(`rate_limit.go:12`), initialised fresh in each `NewRateLimiter` call. Each process instance has its
own independent limiter with no shared state.

Consequences:
- **Restart bypass:** any service restart (rolling deploy, crash-loop, OOM kill) zeroes all counters.
  An attacker who notices a deploy window gets a free burst.
- **Horizontal scale bypass:** with N instances behind a load balancer, an attacker can make N × 10
  requests per minute by round-robining across instances. Audit 6 (multi-instance / deployment) will
  surface the same concern from the infrastructure side.
- **Memory growth:** the cleanup loop runs every `window * 2` (`rate_limit.go:35`), keeping entries
  until 2 minutes of inactivity. Under a sustained low-rate attack from many IPs, the map grows
  without bound until the cleanup pass.

### Recommended control

Replace the in-memory store with Redis (or use the existing `pkg/middleware/RateLimitWithKey` with a
Redis-backed store). Redis `INCR` + `EXPIRE` is the canonical pattern and survives restarts and
scales horizontally. The `RateLimitWithKey` signature (`rate_limit.go:127`) already accepts a custom
`KeyFunc`, so the middleware wiring is already designed for this swap.

**Effort:** 1–2 days (add Redis client, implement `RedisRateLimiter`, swap in router).

**Cross-ref:** Audit 6 (multi-instance state) will likely surface this as a deployment constraint.
Coordinate so the Redis decision is made once.

---

## S-5 — No structured auth audit log (FAST-FOLLOW)

### Current state

Successful logins (`authService.Login`, `authService.LoginMobile`) log nothing. The only evidence
of a successful login is the internal `UpdateLastLoginAt` warning if that call fails
(`auth_service.go:195-197, 238-240`). Failed logins produce no log entry at either the service or
handler layer.

The `/auth/refresh` endpoint is the exception — it has structured `slog` calls for success, reuse,
not-found, and expired states (`auth_handler.go:281-348`) but they use `Debug`/`Info` levels and
include only a token preview, not the user ID.

Account mutations (`UpdateUser`, `ChangePassword`, `UpgradeAccount`, `DeleteUser`) log nothing on
success or failure at the service layer.

### Why this matters

Without an auth audit log, there is no way to:
- Detect an account takeover post-breach (no record of unexpected logins from new IPs/devices).
- Investigate a support ticket ("I didn't change my email").
- Build rate-limit or anomaly detection on top of log data.

### Recommended control

Add structured `slog.Info` calls at the service layer for every security-relevant event:

| Event | Fields to log |
|---|---|
| Successful login | `user_id`, `ip`, `device_info`, `method` (email/mobile) |
| Failed login | `email_or_device_id` (hashed or truncated), `ip`, `reason` |
| Token refresh success | `user_id`, `ip` |
| Token reuse detected | `user_id` (from DB lookup), `ip`, `token_family` |
| Password changed | `user_id`, `ip` |
| Email changed | `user_id`, `old_email_hash`, `ip` |
| Account deleted | `user_id`, `ip` |
| Account locked (S-1) | `user_id`, `ip`, `fail_count` |

Use `slog` with structured key-value pairs (already the project standard). Do **not** log raw
emails or passwords — log hashed or truncated values where PII is involved.

**Effort:** 1 day (add `slog.Info` calls at service layer for each event; no schema change).

---

## S-6 — Vault is a hard startup dependency with no offline key fallback (FAST-FOLLOW)

### Current state

`cmd/user-service/main.go:53-57` calls `initializeVault(cfg)` and exits with `os.Exit(1)` on any
failure. `initializeVault` calls `vaultClient.Health()` (`main.go:258`) and also `GetJWTKeys`
(`main.go:62-68`). If Vault is unavailable at startup — network blip, Vault sealed after restart,
credentials rotated — the service refuses to start entirely.

A file-based fallback exists in `pkg/auth/file_keys.go` (`FileKeyProvider.GetKeys()`,
`file_keys.go:22-46`) but **it is not wired into the users service startup path**. The users
service's `main.go` exclusively calls `auth.NewVaultClient` → `GetJWTKeys`. Other services that
validate JWTs also fetch the public key from Vault at startup with the same hard failure.

**Caveat:** `services/users/internal/auth/vault.go:279-417` implements a `KeyRotationManager` that
polls Vault periodically for key rotation, but this only starts **after** the initial key load
succeeds. There is no retry loop or grace period during initial startup.

### Why this matters

In production, any Vault maintenance window or transient network partition will take the entire auth
system down. At beta scale with a single Vault instance this is a real availability risk. The
`FileKeyProvider` in `pkg/auth` already exists as a fallback mechanism but is unused by the service.

### Recommended control

Two complementary options (pick one, or layer them):

**Option A — Startup retry with timeout.** Add a retry loop (e.g. 5 attempts, 5-second backoff,
30-second total) around the `initializeVault` + `GetJWTKeys` calls so that a brief Vault blip does
not kill the service during rolling deploys.

**Option B — File-based emergency fallback.** Wire `pkg/auth/file_keys.go`'s `FileKeyProvider` as
a fallback when Vault is unreachable. The public key baked into the image (or a mounted secret
volume) allows other services to keep validating JWTs while the auth service is in a degraded state.

Option A is the minimum viable fix. Option B requires a decision about whether baking a key into
the image is acceptable for the deployment model (overlaps Audit 6).

**Effort:** Option A = 2 hours; Option B = 1 day.

---

## S-7 — No CAPTCHA / bot protection on public signup (ROADMAP)

### Current state

`POST /api/v1/auth/register` and `POST /api/v1/auth/register/mobile` are protected only by the
IP-based rate limiter (10 req/min). There is no CAPTCHA, no proof-of-work, and no device
fingerprinting. A bot can register thousands of accounts from a rotating IP pool without friction.

### Why this is roadmap (not beta-blocking)

At beta scale (invite-only or limited open beta with manual review), automated account creation is
not an immediate existential threat. CAPTCHA also introduces UX friction and requires a third-party
dependency (hCaptcha, Cloudflare Turnstile, etc.). The right time to add it is when abuse is
observed or before a fully open public launch.

### Recommended control (when needed)

Add Cloudflare Turnstile or hCaptcha to the registration flow. The mobile registration path
(`/register/mobile`) likely does not need CAPTCHA since the Expo device ID provides implicit
device binding. Coordinate with the mobile app team on the UX impact.

**Effort:** 1–2 days backend + mobile app changes.

---

## Additional observations

### S-2 corollary: register conflict leaks email existence

`POST /api/v1/auth/register` returns HTTP 409 with `"an account with this email already exists"`
(`auth_handler.go:96-98`). This allows an attacker to enumerate valid email addresses. The standard
mitigation is to return 200/201 with a generic message regardless, and send the "account already
exists" information only to the (verified) email address. This is a secondary concern but pairs with
S-2 (verification flow) — once verification is implemented, the conflict disclosure becomes less
useful to an attacker.

### `getClientIP` spoofability

`auth_handler.go:408-426` and `pkg/middleware/rate_limit.go`'s `getClientIP` trust
`X-Forwarded-For` and `X-Real-IP` headers without validation. If the service is exposed directly
(no reverse proxy), an attacker can spoof these headers and bypass the IP-based rate limiter
entirely. In production behind a proper reverse proxy this is fine, but the code does not enforce or
document this deployment requirement. Ensure the production deployment strips/overwrites
`X-Forwarded-For` at the ingress before it reaches the service.

### `RequireSameUser` envelope mismatch (Audit 1 carry-over)

The `RequireSameUser` middleware (`services/users/internal/middleware/authorization.go`) emits
`{"error": "..."}` directly without `pkg/api.WriteError` (no `message` field, no `meta`). This
is the same F-1 issue identified in Audit 1 for auth middleware — same fix applies here: route
through `api.WriteError`.

---

## Security checklist for Audit 7

Before public signups open, all BETA-BLOCKING items below must be verified:

- [ ] **S-1** Per-account failed-login counter + lockout in DB; lock enforced at login; counter
      reset on success.
- [ ] **S-2a** Email validation replaced with `net/mail.ParseAddress` (or equivalent RFC check)
      at all three call sites.
- [ ] **S-2b** Email verification flow implemented: `POST /api/v1/auth/verify-email`, `email_verified`
      column, grace-period enforcement, locked to verified addresses for password reset.
- [ ] **S-3** Password reset flow implemented: `POST /api/v1/auth/forgot-password` and
      `POST /api/v1/auth/reset-password`; new routes in OpenAPI spec.
- [ ] **F-1 (Audit 1)** `RequireSameUser` middleware and any local error responses routed through
      `api.WriteError` for envelope consistency.

Fast-follow during beta:

- [ ] **S-4** In-memory rate limiter replaced with Redis-backed store; coordinate with Audit 6.
- [ ] **S-5** Structured auth audit log events at service layer for all security-relevant actions.
- [ ] **S-6** Vault startup retry loop (Option A) added; evaluate file-key fallback (Option B).

Roadmap (no action for beta):

- [ ] **S-7** CAPTCHA / bot protection on public signup; evaluate before fully open launch.
