---
id: task_b174
title: [P2-C7/H8] Recommender upsert conflicts on content-hash id, not just link → batch dropped every poll
type: task
status: open
priority: 1
labels: [quality,wave2,database]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:46:24Z
updated_at: 2026-08-09T06:46:24Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** P2-C7 / H8 | **Wave 2** | **Recipe:** R7 (strategy §2.5) | **Test level:** `//go:build integration` against real Postgres
**Touches:** services/explore/recommender/internal/db/article_repository.go

## Problem
`article_repository.go:77-127` upserts via `ON CONFLICT (link) DO UPDATE`, but `id` is a *separate* primary key set to `SHA256(cleaned_content)` (per `fetcher/migrations/000002`). When two different links carry the same content — syndicated/re-published articles, the exact case the content-hash id was designed to dedupe — the new row's `link` doesn't conflict but its `id` does, giving an **unhandled** PK violation. Because it is one multi-row `INSERT`, the whole batch fails and silently drops every other new article in that fetch cycle, and it **recurs on every subsequent poll** until the feed auto-disables.

An empty or duplicate `<link>` in a batch also trips "cannot affect row a second time". This is the mechanism behind Part 1's H8.

## What to do
1. Integration test: two articles with different links and the same content hash; assert the batch survives and the dedup behavior is deliberate. Also cover an empty/duplicate `link` in the same batch.
2. Fix: arm the `id` constraint (`ON CONFLICT (id) ...`) and decide explicitly what "same content, different link" should do — state the choice in the PR.

## Done when
- Integration test fails before the fix for the PK-violation reason and passes after; no article is silently dropped.

