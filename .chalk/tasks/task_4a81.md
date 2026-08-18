---
id: task_4a81
title: [Audit/Tier 1] Rate limiter bypass on the auth endpoints: check-then-act insert lets N concurrent first requests through
type: task
status: open
priority: 1
labels: [quality,security,audit]
blocked_by: [task_f927]
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T10:12:41Z
updated_at: 2026-08-17T10:12:41Z
---
**Source:** Cairn Simplification Audit (read-only pass at HEAD `a6c56a1`, 2026-08-16) — https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. One finding, one branch, one PR. Re-verify before fixing — all file:line references below were confirmed at `a6c56a1`.

**Audit tier:** 1 (correctness & security) | **Verified** | Security-relevant, outside the simplification brief.

## Problem
`pkg/middleware/rate_limit.go`, `(*RateLimiter).allow` (lines 63-96) splits one logical counter across **two locks** with a check-then-act insert:
```go
rl.mu.RLock()
b, exists := rl.requests[key]      // line 65
rl.mu.RUnlock()                    // lock released

if !exists {
    b = &bucket{tokens: rl.limit - 1, lastRefill: time.Now()}
    rl.mu.Lock()
    rl.requests[key] = b           // line 74 — last writer wins
    rl.mu.Unlock()
    return true                    // line 76 — every concurrent first request returns true
}
```
N concurrent first requests for the same key all observe `!exists`, all construct a fresh bucket, all `return true`, and the last writer leaves the bucket at `tokens = limit - 1`. The window's budget is spent once, concurrently, with **no accounting** — a burst of N requests costs one token.

**Where it bites:** `services/users/internal/handlers/router.go:86` applies `sharedmw.RateLimit(authRateLimit, authRateLimitWindow)` (default **10 per minute**, lines 69-75) to the auth group — register, login, refresh, password reset. The comment directly above it, line 84, states the intent:
> `// Rate limiting is applied per IP address to mitigate credential stuffing and enumeration attacks`

That is exactly the attack the race reopens: an attacker who fires concurrently rather than serially gets an effectively unbounded first burst per key.

**Why it survived:** the existing concurrency test in `pkg/middleware/rate_limit_test.go` asserts only that nothing crashes — it never asserts a **count**. And `pkg/middleware` has no CI job at all (see task_f927), so even a correct test would not have run.

## What to do
1. **Failing test first:** N goroutines hit `allow` on the same fresh key concurrently; assert **exactly `limit`** succeed. Must fail on main.
2. Collapse the two critical sections into one — take the write lock once and use a get-or-create under it (or `sync.Map`-style LoadOrStore semantics), so the token decrement for a newly created bucket is accounted under the same lock that inserts it.
3. Keep the fix inside `allow`; no consumer or router change should be needed.

## Done when
- A concurrent test proves at most `limit` requests pass per window for a cold key, and it fails before the fix.

## Notes
- `RateLimitRedis` (`pkg/middleware/rate_limit_redis.go`) is a separate sliding-window implementation for multi-instance deploys and is **not** what the users router wires. Don't conflate them; don't 'fix' this by swapping the router to Redis.
- **Blocked by task_f927** — the test lives in `pkg/middleware`, which no CI job currently reaches, so without that job this fix cannot be ratcheted.
