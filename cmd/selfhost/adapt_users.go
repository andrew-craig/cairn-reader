package main

import (
	"context"
	"crypto/rsa"
	"fmt"
	"log/slog"

	"github.com/go-chi/chi/v5"

	usersSelfhost "github.com/cairn-app/cairn-reader/services/users/selfhost"
	userMigrations "github.com/cairn-app/cairn-reader/services/users/migrations"
)

func runUsersMigrations(cfg *Config) error {
	slog.Info("running users migrations")
	return usersSelfhost.RunUsersMigrations(usersSelfhost.UsersConfig{
		DBHost:     cfg.DB.Host,
		DBPort:     fmt.Sprintf("%d", cfg.DB.Port),
		DBUser:     cfg.DB.User,
		DBPassword: cfg.DB.Password,
		DBName:     cfg.DBNameUsers,
		DBSSLMode:  cfg.DB.SSLMode,
	}, userMigrations.FS)
}

func mountUserService(ctx context.Context, cfg *Config, r chi.Router, privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, health *healthChecker, logger *slog.Logger) (func(), error) {
	return usersSelfhost.MountUsers(ctx, usersSelfhost.UsersConfig{
		DBHost:           cfg.DB.Host,
		DBPort:           fmt.Sprintf("%d", cfg.DB.Port),
		DBUser:           cfg.DB.User,
		DBPassword:       cfg.DB.Password,
		DBName:           cfg.DBNameUsers,
		DBSSLMode:        cfg.DB.SSLMode,
		PrivateKey:       privateKey,
		PublicKey:         publicKey,
		JWTAccessExpiry:  cfg.JWTAccessExpiry,
		JWTRefreshExpiry: cfg.JWTRefreshExpiry,
		BcryptCost:       cfg.BcryptCost,
	}, r, logger)
}
