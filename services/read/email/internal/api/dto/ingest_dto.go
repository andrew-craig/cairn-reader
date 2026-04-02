// Package dto provides data transfer objects for API requests and responses.
package dto

// IngestEmailRequest represents the request body for email ingestion from Cloudflare worker
type IngestEmailRequest struct {
	Recipient  string `json:"recipient" validate:"required,email"`
	Sender     string `json:"sender" validate:"required,email"`
	SenderName string `json:"sender_name"`
	Subject    string `json:"subject"`
	HTMLBody   string `json:"html_body"`
	TextBody   string `json:"text_body"`
	ReceivedAt string `json:"received_at" validate:"required"`
}

// IngestEmailResponse represents the response for email ingestion
type IngestEmailResponse struct {
	Accepted bool `json:"accepted"`
}
