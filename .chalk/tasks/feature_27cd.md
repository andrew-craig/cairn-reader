---
id: feature_27cd
title: Extraction feedback: let users report bad article extraction, keep raw source for reproduction
type: feature
status: open
priority: 2
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-11-07T20:22:32Z
updated_at: 2026-11-07T20:22:32Z
---


## Goal

Users can flag an article whose extraction went wrong (body, images, formatting, title, author, description, lead image). The owner can then reproduce the exact failure from the original input and turn it into a readability test fixture. Everything stays inside the Content Service and the local database (also in self-host). No outside services. UI is out of scope.

## Context

- All extraction runs in `ContentProcessor.ProcessHTML` (`services/read/content/internal/processor/content.go`): readability (body, title, author, excerpt, lead image), then sanitizer, hash and URL canonicalization. Today only the output is stored and the raw input is thrown away.
- When readability fails, the raw HTML is sanitized and stored with no flag, so that failure path is invisible.
- `contents` rows are shared across users (dedup). Rows can be rewritten (`UpdateContent`) and deleted (orphan cleanup after 90 days). So a feedback report has to carry its own copy of the evidence, not just point at `contents`.

## Design

1. **`content_sources`** (new table, 1:1 with `contents`, `ON DELETE CASCADE`): `content_id` PK, `raw_html` (5MB check), `readability_failed BOOL`, `created_at`. Written in the same transaction as the new `contents` row on every create path (`CreateFromURL`, `CreateFromHTML`, `BulkCreateFromHTML`). Replaced on `UpdateContent`. Not written on a dedup hit.
2. **Retention:** the worker's cleanup job deletes `content_sources` rows older than `RAW_SOURCE_RETENTION` (duration, default `720h` = 30 days), in batches, like the existing cleanup steps.
3. **`content_feedback`** (new table, no FK to `contents`, so it survives content deletion and rewrites):
   `id, user_id, content_id, categories TEXT[] (≥1, from a fixed set), comment (optional, ≤2000 chars), status ('open'|'resolved'), created_at, updated_at`,
   plus a snapshot taken when the report is submitted: `original_url, title, author, description, image_url, cleaned_html, raw_html, readability_failed`. `raw_html` and `readability_failed` are both nullable and both NULL when the `content_sources` row has already expired.
   `UNIQUE(user_id, content_id)`: resubmitting updates the existing report, refreshes the snapshot and **sets status back to `open`**, so a re-report after a fix reaches triage.
   **Snapshot retention:** resolving a report sets `raw_html` and `cleaned_html` to NULL (the export has already written them to disk). Only metadata, categories and comment are kept. So private email bodies are kept only while a report is open.
   **Exposure:** `raw_html` is unsanitized input. No user-facing endpoint returns `raw_html` or `cleaned_html` from this table. Only the owner's triage CLI reads them.
4. **Categories** (cover every extraction output): `body_missing_content`, `body_extra_content`, `images`, `formatting`, `links`, `title`, `author`, `description`, `lead_image`, `other`.
5. **API** (under the existing `/api/v1/content/user/{user_id}` group, which already enforces that `{user_id}` matches the token):
   - `PUT /{content_id}/feedback`: upsert a report. 404 if the user doesn't have the content. 400 on an empty or unknown category or an over-long comment.
   - `GET /{content_id}/feedback`: the user's own report (categories, comment, status, timestamps only, no snapshot), or 404. The UI will need it to show "reported".
6. **Triage** (layer 2, separate follow-up task): a `cmd/feedback` CLI in the Content Service: `list` (open reports), `export <id> <dir>` (writes `raw.html`, `output.html`, `meta.json` ready to become a fixture), `resolve <id>` (sets status and clears the snapshot bodies, see 3). Until then, triage with SQL.

## Plan

- [ ] Migration `000007_extraction_feedback` (up/down) for both tables plus index `content_sources(created_at)` → verify: migrate up/down in integration test DB
- [ ] `ProcessedContent` gains `RawHTML`, `ReadabilityFailed`; set in `ProcessHTML` → verify: unit test for the success and fallback paths
- [ ] Repository: insert/replace the source in the same tx as the content create/update/bulk create; `DeleteExpiredSources(olderThan, batch)` → verify: integration tests (create stores source, dedup hit doesn't overwrite, update replaces, expiry deletes only old rows)
- [ ] Cleanup job: add the source expiry step; worker reads `RAW_SOURCE_RETENTION` → verify: `cleanup_job_test.go`
- [ ] Feedback repository + service (snapshot from `contents` + `content_sources`, ownership check via `user_contents`) → verify: integration tests incl. expired source → `raw_html` and `readability_failed` NULL, resubmit → upsert and reopen a resolved report
- [ ] Handlers, routes, DTO validation → verify: handler tests (400/404/200, cross-user access rejected, GET response carries no snapshot fields)
- [ ] Docs: `content/api/openapi.yaml`, `services/read/CLAUDE.md`, `docs/CONFIGURATION.md` (new var), self-host and compose env forwarding for `RAW_SOURCE_RETENTION` (see bug_a701 drift)
- [ ] `go test ./...` and integration tests green
- [ ] Follow-up task: triage CLI (layer 2)

## Tradeoffs / open points

- Reports on items older than 30 days have no raw HTML (common for long-lived Reads items). Triage falls back to re-fetching `original_url`. The retention clock starts when the content row is first created, not when it reaches this user: an item deduped against a row another user created more than 30 days ago (a shared RSS article, or an email matched on hash + URL) arrives with no source row even though it's new to this user. A longer retention fixes that at the cost of storage and of keeping private email content longer.
- Storage: raw HTML is usually a few times the size of the cleaned output. Postgres compresses large text automatically.
- The fallback-path flag (`readability_failed`) is also useful on its own, e.g. `SELECT count(*) ... WHERE readability_failed`.
