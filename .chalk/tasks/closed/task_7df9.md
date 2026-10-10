---
id: task_7df9
title: Content service: per-source routing to Feed or Reads
type: task
status: closed
priority: 1
labels: [read,backend]
blocked_by: []
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-08T11:32:05Z
updated_at: 2026-10-09T11:00:47Z
---
See epic_6e4d design section.
- [x] Migration 000005: user_contents.list ('feed'|'reads', NOT NULL, backfill 'reads'), index (user_id, list, added_at DESC); source_routes(user_id, source_type, source_key, list, PK on first three) → up/down integration test
- [x] Email worker: typed payload carries sender_id; content service stores it on contents (new source_sender_id column) → worker test + bulk integration test
- [x] BulkAddToUsersInternal resolves list per user from source_routes; no route → reads → integration tests: RSS→feed, RSS→reads, email sender→feed, no route→reads
- [x] Subscribe (AddContentToUser feed branch) accepts list (default reads) and writes the route; RSS unsubscribe deletes it → handler tests
- [x] PUT /content/user/{id}/subscriptions/{type}/{key}/list — updates route only; new items only (owner decision) → handler test
- [x] list filter on ListUserContents / Search / Count; unified subscriptions response includes each source's list → repo + handler tests
- [x] PATCH /content/user/{id}/{content_id} accepts list (Save to Reads) → handler test
- [x] openapi.yaml updated

## Review
- `list` is stamped inside `UserContentRepository.BulkCreate`'s INSERT (COALESCE of the matching `source_routes` row, else `reads`), so a route change can't be half-applied mid-delivery. `Create` (direct saves) always writes `reads`.
- Email: `EmailContentPayload.SenderID` → bulk-create `source_sender_id` → `contents.source_sender_id`; the route lookup goes through that column.
- Subscribe writes the route after the fetcher subscribe succeeds (500 if the route write fails; the feed then defaults to Reads and can be fixed via the PUT). Unsubscribe deletes it.
- PATCH `list` moves an item without touching `added_at`, so a "Save to Reads" item keeps its original position in Reads ordering. Flagging for the mobile/web tasks.
- Integration suite run against local Postgres 16; the URL-detector network tests fail in this sandbox on main too.
