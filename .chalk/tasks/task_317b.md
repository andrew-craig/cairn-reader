---
id: task_317b
title: Subscription aggregator silently swallows per-source failures (200 with a short list)
type: task
status: open
priority: 2
labels: [quality,read,observability]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-09-05T21:55:18Z
updated_at: 2026-09-05T21:55:18Z
---
**Source:** raised by the PR-gate reviewer on PR #375 (`magpie-reviewer[bot]`, 2026-09-05) as an
"Important, not blocking on its own" observation. Out of scope for that PR — it is pre-existing
code the diff never touched — so it was split out here rather than actioned inline.

## Problem
`SubscriptionAggregatorHandler.ListAllSubscriptions`
(`services/read/content/internal/api/handlers/subscription_aggregator_handler.go:62-88`) fetches
from two sources over HTTP — the Ingest RSS service and the Email Ingest service — and swallows a
failure from either one:

```go
rssSubscriptions, err := h.fetchRSSSubscriptions(r.Context(), userID.String())
if err != nil {
    slog.Error("Failed to fetch RSS subscriptions", "error", err)
    // Don't fail the entire request - just log and continue
    // This allows partial results if one subscription source is down
} else {
    allSubscriptions = append(allSubscriptions, rssSubscriptions...)
}
```

The handler then returns 200 with whatever it collected. A caller cannot distinguish "this user has
no newsletters" from "the email service was unreachable, 404ing, or 403ing", and `total_count`
reports the truncated list as if it were complete. Clients (web, mobile) render a subscription list
that is silently missing a whole source.

Degrading instead of failing is a deliberate choice and the right default — the issue is that the
degradation is invisible above the log line.

## Why it matters
This masked task_d413 (PR #375). In selfhost builds the content service's `/api/v1/internal/*`
catch-all swallowed the email service's internal senders route, so every call to it 404'd. Because
the aggregator logged and carried on, `GET /api/v1/content/user/{id}/subscriptions` kept returning
200 and selfhost email "appeared to work" — the bug survived undetected until it was found by
reading the mount code, not by anything failing. Any future regression in a service-to-service
route feeding this endpoint will hide the same way.

## Options (needs a decision, not yet made)
1. Add a partial-failure field to `dto.ListSubscriptionsResponse` (e.g. `failed_sources: ["email"]`)
   and keep 200. Backwards-compatible for clients that ignore it; lets the apps show a "couldn't
   load newsletters" affordance instead of a silently short list.
2. Return non-200 only when *every* source fails, keeping partial success at 200 with the field
   from option 1.
3. Leave the response shape alone and make the gap detectable by monitoring instead (metric per
   failed source). Weakest — clients still cannot tell, and the project has no metrics stack yet
   (see feature_1a69).

Option 1, optionally with 2, is the most useful and the smallest change. Whichever is chosen,
apply it to both source branches (RSS and email), not just email.

## Done when
- [ ] Decision recorded on the response-shape question above
- [ ] Both the RSS and email branches surface their failure to the caller
- [ ] Handler test covering: one source failing, the other succeeding, and both failing
- [ ] Web/mobile clients either consume the new signal or are confirmed unaffected
