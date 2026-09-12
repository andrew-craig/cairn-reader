---
id: task_f19d
title: Mobile: treat a 5xx auth response as unreachable, not as a rejected credential
type: task
status: open
priority: 2
labels: []
blocked_by: []
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-12T03:54:54Z
updated_at: 2026-09-12T03:54:54Z
---


## Origin
Second half of the magpie-reviewer finding on PR #389 (task_5bd6). The first half —
`parseJsonResponse` throwing a plain `Error` carrying `NetworkError`'s message — was
folded into that PR. This half was deliberately left out because it changes auth error
semantics across four call sites rather than one line.

## The problem
`AuthService.loginWithDevice`, `registerWithDevice`, `loginWithEmail` and
`registerWithEmail` all do:

    const result = await this.parseJsonResponse(response);
    if (!response.ok) {
      throw new Error(result.message || result.error || '<verb> failed');
    }

A 5xx is therefore indistinguishable, by type, from a 4xx credential rejection. Two
consequences:
1. `LoginScreen.handleGetStarted` treats a 5xx device login as "login was rejected, so
   register instead" and makes a second doomed round trip — the exact symptom task_5bd6
   fixed for `NetworkError` and for unparseable bodies, still reachable via 5xx.
2. `NetworkError`'s own doc comment in `utils/errors.ts` already claims it covers "a 5xx
   response", so the code and the documented contract disagree today.

Same bug class as task_cab7, task_c87c, chore_1089 and the PR #389 fix — see the
2026-09-06 and 2026-09-07 entries in `LEARNINGS.md`.

## Scope to decide before implementing
- Whether 5xx becomes `NetworkError` at `parseJsonResponse`/entry-point level, or whether
  these four call sites should throw the `HttpError` (status-carrying) type that
  task_ebf1 added in `utils/errors.ts`, letting callers branch on the status. The second
  is probably the better long-term shape and would let `NetworkError`'s doc comment be
  corrected rather than satisfied.
- Check every caller that branches on these errors before changing the type —
  `utils/retry.ts` classifies by message substring, and `AccountScreen` surfaces
  `error.message` directly.
- Do not change 4xx/credential semantics.

## Verify
A 5xx device login makes exactly one network attempt and does not fall back to register;
a 4xx still falls back. Existing suite stays green; no new lint errors.
