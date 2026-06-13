---
id: task_de90
title: Reader screens: track mutable UI state (favorite/read) in dedicated state, not the article object
type: task
status: open
priority: 2
labels: []
blocked_by: []
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-13T01:37:28Z
updated_at: 2026-06-13T01:37:28Z
---


Deferred from PR #267 review (gemini-code-assist comments #3/#4 on apps/web/src/routes/ReadArticle.tsx).

CONTEXT:
The web article reader stores the whole Article in local state (`const [article, setArticle]`) and mutates properties (isFavorite, isRead) via `setArticle(prev => ...)`. The reviewer suggested tracking these mutable UI properties in dedicated `useState` hooks instead, to follow the codebase convention (used in ExploreArticleDetailScreen for save/vote) and to avoid races where an async update from a previous article lands after navigating to the next one.

WHY DEFERRED (not a quick fix):
- The web reader mirrors apps/mobile/src/screens/ReadArticleDetailScreen.tsx, which ALSO keeps the article in state and mutates it. Changing only web would diverge the two analogous screens.
- The reader genuinely swaps the article on prev/next and on the direct-load fetch fallback, so the Article object cannot simply live in route params untouched — any refactor must keep article swapping working while moving favorite/read out (and re-syncing them on article change).

WHAT TO DO (both platforms, kept consistent):
- apps/web/src/routes/ReadArticle.tsx and apps/mobile/src/screens/ReadArticleDetailScreen.tsx: introduce dedicated state for the mutable UI bits (isFavorite, isRead) seeded from the current article and resynced when the displayed article changes; keep `article` itself for content/swapping.
- Guard async favorite/scroll/status updates against the article id so a late response from a previous article does not mutate the newly displayed one.

VERIFICATION:
- Type-check, lint, build pass on both apps.
- Favoriting article A then quickly navigating to B does not flip B's favorite state.
- Existing reader behavior (scroll persistence, status transitions, prev/next) unchanged.
