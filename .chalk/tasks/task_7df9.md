---
id: task_7df9
title: Content service: per-source routing to Feed or Reads
type: task
status: open
priority: 1
labels: [read,backend]
blocked_by: []
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-08T11:32:05Z
updated_at: 2026-10-08T21:34:50Z
---
See epic_6e4d design section.
- [ ] Migration 000005: user_contents.list ('feed'|'reads', NOT NULL, backfill 'reads'), index (user_id, list, added_at DESC); source_routes(user_id, source_type, source_key, list, PK on first three) → up/down integration test
- [ ] Email worker: typed payload carries sender_id; content service stores it on contents (new source_sender_id column) → worker test + bulk integration test
- [ ] BulkAddToUsersInternal resolves list per user from source_routes; no route → reads → integration tests: RSS→feed, RSS→reads, email sender→feed, no route→reads
- [ ] Subscribe (AddContentToUser feed branch) accepts list (default reads) and writes the route; RSS unsubscribe deletes it → handler tests
- [ ] PUT /content/user/{id}/subscriptions/{type}/{key}/list — updates route only; new items only (owner decision) → handler test
- [ ] list filter on ListUserContents / Search / Count; unified subscriptions response includes each source's list → repo + handler tests
- [ ] PATCH /content/user/{id}/{content_id} accepts list (Save to Reads) → handler test
- [ ] openapi.yaml updated
