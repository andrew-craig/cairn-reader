---
id: task_40b2
title: [P3-C2/H6] MaxBytesReader on all JSON endpoints + depth guards on the recursive email HTML walkers
type: task
status: open
priority: 1
labels: [quality,security,wave1,dos]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:43:57Z
updated_at: 2026-08-09T06:43:57Z
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

