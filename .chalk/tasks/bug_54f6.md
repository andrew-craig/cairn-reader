---
id: bug_54f6
title: Prod Vault init never provisions email-ingest AppRole
type: bug
status: open
priority: 2
labels: [quality,infrastructure,vault,bugfix]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-09-14T11:56:13Z
updated_at: 2026-09-14T11:56:13Z
---
**Source:** follow-up promised by task_2315 (content-service AppRole, merged as #374) and never filed. Recorded here during a backlog housekeeping sweep on 2026-09-14, re-verified against `main` at `ba7ac84`.

## Problem
`infrastructure/docker/prod/docker-compose.yml:456-457` sets, on the `email-ingest` container:
```
- VAULT_ROLE_ID=${EMAIL_INGEST_ROLE_ID}
- VAULT_SECRET_ID=${EMAIL_INGEST_SECRET_ID}
```
and `infrastructure/docker/prod/.env.example:54-55` has placeholders for both — but
`infrastructure/docker/scripts/init-vault-prod.sh` provisions policies and AppRoles for only
**three** services:

| Service | Policy | AppRole | Credentials written |
|---|---|---|---|
| user-service | `:193` | `:257` | `:264-265`, `:303-304` |
| explore-recommender | `:216` | `:270` | `:277-278`, `:307-308` |
| content-service | `:235` | `:283` | `:290-291`, `:311-312` |
| **email-ingest** | **none** | **none** | **none** |

`grep -c email-ingest infrastructure/docker/scripts/init-vault-prod.sh` returns **0**.

A fresh prod deploy therefore has no way to obtain values for those two env vars, so
`email-ingest` starts with empty Vault credentials and cannot fetch the JWT public key —
every authenticated request to it fails. This is the exact bug task_2315 fixed for
content-service, in the one service that task deliberately left out of scope.

## What to do
Follow the pattern task_2315 established for content-service, in the same style as the
surrounding script (`grep -o` extraction, not `jq` — the script does not use `jq` and this
is not the task to introduce it):
1. Add an `email-ingest` policy: read-only on `secret/data/jwt/public-key`, plus
   `auth/token/renew-self` / `auth/token/lookup-self`, identical in scope to the
   `content-service` policy at `:235-251`.
2. Add a matching `vault write auth/approle/role/email-ingest` block.
3. Extract `EMAIL_INGEST_ROLE_ID` / `EMAIL_INGEST_SECRET_ID` into
   `/vault-keys/approle-credentials.env` and into the summary echo output, matching
   `:290-291`, `:311-312` and `:332-333`.

**Path note:** prod compose sets `JWT_PUBLIC_KEY_PATH=secret/data/jwt/public-key` explicitly
for email-ingest (`docker-compose.yml:459`), so the content-service policy path is correct.
The `secret/data/jwt` default in `services/read/email/internal/config/config.go:104` is a
fallback that prod never uses — do not widen the policy to cover it.

## Done when
- [ ] email-ingest policy + AppRole added to `init-vault-prod.sh`, credentials exported
- [ ] `sh -n` clean
- [ ] Run `init-vault-prod.sh` against a real/test Vault: confirm `email-ingest` **and
      `content-service`** role/secret IDs both appear in `/vault-keys/approle-credentials.env`
      and both services authenticate

## Carried-over verification debt from task_2315
task_2315 shipped the content-service AppRole with `sh -n` (syntax check) only — it was
**never run against a live Vault**, and that checklist item was still unticked when the task
was closed. Since this task edits the same script and needs the same live run, verify both
services in one pass rather than filing a third task. That is why the third checklist item
above names content-service as well.
