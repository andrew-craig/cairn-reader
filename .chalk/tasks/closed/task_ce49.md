---
id: task_ce49
title: [iOS native 03] Models, errors, server URL config and HTTP client
type: task
status: closed
priority: 1
labels: [ios]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-14T10:08:43Z
---
## Goal
The typed foundation every service uses: backend DTOs, the `Article` domain model, the two error types offline logic keys off, and one place that performs requests.

## Source
`apps/shared/src/types/*`, `apps/shared/src/utils/{errors,http,date}.ts`, `apps/shared/src/config/api.ts`, `apps/mobile/src/config/storage.ts`, the transforms at the bottom of `apps/mobile/src/services/read.ts`, and the OpenAPI specs under `services/*/api/openapi.yaml` (source of truth for field names).

## Plan
- [x] `Codable` DTOs for user-content list/detail, subscriptions, detect/discover, add-URL, extraction feedback, auth responses, user.
- [x] `Article` (struct, `Identifiable`, `Hashable`, `Sendable`) and `ContentList` (`feed` | `reads`), with the transforms from `transformToArticle` / `transformDetailToArticle` (`word_count / 200` to reading time, `status == "completed"` to `isRead`, first image URL, `list` carried through).
- [x] `NetworkError` (unreachable, timeout, undecodable body) and `HttpError(status:)` (definitive non-2xx). Preserve the distinction exactly; the outbox depends on it.
- [x] `ServerConfig`: default `https://cairn.seatrain.net`, override persisted in `UserDefaults`; changing it clears the session (hook used by the auth task).
- [x] `HTTPClient` over `URLSession` async/await: builds requests against the server URL, maps transport failures to `NetworkError` and non-2xx to `HttpError`, decodes JSON. Takes a `URLSession` so tests inject a `URLProtocol` stub.
- [x] Dates: use `Date.FormatStyle` / `RelativeDateTimeFormatter` instead of porting `formatDate`, `formatPublishedDate`, `pluralize`; keep the same visible wording where the system formatters allow, and note differences in the PR.

## Verify
- Tests: DTO decoding from fixture JSON copied from the OpenAPI examples / RN test fixtures; both transforms; error mapping for timeout, 404, 500, bad JSON.

## Review (2026-11-14)
Built and tested locally (Xcode 26.6, iPhone 17 simulator, warnings as errors): 47 test cases pass.
- `Models/ReadModels.swift`, `Models/UserModels.swift`: `Codable` wire types named after the OpenAPI schemas. One `ContentResponse` covers summary and detail (`cleanedHtml` is nil in list/search results), so one `Article.init(_:)` replaces both RN transforms. `AddURLResponse` is an enum on `type`. Auth request bodies and `UpdateUserContentRequest` are left to the tasks that send them (04, 05).
- Responses are enveloped on the wire (`{data, meta}`, `{data, pagination, meta}`), which the TS types hide; `Envelope<T>` and `Page<T>` model that. Dates decode as `Date` (RFC 3339 with or without fractional seconds, as Go writes them); `UnifiedSubscription.subscribed_at` stays a string because the backend passes it through as one.
- `Article` drops `tags` (always empty) and `notes`/`scrollFraction` (local store fields, added by task 06 if needed).
- `NetworkError` (`unreachable`, `timeout`, `undecodableBody`) and `HttpError(status:message:)`. URL-session cancellation surfaces as `CancellationError`, not a `NetworkError`.
- `ServerConfig`: reads `UserDefaults` on each access (no cached copy). `setURL(_:)` returns whether the server changed and is not `@discardableResult`, so the caller (AuthService, task 04) must handle the session clear. PrivacyInfo now declares `UserDefaults` (`CA92.1`).
- Dates: `Date.relativeDayLabel` keeps the RN buckets with `RelativeDateTimeFormatter` wording. Differences: "Last week" / "Last month" / "Last year" instead of "1 week ago" etc. `pluralize` is not ported; SwiftUI's automatic grammar agreement (`^[\(n) subscription](inflect: true)`) replaces it on the You screen.
