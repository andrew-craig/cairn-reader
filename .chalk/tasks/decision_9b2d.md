---
id: decision_9b2d
title: Web: Record architectural decisions (shared code, token storage)
type: decision
status: open
priority: 2
labels: []
blocked_by: []
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-08T11:32:27Z
updated_at: 2026-06-08T11:32:27Z
---
Two decisions gate the web app scaffold and must be made before significant coding begins.

DECISION 1 — Shared code strategy:
A) Extract apps/shared package (types/, service logic, theme tokens) imported by both apps/mobile and apps/web.
B) Copy-and-adapt: duplicate into apps/web and converge later.
Factors: build tooling complexity, mobile disruption risk, maintenance overhead.

DECISION 2 — Token storage:
A) localStorage (reuse path; XSS-exposed — mitigated by mandatory DOMPurify sanitization).
B) httpOnly cookie auth model (requires backend changes to set/clear cookies).
The requirements doc notes localStorage as the pragmatic default given existing bearer-token APIs.

OUTPUT: A short ADR file at docs/decisions/web-app-adr.md recording both choices and their rationale.

VERIFICATION: docs/decisions/web-app-adr.md exists and documents both decisions with rationale.
