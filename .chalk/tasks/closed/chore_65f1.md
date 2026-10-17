---
id: chore_65f1
title: Docs + OpenAPI completeness review (validate against code)
type: chore
status: closed
priority: 2
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-10-14T11:57:56Z
updated_at: 2026-10-16T10:46:03Z
---


## Plan
Parallel sonnet reviewers, each owning a disjoint file set; claims validated against code, not assumed. Progressive-discovery structure preserved (short summaries → links to detail docs). Strategy/requirements/justification docs are read-only: drift and questions go to the owner.

- [x] A. Root CLAUDE.md/AGENTS.md, README.md, docs/ARCHITECTURE.md, docs/architecture/http-frameworks.md, docs/CONTENT_SERVICE_JWT_AUTH.md, pkg/auth/README.md
- [x] B. services/read/content/api/openapi.yaml
- [x] C. services/read/CLAUDE.md+AGENTS.md, services/read/README.md, services/read/content/README.md
- [x] D. Fetcher + email: fetcher/email openapi.yaml, docs/fetcher_api_reference.md, email CLAUDE.md, email migrations README
- [x] E. services/users/api/openapi.yaml
- [x] F. services/users/CLAUDE.md+AGENTS.md, README.md, migrations README
- [x] G. apps/mobile/CLAUDE.md+AGENTS.md
- [x] H. apps/web/CLAUDE.md
- [x] I. docs/DEPLOYMENT.md, infrastructure/docker/README.md, infrastructure/docker/selfhost/README.md
- [x] J. docs/CONFIGURATION.md, docs/TROUBLESHOOTING.md
- [x] K. docs/TESTING.md, services/users/test/*/README.md
- [x] L. Read-only drift review: product_requirements, detailed_requirements/*, decisions/*, ENGINEERING_PRINCIPLES, QUALITY_REMEDIATION_STRATEGY, LOGGING_STRATEGY, CODE_QUALITY_REVIEW
- [x] Cross-check internal links, CLAUDE.md/AGENTS.md mirrors; commit; PR
- Excluded as point-in-time records: docs/architecture/audit/*, PRE_BETA_AUDIT*, docs/archive/*, LEARNINGS.md, .agents/skills/*
- [x] Owner decisions (2026-10-15): archive Explore content to docs/archive and fix stale facts in core docs; draft Feed/Reads requirements from code for review; keep feed re-enable as a requirement (UI gap); re-base gin examples on chi
- [x] M. product_requirements + web_app_requirements (+ docs/archive/explore_{product,web_app}_requirements.md)
- [x] N. read/users service requirements
- [x] O. web-app-adr, QUALITY_REMEDIATION_STRATEGY, CODE_QUALITY_REVIEW (+ docs/archive/explore_code_quality_findings.md)
- [x] P. ENGINEERING_PRINCIPLES + LOGGING_STRATEGY chi re-base

## Review
- 39 files changed. All 4 OpenAPI specs parse (content spec passes redocly lint). Internal links in changed docs all resolve. CLAUDE.md files are symlinks to AGENTS.md.
- Specs: users (+verify-email, resend-verification), content (+GET /, +GET user/{id}/{content_id}, internal bulk auth) and email (+internal senders route) gained missing routes. The {data,meta} envelope and {error,message,details,meta} error shape are documented throughout.
- Large duplicated detail trimmed in favour of links: mobile CLAUDE.md 1161→~170 lines, users CLAUDE.md API section → route table + spec link, fetcher_api_reference.md → pointer.
- Incident: a reviewer ran `git stash` mid-run. It was restored with no loss.
