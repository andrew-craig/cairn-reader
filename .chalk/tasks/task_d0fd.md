---
id: task_d0fd
title: Web: Bookmarks and Votes screens (FR-10, FR-7)
type: task
status: open
priority: 3
labels: []
blocked_by: []
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-08T11:34:31Z
updated_at: 2026-06-11T22:28:13Z
---
Implement /you/bookmarks and /you/votes.

FR-10 — Bookmarks (/you/bookmarks):
- Fetch bookmarked/saved items using the content list endpoint (same as /read but filtered to favorites or a bookmarks subset per mobile's BookmarksScreen logic).
- Render in the same article row/card layout as /read.
- Clicking an item opens /read/:id.

FR-7 — Votes (/you/votes):
- Show articles the user has voted on. Fetch via ExploreService.getUserVoteStats or the relevant votes endpoint.
- Display upvoted and downvoted articles, with the user's vote shown.

VERIFICATION (agent-testable):
1. /you/bookmarks shows articles the logged-in user has favorited/bookmarked.
2. After favoriting an article in the reader, returning to /you/bookmarks shows it in the list.
3. Clicking a bookmark navigates to /read/:id for that article.
4. /you/votes shows articles the user has upvoted or downvoted.
5. After voting on a recommendation in /explore, /you/votes reflects that vote.
