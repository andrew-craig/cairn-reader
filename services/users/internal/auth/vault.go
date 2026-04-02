package auth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"sync"
	"time"

	vault "github.com/hashicorp/vault/api"
)

// VaultClient wraps the Vault API client
type VaultClient struct {
	client *vault.Client
}

// VaultConfig holds Vault connection configuration
type VaultConfig struct {
	Address   string
	Token     string
	Namespace string
	RoleID    string
	SecretID  string
	AuthPath  string
}

// NewVaultClient creates a new Vault client
func NewVaultClient(cfg *VaultConfig) (*VaultClient, error) {
	config := vault.DefaultConfig()
	config.Address = cfg.Address

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	if cfg.Namespace != "" {
		client.SetNamespace(cfg.Namespace)
	}

	// Authenticate with Vault
	if cfg.Token != "" {
		client.SetToken(cfg.Token)
	} else if cfg.RoleID != "" && cfg.SecretID != "" {
		// Use AppRole authentication
		data := map[string]interface{}{
			"role_id":   cfg.RoleID,
			"secret_id": cfg.SecretID,
		}

		resp, err := client.Logical().Write(fmt.Sprintf("auth/%s/login", cfg.AuthPath), data)
		if err != nil {
			return nil, fmt.Errorf("failed to authenticate with vault: %w", err)
		}

		if resp.Auth == nil {
			return nil, fmt.Errorf("no auth info returned from vault")
		}

		client.SetToken(resp.Auth.ClientToken)
	} else {
		return nil, fmt.Errorf("no vault authentication method provided")
	}

	return &VaultClient{client: client}, nil
}

// GetSecret retrieves a secret from Vault
func (v *VaultClient) GetSecret(path string) (map[string]interface{}, error) {
	secret, err := v.client.Logical().Read(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret from vault: %w", err)
	}

	if secret == nil {
		return nil, fmt.Errorf("secret not found at path: %s", path)
	}

	// KV v2 stores data in secret.Data["data"]
	if data, ok := secret.Data["data"].(map[string]interface{}); ok {
		return data, nil
	}

	// KV v1 stores data directly in secret.Data
	return secret.Data, nil
}

// GetJWTKeys retrieves JWT private and public keys from Vault
func (v *VaultClient) GetJWTKeys(privateKeyPath, publicKeyPath string) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	// Get private key
	privateKeyData, err := v.GetSecret(privateKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get private key: %w", err)
	}

	privateKeyPEM, ok := privateKeyData["key"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("private key not found in secret data")
	}

	privateKey, err := parsePrivateKey(privateKeyPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Get public key
	publicKeyData, err := v.GetSecret(publicKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get public key: %w", err)
	}

	publicKeyPEM, ok := publicKeyData["key"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("public key not found in secret data")
	}

	publicKey, err := parsePublicKey(publicKeyPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return privateKey, publicKey, nil
}

// parsePrivateKey parses a PEM-encoded RSA private key
func parsePrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format
		parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rsaKey, ok := parsedKey.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA private key")
		}
		return rsaKey, nil
	}

	return key, nil
}

// parsePublicKey parses a PEM-encoded RSA public key
func parsePublicKey(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return rsaPub, nil
}

// DatabaseCredentials holds database connection credentials
type DatabaseCredentials struct {
	Host     string
	Port     string
	Username string
	Password string
	Database string
	SSLMode  string
}

// GetDatabaseCredentials retrieves database credentials from Vault
func (v *VaultClient) GetDatabaseCredentials(path string) (*DatabaseCredentials, error) {
	secretData, err := v.GetSecret(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get database credentials: %w", err)
	}

	creds := &DatabaseCredentials{}

	// Extract credentials from secret data
	if host, ok := secretData["host"].(string); ok {
		creds.Host = host
	}
	if port, ok := secretData["port"].(string); ok {
		creds.Port = port
	}
	if username, ok := secretData["username"].(string); ok {
		creds.Username = username
	}
	if password, ok := secretData["password"].(string); ok {
		creds.Password = password
	}
	if database, ok := secretData["database"].(string); ok {
		creds.Database = database
	}
	if sslmode, ok := secretData["sslmode"].(string); ok {
		creds.SSLMode = sslmode
	}

	// Validate required fields
	if creds.Host == "" || creds.Username == "" || creds.Password == "" || creds.Database == "" {
		return nil, fmt.Errorf("incomplete database credentials in Vault")
	}

	// Set defaults
	if creds.Port == "" {
		creds.Port = "5432"
	}
	if creds.SSLMode == "" {
		creds.SSLMode = "require"
	}

	return creds, nil
}

// JWTKeyPair holds RSA private and public keys for JWT operations
type JWTKeyPair struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

// RefreshJWTKeys retrieves fresh JWT keys from Vault (for key rotation)
func (v *VaultClient) RefreshJWTKeys(privateKeyPath, publicKeyPath string) (*JWTKeyPair, error) {
	privateKey, publicKey, err := v.GetJWTKeys(privateKeyPath, publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh JWT keys: %w", err)
	}

	return &JWTKeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, nil
}

// RenewToken renews the Vault token before it expires (if using token auth)
func (v *VaultClient) RenewToken() error {
	secret, err := v.client.Auth().Token().RenewSelf(0)
	if err != nil {
		return fmt.Errorf("failed to renew vault token: %w", err)
	}

	if secret.Auth == nil {
		return fmt.Errorf("no auth info returned from token renewal")
	}

	return nil
}

// Health checks Vault connectivity
func (v *VaultClient) Health() error {
	health, err := v.client.Sys().Health()
	if err != nil {
		return fmt.Errorf("vault health check failed: %w", err)
	}

	if !health.Initialized {
		return fmt.Errorf("vault is not initialized")
	}

	if health.Sealed {
		return fmt.Errorf("vault is sealed")
	}

	return nil
}

// KeyRotationManager handles automatic key rotation and token renewal
type KeyRotationManager struct {
	vaultClient          *VaultClient
	privateKeyPath       string
	publicKeyPath        string
	rotationInterval     time.Duration
	tokenRenewalInterval time.Duration
	currentKeyPair       *JWTKeyPair
	mu                   sync.RWMutex
	stopCh               chan struct{}
	onRotation           func(*JWTKeyPair) error // Callback when keys are rotated
}

// KeyRotationConfig holds configuration for key rotation
type KeyRotationConfig struct {
	PrivateKeyPath       string
	PublicKeyPath        string
	RotationInterval     time.Duration           // How often to check for new keys (0 to disable)
	TokenRenewalInterval time.Duration           // How often to renew Vault token (0 to disable)
	OnRotation           func(*JWTKeyPair) error // Optional callback when keys rotate
}

// NewKeyRotationManager creates a new key rotation manager
func NewKeyRotationManager(vaultClient *VaultClient, config KeyRotationConfig) (*KeyRotationManager, error) {
	// Load initial keys
	keyPair, err := vaultClient.RefreshJWTKeys(config.PrivateKeyPath, config.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load initial JWT keys: %w", err)
	}

	return &KeyRotationManager{
		vaultClient:          vaultClient,
		privateKeyPath:       config.PrivateKeyPath,
		publicKeyPath:        config.PublicKeyPath,
		rotationInterval:     config.RotationInterval,
		tokenRenewalInterval: config.TokenRenewalInterval,
		currentKeyPair:       keyPair,
		stopCh:               make(chan struct{}),
		onRotation:           config.OnRotation,
	}, nil
}

// Start begins the background key rotation and token renewal process
func (krm *KeyRotationManager) Start(ctx context.Context) {
	// Start key rotation goroutine if interval is set
	if krm.rotationInterval > 0 {
		go krm.rotateKeysLoop(ctx)
	}

	// Start token renewal goroutine if interval is set
	if krm.tokenRenewalInterval > 0 {
		go krm.renewTokenLoop(ctx)
	}
}

// Stop stops the background rotation and renewal processes
func (krm *KeyRotationManager) Stop() {
	close(krm.stopCh)
}

// GetCurrentKeyPair returns the current JWT key pair (thread-safe)
func (krm *KeyRotationManager) GetCurrentKeyPair() *JWTKeyPair {
	krm.mu.RLock()
	defer krm.mu.RUnlock()
	return krm.currentKeyPair
}

// rotateKeysLoop periodically checks and rotates JWT keys
func (krm *KeyRotationManager) rotateKeysLoop(ctx context.Context) {
	ticker := time.NewTicker(krm.rotationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-krm.stopCh:
			return
		case <-ticker.C:
			if err := krm.rotateKeys(); err != nil {
				// Log error but don't stop the loop
				slog.Error("key rotation failed",
					slog.Any("error", err),
				)
			}
		}
	}
}

// renewTokenLoop periodically renews the Vault token
func (krm *KeyRotationManager) renewTokenLoop(ctx context.Context) {
	ticker := time.NewTicker(krm.tokenRenewalInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-krm.stopCh:
			return
		case <-ticker.C:
			if err := krm.vaultClient.RenewToken(); err != nil {
				// Log error but don't stop the loop
				slog.Error("vault token renewal failed",
					slog.Any("error", err),
				)
			}
		}
	}
}

// rotateKeys fetches new keys from Vault and updates the current key pair
func (krm *KeyRotationManager) rotateKeys() error {
	// Fetch new keys from Vault
	newKeyPair, err := krm.vaultClient.RefreshJWTKeys(krm.privateKeyPath, krm.publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to fetch new keys: %w", err)
	}

	// Update current key pair atomically
	krm.mu.Lock()
	oldKeyPair := krm.currentKeyPair
	krm.currentKeyPair = newKeyPair
	krm.mu.Unlock()

	// Call rotation callback if provided
	if krm.onRotation != nil {
		if err := krm.onRotation(newKeyPair); err != nil {
			// Rollback to old keys on callback failure
			krm.mu.Lock()
			krm.currentKeyPair = oldKeyPair
			krm.mu.Unlock()
			return fmt.Errorf("rotation callback failed: %w", err)
		}
	}

	return nil
}

// ForceRotation immediately rotates the keys (useful for testing or manual rotation)
func (krm *KeyRotationManager) ForceRotation() error {
	return krm.rotateKeys()
}
