// Package api provides standard API response types and utilities for all Cairn services.
//
// This package defines standardized response formats that ensure consistency across
// all backend services, making API consumption easier for clients.
package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// Meta contains metadata included in all API responses
type Meta struct {
	Timestamp string `json:"timestamp"` // ISO 8601 timestamp
	Version   string `json:"version"`   // API version (e.g., "v1")
}

// NewMeta creates a Meta object with current timestamp and specified API version
func NewMeta(version string) Meta {
	return Meta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   version,
	}
}

// SuccessResponse represents a standard successful API response
type SuccessResponse struct {
	Data interface{} `json:"data"`
	Meta Meta        `json:"meta"`
}

// ErrorResponse represents a standard error API response
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
	Meta    Meta                   `json:"meta"`
}

// PaginationInfo contains pagination metadata
type PaginationInfo struct {
	Cursor  string `json:"cursor,omitempty"` // Base64-encoded cursor for next page
	HasMore bool   `json:"has_more"`         // Whether more results are available
	Limit   int    `json:"limit"`            // Number of items per page
	Offset  int    `json:"offset,omitempty"` // Offset for offset-based pagination
	Total   int    `json:"total,omitempty"`  // Total count (if available)
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Data       interface{}    `json:"data"`
	Pagination PaginationInfo `json:"pagination"`
	Meta       Meta           `json:"meta"`
}

// WriteSuccess writes a successful response with the given data
func WriteSuccess(w http.ResponseWriter, status int, data interface{}, version string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := SuccessResponse{
		Data: data,
		Meta: NewMeta(version),
	}

	json.NewEncoder(w).Encode(response)
}

// WriteError writes an error response with the given status and error details
func WriteError(w http.ResponseWriter, status int, errorCode, message string, details map[string]interface{}, version string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := ErrorResponse{
		Error:   errorCode,
		Message: message,
		Details: details,
		Meta:    NewMeta(version),
	}

	json.NewEncoder(w).Encode(response)
}

// WritePaginated writes a paginated response with the given data and pagination info
func WritePaginated(w http.ResponseWriter, status int, data interface{}, pagination PaginationInfo, version string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := PaginatedResponse{
		Data:       data,
		Pagination: pagination,
		Meta:       NewMeta(version),
	}

	json.NewEncoder(w).Encode(response)
}

// Common error codes
const (
	ErrCodeBadRequest         = "bad_request"
	ErrCodeUnauthorized       = "unauthorized"
	ErrCodeForbidden          = "forbidden"
	ErrCodeNotFound           = "not_found"
	ErrCodeConflict           = "conflict"
	ErrCodeValidation         = "validation_error"
	ErrCodeInternal           = "internal_error"
	ErrCodeMethodNotAllowed   = "method_not_allowed"
	ErrCodeTooManyRequests    = "too_many_requests"
	ErrCodeServiceUnavailable = "service_unavailable"
)
