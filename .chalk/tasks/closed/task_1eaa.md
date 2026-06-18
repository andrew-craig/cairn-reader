---
id: task_1eaa
title: Audit 3: Mobile client/server efficiency
type: task
status: closed
priority: 2
labels: [audit,mobile]
blocked_by: []
parent: epic_9c21
remote_task_url: null
created_at: 2026-06-06T05:06:32Z
updated_at: 2026-06-16T10:32:52Z
---
Audit how the React Native client consumes the API for over-fetching, wasted calls, and resilience. apps/mobile/src/services/{auth,read,explore}.ts and screens (ExploreScreen, ReadScreen, ReadArticleDetailScreen).

Findings to confirm/resolve:
1. HIGH: getUserVoteCounts() fetches ALL votes with limit=10000 to count client-side (explore.ts ~line 269). Needs a backend aggregate endpoint returning {upvotes,downvotes}. Cross-ref Audit 1 (new endpoint) + Audit 2 (votes table scan).
2. Explore over-fetching: fixed 10/page vs ~12 needed buffer => always 2 pages, ~8 wasted items/load. Consider dynamic ?limit.
3. Search has no pagination/has_more handling — large result sets truncated silently.
4. No soft cache/TTL: every screen focus refetches (useFocusEffect) — bandwidth/battery cost on tab switches. Consider stale-while-revalidate.
5. No retry/backoff for transient 5xx/network (only single 401->refresh->retry). Decide policy for beta.
6. Offline: AsyncStorage holds articles but isn't used as a read-through cache; no offline queue. Decide minimum offline behaviour for beta.
7. Confirm token refresh dedup/proactive-refresh is correct (looks good — verify).

Deliverable: findings section ranking each client inefficiency, the server-side changes it implies (feed into Audit 1), and beta-blocking vs fast-follow.
