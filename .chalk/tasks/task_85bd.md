---
id: task_85bd
title: Remove Explore from mobile and web clients
type: task
status: open
priority: 1
labels: [mobile,web]
blocked_by: []
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-08T11:32:05Z
updated_at: 2026-10-08T11:32:05Z
---
Owner decision 2026-10-08: remove Explore completely for now.
- [ ] Mobile: drop Explore tab, ExploreScreen, ExploreArticleDetailScreen, VotesScreen, ExploreService, explore AsyncStorage cache, You→Votes link, nav types
- [ ] Web: drop /explore, /explore/:id, /you/votes routes, Explore.tsx, ExploreArticle.tsx, Votes.tsx, services/explore.ts, Sidebar/BottomNav entries
- [ ] Remove now-unused shared types/helpers (only what this change orphans)
- Done when: no client code references /api/v1/explore; typecheck, lint and tests pass in apps/mobile and apps/web
