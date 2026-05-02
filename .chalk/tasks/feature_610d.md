---
id: feature_610d
title: Extract pkg/rss/ shared package (parse, sanitize, readability, fetch)
type: feature
status: open
priority: 1
labels: []
blocked_by: [task_7479,decision_3727]
parent: epic_6d46
remote_task_url: null
created_at: 2026-05-02T04:18:02Z
updated_at: 2026-05-02T04:18:02Z
---
# Create pkg/rss/ as the single home for stateless RSS-ingestion primitives

## Why

The Explore fetcher, Read fetcher, and Read Content Service all reach for gofeed, bluemonday, and go-readability with subtly different wrappers. Once the readability consolidation (task_7479) and policy decision (decision_3727) land, the wrapper code is identical enough to share.

## Requirements

Create pkg/rss/ with these subpackages. Each subpackage has its own Go module path, godoc, and unit tests.

### pkg/rss/parse/
- gofeed wrapper that produces a canonical struct:
  type Item struct {
    GUID, Title, Author string
    Link string
    PublishedAt *time.Time
    Description string  // raw RSS <description>
    Content     string  // <content:encoded> if present, else Description
  }
  type Feed struct {
    Title, SiteURL, Description string
    Items []Item
  }
- ParseBytes([]byte) (*Feed, error) and ParseReader(io.Reader) (*Feed, error).
- Author resolution order: gofeed item.Author.Name -> dc:creator -> Atom <author><name> -> empty string. Document the limitation (no <author email=...> fallback) in godoc.

### pkg/rss/sanitize/
- Single bluemonday policy implementing the decision from decision_3727.
- Exported func Policy() *bluemonday.Policy and func Sanitize(html string) string.
- Tests: each kept/stripped tag/attribute is covered with one positive and one negative case.

### pkg/rss/readability/
- Thin wrapper around go-shiori/go-readability. Returns a struct with Title, Author, Content (HTML), Excerpt, Image, Length.
- Exported func Extract(url, rawHTML string) (*Result, error).

### pkg/rss/fetch/
- HTTP client with: configurable timeout (default 30s), 10-redirect cap, optional ETag/If-None-Match and Last-Modified/If-Modified-Since support, canonical User-Agent constant from decision_3727.
- Exported func Fetch(ctx, url, FetchOpts) (*Response, error) where Response carries Body []byte, ETag, LastModified, NotModified bool, StatusCode int.
- Connection pooling tuned for multi-feed use (MaxIdleConnsPerHost 10, IdleConnTimeout 90s).

### pkg/rss/hash/
- ContentHash(html []byte) string returning lowercase hex SHA-256.
- One implementation only - both services adopt this so dedup keys are consistent.

## Out of scope for this task

- No service migration in this task (handled in task_<migrate-explore> and task_<migrate-read>).
- No scheduling, no outbox, no DB code.

## Tests

- Each subpackage has table-driven unit tests covering happy path + at least one error case.
- Snapshot test for sanitizer against ~5 real-world feed item HTML samples (commit fixtures under pkg/rss/sanitize/testdata/).

## Acceptance

- go test ./pkg/rss/... passes.
- godoc renders cleanly for all exported symbols.
- No service code imports gofeed, bluemonday, or go-readability directly anymore. (Enforced as a follow-up via grep in the migration tasks.)
