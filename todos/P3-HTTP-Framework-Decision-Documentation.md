# Add HTTP Framework Decision Documentation

**Priority:** P3
**Status:** pending
**Task ID:** 22

## Problem

Different HTTP frameworks used across services (Gin vs stdlib) without documented rationale. Developers may question design decisions or consider refactoring.

## Impact

Inconsistent patterns across codebase reduce clarity about architectural decisions. New team members may not understand why different frameworks are used.

## Current Implementation

- User Service: Gin Web Framework
- Explore Fetcher/Recommender: stdlib net/http
- Read Services: stdlib net/http

## Proposed Solution

Create `docs/architecture/http-frameworks.md` documenting framework choices:

```markdown
# HTTP Framework Decisions

## Current State

### User Service: Gin Web Framework
**Location:** `services/users/`

**Rationale:**
- Complex routing requirements (nested route groups, parameter validation)
- Built-in middleware ecosystem (CORS, recovery, logging)
- Better developer experience for auth-heavy service
- Excellent JSON binding and validation
- Performance optimized for REST APIs

**Trade-offs:**
- Additional dependency (~10MB)
- Framework-specific patterns
- Potential lock-in

### Explore Services: stdlib net/http
**Location:** `services/explore/fetcher/`, `services/explore/recommender/`

**Rationale:**
- Simple API surface (few endpoints)
- Minimal dependencies preferred
- Educational value (explicit HTTP handling)
- Lower memory footprint
- No framework lock-in

**Trade-offs:**
- Manual route parameter extraction
- More boilerplate code
- No built-in validation

## Decision

**Status:** Accepted

We maintain different frameworks based on service complexity:
- **Complex services with many endpoints** → Gin
- **Simple services with few endpoints** → stdlib

## Future Considerations

If explore services grow significantly (>10 endpoints), consider:
1. Migrating to chi (stdlib-compatible router)
2. Extracting common patterns to shared middleware
3. Re-evaluating Gin adoption

## Related

- See `pkg/api/` for shared HTTP utilities
- See `CLAUDE.md` for API conventions
```

## Files to Create

- `docs/architecture/http-frameworks.md`

## Testing

- Verify documentation is readable and accurate
- Check links to referenced services
- Validate rationale matches actual implementation

## Related Tasks

- Task 13: Align on Gin Middleware
