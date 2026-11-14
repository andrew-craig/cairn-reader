---
id: task_ec50
title: [iOS native 05] ReadService and SystemService: content, subscription and feedback endpoints
type: task
status: open
priority: 1
labels: [ios]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-14T10:42:35Z
---
## Goal
Every backend call the screens need, typed, through `AuthService.authorizedRequest`.

## Source
`apps/mobile/src/services/read.ts`, `system.ts`; tests `read.test.ts`, `system.test.ts`. Endpoint table in `apps/mobile/CLAUDE.md`.

## Plan
- [ ] `listUserContents` (cursor pagination, `list`, favourite and status filters), `countUserContents`, `searchUserContents`, `getContentById`.
- [ ] `addContentToUser`, `addURL`, `updateUserContent`, `moveToReads` (`PATCH` with `{ "list": "reads" }`).
- [ ] `detectURL`, `discoverFeed`.
- [ ] `listAllSubscriptions`, `unsubscribeFromRSSFeed`, `setSourceList(type, key, list)`, `getOrCreateEmailAddress`.
- [ ] `getExtractionFeedback`, `submitExtractionFeedback`.
- [ ] `SystemService.serverVersion()` from `GET /health/live`.
- [ ] `listFeedSubscriptions` exists in the RN service; port it only if a ported screen calls it, otherwise leave it out and say so in the PR.

## Verify
- Ported tests against a `URLProtocol` stub: method, path, query and body for each call; error classification passes through unchanged.
