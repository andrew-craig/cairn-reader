# No JTI (JWT ID) for Token Revocation

**Priority:** P3
**Status:** pending
**Task ID:** 9

## Problem

JWTs don't include unique identifiers, making it impossible to revoke individual access tokens if compromised.

## Impact

Cannot revoke individual access tokens if compromised. In case of token theft or security breach, there's no way to invalidate just that token without invalidating all tokens for that user.

## Current Implementation

Current JWT implementation does not include a `jti` (JWT ID) claim.

## Proposed Solution

Add jti claim for token blacklisting capability if needed.

Recommendation: Implement JTI support with a token blacklist/revocation list:
1. Generate unique `jti` claim when creating tokens
2. Store revoked `jti` values in cache (Redis) or database
3. Check revocation list during token validation
4. Set appropriate TTL on blacklist entries to match token expiry

## Files to Modify

- `services/users/internal/auth/jwt.go`
- `pkg/auth/validator.go`

## Testing

- Verify JTI is included in generated tokens
- Test token revocation by adding JTI to blacklist
- Verify revoked tokens are rejected during validation
- Verify non-revoked tokens still work

## Related Tasks

- Task 3: Thread-Safe Key Updates in JWTManager
- Task 4: Thread-Safe Key Updates in Validator
