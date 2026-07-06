---
id: task_179f
title: Mobile: fix archive semantics (hard delete vs status, swallowed errors, dual caches)
type: task
status: open
priority: 2
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-07-05T06:45:55Z
updated_at: 2026-07-05T06:45:55Z
---

Found while investigating the "archived article still shows in Read list"
bug (fixed separately). Out of scope for that fix but worth addressing:

- `ReadArticleDetailScreen.handleArchive` calls `ReadService.deleteUserContent`
  (hard `DELETE /api/v1/content/user/{userId}/{contentId}`) instead of
  `updateUserContent(id, { status: 'archived' })`, even though the shared
  `ContentStatus` type (`apps/shared/src/types/read.ts`) already models
  `'archived'` as a status. Confirm intended behavior — should archiving
  actually delete the user-content association, or just change its status?
- In the same function, a failed backend delete is silently swallowed
  (`console.error` only) and `navigation.goBack()` still runs — the user is
  told the archive succeeded even if the server call failed, so client and
  server state can diverge.
- `apps/mobile/src/services/storage.ts` has two disconnected local article
  caches (`ARTICLES_KEY` used by `getArticles`/`saveArticles`/`deleteArticle`,
  and `READ_LIST_CACHE_KEY` used by `getReadListCache`/`saveReadListCache`).
  `ReadScreen` only reads/writes the latter; `handleArchive` only touches the
  former. Worth reconciling into a single source of truth.
