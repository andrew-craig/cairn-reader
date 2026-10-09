package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/andrew-craig/cairn-reader/pkg/api"
	"github.com/andrew-craig/cairn-reader/pkg/auth"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/api/dto"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/api/middleware"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/models"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/repository"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// SubscriptionAggregatorHandler aggregates subscriptions from multiple sources
type SubscriptionAggregatorHandler struct {
	ingestRSSClient   *service.IngestRSSClient
	emailIngestClient *service.EmailIngestClient
	routeRepo         repository.SourceRouteRepository
}

// NewSubscriptionAggregatorHandler creates a new SubscriptionAggregatorHandler
func NewSubscriptionAggregatorHandler(
	ingestRSSClient *service.IngestRSSClient,
	emailIngestClient *service.EmailIngestClient,
	routeRepo repository.SourceRouteRepository,
) *SubscriptionAggregatorHandler {
	return &SubscriptionAggregatorHandler{
		ingestRSSClient:   ingestRSSClient,
		emailIngestClient: emailIngestClient,
		routeRepo:         routeRepo,
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

	// 3. Fetch email newsletter subscriptions
	if h.emailIngestClient != nil {
		emailSubscriptions, err := h.fetchEmailSubscriptions(r.Context(), userID.String())
		if err != nil {
			slog.Error("Failed to fetch email subscriptions", "error", err)
		} else {
			allSubscriptions = append(allSubscriptions, emailSubscriptions...)
		}
	}

	// Stamp each subscription with its destination list; no route means Reads.
	routes, err := h.routeRepo.ListByUser(r.Context(), userID)
	if err != nil {
		slog.Error("Failed to fetch source routes", "error", err)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to fetch subscriptions", nil, "v1")
		return
	}
	routed := make(map[string]string, len(routes))
	for _, route := range routes {
		routed[route.SourceType+"/"+route.SourceKey.String()] = route.List
	}
	for i := range allSubscriptions {
		sub := &allSubscriptions[i]
		sub.List = models.ListReads
		if list, ok := routed[string(sub.Type)+"/"+sourceKeyOf(sub)]; ok {
			sub.List = list
		}
	}

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
	feedID, err := uuid.Parse(feedIDStr)
	if err != nil {
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

	// The route only has meaning while subscribed; drop it so a later
	// re-subscribe starts from the default rather than a stale choice.
	if err := h.routeRepo.Delete(r.Context(), userID, models.SourceTypeRSS, feedID); err != nil {
		slog.Error("Failed to delete route for unsubscribed feed", "error", err)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Unsubscribed from feed but failed to clear its list", nil, "v1")
		return
	}

	api.WriteSuccess(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Successfully unsubscribed from feed",
	}, "v1")
}

// SetSourceList handles PUT /api/v1/content/user/{user_id}/subscriptions/{type}/{key}/list.
// It changes where a source's future items are delivered (type is "rss" or "email"; key is the
// feed ID or sender ID). Only the route changes: items already delivered stay in their list.
func (h *SubscriptionAggregatorHandler) SetSourceList(w http.ResponseWriter, r *http.Request) {
	authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		slog.Error("user ID not found in context", slog.Any("error", err))
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Authentication context error", nil, "v1")
		return
	}

	userID, err := uuid.Parse(chi.URLParam(r, "user_id"))
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid user ID format", nil, "v1")
		return
	}

	if authenticatedUserID != userID {
		api.WriteError(w, http.StatusForbidden, api.ErrCodeForbidden, "User can only modify their own subscriptions", nil, "v1")
		return
	}

	sourceType := chi.URLParam(r, "type")
	if sourceType != models.SourceTypeRSS && sourceType != models.SourceTypeEmail {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid subscription type. Must be 'rss' or 'email'", nil, "v1")
		return
	}

	sourceKey, err := uuid.Parse(chi.URLParam(r, "key"))
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid subscription key format", nil, "v1")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxSimpleRequestSize)

	var req dto.SetSourceListRequest
	if err := middleware.DecodeJSONBody(r, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid request body", nil, "v1")
		return
	}
	if err := req.Validate(); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeValidation, err.Error(), nil, "v1")
		return
	}

	// Only a source the user is currently subscribed to can be routed; this
	// also stops a PUT after unsubscribe from resurrecting a deleted route.
	subscribed, err := h.isSubscribed(r.Context(), userID, sourceType, sourceKey)
	if err != nil {
		slog.Error("Failed to verify subscription", "error", err)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to verify subscription", nil, "v1")
		return
	}
	if !subscribed {
		api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "Subscription not found", nil, "v1")
		return
	}

	route := &models.SourceRoute{UserID: userID, SourceType: sourceType, SourceKey: sourceKey, List: req.List}
	if err := h.routeRepo.Upsert(r.Context(), route); err != nil {
		slog.Error("Failed to set source list", "error", err)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to update list", nil, "v1")
		return
	}

	api.WriteSuccess(w, http.StatusOK, map[string]any{
		"type": sourceType,
		"key":  sourceKey,
		"list": req.List,
	}, "v1")
}

// isSubscribed reports whether the user currently has the given RSS feed or email sender.
func (h *SubscriptionAggregatorHandler) isSubscribed(ctx context.Context, userID uuid.UUID, sourceType string, sourceKey uuid.UUID) (bool, error) {
	key := sourceKey.String()
	if sourceType == models.SourceTypeRSS {
		resp, err := h.ingestRSSClient.ListUserSubscriptions(ctx, userID.String())
		if err != nil {
			return false, err
		}
		for _, sub := range resp.Subscriptions {
			if sub.FeedID == key {
				return true, nil
			}
		}
		return false, nil
	}

	if h.emailIngestClient == nil {
		return false, nil
	}
	resp, err := h.emailIngestClient.ListUserSenders(ctx, userID.String())
	if err != nil {
		return false, err
	}
	for _, sender := range resp.Senders {
		if sender.ID == key {
			return true, nil
		}
	}
	return false, nil
}

// sourceKeyOf returns the source_routes key for a unified subscription: the feed ID for RSS,
// the sender ID for email.
func sourceKeyOf(sub *dto.UnifiedSubscription) string {
	if sub.Type == dto.SubscriptionTypeRSS && sub.RSSData != nil {
		return sub.RSSData.FeedID
	}
	return sub.ID
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

// fetchEmailSubscriptions fetches and transforms email senders from the Email Ingest service.
// Each unique sender represents a newsletter the user has subscribed to.
func (h *SubscriptionAggregatorHandler) fetchEmailSubscriptions(ctx context.Context, userID string) ([]dto.UnifiedSubscription, error) {
	sendersResponse, err := h.emailIngestClient.ListUserSenders(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sendersResponse == nil {
		return nil, errors.New("nil response from email ingest client")
	}

	unified := make([]dto.UnifiedSubscription, 0, len(sendersResponse.Senders))
	for _, sender := range sendersResponse.Senders {
		title := sender.SenderName
		if title == "" {
			title = sender.SenderEmail
		}
		unified = append(unified, dto.UnifiedSubscription{
			ID:           sender.ID,
			Type:         dto.SubscriptionTypeEmail,
			Title:        title,
			SubscribedAt: sender.CreatedAt,
			EmailData: &dto.EmailSubscriptionData{
				EmailAddress: sender.SenderEmail,
			},
		})
	}

	return unified, nil
}
