//go:build integration
// +build integration

package repository

import (
	"context"
	"testing"

	"github.com/andrew-craig/cairn-reader/services/read/content/internal/models"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type routingFixture struct {
	contentRepo ContentRepository
	ucRepo      UserContentRepository
	routeRepo   SourceRouteRepository
}

func newRoutingFixture(t *testing.T) routingFixture {
	t.Helper()
	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)
	return routingFixture{
		contentRepo: NewContentRepository(testDB.DB),
		ucRepo:      NewUserContentRepository(testDB.DB),
		routeRepo:   NewSourceRouteRepository(testDB.DB),
	}
}

func (f routingFixture) rssContent(t *testing.T, feedID uuid.UUID) *models.Content {
	t.Helper()
	c := &models.Content{
		ContentHash:  uuid.NewString(),
		CleanedHTML:  "<p>x</p>",
		OriginalURL:  "https://example.com/" + uuid.NewString(),
		Title:        "RSS item",
		SourceType:   models.SourceTypeRSS,
		SourceFeedID: &feedID,
	}
	require.NoError(t, f.contentRepo.Create(context.Background(), c))
	return c
}

func (f routingFixture) emailContent(t *testing.T, senderID uuid.UUID) *models.Content {
	t.Helper()
	c := &models.Content{
		ContentHash:    uuid.NewString(),
		CleanedHTML:    "<p>x</p>",
		OriginalURL:    "email://" + uuid.NewString(),
		Title:          "Newsletter",
		SourceType:     models.SourceTypeEmail,
		SourceSenderID: &senderID,
	}
	require.NoError(t, f.contentRepo.Create(context.Background(), c))
	return c
}

// deliver links content to a user the way ingest does and returns the stored list.
func (f routingFixture) deliver(t *testing.T, userID uuid.UUID, c *models.Content) string {
	t.Helper()
	uc := &models.UserContent{UserID: userID, ContentID: c.ID, Status: models.StatusUnread}
	require.NoError(t, f.ucRepo.BulkCreate(context.Background(), []*models.UserContent{uc}))
	stored, err := f.ucRepo.GetByUserAndContent(context.Background(), userID, c.ID)
	require.NoError(t, err)
	require.NotNil(t, stored)
	require.Equal(t, stored.List, uc.List, "BulkCreate must report the list it stored")
	return stored.List
}

func (f routingFixture) route(t *testing.T, userID uuid.UUID, sourceType string, key uuid.UUID, list string) {
	t.Helper()
	require.NoError(t, f.routeRepo.Upsert(context.Background(),
		&models.SourceRoute{UserID: userID, SourceType: sourceType, SourceKey: key, List: list}))
}

func TestUserContentRepository_BulkCreate_ResolvesListFromRoute_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	f := newRoutingFixture(t)
	userID := uuid.New()
	feedToFeed, feedToReads, feedUnrouted := uuid.New(), uuid.New(), uuid.New()
	senderToFeed := uuid.New()

	f.route(t, userID, models.SourceTypeRSS, feedToFeed, models.ListFeed)
	f.route(t, userID, models.SourceTypeRSS, feedToReads, models.ListReads)
	f.route(t, userID, models.SourceTypeEmail, senderToFeed, models.ListFeed)

	require.Equal(t, models.ListFeed, f.deliver(t, userID, f.rssContent(t, feedToFeed)), "RSS routed to feed")
	require.Equal(t, models.ListReads, f.deliver(t, userID, f.rssContent(t, feedToReads)), "RSS routed to reads")
	require.Equal(t, models.ListReads, f.deliver(t, userID, f.rssContent(t, feedUnrouted)), "RSS with no route")
	require.Equal(t, models.ListFeed, f.deliver(t, userID, f.emailContent(t, senderToFeed)), "email sender routed to feed")
	require.Equal(t, models.ListReads, f.deliver(t, userID, f.emailContent(t, uuid.New())), "email sender with no route")
}

func TestUserContentRepository_BulkCreate_RoutesAreIsolatedPerUserAndSourceType_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	f := newRoutingFixture(t)
	routedUser, otherUser := uuid.New(), uuid.New()
	key := uuid.New()

	// A feed route and an email route can share a key without colliding.
	f.route(t, routedUser, models.SourceTypeRSS, key, models.ListFeed)
	f.route(t, routedUser, models.SourceTypeEmail, key, models.ListReads)

	rss := f.rssContent(t, key)
	email := f.emailContent(t, key)
	require.Equal(t, models.ListFeed, f.deliver(t, routedUser, rss))
	require.Equal(t, models.ListReads, f.deliver(t, routedUser, email))

	// The same shared content, delivered to a user with no route, lands in reads.
	require.Equal(t, models.ListReads, f.deliver(t, otherUser, rss), "another user's route must not apply")
}

func TestUserContentRepository_RouteChangeAffectsOnlyNewItems_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	f := newRoutingFixture(t)
	ctx := context.Background()
	userID, feedID := uuid.New(), uuid.New()

	before := f.rssContent(t, feedID)
	require.Equal(t, models.ListReads, f.deliver(t, userID, before))

	f.route(t, userID, models.SourceTypeRSS, feedID, models.ListFeed)
	after := f.rssContent(t, feedID)
	require.Equal(t, models.ListFeed, f.deliver(t, userID, after))

	old, err := f.ucRepo.GetByUserAndContent(ctx, userID, before.ID)
	require.NoError(t, err)
	require.Equal(t, models.ListReads, old.List, "existing item must stay where it was")
}

func TestUserContentRepository_ListFilter_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	f := newRoutingFixture(t)
	ctx := context.Background()
	userID, feedID := uuid.New(), uuid.New()

	f.route(t, userID, models.SourceTypeRSS, feedID, models.ListFeed)
	feedItem := f.rssContent(t, feedID)
	feedItem2 := f.rssContent(t, feedID)
	f.deliver(t, userID, feedItem)
	f.deliver(t, userID, feedItem2)
	readsItem := f.emailContent(t, uuid.New())
	f.deliver(t, userID, readsItem)

	feed, reads := models.ListFeed, models.ListReads

	got, err := f.ucRepo.ListByUserWithCursor(ctx, userID, nil, nil, &feed, 10, nil, nil)
	require.NoError(t, err)
	require.Len(t, got, 2)
	for _, uc := range got {
		require.Equal(t, models.ListFeed, uc.List)
	}

	got, err = f.ucRepo.ListByUserWithCursor(ctx, userID, nil, nil, &reads, 10, nil, nil)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, readsItem.ID, got[0].ContentID)

	got, err = f.ucRepo.ListByUserWithCursor(ctx, userID, nil, nil, nil, 10, nil, nil)
	require.NoError(t, err)
	require.Len(t, got, 3, "no list filter returns both lists")

	n, err := f.ucRepo.CountByUser(ctx, userID, nil, nil, &feed)
	require.NoError(t, err)
	require.Equal(t, 2, n)
	n, err = f.ucRepo.CountByUser(ctx, userID, nil, nil, &reads)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	n, err = f.ucRepo.CountByUser(ctx, userID, nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, 3, n)

	found, err := f.ucRepo.SearchWithCursor(ctx, userID, "Newsletter", &reads, 10, nil, nil)
	require.NoError(t, err)
	require.Len(t, found, 1)
	found, err = f.ucRepo.SearchWithCursor(ctx, userID, "Newsletter", &feed, 10, nil, nil)
	require.NoError(t, err)
	require.Empty(t, found, "search must respect the list filter")
	found, err = f.ucRepo.SearchWithCursor(ctx, userID, "Newsletter", nil, 10, nil, nil)
	require.NoError(t, err)
	require.Len(t, found, 1)
}

func TestUserContentRepository_UpdateMetadata_MovesList_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	f := newRoutingFixture(t)
	ctx := context.Background()
	userID, feedID := uuid.New(), uuid.New()

	f.route(t, userID, models.SourceTypeRSS, feedID, models.ListFeed)
	c := f.rssContent(t, feedID)
	require.Equal(t, models.ListFeed, f.deliver(t, userID, c))

	uc, err := f.ucRepo.GetByUserAndContent(ctx, userID, c.ID)
	require.NoError(t, err)

	reads := models.ListReads
	require.NoError(t, f.ucRepo.UpdateMetadata(ctx, uc.ID, nil, nil, nil, &reads))

	moved, err := f.ucRepo.GetByID(ctx, uc.ID)
	require.NoError(t, err)
	require.Equal(t, models.ListReads, moved.List)
	require.Equal(t, models.StatusUnread, moved.Status, "moving lists must not touch other fields")
}

func TestUserContentRepository_UpdateMetadata_MoveToReadsLandsOnTop_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	f := newRoutingFixture(t)
	ctx := context.Background()
	userID, feedID, senderID := uuid.New(), uuid.New(), uuid.New()

	f.route(t, userID, models.SourceTypeRSS, feedID, models.ListFeed)
	feedItem := f.rssContent(t, feedID)
	require.Equal(t, models.ListFeed, f.deliver(t, userID, feedItem))
	// Delivered after the feed item, so it sorts above it in Reads until the move.
	readsItem := f.emailContent(t, senderID)
	require.Equal(t, models.ListReads, f.deliver(t, userID, readsItem))

	feedUC, err := f.ucRepo.GetByUserAndContent(ctx, userID, feedItem.ID)
	require.NoError(t, err)
	readsUC, err := f.ucRepo.GetByUserAndContent(ctx, userID, readsItem.ID)
	require.NoError(t, err)

	reads := models.ListReads
	require.NoError(t, f.ucRepo.UpdateMetadata(ctx, feedUC.ID, nil, nil, nil, &reads))

	got, err := f.ucRepo.ListByUserWithCursor(ctx, userID, nil, nil, &reads, 10, nil, nil)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, feedItem.ID, got[0].ContentID, "an item moved to Reads lands at the top")
	require.True(t, got[0].AddedAt.After(feedUC.AddedAt))

	// Re-asserting the list an item is already in must not bump it.
	require.NoError(t, f.ucRepo.UpdateMetadata(ctx, readsUC.ID, nil, nil, nil, &reads))
	got, err = f.ucRepo.ListByUserWithCursor(ctx, userID, nil, nil, &reads, 10, nil, nil)
	require.NoError(t, err)
	require.Equal(t, feedItem.ID, got[0].ContentID)
	unchanged, err := f.ucRepo.GetByID(ctx, readsUC.ID)
	require.NoError(t, err)
	require.True(t, readsUC.AddedAt.Equal(unchanged.AddedAt))
}

func TestUserContentRepository_Create_DefaultsToReads_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	f := newRoutingFixture(t)
	userID, feedID := uuid.New(), uuid.New()

	// A route must not influence a direct save (Create), only ingest (BulkCreate).
	f.route(t, userID, models.SourceTypeRSS, feedID, models.ListFeed)
	c := f.rssContent(t, feedID)

	uc := &models.UserContent{UserID: userID, ContentID: c.ID, Status: models.StatusUnread}
	require.NoError(t, f.ucRepo.Create(context.Background(), uc))
	require.Equal(t, models.ListReads, uc.List)
}

func TestSourceRouteRepository_UpsertDeleteList_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	f := newRoutingFixture(t)
	ctx := context.Background()
	userID, otherUser := uuid.New(), uuid.New()
	feedID, senderID := uuid.New(), uuid.New()

	f.route(t, userID, models.SourceTypeRSS, feedID, models.ListFeed)
	f.route(t, userID, models.SourceTypeEmail, senderID, models.ListFeed)
	f.route(t, otherUser, models.SourceTypeRSS, feedID, models.ListFeed)

	// Upsert replaces rather than duplicating.
	f.route(t, userID, models.SourceTypeRSS, feedID, models.ListReads)

	routes, err := f.routeRepo.ListByUser(ctx, userID)
	require.NoError(t, err)
	require.Len(t, routes, 2)
	byKey := map[uuid.UUID]string{}
	for _, r := range routes {
		byKey[r.SourceKey] = r.List
	}
	require.Equal(t, models.ListReads, byKey[feedID])
	require.Equal(t, models.ListFeed, byKey[senderID])

	require.NoError(t, f.routeRepo.Delete(ctx, userID, models.SourceTypeRSS, feedID))
	require.NoError(t, f.routeRepo.Delete(ctx, userID, models.SourceTypeRSS, feedID), "deleting a missing route is not an error")

	routes, err = f.routeRepo.ListByUser(ctx, userID)
	require.NoError(t, err)
	require.Len(t, routes, 1)

	routes, err = f.routeRepo.ListByUser(ctx, otherUser)
	require.NoError(t, err)
	require.Len(t, routes, 1, "deleting one user's route must not touch another's")
}
