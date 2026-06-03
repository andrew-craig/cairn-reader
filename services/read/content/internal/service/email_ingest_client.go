package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/auth"
)

// EmailSenderInfo represents a single sender from the email ingest service.
type EmailSenderInfo struct {
	ID             string `json:"id"`
	SenderEmail    string `json:"sender_email"`
	SenderName     string `json:"sender_name,omitempty"`
	EmailCount     int    `json:"email_count"`
	CreatedAt      string `json:"created_at,omitempty"`
	LastReceivedAt string `json:"last_received_at,omitempty"`
}

// ListSendersResponse represents the response from the internal senders endpoint.
type ListSendersResponse struct {
	Senders []EmailSenderInfo
}

// EmailIngestClient handles service-to-service communication with the Email Ingest service.
type EmailIngestClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// NewEmailIngestClient creates a new Email Ingest service client.
func NewEmailIngestClient(baseURL, internalAPIKey string) *EmailIngestClient {
	return &EmailIngestClient{
		baseURL: baseURL,
		apiKey:  internalAPIKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ListUserSenders fetches the email senders (newsletter subscriptions) for a user.
func (c *EmailIngestClient) ListUserSenders(ctx context.Context, userID string) (*ListSendersResponse, error) {
	url := fmt.Sprintf("%s/api/v1/internal/source/email/user/%s/senders", c.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set(auth.InternalAPIKeyHeader, c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call email ingest service: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("email ingest service error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResponse struct {
		Data []EmailSenderInfo `json:"data"`
	}
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &ListSendersResponse{Senders: apiResponse.Data}, nil
}
