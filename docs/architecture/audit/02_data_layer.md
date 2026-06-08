# Audit 2 — Data Layer & Query Efficiency

**Workstream:** `task_e695` (epic `epic_9c21`, P2)
**Status:** Findings complete
**Date:** 2026-06-08

> This is the data-layer scale workstream. The question is not "does it work" (it does, at dev
> scale) but "what breaks first when thousands of beta users hit a single Postgres instance with
> per-service logical DBs". Most findings here are **FAST-FOLLOW** (safe to fix during beta) — the
> exceptions are the two that either freeze an API contract (**D-1**, payload size) or remove an
> abuse/exhaustion guard (**D-2**, query timeouts), which the epic biases toward treating as
> beta-blocking.

## How to read the classification

| Class | Meaning | Why |
|---|---|---|
| **BETA-BLOCKING** | Freezes a contract, or removes a data-loss/abuse guard that is cheap now and expensive later | Reconcile before signups |
| **FAST-FOLLOW** | Real scale/efficiency issue, but internal and changeable mid-beta | Fix during beta |
| **ROADMAP / NOTE** | Correct-at-beta-scale; documented so it isn't rediscovered as a surprise | Track, no action now |

## Summary table

| ID | Finding | Class |
|---|---|---|
| D-1 | User-content **list** + search return full `cleaned_html` per item (up to 5 MB × 100) | **BETA-BLOCKING** (contract) |
| D-2 | No per-query / statement timeout anywhere in the repository or pool layer | **BETA-BLOCKING** (abuse guard) |
| D-3 | Index review: the two "missing index" leads **resolve to no-action**; one real gap is `(user_id, status, added_at)` | FAST-FOLLOW |
| D-4 | Connection pool + driver inconsistency (the lead's pgx/pq split is **backwards**; lifetimes diverge 5m vs 1h) | FAST-FOLLOW |
| D-5 | `BulkCreate` (contents + user_contents) loops row-by-row instead of one multi-row INSERT | FAST-FOLLOW |
| D-6 | Unused `user_unread_counts` view is a latent `users × articles` CROSS JOIN | FAST-FOLLOW |
| D-7 | Offset pagination on lists that grow per power-user (votes, senders, admin content) | FAST-FOLLOW (overlaps Audit 1 F-3) |
| D-8 | Migrations are versioned + reversible, but explore has duplicate legacy files and untested destructive downs | FAST-FOLLOW / COSMETIC |
| D-9 | `feed_subscriptions` `COUNT(*)`-per-insert trigger; `votes` growth | ROADMAP / NOTE (correct at 100-cap) |

---

## D-1 — User-content list & search return the full article body per item (BETA-BLOCKING)

The list and search endpoints hydrate every row with the **entire** `cleaned_html` (capped at 5 MB
per item by `chk_html_size`, `services/read/content/migrations/000001_initial_schema.up.sql:34`).

Path: `ListUserContents` selects up to `limit` (max 100) `user_contents` rows, collects their
content IDs, calls `contentRepo.GetByIDs` (`user_content_handler.go:163`), and maps each through
`contentToResponse`, which copies `CleanedHTML` verbatim into the DTO
(`services/read/content/internal/api/handlers/content_handler.go:143`; DTO field
`services/read/content/internal/api/dto/content.go:29`). `SearchUserContents` does the identical
hydration (`user_content_handler.go:643-656`). `GetByIDs` itself `SELECT`s `cleaned_html`
(`services/read/content/internal/repository/content.go:480-487`).

**Why it matters for beta:** a reading-list screen that pulls 20 items can transfer **tens of MB**
over a mobile connection just to render titles and statuses; at the 100-item cap it is unbounded in
practice. It also pins large strings through Postgres → Go → JSON for data the list view never
shows. This is the same payload concern Audit 1 raised (F-3) and Audit 3 will see from the client
side.

**Why it's beta-blocking:** the fix is a **list-vs-detail split** — the list/search DTO drops
`cleaned_html` (and likely `description`) and the client fetches the body via
`GET /api/v1/content/{content_id}` when a article is opened. That changes the frozen list-response
shape, so it must be decided **before** the contract freezes. The DB-side change is a projection
(select only list columns), not a schema change.

**Recommendation (BETA-BLOCKING, coordinate Audit 1 & 3):**
1. Add a list projection to the repository (`GetSummariesByIDs` returning title/author/description/
   image/url/timestamps — no `cleaned_html`) so the body never leaves Postgres on a list call.
2. Give `UserContentResponse` a `ContentSummary` for list/search and keep the full `ContentResponse`
   only on the single-item GET. Lock both shapes in the OpenAPI spec.
3. Explore's `GET /api/v1/explore/recommendation` has the same shape (full `articles.content` per
   item) but is **bounded to 5 items** (`services/explore/recommender/internal/db/vote_repository.go`
   pattern; recommender returns a fixed batch), so it is materially less severe — apply the same
   summary projection as a fast-follow.

---

## D-2 — No statement / per-query timeout in the repository or pool layer (BETA-BLOCKING)

Every repository call takes a `context.Context` and threads it into `QueryContext`/`ExecContext`
(good), but **nothing sets a deadline on that context for DB work**, and **no service sets Postgres
`statement_timeout`**. A query only stops if the HTTP client disconnects (cancelling the request
context) or the connection's `MaxLifetime` recycles it — neither bounds a single expensive query.

Evidence (absence): there is no `statement_timeout` in any connection string or `SET` —
`pkg/config/config.go:50-74` builds DSNs without it; the pgx pools (`services/users/internal/
database/db.go:36-48`, `services/explore/recommender/internal/db/config.go:38-54`) set
`MaxConns/MinConns/MaxConnLifetime` only; the `lib/pq` pools (`services/read/*/internal/database/
connection.go`) likewise. The only `context.WithTimeout` uses are for startup pings and outbound
HTTP, not query execution.

**Why it matters / why beta-blocking:** the full-text search (`to_tsvector(...) @@
plainto_tsquery(...)`, `user_content_handler` → `Search`) and any unindexed sort can run long on a
hostile or pathological input; with `MaxOpenConns = 25` per service, a handful of slow queries
exhausts the pool and the service stops serving everyone. A `statement_timeout` is the cheap,
standard abuse/DoS guard, and the epic explicitly biases beta-blocking toward "risks abuse". It is
a one-line change, so there is no reason to defer it past the freeze.

**Recommendation (BETA-BLOCKING):**
- Set a conservative server-side `statement_timeout` (e.g. `5s` for API pools, higher/none for the
  worker/migration pools) via the connection string (`options=-c statement_timeout=5000`) or a
  pool `AfterConnect` hook for pgx. Centralise it in `pkg/config` so all six pools inherit it.
- Optionally wrap repository calls in a `context.WithTimeout` as defence-in-depth, but the DB-side
  `statement_timeout` is the one that actually frees the connection.

---

## D-3 — Index review: the two "missing index" leads resolve to no-action; one real gap (FAST-FOLLOW)

I checked every query against the indexes that actually exist. The task's two specific
"possibly missing index" leads are **not** needed — the access paths are already covered. There is
one genuine (minor) gap.

### `contents(source_feed_id)` — NOT needed (resolve, no action)
The only queries that filter on `source_feed_id` **always also filter `content_hash` and
`source_type = 'rss'`** (`content.go:207` single, `content.go:424` bulk `= ANY($1)`). That is
exactly the partial composite `idx_contents_rss_dedup ON contents(content_hash, source_feed_id)
WHERE source_type = 'rss'` (`000001_initial_schema.up.sql:38`). No code path filters by
`source_feed_id` alone. A standalone index would be dead weight. *(Note: `services/read/CLAUDE.md`
documents a standalone `INDEX(source_feed_id)` that does not exist in the migration and isn't
needed — stale doc, flag for Audit 1/7.)*

### `feed_items(feed_id, processing_status)` — NOT needed (resolve, no action)
No query filters on `feed_id` **and** `processing_status` together. The worker queries filter one or
the other, and each is already covered:
- `WHERE processing_status = 'pending' ORDER BY discovered_at ASC` (`feed_item.go:332`) → covered by
  `idx_feed_items_status ON (processing_status, discovered_at)`.
- `WHERE feed_id = $1 ORDER BY discovered_at DESC` (`feed_item.go:371`) → covered by
  `idx_feed_items_feed ON (feed_id, discovered_at DESC)`.
- `WHERE feed_id = $1 AND item_guid = $2` (`feed_item.go:165`) → covered by `unique_feed_item`.

One modest gap: `WHERE processing_status = 'completed' ORDER BY last_checked_at ASC NULLS FIRST`
(`feed_item.go:351`, the update-recheck worker) is covered for the filter but must **sort** because
`idx_feed_items_status` orders by `discovered_at`, not `last_checked_at`. Bounded by `LIMIT`, so
low urgency — add `(processing_status, last_checked_at)` only if that worker shows up in slow logs.

### Real gap: `user_contents(user_id, status, added_at DESC)` — FAST-FOLLOW
Status-filtered list/cursor queries are `WHERE user_id=$1 AND status=$2 ... ORDER BY added_at DESC`
(`user_content.go:271-283`, `:329-347`). The existing `idx_user_contents_status ON (user_id,
status)` (`000001:73`) serves the filter but **not** the sort, so a status-filtered page does a
sort step; the unfiltered list is fine via `idx_user_contents_user ON (user_id, added_at DESC)`
(`000001:72`). Mobile always sorts by `added_at DESC` and filters by `status`, so this is the one
worth adding.

> Resolving the lead's third sub-question: **`updated_at` is never used as a sort column** anywhere
> in the content repository — every list/search orders by `added_at DESC` (cursor on `(added_at,
> id)`). So no `updated_at` index is warranted. `status` is used as a *filter*, addressed above.

**Recommendation (FAST-FOLLOW):** add `CREATE INDEX idx_user_contents_user_status_added ON
user_contents(user_id, status, added_at DESC)` and drop the now-redundant `idx_user_contents_status`
(the new index is a left-prefix superset). Treat the `feed_items` recheck index as optional.

---

## D-4 — Connection-pool and driver inconsistency (FAST-FOLLOW)

**Correction to the lead.** The task says *"pgx (users/read) vs pq (email/explore)"*. The actual
split is the opposite grouping:

| Service | Driver | Pool config source |
|---|---|---|
| users | **pgx** (`pgxpool`) | `services/users/internal/database/db.go:11,41-48` |
| explore/recommender | **pgx** (`pgxpool`) | `services/explore/recommender/internal/db/config.go:43-54` |
| explore/fetcher | **pgx** (`pgxpool`) | `services/explore/fetcher/internal/db/config.go` |
| read/content | **pq** (`lib/pq`) | `services/read/content/internal/database/connection.go:10,48` |
| read/fetcher | **pq** (`lib/pq`) | `services/read/fetcher/internal/database/connection.go:9,67` |
| read/email | **pq** (`lib/pq`) | `services/read/email/internal/database/connection.go:9,67` |

So it is **pgx → users + explore**, **pq → all of read**. `lib/pq` is in maintenance mode; pgx is
the actively-developed driver and the standard for new Go/Postgres work. Standardising read on pgx
is the long-term direction but is **not** beta-blocking.

**Pool settings also diverge** (the lead's "consistent 25/5/5m" is partly stale):

| Pool | Max | Idle/Min | MaxLifetime | MaxIdleTime |
|---|---|---|---|---|
| users (pgx) | 25 (from cfg) | from cfg | from cfg | — (pgx default) |
| explore (pgx) | 25 (`DB_MAX_CONNS`) | 5 (`DB_MIN_CONNS`) | **5m** | 5m |
| read/content (pq) | 25 | **10** (default) | **5m** | 2m |
| read/email, read/fetcher (pq) | 25 | 5 | **1h** | 10m |

`MaxConnLifetime` ranges from 5m (content/explore) to **1h** (email/fetcher `DefaultConfig`,
`connection.go:49` / `:105`), and content defaults idle to 10 while others use 5. With a single
Postgres instance shared by six logical DBs, total possible connections = Σ MaxOpenConns; at 25×6 =
150 that must stay under the server's `max_connections`. The override knobs only exist on explore
(`DB_MAX_CONNS`/`DB_MIN_CONNS`); the read/pq pools are not env-tunable.

**Recommendation (FAST-FOLLOW):**
- Unify pool defaults in `pkg/config` (`MaxOpenConns`, `MaxIdleConns`, `ConnMaxLifetime`,
  `ConnMaxIdleTime`) and have all six pools read them, with env overrides everywhere (not just
  explore). Pick one `ConnMaxLifetime` (5m is the safer default behind connection-recycling).
- Document the `Σ MaxOpenConns ≤ max_connections` budget (overlaps Audit 6).
- Track "standardise read on pgx" as roadmap, not beta work.

---

## D-5 — Bulk inserts loop row-by-row instead of a single multi-row INSERT (FAST-FOLLOW)

`contentRepository.BulkCreate` (`content.go:530-601`) and `userContentRepository.BulkCreate`
(`user_content.go:661-730`) open a transaction, prepare a **single-row** statement, then call it in
a Go `for` loop — one network round-trip per row. The project's own guidance says the opposite:
*"Prefer batch operations over loops … `INSERT ... VALUES ($1,$2),($3,$4)` — Good; `for range items
{ INSERT }` — Bad"* (`services/read/CLAUDE.md`, Database Operations). These feed the RSS/email
ingest bulk endpoints, so they run on the hot delivery path under fan-out.

**Recommendation (FAST-FOLLOW):** build one parameterised multi-row `INSERT ... VALUES (...),(...)`
(content already does dynamic placeholder construction in `GetByIDs`, so the pattern exists), or use
`pq.CopyIn` / pgx `CopyFrom` after the read→pgx migration. Keep the `ON CONFLICT DO NOTHING` on
user_contents. Bounded by the existing 100-item bulk cap, so correctness is fine today; this is
purely round-trip efficiency.

---

## D-6 — Unused `user_unread_counts` view is a latent `users × articles` CROSS JOIN (FAST-FOLLOW)

Migration `000004_voting_and_recommendations.up.sql` (re)creates
`CREATE OR REPLACE VIEW user_unread_counts AS SELECT ... FROM users u CROSS JOIN articles a LEFT
JOIN user_articles ua ...`. A CROSS JOIN materialises `|users| × |articles|` rows before
aggregating — at beta scale (thousands of users × tens of thousands of articles) that is hundreds of
millions of rows for a single `SELECT`.

**It is currently dead code** — `grep` finds no Go reference to `user_unread_counts` outside the
migrations. So it is a landmine, not an active bug: it survives in the schema and will be tempting to
query from a future "unread badge" feature.

**Recommendation (FAST-FOLLOW):** drop the view in a new migration (it has no consumers), and if an
unread count is needed later, compute it per-user with a `WHERE u.id = $1` predicate instead of a
global CROSS JOIN. Removing it now prevents someone wiring a homepage badge straight into a
full-table cartesian product.

---

## D-7 — Offset pagination on per-user-growing lists (FAST-FOLLOW; overlaps Audit 1 F-3)

`LIMIT/OFFSET` is fine for shallow, bounded lists but degrades linearly with offset depth (Postgres
still scans+discards the skipped rows). The content user-list already uses **keyset/cursor**
pagination correctly (`ListByUserWithCursor`, `user_content.go:319-348`). The remaining offset users:

- `votes` — `GetUserVotedArticles ... ORDER BY v.created_at DESC LIMIT $2 OFFSET $3`
  (`vote_repository.go:257-270`). Power voters page deep.
- email senders, explore recommendation offset (carried from Audit 1 F-3).

`idx_votes_user` covers the filter but deep offsets still cost. Bounded today; flagged so the
**response shape** can be frozen as cursor-capable now (the shared `PaginationInfo` already carries
both `cursor` and `offset`, per Audit 1) and the implementation swapped to keyset during beta
without a contract change.

**Recommendation (FAST-FOLLOW):** keep offset for genuinely-bounded lists; for `votes` (and any
list that grows per active user) lock a cursor-shaped response now and move the query to keyset
(`(created_at, id) < (...)`) when it shows up in slow logs. Coordinate the param/response naming with
Audit 1.

---

## D-8 — Migrations: versioned & reversible, but explore has duplicate legacy files + untested destructive downs (FAST-FOLLOW / COSMETIC)

**Confirmed good:** every service uses `golang-migrate` with numbered `NNNNNN_*.up.sql` /
`*.down.sql` pairs embedded via `embed.go`, applied on startup. Down-migrations exist for every up.

**Two issues:**
1. **Duplicate legacy files (COSMETIC, but confusing):** `services/explore/recommender/migrations`
   and `services/explore/fetcher/migrations` contain BOTH the `golang-migrate` pairs
   (`000001_init.up.sql` …) **and** older un-suffixed copies (`001_init.sql`, `002_…`, …). The
   un-suffixed files are not part of the `golang-migrate` sequence and are dead/duplicative; they
   should be deleted to avoid someone editing the wrong one.
2. **Destructive, almost-certainly-untested downs (FAST-FOLLOW):** `000004` on the recommender does
   a table rename + recreate + data backfill (`ALTER TABLE users RENAME TO users_old; CREATE TABLE
   users ...; INSERT INTO users SELECT ...; DROP TABLE users_old CASCADE`). Its `.down.sql` has to
   reverse that exactly, and nothing in the test suite runs `migrate down` (only forward migration is
   exercised in `testutil`). A down that has never executed is a down that does not work.

**Recommendation:**
- Delete the duplicate un-suffixed explore migration files (COSMETIC).
- Add a CI step (Audit 5 territory) that runs `up → down → up` on a throwaway DB so reversibility is
  actually verified, at least for the latest N migrations. For irreversible/destructive migrations,
  mark the down explicitly as a no-op-with-comment rather than shipping a down that silently corrupts.

---

## D-9 — `feed_subscriptions` COUNT-per-insert trigger; `votes` growth (ROADMAP / NOTE)

Documented so they aren't rediscovered as surprises; **no action for beta**.

- **`check_feed_limit` trigger** runs `SELECT COUNT(*) FROM feed_subscriptions WHERE user_id = NEW.user_id`
  on **every** subscription insert (`000001_initial_schema.up.sql:59-79`). At the 100-feed cap with
  `idx_feed_subs_user`, this counts ≤100 indexed rows — negligible. It only matters if the cap is
  ever raised by orders of magnitude, at which point a counter column or a partial-unique scheme
  would be preferable. Note only.
- **`votes` growth:** one row per (user, article) voted, `UNIQUE(user_id, article_id)`. Crucially,
  per-article display does **not** `COUNT` the votes table — `articles.upvotes/downvotes` are
  **denormalised counters** updated transactionally in `RecordVote`/`RemoveVote`
  (`vote_repository.go:65-148,183-216`), and `GetVoteCounts` reads them O(1) from `articles`
  (`vote_repository.go:225-237`). So the read path is already scale-safe; `votes` grows linearly with
  engagement but is only scanned per-user (`idx_votes_user`) or per-article (`idx_votes_article`).
  Roadmap concern is table size/retention, not query cost.

---

## Data-layer checklist (feeds Audit 7)

Before signups open:

- [ ] **D-1** List/search responses use a summary projection (no `cleaned_html`); list + detail
      shapes frozen in the spec (joint with Audit 1 F-3 / Audit 3).
- [ ] **D-2** `statement_timeout` set on the API connection pools (centralised in `pkg/config`).

Fast-follow during beta:

- [ ] **D-3** Add `idx_user_contents_user_status_added`; drop redundant `idx_user_contents_status`.
      (The two leads' indexes are confirmed unnecessary — no action.)
- [ ] **D-4** Unify pool defaults + env overrides across all six pools; document `Σ MaxOpenConns`
      budget; roadmap: read → pgx.
- [ ] **D-5** Convert both `BulkCreate`s to multi-row INSERT.
- [ ] **D-6** Drop the unused `user_unread_counts` CROSS JOIN view.
- [ ] **D-7** Freeze cursor-shaped pagination for `votes`; swap to keyset when needed.
- [ ] **D-8** Delete duplicate explore migration files; add `up→down→up` CI check.

No action (documented): **D-9** feed-limit trigger and votes growth are correct at beta scale.
