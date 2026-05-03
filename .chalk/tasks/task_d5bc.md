---
id: task_d5bc
title: Remove legacy cleaned_html fallback in outbox_worker (post-task_7479 drain cleanup)
type: task
status: open
priority: 3
labels: []
blocked_by: []
parent: epic_6d46
remote_task_url: null
created_at: 2026-05-02T21:57:43Z
updated_at: 2026-05-03T04:13:27Z
---

# Remove legacy cleaned_html fallback in outbox_worker

## Why
task_7479 changed the fetcher to write `raw_html` into the outbox payload, but kept a
backward-compat fallback in `services/read/fetcher/internal/worker/outbox_worker.go::buildContentItem`
that also accepts the pre-task_7479 `cleaned_html` key. This was intentional so in-flight
outbox entries written by the old fetcher version would still drain after deployment.

Once the queue is fully drained (no `content_outbox` rows produced before the task_7479
deploy remain), the fallback becomes dead code and should be deleted to keep the payload
contract crisp and avoid permanent ambiguity over which key wins.

## When to do this
After task_7479 is deployed AND the legacy queue has drained. Verify with:

```sql
-- All pending/sending entries should be from after the task_7479 deploy timestamp.
SELECT MIN(created_at), COUNT(*)
FROM content_outbox
WHERE delivery_status IN ('pending', 'sending');
```

If `MIN(created_at)` is after the task_7479 deploy time (or the table is empty for
those statuses), the fallback can be removed safely.

## What to remove
1. **`services/read/fetcher/internal/worker/outbox_worker.go::buildContentItem`** —
   delete the `if html == "" { html, _ = payload[models.PayloadKeyCleanedHTML].(string) }`
   block. Simplify the doc comment and the error message to reference only `raw_html`.
2. **`services/read/fetcher/internal/models/outbox_payload.go`** — delete
   `PayloadKeyCleanedHTML` and the doc note about the legacy key.
3. **`services/read/fetcher/internal/worker/outbox_worker_test.go`** — delete:
   - `TestOutboxWorker_BuildContentItem_FallsBackToLegacyCleanedHTML`
   - `TestOutboxWorker_BuildContentItem_PrefersRawHTMLOverLegacyKey`

The producer-side regression in
`services/read/fetcher/internal/processor/item_processor_test.go::TestItemProcessor_OutboxPayloadCarriesRawHTML`
that asserts `cleaned_html` is *not* in the new payload should stay — it's still useful
as a sanity check that the producer never reverts to the old shape.

## Acceptance
- `grep -ri cleaned_html services/read/fetcher/` returns no hits other than the
  producer-side regression assertion noted above.
- All existing unit and integration tests pass.
- A new outbox entry is processed end-to-end (covered by existing tests).

## Risk
Low. Only affects the fetcher's outbox consumption path. If a stale legacy entry
sneaks through after this lands, it will fail with a clear "missing 'raw_html' field"
error and end up in the failed-entries log; redelivery is straightforward.
