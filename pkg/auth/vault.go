package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	vault "github.com/hashicorp/vault/api"
)

// VaultClient wraps the Vault API client for fetching public keys
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

// NewVaultClient creates a new Vault client for fetching public keys
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

// GetPublicKey retrieves the JWT public key from Vault
func (v *VaultClient) GetPublicKey(publicKeyPath string) (*rsa.PublicKey, error) {
	// Get public key
	publicKeyData, err := v.GetSecret(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	publicKeyPEM, ok := publicKeyData["key"].(string)
	if !ok {
		return nil, fmt.Errorf("public key not found in secret data")
	}

	publicKey, err := parsePublicKey(publicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return publicKey, nil
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
