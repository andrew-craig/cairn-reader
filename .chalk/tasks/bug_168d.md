---
id: bug_168d
title: Users service: NUL byte in a request string returns 500 instead of 400
type: bug
status: open
priority: 3
labels: [users,security]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-10-18T02:09:11Z
updated_at: 2026-10-18T02:09:11Z
---
Found by the security suite once bug_d675 made it run (test/security/sql_injection_test.sh, "Null byte in email").

POST /api/v1/auth/login with {"email":"admin@example.com\u0000","password":"password"} returns 500 {"error":"internal_error","message":"failed to authenticate user"}. Postgres rejects the string: `ERROR: invalid byte sequence for encoding "UTF8": 0x00`. The handler maps the driver error to 500.

Not an injection (queries are parameterised), but a client-controlled 500. Every endpoint that passes a JSON string into a query is probably affected (email, device ID, tokens).

Also: the service log line for the 500 is only "http request completed status=500"; the underlying error is not logged, so the cause was only visible in the Postgres log.

Fix direction: reject strings containing NUL as invalid input (400) at validation, once, rather than per endpoint. Then `make test-security` should pass fully.


Adding the security suite to CI (owner decision on bug_d675) waits on this fix.
