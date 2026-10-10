---
id: task_30e3
title: Decommission the Explore backend (fetcher + recommender)
type: task
status: open
priority: 2
labels: [explore,infra]
blocked_by: []
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-08T11:32:05Z
updated_at: 2026-10-10T03:11:10Z
---
Follows client removal so no shipped client calls a removed API.
- [ ] Remove explore services from dev/prod/selfhost docker-compose, selfhost single binary and Makefiles, CI workflows
- [ ] Delete services/explore (recoverable from git history if Explore returns)
- [ ] Drop explore DB creation from init scripts
- [ ] Close/obsolete explore-only tasks: epic_c482, task_b5bd, task_f84d, task_02c8, task_19a9, task_3216, task_5baa (and the explore part of task_603e)
- Done when: go build/test pass for remaining modules; selfhost image builds and serves read/users APIs
