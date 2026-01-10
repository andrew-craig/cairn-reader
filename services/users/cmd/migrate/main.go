package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/cairn-app/cairn-reader/pkg/logging"
	"github.com/cairn-app/cairn-reader/services/users/internal/config"
	"github.com/cairn-app/cairn-reader/services/users/internal/database"
	"github.com/joho/godotenv"
)

func main() {
	// Initialize logger for CLI tool (use text handler for readability)
	logger := logging.NewLogger(logging.Config{
		Level:       "info",
		Format:      "text",
		ServiceName: "migrate",
	})
	slog.SetDefault(logger)

	// Define command flags
	var (
		command        string
		migrationsPath string
		envFile        string
	)

	flag.StringVar(&command, "command", "up", "Migration command: up, down, version")
	flag.StringVar(&migrationsPath, "path", "./migrations", "Path to migrations directory")
	flag.StringVar(&envFile, "env", ".env", "Path to .env file")
	flag.Parse()

	// Load environment variables
	if err := godotenv.Load(envFile); err != nil {
		slog.Warn("could not load env file",
			slog.String("file", envFile),
			slog.Any("error", err),
		)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", slog.Any("error", err))
		os.Exit(1)
	}

	// Convert relative path to absolute path
	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		slog.Error("failed to resolve migrations path",
			slog.String("path", migrationsPath),
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	// Create database config
	dbCfg := &database.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		Database:        cfg.Database.Database,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpenConns:    int32(cfg.Database.MaxOpenConns),
		MaxIdleConns:    int32(cfg.Database.MaxIdleConns),
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	}

	// Execute migration command
	switch command {
	case "up":
		fmt.Println("Running migrations...")
		if err := database.RunMigrations(dbCfg, absPath); err != nil {
			slog.Error("migration failed", slog.Any("error", err))
			os.Exit(1)
		}
		fmt.Println("Migrations completed successfully")

	case "down":
		fmt.Println("Rolling back last migration...")
		if err := database.RollbackMigration(dbCfg, absPath); err != nil {
			slog.Error("rollback failed", slog.Any("error", err))
			os.Exit(1)
		}
		fmt.Println("Rollback completed successfully")

	case "version":
		version, dirty, err := database.GetMigrationVersion(dbCfg, absPath)
		if err != nil {
			slog.Error("failed to get migration version", slog.Any("error", err))
			os.Exit(1)
		}
		fmt.Printf("Current migration version: %d\n", version)
		if dirty {
			fmt.Println("WARNING: Database is in a dirty state. Manual intervention may be required.")
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		fmt.Fprintf(os.Stderr, "Available commands: up, down, version\n")
		os.Exit(1)
	}
}
