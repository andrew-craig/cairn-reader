# Cairn Reader — Code Quality Review

**Date:** 2026-07-05
**Scope:** Full repository — four Go backend services (users, read/content, read/fetcher, read/email, explore), shared `pkg/`, the self-host binary, and the React Native mobile + React web front-ends.
**Method:** Seven parallel focused reviews, one per area, each verifying findings against source (a subset were confirmed with throwaway tests). This document is the coordinator's consolidation.

---

## Executive summary

The codebase is, on the whole, **competently built and consistently structured**: clean repository/service/handler layering in every Go service, RS256 JWT validation that correctly pins the signing method and uses constant-time comparison, a genuinely careful shared RSS pipeline (body-size caps, redirect limits, canonical sanitizer), and thoughtful front-end patterns (refresh-token mutex, stale-while-revalidate caching, optimistic-update rollback guards). Happy-path test coverage is good.

The problems cluster into **six recurring themes** rather than one-off bugs. The most serious are **authorization boundary gaps** in the read and explore services (unauthenticated SSRF and unauthenticated writes to shared content) and a **security feature that silently doesn't work** (refresh-token reuse detection in the users service). Underneath the individual bugs is a consistent story of **organic copy-paste drift**: the "single source of truth in `pkg/`" promise is only partially delivered, and several features marked "done" are incomplete or non-functional in practice.

None of the findings are exotic — they read as the natural gaps of a fast-moving multi-service build, and most are fixable without architectural change.

### Severity tally

| Severity | Count | Areas |
|---|---|---|
| Critical | 5 | read (2), users (1), explore-via-shared CORS (1), email (1×2 outbox) |
| High | ~11 | users (4), read (1), email (2), explore (2), shared (2), mobile (3), web (0) |
| Medium/Low | ~35 | all areas |

---

## Cross-cutting themes (read this first)

These are the patterns worth fixing structurally, because each spans multiple services.

### Theme 1 — Authorization boundaries are assumed, not enforced
Multiple services expose endpoints that comments/docs label "internal only" or "used by internal services," but nothing actually enforces that:
- **read/content**: `POST/PUT /api/v1/content/*`, `/bulk`, `/check-duplicate`, and the URL-detection endpoints sit behind only `RequireHTTPS`. The service's own CLAUDE.md even lists them under "Public Routes (No Auth)." This yields both **unauthenticated SSRF** and **unauthenticated writes to shared content records** (see Critical #1, #2).
- **explore**: `POST /api/v1/explore/article` and the fetcher's `/fetch` / `/sync` triggers have no service-to-service auth — anyone who can reach them can inject articles or hammer triggers.
- **email**: the ingest endpoint has no request body size limit on an explicitly untrusted inbound pipeline.

**Structural fix:** an internal-API-key middleware already exists (`RequireInternalAPIKey`, used correctly on `/api/v1/internal/*` in read/content). Apply it consistently to every endpoint the architecture treats as internal, and add an SSRF-safe HTTP dialer (block loopback/RFC1918/link-local/metadata IPs) to the shared fetch path.

### Theme 2 — Credential material in logs
Three services log secret-equivalent material:
- **users**: raw refresh-token prefix logged on *every* `/auth/refresh` (in both handler and service layers); email-verification token logged in full cleartext.
- **mobile**: `auth.ts` logs token prefixes and full refresh-failure response bodies via unconditional `console.log` (not gated by `__DEV__`).

This directly violates the project's own "never log passwords or hashes" principle. **Fix:** strip token material from all log lines; gate any debug logging behind an env/`__DEV__` flag.

### Theme 3 — Duplication instead of shared code ("single source of truth" only partly delivered)
The same logic is reimplemented in 2–4 places across the repo, so a fix in one copy silently misses the others:
- **HTTP fetch + size-cap + User-Agent**: 4+ independent copies across read/content and read/fetcher — which is *why* the SSRF gap exists in four places at once.
- **Env-var parsing**: four near-identical copies, two of them inside `pkg/` itself (`pkg/env` and `pkg/config`), with divergent duration-parsing behavior.
- **JWT validation**: `services/users/internal/auth/jwt.go` duplicates ~250 lines of `pkg/auth/validator.go` — and the duplicate is **dead in the request path** (only tests call it), so its passing tests give false confidence.
- **HTML sanitization**: email builds its own bluemonday policy instead of the canonical `pkg/rss/sanitize` that read/content correctly delegates to.
- **Client IP extraction**: duplicated between `auth_handler.go` and `pkg/middleware/recovery.go`.
- **Front-end auth/token-refresh state machine**: `apps/web/src/services/auth.ts` is a near-verbatim copy of `apps/mobile/src/services/auth.ts` instead of living in `apps/shared` (which already demonstrates the right injectable-adapter pattern for server-URL logic).

### Theme 4 — Job claiming has no locking (latent under horizontal scaling)
No `FOR UPDATE SKIP LOCKED` or atomic claim-and-update appears **anywhere** in the codebase. Every worker does `SELECT ... WHERE status='pending'` then flips status afterward:
- read/fetcher (`GetPendingItems`, `GetPendingEntries`, `GetFeedsDueForPolling`)
- explore/fetcher (`GetNextFeed`)
- email outbox

Single-replica today, so no active bug — but running any worker with >1 replica (normal for HA) causes duplicate fetches, duplicate outbound traffic to third-party sites, and last-writer-wins races on feed error/tier tracking.

### Theme 5 — Features marked "done" that are incomplete or non-functional
- **users**: refresh-token reuse detection never fires for its actual threat (Critical #3); password-reset tokens are generated and stored but **never delivered** to the user (no mailer, no log, no return value); self-host wiring omits two dependencies, panicking on the reset/verify routes.
- **explore**: `feed_id` is never populated by the production fetch path (all articles have `feed_id = NULL`); the documented discovery/exploration recommendation slot is dead code.
- **email**: `OUTBOX_MAX_RETRIES` config is silently ignored; `raw_emails.content_hash` column is entirely unused.
- **read/content**: `CleanupJob.RunWithBatching` claims to batch deletes to avoid long locks but doesn't actually pass the batch size through — the delete has no `LIMIT`.

### Theme 6 — Tests cover happy paths, not the security/correctness edges
Across every area, the exact code that's risky is the code that's untested: auth boundaries, SSRF validation, concurrent-claim races, token-reuse detection (the one test that exists passes via the wrong code path), the front-end 401-retry/refresh-race logic, and pathological/oversized untrusted input. Coverage numbers look healthy while the load-bearing paths regress silently.

---

## Critical findings

### C1 — Unauthenticated SSRF via URL-fetching endpoints (read/content)
`services/read/content/internal/api/router.go:85-86` registers `POST /api/v1/content/detect` and `/discover-feed` outside `RequireAuth`. `internal/service/url_detector.go:184-344` fetches the caller-supplied URL with no scheme/host validation — loopback, RFC1918, link-local, and cloud-metadata IPs are all reachable. `/discover-feed` additionally fans out 8 concurrent probes per call (useful for internal port/host scanning). Same unvalidated fetch pattern recurs in `processor/content.go:111-140`, `fetcher/.../feed_service.go:245-303`, `conditional_fetcher.go`, and `item_processor.go:212-240`.
**Repro:** anonymous `POST /api/v1/content/detect {"url":"http://169.254.169.254/latest/meta-data/iam/security-credentials/"}` → service fetches it server-side and returns the body.

### C2 — Unauthenticated writes to shared content records (read/content)
`router.go:88-95`: `POST /`, `PUT /{content_id}`, `/bulk`, `/check-duplicate` have only `RequireHTTPS`. `contents` rows are shared across all users (dedup by `content_hash`+`source_feed_id`). Any anonymous caller can `PUT /api/v1/content/{content_id}` with arbitrary HTML and **overwrite the title/body of an article every subscriber has saved** — content injection/defacement affecting other users, with zero authorization check. Confirmed: the router applies neither JWT nor internal-API-key middleware to these routes.

### C3 — Refresh-token reuse detection does not fire for the threat it exists for (users)
`internal/database/refresh_token_repository.go:179-193` — `RevokeToken` **deletes** the row on rotation. `isTokenReused()` can only flag reuse if the same still-present row is queried twice within a 15s grace window. In the real "attacker replays a stolen token after the victim has legitimately rotated it" scenario, the old row is already deleted → `ErrNoRows` → generic "not found," so **no family revocation, no security alert, no `token_reuse_detected` audit event** ever fires. The test at `auth_service_test.go:562-587` sleeps 6s and asserts only `assert.Error` — it passes through the not-found path and never exercises real reuse detection, hiding the gap. This is the signature security feature of the rotation design and it's inert.

### C4 — CORS wildcard-subdomain bypass in shared middleware (pkg)
`pkg/middleware/cors.go:131` — `strings.HasSuffix(origin, domain)` with no leading-dot anchor. `*.example.com` matches `http://evilexample.com`. **Verified with a throwaway test.** Currently unused in production (all services use `DefaultCORSConfig` = wildcard `*`, no credentials), so it's latent — but it's exported shared code with no test covering the bypass shape, so the next service to adopt "CORS with subdomain wildcards" inherits a live credentialed-CORS bypass.

### C5 — Outbox delivery blocks the entire worker for up to ~17h during a downstream outage (email)
`internal/client/content_service_client.go:21-28,146-169` — `DeliverContent` runs its *own* blocking retry loop (`time.After` over delays of 1m/5m/15m/1h/4h/12h) **inside** `deliverBatch`, which processes entries sequentially in a single ticker loop (`internal/worker/outbox_worker.go:70-88`). This duplicates and conflicts with the correct DB-level backoff already present. If the Content Service is degraded, the first failing entry stalls delivery for *every user* for up to ~17 hours. The test patches `retryDelays` to millisecond values, so production timing is never exercised.

---

## High-severity findings by service

### Users service
- **H1** Raw refresh-token prefix logged on every `/auth/refresh` (handler `auth_handler.go:295-307` and service `auth_service.go:398-405`).
- **H2** Email-verification token logged in full cleartext (`email_verification_service.go:84-91`) — anyone with log access can confirm/hijack any email.
- **H3** Password-reset tokens generated and stored but never delivered (`auth_service.go:527-558`) — feature is functionally broken despite being marked done.
- **H4** Self-host wiring omits `PasswordResetTokenRepo` and `EmailVerificationService` (`selfhost/users.go:104-134`) while the router always registers those routes (`router.go:97-102`) → nil-interface panic/500 on `/forgot-password`, `/reset-password`, `/verify-email` in the self-host build.

### Read service (content + fetcher)
- **H5** Duplicate job execution under multiple worker replicas — no row locking anywhere (`repository/feed_item.go:322-338`, `outbox.go:157-170`, `feed.go:292+`). See Theme 4.

### Email ingest service
- **H6** No request body size limit on the untrusted ingest endpoint (`api/handlers/ingest_handler.go:26-30`) — unbounded memory/DB/CPU from a large POST.
- **H7** Divergent hand-maintained bluemonday policy (`processor/content_extractor.go:26-55`) instead of the canonical `pkg/rss/sanitize` — future sanitizer hardening won't reach email content. Also `OUTBOX_MAX_RETRIES` silently ignored (`worker/outbox_worker.go:32-50`).

### Explore service
- **H8** A single malformed/colliding article poisons the whole feed batch (`recommender/internal/db/article_repository.go:77-127`): one multi-row `INSERT ... ON CONFLICT (link)`. An empty/duplicate `<link>` trips "cannot affect row a second time"; and because article `id = ContentHash(content)` (not the link), syndicated duplicate content collides on the primary key and **recurs on every subsequent poll** until the feed auto-disables — silently discarding all good articles in the batch.
- **H9** `feed_id` never populated in production (`fetcher/internal/fetcher/fetcher.go:168-200`) — every article stored with `feed_id = NULL`, breaking documented per-feed metadata/analytics.

### Shared `pkg/` + cross-service
- **H10** Panic-recovery middleware never sees the request ID in **all six routers** (`pkg/middleware/recovery.go:24` + registration order): `Recovery` is registered before `ChiRequestLogger`, so every panic log is stamped `request_id=unknown`, defeating incident correlation. **Verified.**
- **H11** JWT validation duplicated (~250 lines) in `services/users/internal/auth/jwt.go` vs `pkg/auth/validator.go`, and the duplicate is dead in the request path (only tests use it). Security fixes to the canonical validator won't propagate.

### Mobile app
- **H12** A second 401 after token refresh is silently swallowed (`services/read.ts:50-72`, `explore.ts:86-108`) — only a *thrown* refresh failure logs the user out, so a re-rejected retry leaves the user stuck in a broken authenticated UI.
- **H13** `AddArticleScreen` is a dead/duplicate "add article" path that fabricates content IDs (`id: url.trim()`) and writes to an AsyncStorage key **no active screen reads** (`AddArticleScreen.tsx:62-79`); `ReadArticleDetailScreen` still writes to that dead key on every interaction (silent no-op churn).
- **H14** Verbose auth logging exposes token fragments and full response bodies to device logs (`services/auth.ts:353-417`). See Theme 2.

*(Web app: no critical or high findings — the healthiest area. Its two medium items are tokens-in-localStorage with no CSP while rendering untrusted article HTML, and the web/mobile auth-layer duplication of Theme 3.)*

---

## Medium & low findings (condensed)

**Users:** dead `errors.Is(apperrors.ErrTokenExpired)` branch logs normal token expiry at ERROR (wrong sentinel, `auth_service.go:441`); `ChangePassword` doesn't revoke sessions unlike `ResetPassword`; login timing side-channel enables user enumeration; `X-Forwarded-For`/`X-Real-IP` trusted unconditionally → rate-limit bypass + audit-log poisoning; dead `internal/middleware/authorization.go` (~200 lines).

**Read:** `err.Error()` leaked into client JSON across content handlers (pq constraint names exposed); ignored-error-then-nil-deref panic path (`user_content_handler.go:495,581`) returns 500 after a successful write; check-then-insert races surface as 500 instead of 409; `CleanupJob.RunWithBatching` false batching claim; `context.Background()` in work loops blocks shutdown; fetch/size-cap logic duplicated 4×.

**Email:** dead `raw_emails.content_hash` column; hardcoded retry threshold in SQL; `MockAPIKeyRepository` shipped in production source; unbounded recursion on attacker-controlled HTML depth; `sender_email` not case-normalized (duplicate sender rows).

**Explore:** no row locking in `GetNextFeed`; SSRF exposure in feed fetching (no allow/deny-list); internal endpoints unauthenticated; dead `GetLowExposureArticles`; quality-score doc/code sign mismatch (code correct); config-bootstrap duplication + divergent migrate defaults.

**Shared:** env-parsing duplicated 4× with divergent duration semantics; 3 of 6 self-host DBs excluded from `/health/ready` (reports healthy while half the DBs are down); stale `google/uuid v1.1.2` pin in `pkg/auth/go.mod`; inconsistent health-check JSON contracts per service; no JWT clock-skew leeway (`jwt.WithLeeway`) → spurious 401s at expiry with 15-min tokens; dead `StubHandler` comment.

**Mobile:** `HeaderPopover` hardcodes light-theme background; URL detection fires per-keystroke with no debounce/cancel; unmemoized `AuthContext` value → app-wide re-renders; unmemoized list rows → full-viewport re-render on pagination; list screens lack unmount guards on `setState`; two inconsistent `isValidUrl` implementations; shared abort-timeout budget spans request+refresh+retry.

**Web:** entity decoding applied to titles but not descriptions/authors; dead no-op block in `AddLinkModal.tsx:143-145`; unhandled clipboard promise rejection; duplicated `useFocusTrap` hook.

---

## Recommended priority order

1. **Close the authorization gaps** (C1, C2, explore/email internal endpoints) — apply `RequireInternalAPIKey` consistently and add an SSRF-safe dialer to the shared fetch path.
2. **Fix or disable the broken security features** (C3 reuse detection; H3 password-reset delivery) — a feature that appears to protect but doesn't is worse than an absent one.
3. **Stop logging credential material** (H1, H2, H14).
4. **Fix the outbox worker-blocking bug** (C5) before the email pipeline sees real load.
5. **Add row-level locking** (`FOR UPDATE SKIP LOCKED`) before any worker is scaled past one replica (Theme 4).
6. **Consolidate the duplicated code** (Theme 3) so future fixes land once — this is what makes 1–5 stay fixed.
7. Fix C4 (CORS) now even though it's latent — it's a one-line anchor (`"."+domain` suffix check) and prevents a future foot-gun.
8. Backfill tests on the security/correctness edges (Theme 6), starting with the reuse-detection and 401-retry paths.

---

## What's genuinely solid (so it isn't re-litigated)

- `pkg/auth` RS256 validation: pins signing method, checks issuer/audience, constant-time internal API-key comparison — no algorithm-confusion or timing issues.
- `pkg/rss/{fetch,parse,sanitize,hash}`: redirect limits, body-size caps, one canonical sanitizer; read/content correctly delegates to it.
- Users service: atomic SQL for lockout/verification/reset, correct `defer tx.Rollback`, solid happy-path coverage.
- Web `utils/sanitize.ts`: correct DOMPurify usage plus anchor hardening (`target=_blank` + `rel=noopener noreferrer`), well tested.
- Front-end refresh-dedup mutex, stale-while-revalidate caching, and optimistic-update rollback guards are implemented correctly and race-free.
- `cmd/selfhost/web.go` static-file serving correctly guards against path traversal.

---

## Suggested next steps — the next 10 investigations

This first pass was seven **static, per-area** reviews. The highest-value follow-ups deliberately cut along *different axes* — dynamic behavior, cross-service seams, data at rest, and supply chain — rather than re-reading the same files. In rough priority order:

1. **Dynamic verification pass.** Actually build every module and run `go test -race ./...`, `go vet`, `golangci-lint`, plus `tsc --noEmit` / `eslint` on the front-ends. Static review can't see data races, and several findings above (Theme 4 locking, the mobile refresh-race, the users reuse-detection path) are exactly the kind of thing `-race` and a green-but-wrong-path test hide. First job: confirm the whole repo even builds and the suites pass on a clean checkout.

2. **Cross-service contract conformance.** Diff each service's HTTP *client* against the *server* it calls (read/fetcher→content, email→content, explore/fetcher→recommender) and against the four `openapi.yaml` specs. Look for field/enum/status-code drift, error-shape mismatches, and missing idempotency keys across boundaries — the seams are where the per-service reviews had the least visibility.

3. **Database & migration review.** Up/down reversibility, index coverage vs. actual query predicates (are the hot `WHERE`/`ORDER BY` paths indexed?), `ON DELETE` behavior, missing constraints, and the broader fallout of the content-hash-as-primary-key design (finding H8). Also transaction boundaries for the multi-step writes flagged in read/content and users.

4. **Dependency & supply-chain audit.** `govulncheck` across all `go.mod`, `npm audit` on both front-ends, plus an outdated/pinned-dep sweep (the stale `google/uuid` pin is likely not the only one). Check transitive CVEs in the RSS/HTML-parsing and JWT libraries specifically, since those sit on the untrusted-input path.

5. **Infrastructure, deployment & secrets hygiene.** Review `infrastructure/docker`, the Dockerfiles, compose files, the Vault init scripts, and health-probe wiring. What happens when **Vault is down** at startup (every service depends on it for JWT keys)? Are there secrets/default passwords in the repo? Resource limits, non-root containers, image pinning.

6. **Systematic tenancy / IDOR matrix.** One consistent checklist over *every* user-scoped endpoint in all four services, verifying `user_id` scoping is enforced on the query itself (not just a path-param check). The per-service reviews spot-checked this; a full matrix would either close it out or find the gap the spot-checks missed.

7. **Observability & logging audit.** Structured-logging consistency, log levels, PII beyond the token leaks already found, and the metrics/tracing gaps that make incident response hard (this is where the `request_id=unknown` finding, H10, really bites). Is there anything to alert on when the outbox worker wedges (C5)?

8. **Test-suite quality meta-review.** Not "is there coverage" but "does the coverage assert anything." Hunt for tests that pass through the wrong code path (like the reuse-detection test, C3), mocks that have drifted from real signatures, and error paths that are never exercised. Quantify coverage on the *security-critical* files specifically.

9. **DoS / resource-exhaustion sweep.** A systematic pass for unbounded work: pagination without max limits, queries without `LIMIT`, unbounded allocations/recursion (the email HTML-depth issue), and missing body-size caps across *all* services, not just email. Pair each with the concurrency limits (or lack thereof) on the worker pools.

10. **Front-end deep-dive #2 — resilience & accessibility.** Error/empty/offline states, loading skeletons, retry UX, keyboard/screen-reader accessibility, and actual performance profiling of the large article lists (the mobile review flagged unmemoized rows statically; a profiler would quantify the jank and catch anything static review missed).

If run, #1 and #2 should go first — a dynamic pass and the cross-service contract check are the two things most likely to surface a *new* critical that a read-only, single-service lens structurally cannot.
