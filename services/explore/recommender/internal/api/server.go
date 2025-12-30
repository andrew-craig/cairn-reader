// Package api provides the HTTP API layer for the recommender service.
// It handles routing, request/response handling, and authentication middleware.
package api

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/andrew-craig/cairn/pkg/auth"
	"github.com/andrew-craig/cairn/services/explore/recommender/internal/db"
	"github.com/andrew-craig/cairn/services/explore/recommender/internal/recommend"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Server holds the API server dependencies
type Server struct {
	db             *pgxpool.Pool
	articleRepo    db.ArticleRepositoryInterface
	userRepo       db.UserRepositoryInterface
	voteRepo       db.VoteRepositoryInterface
	engine         *recommend.Engine
	authMiddleware *auth.Middleware
	logger         *slog.Logger
}

// NewServer creates a new API server
func NewServer(database *pgxpool.Pool, articleRepo db.ArticleRepositoryInterface, userRepo db.UserRepositoryInterface, voteRepo db.VoteRepositoryInterface, engine *recommend.Engine, authMiddleware *auth.Middleware, logger *slog.Logger) *Server {
	return &Server{
		db:             database,
		articleRepo:    articleRepo,
		userRepo:       userRepo,
		voteRepo:       voteRepo,
		engine:         engine,
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

// Routes sets up the HTTP routes
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Health check endpoints - public
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)

	// Public API routes - no authentication required
	mux.HandleFunc("/explore/articles", s.handleArticles)

	// Protected API routes - require authentication
	mux.Handle("/explore/recommendations/", s.authMiddleware.RequireAuth(
		http.HandlerFunc(s.handleRecommendations),
	))
	mux.Handle("/explore/articles/read", s.authMiddleware.RequireAuth(
		http.HandlerFunc(s.handleMarkAsRead),
	))

	// Voting routes (Phase 3) - require authentication
	// Note: These routes need to be registered in a specific order for proper matching
	mux.Handle("/explore/articles/", s.authMiddleware.RequireAuth(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Route voting endpoints based on path patterns and HTTP method
			if strings.HasSuffix(r.URL.Path, "/vote") {
				switch r.Method {
				case http.MethodPost:
					s.handleVote(w, r)
				case http.MethodDelete:
					s.handleRemoveVote(w, r)
				default:
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
