// Package main provides a CLI for managing Email Ingest Service API keys.
//
// Usage:
//
//	manage_keys list                          # List all keys
//	manage_keys create --name <name>          # Create a new key
//	manage_keys rotate --name <name>          # Rotate a key: create new, revoke old
//	manage_keys revoke --name <name>          # Revoke a key immediately
//
// All commands require the same DB_* environment variables as the service itself.
package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load .env file if present (best-effort; env vars take precedence)
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	db, err := openDB()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	ctx := context.Background()

	switch os.Args[1] {
	case "list":
		if err := cmdList(ctx, db); err != nil {
			log.Fatalf("list failed: %v", err)
		}
	case "create":
		fs := flag.NewFlagSet("create", flag.ExitOnError)
		name := fs.String("name", "", "human-readable key name (required)")
		notes := fs.String("notes", "", "optional notes about the key's purpose")
		if err := fs.Parse(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		if *name == "" {
			log.Fatal("--name is required")
		}
		if err := cmdCreate(ctx, db, *name, *notes); err != nil {
			log.Fatalf("create failed: %v", err)
		}
	case "rotate":
		fs := flag.NewFlagSet("rotate", flag.ExitOnError)
		name := fs.String("name", "", "key name to rotate (required)")
		notes := fs.String("notes", "", "optional notes for the new key")
		if err := fs.Parse(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		if *name == "" {
			log.Fatal("--name is required")
		}
		if err := cmdRotate(ctx, db, *name, *notes); err != nil {
			log.Fatalf("rotate failed: %v", err)
		}
	case "revoke":
		fs := flag.NewFlagSet("revoke", flag.ExitOnError)
		name := fs.String("name", "", "key name to revoke (required)")
		if err := fs.Parse(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		if *name == "" {
			log.Fatal("--name is required")
		}
		if err := cmdRevoke(ctx, db, *name); err != nil {
			log.Fatalf("revoke failed: %v", err)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `Email Ingest API key management tool.

Usage:
  manage_keys list                       List all API keys and their status
  manage_keys create --name <name>       Create a new API key
  manage_keys rotate --name <name>       Rotate a key: creates a new one and revokes the old one
  manage_keys revoke --name <name>       Revoke a key immediately

Required environment variables (same as the service):
  DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME

Optional:
  DB_SSL_MODE  (default: disable)`)
}

// openDB opens a database connection using the same env vars as the service.
func openDB() (*sql.DB, error) {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "cairn_user_email")
	password := getEnv("DB_PASSWORD", "")
	dbname := getEnv("DB_NAME", "ingest_email")
	sslmode := getEnv("DB_SSL_MODE", "disable")

	if password == "" {
		return nil, fmt.Errorf("DB_PASSWORD environment variable is required")
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %w", err)
	}

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		host, portInt, user, password, dbname, sslmode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("cannot reach database: %w", err)
	}
	return db, nil
}

// generateKey returns a cryptographically random 32-byte key encoded as base64 URL.
func generateKey() (rawKey, keyHash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("failed to generate random key: %w", err)
	}
	rawKey = base64.URLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(rawKey))
	keyHash = hex.EncodeToString(sum[:])
	return rawKey, keyHash, nil
}

// cmdList prints all API keys in a table.
func cmdList(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `
		SELECT key_name, status, created_at, expires_at, last_used_at, revoked_at, notes
		FROM api_keys
		ORDER BY created_at DESC
	`)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tSTATUS\tCREATED\tEXPIRES\tLAST USED\tNOTES")

	for rows.Next() {
		var (
			name       string
			status     string
			createdAt  time.Time
			expiresAt  sql.NullTime
			lastUsedAt sql.NullTime
			revokedAt  sql.NullTime
			notes      sql.NullString
		)
		if err := rows.Scan(&name, &status, &createdAt, &expiresAt, &lastUsedAt, &revokedAt, &notes); err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}

		expires := "never"
		if expiresAt.Valid {
			expires = expiresAt.Time.Format("2006-01-02")
		}
		lastUsed := "never"
		if lastUsedAt.Valid {
			lastUsed = lastUsedAt.Time.Format("2006-01-02 15:04")
		}
		notesStr := ""
		if notes.Valid {
			notesStr = notes.String
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			name, status, createdAt.Format("2006-01-02"), expires, lastUsed, notesStr,
		)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return w.Flush()
}

// cmdCreate creates a new API key and prints the raw key value.
func cmdCreate(ctx context.Context, db *sql.DB, name, notes string) error {
	rawKey, keyHash, err := generateKey()
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO api_keys (key_name, key_hash, status, created_by, notes)
		VALUES ($1, $2, 'active', $3, $4)
	`, name, keyHash, currentUser(), nullString(notes))
	if err != nil {
		return fmt.Errorf("insert failed: %w", err)
	}

	fmt.Printf("Created API key %q\n", name)
	fmt.Printf("Raw key (copy now — not stored): %s\n", rawKey)
	fmt.Println("Update INGEST_API_KEY in the service configuration and restart the service.")
	return nil
}

// cmdRotate creates a new key for the given name, then revokes the old one.
// A name suffix "-v2", "-v3", etc. is appended to the new key name.
func cmdRotate(ctx context.Context, db *sql.DB, name, notes string) error {
	// Verify the existing key exists and is active
	var count int
	err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM api_keys WHERE key_name = $1 AND status = 'active'", name,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("no active key with name %q found", name)
	}

	newName := name + "-rotated-" + time.Now().UTC().Format("20060102")
	rawKey, keyHash, err := generateKey()
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Insert new key
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO api_keys (key_name, key_hash, status, created_by, notes)
		VALUES ($1, $2, 'active', $3, $4)
	`, newName, keyHash, currentUser(), nullString(notes)); err != nil {
		return fmt.Errorf("insert new key: %w", err)
	}

	// Revoke the old key
	if _, err = tx.ExecContext(ctx, `
		UPDATE api_keys
		SET status = 'revoked', revoked_at = NOW()
		WHERE key_name = $1 AND status = 'active'
	`, name); err != nil {
		return fmt.Errorf("revoke old key: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	fmt.Printf("Rotated key %q → %q\n", name, newName)
	fmt.Printf("New raw key (copy now — not stored): %s\n", rawKey)
	fmt.Println("Update INGEST_API_KEY in the service configuration and restart the service.")
	fmt.Printf("Old key %q has been revoked.\n", name)
	return nil
}

// cmdRevoke marks a key as revoked.
func cmdRevoke(ctx context.Context, db *sql.DB, name string) error {
	res, err := db.ExecContext(ctx, `
		UPDATE api_keys
		SET status = 'revoked', revoked_at = NOW()
		WHERE key_name = $1 AND status = 'active'
	`, name)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("no active key with name %q found", name)
	}
	fmt.Printf("Key %q revoked.\n", name)
	return nil
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func currentUser() string {
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return "unknown"
}
