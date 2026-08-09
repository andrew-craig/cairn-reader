---
id: task_f0f3
title: [SSRF] Add a guarded dialer to pkg/rss/fetch (blocks loopback/RFC1918/link-local/metadata)
type: task
status: open
priority: 0
labels: [quality,security,wave1,ssrf]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:43:57Z
updated_at: 2026-08-09T06:43:57Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** C1 (SSRF half) | **Wave 1** | **Recipe:** R2 (strategy §2.5) | **Test level:** table-driven adversarial, with a test resolver
**Touches:** pkg/rss/fetch, services/read/content/internal/service/url_detector.go (call site only)

## Problem
The shared fetch path follows any URL the caller supplies — loopback, RFC1918, link-local and cloud-metadata (169.254.169.254) addresses are all reachable. The same unvalidated fetch pattern is copy-pasted in 4+ places: `content/internal/service/url_detector.go:184-344`, `processor/content.go:111-140`, `fetcher/.../feed_service.go:245-303`, `conditional_fetcher.go`, `item_processor.go:212-240` — which is *why* the gap exists in four places at once.

## What to do
1. Implement **one** guarded dialer in `pkg/rss/fetch`. Check **resolved IPs** at `DialContext` time against loopback / RFC1918 / link-local / metadata ranges — checking the hostname before resolution is bypassable via DNS.
2. Table-driven tests: each blocked range; a redirect from a public URL to an internal one; DNS names that resolve to internal IPs (use a test resolver).
3. Wire it into `pkg/rss/fetch` only. The other fetch copies are deleted in the Wave 4 "fetch dedup" task; if a Wave-1 endpoint uses a copy (e.g. `url_detector.go`), point that call site at the shared guard now.

## Done when
- Every blocked range and the redirect-to-internal case are covered by tests that fail before the fix.
- No new fetch helper is introduced — the guard lives in `pkg/rss/fetch`.

