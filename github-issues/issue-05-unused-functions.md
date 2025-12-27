# [MEDIUM] Remove unused functions in User Service

Labels: cleanup, priority:medium, service:users, go

## Problem

The User Service has **4 unused functions** and **1 unused variable** that contribute to code clutter and can confuse developers.

## Impact

- **Dead Code**: Unused functions increase codebase size without adding value
- **Maintenance Burden**: Developers may waste time reading/updating dead code
- **Compilation Error**: The unused variable in example code prevents compilation
- **Confusion**: Other developers may assume these functions are used somewhere

## Affected Files

### 1. Unused Variable (BLOCKS COMPILATION)

**File:** `pkg/auth/examples/explore-service/main.go:90`

```go
pathUserID := r.PathValue("id")  // Declared but not used
```

**Impact:** **HIGH** - This prevents the example from compiling
**Fix:** Remove the line or use the variable in the handler

### 2. Unused Test Cleanup Functions

**File:** `internal/database/user_repository_test.go`

```go
// Line 56
func cleanupTestUserByEmail(t *testing.T, db *DB, email string) {
    _, err := db.Pool.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
    if err != nil {
        t.Logf("Warning: failed to cleanup test user by email: %v", err)
    }
}

// Line 64
func cleanupTestUserByDeviceID(t *testing.T, db *DB, deviceID string) {
    _, err := db.Pool.Exec(context.Background(), "DELETE FROM users WHERE expo_device_id = $1", deviceID)
    if err != nil {
        t.Logf("Warning: failed to cleanup test user by device ID: %v", err)
    }
}
```

**Impact:** Low - These test utilities are not currently used
**Fix:** Either remove them or add them to test cleanup where appropriate

### 3. Unused Middleware Helper

**File:** `internal/middleware/auth.go:161`

```go
func extractTokenFromHeader(r *http.Request) string {
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" {
        return ""
    }

    parts := strings.Split(authHeader, " ")
    if len(parts) != 2 || parts[0] != "Bearer" {
        return ""
    }

    return parts[1]
}
```

**Impact:** Low - Likely replaced by different authentication approach
**Fix:** Remove if truly unused, or document why it's kept for future use

## Recommended Actions

### For the compilation error (Priority 1):
```go
// Option 1: Remove if not needed
// Delete line 90: pathUserID := r.PathValue("id")

// Option 2: Use the variable
pathUserID := r.PathValue("id")
log.Printf("Processing request for user ID: %s", pathUserID)
// ... use pathUserID in the handler logic
```

### For test cleanup functions (Priority 2):
```go
// Option 1: Remove unused functions
// Delete lines 56-69

// Option 2: Add to test cleanup (if useful)
func TestCreateUser(t *testing.T) {
    // ... test code ...

    // Add cleanup
    t.Cleanup(func() {
        cleanupTestUserByEmail(t, db, "test@example.com")
    })
}
```

### For middleware helper (Priority 3):
```go
// If truly unused, remove the entire function
// Delete lines 161-174 (adjust line numbers as needed)
```

## Testing

After fixes, verify with:
```bash
cd services/users

# Check compilation
go build ./...

# Check for unused code
staticcheck ./...

# Run tests
make test

# Verify linting
golangci-lint run ./...
```

## References

- Code Review Report: `CODE_REVIEW_REPORT.md` (Section: "Unused Functions")
- Tool: `staticcheck` (U1000: unused code detection)
- Estimated Effort: **30 minutes**

## Acceptance Criteria

- [ ] Unused variable in `main.go:90` is removed or used
- [ ] Example code compiles successfully: `go build ./pkg/auth/examples/...`
- [ ] Decision made on test cleanup functions (remove or use)
- [ ] `extractTokenFromHeader` function is removed or documented
- [ ] All tests pass: `make test`
- [ ] No unused code warnings: `staticcheck ./...`
- [ ] Code compiles without errors: `go build ./...`
