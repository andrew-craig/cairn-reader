package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestFileKeyProvider_AutoGenerate(t *testing.T) {
	dir := t.TempDir()
	provider := &FileKeyProvider{
		PrivateKeyPath: filepath.Join(dir, "private.pem"),
		PublicKeyPath:  filepath.Join(dir, "public.pem"),
		AutoGenerate:   true,
	}

	priv, pub, err := provider.GetKeys()
	if err != nil {
		t.Fatalf("GetKeys() error = %v", err)
	}
	if priv == nil {
		t.Fatal("expected private key, got nil")
	}
	if pub == nil {
		t.Fatal("expected public key, got nil")
	}

	if priv.N.BitLen() < 2048 {
		t.Errorf("expected >=2048-bit RSA key, got %d", priv.N.BitLen())
	}

	if _, err := os.Stat(provider.PrivateKeyPath); err != nil {
		t.Errorf("private key file not created: %v", err)
	}
	if _, err := os.Stat(provider.PublicKeyPath); err != nil {
		t.Errorf("public key file not created: %v", err)
	}

	// File permissions
	info, err := os.Stat(provider.PrivateKeyPath)
	if err == nil && info.Mode().Perm() != 0600 {
		t.Errorf("private key file permissions = %v, want 0600", info.Mode().Perm())
	}
}

func TestFileKeyProvider_LoadExisting(t *testing.T) {
	dir := t.TempDir()
	privPath := filepath.Join(dir, "private.pem")
	pubPath := filepath.Join(dir, "public.pem")

	// Pre-generate a key and write it
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	privBytes := x509.MarshalPKCS1PrivateKey(key)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes})
	if err := os.WriteFile(privPath, privPEM, 0600); err != nil {
		t.Fatalf("failed to write private key: %v", err)
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
	if err := os.WriteFile(pubPath, pubPEM, 0644); err != nil {
		t.Fatalf("failed to write public key: %v", err)
	}

	provider := &FileKeyProvider{
		PrivateKeyPath: privPath,
		PublicKeyPath:  pubPath,
		AutoGenerate:   false,
	}

	priv, pub, err := provider.GetKeys()
	if err != nil {
		t.Fatalf("GetKeys() error = %v", err)
	}

	if priv.N.Cmp(key.N) != 0 {
		t.Error("loaded private key modulus does not match original")
	}
	if pub.N.Cmp(key.N) != 0 {
		t.Error("loaded public key modulus does not match original")
	}
}

func TestFileKeyProvider_LoadPKCS8(t *testing.T) {
	dir := t.TempDir()
	privPath := filepath.Join(dir, "private.pem")
	pubPath := filepath.Join(dir, "public.pem")

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Write using PKCS8 format to verify fallback path
	privBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("failed to marshal PKCS8: %v", err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	if err := os.WriteFile(privPath, privPEM, 0600); err != nil {
		t.Fatalf("failed to write private key: %v", err)
	}

	provider := &FileKeyProvider{
		PrivateKeyPath: privPath,
		PublicKeyPath:  pubPath,
		AutoGenerate:   false,
	}

	priv, _, err := provider.GetKeys()
	if err != nil {
		t.Fatalf("GetKeys() error = %v", err)
	}
	if priv.N.Cmp(key.N) != 0 {
		t.Error("PKCS8 private key did not round-trip")
	}
}

func TestFileKeyProvider_NoAutoGenerate_MissingFile(t *testing.T) {
	dir := t.TempDir()
	provider := &FileKeyProvider{
		PrivateKeyPath: filepath.Join(dir, "nonexistent-private.pem"),
		PublicKeyPath:  filepath.Join(dir, "nonexistent-public.pem"),
		AutoGenerate:   false,
	}

	_, _, err := provider.GetKeys()
	if err == nil {
		t.Fatal("expected error when file missing and AutoGenerate=false, got nil")
	}
}

func TestFileKeyProvider_InvalidPEM(t *testing.T) {
	dir := t.TempDir()
	privPath := filepath.Join(dir, "private.pem")
	if err := os.WriteFile(privPath, []byte("not a real PEM file"), 0600); err != nil {
		t.Fatalf("failed to write bad pem: %v", err)
	}

	provider := &FileKeyProvider{
		PrivateKeyPath: privPath,
		PublicKeyPath:  filepath.Join(dir, "public.pem"),
		AutoGenerate:   false,
	}

	_, _, err := provider.GetKeys()
	if err == nil {
		t.Fatal("expected error on invalid PEM, got nil")
	}
}

func TestFileKeyProvider_AutoGenerateIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	provider := &FileKeyProvider{
		PrivateKeyPath: filepath.Join(dir, "private.pem"),
		PublicKeyPath:  filepath.Join(dir, "public.pem"),
		AutoGenerate:   true,
	}

	priv1, _, err := provider.GetKeys()
	if err != nil {
		t.Fatalf("first GetKeys() error = %v", err)
	}

	// Second call should load existing file, not regenerate
	priv2, _, err := provider.GetKeys()
	if err != nil {
		t.Fatalf("second GetKeys() error = %v", err)
	}

	if priv1.N.Cmp(priv2.N) != 0 {
		t.Error("second GetKeys() regenerated the key instead of loading existing file")
	}
}

func TestFileKeyProvider_GetPublicKey(t *testing.T) {
	dir := t.TempDir()
	pubPath := filepath.Join(dir, "public.pem")

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
	if err := os.WriteFile(pubPath, pubPEM, 0644); err != nil {
		t.Fatalf("failed to write public key: %v", err)
	}

	provider := &FileKeyProvider{PublicKeyPath: pubPath}
	pub, err := provider.GetPublicKey()
	if err != nil {
		t.Fatalf("GetPublicKey() error = %v", err)
	}
	if pub.N.Cmp(key.N) != 0 {
		t.Error("GetPublicKey returned mismatched modulus")
	}
}

func TestFileKeyProvider_GetPublicKey_MissingFile(t *testing.T) {
	provider := &FileKeyProvider{
		PublicKeyPath: filepath.Join(t.TempDir(), "missing.pem"),
	}
	_, err := provider.GetPublicKey()
	if err == nil {
		t.Fatal("expected error on missing file, got nil")
	}
}
