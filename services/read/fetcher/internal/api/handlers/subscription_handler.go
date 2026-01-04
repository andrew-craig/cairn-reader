package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/andrew-craig/cairn/pkg/api"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/api/dto"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// SubscriptionHandler handles feed subscription-related HTTP requests
type SubscriptionHandler struct {
	feedService service.FeedService
}

// NewSubscriptionHandler creates a new SubscriptionHandler
func NewSubscriptionHandler(feedService service.FeedService) *SubscriptionHandler {
	return &SubscriptionHandler{
		feedService: feedService,
	}
}

// Subscribe handles POST /api/v1/users/:user_id/feeds/subscribe
func (h *SubscriptionHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse user ID from URL
	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid user ID format", nil, "v1")
		return
	}

	// Parse request body
	var req dto.SubscribeFeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid request body", nil, "v1")
		return
	}

	// Validate feed URL
	if strings.TrimSpace(req.FeedURL) == "" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeValidation, "Feed URL is required", nil, "v1")
		return
	}

	// Subscribe to feed
	result, err := h.feedService.Subscribe(ctx, userID, req.FeedURL)
	if err != nil {
		// Check for specific errors
		if strings.Contains(err.Error(), "maximum feed limit") {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, err.Error(), nil, "v1")
			return
		}
		if strings.Contains(err.Error(), "already subscribed") {
			api.WriteError(w, http.StatusConflict, api.ErrCodeConflict, err.Error(), nil, "v1")
			return
		}
		if strings.Contains(err.Error(), "failed to validate feed") || strings.Contains(err.Error(), "failed to parse feed") {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeValidation, "Feed URL is not a valid RSS/Atom feed", nil, "v1")
			return
		}

		log.Printf("Error subscribing to feed: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to subscribe to feed", nil, "v1")
		return
	}

	// Convert to response DTO
	response := dto.ToSubscribeFeedResponse(result)

	// Return response
	api.WriteSuccess(w, http.StatusCreated, response, "v1")
}

// Unsubscribe handles DELETE /api/v1/users/:user_id/feeds/:feed_id
func (h *SubscriptionHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse user ID from URL
	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid user ID format", nil, "v1")
		return
	}

	// Parse feed ID from URL
	feedIDStr := chi.URLParam(r, "feed_id")
	feedID, err := uuid.Parse(feedIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid feed ID format", nil, "v1")
		return
	}

	// Unsubscribe from feed
	if err := h.feedService.Unsubscribe(ctx, userID, feedID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "Subscription not found", nil, "v1")
			return
		}

		log.Printf("Error unsubscribing from feed: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to unsubscribe from feed", nil, "v1")
		return
	}

	// Return success response
	api.WriteSuccess(w, http.StatusOK, dto.MessageResponse{
		Message: "Successfully unsubscribed from feed",
		Success: true,
	}, "v1")
}

// ListSubscriptions handles GET /api/v1/users/:user_id/feeds
func (h *SubscriptionHandler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse user ID from URL
	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid user ID format", nil, "v1")
		return
	}

	// Get user subscriptions
	subscriptions, err := h.feedService.ListUserSubscriptions(ctx, userID)
	if err != nil {
		log.Printf("Error listing subscriptions: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to list subscriptions", nil, "v1")
		return
	}

	// Convert to response DTO
	response := dto.ToListFeedSubscriptionsResponse(subscriptions)

	// Return response
	api.WriteSuccess(w, http.StatusOK, response, "v1")
}

// EnableFeed handles PATCH /api/v1/feeds/:feed_id/enable
func (h *SubscriptionHandler) EnableFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse feed ID from URL
	feedIDStr := chi.URLParam(r, "feed_id")
	feedID, err := uuid.Parse(feedIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid feed ID format", nil, "v1")
		return
	}

	// Enable feed
	if err := h.feedService.EnableFeed(ctx, feedID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "Feed not found", nil, "v1")
			return
		}
		if strings.Contains(err.Error(), "already active") {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Feed is already active", nil, "v1")
			return
		}

		log.Printf("Error enabling feed: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to enable feed", nil, "v1")
		return
	}

	// Return success response
	api.WriteSuccess(w, http.StatusOK, dto.MessageResponse{
		Message: "Feed successfully enabled",
		Success: true,
	}, "v1")
}

// UpdateFeed handles PATCH /api/v1/source/rss/feed/{feed_id}
// Generic feed update endpoint that accepts a request body with fields to update
func (h *SubscriptionHandler) UpdateFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse feed ID from URL
	feedIDStr := chi.URLParam(r, "feed_id")
	feedID, err := uuid.Parse(feedIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid feed ID format", nil, "v1")
		return
	}

	// Parse request body
	var req struct {
		Enabled *bool `json:"enabled,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid request body", nil, "v1")
		return
	}

	// Handle enable/disable if provided
	if req.Enabled != nil {
		if *req.Enabled {
			// Enable feed
			if err := h.feedService.EnableFeed(ctx, feedID); err != nil {
				if strings.Contains(err.Error(), "not found") {
					api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "Feed not found", nil, "v1")
					return
				}
				if strings.Contains(err.Error(), "already active") {
					api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Feed is already active", nil, "v1")
					return
				}

				log.Printf("Error enabling feed: %v", err)
				api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to enable feed", nil, "v1")
				return
			}

			api.WriteSuccess(w, http.StatusOK, dto.MessageResponse{
				Message: "Feed successfully enabled",
				Success: true,
			}, "v1")
		} else {
			// For now, just return success for disabling (may need to implement DisableFeed in service)
			api.WriteSuccess(w, http.StatusOK, dto.MessageResponse{
				Message: "Feed update received",
				Success: true,
			}, "v1")
		}
		return
	}

	// No fields to update
	api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "No fields to update provided", nil, "v1")
}
