//go:build integration
// +build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cairn-app/cairn-reader/services/read/content/internal/api"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/api/dto"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/database"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContentCreationIntegration tests the full content creation flow
// HTTP → Service → Repository → Database
func TestContentCreationIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Cleanup()

	// Create test server
	dbWrapper := &database.DB{DB: testDB.DB}
	router := api.NewRouter(dbWrapper)
	server := httptest.NewServer(router)
	defer server.Close()

	t.Run("CreateContentFromHTML", func(t *testing.T) {
		// Create content via API
		reqBody := dto.CreateContentRequest{
			URL:        "https://example.com/article1",
			HTML:       "<html><body><h1>Test Article</h1><p>Test content here.</p></body></html>",
			SourceType: "web",
		}
		bodyBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)

		resp, err := http.Post(
			server.URL+"/api/v1/contents",
			"application/json",
			bytes.NewReader(bodyBytes),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify response
		var contentResp dto.ContentResponse
		err = json.NewDecoder(resp.Body).Decode(&contentResp)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, contentResp.ID)
		assert.Equal(t, "Test Article", contentResp.Title)
		assert.NotEmpty(t, contentResp.ContentHash)

		// Verify database state
		var count int
		err = testDB.DB.QueryRow("SELECT COUNT(*) FROM contents").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		// Verify content is properly sanitized (no script tags)
		var cleanedHTML string
		err = testDB.DB.QueryRow("SELECT cleaned_html FROM contents WHERE id = $1", contentResp.ID).Scan(&cleanedHTML)
		require.NoError(t, err)
		assert.NotContains(t, cleanedHTML, "<script>")
		assert.Contains(t, cleanedHTML, "Test Article")
	})

	t.Run("GetContentByID", func(t *testing.T) {
		testDB.TruncateAll()

		// First create content
		reqBody := dto.CreateContentRequest{
			URL:        "https://example.com/article2",
			HTML:       "<html><body><h1>Article 2</h1></body></html>",
			SourceType: "web",
		}
		bodyBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)

		resp, err := http.Post(
			server.URL+"/api/v1/contents",
			"application/json",
			bytes.NewReader(bodyBytes),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		var createResp dto.ContentResponse
		err = json.NewDecoder(resp.Body).Decode(&createResp)
		require.NoError(t, err)

		// Now retrieve it
		getResp, err := http.Get(fmt.Sprintf("%s/api/v1/contents/%s", server.URL, createResp.ID))
		require.NoError(t, err)
		defer getResp.Body.Close()

		assert.Equal(t, http.StatusOK, getResp.StatusCode)

		var getContentResp dto.ContentResponse
		err = json.NewDecoder(getResp.Body).Decode(&getContentResp)
		require.NoError(t, err)
		assert.Equal(t, createResp.ID, getContentResp.ID)
		assert.Equal(t, "Article 2", getContentResp.Title)
	})

	t.Run("BulkCreateContent", func(t *testing.T) {
		testDB.TruncateAll()

		// Create multiple contents in bulk
		feedID := uuid.New()
		reqBody := dto.BulkCreateContentRequest{
			Contents: []dto.ContentItem{
				{
					URL:        "https://example.com/bulk1",
					HTML:       "<html><body><h1>Bulk Article 1</h1></body></html>",
					SourceType: "rss",
					FeedID:     &feedID,
				},
				{
					URL:        "https://example.com/bulk2",
					HTML:       "<html><body><h1>Bulk Article 2</h1></body></html>",
					SourceType: "rss",
					FeedID:     &feedID,
				},
			},
		}
		bodyBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)

		resp, err := http.Post(
			server.URL+"/api/v1/contents/bulk",
			"application/json",
			bytes.NewReader(bodyBytes),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var bulkResp dto.BulkCreateContentResponse
		err = json.NewDecoder(resp.Body).Decode(&bulkResp)
		require.NoError(t, err)
		assert.Len(t, bulkResp.Created, 2)
		assert.Len(t, bulkResp.Errors, 0)

		// Verify database state
		var count int
		err = testDB.DB.QueryRow("SELECT COUNT(*) FROM contents").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})
}

// TestUserContentIntegration tests user-content relationship management
func TestUserContentIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Cleanup()

	dbWrapper := &database.DB{DB: testDB.DB}
	router := api.NewRouter(dbWrapper)
	server := httptest.NewServer(router)
	defer server.Close()

	// Create a content first
	contentReqBody := dto.CreateContentRequest{
		URL:        "https://example.com/test-article",
		HTML:       "<html><body><h1>Test Article</h1></body></html>",
		SourceType: "web",
	}
	bodyBytes, err := json.Marshal(contentReqBody)
	require.NoError(t, err)

	resp, err := http.Post(
		server.URL+"/api/v1/contents",
		"application/json",
		bytes.NewReader(bodyBytes),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	var contentResp dto.ContentResponse
	err = json.NewDecoder(resp.Body).Decode(&contentResp)
	require.NoError(t, err)

	userID := uuid.New()

	t.Run("AddContentToUser", func(t *testing.T) {
		// Add content to user
		addReq := dto.AddContentToUserRequest{
			ContentID: contentResp.ID,
		}
		bodyBytes, err := json.Marshal(addReq)
		require.NoError(t, err)

		resp, err := http.Post(
			fmt.Sprintf("%s/api/v1/users/%s/contents", server.URL, userID),
			"application/json",
			bytes.NewReader(bodyBytes),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		// Verify user_contents table
		var count int
		err = testDB.DB.QueryRow("SELECT COUNT(*) FROM user_contents WHERE user_id = $1", userID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("ListUserContents", func(t *testing.T) {
		// List user's contents
		resp, err := http.Get(fmt.Sprintf("%s/api/v1/users/%s/contents", server.URL, userID))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var listResp dto.ListUserContentsResponse
		err = json.NewDecoder(resp.Body).Decode(&listResp)
		require.NoError(t, err)
		assert.Len(t, listResp.Contents, 1)
		assert.Equal(t, "unread", listResp.Contents[0].Status)
		assert.False(t, listResp.Contents[0].IsFavorite)
	})

	t.Run("UpdateUserContent", func(t *testing.T) {
		// Update user content metadata
		updateReq := dto.UpdateUserContentRequest{
			Status:         "reading",
			IsFavorite:     true,
			ScrollPosition: 100,
		}
		bodyBytes, err := json.Marshal(updateReq)
		require.NoError(t, err)

		req, err := http.NewRequest(
			"PATCH",
			fmt.Sprintf("%s/api/v1/users/%s/contents/%s", server.URL, userID, contentResp.ID),
			bytes.NewReader(bodyBytes),
		)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify database state
		var status string
		var isFavorite bool
		var scrollPosition int
		err = testDB.DB.QueryRow(
			"SELECT status, is_favorite, scroll_position FROM user_contents WHERE user_id = $1 AND content_id = $2",
			userID, contentResp.ID,
		).Scan(&status, &isFavorite, &scrollPosition)
		require.NoError(t, err)
		assert.Equal(t, "reading", status)
		assert.True(t, isFavorite)
		assert.Equal(t, 100, scrollPosition)
	})

	t.Run("DeleteUserContent", func(t *testing.T) {
		// Delete user content
		req, err := http.NewRequest(
			"DELETE",
			fmt.Sprintf("%s/api/v1/users/%s/contents/%s", server.URL, userID, contentResp.ID),
			nil,
		)
		require.NoError(t, err)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify deletion
		var count int
		err = testDB.DB.QueryRow(
			"SELECT COUNT(*) FROM user_contents WHERE user_id = $1 AND content_id = $2",
			userID, contentResp.ID,
		).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

// TestSearchIntegration tests PostgreSQL full-text search functionality
func TestSearchIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Cleanup()

	dbWrapper := &database.DB{DB: testDB.DB}
	router := api.NewRouter(dbWrapper)
	server := httptest.NewServer(router)
	defer server.Close()

	userID := uuid.New()

	// Create multiple contents with different titles and authors
	contents := []struct {
		title  string
		author string
		html   string
	}{
		{
			title:  "Understanding Go Programming",
			author: "John Doe",
			html:   "<html><body><h1>Understanding Go Programming</h1><p>By John Doe</p></body></html>",
		},
		{
			title:  "Python Best Practices",
			author: "Jane Smith",
			html:   "<html><body><h1>Python Best Practices</h1><p>By Jane Smith</p></body></html>",
		},
		{
			title:  "Advanced Go Techniques",
			author: "Bob Johnson",
			html:   "<html><body><h1>Advanced Go Techniques</h1><p>By Bob Johnson</p></body></html>",
		},
	}

	for i, c := range contents {
		// Create content with explicit metadata
		ctx := context.Background()
		var contentID uuid.UUID
		err := testDB.DB.QueryRowContext(ctx, `
			INSERT INTO contents (content_hash, cleaned_html, original_url, title, author, source_type)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`, fmt.Sprintf("hash%d", i), c.html, fmt.Sprintf("https://example.com/%d", i), c.title, c.author, "web").
			Scan(&contentID)
		require.NoError(t, err)

		// Add to user's list
		_, err = testDB.DB.ExecContext(ctx, `
			INSERT INTO user_contents (user_id, content_id, status)
			VALUES ($1, $2, 'unread')
		`, userID, contentID)
		require.NoError(t, err)
	}

	t.Run("SearchByTitle", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/api/v1/users/%s/contents/search?q=Go", server.URL, userID))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var searchResp dto.ListUserContentsResponse
		err = json.NewDecoder(resp.Body).Decode(&searchResp)
		require.NoError(t, err)

		// Should find 2 articles with "Go" in the title
		assert.Len(t, searchResp.Contents, 2)
	})

	t.Run("SearchByAuthor", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/api/v1/users/%s/contents/search?q=Jane", server.URL, userID))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var searchResp dto.ListUserContentsResponse
		err = json.NewDecoder(resp.Body).Decode(&searchResp)
		require.NoError(t, err)

		// Should find 1 article by Jane Smith
		assert.Len(t, searchResp.Contents, 1)
		assert.Contains(t, searchResp.Contents[0].Title, "Python")
	})

	t.Run("SearchNoResults", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/api/v1/users/%s/contents/search?q=Rust", server.URL, userID))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var searchResp dto.ListUserContentsResponse
		err = json.NewDecoder(resp.Body).Decode(&searchResp)
		require.NoError(t, err)

		// Should find no articles about Rust
		assert.Len(t, searchResp.Contents, 0)
	})
}

// TestContentUpdatePropagation tests that content updates preserve user-content relationships
func TestContentUpdatePropagation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Cleanup()

	dbWrapper := &database.DB{DB: testDB.DB}
	router := api.NewRouter(dbWrapper)
	server := httptest.NewServer(router)
	defer server.Close()

	// Create original content
	createReq := dto.CreateContentRequest{
		URL:        "https://example.com/updatable-article",
		HTML:       "<html><body><h1>Original Title</h1><p>Original content</p></body></html>",
		SourceType: "web",
	}
	bodyBytes, err := json.Marshal(createReq)
	require.NoError(t, err)

	resp, err := http.Post(
		server.URL+"/api/v1/contents",
		"application/json",
		bytes.NewReader(bodyBytes),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	var contentResp dto.ContentResponse
	err = json.NewDecoder(resp.Body).Decode(&contentResp)
	require.NoError(t, err)
	originalContentID := contentResp.ID

	// Add to two different users
	user1ID := uuid.New()
	user2ID := uuid.New()

	for _, userID := range []uuid.UUID{user1ID, user2ID} {
		addReq := dto.AddContentToUserRequest{
			ContentID: originalContentID,
		}
		bodyBytes, err := json.Marshal(addReq)
		require.NoError(t, err)

		resp, err := http.Post(
			fmt.Sprintf("%s/api/v1/users/%s/contents", server.URL, userID),
			"application/json",
			bytes.NewReader(bodyBytes),
		)
		require.NoError(t, err)
		resp.Body.Close()
	}

	// Update user1's metadata
	updateUserContentReq := dto.UpdateUserContentRequest{
		Status:     "read",
		IsFavorite: true,
	}
	bodyBytes, err = json.Marshal(updateUserContentReq)
	require.NoError(t, err)

	req, err := http.NewRequest(
		"PATCH",
		fmt.Sprintf("%s/api/v1/users/%s/contents/%s", server.URL, user1ID, originalContentID),
		bytes.NewReader(bodyBytes),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err = client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	// Now update the content itself
	updateContentReq := dto.UpdateContentRequest{
		HTML: "<html><body><h1>Updated Title</h1><p>Updated content here</p></body></html>",
	}
	bodyBytes, err = json.Marshal(updateContentReq)
	require.NoError(t, err)

	req, err = http.NewRequest(
		"PUT",
		fmt.Sprintf("%s/api/v1/contents/%s", server.URL, originalContentID),
		bytes.NewReader(bodyBytes),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify content was updated
	var updatedResp dto.ContentResponse
	err = json.NewDecoder(resp.Body).Decode(&updatedResp)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", updatedResp.Title)
	assert.Equal(t, originalContentID, updatedResp.ID) // Same ID

	// Verify user-content relationships still exist
	var count int
	err = testDB.DB.QueryRow(
		"SELECT COUNT(*) FROM user_contents WHERE content_id = $1",
		originalContentID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "Both users should still have the content")

	// Verify user1's metadata was preserved
	var status string
	var isFavorite bool
	err = testDB.DB.QueryRow(
		"SELECT status, is_favorite FROM user_contents WHERE user_id = $1 AND content_id = $2",
		user1ID, originalContentID,
	).Scan(&status, &isFavorite)
	require.NoError(t, err)
	assert.Equal(t, "read", status, "User1's status should be preserved")
	assert.True(t, isFavorite, "User1's favorite flag should be preserved")

	// Verify user2's metadata remains default
	err = testDB.DB.QueryRow(
		"SELECT status, is_favorite FROM user_contents WHERE user_id = $1 AND content_id = $2",
		user2ID, originalContentID,
	).Scan(&status, &isFavorite)
	require.NoError(t, err)
	assert.Equal(t, "unread", status, "User2's status should remain default")
	assert.False(t, isFavorite, "User2's favorite flag should remain default")
}

// TestDeduplicationIntegration tests content deduplication by hash and feed ID
func TestDeduplicationIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Cleanup()

	dbWrapper := &database.DB{DB: testDB.DB}
	router := api.NewRouter(dbWrapper)
	server := httptest.NewServer(router)
	defer server.Close()

	feedID := uuid.New()

	// Create content from RSS feed
	createReq := dto.CreateContentRequest{
		URL:        "https://example.com/rss-article",
		HTML:       "<html><body><h1>RSS Article</h1><p>Some content</p></body></html>",
		SourceType: "rss",
		FeedID:     &feedID,
	}
	bodyBytes, err := json.Marshal(createReq)
	require.NoError(t, err)

	resp, err := http.Post(
		server.URL+"/api/v1/contents",
		"application/json",
		bytes.NewReader(bodyBytes),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	var firstResp dto.ContentResponse
	err = json.NewDecoder(resp.Body).Decode(&firstResp)
	require.NoError(t, err)
	firstContentHash := firstResp.ContentHash

	// Try to create the same content again (should use deduplication)
	resp, err = http.Post(
		server.URL+"/api/v1/contents",
		"application/json",
		bytes.NewReader(bodyBytes),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	var secondResp dto.ContentResponse
	err = json.NewDecoder(resp.Body).Decode(&secondResp)
	require.NoError(t, err)

	// Should return the same content (deduplication)
	assert.Equal(t, firstResp.ID, secondResp.ID, "Should return same content ID")
	assert.Equal(t, firstContentHash, secondResp.ContentHash, "Content hash should match")

	// Verify only one content exists in database
	var count int
	err = testDB.DB.QueryRow("SELECT COUNT(*) FROM contents WHERE content_hash = $1", firstContentHash).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "Should only have one content entry")

	// Test check-duplicates endpoint
	checkReq := dto.CheckDuplicatesRequest{
		FeedID:       feedID,
		ContentItems: []string{firstContentHash, "nonexistenthash"},
	}
	bodyBytes, err = json.Marshal(checkReq)
	require.NoError(t, err)

	resp, err = http.Post(
		server.URL+"/api/v1/contents/check-duplicates",
		"application/json",
		bytes.NewReader(bodyBytes),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	var checkResp dto.CheckDuplicatesResponse
	err = json.NewDecoder(resp.Body).Decode(&checkResp)
	require.NoError(t, err)

	// Should find one existing content
	assert.Len(t, checkResp.Existing, 1, "Should find one existing content")
	assert.Equal(t, firstContentHash, checkResp.Existing[0].ContentHash)
	assert.Equal(t, firstResp.ID, checkResp.Existing[0].ContentID)
}
