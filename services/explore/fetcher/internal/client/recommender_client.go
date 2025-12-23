package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/andrew-craig/cairn-explore/pkg/models"
)

// RecommenderClientInterface defines the interface for submitting articles
type RecommenderClientInterface interface {
	SubmitArticles(ctx context.Context, articles []models.Article) error
}

// RecommenderClient handles communication with the recommender service
type RecommenderClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewRecommenderClient creates a new recommender client
func NewRecommenderClient(baseURL string) *RecommenderClient {
	return &RecommenderClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SubmitArticles sends a batch of articles to the recommender service
func (c *RecommenderClient) SubmitArticles(ctx context.Context, articles []models.Article) error {
	if len(articles) == 0 {
		return nil
	}

	payload, err := json.Marshal(map[string]interface{}{
		"articles": articles,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal articles: %w", err)
	}

	url := fmt.Sprintf("%s/explore/articles", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
