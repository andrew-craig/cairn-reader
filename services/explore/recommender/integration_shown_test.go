//go:build integration

package recommender_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sort"
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

// shownSuite is a self-contained harness for Phase B shown-tracking tests.
// It spins up a real PostgreSQL via testcontainers, mounts the full HTTP
// API behind a generated RSA keypair, and exposes helpers for the four
// scenarios required by task_b054.
type shownSuite struct {
	t          *testing.T
	pool       *pgxpool.Pool
	server     *httptest.Server
	articles   db.ArticleRepositoryInterface
	users      db.UserRepositoryInterface
	privateKey *rsa.PrivateKey
	cleanup    func()
}

func setupShownSuite(t *testing.T) *shownSuite {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool, cleanupDB := testutil.SetupTestDB(t)

	articleRepo := db.NewArticleRepository(pool)
	userRepo := db.NewUserRepository(pool)
	voteRepo := db.NewVoteRepository(pool, userRepo)
	articleRepo.SetUserRepository(userRepo)
	engine := recommend.NewEngine(articleRepo, userRepo)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		cleanupDB()
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	validator := auth.NewValidator(&privateKey.PublicKey)
	middleware := auth.NewMiddleware(validator)

	apiServer := api.NewServer(pool, articleRepo, userRepo, voteRepo, engine, middleware, slog.Default())
	server := httptest.NewServer(apiServer.Routes())

	cleanup := func() {
		server.Close()
		cleanupDB()
	}

	return &shownSuite{
		t:          t,
		pool:       pool,
		server:     server,
		articles:   articleRepo,
		users:      userRepo,
		privateKey: privateKey,
		cleanup:    cleanup,
	}
}

func (s *shownSuite) token(userID uuid.UUID) string {
	s.t.Helper()
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
	signed, err := token.SignedString(s.privateKey)
	if err != nil {
		s.t.Fatalf("failed to sign token: %v", err)
	}
	return signed
}

func (s *shownSuite) seedArticles(count int) []string {
	s.t.Helper()
	ctx := context.Background()
	ids := make([]string, 0, count)
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("shown-test-article-%d-%d", time.Now().UnixNano(), i)
		article := models.Article{
			ID:        id,
			Title:     fmt.Sprintf("Article %d", i),
			Link:      fmt.Sprintf("https://example.com/shown-test/%s", id),
			Content:   "body",
			Published: time.Now().Add(-time.Duration(i) * time.Minute),
		}
		if err := s.articles.Create(ctx, article); err != nil {
			s.t.Fatalf("failed to seed article %d: %v", i, err)
		}
		ids = append(ids, id)
	}
	return ids
}

func (s *shownSuite) getRecommendations(userID uuid.UUID) []models.Article {
	s.t.Helper()
	req, err := http.NewRequest(http.MethodGet, s.server.URL+"/api/v1/explore/recommendation", nil)
	if err != nil {
		s.t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.token(userID))
	req.Header.Set("X-Forwarded-Proto", "https")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.t.Fatalf("recommendation request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		s.t.Fatalf("recommendation returned %d: %s", resp.StatusCode, string(body))
	}

	var envelope struct {
		Data struct {
			Recommendations []models.Article `json:"recommendations"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		s.t.Fatalf("failed to decode recommendation response: %v", err)
	}
	return envelope.Data.Recommendations
}

type shownResponse struct {
	Recorded   int    `json:"recorded"`
	StatusCode int    `json:"-"`
	Status     string `json:"status"`
}

func (s *shownSuite) postShown(userID uuid.UUID, ids []string) shownResponse {
	s.t.Helper()
	body, err := json.Marshal(map[string][]string{"article_ids": ids})
	if err != nil {
		s.t.Fatalf("failed to marshal shown payload: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, s.server.URL+"/api/v1/explore/shown", bytes.NewReader(body))
	if err != nil {
		s.t.Fatalf("failed to build shown request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.token(userID))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-Proto", "https")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.t.Fatalf("shown request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var envelope struct {
		Data shownResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		s.t.Fatalf("failed to decode shown response: %v", err)
	}
	envelope.Data.StatusCode = resp.StatusCode
	return envelope.Data
}

func (s *shownSuite) recommendsCount(articleID string) int {
	s.t.Helper()
	article, err := s.articles.GetByID(context.Background(), articleID)
	if err != nil {
		s.t.Fatalf("failed to fetch article %s: %v", articleID, err)
	}
	return article.Recommends
}

func collectIDs(articles []models.Article) []string {
	ids := make([]string, 0, len(articles))
	for _, a := range articles {
		ids = append(ids, a.ID)
	}
	sort.Strings(ids)
	return ids
}

func intersect(a, b []string) []string {
	set := make(map[string]struct{}, len(a))
	for _, id := range a {
		set[id] = struct{}{}
	}
	out := make([]string, 0)
	for _, id := range b {
		if _, ok := set[id]; ok {
			out = append(out, id)
		}
	}
	return out
}

// TestShown_TwoConsecutiveRecommendationsOverlap proves that
// GetRecommendations no longer writes to recommendations — two consecutive
// reads with no shown event between them must return overlapping IDs.
func TestShown_TwoConsecutiveRecommendationsOverlap(t *testing.T) {
	suite := setupShownSuite(t)
	defer suite.cleanup()

	userID := uuid.New()
	suite.seedArticles(10)

	first := collectIDs(suite.getRecommendations(userID))
	second := collectIDs(suite.getRecommendations(userID))

	if len(first) == 0 || len(second) == 0 {
		t.Fatalf("expected non-empty recommendations, got first=%d second=%d", len(first), len(second))
	}

	if shared := intersect(first, second); len(shared) == 0 {
		t.Errorf("expected overlap between consecutive recommendation calls, got none.\nfirst=%v\nsecond=%v", first, second)
	}
}

// TestShown_MarkedIDsExcludedFromNextRecommendation proves that POST /shown
// is what filters articles out of subsequent recommendations.
func TestShown_MarkedIDsExcludedFromNextRecommendation(t *testing.T) {
	suite := setupShownSuite(t)
	defer suite.cleanup()

	userID := uuid.New()
	suite.seedArticles(10)

	first := suite.getRecommendations(userID)
	if len(first) < 2 {
		t.Fatalf("expected at least 2 recommendations, got %d", len(first))
	}

	// Mark a strict subset as shown.
	shownIDs := []string{first[0].ID, first[1].ID}
	resp := suite.postShown(userID, shownIDs)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /shown, got %d", resp.StatusCode)
	}

	second := collectIDs(suite.getRecommendations(userID))
	for _, id := range shownIDs {
		for _, gotID := range second {
			if id == gotID {
				t.Errorf("article %s was marked shown but still appeared in next recommendation", id)
			}
		}
	}
}

// TestShown_IncrementsRecommendsIdempotently covers (a) that POST /shown
// increments articles.recommends by exactly 1 per fresh (user, article)
// pair, and (b) that a repeated POST for the same pair does not
// double-increment.
func TestShown_IncrementsRecommendsIdempotently(t *testing.T) {
	suite := setupShownSuite(t)
	defer suite.cleanup()

	userID := uuid.New()
	ids := suite.seedArticles(3)

	before := make(map[string]int, len(ids))
	for _, id := range ids {
		before[id] = suite.recommendsCount(id)
	}

	first := suite.postShown(userID, ids)
	if first.StatusCode != http.StatusOK {
		t.Fatalf("first /shown call returned %d", first.StatusCode)
	}
	if first.Recorded != len(ids) {
		t.Errorf("expected recorded=%d on first call, got %d", len(ids), first.Recorded)
	}

	for _, id := range ids {
		if got := suite.recommendsCount(id); got != before[id]+1 {
			t.Errorf("article %s: expected recommends=%d after first shown, got %d", id, before[id]+1, got)
		}
	}

	second := suite.postShown(userID, ids)
	if second.StatusCode != http.StatusOK {
		t.Fatalf("second /shown call returned %d", second.StatusCode)
	}
	if second.Recorded != 0 {
		t.Errorf("expected recorded=0 on duplicate call, got %d", second.Recorded)
	}

	for _, id := range ids {
		if got := suite.recommendsCount(id); got != before[id]+1 {
			t.Errorf("article %s: recommends should stay at %d after duplicate shown, got %d", id, before[id]+1, got)
		}
	}
}

// TestShown_MixedKnownAndUnknownIDs verifies that unknown article IDs are
// skipped (logged warn, no FK explosion) and the response still returns
// 200 with the recorded count reflecting only valid IDs.
func TestShown_MixedKnownAndUnknownIDs(t *testing.T) {
	suite := setupShownSuite(t)
	defer suite.cleanup()

	userID := uuid.New()
	known := suite.seedArticles(2)
	unknown := []string{"does-not-exist-1", "does-not-exist-2"}

	payload := append([]string{}, known...)
	payload = append(payload, unknown...)

	resp := suite.postShown(userID, payload)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Recorded != len(known) {
		t.Errorf("expected recorded=%d (valid only), got %d", len(known), resp.Recorded)
	}

	for _, id := range known {
		if got := suite.recommendsCount(id); got != 1 {
			t.Errorf("article %s: expected recommends=1, got %d", id, got)
		}
	}
}
