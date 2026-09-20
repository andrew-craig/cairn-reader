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
- [x] email-ingest policy + AppRole added to `init-vault-prod.sh`, credentials exported
- [x] `sh -n` clean
- [x] Run `init-vault-prod.sh` against a real/test Vault: confirm `email-ingest` **and
      `content-service`** role/secret IDs both appear in `/vault-keys/approle-credentials.env`
      and both services authenticate

## Carried-over verification debt from task_2315
task_2315 shipped the content-service AppRole with `sh -n` (syntax check) only — it was
**never run against a live Vault**, and that checklist item was still unticked when the task
was closed. Since this task edits the same script and needs the same live run, verify both
services in one pass rather than filing a third task. That is why the third checklist item
above names content-service as well.

## Review

### Change
`infrastructure/docker/scripts/init-vault-prod.sh` — added the `email-ingest` policy,
AppRole, and credential wiring, in the same style/position as the existing three services
(no `jq`, `grep -o` extraction, same heredoc conventions):
- Policy `email-ingest`: read-only on `secret/data/jwt/public-key` + `auth/token/renew-self`
  (update) + `auth/token/lookup-self` (read) — byte-for-byte the same scope as
  `content-service`, added directly after the `content-service` policy block.
- `vault write auth/approle/role/email-ingest` block (same `token_ttl`/`token_max_ttl`/
  `secret_id_ttl`/`secret_id_num_uses` as the other three roles), added after the
  `content-service` AppRole block, with `EMAIL_INGEST_ROLE_ID`/`EMAIL_INGEST_SECRET_ID`
  extracted the same way as `CONTENT_SERVICE_ROLE_ID`/`_SECRET_ID`.
- Both vars added to the `/vault-keys/approle-credentials.env` heredoc and to the summary
  `echo` block at the end of the script.
- `infrastructure/docker/prod/.env.example` already had `EMAIL_INGEST_ROLE_ID`/
  `EMAIL_INGEST_SECRET_ID` placeholders (lines 54-55) from when this bug was originally
  filed — no changes needed there.

Diff: 1 file changed, 40 insertions(+), 0 deletions. Committed on branch `bug_54f6`
(commit `5774e7d`).

### Verification
1. **Syntax**: `sh -n infrastructure/docker/scripts/init-vault-prod.sh` — clean, no output.
2. **Live Vault run** (dev-mode substitute, explicit about scope below):
   - Built the actual `infrastructure/docker/vault-init/Dockerfile` image (adds openssl +
     GNU grep on top of `hashicorp/vault:1.18`, matching what prod's `vault-init` service
     runs) rather than using the bare `hashicorp/vault` image, so the script executed with
     the same tool availability it has in prod.
   - Ran `hashicorp/vault:1.18 server -dev` (root token `root`) on a Docker network named
     `cairn-vault-test`, container name `vault` (matching the hostname the script's health
     check and `VAULT_ADDR` expect).
   - Ran the built vault-init image with the real, unmodified
     `infrastructure/docker/scripts/init-vault-prod.sh` mounted in, `VAULT_ADDR=http://vault:8200`,
     `VAULT_TOKEN=root`, against a scratch `vault-keys` volume.
   - Script completed with exit code 0. Output showed all four policies uploaded
     (`user-service`, `explore-recommender`, `content-service`, `email-ingest`) and all four
     AppRoles created, plus a summary block listing `VAULT_ROLE_ID`/`VAULT_SECRET_ID` for
     Email Ingest alongside the other three services.
   - Read `/vault-keys/approle-credentials.env` from the volume directly: confirmed both
     `CONTENT_SERVICE_ROLE_ID`/`CONTENT_SERVICE_SECRET_ID` and
     `EMAIL_INGEST_ROLE_ID`/`EMAIL_INGEST_SECRET_ID` pairs were present and non-empty.
   - Authenticated both roles against the running Vault:
     `vault write auth/approle/login role_id=... secret_id=...` for content-service and for
     email-ingest each returned a valid `client_token` with `token_policies` correctly
     scoped to `["default", "content-service"]` / `["default", "email-ingest"]` respectively.
   - Used each resulting token to `vault kv get secret/jwt/public-key` — both succeeded.
   - Used each resulting token to `vault kv get secret/jwt/private-key` — both got
     `403 permission denied`, confirming the policy is correctly scoped to public-key only
     (matches the task's explicit note not to widen the policy).
   - Ran the script a second time against the same Vault/volume to check it doesn't error
     on re-run: it re-applied all four policies/AppRoles successfully (role_ids stable,
     secret_ids rotate on each run — this is pre-existing behavior shared by all four
     services, not something introduced by this change, since providing `VAULT_TOKEN`
     directly bypasses the script's "already configured, skip" branch that only triggers
     when `VAULT_TOKEN` is unset).
   - Cleaned up all test containers, the network, the volume, and the locally built test
     image afterward — nothing left running.

### Caveats / gaps
- This exercised **dev-mode** Vault (`vault server -dev`), not a real multi-key-share,
  sealed/unsealed production Vault. The `vault operator init`/`operator unseal` branch of
  the script (lines ~44-95, key-shares=5/threshold=3) was **not exercised** by this test
  run, because dev-mode Vault starts already initialized and unsealed. That code path is
  unchanged by this task's diff (this task only touches the policy/AppRole section), so the
  risk is low, but it means the full init-from-scratch-with-unsealing flow still has never
  been run against a real Vault — only the docs/scripts pattern review and file `sh -n`
  syntax check + prior team knowledge support it.
- Verified with the `hashicorp/vault:1.18` image pinned in `vault-init/Dockerfile`'s default
  `ARG VAULT_VERSION=1.18`, matching what prod actually builds.
