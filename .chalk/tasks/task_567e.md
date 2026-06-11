---
id: task_567e
title: Web: Explore tab (FR-5, FR-6) — recommendations feed and voting
type: task
status: open
priority: 2
labels: []
blocked_by: []
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-08T11:34:02Z
updated_at: 2026-06-11T22:28:13Z
---
Implement /explore and /explore/:id: the discovery feed with up/down voting.

WHAT TO DO:
- Fetch GET /api/v1/explore/recommendation?offset=... (ExploreService.getRecommendations), paginated by offset.
- Render recommendations in a card grid or centered feed column suited to desktop widths.
- Infinite scroll / load-more using offset pagination.
- Best-effort 'shown' reporting: POST /api/v1/explore/shown with batched article IDs for visible articles, mirroring mobile telemetry.
- Up/down vote controls on each card: POST /api/v1/explore/article/{id}/vote ({vote_type}), DELETE to remove vote. Optimistic UI: update the button state immediately, revert on error.
- /explore/:id: full article reader (same reading column as /read/:id) using the explore content field. Actions: mark as read (POST /api/v1/explore/article/{id}/read), up/down vote, save to reading list (via addURL).

VERIFICATION (agent-testable):
1. /explore shows a list/grid of recommended articles with titles and sources.
2. Scrolling to the bottom loads more recommendations (offset increments).
3. Clicking the upvote button on a recommendation highlights it (optimistic UI) and the vote persists after page refresh (GET vote endpoint confirms).
4. Clicking upvote again (or downvote) toggles/removes the vote correctly.
5. Clicking a recommendation opens /explore/:id with the article content rendered.
6. Clicking 'Save to reading list' from the explore reader adds the article to /read.
