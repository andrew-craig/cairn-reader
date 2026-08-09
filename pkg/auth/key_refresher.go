package auth

import (
	"context"
	"crypto/rsa"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// PublicKeyFetcher retrieves the current JWT public key from its source of
// truth. *VaultClient satisfies this directly; it is kept as a minimal
// interface so KeyRefresher can be tested without a real Vault.
type PublicKeyFetcher interface {
	GetPublicKey(publicKeyPath string) (*rsa.PublicKey, error)
}

// KeyRefresher keeps a Validator's public key in sync with its source of
// truth.
//
// The users service rotates its JWT signing key on a timer
// (KeyRotationManager, JWT_KEY_ROTATION_INTERVAL, default 24h). A Validator
// built once at startup from a single GetPublicKey call has no way to learn
// about that rotation, so every token issued after it fails signature
// verification until the process is restarted. KeyRefresher closes that gap
// two ways:
//   - Start polls the fetcher on an interval and pushes any change into the
//     validator, so a rotation is picked up proactively, before it can cause
//     a validation failure.
//   - Once constructed, it is registered as the validator's on-signature-
//     failure hook (see Validator.SetOnSignatureFailure), so if a token still
//     fails (e.g. rotation happened between polls, or polling is disabled)
//     it re-fetches on demand and lets validation retry immediately.
type KeyRefresher struct {
	fetcher       PublicKeyFetcher
	publicKeyPath string
	validator     *Validator

	// minOnDemandGap debounces on-demand refreshes so that repeated
	// signature failures (a client presenting a stale token, or an
	// attacker sending crafted bad signatures) cannot be used to force
	// unbounded reads against the key source.
	minOnDemandGap time.Duration

	mu           sync.Mutex
	lastOnDemand time.Time
}

// NewKeyRefresher creates a KeyRefresher that keeps validator's key current
// by fetching from fetcher at publicKeyPath, and registers itself as
// validator's on-signature-failure hook.
func NewKeyRefresher(fetcher PublicKeyFetcher, publicKeyPath string, validator *Validator) *KeyRefresher {
	kr := &KeyRefresher{
		fetcher:        fetcher,
		publicKeyPath:  publicKeyPath,
		validator:      validator,
		minOnDemandGap: 10 * time.Second,
	}
	validator.SetOnSignatureFailure(kr.refreshOnDemand)
	return kr
}

// Start begins polling the key source every interval, until ctx is done. A
// non-positive interval disables polling; on-demand refresh (triggered by a
// signature failure) still applies regardless.
func (kr *KeyRefresher) Start(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		return
	}
	go kr.pollLoop(ctx, interval)
}

func (kr *KeyRefresher) pollLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := kr.refresh(); err != nil {
				slog.Error("periodic JWT public key refresh failed", slog.Any("error", err))
			}
		}
	}
}

// refreshOnDemand is the validator's on-signature-failure hook.
func (kr *KeyRefresher) refreshOnDemand() {
	kr.mu.Lock()
	if time.Since(kr.lastOnDemand) < kr.minOnDemandGap {
		kr.mu.Unlock()
		return
	}
	kr.lastOnDemand = time.Now()
	kr.mu.Unlock()

	if err := kr.refresh(); err != nil {
		slog.Error("on-demand JWT public key refresh failed", slog.Any("error", err))
	}
}

func (kr *KeyRefresher) refresh() error {
	publicKey, err := kr.fetcher.GetPublicKey(kr.publicKeyPath)
	if err != nil {
		return fmt.Errorf("fetch public key: %w", err)
	}
	kr.validator.UpdatePublicKey(publicKey)
	return nil
}
