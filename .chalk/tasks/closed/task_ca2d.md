---
id: task_ca2d
title: [Audit/Tier 4] Consolidate mobile's two private fetchWithAuth copies onto mobile's own AuthService
type: task
status: closed
priority: 2
labels: [quality,consolidation,audit,mobile]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T12:52:21Z
updated_at: 2026-09-05T09:20:14Z
---
**Source:** Cairn Simplification Audit — https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f · file:line detail supplied by the audit author 2026-08-17 and re-verified at HEAD `a6c56a1`.
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. One finding, one branch, one PR.

**Audit tier:** 4 (frontend) | **Verified.**

## Problem
This is **intra-mobile** duplication. Two service modules each carry a private copy of the authenticated-fetch policy, **byte-identical but for two comment lines**:
- `apps/mobile/src/services/read.ts:25-93` — private static `fetchWithAuth` (:25) + `fetchWithAuthAndRetry` (:81)
- `apps/mobile/src/services/explore.ts:61-127` — the same pair

Web already solved this: `apps/web/src/services/auth.ts:312` carries the shared implementation, and its comment says it "mirrors mobile fetchWithAuth" — i.e. web mirrors a policy mobile itself duplicates.

## What to do
1. Consolidate both copies onto mobile's own `AuthService`, matching what web already does. Three implementations become one.
2. Delete both private copies in the same PR as the last repoint.

## ⚠️ Hazard — error message text is load-bearing
`apps/mobile/src/utils/retry.ts:22-36` decides retryability by **matching on the error message string** (lowercased substring match):
```ts
const msg = error.message.toLowerCase();
if (msg.includes('not authenticated') || msg.includes('session expired') || ...) return false;
```
So the substrings `Session expired. Please log in again.` and `Not authenticated` must survive consolidation **intact**. If the shared implementation rephrases either message, those auth errors silently become **retryable** — the client will retry a request that can never succeed, instead of prompting re-login.

Add a test pinning the message-to-retryability contract before you refactor. (That the contract is expressed as string matching at all is worth a separate note — but do not fix it here.)

## Done when
- One authenticated-fetch implementation on mobile, both private copies deleted, and a test proves auth errors are still classified non-retryable.

## Sequencing vs task_47c1 — do this one first
**task_47c1** ([FE auth layer] move the duplicated web/mobile `auth.ts` into `apps/shared`) is the *cross-platform* move; this is the *intra-mobile* collapse. They are adjacent, not the same ground. Landing this first means **task_47c1 moves one implementation instead of three**. The retry.ts hazard above applies to task_47c1 as well.
