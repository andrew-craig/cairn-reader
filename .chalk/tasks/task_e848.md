---
id: task_e848
title: [read/fetcher] SSRF: all three outbound HTTP clients unguarded (feed_service, feed_fetcher, update_detector)
type: task
status: in_progress
priority: 1
labels: []
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-29T23:49:04Z
updated_at: 2026-09-18T21:32:59Z
---


## Detail (found 2026-08-30 while scoping task_dbca)

`pkg/rss/fetch` carries the SSRF guarded dialer (`fetch.DialContext` / `fetch.NewTransport()`).
`services/read/content` was brought onto it (url_detector.go; processor/content.go via task_fe72 #340).
**`services/read/fetcher` was not** — every one of its outbound HTTP clients uses the default
transport, so none resolve through the guard:

| Constructor | Line | Client | Fetches | Reached from |
|---|---|---|---|---|
| `service.NewFeedService` | `internal/service/feed_service.go:78` | `&http.Client{Timeout: FeedFetchTimeout}` | user-supplied `feedURL` during Subscribe validation | `POST /api/v1/source/rss/user/{id}/subscription` (authed) |
| `fetcher.NewFeedFetcher` | `internal/fetcher/feed_fetcher.go:56-74` | hand-rolled `&http.Transport{}` (TLS/idle knobs, **no DialContext**) → `parser.ParseFromURL` | every subscribed feed URL | timer (ingest_rss_worker) |
| `processor.NewUpdateDetector` | `internal/processor/update_detector.go:56` | `&http.Client{Timeout: config.ContentFetchTimeout}` → `fetcher.NewConditionalFetcher` | `item.ItemURL` from feed content | timer |

All three fetch attacker-influenceable URLs (subscribe with `http://169.254.169.254/...`, or a feed
whose `<link>` points at an internal host). Same class as task_fe72; that task fixed content, this
is the fetcher half.

**Minimal fix** (independent of the task_dbca consolidation): give each client
`Transport: fetch.NewTransport()` (feed_service, update_detector) and add `DialContext: fetch.DialContext`
to feed_fetcher's existing transport. Then migrate the affected unit tests
(`conditional_fetcher_test.go`, `feed_fetcher_test.go`, `parser_test.go`, `feed_service_test.go`,
`item_processor_test.go`) to `fetchtest.AllowLoopback` — they use `httptest` (127.0.0.1) and will
otherwise fail with "blocked address", exactly as explore/fetcher did in bug_96d7.

Priority: this is Tier-1 security (anonymous-ish: any registered user). Should land before task_dbca.

## Resolution

Applied the minimal fix to all three listed constructors, plus a fourth unguarded client found
during implementation: `processor.NewItemProcessor` (`internal/processor/item_processor.go:58`)
also fetches `item.ItemURL` unconditionally (before the subscriber check in `processItem`) using a
bare `&http.Client{Timeout: ...}` — same vulnerability class, reached from the same
`ingest_rss_worker` timer path as `update_detector`. The original audit's table only covered three;
this one was missed. Fixed with the same `Transport: fetch.NewTransport()` pattern.

Changes:
- `feed_service.go`: `Transport: fetch.NewTransport()` on the Subscribe-validation client.
- `feed_fetcher.go`: added `DialContext: fetch.DialContext` to the existing hand-rolled transport.
- `update_detector.go`: `Transport: fetch.NewTransport()` on the conditional-fetch client.
- `item_processor.go`: `Transport: fetch.NewTransport()` on the first-fetch client (new finding, not in original table).

Test migration: of the five files named in the task, only `feed_fetcher_test.go` and
`item_processor_test.go` actually exercise the guarded client against an `httptest` (127.0.0.1)
server — both migrated to `fetchtest.AllowLoopback`. The other three
(`conditional_fetcher_test.go`, `parser_test.go`, `feed_service_test.go`) construct their own
unguarded `http.Client`/pass `server.Client()` directly, or never reach a network call at all, so
they were unaffected and left untouched (verified by running them before making any test changes).

Verified: `go build ./...`, `go vet ./...`, and `go test ./...` all pass in `services/read`;
`gofmt -l .` clean.
