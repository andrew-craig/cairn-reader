---
id: task_997e
title: [Audit/Tier 2] Delete the dead duplicate models package services/explore/pkg/models (98 lines, zero importers)
type: task
status: closed
priority: 2
labels: [quality,deletion,audit]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T10:14:03Z
updated_at: 2026-08-29T23:26:08Z
---
**Source:** Cairn Simplification Audit (read-only pass at HEAD `a6c56a1`, 2026-08-16) — https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. One finding, one branch, one PR. Re-verify before fixing — all file:line references below were confirmed at `a6c56a1`.

**Audit tier:** 2 (deletions — near-zero risk, compiler-verified) | **The audit's #1 recommended first slice.**

## Problem
`services/explore/pkg/models` is a **fully dead duplicate** of the live root `pkg/models`:
```
services/explore/pkg/models/article.go               31
services/explore/pkg/models/user.go                  10
services/explore/pkg/models/vote.go                  12
services/explore/pkg/models/recommendation_event.go  12
services/explore/pkg/models/doc.go                    4
services/explore/pkg/models/feed.go                  29
                                             total   98 lines
```
**Zero importers** — `grep -rn 'explore/pkg/models' --include=*.go .` returns nothing. Every real consumer imports the root package `github.com/andrew-craig/cairn-reader/pkg/models` (services/explore/fetcher/internal/{fetcher,client,db,api}, services/explore/recommender/…).

## Why it is a trap, not just clutter
Its `feed.go` is **stale**: 29 lines against the live `pkg/models/feed.go`'s 31, and the two missing fields are exactly the ones the conditional-GET path depends on:
```go
pkg/models/feed.go:14   ETag         string `json:"etag,omitempty"`
pkg/models/feed.go:15   LastModified string `json:"last_modified,omitempty"`
```
Neither appears anywhere in `services/explore/pkg/models`. A future edit that imports the nearer-looking package would compile and then silently break conditional GET (`If-None-Match` / `If-Modified-Since`) — the live path reads these in `services/explore/fetcher/internal/db/feed_repository.go:64,67` and persists them via `UpdateFetchResult(ctx, feedID, success, etag, lastModified)`.

## What to do
1. Re-confirm zero importers (`grep` + a clean `go build ./...` in services/explore).
2. Delete the directory.
3. Nothing else. No test changes should be needed — no test imports it either.

## Done when
- The directory is gone, `services/explore` builds, and its test suite is unchanged and green.

## Why this one first
The audit picks it as the best opening slice: 98 lines, zero importers, compiler-verified, no test impact, and it removes the stale-`Feed` trap described above.
