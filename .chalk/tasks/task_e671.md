---
id: task_e671
title: Backend: let a user unsubscribe from a newsletter sender (parity with RSS unsubscribe)
type: task
status: open
priority: 2
labels: [read,backend]
blocked_by: []
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-10T06:17:16Z
updated_at: 2026-10-10T06:17:16Z
---
Review of task_7df9: feeds and newsletters are routed identically (source_routes, PUT .../subscriptions/{type}/{key}/list, list stamped on UnifiedSubscription), but only feeds can be removed. RSS has DELETE /content/user/{id}/subscriptions/rss/{feed_id} (proxies to the fetcher, then deletes the route). The email service only has address + senders (list) endpoints, there is no aggregator route for email, and mobile shows 'Unsubscribing from this source type is not yet supported'.
- [ ] Decide semantics: what happens to later mail from an unsubscribed sender (drop it vs. re-create the sender). Needs owner input before building.
- [ ] Email service: endpoint to remove/mute a user's sender → handler + repo tests, openapi.yaml
- [ ] Content service: DELETE /content/user/{id}/subscriptions/email/{sender_id} proxying to it, then deleting the (email, sender_id) source_route, mirroring UnsubscribeRSS → handler test; openapi.yaml
- [ ] Existing items from the sender are kept (same as RSS)
- Done when: unsubscribe works for both source types through the aggregator; integration test shows the route is gone and a later re-subscribe starts from Reads
