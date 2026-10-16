---
id: feature_bd13
title: Mobile: Add feed option on the RSS (feeds) page
type: feature
status: closed
priority: 2
labels: [mobile]
blocked_by: []
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-13T21:22:21Z
updated_at: 2026-10-16T10:49:36Z
---
The RSS subscriptions page (formerly 'Feeds'; SubscriptionListScreen via RssScreen) has no way to add a feed. Users have to go through the global Add Link modal. Add an 'Add feed' entry point on that page that subscribes to an RSS feed and asks for Feed or Reads.
Owner request, 2026-10-14 (after the task_f9ad manual run).

## Plan (2026-10-16, done together with task_9188 in one PR)
- [x] RssScreen header gets an "Add feed" (+) button opening `AddLinkModal mode="feed"`
- [x] Feed mode: placeholder "Feed or website URL"; until a feed is detected, Find feed is the only action (return key runs it), so a page is never saved as an article from this page
- [x] `SubscriptionListScreen` gains `reloadKey`; RssScreen bumps it on success so the new feed shows without leaving the page
- [x] Tests: AddLinkModal feed mode (choice explained, page → Find feed only), RssScreen end to end (subscribe to Feed, list reloads)
## Review
- Mobile type-check clean, lint adds no warnings, jest 41 suites / 303 tests pass.
