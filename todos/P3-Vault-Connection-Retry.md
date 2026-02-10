# Vault Connection Not Retried

**Priority:** P3
**Status:** pending
**Task ID:** 12

## Problem

If Vault is temporarily unavailable at startup, the service fails immediately without retry logic.

## Impact

Startup failures if Vault is temporarily down for maintenance or network issues. Services cannot recover automatically, requiring manual restart.

## Current Implementation

Current implementation attempts to connect to Vault on startup once. If the connection fails, the entire service startup fails.

## Proposed Solution

Implement retry with exponential backoff for Vault connection:

1. Add retry configuration (max retries, initial backoff, max backoff)
2. Implement exponential backoff between retries (e.g., 100ms → 200ms → 400ms)
3. Log retry attempts and failures
4. Set maximum retry duration (e.g., 5 minutes)
5. Fail startup only after all retries exhausted

Example configuration:
- Max retries: 10
- Initial backoff: 100ms
- Max backoff: 30 seconds
- Max total duration: 5 minutes

## Files to Modify

- `services/users/internal/auth/vault.go`
- `pkg/auth/validator.go`

## Testing

- Test successful connection after temporary failure
- Verify exponential backoff behavior
- Test final failure after max retries
- Verify appropriate logging at each step

## Related Tasks

- Task 11: No Vault Response Caching
