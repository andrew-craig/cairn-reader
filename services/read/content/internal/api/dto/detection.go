package dto

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

// DetectURLRequest represents the request body for URL detection
type DetectURLRequest struct {
	URL string `json:"url"`
}

// Validate validates the DetectURLRequest
func (d DetectURLRequest) Validate() error {
	return validation.ValidateStruct(&d,
		validation.Field(&d.URL,
			validation.Required.Error("URL is required"),
			is.URL.Error("URL must be a valid URL"),
		),
	)
}

// DetectURLResponse represents the response for URL detection
type DetectURLResponse struct {
	URL   string  `json:"url"`
	Type  string  `json:"type"`
	Title *string `json:"title"`
}
