//go:build integration

package recommender_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/pkg/models"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/api"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/db"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/recommend"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/testutil"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// testInternalAPIKey is the shared secret these tests use to authenticate
// as the fetcher against the recommender's internal article-submission endpoint.
const testInternalAPIKey = "test-internal-key"

// postArticles submits articles to the recommender's internal endpoint,
// authenticating as the fetcher via X-Internal-API-Key.
func postArticles(t *testing.T, serverURL string, articles []models.Article) *http.Response {
	t.Helper()
	payload, err := json.Marshal(map[string][]models.Article{"articles": articles})
	if err != nil {
		t.Fatalf("Failed to marshal articles: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, serverURL+"/api/v1/explore/article", bytes.NewBuffer(payload))
	if err != nil {
		t.Fatalf("Failed to build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-API-Key", testInternalAPIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to submit articles: %v", err)
	}
	return resp
}

// Integration test configuration
type IntegrationTestSuite struct {
	database    *pgxpool.Pool
	cleanupDB   func()
	server      *httptest.Server
	articleRepo db.ArticleRepositoryInterface
	userRepo    db.UserRepositoryInterface
	voteRepo    db.VoteRepositoryInterface
	engine      *recommend.Engine
	jwtHelper   *testJWTHelper
}

func setupIntegrationTest(t *testing.T) *IntegrationTestSuite {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	database, cleanupDB := testutil.SetupTestDB(t)

	// Initialize repositories
	articleRepo := db.NewArticleRepository(database)
	userRepo := db.NewUserRepository(database)
	voteRepo := db.NewVoteRepository(database, userRepo)

	// Set user repo dependency on article repo
	articleRepo.SetUserRepository(userRepo)

	// Initialize recommendation engine
	engine := recommend.NewEngine(articleRepo, userRepo)

	// Create test JWT setup
	jwtHelper, validator, err := createTestJWTHelper(t)
	if err != nil {
		t.Fatalf("Failed to create test JWT helper: %v", err)
	}

	// Create auth middleware with test validator
	authMiddleware := auth.NewMiddleware(validator)
	internalAuthMiddleware := auth.NewInternalAuthMiddleware(testInternalAPIKey)

	// Create a test logger
	testLogger := slog.Default()

	// Setup HTTP server
	apiServer := api.NewServer(database, articleRepo, userRepo, voteRepo, engine, authMiddleware, internalAuthMiddleware, testLogger)
	server := httptest.NewServer(apiServer.Routes())

	return &IntegrationTestSuite{
		database:    database,
		cleanupDB:   cleanupDB,
		server:      server,
		articleRepo: articleRepo,
		userRepo:    userRepo,
		voteRepo:    voteRepo,
		engine:      engine,
		jwtHelper:   jwtHelper,
	}
}

func (suite *IntegrationTestSuite) teardown() {
	suite.server.Close()
	suite.cleanupDB()
}

// testJWTHelper helps generate JWT tokens for integration tests
type testJWTHelper struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

// createTestJWTHelper creates a test JWT helper with a freshly generated RSA keypair
func createTestJWTHelper(t *testing.T) (*testJWTHelper, *auth.Validator, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}
	publicKey := &privateKey.PublicKey

	// Create helper
	helper := &testJWTHelper{
		privateKey: privateKey,
		publicKey:  publicKey,
	}

	// Create validator
	validator := auth.NewValidator(publicKey)

	return helper, validator, nil
}

// GenerateToken generates a test JWT token for the given user ID
func (h *testJWTHelper) GenerateToken(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := &auth.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "cairn-user-service",
			Audience:  jwt.ClaimStrings{"cairn-api"},
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(h.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// makeAuthenticatedRequest creates an HTTP request with JWT authentication
func makeAuthenticatedRequest(method, url, userID string, body []byte, jwtHelper *testJWTHelper) (*http.Request, error) {
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, err
	}

	// Generate JWT token for the user
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	token, err := jwtHelper.GenerateToken(userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Add auth header
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

// Test 1 & 2: Article submission and deduplication
func TestArticleSubmissionAndDeduplication(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.teardown()

	ctx := context.Background()
	feedID := 1

	// Test article
	article := models.Article{
		ID:          "test-article-1",
		Title:       "Test Article",
		Link:        "https://example.com/test-article",
		Description: "Test description",
		Content:     "Test content",
		Author:      "Test Author",
		Published:   time.Now(),
		FeedID:      &feedID,
	}

	// Submit article via HTTP API (simulating fetcher)
	resp := postArticles(t, suite.server.URL, []models.Article{article})
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	// Verify article was stored
	stored, err := suite.articleRepo.GetByID(ctx, article.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve article: %v", err)
	}

	if stored.Title != article.Title {
		t.Errorf("Expected title %s, got %s", article.Title, stored.Title)
	}

	// Submit same article again (test deduplication)
	article.Title = "Updated Title"
	resp = postArticles(t, suite.server.URL, []models.Article{article})
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("error closing response body: %v", err)
		}
	}()

	// Verify title was updated
	stored, err = suite.articleRepo.GetByID(ctx, article.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated article: %v", err)
	}

	if stored.Title != "Updated Title" {
		t.Errorf("Expected updated title 'Updated Title', got %s", stored.Title)
	}

	t.Log("✓ Article submission and deduplication working correctly")
}

// Test 3: Recommendation algorithm (4 high-quality + 1 low-exposure)
func TestRecommendationAlgorithm(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.teardown()

	ctx := context.Background()
	userID := "test-user-1"

	// Create test articles with different quality scores
	articles := []models.Article{
		{
			ID:         "high-quality-1",
			Title:      "High Quality Article 1",
			Link:       "https://example.com/high-1",
			Content:    "Content",
			Published:  time.Now(),
			Upvotes:    10,
			Downvotes:  0,
			Recommends: 5,
		},
		{
			ID:         "high-quality-2",
			Title:      "High Quality Article 2",
			Link:       "https://example.com/high-2",
			Content:    "Content",
			Published:  time.Now(),
			Upvotes:    8,
			Downvotes:  1,
			Recommends: 3,
		},
		{
			ID:         "high-quality-3",
			Title:      "High Quality Article 3",
			Link:       "https://example.com/high-3",
			Content:    "Content",
			Published:  time.Now(),
			Upvotes:    15,
			Downvotes:  0,
			Recommends: 10,
		},
		{
			ID:         "high-quality-4",
			Title:      "High Quality Article 4",
			Link:       "https://example.com/high-4",
			Content:    "Content",
			Published:  time.Now(),
			Upvotes:    20,
			Downvotes:  2,
			Recommends: 8,
		},
		{
			ID:         "low-exposure-1",
			Title:      "Low Exposure Article",
			Link:       "https://example.com/low-1",
			Content:    "Content",
			Published:  time.Now(),
			Upvotes:    0,
			Downvotes:  0,
			Recommends: 0, // Never recommended
		},
		{
			ID:         "low-quality-1",
			Title:      "Low Quality Article",
			Link:       "https://example.com/low-quality-1",
			Content:    "Content",
			Published:  time.Now(),
			Upvotes:    2,
			Downvotes:  5,
			Recommends: 3,
		},
	}

	// Store articles
	for _, article := range articles {
		if err := suite.articleRepo.Create(ctx, article); err != nil {
			t.Fatalf("Failed to create article: %v", err)
		}
	}

	// Get recommendations
	recommendations, err := suite.engine.GetRecommendations(ctx, userID, 0)
	if err != nil {
		t.Fatalf("Failed to get recommendations: %v", err)
	}

	// Should return exactly 5 articles
	if len(recommendations) != 5 {
		t.Errorf("Expected 5 recommendations, got %d", len(recommendations))
	}

	// Verify low-exposure article is included
	hasLowExposure := false
	for _, rec := range recommendations {
		if rec.ID == "low-exposure-1" {
			hasLowExposure = true
			break
		}
	}

	if !hasLowExposure {
		t.Error("Expected low-exposure article to be included in recommendations")
	}

	t.Log("✓ Recommendation algorithm working correctly")
}

// Test 4 & 5: Upvoting and recommendation updates
func TestUpvotingFlow(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.teardown()

	ctx := context.Background()
	userID := "test-user-2"

	// Create test article
	article := models.Article{
		ID:         "test-upvote-article",
		Title:      "Test Upvote Article",
		Link:       "https://example.com/upvote-test",
		Content:    "Content",
		Published:  time.Now(),
		Upvotes:    0,
		Downvotes:  0,
		Recommends: 1,
	}

	if err := suite.articleRepo.Create(ctx, article); err != nil {
		t.Fatalf("Failed to create article: %v", err)
	}

	// Upvote via API (with authentication)
	votePayload := map[string]string{
		"vote_type": "upvote",
	}
	payload, _ := json.Marshal(votePayload)

	// Create authenticated request
	client := &http.Client{}
	req, err := makeAuthenticatedRequest("POST", suite.server.URL+"/api/v1/explore/article/"+article.ID+"/vote", userID, payload, suite.jwtHelper)
	if err != nil {
		t.Fatalf("Failed to create authenticated request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to submit vote: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify upvote was recorded
	upvotes, downvotes, err := suite.voteRepo.GetVoteCounts(ctx, article.ID)
	if err != nil {
		t.Fatalf("Failed to get vote counts: %v", err)
	}

	if upvotes != 1 {
		t.Errorf("Expected 1 upvote, got %d", upvotes)
	}

	if downvotes != 0 {
		t.Errorf("Expected 0 downvotes, got %d", downvotes)
	}

	// Verify article's upvote count was updated
	updated, err := suite.articleRepo.GetByID(ctx, article.ID)
	if err != nil {
		t.Fatalf("Failed to get updated article: %v", err)
	}

	if updated.Upvotes != 1 {
		t.Errorf("Expected article upvotes to be 1, got %d", updated.Upvotes)
	}

	t.Log("✓ Upvoting flow working correctly")
}

// Test 6 & 7: Downvoting and quality score impact
func TestDownvotingFlow(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.teardown()

	ctx := context.Background()
	userID := "test-user-3"

	// Create article to downvote
	badArticle := models.Article{
		ID:        "bad-article",
		Title:     "Bad Article",
		Link:      "https://example.com/bad",
		Content:   "Content",
		Published: time.Now(),
	}

	if err := suite.articleRepo.Create(ctx, badArticle); err != nil {
		t.Fatalf("Failed to create bad article: %v", err)
	}

	// Downvote the bad article (with authentication)
	votePayload := map[string]string{
		"vote_type": "downvote",
	}
	payload, _ := json.Marshal(votePayload)

	// Create authenticated request
	client := &http.Client{}
	req, err := makeAuthenticatedRequest("POST", suite.server.URL+"/api/v1/explore/article/"+badArticle.ID+"/vote", userID, payload, suite.jwtHelper)
	if err != nil {
		t.Fatalf("Failed to create authenticated request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to submit downvote: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify downvote was recorded
	upvotes, downvotes, err := suite.voteRepo.GetVoteCounts(ctx, badArticle.ID)
	if err != nil {
		t.Fatalf("Failed to get vote counts: %v", err)
	}

	if upvotes != 0 {
		t.Errorf("Expected 0 upvotes, got %d", upvotes)
	}

	if downvotes != 1 {
		t.Errorf("Expected 1 downvote, got %d", downvotes)
	}

	// Verify article's downvote count was updated
	updated, err := suite.articleRepo.GetByID(ctx, badArticle.ID)
	if err != nil {
		t.Fatalf("Failed to get updated article: %v", err)
	}

	if updated.Downvotes != 1 {
		t.Errorf("Expected article downvotes to be 1, got %d", updated.Downvotes)
	}

	t.Log("✓ Downvoting flow working correctly")
}

// Test 8: Deleted articles excluded from recommendations
func TestDeletedArticlesExcluded(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.teardown()

	ctx := context.Background()
	userID := "test-user-4"

	// Create active article
	activeArticle := models.Article{
		ID:        "active-article",
		Title:     "Active Article",
		Link:      "https://example.com/active",
		Content:   "Content",
		Published: time.Now(),
	}

	// Create another article that we'll mark as deleted
	deletedArticle := models.Article{
		ID:        "deleted-article",
		Title:     "Deleted Article",
		Link:      "https://example.com/deleted",
		Content:   "Content",
		Published: time.Now(),
	}

	if err := suite.articleRepo.Create(ctx, activeArticle); err != nil {
		t.Fatalf("Failed to create active article: %v", err)
	}
	if err := suite.articleRepo.Create(ctx, deletedArticle); err != nil {
		t.Fatalf("Failed to create deleted article: %v", err)
	}

	// Mark the second article as deleted using SQL
	_, err := suite.database.Exec(ctx, "UPDATE articles SET deleted = true WHERE id = $1", deletedArticle.ID)
	if err != nil {
		t.Fatalf("Failed to mark article as deleted: %v", err)
	}

	// Get recommendations
	recommendations, err := suite.engine.GetRecommendations(ctx, userID, 0)
	if err != nil {
		t.Fatalf("Failed to get recommendations: %v", err)
	}

	// Verify deleted article is not in recommendations
	for _, rec := range recommendations {
		if rec.ID == "deleted-article" {
			t.Error("Deleted article should not be in recommendations")
		}
	}

	t.Log("✓ Deleted articles correctly excluded from recommendations")
}

// Test: End-to-end flow
func TestEndToEndFlow(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.teardown()

	userID := "test-user-e2e"
	feedID := 1

	// 1. Fetcher submits articles
	articles := []models.Article{
		{
			ID:        "e2e-article-1",
			Title:     "E2E Article 1",
			Link:      "https://example.com/e2e-1",
			Content:   "Content 1",
			Published: time.Now(),
			FeedID:    &feedID,
		},
		{
			ID:        "e2e-article-2",
			Title:     "E2E Article 2",
			Link:      "https://example.com/e2e-2",
			Content:   "Content 2",
			Published: time.Now(),
			FeedID:    &feedID,
		},
		{
			ID:        "e2e-article-3",
			Title:     "E2E Article 3",
			Link:      "https://example.com/e2e-3",
			Content:   "Content 3",
			Published: time.Now(),
			FeedID:    &feedID,
		},
		{
			ID:        "e2e-article-4",
			Title:     "E2E Article 4",
			Link:      "https://example.com/e2e-4",
			Content:   "Content 4",
			Published: time.Now(),
			FeedID:    &feedID,
		},
		{
			ID:        "e2e-article-5",
			Title:     "E2E Article 5",
			Link:      "https://example.com/e2e-5",
			Content:   "Content 5",
			Published: time.Now(),
			FeedID:    &feedID,
		},
	}

	resp := postArticles(t, suite.server.URL, articles)
	if err := resp.Body.Close(); err != nil {
		t.Logf("error closing response body: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	// 2. User gets recommendations (with authentication)
	client := &http.Client{}
	req, err := makeAuthenticatedRequest("GET", suite.server.URL+"/api/v1/explore/recommendation", userID, nil, suite.jwtHelper)
	if err != nil {
		t.Fatalf("Failed to create authenticated request: %v", err)
	}

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to get recommendations: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("error closing response body: %v", err)
		}
	}()

	var recResp struct {
		UserID          string           `json:"user_id"`
		Recommendations []models.Article `json:"recommendations"`
		Count           int              `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&recResp); err != nil {
		t.Fatalf("Failed to decode recommendations: %v", err)
	}
	recommendations := recResp.Recommendations

	if len(recommendations) != 5 {
		t.Errorf("Expected 5 recommendations, got %d", len(recommendations))
	}

	// 3. User upvotes first article (with authentication)
	if len(recommendations) > 0 {
		votePayload := map[string]string{
			"vote_type": "upvote",
		}
		payload, _ := json.Marshal(votePayload)

		req, err := makeAuthenticatedRequest("POST", suite.server.URL+"/api/v1/explore/article/"+recommendations[0].ID+"/vote", userID, payload, suite.jwtHelper)
		if err != nil {
			t.Fatalf("Failed to create authenticated request: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to submit vote: %v", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("error closing response body: %v", err)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for vote, got %d", resp.StatusCode)
		}

		// 4. Verify vote was recorded
		ctx := context.Background()
		upvotes, _, err := suite.voteRepo.GetVoteCounts(ctx, recommendations[0].ID)
		if err != nil {
			t.Fatalf("Failed to get vote counts: %v", err)
		}

		if upvotes != 1 {
			t.Errorf("Expected 1 upvote, got %d", upvotes)
		}
	}

	t.Log("✓ End-to-end flow working correctly")
}
