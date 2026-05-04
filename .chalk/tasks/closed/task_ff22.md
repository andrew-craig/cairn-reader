---
id: task_ff22
title: Implement true cursor-based pagination in Read Content Service
type: task
status: closed
priority: 2
labels: []
blocked_by: []
parent: 
remote_task_url: null
created_at: 2026-04-08T22:32:10Z
updated_at: 2026-05-04T08:15:22Z
---

## Description

The Read Content Service (`services/read/content`) uses offset-based pagination but labels it as cursor-based.

**Current behaviour** (`user_content_handler.go:150-153`):
- Accepts `limit` + `offset` query params
- Returns a `cursor` field containing the next integer offset (e.g. `"20"`, `"40"`)
- This is purely offset-based; the "cursor" is just the next offset serialised as a string

**Required behaviour** (per `docs/detailed_requirements/read_service_requirements.md`):
- Cursor-based pagination using `added_at` timestamp (+ `id` as tiebreaker) as the opaque cursor
- Client passes `cursor=<opaque_token>` instead of `offset=<int>`
- Stable across inserts/deletes — offset pagination drifts when items are added or removed mid-list

**Scope:**
- `GET /api/v1/users/:user_id/contents` list endpoint
- `GET /api/v1/users/:user_id/contents/search` search endpoint
- Update repository queries to use `WHERE added_at < :cursor_time OR (added_at = :cursor_time AND id < :cursor_id)` keyset pagination
- Encode cursor as base64 JSON `{"t":"<RFC3339>","id":"<uuid>"}` so it is opaque to callers
- Remove `offset` and `total` from response (not meaningful for cursor pagination); keep `has_more`, `limit`, `next_cursor`
