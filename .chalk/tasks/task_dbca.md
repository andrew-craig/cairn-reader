---
id: task_dbca
title: [Fetch dedup] Collapse the 4+ HTTP fetch+size-cap copies onto pkg/rss/fetch
type: task
status: open
priority: 2
labels: [quality,wave4,consolidation]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:53:56Z
updated_at: 2026-08-13T11:10:21Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** Theme 3 (fetch duplication) | **Wave 4** | **Recipe:** R11 (strategy §2.5)
**Touches:** services/read/content/internal/service/url_detector.go, processor/content.go, services/read/fetcher feed_service.go, conditional_fetcher.go, item_processor.go
**Blocked by:** the Wave 1 SSRF-dialer task — `pkg/rss/fetch` must already carry the guard, or consolidating onto it spreads nothing.

## Problem
The HTTP fetch + size-cap + User-Agent logic is reimplemented in 4+ independent copies across read/content and read/fetcher — which is **why** the SSRF gap existed in four places at once. `pkg/rss/fetch` is the canonical implementation (redirect limit + `io.LimitReader`) and, after Wave 1, the only one with the SSRF guard.

## What to do (consolidation sequence — deleting code, so order matters)
1. **Characterize first:** diff each copy against `pkg/rss/fetch` and write tests on the canonical implementation covering the behaviors each copy's callers rely on (timeouts, size caps, User-Agent, conditional-request headers). Where they diverge, pick one behavior **explicitly** and say so in the PR.
2. Repoint callers one call-site group at a time.
3. Delete each duplicate **in the same PR** as its last repoint — a surviving duplicate defeats the purpose.

## Done when
- Every HTTP fetch on the article/feed path goes through `pkg/rss/fetch`; no copy remains.

