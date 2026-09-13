---
id: task_c894
title: Mobile: a live 5xx on a mutation is dropped, but the same 5xx on an outbox replay is retried
type: task
status: open
priority: 2
labels: [mobile,offline]
blocked_by: []
parent: 
remote_task_url: null
created_at: 2026-09-12T10:45:39Z
updated_at: 2026-09-12T22:37:45Z
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
