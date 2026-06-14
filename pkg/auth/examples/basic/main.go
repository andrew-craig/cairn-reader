// Package main demonstrates basic usage of the Cairn auth package
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/cairn-app/cairn-reader/pkg/auth"
)

func main() {
	// 1. Setup Vault client and fetch public key
	vaultClient, err := auth.NewVaultClient(&auth.VaultConfig{
		Address: "http://localhost:8200", // Vault address
		Token:   "dev-token",             // Development token (use AppRole in production)
	})
	if err != nil {
		log.Fatalf("Failed to create Vault client: %v", err)
	}

	// Check Vault health
	if err := vaultClient.Health(); err != nil {
		log.Fatalf("Vault health check failed: %v", err)
	}

	// Fetch JWT public key from Vault
	publicKey, err := vaultClient.GetPublicKey("secret/jwt/public-key")
	if err != nil {
		log.Fatalf("Failed to get public key: %v", err)
	}

	// 2. Create JWT validator and middleware
	validator := auth.NewValidator(publicKey)
	middleware := auth.NewMiddleware(validator)

	// 3. Setup HTTP routes
	mux := http.NewServeMux()

	// Protected endpoint - requires valid JWT
	mux.Handle("/api/protected", middleware.RequireAuth(http.HandlerFunc(handleProtected)))

	// Public endpoint with optional auth
	mux.Handle("/api/public", middleware.OptionalAuth(http.HandlerFunc(handlePublic)))

	// Health check - no auth required
	mux.HandleFunc("/health", handleHealth)

	// 4. Start server
	addr := ":8080"
	log.Printf("Server starting on %s", addr)
	log.Printf("Endpoints:")
	log.Printf("  - GET  /api/protected (requires JWT)")
	log.Printf("  - GET  /api/public    (optional JWT)")
	log.Printf("  - GET  /health        (public)")
	log.Fatal(http.ListenAndServe(addr, mux))
}

func handleProtected(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context (guaranteed to exist with RequireAuth)
	userID := auth.MustGetUserID(r.Context())

	response := map[string]interface{}{
		"message": "Successfully accessed protected endpoint",
		"user_id": userID.String(),
		"path":    r.URL.Path,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handlePublic(w http.ResponseWriter, r *http.Request) {
	var response map[string]interface{}

	// OptionalAuth has already rejected any presented-but-invalid token with
	// 401 before reaching here. So this request is either anonymous or
	// fully authenticated — never "tried to authenticate and failed".
	if auth.IsAuthenticated(r.Context()) {
		userID, _ := auth.GetUserIDFromContext(r.Context())
		response = map[string]interface{}{
			"message":       "Welcome, authenticated user",
			"user_id":       userID.String(),
			"authenticated": true,
		}
	} else {
		response = map[string]interface{}{
			"message":       "Welcome, anonymous user",
			"authenticated": false,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"healthy"}`)
}
