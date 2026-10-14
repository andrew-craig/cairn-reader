---
id: bug_d675
title: Users service: integration/security test harness runs nothing or the wrong thing
type: bug
status: open
priority: 2
labels: [testing,users]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-10-14T20:28:44Z
updated_at: 2026-10-14T20:28:44Z
---
Found in docs review chore_65f1 (2026-10-15).
- services/users/Makefile: test-integration and test-all run go test without -tags=integration, confirmed 'no tests to run'. Also the .PHONY line and help output are out of sync with the real targets (dev, test-all, test-coverage, migrate-version, db-create, db-drop).
- test/integration/testutil/setup.go defaults JWT paths to secret/data/cairn/jwt/..., while init-test-vault.sh writes secret/jwt/private-key and public-key, so the tests silently use generated keys.
- docker-compose.test.yml mounts ../../migrations into initdb.d, which includes *.down.sql files, so the schema may be wrong (unverified).
- test/security/*.sh send no X-Forwarded-Proto: https, so they get 403. rate_limit_test.sh assumes 10/min but the default is 100.
- Neither suite runs in CI (go-checks.yml only runs ./internal/...). Decide whether they should.
