---
id: task_30e3
title: Decommission the Explore backend (fetcher + recommender)
type: task
status: closed
priority: 2
labels: [explore,infra]
blocked_by: []
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-08T11:32:05Z
updated_at: 2026-10-12T07:04:48Z
---
Follows client removal so no shipped client calls a removed API.
- [x] Remove explore services from dev/prod/selfhost docker-compose, selfhost single binary and Makefiles, CI workflows
- [x] Delete services/explore (recoverable from git history if Explore returns)
- [x] Drop explore DB creation from init scripts
- [x] Close/obsolete explore-only tasks: epic_c482, task_b5bd, task_f84d, task_02c8, task_19a9, task_3216, task_5baa (and the explore part of task_603e)
- Done when: go build/test pass for remaining modules; selfhost image builds and serves read/users APIs

## Review (2026-10-12)
Branch: `chore/task_30e3-decommission-explore`.
- Deleted services/explore, the selfhost explore adapters, explore Docker build workflows, the explore-recommender Vault policy, pkg/auth/examples/explore-service and the selfhost feeds.txt.
- cmd/selfhost: dropped the recommender/fetcher mounts and migrations, their config (DB_NAME_RECOMMENDER/FETCHER, FEED_LIST_*, FETCH_INTERVAL, ARTICLE_RETENTION_DAYS) and go.mod requirement.
- Compose (dev/prod/selfhost), .env examples, init-databases.sh (both), init-vault-prod.sh, Caddyfile, go-checks.yml, docker-test.yml, verify-docker-builds.sh, .gitignore, selfhost README.
- Verified: go build/vet/test pass for every remaining module (and `go vet -tags=integration` for cmd/selfhost). Built the selfhost image from source and ran it as an isolated compose project on a fresh volume: 4 DBs created, /health/ready healthy (users, content, rss, email), register, save link (Reads), list, subscriptions, RSS subscribe to Feed all OK, /api/v1/explore/* returns 404.
- Existing deployments keep orphaned cairn_recommender / cairn_fetcher databases. Drop them by hand if wanted; nothing reads them.
- Prose docs moved to task_082f. Explore parts of task_603e, task_fbb9, task_6f3a marked obsolete.
