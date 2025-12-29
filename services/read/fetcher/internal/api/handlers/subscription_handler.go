package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

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
		respondWithError(w, http.StatusBadRequest, "invalid_user_id", "Invalid user ID format")
		return
	}

	// Parse request body
	var req dto.SubscribeFeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// Validate feed URL
	if strings.TrimSpace(req.FeedURL) == "" {
		respondWithError(w, http.StatusBadRequest, "invalid_feed_url", "Feed URL is required")
		return
	}

	// Subscribe to feed
	result, err := h.feedService.Subscribe(ctx, userID, req.FeedURL)
	if err != nil {
		// Check for specific errors
		if strings.Contains(err.Error(), "maximum feed limit") {
			respondWithError(w, http.StatusBadRequest, "feed_limit_reached", err.Error())
			return
		}
		if strings.Contains(err.Error(), "already subscribed") {
			respondWithError(w, http.StatusConflict, "already_subscribed", err.Error())
			return
		}
		if strings.Contains(err.Error(), "failed to validate feed") || strings.Contains(err.Error(), "failed to parse feed") {
			respondWithError(w, http.StatusBadRequest, "invalid_feed", "Feed URL is not a valid RSS/Atom feed")
			return
		}

		log.Printf("Error subscribing to feed: %v", err)
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Failed to subscribe to feed")
		return
	}

	// Convert to response DTO
	response := dto.ToSubscribeFeedResponse(result)

	// Return response
	respondWithJSON(w, http.StatusCreated, response)
}

// Unsubscribe handles DELETE /api/v1/users/:user_id/feeds/:feed_id
func (h *SubscriptionHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse user ID from URL
	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid_user_id", "Invalid user ID format")
		return
	}

	// Parse feed ID from URL
	feedIDStr := chi.URLParam(r, "feed_id")
	feedID, err := uuid.Parse(feedIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid_feed_id", "Invalid feed ID format")
		return
	}

	// Unsubscribe from feed
	if err := h.feedService.Unsubscribe(ctx, userID, feedID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondWithError(w, http.StatusNotFound, "subscription_not_found", "Subscription not found")
			return
		}

		log.Printf("Error unsubscribing from feed: %v", err)
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Failed to unsubscribe from feed")
		return
	}

	// Return success response
	respondWithJSON(w, http.StatusOK, dto.MessageResponse{
		Message: "Successfully unsubscribed from feed",
		Success: true,
	})
}

// ListSubscriptions handles GET /api/v1/users/:user_id/feeds
func (h *SubscriptionHandler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse user ID from URL
	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid_user_id", "Invalid user ID format")
		return
	}

	// Get user subscriptions
	subscriptions, err := h.feedService.ListUserSubscriptions(ctx, userID)
	if err != nil {
		log.Printf("Error listing subscriptions: %v", err)
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Failed to list subscriptions")
		return
	}

	// Convert to response DTO
	response := dto.ToListFeedSubscriptionsResponse(subscriptions)

	// Return response
	respondWithJSON(w, http.StatusOK, response)
}

// EnableFeed handles PATCH /api/v1/feeds/:feed_id/enable
func (h *SubscriptionHandler) EnableFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse feed ID from URL
	feedIDStr := chi.URLParam(r, "feed_id")
	feedID, err := uuid.Parse(feedIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid_feed_id", "Invalid feed ID format")
		return
	}

	// Enable feed
	if err := h.feedService.EnableFeed(ctx, feedID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondWithError(w, http.StatusNotFound, "feed_not_found", "Feed not found")
			return
		}
		if strings.Contains(err.Error(), "already active") {
			respondWithError(w, http.StatusBadRequest, "already_active", "Feed is already active")
			return
		}

		log.Printf("Error enabling feed: %v", err)
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Failed to enable feed")
		return
	}

	// Return success response
	respondWithJSON(w, http.StatusOK, dto.MessageResponse{
		Message: "Feed successfully enabled",
		Success: true,
	})
}

// Helper functions

func respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func respondWithError(w http.ResponseWriter, statusCode int, errorCode, message string) {
	respondWithJSON(w, statusCode, dto.ErrorResponse{
		Error:   errorCode,
		Message: message,
		Code:    statusCode,
	})
}
