---
id: task_ea51
title: Web: Feeds screen (FR-8) — RSS subscription list and unsubscribe
type: task
status: open
priority: 2
labels: []
blocked_by: [task_58b2]
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-08T11:34:16Z
updated_at: 2026-06-08T11:34:16Z
---
Implement /you/feeds: the user's RSS feed subscriptions.

WHAT TO DO:
- Fetch subscriptions via GET /api/v1/content/user/{userId}/subscriptions (ReadService.listAllSubscriptions), filtering to non-email types.
- Display each feed with its title/URL and an Unsubscribe button.
- Unsubscribe: DELETE /api/v1/content/user/{userId}/subscriptions/rss/{feedId}. On success, remove the feed from the list. Existing reading list items are preserved (inform user if needed).
- Empty state when no feeds are subscribed.

VERIFICATION (agent-testable):
1. /you/feeds shows all RSS feed subscriptions for the logged-in user.
2. Each feed displays a name/title and an Unsubscribe button.
3. Clicking Unsubscribe on a feed removes it from the list immediately (optimistic or confirmed).
4. After unsubscribing, refreshing /you/feeds confirms the feed is no longer listed.
5. A user with no feed subscriptions sees a meaningful empty state.
