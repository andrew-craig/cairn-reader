---
id: task_ccf3
title: Offline mobile: read cached article bodies in the reader screens (piece 4/7)
type: task
status: open
priority: 3
labels: [mobile,offline]
blocked_by: [task_c73b]
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-05T23:13:48Z
updated_at: 2026-09-05T23:14:18Z
---
Offline reader for feature_90a5.

Scope:
- ReadArticleDetailScreen (apps/mobile/src/screens/ReadArticleDetailScreen.tsx:42-58): before calling ReadService.getContentById, check OfflineStore for cleaned_html by content_id and render it if present. Only hit the network when the body is missing and the device is online.
- When offline AND the body is not cached: show a clear 'Not available offline' empty state instead of the current blank body / infinite spinner.
- Resolve the article by id from OfflineStore, not only from React Navigation route params, so an offline deep link / resumed session into a detail screen works.
- ExploreArticleDetailScreen: same by-id resolution from OfflineStore (Explore bodies are cached in piece 3); 'Not available offline' state for uncached.
- react-native-render-html renders remote <img> as-is — images will be broken offline; that is the accepted 'text-only offline' scope (see feature_9d64 for image caching).

Verify: cached article opens with full body offline; uncached article offline shows the 'Not available offline' state; online behaviour unchanged; existing ReadArticleDetailScreen / ExploreScreen tests still pass; new tests for the offline branches.

Blocked by: task_c73b (piece 3).
