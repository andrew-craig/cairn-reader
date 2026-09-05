---
id: task_e848
title: [read/fetcher] SSRF: all three outbound HTTP clients unguarded (feed_service, feed_fetcher, update_detector)
type: task
status: open
priority: 1
labels: []
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-29T23:49:04Z
updated_at: 2026-08-29T23:49:19Z
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
