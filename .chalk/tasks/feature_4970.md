---
id: feature_4970
title: Reading time: compute word_count in backend, surface across all apps
type: feature
status: open
priority: 2
labels: [backend,read-service,mobile,web]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-06-12T13:23:16Z
updated_at: 2026-06-12T13:23:16Z
---
Reading time (estimated minutes) is currently non-functional: the frontend transforms compute it as ceil(word_count / 200), but the backend never produces a word_count. As a result it can never render against the real backend, so it was removed from the web app reading list (task_6305). This task implements word_count end-to-end so reading time can be shown in all apps.

NOT PART OF epic_f54e (desktop web app) — this is a cross-cutting backend + apps feature.

BACKGROUND (verified 2026-06-12):
- No word_count field in the Content model: services/read/content/internal/models/models.go.
- No word_count column in any migration: services/read/content/migrations/.
- The readability pipeline derives a plain-text character Length (pkg/rss/readability/readability.go) that is never propagated; no word counting anywhere.
- ContentResponse DTO (services/read/content/internal/api/dto/content.go) and the OpenAPI spec (services/read/content/api/openapi.yaml) have no word_count.
- Frontend already references it: apps/shared/src/types/read.ts (ContentResponse.word_count), apps/mobile transform+tests, and (until task_6305 follow-up) apps/web.

WHAT TO DO:
Backend (services/read/content):
- Add word_count (int) to the Content model and a DB migration adding the column (nullable).
- Compute word count during content extraction/processing (count words of the cleaned plain text; the readability Result already exposes Length — derive a word count alongside it) and persist it. Apply for both manual page adds and RSS-ingested content.
- Expose word_count in ContentResponse DTO (json tag) and document it in openapi.yaml.
- Decide on backfill for existing rows (e.g. lazy compute on read, or a one-off backfill); existing content currently has none.

Shared (apps/shared):
- ContentResponse.word_count already typed — confirm it matches the backend (int/optional).

Mobile (apps/mobile):
- transformToArticle already sets readingTime from word_count; confirm display where intended (article detail/list).

Web (apps/web):
- Re-add reading time to transformToArticle (services/read.ts) and the ArticleRow meta line (components/ArticleRow.tsx) — this reverts the removal made in the task_6305 follow-up.

VERIFICATION:
1. Backend unit/integration test: extracting/adding a content-rich page yields a word_count > 0; it is returned by GET /api/v1/content/user/{userId}.
2. Adding an article via the API against a running instance returns word_count in the response.
3. Web reading list shows 'N min read' for articles with word_count; mobile shows reading time where designed.
4. Items without extractable text (e.g. RSS stubs) have null/absent word_count and degrade gracefully (no reading time shown).
