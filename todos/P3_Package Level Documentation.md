# Add Package-Level Documentation

**Priority:** P3
**Status:** pending
**Task ID:** 17

## Problem

User service has excellent package documentation, while explore services have minimal package comments. This creates inconsistent documentation standards across the codebase.

## Impact

Developers unfamiliar with a service need to read implementation code to understand package purposes. Makes onboarding slower and reduces code maintainability.

## Current Implementation

**Good example** (`services/users/internal/database/db.go`):
```go
// Package database provides PostgreSQL database connectivity and repository
// implementations for the user service. It uses pgx for high-performance
// PostgreSQL operations with connection pooling.
package database
```

**Minimal example** (`services/explore/fetcher/internal/db/config.go`):
```go
package db
```

## Proposed Solution

Add comprehensive package documentation to all packages in Explore services:

For `services/explore/fetcher/internal/db/`:
```go
// Package db provides PostgreSQL database connectivity for the fetcher service.
// It manages feed sources and tracks fetch history using pgx/v5.
//
// The main types are:
//   - Config: Database connection configuration
//   - FeedRepository: CRUD operations for RSS feed sources
//   - HistoryRepository: Tracking of fetch attempts
//
// Example usage:
//
//    cfg := &db.Config{
//        Host: "localhost",
//        Port: "5432",
//        // ... other fields
//    }
//    conn, err := cfg.Connect(ctx)
//    if err != nil {
//        log.Fatal(err)
//    }
//    defer conn.Close()
//
//    repo := db.NewFeedRepository(conn)
//    feeds, err := repo.List(ctx)
package db
```

Apply similar pattern to:
- `services/explore/fetcher/internal/fetcher/`
- `services/explore/fetcher/internal/sync/`
- `services/explore/recommender/internal/db/`
- `services/explore/recommender/internal/recommend/`
- `services/explore/recommender/internal/api/`

## Files to Modify

- `services/explore/fetcher/internal/db/db.go` (or main file in each package)
- `services/explore/fetcher/internal/fetcher/recommend.go`
- `services/explore/fetcher/internal/sync/sync.go`
- `services/explore/recommender/internal/db/db.go`
- `services/explore/recommender/internal/recommend/engine.go`
- `services/explore/recommender/internal/api/api.go`

## Testing

- Run `go doc ./...` to verify documentation renders correctly
- Review generated documentation for clarity and accuracy
- Ensure examples in comments are valid Go code

## Notes

- Follow the pattern established in User Service
- Include main types and their purposes
- Add example usage where helpful
- Keep descriptions concise but informative
