---
id: task_f4ac
title: Web: Feed and Reads routes
type: task
status: closed
priority: 2
labels: [web]
blocked_by: []
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-08T11:32:05Z
updated_at: 2026-10-11T11:59:52Z
---
Mirror of the mobile task: /feed and /reads routes (Read.tsx → Reads with list filter, new Feed.tsx), Save to Reads action, Feed/Reads choice in AddLinkModal, per-source Feed/Reads toggle on both /you/feeds and /you/newsletters (they stay separate routes). Rename /you/feeds to /you/rss and its nav label to RSS so "Feed" only means the list; unsubscribe stays RSS-only.
- Done when: tests pass; verified in browser against staging (webapp-staging-test skill)

## Plan (drafted 2026-10-11)
Branch: `feat/web-feed-reads-routes` off `origin/main` (mobile #420 and backend #414/#416/#421 are merged).
Shape: bottom-up like mobile (service → shared list plumbing → screens → nav), test each layer before the next. Shared types already carry `list`, `ContentList`, `SourceRouteType`, `SetSourceListResponse`.
### Findings from reading the web code
- `apps/web/src/services/read.ts` is a *port* of the mobile service and has none of the new calls: `listUserContents`/`searchUserContents`/`countUserContents` don't send `list`, and `setSourceList`/`moveToReads` don't exist. `addURL` passes its request through, so it only needs the caller to set `list`.
- `Read.tsx` (`/read`, `/read/:id`) is used by Login, ReadArticle (Back/Archive/Delete navigate to `/read`), SearchModal, BottomNav, Sidebar, App redirects.
- `Feeds.tsx` has no routing UI; `Newsletters.tsx` rows are display-only (no toggle).
- `ReadArticle` always calls `updateUserContent({status:'reading'})` on open, throttled scroll saves, and `completed` at 95%. Mobile dropped all of that for Feed items (review follow-up); web must too.
### 1. Service + tests → verify: `services/read.test.ts`
- [x] `listUserContents`, `countUserContents`, `searchUserContents` send `list` when given.
- [x] Add `setSourceList(type, key, list)` (PUT `/content/user/{id}/subscriptions/{type}/{key}/list`, same error handling as mobile) and `moveToReads(id)` = `updateUserContent(id, {list:'reads'})`.
- [x] Tests: query-string cases, PUT URL/body, error path.
### 2. Routes: Reads and Feed → verify: new `Reads`/`Feed` route tests, existing tests green
- [x] Rename `/read` → `/reads` (`Read.tsx` → `Reads.tsx`, `Read.css` stays shared) and `/read/:id` → `/reads/:id`; fetch with `list:'reads'`.
- [x] New `Feed.tsx` at `/feed`, `/feed/:id`. The infinite-scroll/refresh/observer logic in `Reads.tsx` is ~100 lines; one `ArticleList` route component parameterised by `list` (keyed per route) replaces `Read.tsx`, so no hook extraction or copy was needed. Feed: no unread/status UI; search placeholder "Search your feed".
- [x] Update every `/read` reference: App routes + `RootRedirect` (→ `/reads`), Login, BottomNav, Sidebar, SearchModal navigate, ReadArticle Back/Archive/Delete.
- [x] `SearchModal` takes a `list` prop and passes it to `searchUserContents`, so a Feed search can't return Reads items.
- [x] Nav: BottomNav and Sidebar become Feed | Reads | You; add a Feed icon.
### 3. Reader → verify: extend `ReadArticle.test.tsx`
- [x] Reader mounted for both `/feed/:id` and `/reads/:id`; decide Feed vs Reads from `article.list` (not the URL) so Feed favorites opened from Bookmarks behave correctly, and Back goes to the matching list.
- [x] Feed items: swap Archive for "Save to Reads" (`ReadService.moveToReads`, then navigate back); keep Favorite (exempts from 30-day retention) and Delete; no `reading`/`completed`/scroll-position writes.
- [x] `FloatingActionBar` needs a Save-to-Reads icon (check existing set).
### 4. Add Link → verify: `AddLinkModal.test.tsx`
- [x] When detection says `feed`, show a Feed/Reads choice (default Reads) and send `list` in `addURL`. Clear the choice/detection when the URL changes (mobile review lesson: stale feed `list` must not be sent for a page). Direct page saves send no `list`.
- [x] Remove the empty `if (response.type === 'feed') {}` block while touching it; update its comment referencing `/you/feeds`.
### 5. RSS and Newsletters toggles → verify: new tests for both routes
- [x] Shared `SourceListToggle` component (segmented Feed/Reads) with optimistic update, rollback and error message on failure; helper text "Applies to new items only."
- [x] Wire into both screens: RSS key = `rss_data.feed_id`, email key = subscription `id`.
- [x] Rename `/you/feeds` → `/you/rss`, `Feeds.tsx` → `Rss.tsx`, nav label and You/Sidebar entries to RSS, sr-only heading and empty-state copy ("Add a feed URL from Reads or Feed" ...). Unsubscribe stays RSS-only; Newsletters keeps its name.
### 6. Verify end to end
- [x] `tsc --noEmit`, `eslint` (0 errors), `vitest` (53 web tests) and `npm run build` in `apps/web`; shared 32 tests unaffected.
- [ ] Browser run via `webapp-staging-test` against the staging backend with the shared test account: subscribe as Feed → new items in Feed not Reads; toggle to Reads → only new items move; Save to Reads puts the item at top of Reads (needs #421 deployed to staging); RSS/Newsletters toggles persist across reload.
### Open questions
1. **Old URLs (decided 2026-10-11):** `/read` and `/you/feeds` removed outright, no redirects.
2. **Staging deploy:** owner is updating staging to include #421 (2026-10-11); browser run waits on that.

## Progress (2026-10-11)
- Steps 1-5 implemented on `claude/eager-davinci-teaqo1` (the session branch, instead of `feat/web-feed-reads-routes`). Automated checks green (see step 6).
- Notes: `Read.tsx` became `routes/ArticleList.tsx` (one component, `list` prop); reader behaviour keys off `article.list`; Feed has no Add link (Reads-only, like mobile); Bookmarks opens `/{article.list}/{id}`; AddLinkModal clears detection and the Feed/Reads choice on every URL change.
- Staging run (cairn.seatrain.net, 2026-10-11): DB had been reset, so the test account `cairn.web.test@seatrain.net` was re-registered and re-seeded (HN as Feed, The Verge as Reads, example.com page).
  - Verified in headless Chromium: login lands on /reads; Feed|Reads|You nav (sidebar and bottom bar); Add Link asks Feed or Reads for feeds and the choice persisted on the subscriptions; page save lands in Reads; RSS Feed/Reads toggle persists across reload and reverts; /feed and /reads fetch their own list.
  - Not verified: items routed to Feed (no items were delivered to either feed in ~15 min; subscribing does not backfill), so Feed reader, Save to Reads and the Reads-top ordering (#421) are covered by unit tests only. `GET/POST /api/v1/source/email/user/{id}/address` returns 500 on staging, so the Newsletters toggle was not exercised in the browser (unit-tested).
  - To finish: once a feed delivers new items, open one from /feed, Save to Reads, confirm it tops /reads.
- Staging re-run (2026-10-11, after Hacker News delivered 8 items): /feed lists them; the reader shows Back/Favorite/Save to Reads/Next/Delete/Open original (no Archive) and makes no write requests while reading and scrolling; Save to Reads sent `PATCH {list:'reads'}`, removed the item from /feed and put it at the top of /reads (#421 confirmed on staging).
- Newsletters toggle not exercised in the browser: the email-address endpoint works again but the account has no senders (needs an inbound email). It uses the same `SourceListToggle` as RSS (browser-verified) and has unit tests.
