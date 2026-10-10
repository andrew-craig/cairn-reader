---
id: task_f9ad
title: Mobile: Feed and Reads tabs
type: task
status: open
priority: 1
labels: [mobile]
blocked_by: [task_85bd,task_47d5,task_179f,task_317b]
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-08T11:32:05Z
updated_at: 2026-10-09T11:00:47Z
---
- [ ] Shared types: list on UserContentResponse + UnifiedSubscription; ReadService.listUserContents({list}), setSourceList, moveToReads
- [ ] Tabs: Feed | Reads | You
- [ ] Reads = current ReadScreen with list='reads'; offline store, prefetch and outbox scoped to Reads only
- [ ] Feed = ArticleListScreen + useCursorArticleList({list:'feed'}); opens in the existing reader; 'Save to Reads' action instead of status triage; no unread counts; online-only with a stale cache
- [ ] AddLinkModal: 'Add Feed' asks Feed or Reads before subscribing
- [ ] Sources screen: per-row Feed/Reads toggle (applies to new items only — say so in the UI)
- Done when: typecheck, lint, tests pass; manual run shows routed items landing in the right tab
