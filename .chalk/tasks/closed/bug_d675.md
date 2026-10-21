---
id: bug_d675
title: Users service: integration/security test harness runs nothing or the wrong thing
type: bug
status: closed
priority: 2
labels: [testing,users]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-10-14T20:28:44Z
updated_at: 2026-10-21T20:44:30Z
---
Found in docs review chore_65f1 (2026-10-15).
- services/users/Makefile: test-integration and test-all run go test without -tags=integration, confirmed 'no tests to run'. Also the .PHONY line and help output are out of sync with the real targets (dev, test-all, test-coverage, migrate-version, db-create, db-drop).
- test/integration/testutil/setup.go defaults JWT paths to secret/data/cairn/jwt/..., while init-test-vault.sh writes secret/jwt/private-key and public-key, so the tests silently use generated keys.
- docker-compose.test.yml mounts ../../migrations into initdb.d, which includes *.down.sql files, so the schema may be wrong (unverified).
- test/security/*.sh send no X-Forwarded-Proto: https, so they get 403. rate_limit_test.sh assumes 10/min but the default is 100.
- Neither suite runs in CI (go-checks.yml only runs ./internal/...). Decide whether they should.

## Review (2026-10-18)
Reproduced each item, then fixed and re-ran against a local Postgres + Vault.

- [x] `make test-integration`/`test-all` ran zero tests: added `-tags=integration`. Fixed `.PHONY` and the help output (`dev` was missing).
- [x] initdb mounted `migrations/` and ran every `.down.sql` too, in name order. **Confirmed broken**: `000003_…down.sql` fails, init aborts, and the schema stops at migration 2. Removed the mount. `SetupTestEnvironment` now applies the embedded `migrations.FS` via `database.RunMigrationsFS`, the service's own migration path.
- [x] Vault: the defaults read `secret/data/cairn/jwt/*`, and the init script wrote the PEM under `value`, but `GetJWTKeys` reads `key`. Both now match the service (`secret/data/jwt/{private,public}-key`, field `key`).
- [x] Found while running: `WaitForDatabase` built its config without pool sizes ("MaxSize must be >= 1"), so every test failed after 30s. Both helpers now share `testDBConfig()`.
- [x] Found while running: `Logout with empty token` asserted success, but the service, its unit test and the OpenAPI spec all reject an empty token (`ErrInvalidInput` → 400). Aligned the assertion.
- [x] Security scripts: added a `curl` wrapper that sends `X-Forwarded-Proto: https`. Found while running: `((n++))` under `set -e` exits the script when n is 0, so every script stopped after its first check, and SQL injection reported "0 tests, all passed". Replaced with `n=$((n + 1))`.
- [x] rate_limit: `X-RateLimit-*` headers are only set on 429 responses, so the old "headers on first request" check could never pass. The limit now comes from `RATE_LIMIT_REQUESTS` (default 100, same as the service), and the header checks run on the 429.
- [x] auth_bypass: SQL-injection login expects 401 (it only accepted 400). The path-traversal check uses `--path-as-is`, because curl collapsed `../` and sent `/etc/passwd`.
- [x] CI (owner: yes, 2026-10-18): `test-integration-users` in go-checks.yml starts the suite's own docker-compose.test.yml (incl. `test-vault-init`, gated by `docker compose wait`) and runs `go test -race -tags=integration`. Checked locally: the init script's `kv put … key=` seeding works against the real Vault image, and the suite passes with `-race` (~100s).
- [x] Security suite in CI (2026-10-21, after bug_168d): `test-integration-users` now builds and starts the service against the same compose Postgres (5433) and Vault (8201) after the integration tests, waits for `/health/ready`, and runs `make test-security`; the failure step also dumps the service log. Verified locally in the same order (integration suite, then service + security suite on the same DB): all pass. TESTING.md and test/security/README.md no longer say the scripts are not in CI.

Results: integration 78 subtests pass from an empty DB. Security: auth_bypass 13/13, rate_limit 7/7, sql_injection 29/30. The one failure is real: a NUL byte returns 500, filed as bug_168d.
Not verified here: the `test-vault-init` image build, because the sandbox can't reach the Alpine package mirror. Vault was seeded by hand with the same `kv put … key=` commands.
