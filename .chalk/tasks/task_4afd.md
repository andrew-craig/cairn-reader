---
id: task_4afd
title: Review and harden OptionalAuth middleware behavior
type: task
status: in_progress
priority: 3
labels: []
blocked_by: []
parent: null
created_at: 2026-03-21T04:20:37Z
updated_at: 2026-05-10T22:41:51Z
---
OptionalAuth silently ignores invalid tokens and proceeds unauthenticated. Security concern. Audit endpoints, add logging, document in API specs.

## Audit

Searched the repo for production usage of `Middleware.OptionalAuth` —
no service router uses it. The only references are in the package's own
test file, the in-package example programs under `pkg/auth/examples/`,
the package `README.md`, and the architecture doc
`docs/architecture/http-frameworks.md`. So this is a hardening of a
shared primitive, not a per-endpoint audit.

## Threat model

`OptionalAuth` exists for endpoints where credentials are optional —
authenticated callers get a personalized view, anonymous callers get a
public one. The "optional" part is whether credentials are *presented*,
not whether they are *valid*. Today both cases (no header / bad token)
take the same branch, which means:

- A client with an expired or tampered token receives an anonymous
  response silently, with no signal that their credentials failed.
- Token-substitution / fuzzing attempts are invisible in logs.
- The fail-closed property of `RequireAuth` is lost the moment a route
  is migrated to `OptionalAuth`.

## Plan

- [x] Audit production routers — no current usage
- [x] Distinguish "no header" (continue anonymous) from "header present
      but invalid" (reject with 401) in `OptionalAuth`
- [x] Log invalid-token attempts at `Warn` level with method/path/
      remote_addr/error class — never the token itself
- [x] Update middleware tests for the new rejection paths and keep
      existing missing-token coverage
- [x] Update `pkg/auth/README.md` and `docs/architecture/http-frameworks.md`
      to document the strict semantics
- [x] Update example programs to reflect the new contract
- [x] Run `go test ./pkg/auth/...`

## Review

Behavior is now fail-closed when an `Authorization` header is supplied
but the token is unusable, and stays permissive when no header is
present. No production endpoint changes were needed because no service
currently uses `OptionalAuth`; future adopters get the hardened
semantics by default.
