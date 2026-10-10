---
id: task_42b5
title: Web: same management actions on Feeds and Newsletters routes (Feed/Reads toggle, unsubscribe)
type: task
status: open
priority: 2
labels: [web]
blocked_by: [task_317b]
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-10T06:17:16Z
updated_at: 2026-10-10T06:17:16Z
---
Web mirror of the mobile parity task, for apps/web/src/routes/Feeds.tsx and Newsletters.tsx.
- [ ] Shared types and service call for setSourceList
- [ ] Per-row Feed/Reads control on both routes, with the new-items-only note
- [ ] Newsletters unsubscribe once the backend task lands
- [ ] Feed/Reads choice when adding a feed
- Done when: tests pass; verified in browser against staging (webapp-staging-test skill)
