# Explore — archived code-quality findings

**Archived because the Explore service was removed in #425** (fetcher, recommender, and the Explore tab). These findings were extracted verbatim from [CODE_QUALITY_REVIEW.md](/docs/CODE_QUALITY_REVIEW.md) (dated 2026-07-05) and from [QUALITY_REMEDIATION_STRATEGY.md](/docs/QUALITY_REMEDIATION_STRATEGY.md). Original finding IDs are kept so references stay traceable. Paths below refer to deleted code (`services/explore/...`). Findings that spanned Explore and other services remain in the ledger with the Explore parts removed.

---

## Part 1 — Theme-level mentions

### Theme 1 — Authorization boundaries (Explore part)
- **explore**: `POST /api/v1/explore/article` and the fetcher's `/fetch` / `/sync` triggers have no service-to-service auth — anyone who can reach them can inject articles or hammer triggers. (Remediation: PR #317.)

### Theme 4 — Job claiming (Explore part)
- explore/fetcher (`GetNextFeed`) — no row locking. (Remediation: PR #339 covered read/fetcher, explore/fetcher and email outbox.)

### Theme 5 — Features marked "done" (Explore part)
- **explore**: `feed_id` is never populated by the production fetch path (all articles have `feed_id = NULL`); the documented discovery/exploration recommendation slot is dead code.

## Part 1 — High findings

### Explore service
- **H8** A single malformed/colliding article poisons the whole feed batch (`recommender/internal/db/article_repository.go:77-127`): one multi-row `INSERT ... ON CONFLICT (link)`. An empty/duplicate `<link>` trips "cannot affect row a second time"; and because article `id = ContentHash(content)` (not the link), syndicated duplicate content collides on the primary key and **recurs on every subsequent poll** until the feed auto-disables — silently discarding all good articles in the batch. (Remediation: PR #322, with P2-C7.)
- **H9** `feed_id` never populated in production (`fetcher/internal/fetcher/fetcher.go:168-200`) — every article stored with `feed_id = NULL`, breaking documented per-feed metadata/analytics.

## Part 1 — Medium & low

**Explore:** no row locking in `GetNextFeed`; SSRF exposure in feed fetching (no allow/deny-list); internal endpoints unauthenticated; dead `GetLowExposureArticles`; quality-score doc/code sign mismatch (code correct); config-bootstrap duplication + divergent migrate defaults.

## Part 2 findings

### P2-C7 — Recommender batch drops all articles on a content-hash collision (mechanism behind Part 1 explore-H8)
`services/explore/recommender/internal/db/article_repository.go:77-127` upserts via `ON CONFLICT (link) DO UPDATE`, but `id` is a *separate* primary key set to `SHA256(cleaned_content)` (per `fetcher/migrations/000002`). When two different links carry the same content (syndicated/re-published articles — the exact case the content-hash ID was designed to dedupe), the new row's `link` doesn't conflict but its `id` does → an **unhandled** PK violation, and because it's one multi-row `INSERT`, the whole batch fails and silently drops every other new article in that fetch cycle. Fix: arm the `id` constraint (`ON CONFLICT (id) ...`). (Remediation: PR #322.)

### Also from the DB pass (Explore parts)
- **[High] Every `explore/recommender` `down.sql` (000001–000006) is an empty comment stub** — including the two destructive user-ID-migration steps (000004, 000006). `make migrate-down` silently "succeeds" while reversing nothing — false reversibility.
- Medium: unindexed `ILIKE '%…%'` search on the unbounded `articles` table (no `pg_trgm`). Low: `explore/*` uses naive `TIMESTAMP` while everything else uses `TIMESTAMP WITH TIME ZONE`.

### Cross-service contracts (Explore part)
- **Solid:** the one seam that shared a Go type (`explore` fetcher→recommender used `pkg/models.Article` on both sides) never drifted. (`pkg/models` still exists but has no Go importers outside `cmd/selfhost`'s `replace` directive, so it is no longer a live reference pattern.)

## Part 2 — Other Explore mentions

- Dependency audit (P3-H1): pgx SQLi reachable at `services/explore/recommender/internal/db/vote_repository.go:289 GetUserVotedArticles`. `services/explore` pinned `x/net` v0.42.0 (P3-C1 list).
- `golang.org/x/crypto/ssh` — 5 advisories (GO-2026-5013/5017/5018/5019/5020, fixed v0.52.0) in `services/explore` at v0.40.0, reachable only through `testcontainers-go` test helpers (`internal/testutil/db.go`).
- Docker/Moby CVEs in `services/explore` — reachable only via `testcontainers-go` test helpers.
- Mobile: H12 also named `explore.ts:86-108` (second 401 after refresh swallowed). Mobile "Read/Explore/Bookmarks/Votes have no error state".
- Self-host readiness: explore-recommender and explore-fetcher were two of the 3 of 6 databases missing from `/health/ready` (fixed by PR #403 for all 6).
- Tenancy: unauthenticated article injection into the shared explore catalog; explore ingest endpoints needed `RequireInternalAPIKey`.
- Key rotation (P2-C1): explore-recommender called `vaultClient.GetPublicKey()` once at startup like content and email-ingest.
- Test gating: `services/explore` gated integration tests with `//go:build integration` like its siblings.
- Pagination: `recommender/handlers.go:450,496` clamped `limit` to ≤100.

## Remediation-playbook references to Explore code

Previously in [QUALITY_REMEDIATION_STRATEGY.md](/docs/QUALITY_REMEDIATION_STRATEGY.md):

- **Reference pattern, request body caps (413):** `explore/recommender` handlers' `http.MaxBytesReader` usage (10MB/16KB/1KB per handler). Replaced by the live equivalent `services/read/content/internal/api/handlers/limits.go`.
- **Reference pattern, cross-service DTO that can't drift:** `pkg/models.Article` shared by both sides of the explore seam. Retired (no live seam shares a Go type).
- **R7 note:** testcontainers was already used in explore (`internal/testutil`). Live equivalents: `services/read/{content,fetcher,email}/internal/testutil` and `services/users/test/integration/testutil`.
- **Ledger rows (both ticked):** #317 Explore auth (article-inject + fetch/sync trigger endpoints unauthenticated); #322 P2-C7 / H8 recommender upsert conflicting on content-hash `id`. #339 (`FOR UPDATE SKIP LOCKED`) also covered explore/fetcher.
