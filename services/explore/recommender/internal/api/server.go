// Package api provides the HTTP API layer for the recommender service.
// It handles routing, request/response handling, and authentication middleware.
package api

import (
	"log/slog"
	"net/http"
	"strings"

	pkgapi "github.com/cairn-app/cairn-reader/pkg/api"
	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/db"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/recommend"
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

// Routes sets up the HTTP routes (v1 API)
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Health check endpoints (Kubernetes-compatible) - public
	mux.HandleFunc("/health/live", s.handleLiveness)
	mux.HandleFunc("/health/ready", s.handleReadiness)

	// Public API routes - no authentication required
	mux.HandleFunc("/api/v1/explore/article", s.handleArticles)

	// Protected API routes - require authentication
	mux.Handle("/api/v1/explore/recommendation/", s.authMiddleware.RequireAuth(
		http.HandlerFunc(s.handleRecommendations),
	))

	// Voting routes - require authentication
	// Note: These routes need to be registered in a specific order for proper matching
	mux.Handle("/api/v1/explore/article/", s.authMiddleware.RequireAuth(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Route voting and read endpoints based on path patterns and HTTP method
			if strings.HasSuffix(r.URL.Path, "/vote") {
				switch r.Method {
				case http.MethodPost:
					s.handleVote(w, r)
				case http.MethodDelete:
					s.handleRemoveVote(w, r)
				case http.MethodGet:
					s.handleGetVotes(w, r)
				default:
					pkgapi.WriteError(w, http.StatusMethodNotAllowed, pkgapi.ErrCodeMethodNotAllowed, "Method not allowed", nil, "v1")
				}
			} else if strings.HasSuffix(r.URL.Path, "/read") {
				if r.Method == http.MethodPost {
					s.handleMarkAsRead(w, r)
				} else {
					pkgapi.WriteError(w, http.StatusMethodNotAllowed, pkgapi.ErrCodeMethodNotAllowed, "Method not allowed", nil, "v1")
				}
			} else {
				pkgapi.WriteError(w, http.StatusNotFound, pkgapi.ErrCodeNotFound, "Endpoint not found", nil, "v1")
			}
		}),
	))

	return s.loggingMiddleware(mux)
}
