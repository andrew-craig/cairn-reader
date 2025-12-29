# Read Service Engineering Principles Review

**Date:** 2025-12-29
**Reviewer:** Claude
**Scope:** Content Service and Fetcher Service implementation vs. Engineering Principles

## Executive Summary

This review identifies mismatches between the Read Service implementation and the established Engineering Principles documented in `/docs/ENGINEERING_PRINCIPLES.md`. The Read Service is largely well-architected and follows most core principles, but there are several areas where implementation choices diverge from documented standards.

**Overall Assessment:** 🟡 Mostly Compliant with Notable Deviations

---

## ✅ What's Working Well

The Read Service correctly implements these key principles:

1. **✅ Layered Architecture (Handler → Service → Repository)**
   - Clear separation of concerns across all services
   - Handlers in `internal/api/handlers/`
   - Services in `internal/service/`
   - Repositories in `internal/repository/`

2. **✅ Dependency Injection**
   - All dependencies injected via constructors
   - No global variables or singletons
   - See: `services/read/content/internal/api/router.go:24-34`

3. **✅ Context-Driven Cancellation**
   - All repository methods accept `context.Context` as first parameter
   - All service methods accept `context.Context`
   - Database operations properly use context

4. **✅ Error Handling Pattern**
   - Repositories wrap errors with context using `fmt.Errorf("...: %w", err)`
   - Services define domain logic
   - Handlers convert errors to HTTP responses
   - See: `services/read/content/internal/repository/content.go:99`

5. **✅ Database-Per-Service Architecture**
   - Content Service has its own database (`cairn_content`)
   - Fetcher Service has its own database (`cairn_rss`)
   - Services communicate via REST APIs only

6. **✅ Graceful Shutdown**
   - Both services implement proper graceful shutdown
   - 30-second timeout for outstanding requests
   - See: `services/read/content/cmd/content/main.go:71-86`

---

## ❌ Mismatches & Deviations

### 1. **Logging Library Inconsistency** 🔴 HIGH PRIORITY

**Engineering Principle:**
> Use `log/slog` (stdlib) for structured logging with zero configuration

**Current Implementation:**
- Main services use `go.uber.org/zap` for structured logging
- Middleware uses standard `log` package (unstructured)

**Evidence:**
- `services/read/content/cmd/content/main.go:22` - Uses `go.uber.org/zap`
- `services/read/content/internal/api/middleware/logging.go:47` - Uses `log.Printf`
- `services/read/go.mod:18` - `go.uber.org/zap v1.27.1`

**Impact:**
- Inconsistent logging format across the application
- Middleware logs are unstructured (hard to parse/query)
- Deviates from documented standard library preference

**Recommendation:**
Replace zap with `log/slog` throughout the Read Service to match User Service and Explore Service patterns. Update middleware to use structured logging:

```go
// Instead of:
log.Printf("[%s] %s %s - Status: %d - Duration: %v", ...)

// Use:
slog.Info("request completed",
    slog.String("method", r.Method),
    slog.String("path", r.URL.Path),
    slog.Int("status", wrapped.statusCode),
    slog.Duration("duration", duration),
)
```

---

### 2. **HTTP Framework Not Documented** 🟡 MEDIUM PRIORITY

**Engineering Principle:**
> - `github.com/gin-gonic/gin` for HTTP framework (User Service)
> - `net/http` (stdlib) for simple services (Explore Service)

**Current Implementation:**
- Read Service uses `github.com/go-chi/chi/v5` router

**Evidence:**
- `services/read/go.mod:9` - `github.com/go-chi/chi/v5 v5.2.3`
- `services/read/content/internal/api/router.go:12` - Uses chi router

**Impact:**
- Technology stack documentation is incomplete
- New developers won't know chi is an approved framework
- Inconsistency in framework choices across services

**Recommendation:**
Update `docs/ENGINEERING_PRINCIPLES.md` Technology Stack section to document chi/v5 as an approved lightweight HTTP router:

```markdown
| Library | Version | Purpose | Rationale |
|---------|---------|---------|-----------|
| `github.com/go-chi/chi/v5` | v5.2.3 | HTTP framework (Read Service) | Lightweight, idiomatic Go, good middleware support |
```

**Note:** Chi is a reasonable choice for the Read Service. The issue is documentation, not the technical decision.

---

### 3. **Test Utilities Directory Naming** 🟡 MEDIUM PRIORITY

**Engineering Principle:**
> Create `internal/testutil/helpers.go` for shared test helpers

**Current Implementation:**
- Read Service uses `internal/testhelpers/` directory

**Evidence:**
- `services/read/content/internal/testhelpers/database.go`
- `services/read/fetcher/internal/testhelpers/database.go`

**Impact:**
- Inconsistent directory naming across services
- Makes it harder to find test utilities (developers look for `testutil`)
- Violates principle of consistency

**Recommendation:**
Rename `internal/testhelpers/` to `internal/testutil/` to match the documented pattern:

```bash
mv services/read/content/internal/testhelpers services/read/content/internal/testutil
mv services/read/fetcher/internal/testhelpers services/read/fetcher/internal/testutil
```

Update all import paths accordingly.

---

### 4. **Configuration Management Pattern** 🟢 LOW PRIORITY

**Engineering Principle:**
> Group related config into structs (User Service pattern) in `internal/config/config.go`

**Current Implementation:**
- Configuration implemented directly in `main.go`
- No centralized config package
- No validation method

**Evidence:**
- `services/read/content/cmd/content/main.go:89-108` - Config in main.go
- `services/read/fetcher/cmd/fetcher/main.go:89-108` - Config in main.go

**Impact:**
- Less organized than documented pattern
- No centralized config validation
- Harder to test configuration logic

**Recommendation:**
Create `internal/config/config.go` following User Service pattern:

```go
package config

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
}

type ServerConfig struct {
    Port string
}

type DatabaseConfig struct {
    Host     string
    Port     int
    User     string
    Password string
    DBName   string
    SSLMode  string
}

func Load() (*Config, error) {
    cfg := &Config{
        Server: ServerConfig{
            Port: getEnv("PORT", "8080"),
        },
        Database: DatabaseConfig{
            Host: getEnv("DB_HOST", "localhost"),
            // ...
        },
    }

    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }

    return cfg, nil
}

func (c *Config) Validate() error {
    // Validation logic
    return nil
}
```

**Note:** This is a lower priority improvement since the current approach works functionally.

---

### 5. **Documentation Gap: Additional Libraries** 🟡 MEDIUM PRIORITY

**Engineering Principle:**
> Document all core libraries in Technology Stack section

**Current Implementation:**
- Read Service uses several libraries not documented in Engineering Principles

**Evidence (from `services/read/go.mod`):**
- `github.com/sony/gobreaker v0.5.0` - Circuit breaker (not documented)
- `github.com/robfig/cron/v3 v3.0.1` - Job scheduling (not documented)
- `github.com/microcosm-cc/bluemonday v1.0.27` - HTML sanitization (documented in requirements, not in principles)
- `github.com/go-shiori/go-readability v0.0.0-20251205110129-5db1dc9836f0` - Content extraction (documented in requirements, not in principles)

**Impact:**
- Incomplete technology stack documentation
- New developers won't know these are approved libraries
- No documented rationale for library choices

**Recommendation:**
Update `docs/ENGINEERING_PRINCIPLES.md` Technology Stack section to include Read Service libraries:

```markdown
### Read Service Specific Libraries

| Library | Version | Purpose | Rationale |
|---------|---------|---------|-----------|
| `github.com/go-shiori/go-readability` | latest | Content extraction | Extract readable content from HTML |
| `github.com/microcosm-cc/bluemonday` | v1.0.27 | HTML sanitization | Security - XSS prevention |
| `github.com/sony/gobreaker` | v0.5.0 | Circuit breaker | Resilience for external API calls |
| `github.com/robfig/cron/v3` | v3.0.1 | Job scheduling | Background jobs (cleanup, polling) |
| `github.com/go-chi/chi/v5` | v5.2.3 | HTTP router | Lightweight, idiomatic, good middleware |
```

---

## 📊 Compliance Summary

| Principle | Status | Priority |
|-----------|--------|----------|
| Layered Architecture | ✅ Compliant | - |
| Dependency Injection | ✅ Compliant | - |
| Context Cancellation | ✅ Compliant | - |
| Error Handling | ✅ Compliant | - |
| Database-Per-Service | ✅ Compliant | - |
| Graceful Shutdown | ✅ Compliant | - |
| Logging (slog) | ❌ Non-compliant | 🔴 High |
| HTTP Framework Documentation | ⚠️ Undocumented | 🟡 Medium |
| Test Directory Naming | ⚠️ Deviation | 🟡 Medium |
| Config Management Pattern | ⚠️ Simplified | 🟢 Low |
| Library Documentation | ⚠️ Incomplete | 🟡 Medium |

**Legend:**
- ✅ Compliant - Follows principles exactly
- ⚠️ Deviation - Works but doesn't match documented pattern
- ❌ Non-compliant - Contradicts documented standard

---

## 🎯 Recommended Actions

### Immediate (Before Next Code Review)

1. **Replace zap with log/slog** 🔴
   - Update main.go to use slog instead of zap
   - Update middleware to use slog for structured logging
   - Remove zap dependency from go.mod

2. **Update Engineering Principles Documentation** 🟡
   - Add chi/v5 to approved HTTP frameworks
   - Document Read Service specific libraries
   - Add rationale for each library choice

### Short-term (Next Sprint)

3. **Rename test directory** 🟡
   - Rename `internal/testhelpers/` to `internal/testutil/`
   - Update all import paths
   - Ensure consistency with other services

### Long-term (Future Refactor)

4. **Centralize configuration management** 🟢
   - Create `internal/config/config.go`
   - Add validation logic
   - Follow User Service pattern for consistency

---

## 📝 Conclusion

The Read Service demonstrates strong adherence to core architectural principles, particularly in layered architecture, dependency injection, and error handling patterns. The main areas of concern are:

1. **Logging inconsistency** - This should be addressed promptly to ensure uniform logging across all services
2. **Documentation gaps** - The Engineering Principles document needs updates to reflect actual technology choices

The deviations found are primarily documentation issues rather than fundamental architectural problems. The code quality is good, and most deviations represent reasonable implementation choices that simply need to be documented.

**Overall Grade:** B+ (Good implementation with documentation gaps)

---

## 🔍 Cross-References

**Engineering Principles Document:** `/docs/ENGINEERING_PRINCIPLES.md`

**Key Sections Referenced:**
- Technology Stack (lines 407-498)
- Development Standards - Code Organization (lines 522-611)
- Testing Philosophy (lines 912-1247)
- Common Patterns (lines 1441-1755)

**Related Files:**
- Read Service Requirements: `/services/read/requirements.md`
- Content Service: `/services/read/content/`
- Fetcher Service: `/services/read/fetcher/`
