---
id: feature_4524
title: Save read progress for Feed items on web and mobile
type: feature
status: open
priority: 3
labels: [web,mobile,feed]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-11-11T21:41:11Z
updated_at: 2026-11-11T21:44:10Z
---
## Context
Feed items should save reading progress (scroll position) on both web and mobile, as Reads items already do. Today neither client does, so the partially read title indicator (PR #452, shared `isPartiallyRead`) never shows on Feed rows:
- Web: `apps/web/src/routes/ReadArticle.tsx` skips the scroll saves (throttled save and unmount flush) when `isFeed`.
- Mobile: `apps/mobile/src/screens/ReadArticleDetailScreen.tsx` gates scroll tracking on `tracksProgress = article.list === 'reads'`.

## Plan
- [ ] Web: save `scroll_position` for Feed items in ReadArticle.tsx (throttled save + unmount flush); restore it when a Feed item is reopened
- [ ] Mobile: save scroll progress for Feed items in ReadArticleDetailScreen via `ArticleMutations.saveScrollPosition`; restore it on reopen
- [ ] Mobile: Feed items are not in `ArticleStore`, so confirm the store write is a harmless no-op and the backend PATCH / `scroll_position` outbox row still works offline
- [ ] Mobile: make the Feed list (and `FeedCache`) show the new progress when returning from the reader, not only after the next refresh
- [ ] Feed items are never marked completed, so check how a fully scrolled Feed item looks in the list with `isPartiallyRead`
- [ ] Check Save to Reads keeps the saved `scroll_position`
- [ ] Tests: web ReadArticle and mobile ReadArticleDetailScreen save scroll position for Feed items
- [ ] Docs: update apps/web/CLAUDE.md and apps/mobile/CLAUDE.md (mobile currently says Feed progress tracking is off and Feed items only produce `list` and `is_favorite` outbox rows)
