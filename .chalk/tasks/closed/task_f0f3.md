---
id: task_f0f3
title: [SSRF] Add a guarded dialer to pkg/rss/fetch (blocks loopback/RFC1918/link-local/metadata)
type: task
status: closed
priority: 0
labels: [quality,security,wave1,ssrf]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:43:57Z
updated_at: 2026-08-13T11:10:21Z
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

## Review

Re-verified on current `main`: `pkg/rss/fetch.Fetch` dialed any caller-supplied URL with no IP
validation, and `url_detector.go`'s `DetectURL`/`DiscoverFeeds` (behind `/api/v1/content/detect`
and `/discover-feed`, JWT-protected since #311 but reachable by any logged-in user) used its own
unguarded `http.Client`. Confirmed by temporarily removing the new `DialContext` wiring and
re-running the new adversarial tests — they failed with real dial attempts/connection-refused
errors instead of `ErrBlockedAddress`, proving they exercise the guard for the right reason.

**Changes:**
- `pkg/rss/fetch/guard.go` (new): `DialContext`, a `net.Dialer.DialContext`-shaped guarded dialer.
  Resolves the host, rejects the dial if any resolved IP is loopback/private (RFC1918+RFC4193)/
  link-local (covers the 169.254.169.254 cloud-metadata address)/unspecified, then dials the
  exact validated IP — so DNS can't change between the check and the connect. `Resolver` is an
  injectable interface (`*net.Resolver` satisfies it) so tests can exercise DNS-based bypass
  attempts without a real DNS server; `AllowLoopbackForTesting`/`ContextWithResolver`/
  `FakeResolver` are the test-support surface (httptest servers bind to loopback, which the guard
  must otherwise block).
- `pkg/rss/fetch/fetch.go`: `sharedClient`'s `Transport.DialContext` now points at the guard.
- `pkg/rss/fetch/guard_test.go` (new): table-driven blocked-range tests (loopback v4/v6, all three
  RFC1918 blocks, link-local, the metadata IP, IPv6 link-local/unique-local, unspecified),
  redirect-from-public-to-metadata-IP, hostname-resolves-to-internal-IP via a fake resolver,
  mixed-resolution (any blocked IP among several rejects the whole lookup), and a
  does-not-over-block-legitimate-hosts check.
- `pkg/rss/fetch/fetch_test.go`: existing httptest-based tests now wrap their context with
  `fetch.AllowLoopbackForTesting` so they keep exercising real loopback fetches post-guard.
- `services/read/content/internal/service/url_detector.go`: `NewURLDetector`'s `httpClient` now
  uses `fetch.DialContext` as its `Transport.DialContext` — the one Wave-1 call site the task
  scoped in. `processor/content.go`, `fetcher/.../feed_service.go`, `conditional_fetcher.go`, and
  `item_processor.go` are untouched; they're deleted in the Wave 4 fetch-dedup task.
- `services/read/content/internal/service/url_detector_test.go`: added
  `TestNewURLDetector_UsesGuardedDialer` (pointer-compares the wired `DialContext` against
  `fetch.DialContext`, so it fails deterministically pre-fix without depending on network
  reachability), and wrapped the existing httptest-based tests' contexts the same way as
  `fetch_test.go`.

**Verified:** `gofmt -l .`, `go vet ./...`, `golangci-lint run ./...` (0 issues), and
`go test -race ./...` all green in both `pkg/rss` and `services/read`.

