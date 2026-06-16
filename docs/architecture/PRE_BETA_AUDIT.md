# Pre-Beta Architecture Audit — Proposed Next Steps

**Epic:** `epic_9c21` — Pre-beta architecture audit
**Status:** Findings consolidated — decision-ready
**Date:** 2026-06-16
**Source workstreams:** `docs/architecture/audit/01_api_contract.md` … `06_infra.md`

> This is the consolidation of the six audit workstreams into a single go/no-go
> document. It exists to answer one question — *can we open Cairn to a public beta
> today, and if not, what is the shortest path to "yes"* — and to make sure the
> expensive-to-change decisions (API contracts, data shapes, auth model) are made
> **before** the contract freezes under shipped clients.

---

## 1. Executive summary

**Verdict: NOT YET — conditional go.** The system is architecturally sound and correct at
dev scale. The codebase's security baseline is genuinely good (bcrypt-12, RS256 with
algorithm enforcement, refresh-token rotation with reuse detection, IDOR protection,
parameterized queries, HTML sanitization), the data model is sensible, and the service
boundaries are clean. **It is not, however, ready to accept public signups today.** Three
classes of problem block a public beta, and all three are cheapest to fix now:

1. **Contract-freeze items.** The OpenAPI specs that a beta freeze locks do *not* match
   the implemented routes (read content + RSS fetcher), the error envelope diverges at the
   middleware layer, and two correctness bugs in the mobile client (silently-wrong vote
   counts, silently-truncated explore search) require *new* server endpoints that must be
   specced before the contract freezes. Ship these after clients are in the wild and you
   pay for them with forced upgrades and dual-support.
2. **Public-surface security gaps.** No per-account login lockout (only IP rate limiting,
   trivially bypassed by distributed brute force), email validation that only checks for
   `@` with no verification flow, and no password-reset/account-recovery path. These are
   table stakes for a public signup form.
3. **Operational blind spots & data-durability risk.** There is **zero** metrics
   instrumentation — no `/metrics`, Prometheus, or Grafana — so production latency, error
   rates, and saturation are unobservable. Backups are documented but the referenced script
   does not exist and nothing is automated. The prod Postgres `max_connections` is unset
   (default 100) while the running containers can demand ~175 connections.

None of these is a deep redesign. The beta-blocking list below is ~10 concrete,
mostly-small remediations plus two existing tasks (metrics, backups) that need
unblocking. Once they land, Cairn is ready for a **single-instance private/public beta**
with a documented scaling ceiling. The horizontal-scaling, HA-database, and Vault-KMS work
is correctly deferred to roadmap.

> **Process note:** several existing remediation tasks (`task_644a`, `task_652a`,
> `task_5dcb`, `task_48ea`, `task_6f3a`, `task_8392`, `feature_1a69`) are currently
> **blocked on `decision_4052`** ("Decide to offer hosted service"). The metrics and
> backup items among these are beta-blocking regardless of the hosting decision — that
> decision should be made (or these tasks de-coupled from it) so the work can start.

---

## 2. How to read the classification

| Class | Meaning |
|---|---|
| **BETA-BLOCKING** | Freezes/breaks a client contract, exposes an exploitable gap, or risks data loss / unobservable production. Must land before signups open. |
| **FAST-FOLLOW** | Real gap, but internal/changeable mid-beta without breaking shipped clients. Fix during early beta. |
| **ROADMAP** | Correct at beta scale; documented so it isn't rediscovered as a surprise. Post-beta hardening/scaling. |

---

## 3. Per-workstream findings

Each finding links to its detailed write-up in `docs/architecture/audit/`. Severity, current
state, recommendation, and effort are summarised; the source docs carry the file:line evidence.

### 3.1 API contract & stability — `01_api_contract.md` (`task_2a40`)

| ID | Finding | Class | Effort |
|---|---|---|---|
| F-1 | Handlers converged on the `pkg/api` envelope, but middleware paths (auth 401, panic 500, content-type 400) emit divergent shapes a client can't parse uniformly | **BETA-BLOCKING** | S |
| F-2 | Health-check format inconsistent (email returns plain-text `OK`, explore adds non-standard `timestamp`) | FAST-FOLLOW | S |
| F-3 | Pagination mixed (cursor vs offset vs none); param + response shape must freeze deliberately; subscriptions aggregator unbounded | **BETA-BLOCKING** (shape) / FAST-FOLLOW (unbounded) | M |
| F-4 | Read **content** OpenAPI spec documents paths that don't exist (`/users/{id}/contents` vs real `/content/user/{id}`) | **BETA-BLOCKING** | S |
| F-5 | Read **fetcher** (RSS) OpenAPI spec drifts (`/users/{id}/feeds` vs real `/source/rss/user/{id}/subscription`) | **BETA-BLOCKING** | S |
| F-6 | users/explore/fetcher specs document an outdated/partial error schema; email spec is correct (use as template) | **BETA-BLOCKING** | S |
| F-7 | Routes implemented but absent from specs; explore spec conflates recommender + fetcher | FAST-FOLLOW | S |
| — | Versioning: every public endpoint under `/api/v1` | PASS | — |

**Reconciliation rule for all spec drift:** the shipped mobile client is ground truth — make
the **spec match the implementation**, never the reverse, because the client is already pinned.

### 3.2 Data layer & query efficiency — `02_data_layer.md` (`task_e695`)

| ID | Finding | Class | Effort |
|---|---|---|---|
| D-1 | User-content list + search hydrate full `cleaned_html` per item (≤5 MB × up to 100) — tens of MB per list page | **BETA-BLOCKING** (contract) | M |
| D-2 | No statement/query timeout in any pool or the repo layer — a few slow FTS queries exhaust the 25-conn pool | **BETA-BLOCKING** (abuse guard) | S |
| D-3 | Index review: both "missing index" leads resolve to no-action; one real gap is `(user_id, status, added_at)` | FAST-FOLLOW | S |
| D-4 | Pool/driver inconsistency (lead's pgx/pq split is backwards; lifetimes diverge 5m vs 1h; only explore env-tunable) | FAST-FOLLOW | M |
| D-5 | Both `BulkCreate`s loop row-by-row instead of one multi-row INSERT | FAST-FOLLOW | S |
| D-6 | Unused `user_unread_counts` view is a latent `users × articles` CROSS JOIN (dead code) | FAST-FOLLOW | S |
| D-7 | Offset pagination on `votes`/senders degrades at depth (overlaps F-3) | FAST-FOLLOW | M |
| D-8 | Migrations versioned/reversible, but explore ships duplicate legacy files and destructive downs are untested | FAST-FOLLOW | S |
| D-9 | feed-limit `COUNT(*)`-per-insert trigger and votes growth — correct at beta scale (votes display uses denormalised counters) | ROADMAP / NOTE | — |

### 3.3 Mobile client/server efficiency — `03_mobile.md` (`task_1eaa`)

| ID | Finding | Class | Effort |
|---|---|---|---|
| M-1 | `getUserVoteStats()` sends `limit=10000`; server silently caps at 100 (default 20) → **vote counts are wrong** for active users | **BETA-BLOCKING** | S |
| M-2 | Explore search is a client-side `useMemo` filter over in-memory articles only → **silent result truncation**, no backend `?q=` call | **BETA-BLOCKING** | M |
| M-3 | `useFocusEffect` full-reset refetch on Read/You/Bookmarks/Votes screens — fresh round-trip on every tab switch | FAST-FOLLOW | S |
| M-4 | No retry/back-off for transient 5xx/network; only 401→refresh→retry exists; no per-request timeout except `detectURL` | FAST-FOLLOW | S |
| M-5 | AsyncStorage is write-only from the list's view; load blocks on network → blank screen offline | FAST-FOLLOW | M |
| M-6 | Explore page size (10/page) matches backend; initial multi-page buffer is an intentional viewport-fill, not waste | PASS | — |
| M-7 | Token refresh dedup + proactive (5-min) refresh implemented correctly | PASS | — |

M-1 and M-2 each require a **new server endpoint** (see API-freeze checklist) — these are why
mobile correctness is a contract-freeze concern, not just a client polish item.

### 3.4 Security & auth hardening — `04_security.md` (`task_659b`)

Baseline confirmed-good (verified, not re-litigated): bcrypt-12, RS256 + algorithm
enforcement, refresh-token rotation + reuse-detection family revocation, SHA-256 token
hashing, `RequireSameUser` + service-layer IDOR checks, parameterized queries, bluemonday
sanitization, security headers, HTTPS enforcement, no hardcoded secrets.

| ID | Finding | Class | Effort |
|---|---|---|---|
| S-1 | No per-account failed-login tracking / lockout — only IP rate limit (10/min), bypassed by distributed brute force | **BETA-BLOCKING** | M |
| S-2 | Email validation is `strings.Contains(@)` only — no RFC check, no verification flow (3 call sites) | **BETA-BLOCKING** | M |
| S-3 | No password reset / account-recovery flow | **BETA-BLOCKING** | M |
| S-4 | Rate limiter is in-memory/per-instance (cross-ref I-4) | FAST-FOLLOW | M |
| S-5 | No structured auth audit log (logins, refreshes, account changes) | FAST-FOLLOW | S |
| S-6 | Vault is a hard startup dependency with no offline key fallback | FAST-FOLLOW | S |
| S-7 | No CAPTCHA / bot protection on public signup | ROADMAP | M |

S-2 and S-3 add **three new public endpoints** (`verify-email`, `forgot-password`,
`reset-password`) and an `email_verified` field — these feed the API-freeze checklist.

### 3.5 Observability & operational readiness — `05_observability.md` (`task_ed30`)

| ID | Finding | Class | Existing task | Effort |
|---|---|---|---|---|
| O-1 | No metrics endpoint, no Prometheus, no Grafana, no alerting — production unobservable | **BETA-BLOCKING** | `task_644a` + `task_652a` | L |
| O-2 | No cross-service request-trace propagation (X-Request-ID stays within a service) | FAST-FOLLOW | NEW | S |
| O-3 | No container resource limits in prod/selfhost compose | FAST-FOLLOW | `task_48ea` | S |
| O-4 | Integration tests exist + are tagged but CI never runs them — inter-service breakage merges undetected | FAST-FOLLOW | `task_6f3a` | M |
| O-5 | User-service `http.Server` has no Read/Write/Idle timeouts (infinite) → slow-loris exhausts the auth gateway | **BETA-BLOCKING** | NEW | S |
| O-6 | Log format defaults to `text` in prod (`LOG_FORMAT` unset); no log shipping stack | FAST-FOLLOW | `feature_1a69` | S→M |

### 3.6 Infrastructure reliability & scaling — `06_infra.md` (`task_1d08`)

| ID | Finding | Class | Decision needed | Effort |
|---|---|---|---|---|
| I-1 | Single PostgreSQL instance — no replication/failover | **BETA-BLOCKING** | Managed DB *or* documented single-instance + rehearsed restore | M |
| I-2 | Backups not automated — `scripts/backup.sh` referenced in docs but **does not exist**; no cron, no offsite | **BETA-BLOCKING** | Execute `task_5dcb` before beta | M |
| I-3 | Vault file storage (not HA) + all 5 Shamir unseal keys stored plaintext in `vault-keys` volume | **BETA-BLOCKING** | Accept + back up volume for beta; `task_ece2` pre-public-launch | M |
| I-4 | In-memory rate limiter breaks under multiple instances (currently single-instance by design) | FAST-FOLLOW | Document single-instance topology; defer Redis | S |
| I-5 | `sslmode=disable` on all internal DB connections | FAST-FOLLOW | Accept within Docker network; document | S |
| I-6 | Connection ceiling: 9 containers × ≤25 conns = **~175** vs Postgres default `max_connections=100` (**unset** in prod compose) | **BETA-BLOCKING** | Set `max_connections`≥200 in prod + cap pools | S |
| I-7 | Email `X-API-Key` has no rotation tooling, no expiry, no per-key rate limit (hash storage is correct) | FAST-FOLLOW | Rotation runbook + per-key limit | M |

---

## 4. Proposed next steps (prioritised)

### 4.1 BETA-BLOCKING — must fix before opening signups

| # | Action | Findings | Task |
|---|---|---|---|
| 1 | Reconcile OpenAPI specs with implemented routes + unify error envelope through `pkg/api` | F-1, F-3, F-4, F-5, F-6 | `task_065e` |
| 2 | Split content list vs detail: drop `cleaned_html` from list/search responses (summary projection) | D-1 | `task_bfad` |
| 3 | Add `statement_timeout` to all API DB connection pools (centralise in `pkg/config`) | D-2 | `task_0b27` |
| 4 | Add `GET /api/v1/explore/user/vote-stats` aggregate endpoint; remove the `limit=10000` client count | M-1 | `task_12e0` |
| 5 | Add `GET /api/v1/explore/search` server-side search (or remove the Explore search UI) | M-2 | `task_0502` |
| 6 | Per-account failed-login tracking + lockout | S-1 | `task_f416` |
| 7 | RFC email validation + email verification flow | S-2 | `task_2c1f` |
| 8 | Forgot-password / reset-password flow | S-3 | `task_fe44` |
| 9 | Set Read/Write/Idle timeouts on the user-service HTTP server | O-5 | `task_fc2f` |
| 10 | Set Postgres `max_connections` + cap connection pools in prod compose | I-6 | `task_7683` |
| 11 | Metrics + dashboards (`/metrics`, Prometheus, Grafana) | O-1 | `task_644a` (existing) |
| 12 | Critical alerts (error rate, latency, health, pool utilisation) | O-1 | `task_652a` (existing) |
| 13 | Automated, tested, offsite backups (ship `backup.sh` + cron container) | I-2 | `task_5dcb` (existing) |
| 14 | Single-instance SPOF decision recorded + restore runbook rehearsed | I-1 | covered by #13 + decision |

### 4.2 FAST-FOLLOW — fix within the first weeks of beta

| # | Action | Findings | Task |
|---|---|---|---|
| 15 | Mobile resilience: focus-refetch TTL, retry/back-off, offline stale-fallback | M-3, M-4, M-5 | `task_5229` |
| 16 | Standardise health-check JSON; document undocumented routes / split explore spec | F-2, F-7 | `task_396d` |
| 17 | Data-layer cleanup: `(user_id,status,added_at)` index, pool unification, multi-row INSERT, drop unused view, migration up→down→up CI | D-3, D-4, D-5, D-6, D-8 | `task_9208` |
| 18 | Security fast-follow: Redis-backed rate limiter, auth audit log, Vault startup retry | S-4, S-5, S-6 | `task_126b` |
| 19 | Propagate X-Request-ID across service-to-service HTTP calls | O-2 | `task_9b71` |
| 20 | Infra fast-follow: document single-instance topology, sslmode note, email key rotation + per-key limit | I-4, I-5, I-7 | `task_c542` |
| 21 | Container resource limits in prod/selfhost compose | O-3 | `task_48ea` (existing) |
| 22 | Run integration tests in CI against a Postgres service container | O-4 | `task_6f3a` (existing) |
| 23 | `LOG_FORMAT=json` in prod + log shipping/aggregation stack | O-6 | `feature_1a69` (existing) |
| 24 | Cursor pagination shape for `votes`/senders (freeze param/response now, swap to keyset when needed) | D-7, F-3 | folded into `task_065e` / `task_9208` |
| 25 | Optimize N+1 query in recommendation recording | (Audit 2 cross-ref) | `task_3216` (existing) |

### 4.3 ROADMAP — post-beta scaling & hardening

| # | Action | Findings | Task |
|---|---|---|---|
| 26 | Managed / replicated Postgres (HA, failover) | I-1 | roadmap |
| 27 | Vault KMS auto-unseal (remove plaintext unseal keys) | I-3 | `task_ece2` (existing) |
| 28 | Load testing to size resource limits | (informs O-3) | `task_8392` (existing) |
| 29 | JTI claim for JWT token revocation | (Audit 4 cross-ref) | `feature_e001` (existing) |
| 30 | CAPTCHA / bot protection on public signup | S-7 | roadmap |
| 31 | Horizontal scaling story (multi-instance, shared state) | I-4, S-4 | roadmap |

---

## 5. Action → chalk task mapping

Newly created under `epic_9c21` by this consolidation:

| Task | Title | Class | Findings |
|---|---|---|---|
| `task_065e` | API freeze: reconcile OpenAPI specs with routes + unify error envelope | BETA-BLOCKING | F-1,F-3,F-4,F-5,F-6 |
| `task_bfad` | Split content list vs detail payload (drop `cleaned_html` from lists) | BETA-BLOCKING | D-1 |
| `task_0b27` | Add `statement_timeout` to all API DB connection pools | BETA-BLOCKING | D-2 |
| `task_12e0` | Add explore vote-stats aggregate endpoint | BETA-BLOCKING | M-1 |
| `task_0502` | Add explore server-side search endpoint | BETA-BLOCKING | M-2 |
| `task_f416` | Per-account failed-login tracking + lockout | BETA-BLOCKING | S-1 |
| `task_2c1f` | RFC email validation + email verification flow | BETA-BLOCKING | S-2 |
| `task_fe44` | Forgot-password / reset-password flow | BETA-BLOCKING | S-3 |
| `task_fc2f` | Set HTTP timeouts on user-service | BETA-BLOCKING | O-5 |
| `task_7683` | Set Postgres `max_connections` + cap pools in prod compose | BETA-BLOCKING | I-6 |
| `task_5229` | Mobile resilience (TTL, retry/backoff, offline fallback) | FAST-FOLLOW | M-3,M-4,M-5 |
| `task_396d` | Standardise health-check JSON + document undocumented routes | FAST-FOLLOW | F-2,F-7 |
| `task_9208` | Data-layer fast-follow cleanup | FAST-FOLLOW | D-3,D-4,D-5,D-6,D-8 |
| `task_126b` | Security fast-follow (Redis limiter, audit log, Vault retry) | FAST-FOLLOW | S-4,S-5,S-6 |
| `task_9b71` | Propagate X-Request-ID across services | FAST-FOLLOW | O-2 |
| `task_c542` | Infra fast-follow (single-instance docs, sslmode, email key rotation) | FAST-FOLLOW | I-4,I-5,I-7 |

Existing tasks linked (not duplicated):

| Task | Title | Class | Findings | Note |
|---|---|---|---|---|
| `task_644a` | Add Prometheus metrics and Grafana dashboards | BETA-BLOCKING | O-1 | blocked on `decision_4052` — unblock |
| `task_652a` | Set up alerting system for pre-go-live | BETA-BLOCKING | O-1 | blocked on `decision_4052` — unblock |
| `task_5dcb` | Add database backup script and cron container | BETA-BLOCKING | I-2 | blocked on `decision_4052` — unblock |
| `task_48ea` | Add Docker resource constraints to production compose | FAST-FOLLOW | O-3, I-6 | pair with `task_7683` |
| `task_6f3a` | Add integration test workflow with Docker/Postgres | FAST-FOLLOW | O-4 | |
| `feature_1a69` | Add log monitoring/aggregation stack for production | FAST-FOLLOW | O-6 | |
| `task_3216` | Optimize N+1 query in recommendation recording | FAST-FOLLOW | Audit 2 | |
| `task_8392` | Implement load testing for pre-go-live | ROADMAP | informs O-3 | |
| `task_ece2` | Migrate Vault to auto-unseal via KMS | ROADMAP | I-3 | pre-public-launch |
| `feature_e001` | Add JTI claim for JWT token revocation | ROADMAP | Audit 4 | |

---

## 6. API freeze checklist

These contract changes **must land before signups open** — they pin under shipped clients.
Aggregated from Audit 1 (F-series), Audit 2 (D-1), Audit 3 (M-1/M-2), and Audit 4 (S-2/S-3).

- [ ] **Spec ↔ route reconciliation** — content (`/api/v1/content/...`, `check-duplicate`) and
      fetcher (`/api/v1/source/rss/...`) OpenAPI paths match the routers (F-4, F-5). `task_065e`
- [ ] **Error envelope** — auth-401, panic-500, content-type-400 all emit the `pkg/api`
      `{error,message,details,meta}` shape; users/explore/fetcher specs document it (F-1, F-6). `task_065e`
- [ ] **Pagination shape** — request param + response `pagination` fields locked per endpoint and
      reflected in specs (cursor for unbounded lists, offset for bounded) (F-3, D-7). `task_065e`
- [ ] **List vs detail DTO** — list/search responses use a summary projection without
      `cleaned_html`; list and detail shapes frozen in the spec (D-1). `task_bfad`
- [ ] **New endpoint** `GET /api/v1/explore/user/vote-stats` → `{upvotes,downvotes}` specced (M-1). `task_12e0`
- [ ] **New endpoint** `GET /api/v1/explore/search?q=` specced (M-2). `task_0502`
- [ ] **New endpoints** `POST /api/v1/auth/verify-email`, `POST /api/v1/auth/forgot-password`,
      `POST /api/v1/auth/reset-password` specced; `email_verified` field on the user DTO (S-2, S-3). `task_2c1f`, `task_fe44`
- [ ] **Drift guard** — unit tests route through the production router so spec drift fails CI
      (defensive; ties to O-4 / `task_6f3a`).

Once every box above is checked and the non-contract beta-blockers (§4.1 #6, #9, #10, #11–14)
are done, the API surface can be declared frozen and public signups can open on a
single-instance deployment with a documented scaling ceiling.
