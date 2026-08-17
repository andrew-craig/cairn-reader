---
id: task_b615
title: [Audit/Tier 1] URL validation is skipped whenever content_id is also present, on the flow that prefers the URL
type: task
status: open
priority: 1
labels: [quality,security,audit]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T12:49:39Z
updated_at: 2026-08-17T12:49:39Z
---
**Source:** Cairn Simplification Audit — https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f · file:line detail supplied by the audit author 2026-08-17 and re-verified at HEAD `a6c56a1`.
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. One finding, one branch, one PR.

**Audit tier:** 1 (correctness & security) | **Verified.**

## Problem
`AddContentToUserRequest.Validate()` gates **both** the presence check and the URL format check on `ContentID` being nil — `services/read/content/internal/api/dto/user_content.go:36-41`:
```go
validation.Field(&a.URL,
    validation.When(a.ContentID == nil,
        validation.Required.Error("Either 'url' or 'content_id' is required"),
        is.URL.Error("URL must be a valid URL"),
    ),
),
```
But the handler **prefers the URL whenever it is non-empty**, regardless of `ContentID` — `services/read/content/internal/api/handlers/user_content_handler.go:309-318`:
```go
if req.URL != nil && *req.URL != "" {
    h.handleURLBasedSubmission(w, r, userID, &req)   // NEW FLOW
} else if req.ContentID != nil {
    h.handleContentIDBasedSubmission(w, r, userID, &req)
}
```
So a request carrying **both** fields:
```json
{"url": "not-a-url", "content_id": "<uuid>"}
```
skips `is.URL` entirely and is then handed to the URL flow. The validator's guard and the handler's dispatch disagree about which field wins.

## What to do
1. **Failing test first:** POST with a malformed `url` **and** a valid `content_id`; assert 400. Fails on main (the request proceeds down the URL path).
2. Make validation agree with dispatch: validate `URL`'s format **whenever it is non-empty**, independently of `ContentID`; keep the "at least one of" requirement as its own rule.
3. Don't change the handler's precedence — the fix is to validate what the handler actually uses.

## Done when
- A malformed URL is rejected on every path that would consume it, proven by a test that fails on main.

## Relationship to the SSRF gap (task_fe72)
**Distinct finding, different file, different fix — neither closes the other.** They compose: this one lets an unvalidated URL through, task_fe72 lets the fetch of it go unguarded. Fixing validation does not make the fetcher safe, and guarding the fetcher does not make the input valid. Do both.
