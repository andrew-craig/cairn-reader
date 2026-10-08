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
Restructure the app from "Explore + Read" (split by where content comes from) into two lists the user chooses between: **Feed** (infinite scroll, skim) and **Reads** (each item is read or triaged). The user decides per source:
- RSS subscription → Feed or Reads (chosen at subscribe time, changeable later)
- Email sender → Feed or Reads (changeable per sender)
- Directly saved links, and Explore articles saved with "save" → always Reads

## How it works today (what the design reuses)
- Explore tab = global Kagi Small Web recommendations from the explore recommender. No user subscriptions are involved.
- Read tab = `user_contents` in the content service. RSS (fetcher outbox) and email (email outbox) both deliver through `POST /api/v1/internal/content/user/bulk`, and direct saves go through `POST /content/user/{id}`.
- The content service is already the gateway for subscriptions. Subscribe goes through `AddContentToUser` → `IngestRSSClient.SubscribeUserToFeed`, and list/unsubscribe through `SubscriptionAggregatorHandler`.
- Clients: `ArticleListScreen` + `useCursorArticleList` (cursor paging), `ReadArticleDetailScreen` (reader, mutations, offline outbox), `SubscriptionListScreen` (Feeds/Newsletters management).

## Key design decision: the content service owns the routing
- New table `source_routes(user_id, source_type, source_key, list)` in `content_service`. `source_key` is the feed ID for RSS and the sender ID for email.
- New column `user_contents.list` (`'feed' | 'reads'`, NOT NULL). Backfill every existing row to `reads`, since that matches what users see today.
- At ingest, `BulkAddToUsersInternal` looks up the route for (user, source) and stamps `list`. Without a route, RSS defaults to the value chosen at subscribe, and email uses the default for new senders (see open questions). Direct saves always get `reads`.
- Why not store the route on the subscription in the fetcher/email services? Then each producer would need per-user destinations in its outbox. The fetcher outbox currently holds one row per item with `user_ids[]`. Moving existing items would also need a new cross-service call. Owning the route in the content service keeps producers almost unchanged, and re-routing a source becomes a single `UPDATE ... JOIN contents` in one database. It also fits the content service's existing role as the subscriptions gateway.
- Rejected alternative: a separate Feed store or service. It would duplicate storage, the reader, mutations and search, all of which already work for `user_contents`.

## Phase 0: prerequisites (do first)
1. **task_499a: type the outbox payloads (fetcher + email).** Phase 1 adds `sender_id` to the email payload. Today producer and consumer only agree by convention, so a new field could be silently dropped. Land the typed payload first, then add the field.
2. **task_179f: mobile archive semantics.** Feed items need clear "dismiss / save to Reads / archive" semantics, and archive is currently a hard DELETE with swallowed errors and two caches. Settle what archive means before adding a second list that shares the reader.
3. **task_317b: aggregator swallows per-source failures.** The subscriptions screen becomes where users set routing. A silently missing source would mean a user can't see, or misroutes, a source. Surface partial failure before building on it.
4. **Naming collision:** "Feeds" already names the RSS-subscription management screen (`FeedsScreen`, `/you/feeds`). Fold Feeds + Newsletters into one **Sources** screen (reusing `SubscriptionListScreen`) so "Feed" can name the list.
(task_6fe1, the content create pipeline, is *not* a prerequisite. This work touches `user_contents`, not `contents` creation.)

## Phase 1: backend (content service + email)
- [ ] Migration 000005: `user_contents.list` + backfill `reads` + index `(user_id, list, added_at DESC)`; `source_routes` table → verify: migration up/down integration test
- [ ] Email payload carries `sender_id` (the typed payload from 499a); the content service stores it on `contents.metadata` or a new `source_sender_id` column → verify: worker test + bulk integration test
- [ ] `BulkAddToUsersInternal` resolves the list per user from `source_routes` → verify: integration tests for RSS→feed, RSS→reads, email sender→feed, no route→default
- [ ] Subscribe (`AddContentToUser` feed branch) accepts `list` and writes the route. Unsubscribe deletes the route. → verify: handler tests
- [ ] `PUT /content/user/{id}/subscriptions/{type}/{key}/list {list}` updates the route and moves the source's existing items (see open questions) → verify: integration test
- [ ] `ListUserContents` / `Search` / `Count` take a `list` filter; the unified subscriptions response includes each source's `list` → verify: repo + handler tests, openapi.yaml updated
- [ ] `PATCH /content/user/{id}/{content_id}` accepts `list` ("save to Reads" from the Feed) → verify: handler test
- [ ] Feed retention: extend the existing cleanup job to delete `list='feed'` rows that are not favorited and older than N days. The existing orphan trigger and cleanup then reclaim `contents`. → verify: cleanup job test
- [ ] Fetcher: no change beyond 499a (`source_feed_id` is already in the payload)

## Phase 2: clients (mobile first, then web)
- [ ] Shared types: `list` on `UserContentResponse` and `UnifiedSubscription`; `ReadService.listUserContents({list})`, `setSourceList`, `moveToReads`
- [ ] Tabs become **Feed | Reads | You**
- [ ] **Reads** = current `ReadScreen` with `list='reads'`. The offline store, prefetch and outbox stay scoped to Reads only, so Feed items are never prefetched.
- [ ] **Feed** = `ArticleListScreen` + `useCursorArticleList({list:'feed'})`. Opening an item uses the existing reader. Feed items show a "Save to Reads" action instead of status triage, and there are no unread counts. Online-only, with a stale cache like Explore today.
- [ ] Explore recommendations: placement depends on open question 1 (recommended: a "Discover" segment inside Feed that reuses `ExploreScreen` almost unchanged)
- [ ] AddLinkModal: when the URL is a feed, "Add Feed" asks "Feed or Reads?" before subscribing
- [ ] Sources screen: a per-row Feed/Reads toggle for RSS feeds and email senders
- [ ] Web: the same changes in `routes/` (`Read.tsx` → Reads, new `Feed.tsx`, `Explore.tsx` → Discover, `Feeds.tsx`+`Newsletters.tsx` → Sources)
- [ ] Docs: ARCHITECTURE.md, service CLAUDE.md files, openapi specs

## Open questions (plan assumes the recommended answer)
1. Explore recommendations: (a) a Discover segment inside the Feed tab **[recommended]**, (b) mixed into the user's Feed, or (c) a separate third tab?
2. Default list for an email sender seen for the first time: Reads (today's behaviour) **[recommended]** or Feed?
3. When a source is re-routed, move its existing items? Recommended: move items that are unread and not favorited; leave anything in progress or favorited where it is.
4. Feed retention window N (proposed 30 days).
