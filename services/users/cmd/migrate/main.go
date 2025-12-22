package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/andrew-craig/cairn-core/user-service/internal/config"
	"github.com/andrew-craig/cairn-core/user-service/internal/database"
	"github.com/joho/godotenv"
)

func main() {
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
		log.Printf("Warning: Could not load %s file: %v", envFile, err)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Convert relative path to absolute path
	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		log.Fatalf("Failed to resolve migrations path: %v", err)
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
			log.Fatalf("Migration failed: %v", err)
		}
		fmt.Println("Migrations completed successfully")

	case "down":
		fmt.Println("Rolling back last migration...")
		if err := database.RollbackMigration(dbCfg, absPath); err != nil {
			log.Fatalf("Rollback failed: %v", err)
		}
		fmt.Println("Rollback completed successfully")

	case "version":
		version, dirty, err := database.GetMigrationVersion(dbCfg, absPath)
		if err != nil {
			log.Fatalf("Failed to get migration version: %v", err)
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
