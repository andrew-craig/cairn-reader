---
id: task_e0f7
title: Web: Article reader (FR-4) — sanitized HTML, scroll persistence, reader actions
type: task
status: open
priority: 1
labels: []
blocked_by: [task_6305]
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-08T11:33:33Z
updated_at: 2026-06-08T11:33:33Z
---
Implement /read/:id: the full article reader with sanitized HTML rendering, scroll persistence, and all reader actions.

WHAT TO DO:
- Render article content from Article.content (cleaned_html). Sanitize with DOMPurify before injecting into the DOM. External links must have rel='noopener noreferrer'.
- Reading column: max-width ~65–75ch, centered, Crimson Pro headings, Inter body, generous line height. Display title, author/site, published date, and reading time above the article.
- Scroll position persistence: on scroll, debounce PATCH /api/v1/content/user/{userId}/{contentId} with scroll_position. On re-open, restore scroll position.
- Status transitions via PATCH: mark reading when opened, completed when scrolled near the end.
- Reader actions: Favorite/unfavorite (PATCH is_favorite), Archive (PATCH status='archived'), Delete (DELETE endpoint), Open original (window.open in new tab).
- Prev/next navigation: use the list + index passed from /read to navigate to the adjacent article without returning to the list.
- Mark article as reading (status update) when the reader is opened.

SECURITY: DOMPurify sanitization is mandatory and must be tested explicitly.

VERIFICATION (agent-testable):
1. Opening a saved article at /read/:id renders the article title and body content without layout errors.
2. Leaving the article and reopening it restores the previous scroll position (within ~50px).
3. Clicking Favorite toggles the favorite state (icon updates); reloading confirms the state persisted.
4. Clicking Archive changes article status to archived; the article no longer appears in the default reading list.
5. Clicking Delete removes the article and navigates back to /read; the article is gone from the list.
6. Clicking 'Open original' opens a new tab with the article's original URL.
7. Prev/Next buttons navigate to adjacent articles in the list without returning to /read.
8. Injecting a script tag via cleaned_html does not execute (DOMPurify removes it).
