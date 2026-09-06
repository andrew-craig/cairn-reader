---
id: task_a8a4
title: Mobile: single SQLite-backed local article store (replaces AsyncStorage article caches)
type: task
status: in_progress
priority: 2
labels: [mobile,offline]
blocked_by: []
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-05T23:36:10Z
updated_at: 2026-09-06T03:16:28Z
---
Phase 1 of feature_90a5. Introduce one local store (expo-sqlite) for read-list articles: metadata, user state (status/is_favorite/scroll_position/updated_at) and a nullable body column. ReadScreen, BookmarksScreen (query is_favorite from the store; it has no cache today) and ReadArticleDetailScreen read from the store first and refresh from the network. No import from the old AsyncStorage caches: the store repopulates on first sync. Removes ARTICLES_KEY and READ_LIST_CACHE_KEY from storage.ts (EXPLORE_CACHE_KEY stays). Clears the store on logout. Supersedes the dual-cache part of task_179f.

## Scope clarification (tech lead, 2026-09-06)

Verified against `main` at d2f92e3 (task_cab7 merged).

### What is actually broken today

`ARTICLES_KEY` is a **write-only store**. `StorageService.getArticles()` has no
caller anywhere outside `storage.test.ts`. Meanwhile four user actions write to it:

| Caller | Write |
|---|---|
| `AddArticleScreen.tsx:79` | `addArticle` |
| `ReadArticleDetailScreen.tsx:100` | `updateArticle` (mark read) |
| `ReadArticleDetailScreen.tsx:145` | `updateArticle` (scroll position) |
| `ReadArticleDetailScreen.tsx:173` | `updateArticle` (favourite) |
| `ReadArticleDetailScreen.tsx:194` | `deleteArticle` (archive) |

Every one of those persists into a black hole. `ReadScreen` reads a *different*
key (`READ_LIST_CACHE_KEY`, a whole-list JSON blob) which no other screen writes,
so a favourite toggled in the detail screen is invisible to the list cache.
`BookmarksScreen` has no cache at all and shows an error banner when offline.
That split is the real defect; one store fixes it.

### Design decisions (made here — do not re-litigate, raise it if you disagree)

**1. One table, `articles`, keyed by content id.** Plain SQL in a single module
`src/services/articleStore.ts` exporting `ArticleStore` (object literal of async
methods, matching the existing `StorageService` shape). No ORM, no query builder,
no repository abstraction. Columns cover the `Article` fields the screens read,
plus a nullable `body` column for cleaned HTML.

**2. Sync is upsert-only. Never bulk-delete on sync.** A list page from the server
upserts its rows; rows already stored are updated, not cleared. Deletion happens
only on the explicit archive/delete path. Consequence, accepted: an article
archived on another device lingers locally until that path runs. Reconciling that
is sync-model work and is out of scope for this feature.

**3. Do not paginate from SQLite.** `useCursorArticleList` keeps driving cursor
pagination against the network exactly as it does now. The store serves the
*initial* render (and the offline case) only: `ReadScreen` reads the 100 most
recent stored articles, `BookmarksScreen` reads stored rows `WHERE is_favorite`.
The store is a read-through cache, not a query engine.

**4. Bodies are cached opportunistically, not prefetched.** When
`ReadArticleDetailScreen` already fetches a body via `getContentById`, write it to
`body`; on next open, render the stored body first and refresh from the network.
That alone makes re-opening a recently read article work offline. Bulk prefetch,
`content_hash` diffing, the "Not available offline" state and eviction caps are
**task_c55c** — do not build them here.

**5. No migration from the old AsyncStorage caches.** Per CLAUDE.md §4, delete the
old paths; the store repopulates on first sync.

### Testing: use real SQL

`node:sqlite` is built into Node 24, and mobile CI runs Node 24
(`.github/workflows/mobile-checks.yml:25`). Write a jest mock for `expo-sqlite`
that adapts `openDatabaseAsync` onto a `DatabaseSync(':memory:')` — roughly a
40-line adapter mapping `execAsync`/`runAsync`/`getAllAsync`/`getFirstAsync` onto
`exec`/`prepare().run()/.all()/.get()`. This makes the store's **real schema and
real queries execute in unit tests**, rather than asserting against a fake that
can't disagree with you. No new dependency: `node:sqlite` is built in.

Screens are tested against a mocked `ArticleStore` (the existing
`ExploreScreen.test.tsx` mocks `StorageService` the same way).

### Plan
- [x] Add `expo-sqlite` with `npx expo install expo-sqlite` (SDK 54-compatible
      version, not a hand-picked one). → verify: the Metro Bundle CI check passes.
- [x] `src/services/articleStore.ts`: schema + idempotent open/create, `upsertMany`
      (list page), `listRecent(limit)`, `listFavorites()`, `getById`, `saveBody`,
      `updateUserState` (isRead/isFavorite/scrollFraction), `remove`, `clear`.
      Exported from `src/services/index.ts`. → verify: unit tests against the
      `node:sqlite`-backed mock covering upsert-updates-not-duplicates, ordering,
      the favourites filter and null-body round-tripping.
- [x] `ReadScreen`: replace `getReadListCache`/`saveReadListCache` with
      `ArticleStore.listRecent` on focus and `upsertMany` after a successful sync;
      archive removes the row. → verify: test asserts stored articles render before
      any network call resolves.
- [x] `BookmarksScreen`: render `listFavorites()` first, then refresh from the
      network. → verify: test asserts bookmarks render from the store while the
      network call is pending, and that a network failure no longer blanks the list.
- [x] `ReadArticleDetailScreen`: the five `StorageService` calls above become
      `ArticleStore` calls; persist the body fetched by `getContentById`; prefer a
      stored body over the loading spinner on open. → verify: test asserts a stored
      body renders without `getContentById` resolving.
- [x] `AddArticleScreen:79`: `StorageService.addArticle` → `ArticleStore.upsertMany`.
- [x] Clear the store on logout (`AuthContext.logout`, alongside `AuthService.logout`).
      → verify: test asserts `clear` is called.
- [x] Delete `ARTICLES_KEY`, `READ_LIST_CACHE_KEY` and their now-orphaned
      `StorageService` methods; keep `EXPLORE_CACHE_KEY` and its two methods. Update
      `storage.test.ts` and the StorageService section of `apps/mobile/CLAUDE.md`.
      → verify: `npx expo-doctor`-independent — the Dead Code CI check passes.

### Constraints
- Do not touch `ExploreScreen`, `EXPLORE_CACHE_KEY`, or the explore cache methods.
  Explore articles are never written to the store.
- Do not change `useCursorArticleList`.
- Do not implement anything listed under task_c55c (decision 4).
- Keep `articleStore.ts` under ~250 lines. If it grows past that, the design drifted.

### Verification gate (all must pass before you report done)
1. `npx jest` — whole mobile suite green.
2. `npm run type-check` — clean.
3. `npm run lint` — no new errors *or* warnings in files you touched.
4. **Regression proof:** for each new behaviour test, confirm it fails against the
   pre-change code. Report which tests fail and why. A test that passes both before
   and after is a regression guard, not a proof — label it as such.

Report honestly what you did *not* verify. Nothing here is device-tested; that is
task_de93.

## Implementation notes (2026-09-06)

### Files changed
- `apps/mobile/package.json`, `apps/mobile/app.json`: added `expo-sqlite` via
  `npx expo install expo-sqlite` (resolved to `~16.0.10`, SDK 54-compatible);
  the installer also registered the `expo-sqlite` config plugin in `app.json`.
- `apps/mobile/src/services/articleStore.ts` (new, 218 lines, under the ~250
  line guideline): the `articles`
  table + `ArticleStore` object literal (`upsertMany`, `listRecent`,
  `listFavorites`, `getById`, `saveBody`, `updateUserState`, `remove`,
  `clear`). One UPSERT statement drives `upsertMany`; `body` is preserved via
  `COALESCE(excluded.body, articles.body)` since list pages never carry
  `content` (see design decision below).
- `apps/mobile/__mocks__/expo-sqlite.js` (new): adapts `openDatabaseAsync` /
  `execAsync` / `runAsync` / `getAllAsync` / `getFirstAsync` onto
  `node:sqlite`'s `DatabaseSync(':memory:')`.
- `apps/mobile/src/services/articleStore.test.ts` (new): 12 tests against the
  real-SQL mock (upsert-no-duplicate, upsert-only/no-bulk-delete, ordering,
  favourites filter, body round-trip incl. COALESCE-preserve, updateUserState
  per-field, remove, clear).
- `apps/mobile/src/services/index.ts`: export `articleStore`.
- `apps/mobile/src/services/storage.ts`: removed `ARTICLES_KEY`,
  `READ_LIST_CACHE_KEY` and their methods (`getArticles`, `saveArticles`,
  `addArticle`, `updateArticle`, `deleteArticle`, `clearAllArticles`,
  `getReadListCache`, `saveReadListCache`). `EXPLORE_CACHE_KEY` and its two
  methods are untouched.
- `apps/mobile/src/services/storage.test.ts`: rewritten to cover only
  `getExploreCache`/`saveExploreCache` (the ARTICLES_KEY/READ_LIST_CACHE_KEY
  tests no longer have a subject).
- `apps/mobile/src/screens/ReadScreen.tsx`: focus effect now calls
  `ArticleStore.listRecent(100)` instead of `StorageService.getReadListCache`;
  `onResetLoaded` calls `ArticleStore.upsertMany`; archive calls
  `ArticleStore.remove`.
- `apps/mobile/src/screens/ReadScreen.test.tsx` (new): stored article renders
  while `ReadService.listUserContents` is still pending.
- `apps/mobile/src/screens/BookmarksScreen.tsx`: focus effect now primes
  `articles`/`loading` from `ArticleStore.listFavorites()` before calling
  `load(true)`; `onResetLoaded` now also calls `ArticleStore.upsertMany` (see
  design decision below).
- `apps/mobile/src/screens/BookmarksScreen.test.tsx` (new): stored favourite
  renders while the network call is pending, and survives a network failure
  (no more blank-to-error).
- `apps/mobile/src/screens/ReadArticleDetailScreen.tsx`: the five
  `StorageService` call sites (mark-read, two scroll-position saves, favourite
  toggle, archive) now call the matching `ArticleStore` method. The
  content-loading effect now first checks `ArticleStore.getById` for a cached
  body (rendering it and clearing `contentLoading` immediately) while the
  `getContentById` network fetch still runs in the background; if that fetch
  returns a body, it's persisted via `ArticleStore.saveBody`.
- `apps/mobile/src/screens/ReadArticleDetailScreen.test.tsx` (new): cached
  body renders while `getContentById` is still pending.
- `apps/mobile/src/screens/AddArticleScreen.tsx`: `StorageService.addArticle`
  → `ArticleStore.upsertMany([article])`.
- `apps/mobile/src/contexts/AuthContext.tsx`: `logout()` now calls
  `ArticleStore.clear()` after `AuthService.logout()`.
- `apps/mobile/src/contexts/AuthContext.test.tsx`: added `ArticleStore` to the
  `../services` mock and a new `AuthProvider.logout` describe block asserting
  `ArticleStore.clear` is called (via `renderHook`).
- `apps/mobile/AGENTS.md` (== `CLAUDE.md`, symlinked): rewrote the
  StorageService section (explore-cache only now), added an ArticleStore
  section, and touched the three other spots that described AsyncStorage as
  the app's only local persistence (Key Features bullet, project structure
  tree, State Management / Service Layer Pattern lists).

### Design decisions made here, not covered by the task file
1. **`upsertMany` overwrites everything except `body`.** The task's decision 2
   says "upsert-only, never bulk-delete"; it doesn't say which columns a
   re-sync should touch. I overwrite every field from the incoming `Article`
   on every upsert (matching the old `saveReadListCache`/`saveArticles`
   whole-blob-replace behaviour) *except* `body`, which is preserved via
   `COALESCE` when the incoming article has no `content` — true for every
   list-page response, per `transformToArticle`. Without this, any list
   refresh (including the periodic `ReadScreen` focus refetch) would erase a
   body cached by decision 4 the moment the list synced again.
2. **`BookmarksScreen.onResetLoaded` now also calls `ArticleStore.upsertMany`.**
   The plan bullet for BookmarksScreen only mentions reading
   `listFavorites()`; it doesn't say whether Bookmarks' own network page
   should feed the store. I added it because decision 2 states generally that
   "a list page from the server upserts its rows," and because without it, a
   favourite whose article was never synced through `ReadScreen`'s most-recent
   100 window (e.g. an older article, or a device that only ever opens
   Bookmarks) would have no row for `ReadArticleDetailScreen`'s
   `updateUserState` to land on — that call is a silent no-op against a
   missing id.
3. **`ReadArticleDetailScreen`'s cached-body lookup uses a functional
   `setArticle` update** (`current.content ? current : {...current, content}`)
   so a fast local cache hit can't clobber a network response that resolves
   first, and a slow cache hit can't clobber a network response that already
   arrived. Not specified in the task; seemed necessary to avoid flicker/loss
   given both requests are in flight together.

### Verification gate results
1. **`npx jest`** — 22 suites, 141 tests, all passing.
2. **`npm run type-check`** — clean (`tsc --noEmit`, no output).
3. **`npm run lint`** — 0 errors, 12 warnings, all pre-existing and in files I
   did not touch except one: `AddArticleScreen.tsx:18` `'AuthService' is
   defined but never used`. Confirmed via `git stash` + re-lint against
   unmodified `main` that this warning predates my change (I only edited that
   import line to swap `StorageService` for `ArticleStore`; `AuthService` was
   already dead on that same line before I touched it). Left as-is per
   "don't clean up unrelated dead code, mention it instead."
4. **Regression proof** — confirmed by temporarily restoring the pre-change
   source file(s) via `git stash push -- <file(s)>` (keeping the new test
   files in place), running the affected test, then `git stash pop`:
   - `ReadScreen.test.tsx` (stashed `ReadScreen.tsx` + `storage.ts` together,
     to get true pre-change behaviour rather than a "method doesn't exist"
     crash): **fails** — stuck on the loading spinner, `getReadListCache()`
     returns `null` from an empty AsyncStorage mock, no stored article shown.
   - `BookmarksScreen.test.tsx` (stashed `BookmarksScreen.tsx`): **both tests
     fail** — first test times out with no store to read from; second test's
     failure output shows the exact bug this task fixes ("Couldn't load your
     bookmarks..." error screen with a Retry button, instead of the stored
     favourite).
   - `ReadArticleDetailScreen.test.tsx` (stashed
     `ReadArticleDetailScreen.tsx`): **fails** — stuck on the spinner, no
     store lookup exists in the old effect.
   - `AuthContext.test.tsx`, new `AuthProvider.logout` test (stashed
     `AuthContext.tsx`): **fails** — `ArticleStore.clear` never called
     (0 calls); the three pre-existing task_cab7 tests in the same file still
     pass unchanged, confirming the mock/file split didn't affect them.
   - `articleStore.test.ts` (12 tests): these test a wholly new file, so
     "pre-change" is simply its absence — not a meaningful regression proof
     in the same sense. They are regression *guards* going forward, not
     proofs of a behaviour change.
   - `storage.test.ts`: rewritten to drop tests for deleted methods and add
     coverage for the untouched `getExploreCache`/`saveExploreCache`. These
     pass identically before and after (the explore-cache code didn't
     change) — a regression guard, not a proof.

### Not verified / could not verify
- **Metro Bundle CI check (`npx expo export --platform ios`)**: Metro itself
  bundled all 1795 modules successfully (including `expo-sqlite`, confirming
  module resolution is fine), but the subsequent Hermes bytecode-compile step
  failed in this sandbox: `hermesc` under
  `node_modules/react-native/sdks/hermesc/linux64-bin/` is an x86-64 ELF
  binary and this sandbox is aarch64 (`uname -m` → `aarch64`). I confirmed
  this is a pre-existing environment limitation, not something my change
  caused, by running the same `hermesc` binary directly on a trivial `.js`
  file (`Exec format error`, independent of any app code). This will pass in
  the real GitHub Actions CI (`ubuntu-latest`, x86-64) but I could not
  personally verify the Hermes step end-to-end here.
- **Device/simulator testing**: none attempted, per the task's own note —
  that's task_de93.
- **`npx expo-doctor`**: not run (the task lists it as "independent" of the
  Dead Code check, and I ran `npx knip` directly instead, which passed with
  no output/issues).
- I did not verify behaviour under real concurrent multi-device sync
  (decision 2's accepted staleness) beyond what's covered by the upsert-only
  unit tests — that reconciliation is explicitly out of scope (task_c55c
  territory).

## Review (tech lead, 2026-09-06)

Implemented by a subagent, **independently re-verified by the reviewer** — every
gate below was run by the reviewer, not taken on the implementer's report.

| Gate | Result |
|---|---|
| `npx jest` (whole mobile suite) | 22 suites, 142 tests, all pass |
| `npm run type-check` | clean |
| `npm run lint` | 0 errors, 12 warnings — count unchanged from before this task; none in a touched file |
| Regression proof (5 source files reverted, new tests kept) | exactly 5 failures, matching the 5 behaviour tests |
| Regression proof (logout fix only, defect reintroduced) | exactly 1 failure, the intended test |

The first regression proof reverted `ReadScreen.tsx`, `BookmarksScreen.tsx`,
`ReadArticleDetailScreen.tsx`, `AuthContext.tsx` and `storage.ts` to their `main`
versions while keeping `articleStore.ts`, the `expo-sqlite` mock and all new
tests. Exactly five tests failed — the ReadScreen and both BookmarksScreen
offline-first cases, the detail-screen stored-body case, and the logout store
clear — while the three pre-existing `AuthContext` tests were unaffected. That is
the correct shape.

**Accepted.**

### What was good
- The `node:sqlite` mock (`__mocks__/expo-sqlite.js`, ~30 lines) makes the store's
  **real schema and real SQL execute under test**. This is the difference between
  testing the store and testing a fake that cannot disagree with the
  implementation, and it cost no new dependency.
- `body = COALESCE(excluded.body, articles.body)` in the upsert. List responses
  never carry `cleaned_html`, so a plain overwrite would have erased every cached
  body on each list refresh — a bug that would only have surfaced later, in
  task_c55c, as "prefetch works but bodies keep vanishing". The implementer found
  this unprompted and covered it with a test.
- Genuinely surgical: `articleStore.ts` is 218 lines of plain SQL with no ORM, no
  query builder and no repository abstraction.

### Defect found in review and fixed
`AuthContext.logout` placed the new `await ArticleStore.clear()` between
`AuthService.logout()` and `setUser(null)`, inside a try whose catch rethrows
without ever calling `setUser(null)`. A SQLite failure would therefore wipe the
tokens but leave the context reporting the user as authenticated — the app keeps
rendering while every request 401s — **and** leave the articles on disk, so on a
shared device the next user would see the previous user's articles on ReadScreen's
first render. Fixed by making the added call unable to alter the existing control
flow:

```js
await ArticleStore.clear().catch((err) =>
  console.error('Failed to clear local articles on logout:', err),
);
```

This is the **same class of defect as the PR #381 review finding** on task_cab7:
adding a call that can throw inside an existing try block widens that block's
failure surface beyond what the pre-existing code could do. Second occurrence in
this feature — see LEARNINGS.md.

### Known gaps, deliberately left
- **Offline user-state edits are lost on the next sync.** `upsertMany` overwrites
  `is_read`/`is_favorite`/`scroll_fraction` from the server page. Favouriting an
  article offline updates the store, but `updateUserContent` fails, and the next
  list sync overwrites the local value. This is the sync model (last-write-wins),
  explicitly out of scope for this feature.
- Articles archived on another device linger locally until the archive path runs
  here — the accepted consequence of decision 2 (upsert-only).
- `getDb()` caches `dbPromise` at module scope; a rejected open is cached
  permanently. Low severity (an `openDatabaseAsync` failure is effectively fatal),
  noted rather than fixed.
- `upsertMany` awaits per row rather than using one transaction. Fine at
  `PAGE_SIZE`; would matter if the store ever ingests large batches.
- `BookmarksScreen` renders stored favourites with no stale indicator, where
  `ReadScreen` sets `isStale`. Cosmetic inconsistency, out of scope.
- **Metro Bundle was not verified locally.** Metro resolved and bundled all 1795
  modules including `expo-sqlite`, but the Hermes bytecode step cannot run in this
  sandbox — `hermesc` ships as an x86-64 binary and this host is aarch64. Verified
  to be a pre-existing environment limit (the same binary fails on a trivial file
  with no app code). It must be confirmed green on CI before merge.
- **Nothing is device-tested.** All offline behaviour here is exercised through
  mocks. That is task_de93, and it remains four phases away.

**Not committed** — left in the working tree on branch
`task_a8a4-sqlite-article-store` pending the go-ahead.

## Review follow-up (PR #382, 2026-09-06)

One review comment on the pushed commit (0e079b2), rated Important. It was
correct, and it caught something the tech lead's own review had mis-assessed.

**The defect.** In `ReadScreen:90` and `BookmarksScreen`, the network refresh
`load(true)` was called only *inside* the `ArticleStore.listRecent()` /
`listFavorites()` `.then()` callback, which had no `.catch`. A rejected SQLite
read therefore skipped `load(true)` entirely and left the screen on its loading
spinner forever. `ReadArticleDetailScreen`'s `getById` chain had the same missing
handler, producing an unhandled rejection.

This was a regression introduced by this task. The `StorageService.getReadListCache()`
it replaced wrapped its body in try/catch and returned `null`, so it could never
reject and the network load always fired (`git show d2f92e3:apps/mobile/src/services/storage.ts`).

**Why the reviewer's suggested fix was not taken.** The comment suggested adding
`.catch(() => { void load(true); })` at each call site. That treats the symptom.
Three call sites had already forgotten the handler, and every future caller would
have to remember — the same per-call-site fragility that `chore_1089` exists to
eliminate for `fetch`. The root cause is that `ArticleStore`'s read methods
silently changed the contract the screens were written against.

**Fix.** `listRecent`, `listFavorites` and `getById` now catch, log via
`console.error` and resolve to `[]`/`[]`/`null` — a read-through cache degrades to
"nothing cached" rather than blocking its callers. The contract is documented on
each method. Write methods (`upsertMany`, `saveBody`, `updateUserState`, `remove`,
`clear`) still propagate, because their call sites already catch and log, and
`AuthContext.logout` depends on `clear()` rejecting so its own `.catch` fires. A
test pins that read/write asymmetry so it cannot be "tidied" away later.

`getDb()` also now resets `dbPromise = null` on a failed open, so a single
transient failure no longer poisons every subsequent call. That is what makes the
degradation above recoverable rather than permanent.

**No screen file was edited.** That was the test of whether the fix was at the
right layer, and it held.

**Reviewer's own error, recorded.** The tech lead's review section above lists the
cached-rejected-open as a "low severity, noted rather than fixed" gap. That
assessment was wrong: it was judged in isolation, without tracing what the screens
do when a read rejects. Chained to the missing `.catch`, a transient DB failure
became a permanent spinner. See LEARNINGS.md.

**Verification (re-run by the reviewer, not taken on report):** `npx jest` — 23
suites / 148 tests pass. `npm run type-check` — clean. `npm run lint` — 0 errors,
12 pre-existing warnings, none in a touched file. Regression proof: reverting
`articleStore.ts` to 0e079b2 while keeping the new tests fails exactly the 4 new
store-level tests plus the new screen-level test, with the 12 pre-existing store
tests unaffected. The write-still-rejects test passes both before and after — a
regression guard, correctly labelled as such.

The screen-level test (`ReadScreen.storeFailure.test.tsx`) is the strongest piece:
it does not mock `ArticleStore` at all, but mocks `openDatabaseAsync` to reject and
drives the real screen against the real store. Against the pre-fix code it times
out waiting for `listUserContents` — reproducing the reported defect end to end.
