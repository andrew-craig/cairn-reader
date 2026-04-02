package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVaultConfig tests VaultConfig structure
func TestVaultConfig(t *testing.T) {
	cfg := &VaultConfig{
		Address:   "http://localhost:8200",
		Token:     "test-token",
		Namespace: "test-namespace",
		RoleID:    "test-role-id",
		SecretID:  "test-secret-id",
		AuthPath:  "approle",
	}

	assert.Equal(t, "http://localhost:8200", cfg.Address)
	assert.Equal(t, "test-token", cfg.Token)
	assert.Equal(t, "test-namespace", cfg.Namespace)
	assert.Equal(t, "test-role-id", cfg.RoleID)
	assert.Equal(t, "test-secret-id", cfg.SecretID)
	assert.Equal(t, "approle", cfg.AuthPath)
}

// TestDatabaseCredentials tests DatabaseCredentials structure
func TestDatabaseCredentials(t *testing.T) {
	creds := &DatabaseCredentials{
		Host:     "localhost",
		Port:     "5432",
		Username: "cairn_user",
		Password: "secret",
		Database: "cairn_db",
		SSLMode:  "require",
	}

	assert.Equal(t, "localhost", creds.Host)
	assert.Equal(t, "5432", creds.Port)
	assert.Equal(t, "cairn_user", creds.Username)
	assert.Equal(t, "secret", creds.Password)
	assert.Equal(t, "cairn_db", creds.Database)
	assert.Equal(t, "require", creds.SSLMode)
}

// TestParsePrivateKey tests parsing RSA private keys in different formats
func TestParsePrivateKey(t *testing.T) {
	tests := []struct {
		name    string
		keyGen  func() (*rsa.PrivateKey, string, error)
		wantErr bool
	}{
		{
			name: "PKCS1 format",
			keyGen: func() (*rsa.PrivateKey, string, error) {
				key, err := rsa.GenerateKey(rand.Reader, 2048)
				if err != nil {
					return nil, "", err
				}
				keyBytes := x509.MarshalPKCS1PrivateKey(key)
				pemBytes := pem.EncodeToMemory(&pem.Block{
					Type:  "RSA PRIVATE KEY",
					Bytes: keyBytes,
				})
				return key, string(pemBytes), nil
			},
			wantErr: false,
		},
		{
			name: "PKCS8 format",
			keyGen: func() (*rsa.PrivateKey, string, error) {
				key, err := rsa.GenerateKey(rand.Reader, 2048)
				if err != nil {
					return nil, "", err
				}
				keyBytes, err := x509.MarshalPKCS8PrivateKey(key)
				if err != nil {
					return nil, "", err
				}
				pemBytes := pem.EncodeToMemory(&pem.Block{
					Type:  "PRIVATE KEY",
					Bytes: keyBytes,
				})
				return key, string(pemBytes), nil
			},
			wantErr: false,
		},
		{
			name: "Invalid PEM",
			keyGen: func() (*rsa.PrivateKey, string, error) {
				return nil, "not a valid pem", nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalKey, pemStr, err := tt.keyGen()
			if err != nil {
				t.Fatalf("Failed to generate test key: %v", err)
			}

			parsedKey, err := parsePrivateKey(pemStr)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, parsedKey)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, parsedKey)
				// Verify the parsed key matches the original
				assert.Equal(t, originalKey.N, parsedKey.N)
			}
		})
	}
}

// TestParsePublicKey tests parsing RSA public keys
func TestParsePublicKey(t *testing.T) {
	tests := []struct {
		name    string
		keyGen  func() (*rsa.PublicKey, string, error)
		wantErr bool
	}{
		{
			name: "Valid public key",
			keyGen: func() (*rsa.PublicKey, string, error) {
				privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
				if err != nil {
					return nil, "", err
				}
				publicKey := &privateKey.PublicKey
				keyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
				if err != nil {
					return nil, "", err
				}
				pemBytes := pem.EncodeToMemory(&pem.Block{
					Type:  "PUBLIC KEY",
					Bytes: keyBytes,
				})
				return publicKey, string(pemBytes), nil
			},
			wantErr: false,
		},
		{
			name: "Invalid PEM",
			keyGen: func() (*rsa.PublicKey, string, error) {
				return nil, "not a valid pem", nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalKey, pemStr, err := tt.keyGen()
			if err != nil {
				t.Fatalf("Failed to generate test key: %v", err)
			}

			parsedKey, err := parsePublicKey(pemStr)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, parsedKey)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, parsedKey)
				// Verify the parsed key matches the original
				assert.Equal(t, originalKey.N, parsedKey.N)
				assert.Equal(t, originalKey.E, parsedKey.E)
			}
		})
	}
}

// TestKeyRotationConfig tests KeyRotationConfig structure
func TestKeyRotationConfig(t *testing.T) {
	callbackCalled := false
	callback := func(kp *JWTKeyPair) error {
		callbackCalled = true
		return nil
	}

	cfg := KeyRotationConfig{
		PrivateKeyPath:       "secret/data/jwt/private-key",
		PublicKeyPath:        "secret/data/jwt/public-key",
		RotationInterval:     24 * time.Hour,
		TokenRenewalInterval: 1 * time.Hour,
		OnRotation:           callback,
	}

	assert.Equal(t, "secret/data/jwt/private-key", cfg.PrivateKeyPath)
	assert.Equal(t, "secret/data/jwt/public-key", cfg.PublicKeyPath)
	assert.Equal(t, 24*time.Hour, cfg.RotationInterval)
	assert.Equal(t, 1*time.Hour, cfg.TokenRenewalInterval)
	assert.NotNil(t, cfg.OnRotation)

	// Test callback
	err := cfg.OnRotation(&JWTKeyPair{})
	assert.NoError(t, err)
	assert.True(t, callbackCalled)
}

// TestJWTKeyPair tests JWTKeyPair structure
func TestJWTKeyPair(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	publicKey := &privateKey.PublicKey

	keyPair := &JWTKeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}

	assert.NotNil(t, keyPair.PrivateKey)
	assert.NotNil(t, keyPair.PublicKey)
	assert.Equal(t, publicKey, keyPair.PublicKey)
	assert.Equal(t, privateKey, keyPair.PrivateKey)
}

// TestKeyRotationManager_GetCurrentKeyPair tests thread-safe key pair retrieval
func TestKeyRotationManager_GetCurrentKeyPair(t *testing.T) {
	// Note: This is a minimal test. Full integration test would require a Vault server.
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keyPair := &JWTKeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}

	// Create a minimal KeyRotationManager for testing
	krm := &KeyRotationManager{
		currentKeyPair: keyPair,
		stopCh:         make(chan struct{}),
	}

	// Test GetCurrentKeyPair
	retrieved := krm.GetCurrentKeyPair()
	assert.NotNil(t, retrieved)
	assert.Equal(t, keyPair.PrivateKey, retrieved.PrivateKey)
	assert.Equal(t, keyPair.PublicKey, retrieved.PublicKey)

	// Test that Stop doesn't panic
	krm.Stop()
}

// TestKeyRotationManager_Start tests starting key rotation manager
func TestKeyRotationManager_Start(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keyPair := &JWTKeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}

	tests := []struct {
		name                 string
		rotationInterval     time.Duration
		tokenRenewalInterval time.Duration
	}{
		{
			name:                 "Both intervals disabled",
			rotationInterval:     0,
			tokenRenewalInterval: 0,
		},
		{
			name:                 "Only rotation enabled",
			rotationInterval:     1 * time.Hour,
			tokenRenewalInterval: 0,
		},
		{
			name:                 "Only renewal enabled",
			rotationInterval:     0,
			tokenRenewalInterval: 1 * time.Hour,
		},
		{
			name:                 "Both enabled",
			rotationInterval:     1 * time.Hour,
			tokenRenewalInterval: 30 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			krm := &KeyRotationManager{
				rotationInterval:     tt.rotationInterval,
				tokenRenewalInterval: tt.tokenRenewalInterval,
				currentKeyPair:       keyPair,
				stopCh:               make(chan struct{}),
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Start should not panic
			assert.NotPanics(t, func() {
				krm.Start(ctx)
			})

			// Stop should not panic
			assert.NotPanics(t, func() {
				krm.Stop()
			})

			// Cancel context
			cancel()
		})
	}
}

// TestParsePrivateKey_ErrorCases tests additional error scenarios
func TestParsePrivateKey_ErrorCases(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		key, err := parsePrivateKey("")
		assert.Error(t, err)
		assert.Nil(t, key)
	})

	t.Run("invalid base64", func(t *testing.T) {
		key, err := parsePrivateKey("-----BEGIN RSA PRIVATE KEY-----\ninvalid!!!\n-----END RSA PRIVATE KEY-----")
		assert.Error(t, err)
		assert.Nil(t, key)
	})

	t.Run("wrong key type (public key instead of private)", func(t *testing.T) {
		// Generate a public key
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		publicKey := &privateKey.PublicKey
		keyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
		require.NoError(t, err)

		pemBytes := pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: keyBytes,
		})

		key, err := parsePrivateKey(string(pemBytes))
		assert.Error(t, err)
		assert.Nil(t, key)
	})
}

// TestParsePublicKey_ErrorCases tests additional error scenarios
func TestParsePublicKey_ErrorCases(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		key, err := parsePublicKey("")
		assert.Error(t, err)
		assert.Nil(t, key)
	})

	t.Run("invalid base64", func(t *testing.T) {
		key, err := parsePublicKey("-----BEGIN PUBLIC KEY-----\ninvalid!!!\n-----END PUBLIC KEY-----")
		assert.Error(t, err)
		assert.Nil(t, key)
	})

	t.Run("wrong key type (private key instead of public)", func(t *testing.T) {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		keyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
		pemBytes := pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: keyBytes,
		})

		key, err := parsePublicKey(string(pemBytes))
		assert.Error(t, err)
		assert.Nil(t, key)
	})

	t.Run("EC key instead of RSA", func(t *testing.T) {
		// This test would require generating an EC key, which would need crypto/ecdsa
		// For now, we can test with malformed data
		pemData := `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE
-----END PUBLIC KEY-----`

		key, err := parsePublicKey(pemData)
		assert.Error(t, err)
		assert.Nil(t, key)
	})
}

// TestDatabaseCredentials_Validation tests validation of database credentials
func TestDatabaseCredentials_Validation(t *testing.T) {
	t.Run("valid credentials", func(t *testing.T) {
		creds := &DatabaseCredentials{
			Host:     "localhost",
			Port:     "5432",
			Username: "user",
			Password: "pass",
			Database: "db",
			SSLMode:  "require",
		}

		assert.NotEmpty(t, creds.Host)
		assert.NotEmpty(t, creds.Port)
		assert.NotEmpty(t, creds.Username)
		assert.NotEmpty(t, creds.Password)
		assert.NotEmpty(t, creds.Database)
		assert.NotEmpty(t, creds.SSLMode)
	})

	t.Run("default port and sslmode", func(t *testing.T) {
		creds := &DatabaseCredentials{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
			Database: "db",
		}

		assert.Empty(t, creds.Port)
		assert.Empty(t, creds.SSLMode)
	})
}

// TestJWTKeyPair_Operations tests key pair operations
func TestJWTKeyPair_Operations(t *testing.T) {
	t.Run("key pair with matching keys", func(t *testing.T) {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		keyPair := &JWTKeyPair{
			PrivateKey: privateKey,
			PublicKey:  &privateKey.PublicKey,
		}

		// Verify the public key matches the private key
		assert.Equal(t, privateKey.N, keyPair.PublicKey.N)
		assert.Equal(t, privateKey.E, keyPair.PublicKey.E)
	})

	t.Run("nil key pair", func(t *testing.T) {
		keyPair := &JWTKeyPair{}

		assert.Nil(t, keyPair.PrivateKey)
		assert.Nil(t, keyPair.PublicKey)
	})
}

// TestKeyRotationConfig_Validation tests rotation config validation
func TestKeyRotationConfig_Validation(t *testing.T) {
	t.Run("valid config with callback", func(t *testing.T) {
		callbackCalled := false
		cfg := KeyRotationConfig{
			PrivateKeyPath:       "secret/jwt/private",
			PublicKeyPath:        "secret/jwt/public",
			RotationInterval:     1 * time.Hour,
			TokenRenewalInterval: 30 * time.Minute,
			OnRotation: func(kp *JWTKeyPair) error {
				callbackCalled = true
				return nil
			},
		}

		assert.NotEmpty(t, cfg.PrivateKeyPath)
		assert.NotEmpty(t, cfg.PublicKeyPath)
		assert.Greater(t, cfg.RotationInterval, time.Duration(0))
		assert.Greater(t, cfg.TokenRenewalInterval, time.Duration(0))
		assert.NotNil(t, cfg.OnRotation)

		err := cfg.OnRotation(&JWTKeyPair{})
		assert.NoError(t, err)
		assert.True(t, callbackCalled)
	})

	t.Run("config without callback", func(t *testing.T) {
		cfg := KeyRotationConfig{
			PrivateKeyPath:   "secret/jwt/private",
			PublicKeyPath:    "secret/jwt/public",
			RotationInterval: 1 * time.Hour,
		}

		assert.Nil(t, cfg.OnRotation)
	})

	t.Run("config with disabled rotation", func(t *testing.T) {
		cfg := KeyRotationConfig{
			PrivateKeyPath:       "secret/jwt/private",
			PublicKeyPath:        "secret/jwt/public",
			RotationInterval:     0,
			TokenRenewalInterval: 0,
		}

		assert.Zero(t, cfg.RotationInterval)
		assert.Zero(t, cfg.TokenRenewalInterval)
	})
}

// TestKeyRotationManager_ThreadSafety tests concurrent access to key pair
func TestKeyRotationManager_ThreadSafety(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keyPair := &JWTKeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}

	krm := &KeyRotationManager{
		currentKeyPair: keyPair,
		stopCh:         make(chan struct{}),
	}

	// Test concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				kp := krm.GetCurrentKeyPair()
				assert.NotNil(t, kp)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	krm.Stop()
}

// TestKeyRotationManager_StopMultipleTimes tests that Stop can be called multiple times
func TestKeyRotationManager_StopMultipleTimes(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keyPair := &JWTKeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}

	krm := &KeyRotationManager{
		currentKeyPair: keyPair,
		stopCh:         make(chan struct{}),
	}

	// First stop should work
	assert.NotPanics(t, func() {
		krm.Stop()
	})

	// Second stop should panic (closing closed channel)
	// This is expected behavior - we document that Stop should only be called once
	assert.Panics(t, func() {
		krm.Stop()
	})
}

// TestKeyRotationManager_ContextCancellation tests context cancellation
func TestKeyRotationManager_ContextCancellation(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keyPair := &JWTKeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}

	tests := []struct {
		name                 string
		rotationInterval     time.Duration
		tokenRenewalInterval time.Duration
	}{
		{
			name:                 "rotation enabled",
			rotationInterval:     100 * time.Millisecond,
			tokenRenewalInterval: 0,
		},
		{
			name:                 "renewal enabled",
			rotationInterval:     0,
			tokenRenewalInterval: 100 * time.Millisecond,
		},
		{
			name:                 "both enabled",
			rotationInterval:     100 * time.Millisecond,
			tokenRenewalInterval: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			krm := &KeyRotationManager{
				rotationInterval:     tt.rotationInterval,
				tokenRenewalInterval: tt.tokenRenewalInterval,
				currentKeyPair:       keyPair,
				stopCh:               make(chan struct{}),
			}

			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()

			krm.Start(ctx)

			// Wait for context to be cancelled
			<-ctx.Done()

			// Give goroutines time to exit
			time.Sleep(10 * time.Millisecond)

			// Should be able to stop without panic
			assert.NotPanics(t, func() {
				krm.Stop()
			})
		})
	}
}

// TestVaultConfig_DefaultValues tests default value handling
func TestVaultConfig_DefaultValues(t *testing.T) {
	t.Run("minimal config with token", func(t *testing.T) {
		cfg := &VaultConfig{
			Address: "http://localhost:8200",
			Token:   "test-token",
		}

		assert.NotEmpty(t, cfg.Address)
		assert.NotEmpty(t, cfg.Token)
		assert.Empty(t, cfg.Namespace)
		assert.Empty(t, cfg.RoleID)
		assert.Empty(t, cfg.SecretID)
	})

	t.Run("minimal config with approle", func(t *testing.T) {
		cfg := &VaultConfig{
			Address:  "http://localhost:8200",
			RoleID:   "role-123",
			SecretID: "secret-456",
			AuthPath: "approle",
		}

		assert.NotEmpty(t, cfg.Address)
		assert.NotEmpty(t, cfg.RoleID)
		assert.NotEmpty(t, cfg.SecretID)
		assert.NotEmpty(t, cfg.AuthPath)
		assert.Empty(t, cfg.Token)
	})

	t.Run("config with namespace", func(t *testing.T) {
		cfg := &VaultConfig{
			Address:   "http://localhost:8200",
			Token:     "test-token",
			Namespace: "admin/team",
		}

		assert.NotEmpty(t, cfg.Namespace)
	})
}

// Benchmark for key parsing operations
func BenchmarkParsePrivateKey(b *testing.B) {
	// Generate a test key once
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		b.Fatal(err)
	}

	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	})
	pemStr := string(pemBytes)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parsePrivateKey(pemStr)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParsePublicKey(b *testing.B) {
	// Generate a test key once
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		b.Fatal(err)
	}

	publicKey := &privateKey.PublicKey
	keyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		b.Fatal(err)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: keyBytes,
	})
	pemStr := string(pemBytes)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parsePublicKey(pemStr)
		if err != nil {
			b.Fatal(err)
		}
	}
}
