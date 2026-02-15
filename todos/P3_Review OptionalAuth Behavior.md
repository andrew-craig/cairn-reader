# Review OptionalAuth Behaviour

**Priority:** P3
**Status:** pending
**Task ID:** 14

## Problem

OptionalAuth behavior: When invalid token provided, OptionalAuth continues without authentication. This is documented but could allow requests that appear authenticated to proceed unauthenticated.

## Impact

Security concern where clients might believe they're authenticated when they're actually not. Could lead to:
- Users accessing resources they shouldn't have access to
- Confusion in audit logs about who accessed what
- Logic errors in handlers that assume optional auth provided valid tokens

## Current Implementation

OptionalAuth middleware in `pkg/auth/middleware.go` ignores invalid tokens and proceeds without setting user ID in context.

## Current Behavior

- Valid token → extracts user ID and sets in context
- No token → proceeds without user ID in context
- Invalid token → **proceeds without user ID (silently ignores error)**

## Proposed Solution

Review and document the OptionalAuth behavior more explicitly:

1. **Audit all endpoints using OptionalAuth** - identify which endpoints actually use it
2. **Update middleware to log invalid tokens** - for security auditing
3. **Clarify intent in code** - add comments explaining when/why OptionalAuth is appropriate
4. **Consider alternative names** - if semantics unclear, rename to better reflect behavior
5. **Document in API specs** - clearly mark which endpoints have optional vs required auth

Add validation in handlers to check if user ID exists when it's required for the operation.

## Files to Modify

- `pkg/auth/middleware.go`
- Service handlers using OptionalAuth
- API documentation

## Testing

- Verify valid tokens extract user ID correctly
- Verify missing tokens proceed without user ID
- Verify invalid tokens proceed without user ID
- Audit that handlers validate user ID when required
- Verify logging captures invalid token attempts

## Related Tasks

- Task 13: Align on Gin Middleware
- Task 7: Replace Panic-Based User ID Extraction with Error Handling
