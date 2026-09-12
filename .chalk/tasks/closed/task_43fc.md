---
id: task_43fc
title: Docs: offline reading architecture, requirements and mobile CLAUDE.md
type: task
status: closed
priority: 3
labels: [docs,offline]
blocked_by: []
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-05T23:36:10Z
updated_at: 2026-09-12T22:37:45Z
---
Phase 5 of feature_90a5. Document the local store, prefetch and outbox in apps/mobile/CLAUDE.md and docs/ARCHITECTURE.md; move offline reading out of Future Enhancements in docs/product_requirements.md; note explicit non-goals (web/PWA, image caching).

## Review

Docs-only change on branch `task_43fc-offline-docs`, committed locally (not pushed, no PR).
All claims were verified against the shipped source under `apps/mobile/src/` before writing,
not against the feature_90a5 plan text.

**Files changed** (note: `apps/mobile/CLAUDE.md` and the root `/CLAUDE.md` are symlinks to
`AGENTS.md` in the same directories — edited the real files):
- `apps/mobile/AGENTS.md` (= `apps/mobile/CLAUDE.md`): added `Outbox`, `ArticleMutations`,
  `SyncTrigger`, the `useNetworkStatus`/`network.ts` connectivity pair, `useSyncTrigger`, and
  `errors.ts` (`NetworkError`/`HttpError`) to the Services section, in the same
  method-signature style as the existing `ArticleStore`/`ArticlePrefetchService` entries
  (which were already documented from an earlier phase). Updated the project structure tree
  (added `db.ts`, `outbox.ts`, `articleMutations.ts`, `syncTrigger.ts`, `errors.ts`,
  `OfflineBanner.tsx`, `SyncTriggerEffect.tsx`, and a new `hooks/` entry), Key Features, Local
  Persistence, Service Layer Pattern, and the `ReadScreen`/`ReadArticleDetailScreen` bullets.
- `docs/ARCHITECTURE.md`: added an "Offline Reading" subsection under Mobile App (store,
  prefetch, outbox, connectivity/sync triggers, conflict resolution, out-of-scope), and
  updated the Core Responsibilities/Technology Stack bullets to mention SQLite/expo-network.
- `docs/product_requirements.md`: the offline requirement already existed under "Reading
  progress saved and synced" (it's the line feature_90a5 was written against) but was
  duplicated by two stale bullets under Future Enhancements → Mobile Features ("Offline
  reading mode", "Download for offline access"). Removed those two bullets and expanded the
  existing requirement in place with shipped detail (SQLite, which fields sync) and an
  explicit non-goals list.

**Concrete facts verified from source** (file:line references are to the pre-doc-change tree):
- `db.ts`: single SQLite file (`cairnreader.db`), `PRAGMA user_version` migration ladder;
  v2 adds `content_hash`, v3 adds the `outbox` table (`PRIMARY KEY (article_id, field)`,
  implicit rowid — no `WITHOUT ROWID`, so `Outbox.drain()` can break `created_at` ties on
  `rowid`).
- `articlePrefetch.ts`: `PREFETCH_LIMIT = 100`, `CONCURRENCY = 3`, newest-first
  (`ORDER BY added_at DESC`), single-flight via an `inFlight` module boolean, aborts the rest
  of the batch on `NetworkError` but only skips-and-logs any other per-article error.
  Candidates are `is_read = 0 AND body IS NULL` — `upsertMany`'s hash diff is what clears
  `body` on a changed hash, so prefetch itself does no hash comparison.
- `outbox.ts`: `OutboxField = 'status' | 'is_favorite' | 'scroll_position' | 'delete'`.
  `sendRow` outcomes — success: 2xx, or 404 on a replayed `delete`; drop: a definitive 4xx
  other than 401; halt: `NetworkError`, 401, or 5xx (401 reaching here means `fetchWithAuth`'s
  own internal retry already failed). `drain()` stops at the first `halt`, bumping `attempts`
  on that row only, leaving the rest of the batch queued. Enqueuing a `delete` first deletes
  any other pending row for that article.
- `articleStore.ts` UPSERT_SQL: a pending `delete` row skips the insert entirely (`WHERE NOT
  EXISTS`); any other pending outbox row for an article freezes `is_read`/`is_favorite`/
  `read_at`/`scroll_fraction` at their stored values instead of accepting the server's
  (keyed on `article_id`, not per-field). `content_hash`/`body`: body is cleared when the
  incoming hash differs from the stored one, kept otherwise.
- `articleStore.ts` `clear()`: deletes from both `articles` and `outbox` — confirmed logout
  (`AuthContext.tsx` `logout()`) calls `ArticleStore.clear()`, so a queued write can't replay
  against a different account.
- `articleMutations.ts`: writes the store first, then attempts the network call; only
  `NetworkError` gets queued to the outbox — any other rejection (4xx, auth failure) rethrows
  unchanged. `markReading` has no store write (the store only tracks the `is_read` boolean,
  not an intermediate "reading" status).
- `syncTrigger.ts`: fixed order `Outbox.drain()` then `ArticlePrefetchService.run()`, each
  isolated from the other's rejection (`runConsumersIsolated`), single-flight.
- `useSyncTrigger.ts`: fires `SyncTrigger.run()` only on a *transition* offline→online or
  background→foreground, not merely on being online/active.
- `useNetworkStatus.ts`/`utils/network.ts`: an unknown/undetermined network state counts as
  online; only explicit `isConnected: false` is offline.
- `errors.ts`: `NetworkError` = unreachable/timeout/unparseable body; `HttpError` carries
  `status` for a definitive non-2xx.
- `ReadArticleDetailScreen.tsx`: resolves the article by id from `ArticleStore` (merging
  store body/hash with fresh route-param metadata), skips the network fetch when the stored
  hash already matches the fresh one, never fetches while offline, and shows "Not available
  offline" when offline with no stored body.
- `ReadScreen.tsx`: `STORED_ARTICLES_LIMIT = 100`; every list reset calls
  `ArticleStore.upsertMany` then `SyncTrigger.run()` (in that order — prefetch must see the
  post-upsert store state), so pull-to-refresh drains the outbox too, not just
  reconnect/foreground.
- `RootNavigator.tsx`: `OfflineBanner` renders in the loading, logged-out, and authenticated
  states; `SyncTriggerEffect` (which mounts `useSyncTrigger`) renders only in the
  authenticated state, so sync never runs logged out.
- Non-goal cross-checks: `docs/detailed_requirements/web_app_requirements.md` lines 18 and
  447 do list "Offline-first / PWA installability" / "PWA / offline reading and
  installability" as out of scope for the web app (the path in the feature_90a5 doc,
  `docs/web_app_requirements.md`, is wrong — the file actually lives under
  `docs/detailed_requirements/`; used the correct path in both edited docs). `feature_9d64`
  ("Image hosting and optimization") is the open feature tracking image caching.

**Divergence between plan text (feature_90a5.md) and shipped code**: none found that
mattered for the docs. The plan's phase descriptions (cap of 100, 3 workers implied by
"bounded concurrency", content_hash diffing, outbox coalescing/404/ordering rules, LWW) all
matched the code exactly. The only inaccuracy was in feature_90a5's own "Out of scope"
section, not in the code: it cites `docs/web_app_requirements.md` for the web/PWA non-goal,
but that file is actually at `docs/detailed_requirements/web_app_requirements.md`. Docs were
written against the code and existing repo file layout, not against that plan text.
