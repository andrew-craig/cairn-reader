---
id: task_ec50
title: [iOS native 05] ReadService and SystemService: content, subscription and feedback endpoints
type: task
status: closed
priority: 1
labels: [ios]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-14T20:26:51Z
---
## Goal
Every backend call the screens need, typed, through `AuthService.authorizedRequest`.

## Source
`apps/mobile/src/services/read.ts`, `system.ts`; tests `read.test.ts`, `system.test.ts`. Endpoint table in `apps/mobile/CLAUDE.md`.

## Plan
- [x] `listUserContents` (cursor pagination, `list`, favourite and status filters), `countUserContents`, `searchUserContents`, `getContentById`.
- [x] ~~`addContentToUser`~~ (folded into `addURL`, see Review), `addURL`, `updateUserContent`, `moveToReads` (`PATCH` with `{ "list": "reads" }`).
- [x] `detectURL`, `discoverFeed`.
- [x] `listAllSubscriptions`, `unsubscribeFromRSSFeed`, `setSourceList(type, key, list)`, `getOrCreateEmailAddress`.
- [x] `getExtractionFeedback`, `submitExtractionFeedback`.
- [x] `SystemService.serverVersion()` from `GET /health/live`.
- [x] `listFeedSubscriptions` exists in the RN service; port it only if a ported screen calls it, otherwise leave it out and say so in the PR.

## Verify
- Ported tests against a `URLProtocol` stub: method, path, query and body for each call; error classification passes through unchanged.

## Review
- `Services/ReadService.swift` (struct over `AuthService`) and `Services/SystemService.swift` (over `HTTPClient`). Reads that RN sent with retry (list, count, search, subscriptions) use `authorizedRequestWithRetry`; everything else `authorizedRequest`. Errors pass through unchanged; `getExtractionFeedback` maps 404 to nil; `detectURL`/`discoverFeed` keep RN's swallow-to-unknown/empty behaviour (cancellation still throws).
- `HTTPRequest.timeout` added for detect (10s) and discover (15s).
- `addContentToUser` not ported: the backend ignores RN's `source_type`, so it is the same `POST .../user/{id}` as `addURL` (and RN mistyped its response). The in-article link save calls `addURL(AddURLRequest(url:))`.
- `listFeedSubscriptions` not ported: no RN screen calls it (deprecated, superseded by `listAllSubscriptions`).
- `UpdateUserContentRequest` leaves out `notes`: nothing in the app sets it.
- RN's `total_count` on list/search results is dropped: the backend sends no total (`countUserContents` gives one).
- Verified: `xcodebuild test` on iPhone 17 simulator (iOS 26), warnings as errors; new suites in `CairnReaderTests/ReadServiceTests.swift`.
