# [MEDIUM] Fix context key type safety in User Service

Labels: bug, priority:medium, service:users, go

## Problem

The User Service middleware tests use built-in `string` type as context keys, which can cause collisions if different packages use the same string key.

## Impact

- **Potential Collisions**: Different packages using the same string key will overwrite each other's context values
- **Subtle Bugs**: Context value collisions are hard to debug and may only occur in specific scenarios
- **Best Practice Violation**: Go documentation recommends using custom types for context keys

## Affected Files

**File:** `pkg/auth/middleware_test.go`

### Instance 1 (Line 345)
```go
ctx := context.WithValue(r.Context(), "userID", userID)  // BAD
```

### Instance 2 (Line 352)
```go
ctx := context.WithValue(r.Context(), "userID", userID)  // BAD
```

## Why This Is a Problem

From the Go documentation:

> The provided key must be comparable and should not be of type string or any other built-in type to avoid collisions between packages using context.

If two different packages use `"userID"` as a context key, they will conflict:

```go
// Package A
ctx = context.WithValue(ctx, "userID", "user-123")

// Package B (different package, same context)
ctx = context.WithValue(ctx, "userID", "admin-456")  // Overwrites Package A's value!
```

## Recommended Fix

Define a custom unexported type for context keys:

### Step 1: Create custom context key type

**File:** `pkg/auth/middleware.go` (or create `pkg/auth/context.go`)

```go
// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// Context key constants
const (
    userIDKey contextKey = "userID"
)

// SetUserIDInContext adds the user ID to the request context
func SetUserIDInContext(ctx context.Context, userID string) context.Context {
    return context.WithValue(ctx, userIDKey, userID)
}

// GetUserIDFromContext retrieves the user ID from the request context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
    userID, ok := ctx.Value(userIDKey).(string)
    return userID, ok
}
```

### Step 2: Update middleware code

**File:** `pkg/auth/middleware.go`

```go
// Before
ctx := context.WithValue(r.Context(), "userID", userID)

// After
ctx := SetUserIDInContext(r.Context(), userID)
```

### Step 3: Update middleware tests

**File:** `pkg/auth/middleware_test.go`

```go
// Before (Lines 345, 352)
ctx := context.WithValue(r.Context(), "userID", userID)

// After
ctx := SetUserIDInContext(r.Context(), userID)
```

### Step 4: Update code that reads from context

```go
// Before
userID := r.Context().Value("userID").(string)

// After
userID, ok := GetUserIDFromContext(r.Context())
if !ok {
    // handle missing user ID
}
```

## Benefits

1. **Type Safety**: Custom type prevents collisions between packages
2. **Better API**: Helper functions provide cleaner, more discoverable API
3. **Error Handling**: Can handle missing values gracefully with boolean return
4. **Documentation**: Helper functions can be documented with godoc comments
5. **Consistency**: Single source of truth for context key strings

## Testing

After fixes, verify with:
```bash
cd services/users

# Check for context key warnings
staticcheck ./...

# Run all tests
make test

# Verify linting
golangci-lint run ./...
```

## References

- Code Review Report: `CODE_REVIEW_REPORT.md` (Section: "Context Key Type Safety")
- Tool: `staticcheck` (SA1029: should not use built-in type as key for value)
- Go Documentation: [context.WithValue](https://pkg.go.dev/context#WithValue)
- Go Blog: [Context and structs](https://go.dev/blog/context-and-structs)
- Estimated Effort: **30 minutes**

## Acceptance Criteria

- [ ] Custom `contextKey` type is defined
- [ ] Helper functions `SetUserIDInContext` and `GetUserIDFromContext` are created
- [ ] All middleware code uses helper functions instead of string keys
- [ ] All test code uses helper functions
- [ ] All code that reads from context uses the getter helper
- [ ] No staticcheck warnings (SA1029): `staticcheck ./...`
- [ ] All tests pass: `make test`

## Example Implementation

See this complete example:

```go
// pkg/auth/context.go
package auth

import "context"

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
    userIDKey contextKey = "userID"
)

// SetUserIDInContext adds the user ID to the request context.
// Returns a new context with the user ID value.
func SetUserIDInContext(ctx context.Context, userID string) context.Context {
    return context.WithValue(ctx, userIDKey, userID)
}

// GetUserIDFromContext retrieves the user ID from the request context.
// Returns the user ID and true if found, or empty string and false if not found.
func GetUserIDFromContext(ctx context.Context) (string, bool) {
    userID, ok := ctx.Value(userIDKey).(string)
    return userID, ok
}
```
