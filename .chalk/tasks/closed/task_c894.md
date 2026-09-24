---
id: task_c894
title: Mobile: a live 5xx on a mutation is dropped, but the same 5xx on an outbox replay is retried
type: task
status: closed
priority: 2
labels: [mobile,offline]
blocked_by: []
parent: 
remote_task_url: null
created_at: 2026-09-12T10:45:39Z
updated_at: 2026-09-23T07:53:03Z
---
Found by the tech lead reviewing feature_90a5 before closing it out. Same bug class as
task_cab7, task_c87c, task_5bd6 and task_f19d — see the 2026-09-12 entry in
`LEARNINGS.md`, which is about exactly this: fixing the throw site does not fix the
catch site.

## The inconsistency

The two halves of the offline write path classify an identical `HttpError(5xx)`
differently.

`outbox.ts`'s `sendRow` treats 5xx as **transient** — `halt`, keep the row, bump
`attempts`, retry on the next drain:
```ts
if (error instanceof HttpError && error.status !== 401 && error.status < 500) {
  return 'drop';
}
// NetworkError, HttpError(401/5xx), or anything else unrecognized.
return 'halt';
```

`articleMutations.ts`'s `withOutboxOnNetworkError` treats the same 5xx as
**definitive** — only `NetworkError` is queued, everything else rethrows:
```ts
if (error instanceof NetworkError) {
  await Outbox.enqueue(articleId, field, payload);
  return;
}
throw error;
```

`ReadService.updateUserContent` and `deleteUserContent` both throw
`HttpError(response.status, ...)` (`read.ts:179`, `read.ts:209`), so a live 5xx reaches
that `throw error` — after the local store write has already landed.

## Why it matters

Online, backend returns 503 on archive: `ArticleStore.remove()` has already run, the
DELETE is rethrown, and **nothing is queued**. The article is gone from the device and
still present on the server. Same shape for `markCompleted`/`setFavorite`/
`saveScrollPosition`: the store shows the new value, the server never saw it, and no
outbox row exists to freeze the column, so the next list sync silently reverts the
user's action.

Not data loss — it self-heals on the next sync — but the user sees an action take
effect and then undo itself, and it is the one case the outbox was built for that the
outbox never sees.

## Scope to decide

Whether `withOutboxOnNetworkError` should queue on the same set `sendRow` calls
retryable (NetworkError, 401, 5xx) rather than on `NetworkError` alone. If so, the
predicate belongs in one place shared by both modules, not duplicated — the duplication
is what let the two drift apart.

Note a 401 is already handled differently on purpose (`fetchWithAuth` retries once
internally), so confirm that case rather than assuming symmetry.

→ verify: a unit test asserting a live `HttpError(503)` on each `ArticleMutations`
method leaves a queued outbox row; the existing rethrow-on-4xx tests still pass.

## Review

**Root cause confirmed:** `outbox.ts`'s `sendRow` and `articleMutations.ts`'s
`withOutboxOnNetworkError` each independently classified the same `HttpError`/
`NetworkError` types, and had drifted: `sendRow` treated `NetworkError`,
`HttpError(401)`, and `HttpError(5xx)` as retryable; `withOutboxOnNetworkError`
treated only `NetworkError` as retryable, rethrowing everything else — including a
live 5xx — after the local store write had already landed.

**Fix:**
- Extracted the classification into one exported predicate,
  `isRetryable(error: unknown): boolean`, in `apps/mobile/src/services/outbox.ts`.
  It returns `false` only for a definitive 4xx other than 401 (a direct extraction
  of `sendRow`'s existing `drop` condition, inverted) — `true` for `NetworkError`,
  `HttpError(401)`, `HttpError(5xx)`, or anything unrecognized.
- `sendRow` now calls `isRetryable` instead of inlining the check; behavior
  unchanged (verified — all pre-existing `outbox.test.ts` cases still pass
  unmodified).
- `articleMutations.ts` imports `isRetryable` from `./outbox` and its helper is
  renamed `withOutboxOnRetryableError` (the old name became misleading once it
  queues on more than `NetworkError`) to call `isRetryable(error)` instead of
  `error instanceof NetworkError`. The `NetworkError` import is no longer needed
  there and was removed.
- Updated the doc comments on both functions and the `ArticleMutations`/project-tree
  entries in `apps/mobile/AGENTS.md` (symlink target of `apps/mobile/CLAUDE.md`) to
  describe the corrected behavior instead of "queues on `NetworkError` only."

**401 handling — investigated, not mirrored blindly:** traced the actual throw path
instead of assuming symmetry with `sendRow`. `AuthService.fetchWithAuth` (shared,
`apps/shared/src/services/auth.ts`) already retries once internally on a live 401;
if that retry also fails, it clears tokens and throws a plain
`Error('Session expired. Please log in again.')` — not an `HttpError(401)`. Since
`ReadService.updateUserContent`/`deleteUserContent` call `fetchWithAuth` (not a raw
fetch), an `HttpError(401)` can never actually reach either `sendRow` or
`withOutboxOnRetryableError` from a real request today. `isRetryable` still treats
`HttpError(401)` as retryable for consistency with `sendRow`'s existing documented
contract (defensive, in case that ever changes) — but no behavior change results
from it in practice, and the "session expired" plain `Error` is intentionally left
out of scope: it doesn't match `NetworkError` or `HttpError`, so it still rethrows
exactly as before. Queuing writes behind a dead session was not part of this bug's
scope.

**Verification:**
- Wrote failing tests first: added an `HttpError(503)` case to each
  `ArticleMutations` method's describe block in `articleMutations.test.ts`
  (`markCompleted`, `markReading`, `saveScrollPosition`, `setFavorite`, `archive`).
  Confirmed all 5 failed against the pre-fix code (`git stash` the two source
  files, rerun — 5 failed / 13 passed), each failing with the live `HttpError`
  rejecting the promise instead of resolving with a queued row.
- Applied the fix (`git stash pop`), reran: all 18 tests in
  `articleMutations.test.ts` and all 13 in `outbox.test.ts` pass (31/31).
- Full mobile suite: `npx jest` → 37 suites / 278 tests, all passing (one
  `LoginScreen.test.tsx` timeout on the first full-suite run was reproduced as
  flaky under parallel load — passed both standalone and on a subsequent full-suite
  rerun, and passed identically on the unmodified tree, so unrelated to this
  change).
- `npm run type-check` (`tsc --noEmit`): clean.
- `npx eslint` on the four changed source/test files: clean.

**Files changed:**
- `apps/mobile/src/services/outbox.ts` — extracted and exported `isRetryable`.
- `apps/mobile/src/services/articleMutations.ts` — use shared `isRetryable`,
  renamed `withOutboxOnNetworkError` → `withOutboxOnRetryableError`.
- `apps/mobile/src/services/articleMutations.test.ts` — updated mock to expose the
  real `isRetryable` via `jest.requireActual` alongside the mocked `Outbox`; added
  5 new `HttpError(503)` enqueue tests; updated header comment.
- `apps/mobile/AGENTS.md` (target of `apps/mobile/CLAUDE.md` symlink) — corrected
  the `ArticleMutations` description and project-tree comment.
- `LEARNINGS.md` — added a 2026-09-20 entry documenting this as a sixth instance of
  the "throw site fixed, catch site drifted" bug class.

Branch: `task_c894`. Pushed as PR #401.

## Follow-up (2026-09-20): fixed a regression flagged by automated review

The `magpie-reviewer` bot on PR #401 caught a real bug: `isRetryable` was defined
as `!(error instanceof HttpError && error.status !== 401 && error.status < 500)`
— a literal inversion of `sendRow`'s old drop-check — which returns `true` for
**any** non-`HttpError`, not just `NetworkError`. `withOutboxOnRetryableError`
therefore silently enqueued the plain `Error('Session expired...')` that
`fetchWithAuth` throws when its own internal 401 retry fails, instead of
rethrowing it as the PR's own doc comments and LEARNINGS entry claimed. No
existing test covered a plain-Error/session-expired rethrow, so it slipped
through review.

**Fix:** narrowed `isRetryable` to a positive allowlist (`NetworkError`,
`HttpError(401)`, `HttpError(5xx)`) instead of an inverted drop-check.
`sendRow`'s drop-check now reads `error instanceof HttpError && !isRetryable(error)`
— the added `instanceof HttpError` guard keeps its own "unrecognized → halt"
fallback intact, since `sendRow` and `withOutboxOnRetryableError` genuinely want
different policy for an error neither one recognizes (halt-and-retry vs.
rethrow), which was never something they needed to share.

Verified the regression test fails against the pre-fix predicate (confirmed by
temporarily reverting it) and passes after. Full suite: 279/279 tests, `tsc
--noEmit` and `eslint` clean. Pushed as a follow-up commit (`38f6a9a`) on the
same PR #401 branch.

**Landed:** merged to `main` as #401. Closed during backlog housekeeping on 2026-09-23.
