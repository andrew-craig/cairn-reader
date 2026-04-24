---
id: feature_c95a
title: Add feed discovery via HTML scanning
type: feature
status: closed
priority: 2
labels: []
blocked_by: []
parent: null
created_at: 2026-03-21T04:20:37Z
updated_at: 2026-04-24T11:29:27Z
---
When user enters a website URL and taps Find Feed, scan the HTML for RSS/Atom link tags and present discovered feeds.

## Intended Behavior

1. User clicks add → sees URL box, "Add" button, and "Find feed" button
2. On paste, auto-detect URL type. If it's a feed, merge both buttons into one "Add Feed" button
3. "Add" button → backend auto-detects type (feed or page) and saves accordingly (already works)
4. "Find feed" button → probe standard feed locations + parse HTML `<link rel="alternate">` tags to discover the RSS feed for a page. On success, update the URL box with the discovered feed URL so the user can add it
5. If no feed found, show an error message

## Architecture

- **Backend** (Content Service): New `POST /api/v1/content/discover-feed` endpoint with feed discovery logic
- **Mobile**: Call new endpoint from "Find feed" button, update URL box on success, merge buttons when auto-detect finds a feed

## Sub-tasks

- `task_59f7`: Backend feed discovery service
- `task_370d`: Backend discover-feed endpoint
- `task_31b1`: Mobile service method and types
- `task_d23d`: Mobile AddLinkModal UI changes
