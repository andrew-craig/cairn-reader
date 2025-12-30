package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

const (
	defaultTestDBHost     = "localhost"
	defaultTestDBPort     = "5434"
	defaultTestDBUser     = "cairn_rss"
	defaultTestDBPassword = "cairn_rss_pass"
	defaultTestDBName     = "cairn_rss_test"
)

// TestDatabase holds a test database connection and provides cleanup
type TestDatabase struct {
	DB       *sql.DB
	DBName   string
	connInfo ConnectionInfo
	t        *testing.T
}

// ConnectionInfo holds database connection parameters
type ConnectionInfo struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// GetTestConnectionInfo returns connection info from environment or defaults
func GetTestConnectionInfo() ConnectionInfo {
	return ConnectionInfo{
		Host:     getEnv("TEST_DB_HOST", defaultTestDBHost),
		Port:     getEnv("TEST_DB_PORT", defaultTestDBPort),
		User:     getEnv("TEST_DB_USER", defaultTestDBUser),
		Password: getEnv("TEST_DB_PASSWORD", defaultTestDBPassword),
		DBName:   getEnv("TEST_DB_NAME", defaultTestDBName),
		SSLMode:  getEnv("TEST_DB_SSL_MODE", "disable"),
	}
}

// SetupTestDatabase creates a new test database, runs migrations, and returns a connection
func SetupTestDatabase(t *testing.T) *TestDatabase {
	t.Helper()

	connInfo := GetTestConnectionInfo()

	// Create a unique database name for this test run
	testDBName := fmt.Sprintf("%s_%d", connInfo.DBName, time.Now().UnixNano())

	// Connect to postgres database to create test database
	postgresConnStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s",
		connInfo.Host, connInfo.Port, connInfo.User, connInfo.Password, connInfo.SSLMode,
	)

	adminDB, err := sql.Open("postgres", postgresConnStr)
	require.NoError(t, err, "Failed to connect to postgres database")
	defer adminDB.Close()

	// Create test database
	_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName))
	require.NoError(t, err, "Failed to create test database")

	// Connect to the new test database
	testConnStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		connInfo.Host, connInfo.Port, connInfo.User, connInfo.Password, testDBName, connInfo.SSLMode,
	)

	testDB, err := sql.Open("postgres", testConnStr)
	require.NoError(t, err, "Failed to connect to test database")

	// Wait for database to be ready
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = testDB.PingContext(ctx)
	require.NoError(t, err, "Failed to ping test database")

	// Run migrations using SQL files directly
	runMigrationsFromSQL(t, testDB)

	return &TestDatabase{
		DB:       testDB,
		DBName:   testDBName,
		connInfo: connInfo,
		t:        t,
	}
}

// runMigrationsFromSQL applies database migrations by reading SQL files directly
func runMigrationsFromSQL(t *testing.T, db *sql.DB) {
	t.Helper()

	// Find the migrations directory
	migrationsPath := findMigrationsPath(t)
	if migrationsPath == "" {
		t.Log("Warning: migrations directory not found, assuming schema exists")
		return
	}

	// Read and execute migration files
	files, err := filepath.Glob(filepath.Join(migrationsPath, "*up.sql"))
	if err != nil {
		t.Logf("Warning: failed to read migration files: %v", err)
		return
	}

	ctx := context.Background()
	for _, file := range files {
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			t.Logf("Warning: failed to read migration file %s: %v", file, err)
			continue
		}

		_, err = db.ExecContext(ctx, string(sqlBytes))
		if err != nil {
			t.Logf("Warning: failed to execute migration %s: %v", file, err)
			// Don't fail - schema might already exist
		}
	}
}

// findMigrationsPath searches for the migrations directory
func findMigrationsPath(t *testing.T) string {
	t.Helper()

	possiblePaths := []string{
		"./migrations",
		"../../migrations",
		"../../../migrations",
		"../../../../migrations",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// Cleanup drops the test database
func (td *TestDatabase) Cleanup() {
	td.t.Helper()

	// Close the test database connection
	if td.DB != nil {
		td.DB.Close()
	}

	// Connect to postgres database to drop test database
	postgresConnStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s",
		td.connInfo.Host, td.connInfo.Port, td.connInfo.User, td.connInfo.Password, td.connInfo.SSLMode,
	)

	adminDB, err := sql.Open("postgres", postgresConnStr)
	if err != nil {
		td.t.Logf("Warning: Failed to connect to postgres for cleanup: %v", err)
		return
	}
	defer adminDB.Close()

	// Terminate existing connections
	_, err = adminDB.Exec(fmt.Sprintf(
		"SELECT pg_terminate_backend(pg_stat_activity.pid) FROM pg_stat_activity WHERE pg_stat_activity.datname = '%s' AND pid <> pg_backend_pid()",
		td.DBName,
	))
	if err != nil {
		td.t.Logf("Warning: Failed to terminate connections: %v", err)
	}

	// Drop test database
	_, err = adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", td.DBName))
	if err != nil {
		td.t.Logf("Warning: Failed to drop test database: %v", err)
	}
}

// TruncateAll truncates all tables in the test database (faster than recreating)
func (td *TestDatabase) TruncateAll() {
	td.t.Helper()

	tables := []string{
		"content_outbox",
		"feed_items",
		"feed_subscriptions",
		"feeds",
	}

	for _, table := range tables {
		_, err := td.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		require.NoError(td.t, err, "Failed to truncate table %s", table)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
