# Audit 1 — API Contract Consistency & Stability

**Workstream:** `task_2a40` (epic `epic_9c21`, P1 — most time-sensitive)
**Status:** Findings complete
**Date:** 2026-06-07

> This is the contract-freeze workstream. Once the public beta opens, every shape
> below is pinned by mobile apps in the wild. Everything classified **BETA-BLOCKING**
> must land before signups open; **FAST-FOLLOW** can land during beta without breaking
> clients; **COSMETIC** is cleanup with no client impact.

## How to read the classification

| Class | Meaning | Why |
|---|---|---|
| **BETA-BLOCKING** | Contract-affecting; changing it post-freeze forces client upgrades or dual-support | Must reconcile before freeze |
| **FAST-FOLLOW** | Inconsistent but not client-frozen (internal/ops surface) | Safe to fix during beta |
| **COSMETIC** | No wire impact (docs, comments, tests) | Cleanup |

## Summary table

| ID | Finding | Class |
|---|---|---|
| F-1 | Error/success envelope unified in handlers, but middleware/framework paths diverge | **BETA-BLOCKING** |
| F-2 | Health-check format inconsistent (email plain text, explore adds timestamp) | FAST-FOLLOW |
| F-3 | Pagination inconsistent (cursor vs offset vs none) + unbounded list endpoints | **BETA-BLOCKING** (naming) / FAST-FOLLOW (unbounded) |
| F-4 | Read **content** OpenAPI spec does not match implemented routes | **BETA-BLOCKING** |
| F-5 | Read **fetcher** (RSS) OpenAPI spec does not match implemented routes | **BETA-BLOCKING** |
| F-6 | OpenAPI specs document an outdated/partial error schema | **BETA-BLOCKING** |
| F-7 | Routes implemented but absent from specs; explore spec mixes two services | FAST-FOLLOW |
| —  | Versioning: every public endpoint correctly under `/api/v1` | PASS (no action) |

---

## F-1 — Error/success envelope: unified in handlers, divergent in middleware (BETA-BLOCKING)

**Correction to the original lead.** The task description's premise — *"users returns `{error}`,
read/explore return `{error,message,details}`, email wraps everything in `{data,meta}`"* — is **stale**.
The handler layer has since converged on the shared helpers in `pkg/api/response.go`:

- `WriteError` → `{error, message, details?, meta:{timestamp,version}}` (`pkg/api/response.go:70-83`)
- `WriteSuccess` → `{data, meta}` (`pkg/api/response.go:57-68`)
- `WritePaginated` → `{data, pagination, meta}` (`pkg/api/response.go:85-97`)

Usage is dominated by the shared helpers (e.g. content service: 87× `api.WriteError`, 15× `api.WriteSuccess`,
2× `api.WritePaginated`). The `{data,meta}` wrapper is **not** unique to email — every service uses it.

**The real, remaining inconsistency is at the middleware / framework layer**, which bypasses `pkg/api`
and emits envelopes a client cannot parse uniformly:

| Path | Wire shape | Diverges how | Evidence |
|---|---|---|---|
| Auth 401 (all protected routes) | `{error, message}` | no `meta` | `pkg/auth/middleware.go:27-31,99-107` |
| Panic 500 (all routes) | `{error, request_id}` | `error` is a sentence, not a code; no `message`/`details`/`meta`; extra `request_id` | `pkg/middleware/recovery.go:42-46` |
| Content-Type 400 (content svc) | `{error, message, details}` | no `meta`; uses local `dto.ErrorResponse` | `services/read/content/internal/api/middleware/validation.go:19` via `.../middleware/error_handler.go:10-21`, `.../api/dto/content.go` |

A client that switches on `body.error` as an enum and reads `body.meta` will succeed on handler errors
and fail on the most common error a client actually hits (a 401). This is contract-affecting.

**Canonical convention (proposed):** the `pkg/api` `ErrorResponse` envelope is the single source of truth —
`{ error: <code from pkg/api constants>, message: <human readable>, details?: object, meta: {timestamp, version} }`.

**Remediation (before freeze):**
1. Route `pkg/auth/middleware.go` 401s through `api.WriteError(..., api.ErrCodeUnauthorized, ...)` and delete the local `ErrorResponse`.
2. Make `pkg/middleware/recovery.go` emit `api.WriteError(w, 500, api.ErrCodeInternal, "an unexpected error occurred...", map{"request_id": id}, "v1")` so the code is a stable enum and `request_id` moves into `details`.
3. Replace the content service's local `middleware.WriteError`/`dto.ErrorResponse` (Content-Type guard) with `api.WriteError`; remove the now-orphaned local helper and DTO.

---

## F-2 — Health-check format inconsistent (FAST-FOLLOW)

`/health/live` and `/health/ready` are **internal orchestrator probes, not part of the client-frozen
contract**, so this is not beta-blocking — but the formats should still be unified (cheap now).

| Service | `/health/live` body | Content-Type | Evidence |
|---|---|---|---|
| users | `{"status":"healthy"}` | json | `services/users/internal/handlers/health.go:37-43` |
| read/content | `{"status":"healthy"}` | json | `services/read/content/internal/api/router.go:48-52` |
| read/fetcher | `{"status":"healthy"}` | json | `services/read/fetcher/internal/api/router.go:41-45` |
| explore/recommender | `{"status":"healthy"}` | json | `services/explore/recommender/internal/api/handlers.go:29-37` |
| explore/fetcher | `{"status":"healthy","timestamp":...}` | json | `services/explore/fetcher/cmd/explore_fetcher/main.go:207-222` |
| **read/email** | **`OK`** | **none set** | `services/read/email/internal/api/router.go:74-88` |

**Canonical convention (proposed):** liveness `{"status":"healthy"}`; readiness
`{"status":"healthy"|"unhealthy","checks":{<dep>:"ok"|"error"}}`; `Content-Type: application/json`; 200/503.

**Remediation (FAST-FOLLOW):** convert email's `handleLiveness`/`handleReadiness` to JSON with Content-Type;
drop the non-standard `timestamp` from explore/fetcher (or add it everywhere — pick one).

---

## F-3 — Pagination inconsistent + unbounded list endpoints (BETA-BLOCKING naming / FAST-FOLLOW scale)

The *names and response shape* of pagination freeze with the contract, so divergence here is partly
beta-blocking. The original lead ("read/explore use limit/offset; users has none") is **partly stale** —
content has since moved to cursor pagination.

| Endpoint | Style | Params | Cap | Evidence |
|---|---|---|---|---|
| `GET /api/v1/content/user/{id}` | **cursor** | `limit`,`cursor` | 100 | `services/read/content/internal/api/handlers/user_content_handler.go:84-199` |
| `GET /api/v1/content/user/{id}/search` | **cursor** | `q`,`limit`,`cursor` | 100 | `.../user_content_handler.go:580-670` |
| `GET /api/v1/content/user/{id}/subscriptions` | **none** | — | unbounded | `services/read/content/internal/api/router.go:106` |
| `GET /api/v1/source/email/.../senders` | **offset** | `limit`,`offset` | 100 | `services/read/email/internal/api/handlers/sender_handler.go:27-80` |
| `GET /api/v1/explore/recommendation` | **offset-only** | `offset` (fixed batch 5) | — | `services/explore/recommender/internal/api/handlers.go:124-156` |
| `GET /api/v1/explore/user/votes` | **offset** | `limit`,`offset` | 100 | `.../handlers.go:414-452` |
| `GET /api/v1/source/rss/user/{id}/subscription` | **none** | — | bounded (100 feeds by trigger) | `services/read/fetcher/internal/api/handlers/subscription_handler.go:122-146` |
| users service | none (no list endpoints) | — | — | `services/users/internal/handlers/router.go:40-109` |

Two distinct problems:

1. **Mixed contract (BETA-BLOCKING):** cursor (`cursor`) vs offset (`offset`) vs offset-only across services.
   The shared `PaginationInfo` (`pkg/api/response.go:41-48`) already carries *both* `cursor` and `offset`,
   so the envelope is fine — but the **request** param names and the response `pagination` fields a client
   reads differ per endpoint. Decide the convention before freeze.
2. **Unbounded lists (FAST-FOLLOW / overlaps Audit 2 & 3):** the content **subscriptions aggregator**
   (`services/read/content/internal/api/handlers/subscription_aggregator_handler.go`) returns an
   unpaginated, aggregated list (RSS feeds, capped at 100, + email senders, *uncapped*); `explore/user/votes`
   uses offset, which degrades on power users.

**Canonical convention (proposed):** offset/limit (`limit`,`offset`, default 20, max 100) is acceptable for
**bounded** lists (email senders, RSS feeds, votes). Use **cursor** (`limit`,`cursor`) for lists that grow
without bound (user contents — already correct). Document the choice per endpoint in the spec so it freezes
deliberately. Paginating the subscriptions aggregator is a scale concern; defer the implementation to Audit 2/3
but lock the *param/response shape* now.

---

## F-4 — Read **content** OpenAPI spec does not match implemented routes (BETA-BLOCKING)

Carried over and re-confirmed from the task's verified findings. The spec — the artifact a freeze locks —
documents paths that do not exist. Ground truth = `services/read/content/internal/api/router.go:79-113`
(everything under singular `/api/v1/content`, user routes nested as `/user/{user_id}`), and the mobile client
agrees (`apps/mobile/src/services/read.ts:96,144,183,218,250`).

| Real path (router + mobile client) | OpenAPI spec (`services/read/api/openapi.yaml`) |
|---|---|
| `GET/POST /api/v1/content/user/{user_id}` | `/api/v1/users/{user_id}/contents` (L483) |
| `GET /api/v1/content/user/{user_id}/search` | `/api/v1/users/{user_id}/contents/search` (L602) |
| `PATCH/DELETE /api/v1/content/user/{user_id}/{content_id}` | `/api/v1/users/{user_id}/contents/{content_id}` (L661) |
| `POST /api/v1/content` ; `GET/PUT /api/v1/content/{content_id}` | `/api/v1/contents` ; `/api/v1/contents/{id}` (L291,L332) |
| `POST /api/v1/content/check-duplicate` | `/api/v1/contents/check-duplicates` (L448) |

Spec diverges on three axes: segment `content` vs `contents`; nesting `/content/user/{id}` vs
`/users/{id}/contents`; `check-duplicate` vs `check-duplicates`.

**Resolved (task_6525):** the divergent spec was the stale *aggregate* `services/read/api/openapi.yaml`,
which has been deleted. The canonical per-sub-service spec `services/read/content/api/openapi.yaml` already
pins the correct singular forms (`/api/v1/content`, `/content/user/{user_id}`, `check-duplicate`). The
remaining follow-ups below (handler doc-comments, routing unit tests through the production router) are
tracked separately and are not spec-drift.

**Remediation (BETA-BLOCKING):** the shipped client already pins the singular form, so make the **spec match
the implementation** (cheapest correct move). Then fix the contradictory handler doc-comments
(`user_content_handler.go:83` is wrong, `:202` is right) and re-point the unit tests through the production
router (`user_content_handler_test.go` builds its own chi context and passes against the *wrong* paths;
only `integration_test.go:105` exercises the real route) so drift is caught going forward.

---

## F-5 — Read **fetcher** (RSS) OpenAPI spec does not match implemented routes (BETA-BLOCKING)

Same pattern as F-4, independently confirmed. Router ground truth = `services/read/fetcher/internal/api/router.go:73-86`.
Both clients agree with the router, not the spec: the mobile client calls
`/api/v1/source/rss/user/{userId}/subscription` (`apps/mobile/src/services/read.ts:442`) and the content
service's internal RSS client does too (`services/read/content/internal/service/ingest_rss_client.go:97,153,189`).

| Real path (router + both clients) | OpenAPI spec (`services/read/fetcher/api/openapi.yaml`) |
|---|---|
| `POST /api/v1/source/rss/user/{user_id}/subscription` | `/api/v1/users/{user_id}/feeds/subscribe` |
| `GET /api/v1/source/rss/user/{user_id}/subscription` | `/api/v1/users/{user_id}/feeds` |
| `DELETE /api/v1/source/rss/user/{user_id}/subscription/{feed_id}` | `/api/v1/users/{user_id}/feeds/{feed_id}` |
| `PATCH /api/v1/source/rss/feed/{feed_id}` | `/api/v1/feeds/{feed_id}/enable` |

**Remediation (BETA-BLOCKING):** make the spec match the implementation (`/api/v1/source/rss/...`), consistent
with the F-4 decision. Both clients already ship the `/source/rss` form.

---

## F-6 — OpenAPI specs document an outdated/partial error schema (BETA-BLOCKING)

The spec is the frozen artifact, so a spec that misdescribes the error envelope freezes a lie.

| Spec | Documented error schema | Matches real `{error,message,details,meta}`? |
|---|---|---|
| `services/users/api/openapi.yaml` (~L897-906) | `{error}` only | No — missing message/details/meta |
| `services/explore/api/openapi.yaml` (~L618-625) | `{error}` only | No |
| `services/read/fetcher/api/openapi.yaml` (~L377-388) | `{error, code, details}` | No — `code` doesn't exist; missing message/meta |
| `services/read/email/api/openapi.yaml` (~L410-425) | `{error, message, details, meta}` | **Yes** (use as the template) |

**Remediation (BETA-BLOCKING):** update users, explore, and fetcher specs to the canonical envelope (F-1),
using the email spec's schema as the reference component (factor it into a shared `ErrorResponse` schema).

---

## F-7 — Routes implemented but undocumented; explore spec conflates two services (FAST-FOLLOW)

- `POST /api/v1/explore/shown` exists in the router (`services/explore/recommender/internal/api/server.go:68`) but is absent from the spec.
- `GET /api/v1/internal/source/email/user/{user_id}/senders` exists (`services/read/email/internal/api/router.go:66`) but is undocumented (internal endpoint — document or explicitly mark internal-only).
- `services/explore/api/openapi.yaml` documents recommender (port 8081) **and** fetcher (`/api/v1/explore/feed/*`, port 8080) paths in one file, blurring the two services' surfaces.

**Remediation (FAST-FOLLOW):** add the missing routes to specs (or annotate internal-only); split or clearly
section the explore spec by service/port.

---

## Versioning (item 5) — PASS

Every public and internal client-facing endpoint is mounted under `/api/v1`; only `/health/*` probes and the
`/` root info endpoint sit outside, which is correct. No action.

Prefixes by service: users `/api/v1/{auth,user}`; content `/api/v1/content`, `/api/v1/internal`;
fetcher `/api/v1/source/rss`; email `/api/v1/source/email`, `/api/v1/internal/source/email`;
explore recommender `/api/v1/explore`; explore fetcher `/api/v1/explore/feed`.

---

## API-freeze checklist (feeds Audit 7)

Before signups open, the following must be true:

- [ ] **F-1** Auth-401, panic-500, and content Content-Type-400 all emit the `pkg/api` error envelope.
- [x] **F-4** Stale aggregate `services/read/api/openapi.yaml` deleted (task_6525); canonical `services/read/content/api/openapi.yaml` already matches the content router (singular `/content`, `check-duplicate`).
- [ ] **F-5** `services/read/fetcher/api/openapi.yaml` paths match the fetcher router (`/source/rss/...`).
- [ ] **F-6** users/explore/fetcher specs document the `{error,message,details,meta}` envelope.
- [ ] **F-3** Pagination param/response shape locked per endpoint and reflected in specs.
- [ ] (Defensive) Unit tests route through the production router so spec drift fails CI (see Audit 5).

Deferrable into beta: **F-2** (health-check JSON), **F-3** unbounded-list pagination *implementation*,
**F-7** (undocumented routes, explore spec split).
