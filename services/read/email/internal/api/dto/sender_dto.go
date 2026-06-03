// Package dto provides data transfer objects for API requests and responses.
package dto

// SenderResponse represents a sender in the API response
type SenderResponse struct {
	ID             string `json:"id"`
	SenderEmail    string `json:"sender_email"`
	SenderName     string `json:"sender_name,omitempty"`
	EmailCount     int    `json:"email_count"`
	CreatedAt      string `json:"created_at,omitempty"`
	LastReceivedAt string `json:"last_received_at,omitempty"`
}

// ListSendersResponse represents the response for listing senders
type ListSendersResponse struct {
	Data []SenderResponse `json:"data"`
}
