package db

import (
	"database/sql"
	"fmt"

	"github.com/cairn-app/cairn-reader/pkg/env"
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
		env.GetString("DB_HOST", "localhost"),
		env.GetInt("DB_PORT", 5432),
		env.GetString("DB_USER", "explore_fetcher"),
		env.GetString("DB_PASSWORD", "password"),
		env.GetString("DB_NAME", "explore_fetcher"),
		env.GetString("DB_SSLMODE", "disable"),
	)

	db, err := sql.Open("postgres", connString)
	if err != nil {
		return fmt.Errorf("failed to open database connection for migrations: %w", err)
	}
	defer func() { _ = db.Close() }()

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
