---
id: task_8f0c
title: Web: Add link modal (FR-3) — URL detection, feed discovery, save
type: task
status: open
priority: 2
labels: []
blocked_by: [task_6305]
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-08T11:33:50Z
updated_at: 2026-06-08T11:33:50Z
---
Implement the Add Link modal: URL input with type detection and save, reusing mobile's AddLinkModal flow.

WHAT TO DO:
- Add-link button in the top action bar (right of search on /read) opens a modal.
- User enters a URL. On submit (or after debounce): call POST /api/v1/content/detect (10s timeout) and POST /api/v1/content/discover-feed (15s timeout) to classify as page vs feed and surface discovered feeds.
- If a feed is detected, show the feed options (as in mobile's AddLinkModal).
- Submit via POST /api/v1/content/user/{userId} (addURL). On success (type:'page'): add article to reading list. On success (type:'feed'): show confirmation and refresh feeds.
- Match mobile timeout and error handling behavior.

VERIFICATION (agent-testable):
1. Clicking the add-link button opens the modal with a URL input field.
2. Entering a valid article URL (e.g. a news article) and submitting saves it; closing the modal shows the new article at the top of the reading list.
3. Entering a known RSS feed URL shows feed discovery UI with the detected feed title.
4. Confirming the feed subscription creates a feed subscription (visible in /you/feeds).
5. Entering an invalid URL (e.g. 'not-a-url') shows a user-visible error message.
6. Closing the modal without submitting leaves the reading list unchanged.
