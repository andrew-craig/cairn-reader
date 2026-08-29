package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/andrew-craig/cairn-reader/pkg/auth"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// RequireSameUser ensures that the authenticated user can only access their own resources
// This middleware expects a URL parameter named "id" representing the user ID being accessed
// and checks that it matches the authenticated user's ID from the JWT token
func RequireSameUser() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the user ID from the JWT token (set by JWTAuth middleware)
			authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "authentication required",
				})
				return
			}

			// Get the user ID from the URL parameter
			requestedUserIDStr := chi.URLParam(r, "id")
			if requestedUserIDStr == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "user ID parameter is required",
				})
				return
			}

			// Parse the requested user ID
			requestedUserID, err := uuid.Parse(requestedUserIDStr)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "invalid user ID format",
				})
				return
			}

			// Check if the authenticated user matches the requested user
			if authenticatedUserID != requestedUserID {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "you can only access your own resources",
				})
				return
			}

			// User is authorized to access this resource
			next.ServeHTTP(w, r)
		})
	}
}

// RequireSameUserWithCustomParam is similar to RequireSameUser but allows specifying
// a custom parameter name instead of "id"
func RequireSameUserWithCustomParam(paramName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the user ID from the JWT token (set by JWTAuth middleware)
			authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "authentication required",
				})
				return
			}

			// Get the user ID from the URL parameter
			requestedUserIDStr := chi.URLParam(r, paramName)
			if requestedUserIDStr == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "user ID parameter is required",
				})
				return
			}

			// Parse the requested user ID
			requestedUserID, err := uuid.Parse(requestedUserIDStr)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "invalid user ID format",
				})
				return
			}

			// Check if the authenticated user matches the requested user
			if authenticatedUserID != requestedUserID {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "you can only access your own resources",
				})
				return
			}

			// User is authorized to access this resource
			next.ServeHTTP(w, r)
		})
	}
}

// OwnerCheckFunc is a function that returns the owner's user ID
type OwnerCheckFunc func(r *http.Request) (uuid.UUID, error)

// RequireOwnership is a flexible authorization middleware that takes a function
// to determine if the authenticated user owns the resource
// The ownerCheckFunc should return the owner's user ID and any error
func RequireOwnership(ownerCheckFunc OwnerCheckFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the user ID from the JWT token (set by JWTAuth middleware)
			authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "authentication required",
				})
				return
			}

			// Get the resource owner ID using the provided function
			resourceOwnerID, err := ownerCheckFunc(r)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": err.Error(),
				})
				return
			}

			// Check if the authenticated user owns the resource
			if authenticatedUserID != resourceOwnerID {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "you can only access your own resources",
				})
				return
			}

			// User is authorized to access this resource
			next.ServeHTTP(w, r)
		})
	}
}

// CheckUserIDParam validates that a user ID parameter exists and is valid
// This is a helper middleware that can be used before authorization checks
func CheckUserIDParam(paramName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userIDStr := chi.URLParam(r, paramName)
			if userIDStr == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "user ID parameter is required",
				})
				return
			}

			_, err := uuid.Parse(userIDStr)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "invalid user ID format",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
