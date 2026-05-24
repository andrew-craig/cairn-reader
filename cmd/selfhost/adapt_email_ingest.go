package main

import (
	"context"
	"crypto/rsa"
	"fmt"
	"log/slog"

	"github.com/go-chi/chi/v5"

	emailMigrations "github.com/cairn-app/cairn-reader/services/read/email/migrations"
	emailSelfhost "github.com/cairn-app/cairn-reader/services/read/email/selfhost"
)

func runEmailMigrations(cfg *Config) error {
	slog.Info("running email migrations")
	return emailSelfhost.RunEmailMigrations(emailSelfhost.EmailConfig{
		DBHost:     cfg.DB.Host,
		DBPort:     cfg.DB.Port,
		DBUser:     cfg.DB.User,
		DBPassword: cfg.DB.Password,
		DBName:     cfg.DBNameEmail,
		DBSSLMode:  cfg.DB.SSLMode,
	}, emailMigrations.FS)
}

func mountEmailIngest(ctx context.Context, cfg *Config, r chi.Router, publicKey *rsa.PublicKey, health *healthChecker, logger *slog.Logger) (func(), error) {
	sqlDB, closer, err := emailSelfhost.MountEmail(ctx, emailSelfhost.EmailConfig{
		DBHost:         cfg.DB.Host,
		DBPort:         cfg.DB.Port,
		DBUser:         cfg.DB.User,
		DBPassword:     cfg.DB.Password,
		DBName:         cfg.DBNameEmail,
		DBSSLMode:      cfg.DB.SSLMode,
		EmailDomain:    cfg.EmailDomain,
		ContentBaseURL: fmt.Sprintf("http://localhost:%s", cfg.Port),
		IngestAPIKey:   cfg.EmailIngestAPIKey,
	}, r, publicKey, logger)
	if err != nil {
		return nil, err
	}

	health.addDB("email", sqlDB)
	return closer, nil
}
