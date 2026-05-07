package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/cairn-app/cairn-reader/pkg/api"
	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/api/dto"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// SubscriptionAggregatorHandler aggregates subscriptions from multiple sources
type SubscriptionAggregatorHandler struct {
	ingestRSSClient *service.IngestRSSClient
	// Future: add other subscription service clients
	// socialClient  *service.SocialClient
	// emailClient   *service.EmailClient
}

// NewSubscriptionAggregatorHandler creates a new SubscriptionAggregatorHandler
func NewSubscriptionAggregatorHandler(
	ingestRSSClient *service.IngestRSSClient,
) *SubscriptionAggregatorHandler {
	return &SubscriptionAggregatorHandler{
		ingestRSSClient: ingestRSSClient,
	}
}

// ListAllSubscriptions handles GET /api/v1/content/user/{user_id}/subscriptions
// Returns a unified list of all subscriptions from all sources (RSS, social, email, etc.)
func (h *SubscriptionAggregatorHandler) ListAllSubscriptions(w http.ResponseWriter, r *http.Request) {
	// Extract authenticated user ID from context
	authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		slog.Error("user ID not found in context", slog.Any("error", err))
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Authentication context error", nil, "v1")
		return
	}

	// Get user ID from URL
	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid user ID format", nil, "v1")
		return
	}

	// Verify user owns the subscriptions
	if authenticatedUserID != userID {
		api.WriteError(w, http.StatusForbidden, api.ErrCodeForbidden, "User can only access their own subscriptions", nil, "v1")
		return
	}

	// Aggregate subscriptions from all sources
	allSubscriptions := []dto.UnifiedSubscription{}

	// 1. Fetch RSS subscriptions
	rssSubscriptions, err := h.fetchRSSSubscriptions(r.Context(), userID.String())
	if err != nil {
		slog.Error("Failed to fetch RSS subscriptions", "error", err)
		// Don't fail the entire request - just log and continue
		// This allows partial results if one subscription source is down
	} else {
		allSubscriptions = append(allSubscriptions, rssSubscriptions...)
	}

	// 2. Future: Fetch social feed subscriptions
	// socialSubscriptions, err := h.fetchSocialSubscriptions(r.Context(), userID.String())
	// if err != nil {
	//     slog.Error("Failed to fetch social subscriptions", "error", err)
	// } else {
	//     allSubscriptions = append(allSubscriptions, socialSubscriptions...)
	// }

	// 3. Future: Fetch email subscriptions
	// emailSubscriptions, err := h.fetchEmailSubscriptions(r.Context(), userID.String())
	// if err != nil {
	//     slog.Error("Failed to fetch email subscriptions", "error", err)
	// } else {
	//     allSubscriptions = append(allSubscriptions, emailSubscriptions...)
	// }

	// Build response
	response := dto.ListSubscriptionsResponse{
		Subscriptions: allSubscriptions,
		TotalCount:    len(allSubscriptions),
	}

	api.WriteSuccess(w, http.StatusOK, response, "v1")
}

// UnsubscribeRSS handles DELETE /api/v1/content/user/{user_id}/subscriptions/rss/{feed_id}.
// It removes the user's RSS subscription via the Ingest RSS service. Already-delivered
// articles in the user's reading list are preserved; only future deliveries are stopped.
func (h *SubscriptionAggregatorHandler) UnsubscribeRSS(w http.ResponseWriter, r *http.Request) {
	authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		slog.Error("user ID not found in context", slog.Any("error", err))
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Authentication context error", nil, "v1")
		return
	}

	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid user ID format", nil, "v1")
		return
	}

	if authenticatedUserID != userID {
		api.WriteError(w, http.StatusForbidden, api.ErrCodeForbidden, "User can only modify their own subscriptions", nil, "v1")
		return
	}

	feedIDStr := chi.URLParam(r, "feed_id")
	if _, err := uuid.Parse(feedIDStr); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid feed ID format", nil, "v1")
		return
	}

	if err := h.ingestRSSClient.UnsubscribeUserFromFeed(r.Context(), userID.String(), feedIDStr); err != nil {
		if errors.Is(err, service.ErrSubscriptionNotFound) {
			api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "Subscription not found", nil, "v1")
			return
		}
		slog.Error("Failed to unsubscribe from RSS feed", "error", err)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to unsubscribe from feed", nil, "v1")
		return
	}

	api.WriteSuccess(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Successfully unsubscribed from feed",
	}, "v1")
}

// fetchRSSSubscriptions fetches and transforms RSS subscriptions from Ingest RSS service
func (h *SubscriptionAggregatorHandler) fetchRSSSubscriptions(ctx context.Context, userID string) ([]dto.UnifiedSubscription, error) {
	// Call Ingest RSS service
	rssResponse, err := h.ingestRSSClient.ListUserSubscriptions(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Transform to unified format
	unified := make([]dto.UnifiedSubscription, 0, len(rssResponse.Subscriptions))
	for _, rssSub := range rssResponse.Subscriptions {
		unified = append(unified, dto.UnifiedSubscription{
			ID:           rssSub.ID,
			Type:         dto.SubscriptionTypeRSS,
			Title:        rssSub.FeedTitle,
			SubscribedAt: rssSub.SubscribedAt,
			RSSData: &dto.RSSSubscriptionData{
				FeedID:        rssSub.FeedID,
				FeedURL:       rssSub.FeedURL,
				PollingTier:   rssSub.PollingTier,
				LastFetchedAt: rssSub.LastFetchedAt,
			},
		})
	}

	return unified, nil
}

// Future: Add methods for other subscription sources
//
// func (h *SubscriptionAggregatorHandler) fetchSocialSubscriptions(ctx context.Context, userID string) ([]dto.UnifiedSubscription, error) {
//     // Call social service
//     socialResponse, err := h.socialClient.ListUserSubscriptions(ctx, userID)
//     if err != nil {
//         return nil, err
//     }
//
//     // Transform to unified format
//     unified := make([]dto.UnifiedSubscription, 0, len(socialResponse.Subscriptions))
//     for _, socialSub := range socialResponse.Subscriptions {
//         unified = append(unified, dto.UnifiedSubscription{
//             ID:           socialSub.ID,
//             Type:         dto.SubscriptionTypeSocial,
//             Title:        socialSub.DisplayName,
//             SubscribedAt: socialSub.SubscribedAt,
//             SocialData: &dto.SocialSubscriptionData{
//                 Platform: socialSub.Platform,
//                 Handle:   socialSub.Handle,
//             },
//         })
//     }
//
//     return unified, nil
// }
//
// func (h *SubscriptionAggregatorHandler) fetchEmailSubscriptions(ctx context.Context, userID string) ([]dto.UnifiedSubscription, error) {
//     // Similar implementation for email subscriptions
//     return nil, nil
// }
