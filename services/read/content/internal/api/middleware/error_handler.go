package middleware

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/andrew-craig/cairn/services/read/content/internal/api/dto"
)

// Recovery middleware recovers from panics and returns a 500 error
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC: %v", err)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)

				response := dto.ErrorResponse{
					Error:   "internal_server_error",
					Message: "An internal server error occurred",
				}
				json.NewEncoder(w).Encode(response)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// WriteError writes a JSON error response
func WriteError(w http.ResponseWriter, statusCode int, errorType string, message string, details map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := dto.ErrorResponse{
		Error:   errorType,
		Message: message,
		Details: details,
	}
	json.NewEncoder(w).Encode(response)
}

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
