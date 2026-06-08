---
id: task_3b3c
title: Web: Search modal (FR-2)
type: task
status: open
priority: 2
labels: []
blocked_by: [task_6305]
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-08T11:33:41Z
updated_at: 2026-06-08T11:33:41Z
---
Implement the search modal/overlay for searching saved content.

WHAT TO DO:
- A search bar is shown at the top of the /read list. Activating it (focus/click) opens a modal/overlay or expands inline.
- Call GET /api/v1/content/user/{userId}/search?q=... (ReadService.searchUserContents) on input change (debounced).
- Render results in the same article row/card layout as the reading list.
- Clear action (X button or Escape key) closes the modal and returns to the full reading list.
- Clicking a result navigates to /read/:id.

VERIFICATION (agent-testable):
1. Clicking the search bar on /read opens the search UI.
2. Typing a term that appears in a saved article's title returns that article in the results.
3. Typing a term with no matches shows an empty state (no results message, no error).
4. Pressing Escape or clicking the clear button closes search and shows the full reading list.
5. Clicking a search result navigates to the article reader for that article.
