# Cairn Web App - Explore Requirements (Archived)

> **Archived.** The Explore tab and backend were removed in #425 (Oct 2026). These requirements were moved out of `docs/detailed_requirements/web_app_requirements.md` for historical reference, with the original wording preserved. They do not describe shipped behaviour. See also [explore_product_requirements.md](explore_product_requirements.md).

## Reuse and architecture mentions

- Reused mobile service: `services/explore.ts` (alongside `auth.ts` and `read.ts`).
- Architecture diagram entry: `Explore Service (recommendations, votes)` behind the same REST + JWT bearer API.
- Data models: the `transformArticle` (explore) mapping carried over alongside `transformToArticle` (read).

## Navigation and routes

The mobile bottom tab bar was `Read`, `Explore`, `You`; the web sidebar mirrored it with the same three destinations.

| Route | Screen (mobile equivalent) | Purpose |
|---|---|---|
| `/explore` | `ExploreScreen` | Recommended articles to discover |
| `/explore/:id` | `ExploreArticleDetailScreen` | Full article reader (recommendation) |
| `/you/votes` | `VotesScreen` | Articles the user up/down-voted |

## FR-4 (excerpt): Article Reader for recommendations

- Recommendations used the explore `content` field (saved articles use `content.cleaned_html`).
- For recommendation articles (`/explore/:id`): support **mark as read** (`POST /api/v1/explore/article/{id}/read`) and **up/down vote** (see FR-6), plus saving the article to the reading list.

## FR-5: Explore (Discovery)

- Fetch recommendations via `GET /api/v1/explore/recommendation?offset=...` (`ExploreService.getRecommendations`), paginated by offset.
- Display recommended articles in a card layout suited to desktop (e.g. a responsive grid or a centered feed column).
- Best-effort **"shown" reporting** via `POST /api/v1/explore/shown` (batched article IDs), matching mobile telemetry behavior.
- Support **up/down vote** directly from the feed and from the reader (FR-6).

## FR-6: Voting

- Up/down vote and remove vote on recommendation articles:
  - `POST /api/v1/explore/article/{id}/vote` with `{ vote_type: 'upvote' | 'downvote' }`.
  - `DELETE /api/v1/explore/article/{id}/vote` to remove.
  - `GET /api/v1/explore/article/{id}/vote` for counts and the user's current vote.
- Optimistic UI for vote toggles is acceptable, matching the responsive feel of mobile.

## FR-7 (excerpt): You counts

- **Votes** item in the You sub-menu, with a count from vote stats (`ExploreService.getUserVoteStats`).
