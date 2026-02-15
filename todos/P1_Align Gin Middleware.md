# Align on Gin Middleware

**Priority:** P3
**Status:** pending
**Task ID:** 13

## Problem

Framework variations across services: User Service uses Gin with `pkg/auth.GinMiddleware`, while Read/Explore services use net/http with `pkg/auth.Middleware`. Both use the same underlying validator but have different middleware patterns.

## Impact

Code duplication and inconsistency across services. Developers need to understand two different middleware patterns. Harder to maintain and extend authentication across services.

## Current Implementation

- User Service: Uses Gin framework with Gin-specific middleware wrappers
- Read/Explore Services: Use stdlib net/http with custom middleware

Both ultimately use the same `pkg/auth.Validator` for token validation.

## Current Status

This inconsistency is documented as acceptable since both use the same underlying validator. However, it could be aligned further if maintainability becomes an issue.

## Proposed Solution

Options:
1. **Accept Current State** - Keep Gin for complex services (User) and stdlib for simple services (Explore)
   - Trade-off: Some duplication but clear separation by complexity

2. **Unified Middleware Wrapper** - Create a generic middleware factory that works with both
   - Benefits: Single implementation to maintain
   - Trade-off: Additional abstraction layer

3. **Migrate Explore to Gin** - Use Gin across all services
   - Benefits: Complete consistency
   - Trade-off: Additional dependencies in simple services

See [docs/architecture/http-frameworks.md](/docs/architecture/http-frameworks.md) for full framework decision documentation.

## Files to Modify

- `pkg/auth/middleware.go`
- `pkg/auth/gin_adapter.go`
- Service-specific middleware implementations

## Testing

- Verify both Gin and stdlib middleware validate tokens correctly
- Test with valid, invalid, and expired tokens
- Test optional auth behavior with both frameworks

## Related Tasks

- Task 14: Review OptionalAuth behaviour
- Task 22: Add HTTP Framework Decision Documentation
