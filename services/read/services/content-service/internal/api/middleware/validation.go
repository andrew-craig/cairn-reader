package middleware

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/andrew-craig/cairn-read/services/content-service/internal/models"
)

// ValidateJSON validates that the request body contains valid JSON
func ValidateJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only validate POST, PUT, PATCH requests
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			contentType := r.Header.Get("Content-Type")
			if !strings.HasPrefix(contentType, "application/json") {
				WriteError(w, http.StatusBadRequest, "invalid_content_type", "Content-Type must be application/json", nil)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// DecodeJSONBody decodes a JSON request body into the provided struct
func DecodeJSONBody(r *http.Request, dst interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(dst)
	if err != nil {
		if err == io.EOF {
			return err
		}
		return err
	}

	return nil
}

// ValidateStatus validates that a status string is valid
func ValidateStatus(status string) bool {
	return models.ValidateStatus(status)
}

// ValidateSourceType validates that a source type string is valid
func ValidateSourceType(sourceType string) bool {
	return sourceType == models.SourceTypeRSS || sourceType == models.SourceTypeWeb
}
