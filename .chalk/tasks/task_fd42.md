---
id: task_fd42
title: [Readiness] Self-host /health/ready checks only 3 of 6 DBs → healthy while half the system is down
type: task
status: open
priority: 2
labels: [quality,wave3,ops]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:53:56Z
updated_at: 2026-08-09T06:53:56Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** readiness probe lies (Part 1 shared-#5, confirmed from the deploy side in Part 2) | **Wave 3** | **Recipe:** R6 (strategy §2.5)
**Touches:** cmd/selfhost health wiring, .github/workflows/docker-test.yml

## Problem
`addDB` is called for only **3 of the 6** self-host databases. Outages of users, explore-recommender or explore-fetcher still report `healthy`. Worse, the CI smoke test in `docker-test.yml` curls that same endpoint — so CI reports a **false green** while half the system is down.

## What to do
1. Test first: take one of the three unchecked DBs offline and assert `/health/ready` reports unhealthy. Fails on main.
2. Register all 6 databases with `addDB`.
3. Confirm the compose smoke test in `docker-test.yml` actually fails when a DB is down — otherwise the ratchet is still fake.

## Done when
- `/health/ready` reflects all 6 databases and the CI smoke test detects a downed one.

---

## Re-confirmed by the Cairn Simplification Audit (2026-08-17)

Independently re-verified at HEAD `a6c56a1` and listed under the audit's Tier 1 (correctness & security). No new task was created — this one owns the finding.
**Audit report:** https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f

**One addition to step 3.** The audit found *why* the CI ratchet is fake, and it is worse than "the smoke test doesn't assert enough": the smoke job **never runs at all**. `.github/workflows/docker-test.yml:326-329` declares `needs: [build-selfhost]` but gates on `if: needs.changes.outputs.selfhost == 'true'` — `changes` is absent from `needs:`, so that expression resolves to empty and the condition can never be true.

So step 3 ("confirm the compose smoke test actually fails when a DB is down") cannot be satisfied until that job is fixed. That fix is tracked separately as **task_7722**. Sequence task_7722 first, or this task's ratchet remains unverifiable.

---

## Review (2026-09-20, branch `task_fd42`)

### Re-verification on current main (HEAD `e285467`)
Confirmed the finding is still live. `cmd/selfhost/health.go`'s `addDB` was called from exactly 3 of 6 `adapt_*.go` files:
- **Wired:** `adapt_content.go` ("content"), `adapt_email_ingest.go` ("email"), `adapt_ingest_rss.go` ("rss") — all three use `database/sql` (`*sql.DB`) under the hood.
- **Missing:** `adapt_users.go`, `adapt_explore_recommender.go`, `adapt_explore_fetcher.go` — exactly the three named in the finding (users, explore-recommender, explore-fetcher). All three are built on `*pgxpool.Pool` (pgx), not `database/sql`, which is *why* they were never wired the same way `addDB` was — `addDB`'s signature only accepted `*sql.DB`.

### task_7722 discrepancy
Independently confirmed what the task description reported: `chalk show task_7722` returns `status: closed`, but `.github/workflows/docker-test.yml` at current HEAD still had the bug at the `selfhost-compose-smoke` job — `needs: [build-selfhost]` with `if: needs.changes.outputs.selfhost == 'true'`, `changes` absent from `needs:`. `git log --all --grep="task_7722"` and `git log -- .github/workflows/docker-test.yml` show no commit ever touched this. task_7722 was closed without the fix landing. **Flagging for the reviewer to decide whether to reopen task_7722** (or leave it closed and note this task absorbed the fix — see below).

Applied the one-line task_7722 fix as a hard prerequisite in this same branch/commit (scoped to exactly the line the task described, no more): `needs: [build-selfhost]` → `needs: [changes, build-selfhost]`, matching the `needs: [changes]` pattern every other job in the file uses.

### Fix (this task's actual scope)
1. `cmd/selfhost/health.go`: `healthChecker.checks` changed from `map[string]*sql.DB` to `map[string]pinger`, where `pinger` is a 1-method interface (`Ping(ctx) error`). Added `sqlPinger` to adapt `*sql.DB.PingContext` to the interface (so `addDB` keeps its exact old signature/behavior for the 3 already-wired services) and a new `addPinger(name string, p pinger)` for anything that already implements `Ping(ctx) error` — `*pgxpool.Pool` does, natively, so no adapter needed there.
2. `services/users/selfhost/users.go` (`MountUsers`), `services/explore/recommender/selfhost/recommender.go` (`Mount`), `services/explore/fetcher/selfhost/fetcher.go` (`Mount`): each now returns its `*pgxpool.Pool` alongside the existing `(func(), error)`, mirroring the pattern `contentSelfhost.Mount` / `emailSelfhost.MountEmail` / `rssSelfhost.Mount` already used for `*sql.DB`.
3. `cmd/selfhost/adapt_users.go`, `adapt_explore_recommender.go`, `adapt_explore_fetcher.go`: each now captures the pool and calls `health.addPinger("users"|"explore-recommender"|"explore-fetcher", pool)` before returning the closer — same shape as the existing `health.addDB(...)` call sites in the other three adapters.
4. `services/users/selfhost/users_test.go`: updated the one existing call site of `MountUsers` for the new 3-value return.
5. `.github/workflows/docker-test.yml`: task_7722's one-line `needs:` fix (above).

### Test written, and its before/after result
New file: `cmd/selfhost/health_readiness_integration_test.go` (`//go:build integration`, matching the existing `test-integration-selfhost` CI job and the pattern in `router_internal_routes_test.go`). `TestReadiness_DetectsDownedUsersDB`:
- Creates a real ephemeral test DB (`createTestDatabase` helper, already in the package), runs users migrations, mounts the real user service via `mountUserService` (the exact wrapper `main.go` calls).
- Sanity-checks `/health/ready` is `200 healthy` while the DB is actually up.
- Simulates the users DB going down *after* a clean start (closes the pool via the returned closer — chosen over pointing at a dead host up front because `userDB.New` pings eagerly at connect time, so an unreachable host fails `Mount` itself rather than reaching the readiness probe; closing the live connection reproduces the actual "was up, went down" scenario a readiness probe exists to catch).
- Asserts `/health/ready` now reports `503 unhealthy`.

Verified against a disposable local Postgres container (`postgres:16-alpine`, port 15432, unrelated to any other Postgres on the host):
- **Before the fix** (test written against unmodified `main`): `FAIL` — `status = 200, want 503`, body `{"checks":{},"status":"healthy"}`. Confirms the users DB was never in `health.checks` at all, so closing its connection had zero effect on the readiness report — exactly the lie the finding describes.
- **After the fix**: `PASS`.

Also ran the full existing suite after the fix:
- `cmd/selfhost`: `go build ./...`, `go vet ./...`, `go test ./...` (unit) all clean; `go test -tags=integration ./...` (full integration suite, real Postgres) all clean.
- `services/users`: `go build ./...`, `go vet ./...`, `go test ./internal/...` clean. (`services/users/selfhost`'s pre-existing `TestMountUsers_VerifyEmailDoesNotPanic` was independently confirmed to skip identically on unmodified `main` with the same disposable Postgres — `unable to create connection pool: MaxSize must be >= 1` — a pre-existing test-harness issue unrelated to this change; not touched.)
- `services/explore`: `go build ./...`, `go vet ./...`, `go test ./...` clean.
- `gofmt -l` clean on every changed package.
- Note: `-race` isn't usable in this sandbox (`ThreadSanitizer: unsupported VMA range` — a kernel/arch limitation of the sandbox, not the code); all the above were also run without `-race` for that reason.

### CI smoke-test verification — what I could and couldn't confirm
Could not actually execute the GitHub Actions workflow (no `act` or equivalent available, and it would need a real image push/pull round-trip). What I did verify:
- YAML parses (`python3 -c "import yaml; yaml.safe_load(...)"`) — the `needs:` edit didn't break the file.
- Confirmed the fixed `needs: [changes, build-selfhost]` now matches the shape every other conditional job in the file already uses (`needs: [changes]`), and that referencing `needs.changes.outputs.selfhost` in `if:` is now valid (previously `changes` wasn't a listed dependency of the job at all, so the expression could never be true and the job silently never ran).
- Confirmed via `cmd/selfhost` integration test (above) that with all 6 DBs now registered, `/health/ready` correctly flips to `503`/`unhealthy` the moment any one of them (tested: users) goes down after a clean start — this is the exact mechanism the smoke test's "Wait for readiness" loop depends on, since it polls with `curl -fsS` (which only succeeds on 2xx, so a `503` keeps the loop retrying until it times out and fails the step).

**New discrepancy found, flagging but NOT fixed (out of this task's scope per explicit instruction):** even after the task_7722 `needs:` fix lets `selfhost-compose-smoke` actually run, the job's "Wait for readiness" / "Check liveness" steps curl `http://localhost:8080/health/ready` and `.../health/live` (lines ~347, 349, 360), but the compose stack in `infrastructure/docker/selfhost/docker-compose.yml` publishes the `cairn` container on `${PORT:-8099}:8099` (confirmed: `.env.example` has `PORT=8099` commented-out as the default, and the CI step's `.env` setup only overrides `DB_PASSWORD`, so the default 8099 applies; the Dockerfile also `EXPOSE`s 8099). So once task_7722's fix lands, this job will fail immediately with connection-refused on port 8080 regardless of DB health — it would need its own fix (curl port 8080 → 8099) to actually become a working ratchet. Recommend filing this as a new task; did not fix it here since I was told to scope task_7722's part to exactly the one `needs:` line.

### Files changed
- `cmd/selfhost/health.go`
- `cmd/selfhost/adapt_users.go`
- `cmd/selfhost/adapt_explore_recommender.go`
- `cmd/selfhost/adapt_explore_fetcher.go`
- `cmd/selfhost/health_readiness_integration_test.go` (new)
- `services/users/selfhost/users.go`
- `services/users/selfhost/users_test.go`
- `services/explore/recommender/selfhost/recommender.go`
- `services/explore/fetcher/selfhost/fetcher.go`
- `.github/workflows/docker-test.yml`

Status field left untouched — leaving `open`/close decision to the reviewer, per instructions.

