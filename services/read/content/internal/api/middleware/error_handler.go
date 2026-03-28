package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/cairn-app/cairn-reader/services/read/content/internal/api/dto"
)

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
