package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

// RunMigrations runs all pending database migrations
func RunMigrations(migrationsPath string) error {
	// Create a standard database/sql connection for migrations
	connString := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		getEnv("DB_HOST", "localhost"),
		getEnvInt("DB_PORT", 5432),
		getEnv("DB_USER", "cairn"),
		getEnv("DB_PASSWORD", "cairn_password"),
		getEnv("DB_NAME", "cairn_db"),
		getEnv("DB_SSLMODE", "disable"),
	)

	db, err := sql.Open("postgres", connString)
	if err != nil {
		return fmt.Errorf("failed to open database connection for migrations: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("failed to close database connection", slog.Any("error", err))
		}
	}()

	// Create postgres driver instance
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

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

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an environment variable as an integer or returns a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
