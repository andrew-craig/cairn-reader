---
id: task_8ba5
title: Review docs/TESTING.md integration checklists for uncompleted todos
type: task
status: in_progress
priority: 2
labels: []
blocked_by: []
parent: epic_a3df
created_at: 2026-03-21T04:20:36Z
updated_at: 2026-04-08T09:03:30Z
---
Read docs/TESTING.md (lines 407-596) which contains extensive manual integration testing checklists. Check each item against the codebase.

## Plan

### Findings
Reviewed lines 399-840+ of TESTING.md against the actual codebase. Found widespread discrepancies:

#### Explore Service E2E Test Plan (lines 399-640)
- [ ] Health endpoints: docs say `GET /health`, actual is `GET /health/live` and `GET /health/ready`
- [ ] Health response: docs say `{"status":"healthy"}`, need to match actual response
- [ ] Article submission: docs say `POST /api/v1/articles`, actual is `POST /api/v1/explore/article`
- [ ] Recommendations: docs say `GET /api/v1/recommendations/{userID}`, actual is `GET /api/v1/explore/recommendation/{user_id}`
- [ ] Mark-as-read: docs say `POST /api/v1/articles/read` with user_id in body, actual is `POST /api/v1/explore/article/{article_id}/read` with JWT auth
- [ ] All curl test commands use wrong URLs/format
- [ ] Known Limitations section (lines 631-640) entirely outdated — all 5 items now implemented

#### Read Service (lines 644-788)
- [ ] Path refs: `content-service` → `content`, `rss-fetcher-service` → `fetcher`
- [ ] Docker services: `content-db rss-db` → `cairn-db`
- [ ] Test helper: `testhelpers.SetupTestDatabase(t)` → `testutil.SetupTestDB(t)`
- [ ] Cleanup: `testDB.Cleanup()` → `testutil.CleanupTestDB(t, database)` + `database.Close()`

#### User Service (lines 792-826)
- [ ] Docker service: `vault users-db` → `vault cairn-db`

### Approach
Fix all discrepancies in a single pass through TESTING.md.
