# No Vault Response Caching

**Priority:** P3
**Status:** pending
**Task ID:** 11

## Problem

Keys are fetched synchronously on startup from Vault. There is no caching strategy with TTL (Time To Live).

## Impact

Every key retrieval makes a request to Vault. If Vault has high latency or many keys need to be validated, performance could be impacted. Startup performance depends on Vault availability.

## Current Implementation

Current implementation fetches keys from Vault on startup but doesn't cache responses with TTL.

## Proposed Solution

Implement caching with TTL for Vault responses:

1. Cache key responses in-memory with configurable TTL
2. Implement cache invalidation/refresh before TTL expires
3. Add metrics for cache hits/misses
4. Handle cache errors gracefully (fall back to Vault on cache miss)

This reduces Vault load and improves startup performance.

## Files to Modify

- `services/users/internal/auth/vault.go`
- `pkg/auth/validator.go`

## Testing

- Verify keys are cached after first fetch
- Verify cache expires after TTL
- Verify cache invalidation works
- Verify fallback to Vault on cache miss

## Related Tasks

- Task 12: Vault Connection Not Retried
