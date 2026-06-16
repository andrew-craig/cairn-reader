# Audit 6 — Infrastructure Reliability & Scaling

**Workstream:** `task_1d08` (epic `epic_9c21`, P2)
**Status:** Findings complete
**Date:** 2026-06-16

> This is the infrastructure reliability workstream. The question is not "does it work
> on a single dev machine" (it does), but "what breaks first when beta users hit a
> production single-server deployment, and what must be decided or fixed before signups open".
> Every finding below was verified against real config files and code — stale task leads are
> corrected.

## How to read the classification

| Class | Meaning | Why |
|---|---|---|
| **BETA-BLOCKING** | Data loss risk, silent breakage, or safety guard that is cheap now and expensive later | Must reconcile before signups open |
| **FAST-FOLLOW** | Real operational risk; acceptable for a private beta cohort if acknowledged, must land before public scale | Fix during beta |
| **ROADMAP** | Correct at beta scale; documented so it isn't rediscovered as a surprise | Track, no action now |

## Summary table

| ID | Finding | Class | Decision needed |
|---|---|---|---|
| I-1 | Single PostgreSQL instance — no replication, no failover | **BETA-BLOCKING** | Managed DB or documented single-instance + tested restore |
| I-2 | Backups not automated — script in docs only, no cron container, no offsite store | **BETA-BLOCKING** | Execute `task_5dcb`: ship the script + cron container before beta |
| I-3 | Vault uses file storage (not HA) + unseal keys stored plaintext in `vault-keys` volume | **BETA-BLOCKING** | Accept risk + volume backup plan, or execute `task_ece2` (KMS auto-unseal) |
| I-4 | In-memory rate limiter breaks under multiple instances; currently single-instance by design | FAST-FOLLOW | Document single-instance as the beta topology; defer Redis-backed limiter |
| I-5 | `sslmode=disable` on all internal DB connections within Docker network | FAST-FOLLOW | Accept for beta (private Docker network); document the decision |
| I-6 | Connection ceiling: 9 containers × up to 25 conns each = **up to 175 connections** vs Postgres default max_connections **100** (unset in prod compose) | **BETA-BLOCKING** | Set `max_connections` ≥ 200 in prod postgres command args, or reduce pool caps |
| I-7 | Email ingest: `X-API-Key` validated by SHA-256 hash in DB; no key rotation tooling, no per-key abuse limit | FAST-FOLLOW | Rotation runbook + consider per-key rate limit |

---

## I-1 — Single PostgreSQL instance: no replication or failover (BETA-BLOCKING)

**Current state.** Production is a single `postgres:18-alpine` container
(`infrastructure/docker/prod/docker-compose.yml:139`) backed by a named Docker volume
`cairn_db_data` on the host filesystem. There is no streaming replication, no read replica,
no failover automation, and no managed-database alternative configured. If the container
crashes, Docker restarts it (restart: `unless-stopped`), but a corrupted volume, filesystem
failure, or accidental `docker compose down -v` destroys all user data.

**Evidence:**
- `infrastructure/docker/prod/docker-compose.yml:139–176` — single `cairn-db` service with one volume.
- `docs/DEPLOYMENT.md:333–340` — production docs recommend a single-VPS deployment.
- `docs/DEPLOYMENT.md:909` — mentions "Consider managed PostgreSQL (AWS RDS, etc.)" as a bullet in production considerations, but no action is taken.

**Decision needed.** Two viable options for beta:

1. **Accept the SPOF** (single-instance) with compensating controls: automated daily backup (`task_5dcb`, see I-2), a tested restore runbook, and operator alerting so downtime is detected within minutes. Appropriate for a private beta cohort where SLA expectations are low.
2. **Migrate to a managed DB** (AWS RDS Postgres, Cloud SQL, Neon, etc.) before signups. Eliminates the hardware SPOF and gives point-in-time recovery, but adds cost and configuration effort.

**Recommendation.** For beta: accept the SPOF + implement `task_5dcb` immediately + document and rehearse the restore procedure (time-to-restore target). Capture "migrate to managed DB" as a post-beta roadmap item in `task_5dcb`'s parent epic. The single-VPS architecture is appropriate for a private beta, not for a public launch at scale.

**Effort.** Restore runbook + `task_5dcb`: 1–2 days. Managed DB migration: 2–4 days.

**Existing task.** `task_5dcb` covers the backup side. **No new task needed** — add "test restore procedure" and "single-instance risk accepted for beta" decisions to `task_5dcb`.

---

## I-2 — Backups not automated: script in docs only, no cron container, no offsite store (BETA-BLOCKING)

**Current state.** `docs/DEPLOYMENT.md:657–695` documents a `scripts/backup.sh` and a
host-level cron schedule. That script **does not exist** as an actual file in the repository —
`task_5dcb` records this gap. No Docker service runs as a backup cron. There is no S3 or
offsite upload step in the documented script. The production compose file has no backup container.

**Evidence:**
- `infrastructure/docker/prod/docker-compose.yml` — no `backup` or `pgdump` service.
- `infrastructure/docker/scripts/` — contains `init-databases.sh`, `init-vault.sh`, `init-vault-prod.sh`, `unseal-vault.sh`, but **no** `backup.sh`.
- `docs/DEPLOYMENT.md:659` — "Create `scripts/backup.sh`:" (present tense directive, file absent).
- `task_5dcb` — status `open`, blocked by `decision_4052`.

**Why beta-blocking.** Any data ingested during beta is user data. Without automated backups,
a volume loss event results in complete, unrecoverable user data loss. A manual cron on the
host is operationally fragile (needs the cairn user's crontab to survive host re-provisioning).
Offsite storage (e.g. S3) is mandatory for data durability at any level of user trust.

**Recommendation.**
1. Unblock `task_5dcb` from `decision_4052` (the decision is: implement for beta, accept single-instance).
2. Create `infrastructure/docker/scripts/backup.sh` that runs `pg_dumpall` piped to gzip, uploads to S3 (or compatible), and purges local copies older than 7 days.
3. Add a lightweight `backup` container to `prod/docker-compose.yml` using `docker-library/docker` or a minimal cron image, mounting the Docker socket or using `pg_dump` directly against the `cairn-db` service. Schedule daily at 02:30 (after the content cleanup cron at 02:00).
4. Add a restore runbook to `docs/DEPLOYMENT.md` and test it once before beta opens.

**Effort.** 1–2 days.

**Existing task.** `task_5dcb` covers this directly. **No new task needed.**

---

## I-3 — Vault: file storage backend (not HA) + unseal keys stored plaintext in volume (BETA-BLOCKING)

**Current state.** Vault is configured with `storage "file"` at `/vault/data`
(`infrastructure/docker/vault-config/vault.hcl:1–3`). File storage is a single-node, non-HA
backend: if the `vault-data` Docker volume is corrupted or lost, all secrets (JWT RSA keys,
AppRole credentials) are permanently lost and every service loses its authentication
credentials. There is no seal configuration in `vault.hcl` — Vault uses Shamir unsealing.

The unseal watchdog (`infrastructure/docker/scripts/unseal-vault.sh`) reads 3 of 5 unseal
keys from `/vault-keys/UNSEAL_KEYS.txt` — a plaintext file in the `vault-keys` Docker volume.
The `init-vault-prod.sh` script explicitly writes all 5 keys to that file
(`infrastructure/docker/scripts/init-vault-prod.sh:69–84`) and includes a warning that the
operator "should retrieve and delete this file" — but the watchdog *requires* the file to be
present to unseal on every restart, so deletion is impossible in practice. The keys live in
plaintext on the Docker host indefinitely.

**Evidence:**
- `infrastructure/docker/vault-config/vault.hcl:1–3` — `storage "file"`.
- `infrastructure/docker/scripts/unseal-vault.sh:13,59–71` — reads keys from `UNSEAL_KEYS.txt`.
- `infrastructure/docker/scripts/init-vault-prod.sh:69–84` — writes all 5 keys + warning comment.
- `infrastructure/docker/prod/docker-compose.yml:117–135` — `vault-unseal` mounts `vault-keys:ro`.
- `task_ece2` — "Migrate Vault to auto-unseal via KMS", status `open`, blocked by `decision_4052`.

**Risk assessment.** The SPOF chain is:
1. `vault-data` volume lost → JWT keys lost → all services fail to authenticate → total outage.
2. `vault-keys` volume read by an attacker or leaked via host compromise → all 5 Shamir keys exposed → Vault fully compromised.

For a private beta on a single trusted VPS, this is an **accepted operational risk** if:
- Both volumes are included in the backup procedure (I-2).
- Host access is locked down (SSH key-only, firewall).
- The plaintext-keys risk is documented and understood.

**Decision needed.** Three options:
1. **Accept for beta** with volume backup + restricted host access + documented risk.
2. **KMS auto-unseal** (`task_ece2`): use AWS KMS or GCP CKMS to auto-unseal; removes the plaintext-key problem and the manual watchdog. Moderate effort.
3. **Managed Vault** (HCP Vault Secrets, Infisical, etc.): eliminates the entire Vault SPOF. Higher effort.

**Recommendation.** Accept option 1 for the private beta: add `vault-data` and `vault-keys` to the backup scope, document the risk, and gate public launch on `task_ece2`. The plaintext keys are a serious long-term concern but acceptable on a single, well-guarded VPS for a closed beta.

**Effort.** Option 1 (accept + backup): 0.5 days documentation. Option 2 (KMS): 2–3 days.

**Existing task.** `task_ece2` covers KMS migration. **No new task needed** — accept risk for beta, schedule `task_ece2` for pre-public-launch.

---

## I-4 — In-memory rate limiter breaks under multiple instances (FAST-FOLLOW)

**Current state.** The auth rate limiter (`pkg/middleware/rate_limit.go:12–41`) is a pure
in-memory token bucket: a `map[string]*bucket` protected by `sync.RWMutex` and a background
cleanup goroutine. Each process instance holds its own independent state. Under a
multi-instance deployment (e.g. `docker compose scale user-service=3`), a client can
bypass the limit by distributing requests across instances: each instance sees at most
`1/N` of the traffic and allows it.

**Evidence:**
- `pkg/middleware/rate_limit.go:12–18` — `RateLimiter` struct with `requests map[string]*bucket`.
- `services/users/internal/handlers/router.go:81–84` — rate limiter applied to auth endpoints only.
- `infrastructure/docker/prod/Caddyfile:35–45` — Caddy routes `user-service:8080` as a **single upstream**; no `upstream` block with multiple backends.

**Correction to the task lead.** The Caddyfile uses a single static upstream per service (no multi-instance routing). The current production topology is therefore **single-instance by design**, and the in-memory limiter works correctly for that topology. This is not a live bug — it is a scaling constraint that must be documented.

**Decision needed.** Formally declare: **beta runs single-instance per service**. Document the single-instance ceiling so operators do not `docker compose scale` without first deploying a Redis-backed limiter or removing the assumption. If horizontal scaling is needed before beta ends, `task_ece2`-style coordination (shared state) is required. Suggest creating a tracking issue for "Redis-backed rate limiter for horizontal scaling".

**Recommendation.** Accept for beta. Add a comment to `Caddyfile` and `docs/DEPLOYMENT.md` Scaling section noting that `docker compose scale` is **not supported** without a Redis limiter. File a ROADMAP task for Redis-backed limiter if/when horizontal scaling becomes a requirement.

**Effort.** Documentation: 1 hour. Redis-backed limiter: 1–2 days.

**Existing task.** None covers this directly. **NEW TASK recommended:** "Document single-instance topology as beta constraint; create Redis-backed rate-limiter plan" — FAST-FOLLOW.

---

## I-5 — DB `sslmode=disable` internally within Docker network (FAST-FOLLOW)

**Current state.** Every service that connects to `cairn-db` uses `sslmode=disable` (or
`DB_SSL_MODE=disable`). The connections are internal — no traffic leaves the Docker bridge
network (`cairn-network`, driver: bridge).

**Evidence:**
- `infrastructure/docker/prod/docker-compose.yml:230` — `explore-recommender`: `DB_SSLMODE=disable`
- `infrastructure/docker/prod/docker-compose.yml:270` — `explore-fetcher`: `DB_SSLMODE=disable`
- `infrastructure/docker/prod/docker-compose.yml:302` — `content-service`: `DB_SSLMODE=disable`
- `infrastructure/docker/prod/docker-compose.yml:338` — `content-worker`: `DB_SSLMODE=disable`
- `infrastructure/docker/prod/docker-compose.yml:366` — `ingest-rss`: `DB_SSLMODE=disable`
- `infrastructure/docker/prod/docker-compose.yml:394` — `ingest-rss-worker`: `DB_SSLMODE=disable`
- `infrastructure/docker/prod/docker-compose.yml:424` — `email-ingest`: `DB_SSL_MODE=disable`
- `infrastructure/docker/prod/docker-compose.yml:458` — `email-ingest-worker`: `DB_SSL_MODE=disable`
- `user-service` uses `DB_HOST=cairn-db` on the same Docker network (no `DB_SSLMODE` set; users service uses pgx which defaults to `prefer` — effectively no enforced TLS).

**Assessment.** Within a single-host Docker bridge network, inter-container traffic does not leave the kernel network stack; TLS in this context provides negligible security benefit (an attacker who can sniff kernel network traffic has already compromised the host). `sslmode=disable` is the conventional and correct setting here. Audit 2 (D-4) already noted this; the data matches.

**Decision.** Accept `sslmode=disable` for beta within Docker. If the architecture ever moves to separate hosts (e.g. app server ↔ managed RDS), TLS **must** be enabled — document this prerequisite in `docs/DEPLOYMENT.md`.

**Recommendation.** No code change. Add one sentence to the Production Considerations section of `docs/DEPLOYMENT.md`: "SSL is disabled for intra-Docker connections; re-enable if Postgres moves to a separate host."

**Effort.** 15 minutes documentation.

**Existing task.** None needed.

---

## I-6 — Connection ceiling: 9 containers × default 25 conns = up to 175+; Postgres `max_connections` unset in prod compose (BETA-BLOCKING)

**Correction to the task lead.** The task premise ("6 services × 25 = 150 vs Postgres default 100") is **understated**. The actual prod compose launches **9 containers** that each hold a connection pool. Furthermore, `max_connections` is **not configured at all** in the production Postgres service — the default of **100** applies.

**Connection math (prod compose, all defaults):**

| Container | Pool source | Default `MaxOpenConns` |
|---|---|---|
| user-service | `config.go:91` env `DB_MAX_OPEN_CONNS` default | 25 |
| explore-recommender | `config.go:51` env `DB_MAX_CONNS` default | 25 |
| explore-fetcher | `config.go:59` env `DB_MAX_CONNS` default | 25 |
| content-service | `connection.go:56` code default | 25 |
| ingest-rss (API) | `connection.go:47` DefaultConfig | 25 |
| email-ingest (API) | `connection.go:47` DefaultConfig | 25 |
| content-worker | `main.go:123` env `DB_MAX_OPEN_CONNS` default | 5 |
| ingest-rss-worker | `main.go:198` env `DB_MAX_OPEN_CONNS` default | 10 |
| email-ingest-worker | `connection.go:47` DefaultConfig (no override in main) | 25 |
| **Total possible** | | **190** |

Postgres's built-in overhead (`superuser_reserved_connections = 3`) means effective user slots
= `max_connections − 3 = 97`. **190 possible connections vs 97 available slots is a 2× oversubscription.** Under normal operation the pools will not all be simultaneously saturated, but a load spike or slow-query storm (the D-2 concern from Audit 2) can exhaust connections, causing `FATAL: sorry, too many clients already` across all services simultaneously.

**Evidence:**
- `infrastructure/docker/prod/docker-compose.yml:139–176` — no `command:` args on `cairn-db` (confirmed: line 62 shows only `command:` present on `selfhost` compose, not on prod).
- `infrastructure/docker/selfhost/docker-compose.yml:52–71` — selfhost correctly sets `max_connections=${POSTGRES_MAX_CONNECTIONS:-20}` and `DB_MAX_CONNS=3`, showing the team knows how to configure this.
- Pool defaults confirmed in: `services/users/internal/config/config.go:91`, `services/explore/recommender/internal/db/config.go:51`, `services/explore/fetcher/internal/db/config.go:59`, `services/read/content/internal/database/connection.go:56`, `services/read/fetcher/internal/database/connection.go:47`, `services/read/email/internal/database/connection.go:47`, `services/read/content/cmd/worker/main.go:123`, `services/read/fetcher/cmd/ingest_rss_worker/main.go:198`.

**Decision needed.** Two clean options, not mutually exclusive:

1. **Raise `max_connections`** in the prod postgres `command:` args (same pattern as selfhost). A value of `200` with shared_buffers tuned to match (each idle connection uses ~5–10 MB) is appropriate for the recommended 8 GB VPS.
2. **Lower pool caps** via env vars in the prod compose. Workers use small pools already; the API services can be capped at 10–15 with no user-visible impact at beta scale.

Both should be done: raise the server ceiling and cap the pools explicitly so the sum is documented and enforceable.

**Recommendation (BETA-BLOCKING).**
1. Add to `prod/docker-compose.yml` under `cairn-db`:
   ```yaml
   command:
     - "postgres"
     - "-c"
     - "max_connections=200"
     - "-c"
     - "shared_buffers=256MB"
   ```
2. Add `DB_MAX_OPEN_CONNS=15` (or `DB_MAX_CONNS=15`) to every API-service `environment:` block in the prod compose. Workers are already low (5, 10).
3. Document the sum in a comment in the compose file (pattern borrowed from selfhost).
4. This also resolves Audit 2 D-4's recommendation to "document the Σ MaxOpenConns budget".

**Effort.** 2 hours (config change + verification).

**Existing task.** Audit 2 D-4 flagged this as FAST-FOLLOW; this audit upgrades it to **BETA-BLOCKING** because the prod compose is missing `max_connections` entirely. **No existing task covers the prod-compose fix** — recommend filing a new task: "Set max_connections in prod compose and cap pool sizes" (BETA-BLOCKING).

---

## I-7 — Email ingest X-API-Key: no rotation tooling, no per-key abuse rate limit (FAST-FOLLOW)

**Current state.** The Cloudflare email worker authenticates to the email ingest service using
a static `X-API-Key` (`infrastructure/cloudflare/email-worker/wrangler.toml:11–13`; secret set
via `wrangler secret put`). The ingest service validates it by SHA-256-hashing the raw key and
comparing against the `api_keys` table (`services/read/email/internal/repository/apikey.go:33–55`).
The `api_keys` schema supports `status`, `expires_at`, and `last_used_at`
(`services/read/email/internal/repository/apikey.go:39–44`), which is a solid foundation.

**What is missing:**
1. **No rotation tooling.** There is no admin endpoint or script to rotate the key (generate new, insert new row, update Cloudflare secret, deactivate old row). The documented path is manual DB surgery + `wrangler secret put`.
2. **No per-key abuse rate limit.** Any caller who knows the API key (or obtains it via host compromise) can POST arbitrary email payloads at unlimited rate. The ingest endpoint is the most abusable surface because it writes directly to the DB and triggers downstream content processing.
3. **The key is effectively permanent.** `expires_at` supports expiry but nothing sets it, and the worker sets `secret_id_num_uses=0` (unlimited uses) and `secret_id_ttl=0` for AppRole credentials — the same unbounded-lifetime pattern.

**Evidence:**
- `infrastructure/cloudflare/email-worker/wrangler.toml:11–13` — `INGEST_API_KEY` is a wrangler secret.
- `infrastructure/cloudflare/email-worker/src/index.js:56` — `"X-API-Key": env.INGEST_API_KEY` sent on every email.
- `services/read/email/internal/api/middleware/apikey.go:31–54` — middleware validates key per request; no rate check.
- `services/read/email/internal/repository/apikey.go:33–57` — SHA-256 hash + DB UPDATE; correct design.

**Assessment.** The SHA-256 hash storage and `last_used_at` tracking are good design. The risk is operational: a compromised or leaked key has no bounded TTL and no independent abuse gate. For a beta with a small known set of Cloudflare-sourced emails, the attack surface is limited — but the email ingest endpoint is public-facing (routed through Caddy at `/api/v1/source/email/ingest`).

**Recommendation (FAST-FOLLOW).**
1. Write a key-rotation runbook in `docs/DEPLOYMENT.md` covering: generate new raw key, `INSERT INTO api_keys`, update `wrangler secret`, `UPDATE api_keys SET status='revoked'` on old row.
2. Set `expires_at` on new keys (e.g. 90-day rolling) and alert when approaching expiry.
3. Add a lightweight per-key request rate limit at the middleware layer (e.g. 10 ingests/minute — email newsletters are low-frequency by nature). The existing in-memory limiter infrastructure (`pkg/middleware/rate_limit.go`) supports a custom key function (`RateLimitWithKey`) that could key on the API key value.

**Effort.** Runbook: 1 hour. Rate limit + expiry: 0.5 day.

**Existing task.** None covers this. **NEW TASK recommended:** "Email API key rotation runbook + per-key rate limit" (FAST-FOLLOW).

---

## Infra checklist for Audit 7

Before signups open, the following must be true:

- [ ] **I-1** Single-instance SPOF decision recorded; restore runbook written and rehearsed (or managed DB in place).
- [ ] **I-2** `task_5dcb` delivered: `scripts/backup.sh` exists, cron container in prod compose, S3 upload verified.
- [ ] **I-3** Vault volume (`vault-data`, `vault-keys`) included in backup scope; plaintext-key risk documented; `task_ece2` scheduled for pre-public-launch.
- [ ] **I-6** `max_connections=200` (or equivalent) set in prod `cairn-db` command args; pool caps (≤15) added to all API service env blocks in prod compose.

Fast-follow during beta:

- [ ] **I-4** Single-instance topology documented in `docs/DEPLOYMENT.md` and Caddyfile comment; ROADMAP task filed for Redis-backed rate limiter.
- [ ] **I-5** `docs/DEPLOYMENT.md` note added: re-enable TLS if Postgres moves off the local host.
- [ ] **I-7** Email API key rotation runbook written; per-key rate limit implemented; key expiry set on bootstrap.

---

## Existing tasks this workstream maps to

| Task | Title | Status | Relevance |
|---|---|---|---|
| `task_5dcb` | Add database backup script and cron container | open (blocked `decision_4052`) | Directly covers I-2; unblocking is **BETA-BLOCKING** |
| `task_ece2` | Migrate Vault to auto-unseal via KMS | open (blocked `decision_4052`) | Covers I-3 long-term; accepted for beta, required pre-public-launch |
| `task_48ea` | Add Docker resource constraints to production compose | open (blocked `decision_4052`) | Adjacent to I-6 (pool caps); complementary — do both in the same PR |
| NEW | Set `max_connections` in prod compose + cap pool sizes | — | **BETA-BLOCKING** I-6 fix; create as child of `task_48ea` or standalone |
| NEW | Document single-instance topology + Redis-backed rate limiter plan | — | FAST-FOLLOW I-4 |
| NEW | Email API key rotation runbook + per-key rate limit | — | FAST-FOLLOW I-7 |
