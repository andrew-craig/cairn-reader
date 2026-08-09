package auth

import (
	"context"
	"crypto/rsa"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakeKeyFetcher is a fake PublicKeyFetcher standing in for Vault. It lets a
// test simulate an out-of-band key rotation (as performed by the users
// service's KeyRotationManager) by swapping the key it returns, and counts
// how many times it was called so debounce behavior can be asserted.
type fakeKeyFetcher struct {
	mu    sync.Mutex
	key   *rsa.PublicKey
	calls int
}

func newFakeKeyFetcher(key *rsa.PublicKey) *fakeKeyFetcher {
	return &fakeKeyFetcher{key: key}
}

func (f *fakeKeyFetcher) GetPublicKey(_ string) (*rsa.PublicKey, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.key, nil
}

func (f *fakeKeyFetcher) rotate(key *rsa.PublicKey) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.key = key
}

func (f *fakeKeyFetcher) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// TestKeyRefresher_OnDemandRecoversWithoutRestart reproduces P2-C1: a
// Validator built once from a stale key must recover on its own, with no
// process restart and no explicit Start() call, purely from the
// on-signature-failure hook that NewKeyRefresher wires up.
func TestKeyRefresher_OnDemandRecoversWithoutRestart(t *testing.T) {
	_, oldPub := generateTestKeys(t)
	newPriv, newPub := generateTestKeys(t)

	validator := NewValidator(oldPub)
	fetcher := newFakeKeyFetcher(oldPub)
	NewKeyRefresher(fetcher, "secret/data/jwt/public-key", validator)

	// Simulate the users service rotating its signing key out-of-band: the
	// source of truth (Vault) now serves the new key, and new tokens are
	// signed with the new private key. The validator was never told.
	fetcher.rotate(newPub)
	token := generateTestToken(t, newPriv, uuid.New(), "cairn-user-service", "cairn-api", time.Hour)

	claims, err := validator.ValidateToken(token)
	if err != nil {
		t.Fatalf("expected validation to recover via on-demand refresh, got error: %v", err)
	}
	if claims == nil {
		t.Fatal("expected non-nil claims")
	}
}

// TestKeyRefresher_PollRecoversProactively verifies the periodic-poll path:
// a rotation is picked up on the next tick, with no token validation
// involved at all.
//
// It deliberately never calls ValidateToken. NewKeyRefresher also registers
// the on-signature-failure hook, so any failed validation would refresh the
// key on demand and mask a poll loop that never runs — the key change is
// observed directly through the validator instead, which only refresh() can
// cause. With Start() stubbed out to a no-op this test fails.
func TestKeyRefresher_PollRecoversProactively(t *testing.T) {
	_, oldPub := generateTestKeys(t)
	_, newPub := generateTestKeys(t)

	validator := NewValidator(oldPub)
	fetcher := newFakeKeyFetcher(oldPub)
	kr := NewKeyRefresher(fetcher, "secret/data/jwt/public-key", validator)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	kr.Start(ctx, 5*time.Millisecond)

	fetcher.rotate(newPub)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if validator.GetPublicKey().Equal(newPub) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("key was not refreshed by polling within deadline")
}

// TestKeyRefresher_OnDemandDebounced ensures repeated signature failures
// (e.g. a client hammering the endpoint with a stale or forged token) cannot
// force unbounded reads against the key source.
func TestKeyRefresher_OnDemandDebounced(t *testing.T) {
	_, oldPub := generateTestKeys(t)
	badPriv, _ := generateTestKeys(t) // never registered with the fetcher

	validator := NewValidator(oldPub)
	fetcher := newFakeKeyFetcher(oldPub)
	NewKeyRefresher(fetcher, "secret/data/jwt/public-key", validator)

	token := generateTestToken(t, badPriv, uuid.New(), "cairn-user-service", "cairn-api", time.Hour)

	if _, err := validator.ValidateToken(token); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
	if _, err := validator.ValidateToken(token); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}

	if got := fetcher.callCount(); got != 1 {
		t.Fatalf("expected exactly 1 fetch across two rapid failures (debounced), got %d", got)
	}
}

// TestKeyRefresher_PreRotationTokenStillValid guards against a regression
// where wiring the refresher breaks the ordinary happy path.
func TestKeyRefresher_PreRotationTokenStillValid(t *testing.T) {
	priv, pub := generateTestKeys(t)

	validator := NewValidator(pub)
	fetcher := newFakeKeyFetcher(pub)
	NewKeyRefresher(fetcher, "secret/data/jwt/public-key", validator)

	token := generateTestToken(t, priv, uuid.New(), "cairn-user-service", "cairn-api", time.Hour)

	if _, err := validator.ValidateToken(token); err != nil {
		t.Fatalf("expected valid token to pass, got error: %v", err)
	}
}
