---
id: task_ce49
title: [iOS native 03] Models, errors, server URL config and HTTP client
type: task
status: open
priority: 1
labels: [ios]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-14T07:15:57Z
---
## Goal
The typed foundation every service uses: backend DTOs, the `Article` domain model, the two error types offline logic keys off, and one place that performs requests.

## Source
`apps/shared/src/types/*`, `apps/shared/src/utils/{errors,http,date}.ts`, `apps/shared/src/config/api.ts`, `apps/mobile/src/config/storage.ts`, the transforms at the bottom of `apps/mobile/src/services/read.ts`, and the OpenAPI specs under `services/*/api/openapi.yaml` (source of truth for field names).

## Plan
- [ ] `Codable` DTOs for user-content list/detail, subscriptions, detect/discover, add-URL, extraction feedback, auth responses, user.
- [ ] `Article` (struct, `Identifiable`, `Hashable`, `Sendable`) and `ContentList` (`feed` | `reads`), with the transforms from `transformToArticle` / `transformDetailToArticle` (`word_count / 200` to reading time, `status == "completed"` to `isRead`, first image URL, `list` carried through).
- [ ] `NetworkError` (unreachable, timeout, undecodable body) and `HttpError(status:)` (definitive non-2xx). Preserve the distinction exactly; the outbox depends on it.
- [ ] `ServerConfig`: default `https://cairn.seatrain.net`, override persisted in `UserDefaults`; changing it clears the session (hook used by the auth task).
- [ ] `HTTPClient` over `URLSession` async/await: builds requests against the server URL, maps transport failures to `NetworkError` and non-2xx to `HttpError`, decodes JSON. Takes a `URLSession` so tests inject a `URLProtocol` stub.
- [ ] Dates: use `Date.FormatStyle` / `RelativeDateTimeFormatter` instead of porting `formatDate`, `formatPublishedDate`, `pluralize`; keep the same visible wording where the system formatters allow, and note differences in the PR.

## Verify
- Tests: DTO decoding from fixture JSON copied from the OpenAPI examples / RN test fixtures; both transforms; error mapping for timeout, 404, 500, bad JSON.
