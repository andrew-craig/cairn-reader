# Pre-Beta Architecture Audit — Review Plan

**Status:** Planning
**Epic:** `epic_9c21` — Pre-beta architecture audit
**Owner:** TBD
**Goal:** Assess the Cairn architecture for efficiency, optimisation, consistency, scalability, and operational readiness *before* opening a public beta — while API and contract changes are still cheap to make.

---

## Why now

Once a public beta starts, client/server contracts are effectively frozen: mobile apps in the wild pin to the API shapes we ship, and changing them later means versioning, dual-support, and forced upgrades. This audit front-loads the expensive-to-change decisions (API contracts, data model, auth model) and separates them from work that can safely land *during* beta (observability, scaling, hardening).

The system under review:

| Area | Footprint |
|------|-----------|
| Read service (content + fetcher + email ingest) | ~34.5k LOC |
| Users service (auth) | ~14.7k LOC |
| Explore service (fetcher + recommender) | ~8.3k LOC |
| Mobile app (React Native / Expo) | ~7.8k LOC |
| Shared `pkg` (auth, middleware, rss, config) | ~5.9k LOC |

Backend is Go microservices over a single PostgreSQL instance (per-service logical DBs), JWT/RS256 auth with keys in HashiCorp Vault, fronted by Caddy.

---

## Approach

The audit is split into six independent **workstreams** plus a final **consolidation** step. Each workstream is a chalk task under `epic_9c21`; the consolidation task is blocked by all six and produces the single decision-ready deliverable.

```
epic_9c21  Pre-beta architecture audit
├── task_2a40  Audit 1: API contract consistency & stability      [P1 — most time-sensitive]
├── task_e695  Audit 2: Data layer & query efficiency             [P2]
├── task_1eaa  Audit 3: Mobile client/server efficiency           [P2]
├── task_659b  Audit 4: Security & auth hardening                 [P2]
├── task_ed30  Audit 5: Observability & operational readiness     [P2]
├── task_1d08  Audit 6: Infrastructure reliability & scaling      [P2]
└── task_b783  Audit 7: Consolidate → docs/architecture/PRE_BETA_AUDIT.md
                          (blocked by Audits 1–6)
```

Each workstream produces a **findings section** classifying every issue as **beta-blocking**, **fast-follow**, or **roadmap**. The consolidation step merges these into `docs/architecture/PRE_BETA_AUDIT.md` with a prioritised next-steps plan and an API-freeze checklist, and creates concrete remediation tasks (linking existing ones rather than duplicating).

### Priority rationale

Audit 1 (API contract) is **P1** because it gates the beta freeze — every other contract-affecting finding (e.g. a new vote-aggregate endpoint from Audit 3, a payload-shape change from Audit 2) must reconcile here before the contract is locked. The remaining workstreams are P2 and can proceed in parallel.

---

## Workstreams

### Audit 1 — API contract consistency & stability (`task_2a40`)
The contract-freeze workstream. Reconcile the OpenAPI specs, the implemented routes, and the mobile client. Known leads: inconsistent error envelopes across services (`{error}` vs `{error,message,details}` vs `{data,meta}`), mixed health-check formats, an apparent path mismatch between the mobile client (`/api/v1/content/user/{userId}`) and the spec (`/api/v1/users/{user_id}/contents`), and spec-vs-implementation drift. Output feeds the API-freeze checklist.

### Audit 2 — Data layer & query efficiency (`task_e695`)
Indexes, query shapes, payload sizes, pool config, and migration safety. Known leads: possibly missing indexes (`contents(source_feed_id)`, `feed_items(feed_id, processing_status)`), large list payloads carrying full article HTML, no per-query statement timeouts, and a pgx-vs-pq driver split.

### Audit 3 — Mobile client/server efficiency (`task_1eaa`)
Over-fetching, wasted calls, caching, retry, and offline behaviour. Highest-impact lead: the client fetches *all* votes (`limit=10000`) to count them client-side — needs a server aggregate endpoint (which loops back into Audit 1). Also: explore over-fetching, unpaginated search, no soft cache on screen-focus refetch.

### Audit 4 — Security & auth hardening (`task_659b`)
Public-facing security gaps on an otherwise solid base (bcrypt-12, RS256, token rotation, IDOR checks). Known leads: no account lockout / failed-login tracking (IP-only rate limiting), minimal email validation with no verification flow, no password reset, no auth audit logging.

### Audit 5 — Observability & operational readiness (`task_ed30`)
Can we *run* this in beta? Critical lead: no metrics endpoint / Prometheus / Grafana, so prod latency and error rates are unobservable. Also: no integration tests in CI (contract-drift risk), absent container resource limits, no log aggregation. Links existing tasks rather than duplicating (`task_644a`, `task_652a`, `task_8392`, `task_48ea`, `task_6f3a`, `feature_1a69`).

### Audit 6 — Infrastructure reliability & scaling (`task_1d08`)
Reliability and the single-instance → scaled path. Known leads: single Postgres (no replication/failover), backups documented but not automated, Vault file storage with plaintext unseal keys, in-memory state (rate limiter) that breaks under multiple instances, and a likely `max_connections` ceiling to verify. Links `task_5dcb`, `task_ece2`.

### Audit 7 — Consolidation (`task_b783`)
Produces `docs/architecture/PRE_BETA_AUDIT.md`: executive readiness verdict, per-workstream findings, a prioritised next-steps plan (beta-blocking / fast-follow / roadmap), an action→chalk-task mapping, and the **API freeze checklist**.

---

## Deliverable

`docs/architecture/PRE_BETA_AUDIT.md` — a single document with a clear go/no-go readiness verdict and a prioritised, task-linked list of proposed next steps. That document, not this plan, is the output of the review.

## How to run it

```bash
chalk show epic_9c21          # epic overview
chalk ready --parent=epic_9c21 # unblocked audit workstreams
chalk update task_2a40 --status=in_progress   # claim a workstream (start with Audit 1)
chalk close task_2a40                          # close as each section lands
```

Audit 7 stays blocked in `chalk ready` until all six workstreams are closed, then becomes available to write up the final document.
