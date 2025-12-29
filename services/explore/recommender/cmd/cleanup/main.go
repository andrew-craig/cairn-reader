package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/andrew-craig/cairn/services/explore/recommender/internal/db"
	_ "github.com/lib/pq"
)

func main() {
	log.Println("Article Cleanup Utility")
	log.Println("=======================")

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
			log.Printf("Invalid retention days argument, using default: %d", retentionDays)
		}
	}

	log.Printf("Retention period: %d days", retentionDays)

	// Connect to database
	database, err := connectDB(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Printf("error closing database: %v", err)
		}
	}()

	// Initialize article repository
	articleRepo := db.NewArticleRepository(database)

	// Run cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Printf("Marking articles older than %d days as deleted...", retentionDays)
	softDeleteCount, err := articleRepo.MarkOldArticlesAsDeleted(ctx, retentionDays)
	if err != nil {
		log.Fatalf("Failed to mark articles as deleted: %v", err)
	}
	log.Printf("Marked %d articles as deleted", softDeleteCount)

	// Hard delete old soft-deleted articles
	hardDeleteDays := retentionDays + 30 // Delete articles that have been soft-deleted for 30+ days
	log.Printf("Hard deleting articles older than %d days (and already marked as deleted)...", hardDeleteDays)
	hardDeleteCount, err := articleRepo.HardDeleteOldArticles(ctx, hardDeleteDays)
	if err != nil {
		log.Fatalf("Failed to hard delete articles: %v", err)
	}
	log.Printf("Hard deleted %d articles", hardDeleteCount)

	log.Println("Cleanup completed successfully")
}

func connectDB(config db.Config) (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Host, config.Port, config.User, config.Password, config.DBName)

	database, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Test connection
	if err := database.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to database")
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
		log.Printf("Invalid value for %s: %s, using default: %d", key, valueStr, defaultValue)
		return defaultValue
	}
	return value
}
