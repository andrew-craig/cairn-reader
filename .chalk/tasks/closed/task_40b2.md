---
id: task_40b2
title: [P3-C2/H6] MaxBytesReader on all JSON endpoints + depth guards on the recursive email HTML walkers
type: task
status: closed
priority: 1
labels: [quality,security,wave1,dos]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:43:57Z
updated_at: 2026-08-14T10:17:09Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** P3-C2, H6, uncapped bodies | **Wave 1** | **Recipe:** R5 (strategy §2.5) | **Test level:** oversized body expects 413; programmatically nested HTML
**Touches:** services/read/email/internal/api/handlers/ingest_handler.go, services/read/email/internal/processor/content_extractor.go, .../email_cleaner.go, plus users / read/content / read/fetcher handler groups

## Problem
**P3-C2 (critical, process-crashing):** two findings that combine.

- `services/read/email/internal/api/handlers/ingest_handler.go:27` decodes `IngestEmailRequest` (carrying `HTMLBody`/`TextBody`) via plain `json.NewDecoder` — no `MaxBytesReader`, no field-length limit. Confirms H6.
- `processor/content_extractor.go:84-97` (`htmlToPlainText`'s `walk`) and `email_cleaner.go:58-65` (`walkForRemoval`) are naive recursive DOM walks whose depth is attacker-controlled.

Together: an `HTMLBody` of millions of nested `<div>`s — cheap in bytes because nothing rejects it — stack-overflows, which in Go is a **fatal, unrecoverable runtime error that kills the whole process**, not a per-request panic.

**Systemic gap:** only `explore/recommender` applies `http.MaxBytesReader`. `users`, `read/content`, `read/fetcher` and `read/email` all call `json.NewDecoder(r.Body).Decode(...)` with no cap. Worst case: `POST /api/v1/content` and `/bulk` sit on the **no-auth** route group and decode via `middleware.DecodeJSONBody` (`validation.go:30`) — the 5MB `MaxContentSize` check fires only *after* the full body is in memory, so an anonymous multi-GB body is fully buffered before rejection. Same uncapped pattern in `users` (`auth_handler.go`, `user_handler.go`) and `read/fetcher` (`subscription_handler.go:42,201`, which has no rate limit either). Lower-severity recursion in `content/url_detector.go:229-269` and `:392-411`.

## What to do
1. `http.MaxBytesReader` per handler group, mirroring `explore/recommender`'s limits (10MB / 16KB / 1KB) — tight caps for auth/JSON endpoints, larger for content ingestion. Assert 413 for an oversized body and success just under the cap.
2. Depth guards in both email walkers, and the content `traverse` walkers: a max-depth parameter (100 is generous for real HTML); truncate or reject beyond it.
3. Test with programmatically nested HTML around 10k deep — assert **graceful** handling. Do **not** write a test that actually overflows the stack to prove the old bug.

## Done when
- 413 tests pass on every JSON endpoint group in users / read/content / read/fetcher / read/email; depth-guard tests pass; nothing crashes the test runner.

## Review

**Re-verification (2026-08-14):** Confirmed still live on `main`. `grep -rn MaxBytesReader services/` showed only `explore/recommender`. Note one re-verification finding that changes the depth-guard framing: `golang.org/x/net/html` v0.57.0 (already bumped repo-wide by the earlier R4/#313 task) caps parse-time nesting at 512 open elements and converts what used to be a fatal panic into a normal returned error (`html/parse.go:237-238`, recovered at `parse.go:2225`). So `html.Parse` itself now refuses anything nested deeper than 512 levels before our own walkers ever see it — the literal "millions of nested `<div>`s crash the process" scenario from the finding is already closed by that dependency bump. The recursive walkers (`htmlToPlainText`, `walkForRemoval`, and `url_detector.go`'s two `traverse` funcs) still got depth guards (100, matching the recipe's suggested default) as defense-in-depth / explicit bounding, since that's cheap and matches the recipe, but they're no longer the only thing standing between an attacker and a crash.

**What changed:**
- `http.MaxBytesReader` + 413 handling added to every JSON-decoding handler in `users` (auth_handler.go, user_handler.go), `read/content` (content_handler.go, bulk_handler.go, detection_handler.go, user_content_handler.go), `read/fetcher` (subscription_handler.go), and `read/email` (ingest_handler.go). Caps: 16KB for simple string bodies, 64KB for small id/hash batches (up to 100 items), 10MB for single content/email bodies, 50MB for bulk content (up to 100 items × 5MB).
- Depth guards (max depth 100) added to `htmlToPlainText`'s walk and `walkForRemoval` in `read/email/internal/processor`, and to both `traverse` closures in `read/content/internal/service/url_detector.go`.
- New tests: 413-vs-under-cap tests for every touched handler; depth-guard tests use nesting kept under x/net's 512-node parse cap (300) so they exercise our own walk's bound, not the parser's — per the "don't crash the test runner" instruction.
- `openapi.yaml` updated with `413` responses for every touched endpoint across all four services.

**Unrelated pre-existing issue found during verification (not fixed, out of scope):** `services/users/internal/handlers` has multiple tests (`TestRegister`, `TestUpdateUser`, `TestUpgradeAccount`, etc.) that panic with a nil-pointer dereference and/or get a wrong status code — reproduced identically on unmodified `main` via `git stash`. Two separate causes spotted in passing: (a) some tests read the chi route param as `"id"` while the handlers and `router.go` actually use `"user_id"`, and (b) `services/users/internal/services` tests (`TestAuthService_Login`, `TestAuthService_RefreshAccessToken`) show JWT/token duplication failures suggestive of test-isolation issues. Also confirmed `test/integration` lacking the `//go:build integration` tag makes bare `go test ./...` hang, matching the already-filed `task_8b77`. None of this is touched by this PR; flagging per the strategy's "mention, don't fix" rule for unrelated findings.

**Verification:** `gofmt -l .`, `go vet ./...`, `golangci-lint run ./...` clean on all four modules. New tests pass (`go test -run BodyTooLarge` per service, plus the url_detector/processor depth-guard tests) against a local Postgres for the `users` service (needed for its DB-backed handler test harness).

