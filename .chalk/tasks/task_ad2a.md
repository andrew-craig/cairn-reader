---
id: task_ad2a
title: Mobile: prefetch silently retries an article with no extracted body on every run, forever
type: task
status: open
priority: 2
labels: [mobile,offline]
blocked_by: []
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-12T10:38:48Z
updated_at: 2026-09-12T10:38:48Z
---
Found during the task_de93 coverage check on `apps/mobile/src/services/articlePrefetch.ts`.

`runPool`'s worker (`articlePrefetch.ts:58-77`) does:
```ts
const detail = await ReadService.getContentById(id);
const { content } = ReadService.transformDetailToArticle(detail);
if (content) {
  await ArticleStore.saveBody(id, content);
}
```
`transformDetailToArticle` (`read.ts:519-534`) returns an `Article` with no `content`
whenever `userContent.content` is absent on the response (e.g. the backend's content
record hasn't finished processing, or extraction failed server-side). When that
happens, the `if (content)` branch is simply skipped: no error, no log, no store
write. The row's `body` stays `NULL`, and `content_hash` is untouched (it only ever
changes via a list sync). Since `ArticleStore.listPrefetchCandidates` selects
`WHERE is_read = 0 AND body IS NULL`, that same article is selected again on
**every future prefetch run**, forever, with no backoff, no failure counter, and no
user-visible signal — silent perpetual retry of an article that will plausibly never
succeed.

This branch is untested: every test in `articlePrefetch.test.ts` mocks
`transformDetailToArticle` to return a truthy `content` (`makeCandidate` + `body-${id}`
in the shared fixture), so the `if (content)` false path has zero coverage
(coverage report: `articlePrefetch.ts` line 66 uncovered branch).

## What a test should assert
- `ReadService.transformDetailToArticle` returns an `Article` with `content: undefined`
  for one candidate → `ArticleStore.saveBody` is **not** called for that candidate, no
  error is thrown, and the batch continues normally for the other candidates (matching
  today's actual behavior — this task is about closing the coverage gap and deciding
  whether "retry forever" is the intended behavior, not necessarily changing it
  silently).
- Whether repeat runs against the same still-empty article keep re-attempting it
  indefinitely (they do, today) — call this out explicitly as either accepted
  (document it) or a defect to fix (e.g. skip/log/mark it distinctly from "not yet
  fetched" so it doesn't burn a request on every sync forever).

Not a fix folded into task_de93 per that task's own scope — this is its own task.
