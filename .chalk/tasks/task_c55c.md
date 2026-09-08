---
id: task_c55c
title: Mobile: prefetch article bodies and read from the local store when offline
type: task
status: in_progress
priority: 2
labels: [mobile,offline]
blocked_by: []
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-05T23:36:10Z
updated_at: 2026-09-08T23:06:27Z
---
Phase 3 of feature_90a5 (the user-facing offline reading). After each successful list sync, download cleaned_html for unread/reading articles that lack a body (bounded concurrency, newest first, capped at the 100 most recent Read-list articles) into the store. Explore article content is never synced. Diff by content.content_hash from the list page (present on every list item) so unchanged bodies are never re-downloaded. Detail screen resolves the article by id from the store (not only route params), renders the stored body immediately, and refreshes online when the hash changed. When offline with no stored body, show a clear 'Not available offline' state instead of a blank body. Prefetch runs after list sync on focus, pull-to-refresh, app foreground and reconnect; keep it in a service, not in the screens. Evict bodies on archive/delete and beyond the cap. Images remain remote (out of scope).

## Scope clarification (tech lead, 2026-09-08)
Verified against `main` at 896cd80. Phase 1 (task_a8a4, #382) and Phase 2 (task_c87c,
#383) have landed, plus chore_1089 (#385). `ArticleStore`, `NetworkError`,
`useNetworkStatus`, `isOffline()` and the offline banner all exist. Build on them.

### Blocker the task description does not name: the client throws `content_hash` away
`ReadService.transformToArticle` (`read.ts:494-511`) and `transformDetailToArticle`
(`:534-550`) never copy `content.content_hash` onto `Article`. The shared `Article` type
(`apps/shared/src/types/article.ts`) has no such field and the `articles` table has no
such column. Every part of "diff by `content_hash` so unchanged bodies are never
re-downloaded" is unimplementable until that value is plumbed through. Do this first.

### Design decisions (made here — do not re-litigate; raise it if you disagree)

**1. Add `contentHash?: string` to the shared `Article` type.** One optional field in
`apps/shared/src/types/article.ts`, populated by both transforms and stored in a new
`content_hash` column. Optional, so web (which does not use it) is unaffected. Do not
build a side-channel id→hash map to avoid touching the shared model: the hash is part of
the content model, and `articleStore` already maps `Article`↔row in exactly one place.

**2. The schema needs a real migration, not a wider `CREATE TABLE`.** `getDb()`
(`articleStore.ts:60-90`) runs `CREATE TABLE IF NOT EXISTS`. Phase 1 has already shipped,
so installs exist carrying the v1 table; widening the CREATE statement is a **no-op**
against them and every query naming `content_hash` then fails with `no such column`.
Add a `PRAGMA user_version`-driven migration applied inside the same `getDb()` open.
Keep the v1 CREATE at its current shape and add this change as numbered step 2
(`ALTER TABLE articles ADD COLUMN content_hash TEXT`) so a fresh install and an upgraded
install converge on the same schema by the same path. This is the migration mechanism
for every future phase (task_ebf1's outbox table is next), so get the shape right.
→ verify: a test seeds a DB with the exact v1 schema, runs the store, and asserts the
column exists and pre-existing rows survive; a second open is a no-op. Note
`__mocks__/expo-sqlite.js` caches one in-memory DB per name, so seeding v1 means opening
`cairnreader.db` through the mock before the store's first call, with `jest.resetModules()`
to clear the module-level `dbPromise`.

**3. "unread/reading" is `is_read = 0`. Do not add a status column.** `Article` collapses
`ContentStatus` into `isRead = status === 'completed'` (`read.ts:506`). `unread` and
`reading` are both `isRead: false`, and `archived` rows never reach the Read list, so
`WHERE is_read = 0` is exactly the set this task asks for. Widening `Article` to carry
the four-state status is a larger change than this feature needs.

**4. Prefetch is a service triggered from the existing sync callback.** New
`src/services/articlePrefetch.ts`. `ReadScreen`'s `onResetLoaded` (`ReadScreen.tsx:37-42`)
already fires after every successful first-page load — focus-with-expired-TTL,
pull-to-refresh and Retry all route through it. Call the prefetcher from there, after
`upsertMany`. No screen learns anything about prefetching beyond that one call. Do **not**
call it from `BookmarksScreen.onResetLoaded` (`BookmarksScreen.tsx:25-28`): Read list only.

**5. App-foreground and reconnect triggers are deferred to task_ebf1 (Phase 4).** The task
text lists four triggers. Two of them need an `AppState`/network-change listener that does
not exist in this app today (`grep AppState src/` → nothing), and Phase 4 needs the same
"we're back — drain now" trigger for the outbox. Building it here means shipping a listener
whose only consumer is prefetch, then rebuilding around it one phase later. Ship the two
triggers that come free with the existing sync path; Phase 4 adds the listener and wires
both consumers to it. Record the gap explicitly in the review section: until Phase 4, a
user who regains connectivity must open or pull-to-refresh the Read tab (30s TTL) before
prefetch runs.

**6. Bounded concurrency is a small fixed worker pool.** 3 concurrent `getContentById`
calls, hand-rolled. No new dependency (no `p-limit` or similar).

**7. Never prefetch while offline; abort the batch on the first `NetworkError`.** Guard the
run with `isOffline()` (`utils/network.ts`). A `NetworkError` mid-batch means the connection
went away — abort the remaining work rather than burning 100 failing requests. A
non-network error on one article skips that article and continues.

**8. Single-flight.** A run already in progress must not be restarted by a second sync.
Module-level in-flight guard; a second call while one is running is a no-op, not a queued
second run.

### Selection, staleness and eviction
By the time the prefetcher runs, `upsertMany` has already written the list page's fresh
hash, so the store can no longer compare stored-hash against fresh-hash after the fact.
Resolve that inside `upsertMany`: when the incoming `content_hash` differs from the stored
one, write the new hash **and set `body = NULL`** — a body whose hash no longer matches is
wrong, and keeping it means serving stale text offline. That reduces the prefetch query to
one statement: the 100 most recent rows by `added_at` where `is_read = 0 AND body IS NULL`.
Take this shape unless you find something wrong with it; if you do, say so before building
an alternative.

This changes the `COALESCE` rule at `articleStore.ts:54`, which exists so a list refresh
(no `cleaned_html` in summaries) cannot erase a cached body. Preserve that behaviour when
the incoming hash is absent or equal; drop the body **only** on a genuine hash change.
→ verify: a test pins both halves — refresh with the same hash keeps the body, refresh
with a changed hash clears it, refresh with no hash keeps it.

Eviction: after a run, `UPDATE articles SET body = NULL` for rows outside the 100 most
recent by `added_at`. Archive/delete already removes the whole row (`ArticleStore.remove`);
do not change it.

### Detail screen (`ReadArticleDetailScreen.tsx`)
- Resolve the article by id from the store rather than trusting route params alone, and
  merge the two: route params carry the fresh list metadata (including the fresh hash),
  the store carries the body and the stored hash.
- Stored body present **and** stored hash equals the route-param hash → render it and do
  **not** call `getContentById` at all. That is the point of the hash diff; today the
  screen fetches on every open (`:54`).
- No stored body, or hashes differ → fetch as it does now and `saveBody`.
- Offline (`useNetworkStatus`) with no stored body → a clear "Not available offline" state.
  Today the catch at `:62-66` sets `contentLoading = false` and renders an empty article.
- The existing `ReadArticleDetailScreen.test.tsx` assertions must keep passing.

### Out of scope — do not build
Image caching. Explore prefetch (Explore stays online-only — do not touch `explore.ts` or
`EXPLORE_CACHE_KEY`). Any backend change. A per-article "download" button. Paginating from
SQLite (task_a8a4 decision 3 still holds). The offline mutation outbox (task_ebf1).

### Definition of done
- [ ] `npx jest`, `npm run type-check`, `npm run lint` clean from `apps/mobile`
      (12 pre-existing lint warnings are the baseline — add none)
- [ ] `npm run build` clean in `apps/shared` (the `Article` change touches it)
- [ ] New unit tests: v1→v2 migration; hash-change body invalidation (all three cases);
      prefetch selection; cap eviction; single-flight; offline guard; batch abort on
      `NetworkError`
- [ ] Detail-screen tests: renders the stored body with **no** network call when hashes
      match; "Not available offline" when offline with no stored body
- [ ] Review section filled in on this task file, including the Phase 4 trigger gap
- [ ] One branch, one PR, on `claude/feature-90a5-oversight-g1fkda`
