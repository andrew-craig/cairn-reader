---
id: task_de93
title: Mobile: end-to-end airplane-mode QA pass for offline reading
type: task
status: open
priority: 3
labels: [mobile,offline,qa]
blocked_by: []
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-05T23:46:28Z
updated_at: 2026-09-12T01:44:23Z
---
Final QA for feature_90a5 once phases 1-4 have landed. Walkthrough on device or simulator in airplane mode: cold start offline near token expiry stays logged in; Read and Bookmarks lists show last-synced content; a synced article opens with its full body (images broken, expected); an unsynced article shows the 'Not available offline' state; offline mark-read, favorite, archive and scroll replay on reconnect with backend state matching and no duplicates; offline banner appears and clears with connectivity; online behaviour of Read/Bookmarks/reader unchanged. Also: coverage check on the store, prefetch and outbox modules; fill in the Review section of feature_90a5.md; capture lessons in LEARNINGS.md.

## On-device airplane-mode checklist (tech lead, 2026-09-12)
Split agreed with the user: an agent does the coverage check, the feature Review section
and the `LEARNINGS.md` entries; this walkthrough needs real hardware and stays with the
user. Task stays open until it is run. Every expectation below was read off the shipped
code, not the plan.

Setup: install a build on device/simulator, sign in **online**, open the Read list and
let it settle so the background prefetch runs (`ArticlePrefetchService`, newest first,
cap `PREFETCH_LIMIT = 100`). Note one article you opened and one you did not.

### A. Offline cold start (task_cab7 — the hard blocker)
- [ ] Force-quit, enable airplane mode, relaunch. Lands in the authenticated tree, **not**
      on LoginScreen. Best signal near token expiry (>55 min since last refresh).
- [ ] The offline banner reads "You're offline" and is visible on the loading branch and,
      if you sign out, on LoginScreen too (task_5bd6).
- [ ] Signed out + offline: LoginScreen shows the offline-specific copy, and the submit
      buttons are still **enabled** (deliberate — see task_5bd6 item 5). Tapping Get
      Started makes exactly one failed attempt, not two.

### B. Offline reading (task_a8a4, task_c55c)
- [ ] Read list renders last-synced articles; Bookmarks renders favourites.
- [ ] A prefetched article opens with its full body immediately. Images are broken —
      expected, image caching is feature_9d64.
- [ ] An article whose body was never prefetched shows "Not available offline", not a
      blank body or a spinner that never resolves.
- [ ] Open a reader from a cold start offline (not from a warm list) — the detail screen
      must resolve the article by id from the store, not only from route params.

### C. Offline mutations (task_ebf1)
While still offline, on different articles: mark one read, favourite one, archive one,
and scroll one at least half way. Each must reflect immediately in the UI and survive
backgrounding the app.
- [ ] Leave airplane mode. The banner clears and the outbox drains on reconnect.
- [ ] Verify against the backend (web app or API) that all four landed: status,
      is_favorite, scroll_position, delete.
- [ ] No duplicates and nothing lost. Archive an article offline, reconnect, then archive
      it again — the replayed DELETE 404 must count as success, not surface an error.
- [ ] Scroll one article to several positions offline; exactly one scroll update should
      reach the server (coalesced per article).

### D. No online regressions
- [ ] Fully online: Read list, Bookmarks, reader, pull-to-refresh, add-URL and Explore all
      behave as before. Explore stays online-only by design — it must **not** be cached.
- [ ] Sign out clears the local store: sign out, sign back in offline-capable, confirm no
      previous account's articles are readable.

Record results here, then close. Anything found becomes its own task, not a fix folded in.
