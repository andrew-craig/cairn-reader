---
id: task_963d
title: [P2-C8] Feed-subscribe client unmarshals the wrong response shape → app gets empty data with HTTP 201
type: task
status: closed
priority: 1
labels: [quality,wave2,contracts]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:46:24Z
updated_at: 2026-08-15T10:45:21Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** P2-C8 | **Wave 2** | **Recipe:** R8 (strategy §2.5) | **Test level:** real server handler via httptest, real client pointed at it
**Touches:** services/read/content/internal/service/ingest_rss_client.go, user_content_handler.go

## Problem
`ingest_rss_client.go:144-151` unmarshals the subscribe response into a **nested** `{subscription{…}, feed{…}}` struct, but ingest-rss actually returns a **flat** DTO (`subscription_id` / `feed_id` / `feed_url` / `feed_title` / …) wrapped in the `pkg/api.WriteSuccess` `{data,meta}` envelope, which the client never unwraps here. `json.Unmarshal` does not error on missing keys, so it "succeeds" with a zero-valued struct that propagates through `user_content_handler.go:366-380` to the app: the feed *is* created, but the client receives `feed_id:""`, `title:""`, `subscribed_at:0001-01-01` with a real 201 and no error anywhere.

The sibling `ListUserSubscriptions` in the same file unwraps correctly — this is an isolated regression. Classic bug that unit tests miss because nothing fails.

## What to do
1. Reproduce at the seam: run the real ingest-rss server handler via `httptest`, point the real client at it, assert the decoded struct is **fully populated**. This must fail on current main.
2. Fix the client to unwrap the `pkg/api` `{data,meta}` envelope, matching `ListUserSubscriptions`.
3. Note the hand-maintained response struct as a Wave-4/5 candidate for the shared-type pattern (`pkg/models.Article` is the reference) — do **not** restructure in this bugfix PR.

## Done when
- Seam test fails before the fix and passes after; the app receives a populated subscription on 201.

## Review

Re-verified on `main`: `ingest_rss_client.go`'s `FeedSubscriptionResponse` was still the nested
`{subscription{...}, feed{...}}` shape, `SubscribeUserToFeed` still unmarshalled the raw body into
it directly (no `{data,meta}` unwrap), and the real server handler
(`fetcher/internal/api/handlers/subscription_handler.go:88-91`) still returns the flat
`dto.SubscribeFeedResponse` (`subscription_id`/`feed_id`/`feed_url`/`feed_title`/`is_new_feed`/
`subscribed_at`) via `api.WriteSuccess` — all exactly as described.

**Fix:** `FeedSubscriptionResponse` in `ingest_rss_client.go` is now the flat shape (matching
fetcher's real DTO field names/tags), and `SubscribeUserToFeed` decodes into a `{Data
FeedSubscriptionResponse}` wrapper, mirroring `ListUserSubscriptions`'s existing unwrap. Updated
the one caller, `user_content_handler.go`'s `handleFeedSubmission`: `subscription.Feed.ID` →
`subscription.FeedID`, `subscription.Subscription.*` → the flat fields, and
`subscription.Subscription.UserID` (a field that was never present on the real wire response,
even before this bug — the caller already has the authenticated `userID` in scope) →
`userID.String()`.

**Test** (`content/internal/service/ingest_rss_client_test.go`,
`TestSubscribeUserToFeed_UnwrapsRealServerEnvelope`): serves a real `pkg/api.WriteSuccess`
envelope wrapping the flat DTO shape (fetcher's own `dto` package is under its `internal/` tree
and can't be imported cross-service, so the JSON shape is reproduced locally — the same pattern
already used by `fetcher/internal/client/content_service_client_test.go`'s `writeEnvelope`
helper for the reverse direction) and points the real `IngestRSSClient` at it via `httptest`.
Verified the failure mode by hand with a throwaway program decoding a captured real response into
the old nested struct: every field came back zero-valued (`FeedID=""`, `Title=""`,
`SubscribedAt=0001-01-01`) despite fully-populated input, confirming the finding. Reverted the
client fix (`git stash`) and reran the new test against the old struct: fails to compile against
the old field names, i.e. the old code has no way to expose the real data at all. Restored the fix:
test passes.

**Verification:** `gofmt -l .` clean, `go vet ./...` clean, `golangci-lint run ./content/...` → 0
issues, `go test -race -count=1 ./content/...` all green. No OpenAPI/CLAUDE.md changes needed —
the content service's public `AddFeedResponse`/`FeedSubscriptionDTO` wire shape is unchanged, only
the internal service-to-service decode was wrong. Per the recipe, left the hand-maintained struct
as-is (noted here as a Wave-4/5 shared-type candidate) rather than restructuring in this bugfix PR.

