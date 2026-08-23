---
id: task_fe72
title: [Audit/Tier 1] SSRF guard wired on url_detector but missing on the content processor that fetches the body
type: task
status: closed
priority: 1
labels: [quality,security,audit]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T10:13:13Z
updated_at: 2026-08-23T03:42:31Z
---
**Source:** Cairn Simplification Audit (read-only pass at HEAD `a6c56a1`, 2026-08-16) — https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. One finding, one branch, one PR. Re-verify before fixing — all file:line references below were confirmed at `a6c56a1`.

**Audit tier:** 1 (correctness & security) | **Verified** | Security-relevant, outside the simplification brief.

## Problem
Two clients on the **same authenticated 'add link' flow**; only one is guarded.

`services/read/content/internal/service/url_detector.go:55-68` — guarded, deliberately and with a comment explaining why:
```go
httpClient: &http.Client{
    Timeout: 10 * time.Second,
    // No Proxy is set (deliberately, not an oversight): honoring
    // HTTP(S)_PROXY would send the request to the proxy without
    // ever invoking DialContext on the target host, bypassing the
    // SSRF guard entirely.
    Transport: &http.Transport{
        DialContext: fetch.DialContext,
    },
},
```

`services/read/content/internal/processor/content.go:29-43` (`NewContentProcessor`) — **no Transport at all**, so `http.DefaultTransport`, so no guard:
```go
httpClient: &http.Client{
    Timeout: FetchTimeout,
    CheckRedirect: func(req *http.Request, via []*http.Request) error { ... },
},
```

The detector merely *probes* the URL; the processor is the one that **fetches the body**. Both are reachable from the same user-supplied-URL path, so the guard is bypassed by the component that does the more dangerous work.

## Root cause (representation gap — fix this, not just the symptom)
The shared fetch helper hardcodes its own client with **no injection point**. Any caller needing conditional-GET behaviour must fork the function, and silently loses the guarded dialer along with it. That is why the gap exists here rather than being a one-off oversight — and why it recurred across several call sites historically.

## What to do
1. **Failing test first:** point the processor at a loopback/private address and assert the fetch is refused (`fetch: blocked address ...`). Must fail on main.
2. Give the shared fetch helper a transport/dialer injection point, then build the processor's client through it — don't just paste `DialContext: fetch.DialContext` into `content.go` and call it done, or the next fork reintroduces the hole.
3. Audit for any other client constructed without a Transport on a user-URL path while you are here (report, don't necessarily fix in the same PR).

## Done when
- The processor's fetch is guarded, proven by a test that fails on main, and the shared helper accepts an injected transport so a conditional-GET caller no longer has to fork.

## Related
- **task_dbca** ([Fetch dedup] collapse the 4+ HTTP fetch+size-cap copies onto pkg/rss/fetch) is the consolidation that removes the whole class. This task is the specific live hole; task_dbca is the structural fix. Landing this one first is fine — but do the injection point here so task_dbca has something to consolidate onto.
- bug_96d7 item 2 — the SSRF guard vs `httptest` conflict in explore/fetcher tests. Same guard, different problem; don't weaken the guard to satisfy either.

## Review

Re-verified on current `main` (`a6c56a1`+): both clients still exist exactly as described; `NewContentProcessor`'s `httpClient` had no `Transport`, so `http.DefaultTransport` was used and the guard was bypassed.

**Failing test first:** added `TestContentProcessor_ProcessURL_BlocksInternalAddress` in `content_test.go`, pointing `ProcessURL` at `http://127.0.0.1:1/`. Confirmed it fails on unpatched code (`git stash` the fix, re-run) with `dial tcp 127.0.0.1:1: connect: connection refused` — i.e. the client actually attempted the loopback dial instead of the guard refusing it.

**Fix (injection point, not a paste):**
- `pkg/rss/fetch/fetch.go`: added `NewTransport() *http.Transport`, returning an `*http.Transport` wired to the guarded `DialContext` plus the existing connection-pooling defaults. `sharedClient` (used by `Fetch()`) now builds its transport through this function too, so there's one construction site instead of two.
- `services/read/content/internal/processor/content.go`: `NewContentProcessor` now sets `Transport: fetch.NewTransport()` on its `httpClient`, keeping its own `CheckRedirect`/timeout (no conditional-GET headers needed here, so it doesn't call `Fetch()` directly).

Test now passes; full `errors.Is(err, fetch.ErrBlockedAddress)` chain holds through `net/url.Error`'s `Unwrap()` and the two `%w` wraps in `content.go`.

**Verification:** `gofmt -l`, `go vet ./...`, `golangci-lint run` clean on both `services/read` and `pkg/rss/fetch`; full `go test ./content/...` (services/read module) and `go test ./fetch/...` (pkg/rss module) green.

**Audit for other unguarded clients on user-URL paths (reporting, not fixing here):** `services/read/fetcher/internal/processor/item_processor.go`, `update_detector.go`, `fetcher/feed_fetcher.go`, and `service/feed_service.go` all construct `&http.Client{...}` with no `Transport`/`DialContext` on feed/article-URL fetch paths — same class of gap, already tracked under **task_dbca** (Fetch dedup). `NewTransport()` added here gives that consolidation something to land on.

**Not touched:** `url_detector.go` was already correctly guarded; left as-is (surgical diff, no behavior change needed there).
