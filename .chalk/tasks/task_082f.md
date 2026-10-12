---
id: task_082f
title: Docs: Feed/Reads model and Explore removal
type: task
status: in_progress
priority: 3
labels: [docs]
blocked_by: [task_f9ad]
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-08T11:32:05Z
updated_at: 2026-10-12T21:20:42Z
---
Update docs/ARCHITECTURE.md, root CLAUDE.md, service and app CLAUDE.md files, and openapi specs for the Feed/Reads model and the removed Explore service.

## Added scope (2026-10-12, from task_30e3)
task_30e3 removed Explore from code, compose, CI and scripts, and updated infrastructure/docker/selfhost/README.md. Remaining prose still describing Explore: infrastructure/docker/README.md (services, ports, DB table, vault AppRoles, logs), pkg/auth/README.md ("Complete Example: Explore Service Integration"), docs/DEPLOYMENT.md, docs/CONFIGURATION.md, docs/TESTING.md, docs/ENGINEERING_PRINCIPLES.md, README.md, AGENTS.md / CLAUDE.md files, .agents/skills/webapp-staging-test/SKILL.md. Find with `grep -rli explore --exclude-dir=node_modules --exclude-dir=.chalk`.

## Plan
- [x] Verify Feed/Reads facts against content service, migrations, mobile and web code
- [x] Root AGENTS.md/CLAUDE.md: overview, diagram, doc links, openapi list
- [x] docs/ARCHITECTURE.md: remove Explore, port table, mobile/web diagrams; add Feed vs Reads routing, source_routes, retention
- [x] services/read, services/read/email, services/users, apps/mobile, apps/web docs
- [x] openapi (content): `list` fields and source-list endpoint checked against handlers; search 400 description fixed
- [x] infrastructure/docker/README.md, pkg/auth/README.md (Content Service example), README.md, DEPLOYMENT, CONFIGURATION, TESTING, ENGINEERING_PRINCIPLES, webapp-staging-test SKILL, CONTENT_SERVICE_JWT_AUTH link
- [x] Re-run `grep -rli explore`; remaining hits are historical audit/requirements docs and code comments (reported, not edited)
