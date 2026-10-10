---
id: task_f74f
title: Mobile: same management actions on Feeds and Newsletters screens (Feed/Reads toggle, unsubscribe)
type: task
status: open
priority: 2
labels: [mobile]
blocked_by: [task_317b]
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-10T06:17:16Z
updated_at: 2026-10-10T06:17:16Z
---
Review of task_7df9: both source types now carry list on UnifiedSubscription and share the PUT list endpoint, so both screens need the same controls. Today the shared SubscriptionListScreen only supports unsubscribe for RSS.
- [ ] Shared types: list on UnifiedSubscription; ReadService.setSourceList(type, key, list) (key = rss_data.feed_id or the sender id)
- [ ] Per-row Feed/Reads control on BOTH FeedsScreen and NewslettersScreen, via SubscriptionListScreen; say it applies to new items only
- [ ] Newsletters: enable unsubscribe once the backend newsletter-unsubscribe task lands; remove the 'not yet supported' alert
- [ ] Add Feed asks Feed or Reads before subscribing (RSS only; senders are created by their first email and default to Reads)
- Done when: typecheck, lint, tests pass; both screens behave the same
- Note: task_47d5 (merge into one Sources screen) is intentionally left open; this task keeps the two screens.
