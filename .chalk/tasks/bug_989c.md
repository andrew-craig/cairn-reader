---
id: bug_989c
title: Content: subscribe response list ignores existing route; openapi missing RSS unsubscribe
type: bug
status: in_progress
priority: 2
labels: []
blocked_by: []
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-13T20:20:48Z
updated_at: 2026-10-13T20:22:15Z
---


## Plan
- [x] Failing test: subscribe with `list` omitted and an existing `feed` route reports `reads` (TestAddContentToUser_FeedSubscription_OmittedListReportsExistingRoute)
- [x] Fix: SourceRouteRepository.Get; handler resolves the list via resolveFeedList (explicit list -> upsert, omitted -> existing route else reads, never writes)
- [x] openapi: add DELETE /api/v1/content/user/{user_id}/subscriptions/rss/{feed_id} (also removes the Feed/Reads route)
- [x] go build / vet / test in services/read/content green; no openapi lint target exists

## Review
Root cause: handleFeedSubmission defaulted the response list to reads without consulting source_routes when `list` was omitted. Fix reads the route through the repository; behaviour with an explicit list is unchanged. The repository Get integration assertion needs a database and was skipped here.
