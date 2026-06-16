# Audit 5 — Observability & Operational Readiness

**Workstream:** `task_ed30` (epic `epic_9c21`, P2)
**Status:** Findings complete
**Date:** 2026-06-16

> This is the operational readiness workstream. The question is not "does the system
> work" but "can you *operate* it safely once real users are sending traffic". Every
> finding below maps to a concrete operational risk. Two findings are **BETA-BLOCKING**
> (you cannot detect or bound failure in production without them); the rest are
> **FAST-FOLLOW** (real gaps, fixable without a contract freeze) or **ROADMAP**.

## How to read the classification

| Class | Meaning | Why |
|---|---|---|
| **BETA-BLOCKING** | Gap that leaves production unobservable or unbounded under load | Must land before signups open |
| **FAST-FOLLOW** | Real operational gap, but changeable mid-beta without affecting clients | Fix during early beta |
| **ROADMAP** | Correct at beta scale; document so it isn't rediscovered as a surprise | Track, no action now |

## Summary table

| ID | Finding | Class | Existing task |
|---|---|---|---|
| O-1 | No metrics endpoint, no Prometheus, no Grafana — cannot observe latency/errors/saturation | **BETA-BLOCKING** | `task_644a` + `task_652a` |
| O-2 | No distributed tracing; X-Request-ID propagates within a service but not across services | FAST-FOLLOW | NEW |
| O-3 | No container resource limits in prod or selfhost compose | FAST-FOLLOW | `task_48ea` |
| O-4 | Integration tests exist and are tagged but CI never runs them — breaking inter-service changes merge undetected | FAST-FOLLOW | `task_6f3a` |
| O-5 | Users service has no HTTP ReadTimeout or WriteTimeout — slow client attack can exhaust goroutines | **BETA-BLOCKING** | NEW |
| O-6 | Log format defaults to `text` in prod; LOG_FORMAT not set in prod compose; no log shipping stack | FAST-FOLLOW | `feature_1a69` |

---

## O-1 — No metrics, no dashboards, no alerting (BETA-BLOCKING)

**Correction to the original lead.** The lead is confirmed, not stale.

There is zero metrics instrumentation anywhere in the repository. A thorough search across all Go source files, compose files, and workflow configs finds no reference to `prometheus`, `promhttp`, `/metrics`, or Grafana:

```
grep -r "metrics|prometheus|promhttp|grafana" /home/user/cairn-reader --include="*.go" → 0 matches
grep -r "metrics|prometheus|promhttp|grafana" /home/user/cairn-reader --include="*.yml" → 0 matches
```

The only observability surface is structured `slog` JSON logs and the health probes (`/health/live`, `/health/ready`). Health probes answer "is the process up?" — they say nothing about whether it is *performing*.

**Services and workers without any metrics:**

| Binary | Owns |
|---|---|
| `services/users/cmd/user-service/main.go` | Auth/token latency, login failure rate |
| `services/read/content/cmd/content/main.go` | Article ingest rate, HTML fetch latency |
| `services/read/fetcher/cmd/ingest_rss/main.go` | RSS subscription throughput |
| `services/read/email/cmd/email_ingest/main.go` | Email ingest queue depth |
| `services/explore/recommender/cmd/explore_recommender/main.go` | Recommendation latency, vote throughput |
| `services/explore/fetcher/cmd/explore_fetcher/main.go` | Feed fetch success/error rate |
| Workers: `content/cmd/worker`, `fetcher/cmd/ingest_rss_worker`, `email/cmd/email_ingest_worker` | Job duration, queue depth, failure counts |

**Why beta-blocking.** Without metrics you cannot answer any operational question the moment live traffic starts:
- Is the content service getting slower? (no latency histogram)
- Is the RSS worker falling behind on feeds? (no queue depth gauge)
- Is the users service throwing 5xx to real users? (logs exist but no rate counter or alert)
- Are Postgres connection pools at capacity? (no `pgxpool.Stat` gauge)

**Minimal beta bar (covers `task_644a` + `task_652a`):**
1. Add `github.com/prometheus/client_golang` to each service module.
2. Wrap the chi router with `promhttp`-based middleware recording: HTTP request count by method/path/status, HTTP request duration histogram (p50/p95/p99), active in-flight requests.
3. Expose `/metrics` on a separate internal port (or the same port with IP-restriction at Caddy).
4. Add a `prometheus` + `grafana` service to the prod compose (or use a hosted Grafana Cloud free tier for beta).
5. Wire four critical alerts (`task_652a`): error-rate > 5% over 5 min, p99 latency > 2s, any service health/ready failing, Postgres pool in-use > 80%.
6. Add per-DB pool stats (call `pool.Stat()` in a background goroutine and export as gauges — one iteration per service).

**Worker-specific:** worker job duration (`Histogram`) and failure count (`Counter`) per job type. The cron workers (`cleanup_job`, `outbox_cleanup_job`, `feed_items_cleanup_job`) are currently fire-and-forget with only log output.

**Effort:** 2–3 days of instrumenting + 1 day for compose + dashboards. The prometheus client middleware is ~30 lines; the bulk is the dashboard JSON.

**Existing tasks:** `task_644a` (Add Prometheus metrics and Grafana dashboards) and `task_652a` (Set up alerting system for pre-go-live) cover this finding exactly. Both are blocked on `decision_4052`; that decision should be unblocked immediately since this is beta-blocking.

---

## O-2 — No cross-service request-trace propagation (FAST-FOLLOW)

X-Request-ID is generated or forwarded *within* a single service at the HTTP boundary (`pkg/logging/chi_middleware.go:20-24`):

```go
// pkg/logging/chi_middleware.go:20-24
requestID := r.Header.Get("X-Request-ID")
if requestID == "" {
    requestID = uuid.New().String()
}
w.Header().Set("X-Request-ID", requestID)
```

The ID is stored in context (`ctx = WithRequestID(ctx, requestID)`) and logged with each request (`chi_middleware.go:57`). This is good for single-service log correlation.

**The gap:** When the content service calls the ingest-rss service (via `services/read/content/internal/service/ingest_rss_client.go`), or when the ingest-rss-worker calls the content service, or when explore-fetcher POSTs to explore-recommender — none of those outbound HTTP clients forward the incoming `X-Request-ID` header. Each hop generates a fresh ID. A single user action that touches three services produces three unrelated request IDs with no linkage.

Evidence (absence): searching the HTTP client construction in each inter-service client shows no `X-Request-ID` header injection. For example, `ingest_rss_client.go` builds `http.NewRequest` without copying the incoming request ID from context.

**Why it matters.** For beta, tracing a user-reported failure through logs requires manual grep across multiple services with timestamp correlation — fragile and slow. Full distributed tracing (OpenTelemetry + Jaeger) is a roadmap item. But propagating X-Request-ID costs almost nothing and is the minimum viable cross-service correlation.

**Assessment of "enough for beta":** X-Request-ID-only correlation is *adequate* for beta if it actually propagates. As of now it does not propagate at all, making cross-service debugging materially harder than it needs to be. This is FAST-FOLLOW, not BETA-BLOCKING, because a determined operator can still reconstruct a trace via timestamps — but it adds significant operational friction.

**Recommendation (FAST-FOLLOW):**
1. Add a shared outbound middleware or helper in `pkg/logging` that injects `X-Request-ID` from context into outbound `http.Request` headers.
2. Apply it in every inter-service HTTP client (`ingest_rss_client.go`, explore-fetcher's recommender client, email-worker's content client).
3. Keep the existing per-service generation as the origin for inbound requests that arrive without an ID.

**Effort:** 1 day. No schema changes, no new dependencies.

**Existing task:** None. **NEW task needed** — link to `task_644a` as a prerequisite or complement.

---

## O-3 — No container resource limits in prod or selfhost compose (FAST-FOLLOW)

**Confirmed.** Neither compose file contains any `deploy.resources`, `mem_limit`, `cpus`, or `memswap_limit` directive. The two matches from a grep search are both comments, not configuration:

- `infrastructure/docker/prod/docker-compose.yml:3` → comment: `"# This file demonstrates how to deploy using pre-built images…"`
- `infrastructure/docker/selfhost/docker-compose.yml:51` → comment: `"# Tuned for small self-hosted deployments…"`

The prod compose runs 11 containers (caddy, vault, vault-init, vault-unseal, cairn-db, user-service, explore-recommender, explore-fetcher, content-service, content-worker, ingest-rss, ingest-rss-worker, email-ingest, email-ingest-worker, web) on a single host with no CPU or memory constraints.

**Risk:** a RSS fetch spike or a full-text search surge that consumes unbounded memory on one container can trigger the host OOM-killer on an unrelated container (e.g., the Postgres container). This is the "noisy-neighbour OOM" scenario. At beta scale (hundreds of users) it is a real risk, not theoretical.

**The selfhost compose** also has no limits, though it does tune Postgres parameters (`shared_buffers`, `work_mem`, etc.) via command args — so the DB is bounded but the application containers are not.

**Recommendation (FAST-FOLLOW, `task_48ea`):**
Add `deploy.resources.limits` to each service block. Suggested starting values for a 2–4 GB host:

| Service | Memory limit | CPU limit |
|---|---|---|
| cairn-db | 512m | 1.0 |
| user-service | 256m | 0.5 |
| content-service | 512m | 1.0 |
| ingest-rss / ingest-rss-worker | 256m each | 0.5 each |
| email-ingest / email-ingest-worker | 256m each | 0.5 each |
| explore-recommender | 256m | 0.5 |
| explore-fetcher | 256m | 0.5 |
| vault | 256m | 0.5 |
| caddy | 128m | 0.25 |

Actual values should be validated against `task_8392` load-test results. Set `mem_reservation` (soft limit) below `memory` (hard limit) so Docker can reclaim headroom between bursts.

**Effort:** 1–2 hours to add limits; 1 day to validate against load-test baselines (`task_8392`).

**Existing task:** `task_48ea` (Add Docker resource constraints to production compose) covers this exactly.

---

## O-4 — Integration tests exist but CI never runs them (FAST-FOLLOW)

**Confirmed.** Integration tests are present and properly build-tagged across services:

| Service | Integration test location | Build tag |
|---|---|---|
| users | `services/users/test/integration/` | (no build tag — requires Docker Compose env via `docker-compose.test.yml`) |
| explore/recommender | `services/explore/recommender/integration_test.go` | `//go:build integration` |
| explore/recommender | `services/explore/recommender/internal/db/article_repository_integration_test.go` | `//go:build integration` |
| explore/fetcher | `services/explore/fetcher/integration_test.go` | `//go:build integration` |
| read/content | `services/read/content/integration_test.go` | `//go:build integration` |
| read/fetcher | `services/read/fetcher/integration_test.go` | `//go:build integration` |

The `.github/workflows/go-checks.yml` runs `go test ./...` **without** `-tags integration`, so every tagged test is silently skipped on every PR. Searching the workflow YAML for `-tags`, `integration`, or service container definitions (`services: postgres:`) returns zero matches.

The selfhost compose smoke test (`docker-test.yml:326-371`) does spin up the full stack and hit `/health/ready` and `/health/live`, which is valuable — but it only tests the selfhost consolidated binary, not the individual service contracts.

**Why it matters (ties to Audit 1 contract drift).** Inter-service integration tests are the primary guard against schema drift and broken client-service contracts. Audit 1 found route path mismatches between OpenAPI specs and implementations (F-4, F-5); those were undetected because unit tests build their own chi contexts and do not exercise the real router with a real database. The integration tests that *would* catch this are never run.

**Recommendation (FAST-FOLLOW, `task_6f3a`):**
Add a separate GitHub Actions workflow (or a job in `go-checks.yml` gated on a `run-integration` label or a push-to-main trigger) that:
1. Uses GitHub Actions service containers for Postgres (`services: postgres: image: postgres:18`).
2. Runs `go test -tags integration -count=1 -timeout 10m ./...` for each service.
3. For users service: either port the Docker Compose-based tests to use service containers, or run them via `docker compose -f services/users/test/integration/docker-compose.test.yml up --abort-on-container-exit`.

The simplest unblocking step (1 day) is a single job using service containers for explore and read integration tests, which have the `//go:build integration` tag and do not need Vault. Users integration tests (which need Vault) can follow in a second step.

**Effort:** 1–2 days for CI wiring; the tests themselves already exist.

**Existing task:** `task_6f3a` (Add integration test workflow with Docker/Postgres) covers this exactly.

---

## O-5 — Users service has no HTTP ReadTimeout or WriteTimeout (BETA-BLOCKING)

**Confirmed.** Every other service sets HTTP timeouts explicitly; the users service does not:

| Service | ReadTimeout | WriteTimeout | IdleTimeout | Evidence |
|---|---|---|---|---|
| **user-service** | **MISSING** | **MISSING** | **MISSING** | `services/users/cmd/user-service/main.go:198-201` |
| content-service | 30s | 30s | 120s | `services/read/content/cmd/content/main.go:134-136` |
| ingest-rss | 30s | 30s | 120s | `services/read/fetcher/cmd/ingest_rss/main.go:94-96` |
| email-ingest | 30s | 30s | 120s | `services/read/email/cmd/email_ingest/main.go:150-152` |
| explore-recommender | 15s | 15s | 60s | `services/explore/recommender/cmd/explore_recommender/main.go:137-139` |
| explore-fetcher | 10s | 10s | — | `services/explore/fetcher/cmd/explore_fetcher/main.go:175-176` |
| selfhost (cmd/selfhost) | 30s | 30s | 120s | `cmd/selfhost/main.go:166-168` |

The user-service HTTP server is constructed as:

```go
// services/users/cmd/user-service/main.go:198-201
srv := &http.Server{
    Addr:    ":" + cfg.Server.Port,
    Handler: router,
}
```

No `ReadTimeout`, `WriteTimeout`, or `IdleTimeout` is set, so Go's `net/http` defaults apply — which means these timeouts are **infinite**. A client that opens a connection and trickles bytes (slow-loris attack) or a mobile client on a poor network that holds a connection open indefinitely will tie up a goroutine indefinitely. The users service is the authentication gateway: every login, register, and token-refresh flows through it. Exhausting its goroutines blocks all other services from authenticating users.

Graceful shutdown *is* implemented correctly in the users service (`srv.Shutdown` with a 30s `ShutdownTimeout` sourced from `services/users/internal/config/config.go:82` via `env.GetDuration("SHUTDOWN_TIMEOUT", 30*time.Second)`) — the missing piece is the *per-connection* timeout guards.

**Why beta-blocking.** Missing `ReadTimeout` is a well-known Go HTTP server misconfiguration that enables slowloris-style resource exhaustion. The users service is the sole authentication gateway; bringing it down blocks every other authenticated endpoint. This is an abuse guard that is a one-line fix; there is no reason to defer it.

**Recommendation (BETA-BLOCKING):**
Add timeouts matching the existing pattern in other services:

```go
// services/users/cmd/user-service/main.go (replace lines 198-201)
srv := &http.Server{
    Addr:         ":" + cfg.Server.Port,
    Handler:      router,
    ReadTimeout:  30 * time.Second,
    WriteTimeout: 30 * time.Second,
    IdleTimeout:  120 * time.Second,
}
```

`ReadHeaderTimeout` (5–10s) is also worth adding as defence-in-depth against header-only slowloris attacks.

**Effort:** 5 minutes for the fix; add a test or a comment documenting why the timeouts are set to prevent future removal.

**Existing task:** None. **NEW task needed.** This is a trivial fix; create as a direct child of the beta-readiness epic, not a standalone task.

---

## O-6 — Log format defaults to text in prod; no log shipping stack (FAST-FOLLOW)

**Two distinct sub-findings:**

### O-6a — Text format is the prod default

`pkg/config/config.go:204` sets `LOG_FORMAT` default to `"text"`:

```go
// pkg/config/config.go:204
Format: GetString("LOG_FORMAT", "text"),
```

None of the service blocks in `infrastructure/docker/prod/docker-compose.yml` set `LOG_FORMAT=json`. Only `ENVIRONMENT=production` is set (for user-service and email-ingest; the other services do not even have that). This means all services in prod are emitting `slog` text-format output, not JSON — which is machine-parseable but not structured in a way that any standard log aggregation system (Loki, CloudWatch, Datadog) can parse field-by-field out of the box.

**Fix (trivial, FAST-FOLLOW):** add `LOG_FORMAT=json` and `LOG_LEVEL=info` to every service block in the prod compose. Also set `ENVIRONMENT=production` on the services that are missing it (explore-recommender, explore-fetcher, content-service, ingest-rss, ingest-rss-worker, content-worker, email-ingest-worker). This is consistent with `pkg/logging/logger.go`'s `json` branch (`logger.go:37-38`).

### O-6b — No log aggregation stack

Services emit structured slog output to stdout. In the prod compose, `docker compose logs` is the only consumption path — no Loki, no Promtail, no CloudWatch agent, no Datadog agent is configured. With multiple containers across a single host, there is no way to query logs across services, set log-based alerts, or retain logs beyond the Docker daemon's buffer.

**Why FAST-FOLLOW (not BETA-BLOCKING):** logs are accessible via `docker compose logs` in an emergency; a beta with a small number of users can be monitored manually for the first weeks. The risk is that incidents take longer to diagnose. This becomes BETA-BLOCKING if metrics (`task_644a`) are also absent — you'd have no observability at all. With metrics in place, logs are a secondary debugging tool.

**Recommendation (FAST-FOLLOW, `feature_1a69`):**
- Minimum viable for beta: add Loki + Promtail sidecar to the prod compose. Promtail scrapes Docker log files from `/var/lib/docker/containers/`; Loki provides query via Grafana (same Grafana added for O-1). This reuses the Grafana instance from `task_644a` and adds ~2 compose services.
- Set `LOG_FORMAT=json` and `LOG_LEVEL=info` on all service blocks immediately (5-minute change, no restart of the DB or Vault needed).

**Effort:** 30 minutes for the env-var change; 1 day for Loki/Promtail setup.

**Existing task:** `feature_1a69` (Add log monitoring/aggregation stack for production) covers O-6b. O-6a (missing LOG_FORMAT) is a gap within `feature_1a69` — include it in that task's scope.

---

## Observability checklist for Audit 7

Before signups open, the following must be true:

- [ ] **O-1** At minimum: HTTP request count + duration metrics exported on `/metrics`; Prometheus scraping all services; one Grafana dashboard per service (error rate, p99 latency, active connections). (`task_644a`)
- [ ] **O-1** At minimum: four critical alerts configured: error rate, latency, health probe, pool utilisation. (`task_652a`)
- [ ] **O-5** `services/users/cmd/user-service/main.go` `http.Server` has `ReadTimeout`, `WriteTimeout`, `IdleTimeout` set. (NEW task)

Fast-follow during beta:

- [ ] **O-2** Outbound HTTP clients forward `X-Request-ID` from context across service boundaries. (NEW task)
- [ ] **O-3** `deploy.resources.limits` added to all service blocks in prod and selfhost compose. (`task_48ea`)
- [ ] **O-4** GitHub Actions workflow runs integration tests (`-tags integration`) against a Postgres service container on PRs to main or on merge. (`task_6f3a`)
- [ ] **O-6a** `LOG_FORMAT=json` and `ENVIRONMENT=production` set on all service blocks in prod compose.
- [ ] **O-6b** Loki + Promtail added to prod compose; logs queryable via Grafana. (`feature_1a69`)

No action now (documented): Load testing (`task_8392`) should inform resource limits (`task_48ea`) — sequence them in that order.

---

## Existing tasks this workstream maps to

| Chalk task | Title | Finding(s) | Status |
|---|---|---|---|
| `task_644a` | Add Prometheus metrics and Grafana dashboards | O-1 | open (blocked on `decision_4052`) |
| `task_652a` | Set up alerting system for pre-go-live | O-1 | open (blocked on `decision_4052`) |
| `task_8392` | Implement load testing for pre-go-live | informs O-3 limits | open (blocked on `decision_4052`) |
| `task_48ea` | Add Docker resource constraints to production compose | O-3 | open (blocked on `decision_4052`) |
| `task_6f3a` | Add integration test workflow with Docker/Postgres | O-4 | open (blocked on `decision_4052`) |
| `feature_1a69` | Add log monitoring/aggregation stack for production | O-6b | open (blocked on `decision_4052`) |
| **NEW** | Fix HTTP timeouts on user-service | O-5 | not yet created |
| **NEW** | Propagate X-Request-ID across service boundaries | O-2 | not yet created |
