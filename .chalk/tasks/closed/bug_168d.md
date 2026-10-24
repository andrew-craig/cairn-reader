---
id: bug_168d
title: Users service: NUL byte in a request string returns 500 instead of 400
type: bug
status: closed
priority: 3
labels: [users,security]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-10-18T02:09:11Z
updated_at: 2026-10-21T20:44:30Z
---
Found by the security suite once bug_d675 made it run (test/security/sql_injection_test.sh, "Null byte in email").

POST /api/v1/auth/login with {"email":"admin@example.com\u0000","password":"password"} returns 500 {"error":"internal_error","message":"failed to authenticate user"}. Postgres rejects the string: `ERROR: invalid byte sequence for encoding "UTF8": 0x00`. The handler maps the driver error to 500.

Not an injection (queries are parameterised), but a client-controlled 500. Every endpoint that passes a JSON string into a query is probably affected (email, device ID, tokens).

Also: the service log line for the 500 is only "http request completed status=500"; the underlying error is not logged, so the cause was only visible in the Postgres log.

Fix direction: reject strings containing NUL as invalid input (400) at validation, once, rather than per endpoint. Then `make test-security` should pass fully.


Adding the security suite to CI (owner decision on bug_d675) waits on this fix.

## Review (2026-10-21)
- [x] Reproduced: `sql_injection_test.sh` "Null byte in email" → 500 (29/30).
- [x] One check, not per endpoint: new `decodeJSONBody` (`internal/handlers/decode.go`) replaces the ten copies of the MaxBytesReader + decode + 413/400 block. After decoding it walks the request struct's string fields (incl. `*string`) and returns 400 `string fields must not contain NUL characters` if any contains U+0000. Unknown JSON fields never reach the DB and are ignored; `"\\u0000"` (literal backslash) is not NUL and is accepted.
- [x] Not affected: `?token=` on verify-email (tokens are SHA-256 hashed before any query); headers (net/http rejects NUL in header values); `{user_id}` (parsed as UUID).
- [x] Logging: `writeServiceError` now takes `r` and logs the underlying error at Error on its 500 path (the only path to a 500), so the cause is in the service log, not just Postgres's.
- [x] Tests: `decode_test.go` (table incl. NUL in string/pointer field, escaped backslash, size cap; Login handler returns 400), `TestWriteServiceError_UnmappedIs500` asserts the log line.
Results: unit tests pass; against local Postgres 16 + Vault: integration suite passes, `make test-security` passes all three suites (sql_injection 30/30, "Null byte in email" → 400).
