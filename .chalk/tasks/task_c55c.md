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
- [x] `npx jest`, `npm run type-check`, `npm run lint` clean from `apps/mobile`
      (12 pre-existing lint warnings are the baseline — add none)
- [x] `npm run build` clean in `apps/shared` (the `Article` change touches it) — see
      note below, there is no `build` script; ran `npx tsc --noEmit` instead
- [x] New unit tests: v1→v2 migration; hash-change body invalidation (all three cases);
      prefetch selection; cap eviction; single-flight; offline guard; batch abort on
      `NetworkError`
- [x] Detail-screen tests: renders the stored body with **no** network call when hashes
      match; "Not available offline" when offline with no stored body
- [x] Review section filled in on this task file, including the Phase 4 trigger gap
- [ ] One branch, one PR, on `claude/feature-90a5-oversight-g1fkda` — branch yes,
      commit made; the tech lead opens the PR per the task's working notes

## Review (implementer, 2026-09-08)

### What was built
1. **`content_hash` plumbed through** (the blocker the tech lead flagged):
   - `apps/shared/src/types/article.ts`: `contentHash?: string` added to `Article`.
   - `apps/mobile/src/services/read.ts`: both `transformToArticle` and
     `transformDetailToArticle` now copy `content.content_hash` onto `contentHash`.
2. **`articleStore.ts` migration mechanism** (decision 2): `getDb()`'s `CREATE TABLE`
   statement is untouched (still the v1 shape); a new `migrate(db)` function runs after
   it, reading `PRAGMA user_version` and applying `ALTER TABLE articles ADD COLUMN
   content_hash TEXT` only when the stored version is below 2, then writing the new
   version back — a no-op on a DB already at version 2 (verified: the ALTER never
   fires twice, which would otherwise throw `duplicate column name`).
3. **Hash diff replaces the old body-preserving `COALESCE` rule** (decision under
   "Selection, staleness and eviction"): the `UPSERT_SQL` now sets
   `content_hash = COALESCE(excluded.content_hash, articles.content_hash)` and a `CASE`
   on `body` — keep it when the incoming hash is absent or equal to the stored one
   (both preserve the pre-existing "list refresh can't erase a cached body" behaviour),
   clear it to `NULL` on a genuine hash change. Pinned by three tests (same hash /
   changed hash / no hash).
4. **`ArticleStore.listPrefetchCandidates(limit)`**: `WHERE is_read = 0 AND body IS
   NULL ORDER BY added_at DESC LIMIT $limit` — exactly the one-statement selection the
   spec derives from the hash-diff-on-write design, taken as specified. **`ArticleStore
   .evictBodiesOutsideCap(limit)`**: clears `body` for rows outside the `added_at`-DESC
   top `limit`.
5. **`src/services/articlePrefetch.ts`** (new, `ArticlePrefetchService.run()`):
   guarded by `isOffline()` (decision 7); reads up to 100 candidates; a 3-worker pool
   (decision 6, hand-rolled, no new dependency) pulls from a shared queue; a
   `NetworkError` sets an `aborted` flag checked before each worker grabs its next item
   (decision 7 — the in-flight requests from the initial wave still complete, but no
   worker starts a new one after observing the abort); any other per-article error is
   logged and skipped, batch continues; `evictBodiesOutsideCap(100)` runs after the
   batch regardless of how it ended (see "decisions made here" below); module-level
   `inFlight` boolean makes a concurrent second call a no-op (decision 8).
6. **`ReadScreen.tsx`**: `onResetLoaded` now chains
   `ArticleStore.upsertMany(next).then(() => ArticlePrefetchService.run())` with a
   `.catch` that logs — prefetch runs only after the write that may invalidate bodies
   has actually landed, not alongside it. `BookmarksScreen` is untouched (decision 4).
7. **`ReadArticleDetailScreen.tsx`**: the content-loading effect now resolves the
   stored article by id first (as before), but additionally compares the stored hash
   against the route param's fresh `contentHash`; on a match it returns without ever
   calling `getContentById`. Added a `useNetworkStatus()`-backed offline guard: while
   offline, the network fetch is skipped entirely (it can only fail); a derived render
   check (`!article.content && isOffline`) shows a "Not available offline" message
   instead of the old "spinner clears, empty article renders" behaviour, without
   touching the non-offline empty-body case (out of scope per the spec).

### Files changed
- `apps/shared/src/types/article.ts` — `+1` line.
- `apps/mobile/src/services/read.ts` — `+2` lines (both transforms).
- `apps/mobile/src/services/articleStore.ts` — migration mechanism, hash-aware
  `UPSERT_SQL`, `content_hash` in `ArticleRow`/`articleToParams`/`rowToArticle`,
  `listPrefetchCandidates`, `evictBodiesOutsideCap`.
- `apps/mobile/src/services/articlePrefetch.ts` (new) — the prefetch service.
- `apps/mobile/src/services/index.ts` — export the new service.
- `apps/mobile/src/screens/ReadScreen.tsx` — wire the prefetch call into `onResetLoaded`.
- `apps/mobile/src/screens/ReadArticleDetailScreen.tsx` — hash-skip fetch, offline guard,
  "Not available offline" state.
- Tests: `articleStore.test.ts` (+9 new tests: 3 hash invalidation, 2 prefetch
  selection, 2 eviction, 2 migration), `articlePrefetch.test.ts` (new, 8 tests),
  `ReadArticleDetailScreen.test.tsx` (+4 tests), `ReadScreen.test.tsx` (+1 test, plus
  mocking the new service).
- `apps/mobile/AGENTS.md` (== `CLAUDE.md`) — documented the new service, the two new
  `ArticleStore` methods, the migration mechanism, and `contentHash` on `Article`.

### Decisions made here that the spec left open
1. **Eviction runs after the batch regardless of whether it completed or was aborted
   by a `NetworkError`**, but *not* when the whole run was skipped by the offline
   guard. The spec says "after a run, evict outside the cap" without saying whether an
   aborted run still counts. Eviction is pure bookkeeping independent of the network
   pass (it just enforces the retention cap), so I judged it should still run after an
   abort — the alternative (skipping eviction on abort) has no upside and would let the
   store drift further from the cap on a flaky connection, which is exactly when it
   matters most. Skipping it when offline-guarded is different: nothing changed, so
   there is nothing to reconcile against the cap that a normal sync wouldn't already
   have done more cheaply. Flagging this because it's a real interpretation choice, not
   because I think it's likely wrong.
2. **Hash-skip in the detail screen requires both hashes to be defined and equal.**
   Two undefined hashes are *not* treated as "equal" (i.e. not skipped) — I judged
   this the safer reading of "stored hash equals the route-param hash": we cannot
   confirm equality of two absent values, so falling through to fetch (today's
   behaviour) is the conservative choice. This also matches the pre-existing test
   fixture in `ReadArticleDetailScreen.test.tsx`, which has no `contentHash` on either
   side and still expects a fetch — I kept that test passing rather than reinterpreting
   it.
3. **The "Not available offline" screen renders without the `BottomActionMenu`**, same
   as the existing loading-spinner branch. The spec doesn't say whether Back/Archive
   should stay reachable from that state; I matched the existing spinner branch's shape
   (full-screen replacement, no action menu) rather than inventing a partial layout, since
   nothing in scope asked for one.
4. **`isOffline` is read through a ref inside the content-loading effect**, not added
   to its dependency array, mirroring this file's existing pattern (`articleIdRef`,
   `hasMarkedCompletedRef`, etc.) of reading current values via refs rather than
   re-running effects on every state change. This means a connectivity flip *while* the
   effect is already resolving `ArticleStore.getById` is observed at the point the
   offline check runs (which is what the offline-guard test exercises), but a flip
   *after* the effect has already finished (e.g. content already rendered, or already
   marked unavailable) does not re-trigger a fetch attempt. I judged this consistent
   with the file's existing style rather than a gap, but it's worth a second look.

### Verification (all re-run clean at the end, after the regression proofs below)
| Gate | Result |
|---|---|
| `npx jest` (whole mobile suite) | 31 suites, 199 tests, all pass |
| `npm run type-check` (mobile) | clean |
| `npm run lint` (mobile) | 0 errors, 12 warnings — same 12 as the stated baseline, none in a touched file |
| `apps/shared` — **no `build` script exists** in `apps/shared/package.json` (it's `noEmit`-only, `types` point straight at `src`). Ran `npx tsc --noEmit` directly instead | clean |

**Regression proof 1** — reverted `apps/mobile/src/services/articleStore.ts`,
`apps/mobile/src/services/read.ts` and `apps/shared/src/types/article.ts` to their
pre-change (`main`) versions, kept the new tests, ran `articleStore.test.ts`:
**8 of 26 tests fail** — exactly the 3 hash-invalidation tests, both
`listPrefetchCandidates` tests, both `evictBodiesOutsideCap` tests, and the "adds
content_hash to a v1-seeded DB" migration test. The 18 pre-existing tests are
unaffected. One new test — "does not re-run the ALTER on a second open" — **passes
both before and after**: against the reverted code there is no `content_hash` column
to collide on, so calling `getById` twice never throws either way. That test is a
regression *guard* against a future regression, not a proof of this change; the "adds
content_hash..." test is the one that actually proves the migration exists.

**Regression proof 2** — reverted `apps/mobile/src/screens/ReadArticleDetailScreen.tsx`
and `apps/mobile/src/screens/ReadScreen.tsx`, kept the new tests: **4 of 7 tests fail**
— "renders a stored body with no network call when the stored hash matches" (old code
always fetches), "shows 'Not available offline' when offline with no stored body" (old
code has no such state — sits on the spinner forever since `getContentById` never
resolves in that test), "renders the stored body offline instead of the unavailable
state when one is cached" (old code still calls `getContentById` even though a body is
cached and the device is offline), and "triggers prefetch after a successful sync
upserts the new page" (old `onResetLoaded` never calls the prefetch service). The
remaining 3 tests — the original stored-body-renders-immediately test, the
hash-differs-still-fetches test, and the original ReadScreen store-render test — are
unaffected, which is the right shape: the hash-differs case behaves identically to the
old "always fetch" code, so it was never expected to fail.

After both proofs, all reverted files were restored (`git stash pop` in each case) and
the full suite/type-check/lint were re-run clean (results in the table above).

### Known gaps and limitations — reported honestly, not just what's convenient
- **Phase 4 trigger gap (decision 5), as instructed:** only two of the task's four
  named triggers are wired — focus-with-expired-TTL and pull-to-refresh, both of which
  already route through `ReadScreen.onResetLoaded`. **App foreground and reconnect are
  not implemented.** There is no `AppState` or network-change listener in this app
  (confirmed: `grep -r AppState src/` still finds nothing after this change), and
  building one here means shipping a listener whose only consumer is prefetch, then
  rebuilding around it in task_ebf1 (Phase 4) for the outbox's own "we're back" trigger.
  **Concretely, this means:** a user who backgrounds the app, regains connectivity, and
  returns to it will not have prefetch run until they next open or pull-to-refresh the
  Read tab — and even that only fires if the 30s focus TTL has expired
  (`FOCUS_REFETCH_TTL_MS` in `ReadScreen.tsx`). Reconnecting while already sitting on
  the Read tab does not trigger a prefetch pass at all until the next qualifying focus
  or pull-to-refresh. This is exactly what decision 5 asked for; it is not an oversight,
  but it is a real, currently-shipping gap in "prefetch runs on reconnect."
- **Concurrency is not directly asserted by count.** The worker-pool tests prove
  selection, abort, and single-flight behaviour, but nothing in this suite pins "no
  more than 3 requests are in flight simultaneously" — only that a batch of 5 starts at
  most the first 3 concurrently and stops issuing new ones after an abort (verified via
  microtask-ordering, not an explicit concurrency counter). I reasoned through why the
  microtask ordering makes this deterministic under Node's Promise semantics (documented
  in the test's own comment) rather than asserting it with a counter/mutex-style test,
  which felt like it would test the test harness more than the implementation. A
  reviewer who wants a hard numeric assertion on concurrency should treat this as a gap.
- **No device or simulator testing of any kind.** Every network, connectivity, and
  SQLite behaviour here is exercised through mocks/`node:sqlite`, per this feature's
  running pattern (task_a8a4, task_c87c both note the same). This remains task_de93's
  territory and is the largest unverified assumption across all three phases so far.
- **Metro Bundle / Hermes step not run.** Per the working notes, `hermesc` under
  `node_modules/react-native/sdks/hermesc/linux64-bin/` is x86-64 and this sandbox is
  aarch64 — confirmed pre-existing (same failure mode task_a8a4 recorded), not something
  introduced here. Not attempted; must be confirmed green on CI before merge.
- **Prefetch failure visibility.** A prefetch run that fails entirely (e.g. offline-guard
  trip, or every candidate erroring) is silent to the user beyond the existing
  per-article `console.error` logs — there is no UI signal that "prefetch didn't run" or
  "prefetch partially failed." The spec doesn't ask for one, and I did not add one, but
  it means a user who goes offline mid-session with unprefetched articles gets no
  indication that those articles are about to become unavailable until they actually
  open one — at which point the existing "Not available offline" state (this task) is
  the only feedback they get.
- **`upsertMany`'s per-row `await` loop is unchanged** — still not batched into a single
  transaction (a pre-existing note from task_a8a4's review, not something this task
  touched or was asked to touch). The hash-diff logic adds one more `CASE` per row to
  the same loop; no new performance concern beyond what already existed, but also not
  improved.
- **I did not verify behaviour under a genuinely large (near-cap) Read list** — all
  tests use small fixtures (2-5 rows). The `LIMIT`/`ORDER BY added_at DESC` queries are
  straightforward SQL with no reason to expect different behaviour at 100 rows vs. 5,
  but I have not empirically exercised 100+ rows through the real `node:sqlite`-backed
  mock.
- **Migration path only tested for v1 → v2.** There is exactly one migration step today,
  so the "apply steps in order" claim in the `migrate()` doc comment is asserted by
  design/code-reading, not by a test that chains two real migrations — there's only one
  to chain. This will matter once task_ebf1 adds a second step.
