---
id: decision_3727
title: Pick canonical sanitizer policy and User-Agent for all RSS ingestion
type: decision
status: open
priority: 1
labels: []
blocked_by: []
parent: epic_6d46
remote_task_url: null
created_at: 2026-05-02T04:17:42Z
updated_at: 2026-05-02T04:17:42Z
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
