---
id: epic_381e
title: Move to a cloud-hosted instance for beta customers
type: epic
status: open
priority: 1
labels: [infra,hosting]
blocked_by: []
parent: epic_7c9e
remote_task_url: null
created_at: 2026-11-14T03:20:02Z
updated_at: 2026-11-14T03:20:02Z
---

Run the multi-service stack (`infrastructure/docker/prod/`) on a cloud provider so Cairn can be offered to beta customers as a hosted service. Step toward the Hosted launch epic (epic_7c9e).

## Target topology (beta)
- One **Intel (x86_64 / linux/amd64)** VPS running the prod Docker Compose stack: Caddy, Vault (+ init/unseal), user-service, content-service, content-worker, ingest-rss, ingest-rss-worker, email-ingest, email-ingest-worker, web.
- **Managed PostgreSQL** in the same region, reached over a private network, replacing the `cairn-db` container. The VPS then holds no customer data, so it can be rebuilt or resized freely.
- TLS at the edge (host proxy or provider load balancer) forwarding to Caddy with `X-Forwarded-Proto: https`.
- Cloudflare keeps MX/DNS for the email domain; the Email Worker POSTs to `/api/v1/source/email/*`.

## Provider requirements
- x86_64 VM with block storage + snapshots; region close to users (Sydney if users are mostly in AU).
- Managed Postgres: PG 18 (or accept 17); admin can `CREATE USER` / `CREATE DATABASE ... OWNER` (`init-databases.sh`); `pgcrypto` allowed; ~80+ connections or a built-in PgBouncer (prod pool caps sum to 80); private networking; TLS (`DB_SSLMODE=require`); point-in-time recovery; standby (HA) available later.
- Container runtime allows `cap_add: IPC_LOCK` and persistent volumes (Vault).
- Unrestricted outbound egress (RSS feeds and arbitrary article URLs); a static egress IP is a plus.
- Can pull images from GHCR.
- Off-server object storage for Vault volume backups; log collection and uptime checks on `/health/ready`.

## Known constraints
- Single instance per service for now: workers run in-memory schedulers and user-service rate-limits in memory. Scale vertically until Redis rate limiting (`RateLimitRedis`) and worker locking are in place.
- Vault is the only stateful component left on the VPS (JWT signing keys + unseal keys). Back up `vault-data` / `vault-keys` off-server and keep the unseal keys separately.

## Decision
Owner chose to **fix and use `infrastructure/docker/prod/`** (not delete it). This resolves the decision in task_4f8a, whose blockers must be cleared as part of this epic.

## Plan
- [ ] Build per-service images for linux/amd64 (task_f489)
- [ ] Clear the remaining task_4f8a blockers (enable image publishing, `INTERNAL_API_KEY` on ingest-rss, dev compose Vault vars, delete `docker-test.yml`)
- [ ] Point prod compose at managed Postgres: remove `cairn-db`, make `DB_HOST` / `DB_SSLMODE` configurable, document running `init-databases.sh` against the managed instance
- [ ] Choose provider and provision VPS + managed Postgres
- [ ] TLS/ingress, DNS, Cloudflare Email Worker pointed at the hosted endpoint
- [ ] Backups (managed PITR + Vault volumes), logging, uptime monitoring
- [ ] End-to-end smoke test and beta onboarding
