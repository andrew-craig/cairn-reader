---
id: epic_6e4d
title: Feed vs Reads: user-chosen destination per source
type: epic
status: open
priority: 1
labels: [product,read,mobile,web]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-10-08T07:29:35Z
updated_at: 2026-10-08T07:29:35Z
---
## Goal
Restructure the app from "Explore + Read" into two lists the user chooses between: **Feed** (infinite scroll, skim) and **Reads** (each item is read or triaged). The user decides per source:
- RSS subscription → Feed or Reads (chosen at subscribe time, changeable later)
- Email sender → Feed or Reads (changeable per sender; new senders default to Reads)
- Directly saved links → always Reads
Explore is removed entirely for now.

## Decisions (owner, 2026-10-08)
1. **Explore is removed completely for the moment.** No Discover surface; tabs are Feed | Reads | You.
2. **New email senders default to Reads** (today's behaviour).
3. **Re-routing a source affects new items only.** Existing items stay where they are; no bulk move.
4. **Feed retention: 30 days.** Feed items not favorited are deleted after 30 days.

## How it works today (what the design reuses)
- Read tab = `user_contents` in the content service. RSS (fetcher outbox) and email (email outbox) both deliver through `POST /api/v1/internal/content/user/bulk`, and direct saves go through `POST /content/user/{id}`.
- The content service is already the subscriptions gateway: subscribe goes through `AddContentToUser` → `IngestRSSClient.SubscribeUserToFeed`, and list/unsubscribe through `SubscriptionAggregatorHandler`.
- Clients: `ArticleListScreen` + `useCursorArticleList` (cursor paging), `ReadArticleDetailScreen` (reader, mutations, offline outbox), `SubscriptionListScreen` (Feeds/Newsletters management).

## Design: the content service owns the routing
- New table `source_routes(user_id, source_type, source_key, list)` in `content_service`. `source_key` = feed ID (RSS) or sender ID (email).
- New column `user_contents.list` (`'feed' | 'reads'`, NOT NULL), backfilled to `reads`.
- At ingest, `BulkAddToUsersInternal` stamps `list` from the route for (user, source); no route → `reads`. Direct saves → `reads`.
- Changing a route only updates `source_routes`, so only items delivered afterwards land in the new list.
- Why not on the subscription in the fetcher/email services: each producer's outbox would need per-user destinations (the fetcher outbox is one row per item with `user_ids[]`). Owning it in the content service keeps producers nearly unchanged and fits its gateway role.
- Rejected: a separate Feed store/service — it would duplicate storage, the reader, mutations and search.

## Subtasks (in order)
Prerequisites:
- task_499a — type the outbox payloads (email payload gains `sender_id` in Phase 1)
- task_179f — mobile archive semantics (Feed items share the reader)
- task_317b — aggregator hides per-source failures (Sources screen sets routing)
Then the subtasks under this epic: Explore removal, Sources screen rename, backend routing, Feed retention, mobile, web, docs. See `chalk list --parent=epic_6e4d`.
