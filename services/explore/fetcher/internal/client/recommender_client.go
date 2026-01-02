// Package client provides HTTP clients for communicating with other services.
// Currently contains the recommender client for submitting articles to the
// recommender service.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/andrew-craig/cairn/pkg/models"
)

// RecommenderClientInterface defines the interface for submitting articles.
// This interface enables mocking in tests without requiring a real recommender service.
type RecommenderClientInterface interface {
	SubmitArticles(ctx context.Context, articles []models.Article) error
}

// RecommenderClient handles HTTP communication with the recommender service.
// It implements RecommenderClientInterface and submits articles via POST requests.
type RecommenderClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewRecommenderClient creates a new recommender client.
// The client has a 30-second timeout for HTTP requests to prevent
// hanging connections from blocking the fetcher.
func NewRecommenderClient(baseURL string) *RecommenderClient {
	return &RecommenderClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SubmitArticles sends a batch of articles to the recommender service.
// Articles are submitted as a JSON payload to POST /api/v1/explore/article.
// Returns nil if successful or if the articles slice is empty.
// Accepts both 200 OK and 201 Created as success responses.
func (c *RecommenderClient) SubmitArticles(ctx context.Context, articles []models.Article) error {
	if len(articles) == 0 {
		return nil
	}

	// Marshal articles into JSON payload
	payload, err := json.Marshal(map[string]interface{}{
		"articles": articles,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal articles: %w", err)
	}

	// Build and send HTTP request
	url := fmt.Sprintf("%s/api/v1/explore/article", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error closing response body: %v", err)
		}
	}()

	// Accept both 200 OK and 201 Created as success
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
