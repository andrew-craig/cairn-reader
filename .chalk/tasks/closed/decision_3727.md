---
id: decision_3727
title: Pick canonical sanitizer policy and User-Agent for all RSS ingestion
type: decision
status: closed
priority: 1
labels: []
blocked_by: []
parent: epic_6d46
remote_task_url: null
created_at: 2026-05-02T04:17:42Z
updated_at: 2026-05-02T08:31:16Z
---
# Single source of truth for sanitizer policy and User-Agent

## Why

Three different bluemonday policies and three different User-Agent strings are in use:

User-Agents:
- Explore fetcher: gofeed default (no explicit UA set) - services/explore/fetcher/internal/fetcher/fetcher.go:142 area
- Read fetcher: 'Cairn-RSS-Fetcher/1.0' - services/read/fetcher/internal/fetcher/parser.go:51, conditional_fetcher.go:51
- Read content: 'Mozilla/5.0 (compatible; CairnBot/1.0; +https://github.com/cairn-app/cairn-reader/services/read)' - services/read/content/internal/processor/content.go:134

Bluemonday policies:
- Explore: services/explore/fetcher/internal/sanitizer/sanitizer.go:38-71 (strips <audio>, <video>, <col>, class, id)
- Read content: services/read/content/internal/processor/sanitizer.go:14-62 (keeps <audio>, <video>, <col>, validated class/id, srcset/sizes/media on <source>)

This means the same article can render differently on the Explore feed vs. a user's reading list, and feeds that gate on User-Agent serve different bytes to the two services.

## Requirements

- Decide the canonical User-Agent. Recommendation: 'CairnBot/1.0 (+https://github.com/cairn-app/cairn-reader)' - identifies the bot, has a contact link, no version drift between RSS and content paths.
- Decide the canonical bluemonday policy. Recommendation: Read content's superset (allows audio/video/class/id) - it's strictly more permissive, so adopting it across both services means no content currently allowed becomes blocked.
- Document the decision in this task's review section before any code lands.
- Confirm with the recommender team that adopting the more permissive policy is safe for the recommendation pipeline (no downstream code assumes audio/video are stripped).

## Acceptance

- Decision recorded in this task. Resulting constants land in pkg/rss/ in task_<extract-pkg-rss>.

## Decision (2026-05-02)

### Canonical User-Agent

```
CairnBot/1.0 (+https://github.com/cairn-app/cairn-reader)
```

Used by every outbound HTTP call originating from RSS ingestion: feed polling (Explore + Read fetchers) and full-page content fetches (Read Content). Lands in `pkg/rss/fetch/` as an exported constant, e.g.:

```go
// pkg/rss/fetch/useragent.go
package fetch

// UserAgent is the canonical User-Agent for all Cairn RSS-ingestion HTTP traffic.
const UserAgent = "CairnBot/1.0 (+https://github.com/cairn-app/cairn-reader)"
```

Rationale:
- Single string for both RSS polling and full-content fetches — no version drift between paths, no risk of feeds gating on UA serving different bytes to the two services.
- `CairnBot/1.0` identifies the bot; the GitHub URL gives feed operators a contact point and a place to file abuse reports without requiring the marketing site to host a bot-info page.
- Drops the `Mozilla/5.0 (compatible; ...)` prefix used in `services/read/content/internal/processor/content.go:134`. That prefix exists to dodge UA-based blocking, but it conflates "I'm a browser" with "I'm a bot" and is dishonest. Sites that block well-identified bots will block us either way; we'd rather be transparent.

### Canonical sanitizer policy

Adopt the **Read Content service's policy** verbatim — `services/read/content/internal/processor/sanitizer.go:14-62` — as the single policy used by every sanitizer pass in the pipeline. Lands in `pkg/rss/sanitize/` as `Policy()`/`Sanitize()`.

Concrete allowlist (the superset):

- Text: `p`, `br`, `span`, `div`, `article`, `section`, `h1`–`h6`, `strong`, `b`, `em`, `i`, `u`, `s`, `del`, `ins`, `blockquote`, `q`, `cite`, `code`, `pre`, `kbd`, `samp`, `var`
- Lists: `ul`, `ol`, `li`, `dl`, `dt`, `dd`
- Tables: `table`, `thead`, `tbody`, `tfoot`, `tr`, `th`, `td`, `caption`, `col`, `colgroup`; `colspan`/`rowspan` on `th`/`td`; `scope` on `th`
- Links: `a` with `href`, `title`; `RequireNoReferrerOnLinks(true)`
- Media: `img` with `src`, `alt`, `title`, integer `width`/`height`; `figure`, `figcaption`, `picture`, `source` with `srcset`, `sizes`, `type`, `media`
- Audio/video: `audio`, `video` with `src`, `controls`
- Attributes: `class` (space-separated tokens) on `p`, `div`, `span`, `blockquote`, `code`, `pre`; `id` (space-separated tokens) on `h1`–`h6`
- URL schemes: `http`, `https`, `mailto`

Rationale:
- It's a strict superset of Explore's policy — no content currently rendered by Explore becomes blocked when Explore migrates, so this is a one-way ratchet with no user-visible regression.
- Audio, video, and `<source>` `srcset`/`sizes`/`media` are needed for podcast feeds and responsive images that already render in the Read app today; stripping them would visibly degrade the reading view.
- `class` and `id` are scoped to a small validated set of elements (text containers and headings) and matched against `bluemonday.SpaceSeparatedTokens`, so they don't open a styling-injection vector against any consumer that ships the article HTML to a browser inside its own page.
- bluemonday is centralized in one package — drift physically can't reappear without someone adding a second import in another service, which the migration tasks (`task_5ee6`, `task_69fd`) explicitly forbid.

### Safety check on the more permissive policy

Audited the Explore Recommender (`services/explore/recommender/`) end-to-end for any code path that would trip on the newly-allowed tags/attributes:

- No HTML parsing in the recommender — `description`/`content` are opaque strings throughout.
- No text extraction, tokenization, embeddings, or full-text indexing on article bodies (recommendation engine in `recommender/internal/recommend/engine.go` works only off vote/recommend counts).
- No size CHECK constraints on the `description`/`content` columns in `recommender/migrations/`; the 10 MB request body limit comfortably fits feeds with media tags.
- Recommender returns raw JSON; no server-side HTML rendering.

Conclusion: adopting the superset is safe for the recommendation pipeline.

Note on the original requirement to "confirm with the recommender team": the recommender is owned in-repo and there is no separate team to consult. The code audit above stands in for that confirmation.

### What lands where

- `pkg/rss/fetch/`: `UserAgent` constant; `Fetch()` sets it on every request.
- `pkg/rss/sanitize/`: `Policy()` builder + `Sanitize(html string) string`. Carries forward Read Content's allowlist verbatim.
- Migration tasks (`task_5ee6`, `task_69fd`) delete `services/explore/fetcher/internal/sanitizer/sanitizer.go`, `services/read/content/internal/processor/sanitizer.go`, the inline UA at `services/read/fetcher/internal/fetcher/parser.go:51` and `conditional_fetcher.go:51`, the inline UA at `services/read/content/internal/processor/content.go:134`, and switch the Explore fetcher (currently using gofeed's default UA at `services/explore/fetcher/internal/fetcher/fetcher.go:142`) to `pkg/rss/fetch`.

### Follow-ups (not in this task)

- Sanitizer fixture suite in `pkg/rss/sanitize/testdata/` covering each kept/stripped tag — defined in `feature_610d` acceptance.
- A grep-based CI check (proposed in `feature_610d`) ensuring no service imports `bluemonday` or sets a `User-Agent` header outside `pkg/rss/`.
