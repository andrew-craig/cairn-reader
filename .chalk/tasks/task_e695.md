---
id: task_e695
title: Audit 2: Data layer & query efficiency
type: task
status: open
priority: 2
labels: [audit,database]
blocked_by: []
parent: epic_9c21
remote_task_url: null
created_at: 2026-06-06T05:06:24Z
updated_at: 2026-06-06T05:06:24Z
---
Audit the Postgres data layer for efficiency and scalability under thousands of beta users.

Examine: services/*/migrations, services/*/internal/repository (and database/), pkg/config/config.go (pool defaults), pkg/models.

Findings to confirm/resolve:
1. Possibly missing indexes: contents(source_feed_id) (composite dedup index only helps when filtering both cols); feed_items(feed_id, processing_status); review user_contents sort columns (only added_at DESC indexed — check if updated_at/status sorts are used by mobile).
2. Payload size: GET user contents and explore recommendation return full cleaned_html/content per item — large pages (potentially MBs per 20 items). Consider list-vs-detail split or summary projection. Coordinate with Audit 1 & 3.
3. Query timeouts: connection lifetime capped (5m) but no per-query context timeout in repository layer. Add statement timeouts.
4. feed_subscriptions limit trigger does a COUNT(*) per insert (fine at 100-cap, note for scale).
5. Connection pools consistent (25/5/5m) but driver split: pgx (users/read) vs pq (email/explore) — note standardisation.
6. Confirm migrations are versioned + reversible (golang-migrate) and that down-migrations are tested.
7. Check for unbounded queries / missing LIMIT, and the votes table growth pattern (see Audit 3 vote-count issue).

Deliverable: findings section with each index/query issue, proposed fix (DDL or query change), and beta-blocking vs fast-follow classification.
