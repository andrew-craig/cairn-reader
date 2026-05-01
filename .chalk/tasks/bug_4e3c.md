---
id: bug_4e3c
title: fix(read/fetcher): outbox repository casts to nonexistent delivery_status enum
type: bug
status: open
priority: 1
labels: [outbox,read-fetcher,sql]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-05-01T09:37:43Z
updated_at: 2026-05-01T09:37:43Z
---
The outbox repository at services/read/fetcher/internal/repository/outbox.go:217-218 uses SQL casts like 'failed'::delivery_status and 'pending'::delivery_status, but the migration created the column as varchar(20) with a CHECK constraint, not a Postgres enum type. Calls to IncrementRetryCount fail with: pq: type "delivery_status" does not exist.

Discovered while validating user-subscribed RSS fetch (PR #196). The bug was latent because the upstream NoOpFeedProcessor meant the outbox worker never had work to process. Now that real items are being queued, every retry attempt errors out.

Reproduce:
1. Run the cairn selfhost stack with at least one subscribed RSS feed
2. Wait one poll cycle (~60s)
3. Watch container logs for: 'Failed to increment retry count: pq: type "delivery_status" does not exist'

Fix options (pick one — do not implement without confirming):
A) Drop the ::delivery_status casts in outbox.go and rely on the implicit varchar comparison + CHECK constraint
B) Add a migration that converts delivery_status to a real Postgres enum type, and update the CHECK constraint accordingly

Files:
- services/read/fetcher/internal/repository/outbox.go (search for ::delivery_status)
- services/read/fetcher/migrations/ (current schema for content_outbox)
