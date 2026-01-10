package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/logging"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Initialize logger
	logger := logging.NewLogger(logging.Config{
		Level:       "info",
		Format:      "text",
		ServiceName: "article-cleanup",
	})
	slog.SetDefault(logger)

	slog.Info("Article Cleanup Utility")
	slog.Info("=======================")

	// Database configuration
	dbConfig := db.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "cairn"),
		Password: getEnv("DB_PASSWORD", "cairn_password"),
		DBName:   getEnv("DB_NAME", "cairn_db"),
	}

	// Get retention days from args or env
	retentionDays := getEnvAsInt("ARTICLE_RETENTION_DAYS", 90)
	if len(os.Args) > 1 {
		days, err := strconv.Atoi(os.Args[1])
		if err == nil {
			retentionDays = days
		} else {
			slog.Warn("invalid retention days argument, using default",
				slog.Int("default", retentionDays),
			)
		}
	}

	slog.Info("retention period configured", slog.Int("days", retentionDays))

	// Connect to database
	database, err := connectDB(dbConfig)
	if err != nil {
		slog.Error("failed to connect to database", slog.Any("error", err))
		os.Exit(1)
	}
	defer database.Close()

	// Initialize article repository
	articleRepo := db.NewArticleRepository(database)

	// Run cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	slog.Info("marking old articles as deleted", slog.Int("days", retentionDays))
	softDeleteCount, err := articleRepo.MarkOldArticlesAsDeleted(ctx, retentionDays)
	if err != nil {
		slog.Error("failed to mark articles as deleted", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("marked articles as deleted", slog.Int("count", softDeleteCount))

	// Hard delete old soft-deleted articles
	hardDeleteDays := retentionDays + 30 // Delete articles that have been soft-deleted for 30+ days
	slog.Info("hard deleting old articles", slog.Int("days", hardDeleteDays))
	hardDeleteCount, err := articleRepo.HardDeleteOldArticles(ctx, hardDeleteDays)
	if err != nil {
		slog.Error("failed to hard delete articles", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("hard deleted articles", slog.Int("count", hardDeleteCount))

	slog.Info("cleanup completed successfully")
}

func connectDB(config db.Config) (*pgxpool.Pool, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Host, config.Port, config.User, config.Password, config.DBName)

	ctx := context.Background()
	database, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, err
	}

	// Test connection
	if err := database.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("successfully connected to database")
	return database, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		slog.Warn("invalid environment variable value, using default",
			slog.String("key", key),
			slog.String("value", valueStr),
			slog.Int("default", defaultValue),
		)
		return defaultValue
	}
	return value
}
