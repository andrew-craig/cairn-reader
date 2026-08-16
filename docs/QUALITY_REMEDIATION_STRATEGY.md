# Cairn Reader — Quality Remediation Strategy

**Status:** Active
**Written:** 2026-08-09
**Tracking epic:** `epic_fefa` (chalk)
**Companion document:** [CODE_QUALITY_REVIEW.md](/docs/CODE_QUALITY_REVIEW.md) — the findings ledger this strategy executes against.

This document is a **playbook, not a report**. It tells an engineer (or agent) with no prior context: what to work on next, how to verify the problem is real, how to fix it without breaking things, and how to prove the fix. Follow it top to bottom.

---

## 0. How to use this document

1. Run `chalk ready --parent=epic_fefa`. If a task exists, claim it (`chalk update <id> --status=in_progress`) and jump to [Part 2](#part-2--how-to-fix-things) for the matching recipe.
2. If no tasks exist under the epic, you are the bootstrapper: create the next wave's tasks from the [wave plan](#13-the-wave-plan) (one chalk task per row, `--parent=epic_fefa`, priority as listed) and then claim the first one.
3. Every fix follows the same loop: **re-verify the finding → write a failing test → fix minimally → prove → PR**. No step is optional.
4. When a wave's tasks are all closed, create the next wave's tasks. Waves are strictly ordered; do not cherry-pick from a later wave while an earlier wave has open tasks, unless a task is blocked and says so.

### Rules of engagement (read before touching code)

- **One finding, one branch, one PR.** Small PRs get reviewed and merged; big ones rot.
- **Re-verify before fixing.** The review is dated 2026-07-05; code has moved (e.g. the mobile "Archive = hard delete" finding P2-C5 was fixed in PR #298). Confirm the cited file/line still exhibits the problem. If it's already fixed, note that in the chalk task, close it, and take the next one.
- **The failing test comes first.** If you cannot make a test fail that demonstrates the problem, either the finding is stale (close it) or you haven't understood it yet (stop and re-read). Never ship a fix whose test passed before the fix.
- **Copy the in-repo reference pattern** ([§2.3](#23-reference-patterns--copy-these-dont-invent)) instead of inventing a new one. Most findings have a correct implementation elsewhere in this repo.
- **Surgical diffs.** Don't reformat, rename, or "improve" adjacent code. Every changed line must trace to the finding.
- **Update the ledger.** When you close a task, tick the finding's row in [§1.3](#13-the-wave-plan) (change `[ ]` to `[x]` with the PR number) in the same PR.

---

## Part 1 — Where to look and how to prioritise

### 1.1 The primary source: the existing review

Do **not** start by re-auditing the codebase. A three-pass deep review already exists in [CODE_QUALITY_REVIEW.md](/docs/CODE_QUALITY_REVIEW.md) with ~50 findings, each with file:line evidence, severity, and often a repro. As of 2026-08-09, **essentially none of it has been remediated** (only P2-C5 landed, PR #298). The backlog below *is* the review, ordered.

Read the review's finding text in full before starting its task — this document only carries the ID, a one-line summary, and the recipe to apply.

### 1.2 How priority was assigned (and how to assign it for new findings)

Score each issue on three axes; fix in descending score order:

| Axis | Question | High score looks like |
|---|---|---|
| **Blast radius** | Who is affected when this fires? | All users / other tenants / operator credentials |
| **Reachability** | What does it take to trigger? | Anonymous network request, or a timer (fires on its own) |
| **Deception** | Does something *claim* to protect/work but doesn't? | Green test over a dead code path; security feature that never fires |

The **deception** axis is why some non-exploitable items rank high here: a test that passes through the wrong code path (P2-C3), or a security feature that silently never fires (C3), actively hides regressions, so fixing them multiplies the value of every later fix.

Deprioritise: latent issues gated on conditions that don't hold yet (multi-replica races — Theme 4), style-level lint debt, and anything in the review's "genuinely solid / don't re-litigate" lists.

### 1.3 The wave plan

Each row becomes one chalk task titled `[<ID>] <summary>` under `epic_fefa`. "Recipe" points into Part 2. **P** column = chalk priority.

#### Wave 1 — Stop the bleeding (external exposure + timer-triggered outage)

Everything here is reachable by an anonymous request or fires on a timer. Small, independent fixes.

| Done | Finding | Summary | P | Recipe |
|---|---|---|---|---|
| [x] #309 | P2-C1 | Downstream services fetch the JWT public key once at startup; users service rotates it every 24h → scheduled auth outage | 0 | [R6](#r6--operational-fixes) |
| [x] #311 | C2 + C1 (routes) | read/content write + URL-detection endpoints have no auth: anonymous content overwrite + SSRF | 0 | [R1](#r1--closing-an-authorization-gap) |
| [x] #314 | P2 IDOR | read/fetcher has **zero** inbound auth; user_id taken from URL path → cross-tenant read/write/delete of subscriptions | 0 | [R1](#r1--closing-an-authorization-gap) |
| [x] #317 | Explore auth | explore article-inject + fetch/sync trigger endpoints unauthenticated | 1 | [R1](#r1--closing-an-authorization-gap) |
| [x] #316 | SSRF dialer | Shared fetch path follows any URL (loopback/RFC1918/metadata) — 4+ copies of the fetch logic | 0 | [R2](#r2--ssrf-safe-fetching) |
| [x] #315 | P2-C2, H1, H2, H14 | Credential material in logs: full verification URL at INFO, refresh-token prefixes, mobile token logging | 0 | [R3](#r3--credential-material-in-logs) |
| [x] #313 | P3-C1, P3-H1, P3-H2, toolchain | Dependency bumps: `x/net` ≥ v0.55.0 (live XSS on ingestion path), `pgx` ≥ v5.9.2 (SQLi), Go toolchain patch bump — all 12 modules | 0 | [R4](#r4--dependency-bumps) |
| [x] #319 | P3-C2, H6, uncapped bodies | `http.MaxBytesReader` on all JSON endpoints in users/read/fetcher/email + depth guards on the two recursive email HTML walkers | 1 | [R5](#r5--resource-bounding) |

#### Wave 2 — Broken features and ingestion correctness

The RSS/email→content write path fails under *normal* traffic. These need real-Postgres or wire-level tests (the current mocks are exactly why they were missed).

| Done | Finding | Summary | P | Recipe |
|---|---|---|---|---|
| [x] #321 | P2-C6 | Bulk content insert PK-collides on any mixed new+seen batch → whole RSS batch fails | 1 | [R7](#r7--db-write-path-bugs) |
| [x] #322 | P2-C7 / H8 | Recommender upsert conflicts on content-hash `id`, not just `link` → batch dropped every poll cycle | 1 | [R7](#r7--db-write-path-bugs) |
| [x] #323 | Dedup index | `idx_contents_rss_dedup` is not UNIQUE → concurrent deliveries create duplicate content rows; email/manual path has no dedup at all | 1 | [R7](#r7--db-write-path-bugs) |
| [x] | P2-C8 | Feed-subscribe client unmarshals wrong response shape → app gets empty data with HTTP 201 | 1 | [R8](#r8--cross-service-contract-drift) |
| [x] #331 | 409→500 | Client matches error codes the server never sends → re-subscribe returns 500 instead of 409 | 2 | [R8](#r8--cross-service-contract-drift) |
| [x] #327 | C3 | Refresh-token reuse detection never fires for its actual threat; its test asserts through the wrong path | 1 | [R9](#r9--dead-tests-and-inert-security-features) |
| [x] #320 | P2-C3 | Two dead tests: outbox delivery test never calls the code under test; CORS test can't catch the bypass. Fix C4 (suffix-match bypass) with the test | 1 | [R9](#r9--dead-tests-and-inert-security-features) |
| [x] #326 | C5 | Outbox client's internal 1m→12h retry loop blocks the whole worker for up to ~17h | 1 | [R6](#r6--operational-fixes) |
| [ ] | H3 + H4 | Password-reset tokens never delivered; self-host build panics on reset/verify routes (missing wiring) — fix or explicitly remove the routes | 2 | [R6](#r6--operational-fixes) |
| [ ] | Cleanup batching | `RunWithBatching` doesn't batch: unbounded `DELETE` in one transaction (correct pattern exists at `fetcher/.../feed_item.go:379-401`) | 2 | [R7](#r7--db-write-path-bugs) |

#### Wave 3 — Ratchets and observability (make quality stick)

| Done | Finding | Summary | P | Recipe |
|---|---|---|---|---|
| [x] #328 | Web CI | `apps/web` has **no CI at all** (verified 2026-08-09): add a workflow running `tsc --noEmit`, `eslint`, `vitest` on PRs touching `apps/web/**` — mirror `mobile-checks.yml` | 1 | [R10](#r10--ci-ratchets) |
| [x] #330 | Integration tier | Add a CI job running the `//go:build integration`-tagged tests against a real Postgres service container (repository-layer tests from Wave 2 live here) | 1 | [R10](#r10--ci-ratchets) |
| [ ] | users build tag | `services/users/test/integration` lacks the `//go:build integration` tag every sibling uses → bare `go test ./...` hangs | 2 | [R10](#r10--ci-ratchets) |
| [ ] | H10 + logging | Recovery middleware registered before request-ID middleware in all 6 routers → panics logged with `request_id=unknown`; per-request logger used in only one handler repo-wide | 2 | [R6](#r6--operational-fixes) |
| [ ] | Sentinel bug | Routine token expiry logs at ERROR (wrong sentinel: `auth.ErrTokenExpired` vs `apperrors.ErrTokenExpired`) — makes error-rate monitoring useless | 2 | [R6](#r6--operational-fixes) |
| [ ] | Worker liveness | Outbox/fetcher workers: heartbeat log + `recover()` in loop + circuit-breaker state via slog (currently `fmt.Printf`) | 2 | [R6](#r6--operational-fixes) |
| [ ] | Readiness | Self-host `/health/ready` checks only 3 of 6 DBs → reports healthy while half the system is down (and CI smoke-curls it → false green) | 2 | [R6](#r6--operational-fixes) |
| [ ] | P2-C4 | Decide the prod deploy story: image publishing is `if: false`, Vault init misses two AppRoles, dev compose broken for content-service. Either fix `prod/` or delete it and document selfhost as the only path | 2 | needs owner decision — write up options in the task, ask |

#### Wave 4 — Consolidation (make fixes land once)

Do this *after* Waves 1–2: consolidation is safest when the behavior being consolidated is already correct and tested.

| Done | Finding | Summary | P | Recipe |
|---|---|---|---|---|
| [ ] | H11 | Delete the ~250-line dead JWT-validation duplicate in `services/users/internal/auth/jwt.go`; everything uses `pkg/auth` | 2 | [R11](#r11--consolidating-duplicates) |
| [ ] | Fetch dedup | Collapse the 4+ HTTP fetch+size-cap copies onto `pkg/rss/fetch` (which by now carries the SSRF guard from Wave 1) | 2 | [R11](#r11--consolidating-duplicates) |
| [ ] | Env parsing | Collapse `pkg/env` vs `pkg/config` vs two service-local copies into one, with one duration-parsing behavior | 3 | [R11](#r11--consolidating-duplicates) |
| [ ] | Email sanitizer | Replace email's hand-maintained bluemonday policy with `pkg/rss/sanitize` | 2 | [R11](#r11--consolidating-duplicates) |
| [ ] | FE auth layer | Move the near-verbatim web/mobile `auth.ts` token state machine into `apps/shared` (injectable-adapter pattern already demonstrated there); fix H12 (swallowed second 401) and offline-clears-tokens in the shared copy | 2 | [R11](#r11--consolidating-duplicates) |
| [ ] | Theme 4 | `FOR UPDATE SKIP LOCKED` on all job-claim queries (read/fetcher, explore/fetcher, email outbox) — prerequisite for ever running >1 replica | 2 | [R7](#r7--db-write-path-bugs) |
| [ ] | FE resilience | React error boundaries (both apps), mobile list error states, mobile `accessibilityLabel` on the reader action bar, destructive-action confirmations (web) | 2 | standard FE work; test per [§2.2](#22-choosing-the-test-level) |

#### Wave 5 — Steady state

No fixed task list. Switch to the [continuous process in §1.4](#14-finding-new-issues-once-the-ledger-drains): mediums/lows from the review (batched per service), plus newly discovered issues.

### 1.4 Finding new issues once the ledger drains

When the review backlog is exhausted, generate new targets in this order of cost-effectiveness:

1. **Churn × complexity hotspots.** Files that are both large and frequently edited harbor the next bugs:
   ```bash
   git log --since="6 months ago" --name-only --pretty=format: \
     | grep -E '\.(go|ts|tsx)$' | sort | uniq -c | sort -rn | head -20
   ```
   Cross-reference against `wc -l`. Review the intersection first.
2. **Re-run the review's axes on new code.** The original review worked because each pass took a *different axis* (per-service static, cross-service contracts, DB semantics, dependencies, DoS, tenancy matrix). Any service or feature added since 2026-07-05 has had none of these passes. Apply the same axes to the diff.
3. **The Cairn-specific smell checklist.** Every critical in the review is an instance of one of these. When touching any file, grep for the pattern; when one is found anywhere, file a chalk task:
   - A route registered outside `RequireAuth`/`RequireInternalAPIKey` whose handler reads a `user_id` or writes shared state.
   - `json.NewDecoder(r.Body)` without `http.MaxBytesReader` upstream.
   - Multi-row `INSERT` without `ON CONFLICT` on a table with any uniqueness expectation.
   - A client `struct` for another service's response, hand-maintained (vs. the shared-type pattern `pkg/models.Article`).
   - `SELECT ... WHERE status='pending'` followed by a status update, without `FOR UPDATE SKIP LOCKED`.
   - A test whose assertions would still pass if the function under test were never called (assign-to-`_`, `assert.Error` with no error-type check).
   - Logging anything derived from a token, password, or verification/reset secret.
   - Recursive walks over attacker-supplied structures without a depth guard.
4. **Scheduled dependency sweep.** Monthly: `govulncheck ./...` per Go module, `npm audit` per JS workspace, and check the Go toolchain patch version. This alone caught a live XSS last time.

---

## Part 2 — How to fix things

### 2.1 The universal loop

Every task, regardless of class:

1. **Re-verify.** Open the cited files; confirm the problem exists on `main` today. Findings cite file:line from 2026-07-05 — lines drift.
2. **Write the failing test** at the level chosen from §2.2. Run it; watch it fail *for the reason the finding describes* (not a compile error or setup bug).
3. **Fix minimally**, copying the reference pattern from §2.3 where one exists.
4. **Prove:** your new test passes; the full relevant suite passes (§2.4); lint passes.
5. **PR** with: finding ID in the title, one-paragraph before/after, and the ledger row ticked.

### 2.2 Choosing the test level

The single biggest lesson of the review: **most of these bugs were invisible to unit tests with mocks.** The test must live at the level where the bug lives.

| Bug class | Wrong level (why it missed the bug) | Right level |
|---|---|---|
| Missing auth on a route | Handler test (bypasses the router, so middleware never runs) | `httptest` against the **real router constructor** — assert 401/403 for a request with no/wrong credentials on the exact path |
| DB semantics (PK collision, missing ON CONFLICT, non-unique index, locking) | Mocked repository (mocks don't enforce constraints) | Integration test (`//go:build integration`) against real Postgres with the real migrations applied |
| Cross-service contract drift | Unit test of client or server alone (each agrees with itself) | Decode the client struct from a **captured real response** of the server handler (run the server handler in-process via `httptest`, point the real client at it) |
| Wrong-path tests / inert security features | — | First strengthen the assertion until it fails on today's code (assert the *specific* error type / audit event / side effect), then fix the code |
| SSRF / input bounding | Testing the happy path | Table-driven adversarial inputs: loopback, RFC1918, link-local, metadata IPs, redirects-to-internal; oversized bodies (assert 413); deep nesting (assert bounded, e.g. 10k nested divs — do **not** write a test that actually overflows the stack) |
| Timing/retry behavior (C5) | Test that patches delays to ~0 (hides the 17h stall) | Inject a clock/derive delays from config; assert on the *schedule decision*, not wall time |
| Log hygiene | — | Capture the slog output in the test (swap in a buffer handler); assert secret material absent from every line of the flow |

### 2.3 Reference patterns — copy these, don't invent

The review explicitly identified correct implementations already in the repo. When your task matches, mirror the reference:

| Need | Reference |
|---|---|
| Internal-endpoint auth | `RequireInternalAPIKey` as applied to `/api/v1/internal/*` in read/content's router |
| Request body caps (413) | `explore/recommender` handlers' `http.MaxBytesReader` usage |
| Bounded batch delete | `services/read/fetcher/.../feed_item.go:379-401` |
| HTTP error handling in clients | `Unsubscribe` in the content service's ingest-rss client — branches on HTTP **status**, not error-string matching |
| HTML sanitization | `pkg/rss/sanitize` (the canonical policy) |
| Fetch with limits | `pkg/rss/fetch` (redirect limit + `io.LimitReader`) |
| Cross-service DTO that can't drift | `pkg/models.Article` shared by both sides of the explore seam |
| Front-end URL-injectable adapter | `apps/shared` server-URL pattern |

### 2.4 Verification commands

Run the slice relevant to your change; run the repo-wide set before opening the PR.

```bash
# Go (per module — cd into the module first)
gofmt -l . && go vet ./... && golangci-lint run && go test -race -count=1 ./...

# Integration tier (needs local Postgres; see docs/TESTING.md)
go test -race -tags=integration ./...

# Web
cd apps/web && npx tsc --noEmit && npx eslint . && npx vitest run

# Mobile
cd apps/mobile && npx tsc --noEmit && npx eslint . && npx jest --ci

# Dependency checks (Wave 1 dep task + monthly)
govulncheck ./...        # per Go module
npm audit                # per JS workspace
```

CI runs gofmt, golangci-lint, go vet, and `go test -race` per service (`go-checks.yml`), typecheck/lint/jest for mobile (`mobile-checks.yml`), and typecheck/lint/vitest for web (`web-checks.yml`).

### 2.5 Recipes by finding class

#### R1 — Closing an authorization gap

1. Failing test first: `httptest` against the real router; request the route with no credentials; assert it currently succeeds (this documents the hole), then flip the assertion to expect 401 and make it pass.
2. Decide the boundary type: user-facing → `RequireAuth` (JWT); service-to-service → `RequireInternalAPIKey`. For read/fetcher's user routes, JWT is correct *if* callers have a user context; otherwise internal key **plus** the resource-owning service re-checking ownership. State your choice in the PR.
3. Apply the middleware at the route group in the router — never inside handlers.
4. Update every caller (grep for the path across `services/`) to send the credential, and the service's `openapi.yaml` + CLAUDE.md route table.
5. Add one **router-inventory test**: enumerate the router's routes and assert every route is in an explicit allowlist of intentionally-public paths (`/health/*`, login, etc.). This is the ratchet that stops the next unauthenticated route at PR time.

#### R2 — SSRF-safe fetching

1. Implement one guarded dialer in `pkg/rss/fetch` (check resolved IPs against loopback/RFC1918/link-local/metadata ranges at `DialContext` time — checking the hostname before resolution is bypassable via DNS).
2. Table-driven tests: each blocked range, a redirect from a public URL to an internal one, and DNS names resolving to internal IPs (use a test resolver).
3. Wire it into `pkg/rss/fetch` only. The other fetch copies get deleted in Wave 4; if a Wave-1 endpoint uses a copy (e.g. `url_detector.go`), point that call site at the shared guard now.

#### R3 — Credential material in logs

1. Grep the flagged files (and their whole services) for log calls near token/secret variables.
2. Test: run the flow (refresh, verify-email) with a buffer-backed slog handler; assert no line contains the token material. Then strip the logging.
3. Mobile: gate diagnostics behind `__DEV__`; never log response bodies of auth endpoints.

#### R4 — Dependency bumps

Order matters — cheapest, broadest first:
1. Go toolchain patch bump (all 12 modules' `go` directive + CI `go-version`).
2. `golang.org/x/net` ≥ v0.55.0 everywhere; `pgx` to one version ≥ v5.9.2 everywhere; `google/uuid` in `pkg/auth` to v1.6.0.
3. `go mod tidy` per module; full verification suite; `govulncheck` to confirm the findings clear.
4. Mobile Expo major bump is a **separate task** (real migration risk) — don't bundle it.

#### R5 — Resource bounding

1. `http.MaxBytesReader` per handler group, mirroring explore/recommender's limits (tight caps for auth/JSON endpoints, larger for content ingestion). Assert 413 with an oversized body, success just under the cap.
2. Depth guards in `content_extractor.go` / `email_cleaner.go` walkers: max-depth parameter (100 is generous for real HTML); truncate or reject beyond it. Test with programmatically nested HTML (~10k deep) — assert graceful handling, don't crash the test runner proving the old bug.

#### R6 — Operational fixes

- **P2-C1 (key refresh):** add periodic re-fetch (interval ≪ rotation interval) + re-fetch-once on signature-validation failure (handles out-of-band rotation). Test with a fake key source: rotate, assert validation recovers without restart.
- **C5 (outbox blocking):** delete the client's internal sleep-retry loop entirely — DB-level backoff already exists and is correct. On failure: mark the entry, move on. Test: first entry's downstream fails → assert entry 2 is attempted within the same batch pass.
- **H10 (middleware order):** reorder so request-ID/logger middleware wraps Recovery in all 6 routers; test: panicking test handler → assert the panic log line carries the real request ID.
- **Sentinel bug:** compare against the sentinel the validator actually returns; expiry logs at INFO/WARN. Test asserts the level.
- **Liveness:** heartbeat log per worker tick (even when idle), `recover()` per iteration, circuit-breaker transitions via slog at WARN.

#### R7 — DB write-path bugs

All of these need `//go:build integration` tests against real Postgres with real migrations (testcontainers is already used in explore — reuse `internal/testutil`).

- **P2-C6:** integration test: insert a batch mixing one already-stored item and one new → assert both survive and the new one is created. Fix: filter pre-existing rows before `BulkCreate` (preferred — explicit) or `ON CONFLICT (id) DO NOTHING`.
- **P2-C7:** two articles, different links, same content hash → assert the batch survives and dedup is deliberate. Arm the `id` conflict.
- **Dedup index:** make `idx_contents_rss_dedup` UNIQUE via migration (**check for existing duplicate rows first** — write the dedup backfill in the same migration) + `ON CONFLICT DO NOTHING` at insert. Extend dedup to the email/manual path (currently rss-only).
- **Unbounded deletes:** add `LIMIT` plumbing so `RunWithBatching`'s batch size reaches the SQL; copy `feed_item.go:379-401`. Test: seed rows > batch size, assert multiple bounded passes.
- **Theme 4 (Wave 4):** `FOR UPDATE SKIP LOCKED` in claim queries. Test: two concurrent claimers on the same pending set → assert disjoint claims.

#### R8 — Cross-service contract drift

1. Reproduce at the seam: run the real server handler via `httptest`, point the real client at it, assert the client's decoded struct is fully populated (P2-C8's bug: zero values with a 201).
2. Fix the client to unwrap the `pkg/api` `{data,meta}` envelope like its sibling `ListUserSubscriptions` already does.
3. For error translation: branch on HTTP status codes, not error-code strings (`Unsubscribe` is the reference). Assert 409-shaped server responses surface as 409 to the app.
4. Wherever a hand-maintained response struct caused the drift, note it as a Wave-4/5 candidate for the shared-type pattern; don't restructure in the bugfix PR.

#### R9 — Dead tests and inert security features

1. First make the existing test honest: strengthen assertions until it **fails on current `main`** for the finding's reason (e.g. C3: assert the `token_reuse_detected` audit event and family revocation actually occur in the replay-after-rotation scenario; P2-C3 CORS: add `evilexample.com` and `example.com.evil.com` cases).
2. Then fix the code (C3: stop hard-deleting rotated tokens — mark revoked and retain for the reuse-detection window; C4: anchor the suffix match with a leading dot).
3. In the PR, state what the old test actually exercised — this is how the team learns to spot the pattern.

#### R10 — CI ratchets

Ratchets convert one-time fixes into permanent floors. Each is a small standalone PR:
1. **web-checks.yml:** copy `mobile-checks.yml`'s shape, path-filtered on `apps/web/**`: `tsc --noEmit`, `eslint .`, `vitest run`.
2. **Integration-test job** in `go-checks.yml`: Postgres service container, `go test -tags=integration` per service that has tagged tests. Wave 2's tests are the payload; this job is what keeps them running.
3. **users build tag:** add `//go:build integration` to `services/users/test/integration/*.go` (match siblings).
4. As router-inventory tests (R1 step 5) land per service, they ride the normal test job — no extra CI config.

#### R11 — Consolidating duplicates

Consolidation deletes code; the risk is silent behavior change. Sequence per duplicate:
1. **Characterize first:** write tests against the *canonical* implementation covering the behaviors the duplicate's callers rely on (diff the two implementations to find divergences — e.g. the env-parsing copies disagree on duration semantics; pick one behavior *explicitly* and note it in the PR).
2. Repoint callers to the canonical implementation one call-site group at a time.
3. Delete the duplicate **in the same PR** as the last repoint — a surviving duplicate defeats the purpose.
4. Front-end auth consolidation: port web/mobile `auth.ts` tests over to the shared module first, then fix H12 + offline-token-clearing in the shared copy so both platforms inherit the fix.

### 2.6 Definition of done (every task)

- [ ] Finding re-verified on current `main` before work started (or task closed as stale).
- [ ] A test exists that failed before the fix and passes after, at the level §2.2 prescribes.
- [ ] Relevant §2.4 verification suite green locally (including web checks until web CI exists).
- [ ] Docs touched by the change updated: service `openapi.yaml`, service `CLAUDE.md` route tables, `docs/ARCHITECTURE.md` if a boundary changed.
- [ ] Ledger row in §1.3 ticked with PR number; chalk task closed.
- [ ] PR description: finding ID, what the failing test proves, any behavior deliberately changed.

---

## Appendix — current-state facts this strategy relies on (verified 2026-08-09)

- Review document merged 2026-07-05 (PRs #300, #301); commits since then are styling/docs/mobile-archive only → backlog is essentially untouched.
- CI: `go-checks.yml` = gofmt + golangci-lint + go vet + `go test -race` per service; `mobile-checks.yml` = tsc + eslint + jest + knip; **no workflow runs any check on `apps/web`**; `docker-test.yml` = image builds + selfhost compose smoke (which curls the readiness endpoint that under-reports — Wave 3).
- Test volume: Go 224 source / 111 test files; front-ends 111 source / 10 test files (mobile-heavy) — but per the review, coverage *quality* is the issue, not volume.
- Task tracking: chalk (`.chalk/tasks/`), umbrella epic for this program: `epic_fefa`.
