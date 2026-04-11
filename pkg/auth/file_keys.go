package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

// FileKeyProvider loads RSA JWT keys from PEM files on disk.
// If the files don't exist and AutoGenerate is true, it generates a new key pair.
type FileKeyProvider struct {
	PrivateKeyPath string
	PublicKeyPath  string
	AutoGenerate   bool
}

// GetKeys loads (or generates) the RSA key pair and returns both keys.
func (f *FileKeyProvider) GetKeys() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	// If files don't exist and auto-generate is enabled, create them
	if f.AutoGenerate && !fileExists(f.PrivateKeyPath) {
		if err := f.generateAndSave(); err != nil {
			return nil, nil, fmt.Errorf("failed to auto-generate JWT keys: %w", err)
		}
	}

	privateKey, err := f.loadPrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load private key: %w", err)
	}

	// Derive public key from private key (don't require separate public key file)
	publicKey := &privateKey.PublicKey

	// If public key file exists, also write/verify it; if not, save it
	if !fileExists(f.PublicKeyPath) {
		if err := savePublicKey(f.PublicKeyPath, publicKey); err != nil {
			return nil, nil, fmt.Errorf("failed to save public key: %w", err)
		}
	}

	return privateKey, publicKey, nil
}

// GetPublicKey loads only the public key (for services that only validate tokens).
func (f *FileKeyProvider) GetPublicKey() (*rsa.PublicKey, error) {
	data, err := os.ReadFile(f.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from public key file")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return rsaPub, nil
}

func (f *FileKeyProvider) loadPrivateKey() (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(f.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from private key file")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format
		parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		rsaKey, ok := parsedKey.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA private key")
		}
		return rsaKey, nil
	}

	return key, nil
}

func (f *FileKeyProvider) generateAndSave() error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Ensure directories exist
	if err := os.MkdirAll(filepath.Dir(f.PrivateKeyPath), 0700); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(f.PublicKeyPath), 0700); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	// Save private key
	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	})
	if err := os.WriteFile(f.PrivateKeyPath, privPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Save public key
	if err := savePublicKey(f.PublicKeyPath, &privateKey.PublicKey); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	return nil
}

func savePublicKey(path string, key *rsa.PublicKey) error {
	pubBytes, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})
	return os.WriteFile(path, pubPEM, 0644)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
