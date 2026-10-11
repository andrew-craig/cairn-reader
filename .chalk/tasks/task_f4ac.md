---
id: task_f4ac
title: Web: Feed and Reads routes
type: task
status: in_progress
priority: 2
labels: [web]
blocked_by: []
parent: epic_6e4d
remote_task_url: null
created_at: 2026-10-08T11:32:05Z
updated_at: 2026-10-11T03:11:56Z
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
- [ ] `listUserContents`, `countUserContents`, `searchUserContents` send `list` when given.
- [ ] Add `setSourceList(type, key, list)` (PUT `/content/user/{id}/subscriptions/{type}/{key}/list`, same error handling as mobile) and `moveToReads(id)` = `updateUserContent(id, {list:'reads'})`.
- [ ] Tests: query-string cases, PUT URL/body, error path.
### 2. Routes: Reads and Feed → verify: new `Reads`/`Feed` route tests, existing tests green
- [ ] Rename `/read` → `/reads` (`Read.tsx` → `Reads.tsx`, `Read.css` stays shared) and `/read/:id` → `/reads/:id`; fetch with `list:'reads'`.
- [ ] New `Feed.tsx` at `/feed`, `/feed/:id`. The infinite-scroll/refresh/observer logic in `Reads.tsx` is ~100 lines; extract it into a `useCursorList(fetchPage)` hook used by both rather than copying (mirrors mobile's `useCursorArticleList`). Feed: no unread/status UI; search placeholder "Search your feed".
- [ ] Update every `/read` reference: App routes + `RootRedirect` (→ `/reads`), Login, BottomNav, Sidebar, SearchModal navigate, ReadArticle Back/Archive/Delete.
- [ ] `SearchModal` takes a `list` prop and passes it to `searchUserContents`, so a Feed search can't return Reads items.
- [ ] Nav: BottomNav and Sidebar become Feed | Reads | You; add a Feed icon.
### 3. Reader → verify: extend `ReadArticle.test.tsx`
- [ ] Reader mounted for both `/feed/:id` and `/reads/:id`; decide Feed vs Reads from `article.list` (not the URL) so Feed favorites opened from Bookmarks behave correctly, and Back goes to the matching list.
- [ ] Feed items: swap Archive for "Save to Reads" (`ReadService.moveToReads`, then navigate back); keep Favorite (exempts from 30-day retention) and Delete; no `reading`/`completed`/scroll-position writes.
- [ ] `FloatingActionBar` needs a Save-to-Reads icon (check existing set).
### 4. Add Link → verify: `AddLinkModal.test.tsx`
- [ ] When detection says `feed`, show a Feed/Reads choice (default Reads) and send `list` in `addURL`. Clear the choice/detection when the URL changes (mobile review lesson: stale feed `list` must not be sent for a page). Direct page saves send no `list`.
- [ ] Remove the empty `if (response.type === 'feed') {}` block while touching it; update its comment referencing `/you/feeds`.
### 5. RSS and Newsletters toggles → verify: new tests for both routes
- [ ] Shared `SourceListToggle` component (segmented Feed/Reads) with optimistic update, rollback and error message on failure; helper text "Applies to new items only."
- [ ] Wire into both screens: RSS key = `rss_data.feed_id`, email key = subscription `id`.
- [ ] Rename `/you/feeds` → `/you/rss`, `Feeds.tsx` → `Rss.tsx`, nav label and You/Sidebar entries to RSS, sr-only heading and empty-state copy ("Add a feed URL from Reads or Feed" ...). Unsubscribe stays RSS-only; Newsletters keeps its name.
### 6. Verify end to end
- [ ] `npm run typecheck`, `lint`, `test` in `apps/web` (and `apps/shared` if touched).
- [ ] Browser run via `webapp-staging-test` against the staging backend with the shared test account: subscribe as Feed → new items in Feed not Reads; toggle to Reads → only new items move; Save to Reads puts the item at top of Reads (needs #421 deployed to staging); RSS/Newsletters toggles persist across reload.
### Open questions
1. **Old URLs:** CLAUDE.md says no compatibility layers, so `/read` and `/you/feeds` are removed outright, not redirected. Say so if you want redirects for bookmarked links.
2. **Staging deploy:** step 6 needs the backend routing + #421 on staging; I'll check before the browser run and report if they aren't there.
