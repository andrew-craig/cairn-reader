---
id: task_39dd
title: [Email sanitizer] Replace email's hand-maintained bluemonday policy with pkg/rss/sanitize
type: task
status: open
priority: 2
labels: [quality,wave4,consolidation]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-09T06:53:56Z
updated_at: 2026-08-14T10:17:09Z
---
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. Read the full finding text in docs/CODE_QUALITY_REVIEW.md. One finding, one branch, one PR. Re-verify on main first — cited line numbers are from 2026-07-05 and drift.

**Finding:** H7 | **Wave 4** | **Recipe:** R11 (strategy §2.5)
**Touches:** services/read/email/internal/processor/content_extractor.go:26-55
**Blocked by:** the Wave 1 MaxBytesReader/depth-guard task — it edits the same file; land that first.

## Problem
The email service builds its **own hand-maintained bluemonday policy** (`content_extractor.go:26-55`) instead of the canonical `pkg/rss/sanitize` that read/content correctly delegates to. Future sanitizer hardening will not reach email content — and email is an untrusted inbound path.

## What to do
1. **Characterize first:** diff the two policies. Anything the email policy allows or strips that `pkg/rss/sanitize` does not is a behavior change — enumerate it, decide **explicitly** whether the canonical policy should absorb it, and say so in the PR.
2. Write tests on `pkg/rss/sanitize` covering the email-specific cases you are keeping.
3. Repoint email to `pkg/rss/sanitize` and delete the local policy in the same PR.

## Done when
- Email content is sanitized by `pkg/rss/sanitize`; no second policy remains.

