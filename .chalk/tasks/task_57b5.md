---
id: task_57b5
title: [H11] Delete the ~250-line dead JWT-validation duplicate in services/users/internal/auth/jwt.go
type: task
status: open
priority: 2
labels: [quality,wave4,consolidation]
blocked_by: [task_41e2]
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:53:56Z
updated_at: 2026-08-17T21:31:20Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** H11 | **Wave 4** | **Recipe:** R11 (strategy §2.5)
**Touches:** services/users/internal/auth/jwt.go (delete), its tests

## Problem
`services/users/internal/auth/jwt.go` duplicates roughly 250 lines of `pkg/auth/validator.go`, and the duplicate is **dead in the request path** — only tests call it. Its passing tests give false confidence, and security fixes to the canonical validator will never propagate to it.

## What to do
1. Confirm on current main that nothing in the request path imports it (grep, and check the build still passes with it removed).
2. Port any assertion the duplicate's tests make that `pkg/auth`'s tests do not already cover, onto `pkg/auth`.
3. Delete the duplicate and its tests **in the same PR**.

## Done when
- Only `pkg/auth/validator.go` implements JWT validation; the users service builds and tests pass.

---

## Re-confirmed by the Cairn Simplification Audit (2026-08-17) — with a correction to step 3

This task is the audit's **"a validator fork"** item (Tier 3). Re-verified at HEAD `a6c56a1`.
**Audit report:** https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f

**⚠️ Step 3 as written is unsafe. Do not delete `jwt.go` wholesale — only the validation half is dead.**

`JWTManager`'s **signing / key-management half is live** and deleting the file would break the users service and `task_41e2`'s fix surface. Confirmed non-test caller counts at `a6c56a1`:

| Method | Non-test callers | Status |
|---|---|---|
| `ValidateToken` | 0 | dead |
| `GetTokenInfo` | 0 | dead |
| `ExtractTokenFromHeader` | 0 | dead |
| `GetKeyID` | 0 | dead |
| `ParseTokenWithoutValidation` | 0 | dead |
| **`GetPublicKey`** | **1** | **live** — `internal/handlers/router.go:58` |
| **`UpdateKeys`** | **1** | **live** — the rotation callback in `cmd/user-service/main.go` |

So the deletion is **method-level, not file-level**: remove the five dead validation methods and their tests, keep the signing and key-management surface.

**Blocked by task_41e2 (enforced 2026-08-17, owner decision).** That task fixes rotation never reaching the users service's own validator, and its recommended fix routes the rotation callback through `GetPublicKey`/`UpdateKeys` into `pkg/auth.Validator.UpdatePublicKey`. Deleting those two would remove the seam it needs, so the ordering is now a hard dependency rather than a coordination note: **land task_41e2 first**, then prune around the settled surface.

Re-derive the live/dead split after task_41e2 lands — its fix may change which methods have callers, and the table above is a snapshot from before it.

