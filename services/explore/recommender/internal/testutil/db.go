// Package testutil provides testing utilities for the recommender service.
package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
	testcontainerspostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// SetupTestDB creates a PostgreSQL testcontainer, runs migrations, and returns a pgxpool.Pool
// for testing. It returns the pool and a cleanup function that should be called with defer.
//
// Example usage:
//
//	func TestSomething(t *testing.T) {
//	    if testing.Short() {
//	        t.Skip("Skipping integration test")
//	    }
//
//	    pool, cleanup := testutil.SetupTestDB(t)
//	    defer cleanup()
//
//	    repo := db.NewArticleRepository(pool)
//	    // ... test code
//	}
func SetupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	ctx := context.Background()

	// Start PostgreSQL container
	postgresContainer, err := testcontainerspostgres.Run(ctx,
		"postgres:16-alpine",
		testcontainerspostgres.WithDatabase("test_db"),
		testcontainerspostgres.WithUsername("test_user"),
		testcontainerspostgres.WithPassword("test_password"),
		testcontainerspostgres.BasicWaitStrategies(),
		testcontainerspostgres.WithInitScripts(),
	)
	if err != nil {
		t.Fatalf("Failed to start postgres container: %v", err)
	}

	// Get connection string for pgx
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = postgresContainer.Terminate(ctx)
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Wait for PostgreSQL to be ready
	time.Sleep(2 * time.Second)

	// Run migrations
	if err := runMigrations(connStr); err != nil {
		_ = postgresContainer.Terminate(ctx)
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create pgxpool connection
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		_ = postgresContainer.Terminate(ctx)
		t.Fatalf("Failed to parse pool config: %v", err)
	}

	// Configure pool settings
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		_ = postgresContainer.Terminate(ctx)
		t.Fatalf("Failed to create connection pool: %v", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		_ = postgresContainer.Terminate(ctx)
		t.Fatalf("Failed to ping database: %v", err)
	}

	cleanup := func() {
		pool.Close()
		if err := postgresContainer.Terminate(context.Background()); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}

	return pool, cleanup
}

// runMigrations runs all database migrations for testing
func runMigrations(connStr string) error {
	// Create a standard database/sql connection for migrations
	// (golang-migrate library requires database/sql)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection for migrations: %w", err)
	}
	defer db.Close()

	// Wait for database to be ready
	for i := 0; i < 30; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		if i == 29 {
			return fmt.Errorf("database not ready after 30 attempts")
		}
		time.Sleep(1 * time.Second)
	}

	// Create postgres driver instance
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	// Get migrations path (relative to this file)
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("failed to get current file path")
	}
	migrationsPath := filepath.Join(filepath.Dir(filename), "..", "..", "migrations")

	// Create migrate instance
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Run all pending migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
