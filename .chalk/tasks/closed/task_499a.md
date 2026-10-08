---
id: task_499a
title: [Audit F-S11-1 + F-S08-1/Tier 3] Type the outbox payloads in both services — producer/consumer drift, sequence together (X2)
type: task
status: closed
priority: 2
labels: [quality,consolidation,audit]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T12:50:50Z
updated_at: 2026-10-08T21:34:50Z
---
**Source:** Cairn Simplification Audit — https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f · file:line detail supplied by the audit author 2026-08-17 and re-verified at HEAD `a6c56a1`.
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. One finding, one branch, one PR.

**Audit findings:** F-S11-1 (fetcher) **+** F-S08-1 (email) | **Audit tier:** 3 | **Audit cross-pattern:** X2.

## Problem
**These are two findings in two services with two separate types over two separate DB rows** — but they must be sequenced together or they diverge further.

| Finding | Service | Location |
|---|---|---|
| F-S11-1 | read/fetcher | `services/read/fetcher/internal/models/models.go:121` + `internal/models/outbox_payload.go` |
| F-S08-1 | read/email | `services/read/email/internal/models/models.go:84` |

**The drift is producer↔consumer, not persistence.** The repositories only `json.Marshal`/`json.Unmarshal` the payload generically and **need no logic change** — do not go looking for the bug in `repository/outbox.go`. The problem is that the producing and consuming ends agree only by convention, so a field added on one side is silently absent on the other.

## What to do
1. Characterize what each side actually writes and reads today, per service. A field present in the struct but never populated is part of the finding.
2. Give each payload a single owning type shared by producer and consumer, so the compiler enforces the contract.
3. **Do both services in one pass** (X2) — fixing one alone leaves the other free to drift, and the two payloads are similar enough that a later reader will assume they match.

## Done when
- Producer and consumer of each outbox payload share one type per service, and a field can no longer be added on one side only.

## Note on scope
The repositories are correct as-is. If your diff touches `json.Marshal` in either `repository/outbox.go`, you have probably wandered out of the finding.

## Review (2026-10-08)
- [x] Re-verified both findings at HEAD bc6b823: `ContentPayload map[string]interface{}` in fetcher and email; producer and consumer agreed only by convention.
- [x] Characterized writes vs reads. Fetcher wrote `raw_description` that was never read. Email wrote `published_at` that was never read. Neither client struct has a field for either, so both were dropped.
- [x] Added one payload type per service, shared by producer and consumer: `models.FeedItemPayload` (fetcher) and `models.EmailContentPayload` (email). JSON tags match the existing keys, so rows already stored still decode. Deleted fetcher `outbox_payload.go`.
- [x] `repository/outbox.go` untouched in both services.
- [x] Tests: producer → JSON → consumer round-trip, plus decoding of legacy stored rows. Before the fix these failed to compile against the new types; the old code had no runtime failure.
- [x] gofmt / vet (incl. integration tag) / golangci-lint / go test -race green in services/read and services/read/email. Integration tests compile-checked only (no Postgres).
- Behaviour change: a hand-edited fetcher row with a malformed `source_feed_id`/`published_at` now fails to decode, where it used to be silently skipped.
- Follow-up candidate (not done): email content is delivered without a published date. The email bulk-create client has no `PublishedAt` field.
- No §1.3 ledger row exists for F-S11-1/F-S08-1.
