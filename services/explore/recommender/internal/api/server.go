// Package api provides the HTTP API layer for the recommender service.
// It handles routing, request/response handling, and authentication middleware.
package api

import (
	"log/slog"
	"net/http"

	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/pkg/logging"
	sharedmw "github.com/cairn-app/cairn-reader/pkg/middleware"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/db"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/recommend"
	"github.com/go-chi/chi/v5"
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

// Routes sets up the HTTP routes (v1 API) using chi router
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(sharedmw.Recovery)
	r.Use(logging.ChiRequestLogger(s.logger))
	// CORS is applied globally (before route/method resolution) so it covers health
	// checks and reliably answers browser preflight OPTIONS for Authorization requests.
	r.Use(sharedmw.CORS(sharedmw.DefaultCORSConfig()))
	r.Use(sharedmw.SecureHeadersRelaxed)

	// Health check endpoints (Kubernetes-compatible) - public
	// Support both GET and HEAD methods for health checks (HEAD used by Docker/wget --spider)
	r.Get("/health/live", s.handleLiveness)
	r.Head("/health/live", s.handleLiveness)
	r.Get("/health/ready", s.handleReadiness)
	r.Head("/health/ready", s.handleReadiness)

	// API v1 routes
	r.Route("/api/v1/explore", func(r chi.Router) {
		r.Use(sharedmw.RequireHTTPS)
		// Public API routes - no authentication required
		r.Post("/article", s.handleArticles)

		// Protected recommendation route - user identified by JWT
		r.With(s.authMiddleware.RequireAuth).Get("/recommendation", s.handleRecommendations)

		// Protected shown route - batched mark-as-shown from clients
		r.With(s.authMiddleware.RequireAuth).Post("/shown", s.handleMarkShown)

		// Protected user routes - require authentication, user identified by JWT
		r.Route("/user", func(r chi.Router) {
			r.Use(s.authMiddleware.RequireAuth)

			// Get all articles the authenticated user has voted on
			r.Get("/votes", s.handleGetUserVotedArticles)
		})

		// Protected article routes - require authentication
		r.Route("/article/{article_id}", func(r chi.Router) {
			r.Use(s.authMiddleware.RequireAuth)

			// Vote endpoints
			r.Get("/vote", s.handleGetVotes)
			r.Post("/vote", s.handleVote)
			r.Delete("/vote", s.handleRemoveVote)

			// Read endpoint
			r.Post("/read", s.handleMarkAsRead)
		})
	})

	return r
}
