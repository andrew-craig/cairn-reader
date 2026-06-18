# Audit 3 — Mobile Client/Server Efficiency

**Workstream:** `task_1eaa` (epic `epic_9c21`, P3)
**Status:** Findings complete
**Date:** 2026-06-16

> This is the mobile client/server efficiency workstream. The question is not "does it work" (it does)
> but "what wastes bandwidth, battery, or data-correctness on every real user device before we open
> public beta". Every classified **BETA-BLOCKING** item must be resolved before signups open.

## How to read the classification

| Class | Meaning | Why |
|---|---|---|
| **BETA-BLOCKING** | Incorrect behaviour, silent data truncation, or a contract-affecting server change that cannot be deferred once clients are in the wild | Reconcile before signups open |
| **FAST-FOLLOW** | Real efficiency or UX issue, but safe to ship to beta and fix during beta | Fix during beta |
| **ROADMAP** | Correct at launch scale; documented so it is not rediscovered as a surprise | No action now |

## Summary table

| ID | Finding | Class |
|---|---|---|
| M-1 | `getUserVoteStats()` requests `limit=10000` but server hard-caps at 100 → counts are silently wrong for active users | **BETA-BLOCKING** |
| M-2 | Explore search is client-side only — no backend `?q=` call, large result sets are filtered from whatever is already in memory | **BETA-BLOCKING** |
| M-3 | `useFocusEffect` full-reset refetch on ReadScreen (and YouScreen, BookmarksScreen, VotesScreen) — every tab switch fires a fresh network round-trip | FAST-FOLLOW |
| M-4 | No retry/back-off for transient 5xx or network errors — a single failure surfaces to the user immediately | FAST-FOLLOW |
| M-5 | AsyncStorage holds articles locally but is never used as a read-through cache; app renders nothing until the network responds | FAST-FOLLOW |
| M-6 | Explore page size constant matches the backend (10/page) — no over-fetch waste on normal scrolling | PASS |
| M-7 | Token refresh dedup and proactive refresh are implemented correctly | PASS |

---

## M-1 — `getUserVoteStats()` fetches votes with `limit=10000`; server caps at 100 (BETA-BLOCKING)

**Correction to the original lead.** The lead stated this function fetches ALL votes client-side
and counts them. That is correct, but the lead understated the severity: the server enforces a hard
cap of 100, not 10000, so the client's assumption that it received all votes breaks silently for any
user with more than 100 votes.

**Current state — client (`apps/mobile/src/services/explore.ts:265-289`):**

```typescript
static async getUserVoteStats(): Promise<{ upvotes: number; downvotes: number }> {
  // Fetch all votes with a high limit to get complete counts
  const response = await this.fetchWithAuth(
    `${getServerUrl()}/api/v1/explore/user/votes?limit=10000&offset=0`
  );
  // ...
  const upvotes = data.votes.filter((v) => v.vote_type === 'upvote').length;
  const downvotes = data.votes.filter((v) => v.vote_type === 'downvote').length;
}
```

**Current state — server (`services/explore/recommender/internal/api/handlers.go:428-431`):**

```go
if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 && parsed <= 100 {
    limit = parsed
}
```

When `limit=10000` is sent, the `parsed <= 100` condition is false so the server silently falls
back to its default of `limit=20`. The client then counts only the first 20 votes and displays
that count as the total on the "You" screen (`apps/mobile/src/screens/YouScreen.tsx:87-98`). A user
with 50 votes sees "20 up votes, 0 down votes" instead of their real tally. The data is wrong
without any error signal.

**Also: even if the cap were raised to 10000, the response would carry full article bodies (title,
description, content, categories...) for every voted article — the same D-1 payload concern Audit 2
raised. `GetUserVotedArticles` SELECTs all article columns
(`services/explore/recommender/internal/db/vote_repository.go:259-270`), not just vote counts.**

**Recommendation (BETA-BLOCKING — feeds Audit 1 API-freeze checklist):**

Add a backend aggregate endpoint:

```
GET /api/v1/explore/user/vote-stats
→ { data: { upvotes: N, downvotes: N } }
```

The implementation is a single SQL count:

```sql
SELECT
    COUNT(*) FILTER (WHERE vote_type = 'upvote') AS upvotes,
    COUNT(*) FILTER (WHERE vote_type = 'downvote') AS downvotes
FROM votes v
JOIN users u ON v.user_id = u.id
WHERE u.user_id = $1
```

This returns `{upvotes, downvotes}` in O(1) (one indexed scan on `idx_votes_user`) with no article
payload. The mobile client calls this single endpoint instead of `getUserVoteStats`. Remove
`getUserVoteStats` from the client once the endpoint lands.

**Effort:** backend ~30 lines (handler + repository method + route); client ~5 lines (replace the
call site in `YouScreen.tsx`). Low.

---

## M-2 — Explore search is client-side only; results are bounded by what is already in memory (BETA-BLOCKING)

**Current state (`apps/mobile/src/screens/ExploreScreen.tsx:362-379`):**

```typescript
const filteredArticles = useMemo(() => {
  if (!searchQuery) return articles;
  const q = searchQuery.toLowerCase();
  return articles.filter(
    (a) =>
      a.title.toLowerCase().includes(q) ||
      a.description?.toLowerCase().includes(q) ||
      a.author?.toLowerCase().includes(q)
  );
}, [articles, searchQuery]);
```

The `articles` array is only what has been loaded so far in the scroll session (typically 10–20
items after an initial load). A user who types a search term will silently miss all articles that
have not been paged into memory yet. There is no call to a backend `?q=` search endpoint from the
Explore tab at all.

**Contrast with ReadScreen (`apps/mobile/src/screens/ReadScreen.tsx:83-107`):** the Read screen
calls `ReadService.searchUserContents({ q: query })` which hits the backend full-text search
(`GET /api/v1/content/user/{id}/search`). The Explore tab has no equivalent.

**Why it is beta-blocking:** the explore feed can contain hundreds or thousands of articles across
all users; a user searching for "climate" will see only the subset already rendered. This is a
silent data-truncation bug from the user's perspective, not just a performance concern. Making it
work correctly requires a server-side search endpoint on the recommender.

**Recommendation (BETA-BLOCKING — feeds Audit 1 API-freeze checklist):**

Add a search endpoint to the recommender:

```
GET /api/v1/explore/search?q=<term>&limit=20&offset=0
→ { data: { articles: [...], count: N } }
```

The client replaces the `useMemo` filter with a call to this endpoint when `searchQuery` is set,
analogous to how ReadScreen works. Until the endpoint exists, either remove the search icon from
the Explore tab (to avoid implying full-corpus search) or annotate it clearly as "filter loaded
articles" — but shipping the silent truncation as real search is a user-trust issue for beta.

**Effort:** backend ~60 lines (full-text or LIKE query on articles table + handler + route); client
~20 lines (new `ExploreService.searchArticles` method + replace the `useMemo` filter).

---

## M-3 — `useFocusEffect` full-reset refetch on every tab return (FAST-FOLLOW)

**Current state:** four screens call `useFocusEffect` with a full-reset reload on every focus
event:

| Screen | File | What fires on focus |
|---|---|---|
| ReadScreen | `apps/mobile/src/screens/ReadScreen.tsx:72-81` | `loadReadArticles(reset=true)` — discards cursor, refetches page 1 |
| YouScreen | `apps/mobile/src/screens/YouScreen.tsx:79-126` | `fetchStats()` — three parallel network calls (votes, subscriptions, bookmarks) |
| BookmarksScreen | `apps/mobile/src/screens/BookmarksScreen.tsx:96+` | Full reset load |
| VotesScreen | `apps/mobile/src/screens/VotesScreen.tsx:56-60` | Full reset load (`loadVotes(reset=true)`) |

Every time the user switches tabs and returns — even after one second — the full list is discarded
and re-fetched from page 1. On a mobile connection this is visually jarring (spinner re-appears)
and wastes bandwidth.

**Note on ExploreScreen:** ExploreScreen does NOT use `useFocusEffect`; it loads once on mount and
then paginates on demand. It does flush shown-events on `navigation.blur`
(`ExploreScreen.tsx:244-250`), which is correct. Only the screens above have the refetch pattern.

**Note on ReadScreen rationale:** the comment in ReadScreen (`ReadScreen.tsx:69-71`) explicitly
justifies this: "Reload the list whenever the screen regains focus (e.g. after returning from the
article detail screen where an article may have been archived)." This is a legitimate correctness
concern — a read/archive action on the detail screen should be reflected in the list. The issue is
that a full network round-trip is the only mechanism, even when nothing has changed.

**Recommendation (FAST-FOLLOW):**

For ReadScreen specifically, the refetch-on-focus is functionally correct and relatively low cost
(cursor-paginated, only 20 items). Accept it for now but consider a lightweight approach for beta:
pass a "needs refresh" flag back via navigation params when the detail screen makes a mutation, so
the list only refetches after an actual change.

For YouScreen the three-parallel-call pattern is proportionally more expensive and the displayed
counts do not need to be real-time accurate (they are informational, not transactional). A 60-second
client-side TTL (timestamp + cached result in component state or a module-level cache) would
eliminate the majority of focus-triggered calls with no correctness impact.

For BookmarksScreen and VotesScreen, the same TTL approach applies.

**Effort per screen:** 1–2 hours each to add a TTL guard. Low.

---

## M-4 — No retry/back-off for transient 5xx or network errors (FAST-FOLLOW)

**Current state:** both `ExploreService.fetchWithAuth` (`explore.ts:50-100`) and
`ReadService.fetchWithAuth` (`read.ts:23-72`) implement exactly one retry path: a 401 response
triggers a token refresh and a single retry. For all other errors (5xx, network timeout, DNS
failure), the error propagates immediately to the caller with no retry.

The `detectURL` method (`read.ts:270-304`) is the only caller that adds a client-side timeout
(10-second AbortController). No other call site has a timeout or retry.

**What happens in practice:**
- A transient 502 from the load balancer causes an `Alert.alert('Error', ...)` on ReadScreen and an
  empty list on ExploreScreen.
- A cold-start backend that is slow to respond times out at the OS default (typically 30-60 s on
  mobile) rather than a configured value.

**Recommendation (FAST-FOLLOW):**

Define a beta retry policy and encode it once in `fetchWithAuth`:
- Retry up to 2 times on network error or 5xx, with exponential back-off (e.g. 500ms, 1500ms).
- Add a per-request timeout (e.g. 15s) via `AbortController` to match `detectURL`'s pattern.
- Do not retry 4xx (except the existing 401 path).

This is a single change to the shared `fetchWithAuth` helper in each service file, automatically
covering all callers. The `auth.ts` fetch calls (login, register, refresh) use raw `fetch` directly
and should keep their own simpler handling since they are user-initiated flows.

**Effort:** ~40 lines across `read.ts` and `explore.ts`. Low.

---

## M-5 — AsyncStorage is write-only from the client perspective; no read-through on load (FAST-FOLLOW)

**Current state:** `StorageService` (`apps/mobile/src/services/storage.ts`) provides `getArticles`,
`saveArticles`, `addArticle`, `updateArticle`, `deleteArticle`. It is used in
`ReadArticleDetailScreen.tsx` to persist scroll position and read status locally alongside the
backend PATCH call. It is **not** consulted when loading the list: `ReadScreen` calls
`ReadService.listUserContents()` directly; if that fails, the screen renders an empty list and
shows an error alert. The locally cached articles in AsyncStorage are never shown.

**What the user sees:** on a flight or in a poor-signal area, opening the app shows a spinner that
eventually resolves to an error and an empty list, even though the user's articles were cached from
the last session.

**What AsyncStorage currently stores:** the `@cairnreader:articles` key holds the `Article[]` array.
Looking at how it is populated: `StorageService.addArticle` is called from `ReadArticleDetailScreen`
when adding to read list, and `updateArticle` for scroll/favorite. The store is written piecemeal,
not kept in sync with the server list. It is therefore a partial, possibly-stale local view.

**Recommendation (FAST-FOLLOW):**

For beta, define a minimum offline behaviour: on load, show the last-known cached list immediately,
then silently refresh from the network and update the list if the response succeeds. The pattern
is: `getArticles()` → render → `listUserContents()` → merge & save. This requires reconciling the
two representations (`UserContentResponse` vs `Article`), which is the main engineering cost.

A simpler interim for beta: keep the current network-first approach but show the cached list
(stale) when the network call fails, with a visible "Offline — showing cached data" banner. This
requires no reconciliation logic and prevents the blank-screen failure mode.

**Deciding the minimum offline behaviour is a product decision**, not just an engineering one.
The engineering effort varies: stale-fallback is ~2 hours; full stale-while-revalidate is ~1 day.

**Effort:** 2 hours (stale-fallback) to 1 day (stale-while-revalidate).

---

## M-6 — Explore page size matches backend (PASS)

**The original lead ("fixed 10/page vs ~12 needed buffer") was partially stale and resolves to no
action.**

The backend (`services/explore/recommender/internal/recommend/engine.go:31`) defines
`recommendationPageSize = 10` and always returns exactly that many items (or fewer when the pool
is exhausted). The mobile client declares `RECOMMENDATION_PAGE_SIZE = 10`
(`apps/mobile/src/screens/ExploreScreen.tsx:28`) and uses this constant to detect end-of-feed
(`newRecommendations.length < RECOMMENDATION_PAGE_SIZE`). The values match.

The initial-load loop correctly buffers past the first page until it has filled the screen:
`calculateInitialArticleCount()` computes `ceil(screenHeight / 95px) * 1.5`, which for a typical
phone (~800px) is ~13 articles, triggering a second fetch automatically. This is intentional and
correct — it is not waste, it is a viewport-fill strategy that the screen explicitly calculates
and comments (`ExploreScreen.tsx:34-41`).

**No action.**

---

## M-7 — Token refresh dedup and proactive refresh are implemented correctly (PASS)

**Confirmed correct.** `AuthService.refreshAccessToken` (`apps/mobile/src/services/auth.ts:324-341`)
uses a `isRefreshing` flag and a shared `refreshPromise`:

```typescript
if (this.isRefreshing && this.refreshPromise) {
    return this.refreshPromise;  // concurrent callers wait on the same promise
}
this.isRefreshing = true;
this.refreshPromise = this.doRefreshAccessToken();
```

Concurrent callers (e.g. two simultaneous `fetchWithAuth` calls that each detect a 401) will
block on the same in-flight refresh rather than firing duplicate refresh requests. The `finally`
block clears both flags so the state machine resets correctly after success or failure
(`auth.ts:334-340`).

Proactive refresh fires via `ensureValidToken` (`auth.ts:290-322`) before every authenticated
request. It reads `expiresAt` from the in-memory field (seeded from AsyncStorage at `initialize()`)
and considers the token expired 5 minutes before its actual expiry
(`TOKEN_EXPIRATION_BUFFER_MS = 5 * 60 * 1000`, `auth.ts:20`). This is a standard proactive-refresh
pattern and is correctly implemented.

**One minor observation (not a bug):** on a token-refresh failure in `doRefreshAccessToken`,
`clearTokens()` is called unconditionally (`auth.ts:420-428`), which logs the user out even for
transient network errors during the refresh. This is a conservative but deliberate choice; if it
becomes a UX complaint in beta, a retry on network error before clearing could be added. Not
flagging as a finding.

**No action.**

---

## Mobile checklist for Audit 7

### Beta-blocking items (must resolve before signups open)

- [ ] **M-1** Add `GET /api/v1/explore/vote-stats` endpoint returning `{upvotes,downvotes}` aggregate;
      replace `getUserVoteStats()` client call that incorrectly sends `limit=10000`. *(Feeds Audit 1
      API-freeze checklist — new endpoint must be specced before contract freeze.)*
- [ ] **M-2** Either add `GET /api/v1/explore/search?q=` server-side search endpoint and wire it in
      `ExploreScreen`, or remove the search icon from Explore to avoid silent result truncation. *(Feeds
      Audit 1 API-freeze checklist — new endpoint must be specced before contract freeze.)*

### Fast-follow items (safe to ship to beta, fix during beta)

- [ ] **M-3** Add TTL guard to YouScreen `useFocusEffect` stats fetches; evaluate ReadScreen/Bookmarks/Votes.
- [ ] **M-4** Add retry/back-off (2 retries, exponential) + per-request timeout (15s) to `fetchWithAuth`
      in `read.ts` and `explore.ts`.
- [ ] **M-5** Define and implement minimum offline behaviour (at minimum: show stale cached list when
      network call fails rather than blank screen).

### Server contract changes implied (coordinate with Audit 1 before freeze)

| New endpoint | Service | Purpose | Notes |
|---|---|---|---|
| `GET /api/v1/explore/user/vote-stats` | Explore recommender | Return `{upvotes,downvotes}` aggregate | Replaces the limit=10000 workaround in M-1 |
| `GET /api/v1/explore/search` | Explore recommender | Full-corpus article search with `?q=` | Required for M-2; analogous to `/api/v1/content/user/{id}/search` on the read side |

Both endpoints must appear in `services/explore/api/openapi.yaml` before the API contract freezes
(Audit 1 finding F-7 already flags undocumented explore routes as FAST-FOLLOW; these two are
BETA-BLOCKING because the client correctness depends on them).
