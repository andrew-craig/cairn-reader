---
id: task_1d08
title: Audit 6: Infrastructure reliability & scaling
type: task
status: in_progress
priority: 2
labels: [audit,infra,reliability]
blocked_by: []
parent: epic_9c21
remote_task_url: null
created_at: 2026-06-06T05:06:57Z
updated_at: 2026-06-16T10:23:34Z
---
Audit infra for reliability and the path from single-instance to scaled public beta. infrastructure/docker/{prod,selfhost,dev}, Caddyfile, vault-config, infrastructure/cloudflare/email-worker, docs/DEPLOYMENT.md.

Findings to confirm/resolve:
1. Single PostgreSQL instance = single point of failure, no replication/failover. Decide: managed DB (RDS/Cloud SQL) vs accepted risk + tested restore for beta.
2. Backups documented but NOT automated (pg_dumpall script in docs only, no cron container, no offsite/S3). Cross-ref task_5dcb. Beta needs automated, tested backups.
3. Vault: file storage (not HA) + unseal keys stored plaintext in volume. Cross-ref task_ece2 (KMS auto-unseal). Assess SPOF if volume lost.
4. Rate limiter + any in-memory state breaks under multiple instances — decide single-instance-for-beta vs Redis-backed (cross-ref Audit 4).
5. DB sslmode=disable internally — assess whether acceptable within Docker network for beta.
6. No horizontal-scaling story (Caddy -> single upstream per service). Decide whether beta is single-instance and document the ceiling (note: 6 services x 25 conns = 150 vs Postgres default max_connections 100 — verify this is configured up!).
7. Email path: Cloudflare worker -> email ingest via X-API-Key — review key rotation + abuse.

Deliverable: findings section with reliability/scaling risks, the decision needed for each (accept vs fix for beta), effort, and links to existing related chalk tasks.
