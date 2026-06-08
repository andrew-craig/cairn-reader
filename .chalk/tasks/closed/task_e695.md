---
id: task_e695
title: Audit 2: Data layer & query efficiency
type: task
status: closed
priority: 2
labels: [audit,database]
blocked_by: []
parent: epic_9c21
remote_task_url: null
created_at: 2026-06-06T05:06:24Z
updated_at: 2026-06-08T05:22:41Z
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

## REVIEW (2026-06-08)

Findings delivered: `docs/architecture/audit/02_data_layer.md` (feeds Audit 7 consolidation).

Outcome — 9 findings, classified beta-blocking / fast-follow / roadmap:
- D-1 (BETA-BLOCKING): user-content **list + search** return full `cleaned_html` per item (≤5MB × up to 100). Confirmed via `contentToResponse` (`content_handler.go:143`) used by `ListUserContents`/`SearchUserContents`. Needs list-vs-detail split before the list DTO freezes (joint Audit 1 F-3 / Audit 3). Explore recommendation has the same shape but is bounded to 5 items → less severe.
- D-2 (BETA-BLOCKING): NO statement/query timeout in any of the six pools or the repo layer. Queries only stop on client disconnect or conn recycle; with MaxOpenConns=25 a few slow FTS queries exhaust the pool. One-line `statement_timeout` fix; epic biases abuse guards to beta-blocking.
- D-3 (FAST-FOLLOW): both "missing index" leads RESOLVE TO NO-ACTION — `contents(source_feed_id)` is always queried with `content_hash`+`source_type` (composite partial index covers); `feed_items(feed_id, processing_status)` — no query filters both cols. Real gap is `(user_id, status, added_at DESC)` for status-filtered sorts. `updated_at` is never a sort column (lead's worry moot).
- D-4 (FAST-FOLLOW): lead's driver split is BACKWARDS — pgx = users+explore, pq = all of read. Pool lifetimes diverge (content/explore 5m vs email/fetcher 1h); only explore is env-tunable. Unify in pkg/config; roadmap read→pgx.
- D-5 (FAST-FOLLOW): both `BulkCreate`s loop row-by-row (violates read/CLAUDE.md own guidance) → multi-row INSERT.
- D-6 (FAST-FOLLOW): unused `user_unread_counts` view is a latent `users × articles` CROSS JOIN (dead code, no Go refs) → drop it.
- D-7 (FAST-FOLLOW): offset pagination on `votes`/senders degrades at depth; content list already uses keyset. Freeze cursor shape now (overlaps Audit 1 F-3).
- D-8 (FAST-FOLLOW/COSMETIC): migrations versioned+reversible via golang-migrate, BUT explore ships duplicate legacy un-suffixed files AND destructive `000004` down is untested (suite only runs forward). Add up→down→up CI (Audit 5).
- D-9 (ROADMAP/NOTE): feed-limit COUNT-per-insert trigger and votes growth both correct at beta scale — votes display uses denormalised `articles.upvotes/downvotes` counters (O(1)), not COUNT.

Document ends with a data-layer checklist for Audit 7.
