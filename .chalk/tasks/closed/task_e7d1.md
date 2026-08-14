---
id: task_e7d1
title: [P2-C6] Bulk content insert PK-collides on any mixed new+seen batch → whole RSS batch fails
type: task
status: closed
priority: 1
labels: [quality,wave2,database]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:46:24Z
updated_at: 2026-08-14T12:16:02Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** P2-C6 | **Wave 2** | **Recipe:** R7 (strategy §2.5) | **Test level:** `//go:build integration` against real Postgres with real migrations
**Touches:** services/read/content/internal/service/content_service.go, internal/repository/content.go

## Problem
`BulkCreateFromHTML` (`content_service.go:270-321`) puts *both* already-existing `Content` rows — which already carry a real DB `id` — and genuinely-new rows into one slice, then hands it to `BulkCreate` (`repository/content.go:530-586`), which issues a single multi-row `INSERT` with **no `ON CONFLICT`** and only assigns a UUID when `id == Nil`. The pre-existing row's real `id` violates the `contents` primary key, the whole statement rolls back, and the entire `POST /api/v1/content/bulk` fails — including the valid new items.

This is the production RSS ingestion path (`bulk_handler.go:67`), and re-polled feeds *routinely* contain a mix of seen + new items, so it fires under normal operation. The existing test mocks the repository and masks it entirely.

## What to do
1. Integration test: insert a batch mixing one already-stored item and one new item; assert both survive and the new one is created. It must fail on current main for the PK-collision reason.
2. Fix: filter pre-existing rows before `BulkCreate` (preferred — explicit), or `ON CONFLICT (id) DO NOTHING`.
3. testcontainers is already used in explore — reuse `internal/testutil` rather than inventing new harness code.

## Done when
- Integration test proves the mixed batch survives; the mocked unit test is not the only coverage.


## Review

**Re-verified on main:** confirmed — `BulkCreateFromHTML` appended both pre-existing rows (from `GetByContentHashAndFeedID`, real DB `id`) and new rows into the same `contents` slice passed to `BulkCreate`, which issues one multi-row `INSERT` with no `ON CONFLICT`.

**Failing test first:** added `internal/service/content_bulk_integration_test.go` (`//go:build integration`, real Postgres via `internal/testutil`). It stores one item for real, then re-delivers it mixed with a brand-new item in one `BulkCreateFromHTML` call — reproducing a re-polled RSS feed. Confirmed it fails on pre-fix code with `pq: duplicate key value violates unique constraint "contents_pkey"`.

**Fix:** `BulkCreateFromHTML` now tracks a separate `newContents` slice and only passes genuinely-new items (no pre-existing `id`) to `BulkCreate`. Pre-existing items are still included in the returned `contents` slice so callers get the full batch back. This is the "filter pre-existing rows before `BulkCreate`" option from the recipe.

**Also strengthened** the existing mocked unit test (`TestBulkCreateFromHTML_WithDuplicates`) which previously asserted `BulkCreate` was called with `len(contents) == 2` (i.e. it accepted the bug) — it now asserts `BulkCreate` receives only the new item.

**Verification:**
- `gofmt -l .` clean, `go vet ./...` clean
- `go test -race -count=1 ./...` — all green
- `go test -tags=integration -count=1 ./...` — green except two pre-existing, unrelated `url_detector_integration_test.go` tests that hit live external URLs (NASA RSS feed, example.com) and fail in this sandbox for lack of outbound network access; not touched by this change and not part of the finding.

No API/route boundary changed, so no `openapi.yaml`/`CLAUDE.md` route-table updates needed.
