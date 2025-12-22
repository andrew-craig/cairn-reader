package api

import (
	"net/http"
	"strings"

	"github.com/andrew-craig/cairn-core/user-service/pkg/auth"
	"github.com/andrew-craig/cairn-explore/recommender/internal/db"
	"github.com/andrew-craig/cairn-explore/recommender/internal/recommend"
)

// Server holds the API server dependencies
type Server struct {
	articleRepo    *db.ArticleRepository
	userRepo       *db.UserRepository
	voteRepo       *db.VoteRepository
	engine         *recommend.Engine
	authMiddleware *auth.Middleware
}

// NewServer creates a new API server
func NewServer(articleRepo *db.ArticleRepository, userRepo *db.UserRepository, voteRepo *db.VoteRepository, engine *recommend.Engine, authMiddleware *auth.Middleware) *Server {
	return &Server{
		articleRepo:    articleRepo,
		userRepo:       userRepo,
		voteRepo:       voteRepo,
		engine:         engine,
		authMiddleware: authMiddleware,
	}
}

// Routes sets up the HTTP routes
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Health check - public endpoint
	mux.HandleFunc("/health", s.handleHealth)

	// Public API routes - no authentication required
	mux.HandleFunc("/api/v1/articles", s.handleArticles)

	// Protected API routes - require authentication
	mux.Handle("/api/v1/recommendations/", s.authMiddleware.RequireAuth(
		http.HandlerFunc(s.handleRecommendations),
	))
	mux.Handle("/api/v1/articles/read", s.authMiddleware.RequireAuth(
		http.HandlerFunc(s.handleMarkAsRead),
	))

	// Voting routes (Phase 3) - require authentication
	// Note: These routes need to be registered in a specific order for proper matching
	mux.Handle("/api/v1/articles/", s.authMiddleware.RequireAuth(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Route voting endpoints based on path patterns and HTTP method
			if strings.HasSuffix(r.URL.Path, "/vote") {
				if r.Method == http.MethodPost {
					s.handleVote(w, r)
				} else if r.Method == http.MethodDelete {
					s.handleRemoveVote(w, r)
				} else {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				}
			} else if strings.HasSuffix(r.URL.Path, "/votes") {
				s.handleGetVotes(w, r)
			} else {
				http.NotFound(w, r)
			}
		}),
	))

	return s.loggingMiddleware(mux)
}
