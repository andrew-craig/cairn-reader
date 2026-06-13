---
id: bug_d4ff
title: Reconcile ContentResponse type/transform drift with backend (image_urls, source_type)
type: bug
status: closed
priority: 2
labels: [read-service,web,mobile,shared]
blocked_by: []
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-12T13:46:30Z
updated_at: 2026-06-12T22:16:18Z
---
The frontend ContentResponse type and the transformToArticle mappings don't match what the Read/Content service actually returns, so several mapped fields are permanently inert — most visibly, lead images never render in the reading list.

Parent: epic_f54e (surfaces in the web reading list built in task_6305).

DRIFT (verified 2026-06-12):
Backend DTO services/read/content/internal/api/dto/content.go (ContentResponse) actually emits:
  id, content_hash, cleaned_html, original_url, canonical_url, title, author,
  published_at, description, image_urls (string[]), source_type, source_feed_id,
  created_at, updated_at.
But apps/shared/src/types/read.ts (ContentResponse) instead declares fields the backend never sends:
  - excerpt        -> not emitted (backend only has description)
  - site_name      -> not emitted
  - favicon_url    -> not emitted
  - lead_image_url -> not emitted (backend sends image_urls: string[])
  and omits the real fields image_urls and source_type. (word_count is tracked separately in feature_4970.)

CONSEQUENCES (apps/web + apps/mobile transformToArticle):
  - imageUrl: content.lead_image_url            -> always undefined; should be content.image_urls?.[0]
  - description: content.description || content.excerpt  -> excerpt branch is dead
  - author: content.author || content.site_name -> site_name branch is dead
The OpenAPI spec services/read/content/api/openapi.yaml is also stale: its ContentResponse documents source_url (DTO emits original_url) and omits description/image_urls/source_type/content_hash.

WHAT TO DO:
- Align apps/shared/src/types/read.ts ContentResponse to the real DTO: replace lead_image_url with image_urls: string[]; add source_type; drop excerpt/site_name/favicon_url (or, if richer extraction is genuinely wanted, file a separate backend enhancement rather than leaving phantom fields).
- Fix transformToArticle in BOTH apps/web/src/services/read.ts and apps/mobile/src/services/read.ts: map imageUrl from image_urls?.[0]; remove the dead excerpt/site_name fallbacks. Update apps/mobile/src/services/read.test.ts accordingly.
- Update the OpenAPI ContentResponse schema to match the DTO (original_url, description, image_urls, source_type, content_hash).

VERIFICATION:
1. apps/shared, apps/web, apps/mobile type-check clean after the type change.
2. apps/mobile read.test.ts passes with the corrected mapping.
3. Against cairn.seatrain.net: a content item that has images returns image_urls, and the web reading list (and mobile) render its lead image.
4. No frontend code references lead_image_url/excerpt/site_name/favicon_url after the change (grep clean).
