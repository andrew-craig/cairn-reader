package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/services/read/content/internal/models"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockContentFeedbackRepository is a mock implementation of repository.ContentFeedbackRepository
type MockContentFeedbackRepository struct {
	mock.Mock
}

func (m *MockContentFeedbackRepository) Upsert(ctx context.Context, f *models.ContentFeedback) (*models.ContentFeedback, error) {
	args := m.Called(ctx, f)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ContentFeedback), args.Error(1)
}

func (m *MockContentFeedbackRepository) GetByUserAndContent(ctx context.Context, userID, contentID uuid.UUID) (*models.ContentFeedback, error) {
	args := m.Called(ctx, userID, contentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ContentFeedback), args.Error(1)
}

type feedbackFixture struct {
	handler         *FeedbackHandler
	userContentRepo *MockUserContentRepository
	contentRepo     *MockContentRepository
	feedbackRepo    *MockContentFeedbackRepository
	userID          uuid.UUID
	contentID       uuid.UUID
}

func newFeedbackFixture() *feedbackFixture {
	f := &feedbackFixture{
		userContentRepo: new(MockUserContentRepository),
		contentRepo:     new(MockContentRepository),
		feedbackRepo:    new(MockContentFeedbackRepository),
		userID:          uuid.New(),
		contentID:       uuid.New(),
	}
	f.handler = NewFeedbackHandler(service.NewFeedbackService(f.userContentRepo, f.contentRepo, f.feedbackRepo), f.feedbackRepo)
	return f
}

// request builds a feedback request authenticated as authUserID.
func (f *feedbackFixture) request(method string, authUserID uuid.UUID, body string) *http.Request {
	req := httptest.NewRequest(method, "/api/v1/content/user/"+f.userID.String()+"/"+f.contentID.String()+"/feedback", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", f.userID.String())
	rctx.URLParams.Add("content_id", f.contentID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return addAuthContextToRequest(req, authUserID)
}

func (f *feedbackFixture) savedReport(comment *string) *models.ContentFeedback {
	return &models.ContentFeedback{
		ID:         uuid.New(),
		UserID:     f.userID,
		ContentID:  f.contentID,
		Categories: pq.StringArray{"images", "title"},
		Comment:    comment,
		Status:     models.FeedbackStatusOpen,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func TestPutFeedback_Success(t *testing.T) {
	f := newFeedbackFixture()
	comment := "images are missing"
	author := "Ann"
	content := &models.Content{
		ID: f.contentID, OriginalURL: "https://example.com/a", Title: "T", Author: &author,
		CleanedHTML: "<p>x</p>", ImageURLs: pq.StringArray{"https://example.com/1.png", "https://example.com/2.png"},
	}
	f.userContentRepo.On("GetByUserAndContent", mock.Anything, f.userID, f.contentID).Return(&models.UserContent{ID: uuid.New()}, nil)
	f.contentRepo.On("GetByID", mock.Anything, f.contentID).Return(content, nil)
	f.contentRepo.On("GetSource", mock.Anything, f.contentID).Return(&models.ContentSource{RawHTML: "<html>raw</html>", ReadabilityFailed: true}, nil)
	f.feedbackRepo.On("Upsert", mock.Anything, mock.MatchedBy(func(fb *models.ContentFeedback) bool {
		s := fb.Snapshot
		return fb.UserID == f.userID && fb.ContentID == f.contentID &&
			assert.ObjectsAreEqual(pq.StringArray{"images", "title"}, fb.Categories) &&
			*fb.Comment == comment &&
			s.OriginalURL == "https://example.com/a" && s.Title == "T" && *s.Author == "Ann" &&
			*s.ImageURL == "https://example.com/1.png" && s.CleanedHTML == "<p>x</p>" &&
			*s.RawHTML == "<html>raw</html>" && *s.ReadabilityFailed
	})).Return(f.savedReport(&comment), nil)

	w := httptest.NewRecorder()
	f.handler.PutFeedback(w, f.request(http.MethodPut, f.userID, `{"categories":["images","title"],"comment":"images are missing"}`))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, f.contentID.String(), data["content_id"])
	assert.Equal(t, []interface{}{"images", "title"}, data["categories"])
	assert.Equal(t, comment, data["comment"])
	assert.Equal(t, "open", data["status"])
	f.feedbackRepo.AssertExpectations(t)
}

func TestPutFeedback_ExpiredSourceLeavesRawFieldsNil(t *testing.T) {
	f := newFeedbackFixture()
	f.userContentRepo.On("GetByUserAndContent", mock.Anything, f.userID, f.contentID).Return(&models.UserContent{ID: uuid.New()}, nil)
	f.contentRepo.On("GetByID", mock.Anything, f.contentID).Return(&models.Content{ID: f.contentID, OriginalURL: "https://example.com/a", Title: "T"}, nil)
	f.contentRepo.On("GetSource", mock.Anything, f.contentID).Return(nil, nil)
	f.feedbackRepo.On("Upsert", mock.Anything, mock.MatchedBy(func(fb *models.ContentFeedback) bool {
		s := fb.Snapshot
		return s.RawHTML == nil && s.ReadabilityFailed == nil && s.ImageURL == nil
	})).Return(f.savedReport(nil), nil)

	w := httptest.NewRecorder()
	f.handler.PutFeedback(w, f.request(http.MethodPut, f.userID, `{"categories":["other"]}`))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	f.feedbackRepo.AssertExpectations(t)
}

func TestPutFeedback_InvalidBody(t *testing.T) {
	long := strings.Repeat("a", 2001)
	cases := []struct {
		name string
		body string
	}{
		{"empty categories", `{"categories":[]}`},
		{"missing categories", `{"comment":"x"}`},
		{"unknown category", `{"categories":["images","bogus"]}`},
		{"empty category", `{"categories":[""]}`},
		{"duplicate categories", `{"categories":["images","images"]}`},
		{"comment too long", `{"categories":["images"],"comment":"` + long + `"}`},
		{"unknown field", `{"categories":["images"],"extra":1}`},
		{"malformed json", `{"categories":`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFeedbackFixture()
			w := httptest.NewRecorder()
			f.handler.PutFeedback(w, f.request(http.MethodPut, f.userID, tc.body))

			assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			f.feedbackRepo.AssertNotCalled(t, "Upsert", mock.Anything, mock.Anything)
		})
	}
}

func TestPutFeedback_CommentLimitCountsCharacters(t *testing.T) {
	f := newFeedbackFixture()
	comment := strings.Repeat("é", 2000) // 4000 bytes, 2000 characters
	f.userContentRepo.On("GetByUserAndContent", mock.Anything, f.userID, f.contentID).Return(&models.UserContent{ID: uuid.New()}, nil)
	f.contentRepo.On("GetByID", mock.Anything, f.contentID).Return(&models.Content{ID: f.contentID}, nil)
	f.contentRepo.On("GetSource", mock.Anything, f.contentID).Return(nil, nil)
	f.feedbackRepo.On("Upsert", mock.Anything, mock.Anything).Return(f.savedReport(&comment), nil)

	w := httptest.NewRecorder()
	f.handler.PutFeedback(w, f.request(http.MethodPut, f.userID, `{"categories":["other"],"comment":"`+comment+`"}`))

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestPutFeedback_NotOwned(t *testing.T) {
	f := newFeedbackFixture()
	f.userContentRepo.On("GetByUserAndContent", mock.Anything, f.userID, f.contentID).Return(nil, nil)

	w := httptest.NewRecorder()
	f.handler.PutFeedback(w, f.request(http.MethodPut, f.userID, `{"categories":["images"]}`))

	assert.Equal(t, http.StatusNotFound, w.Code)
	f.feedbackRepo.AssertNotCalled(t, "Upsert", mock.Anything, mock.Anything)
}

func TestFeedback_PathErrors(t *testing.T) {
	f := newFeedbackFixture()
	other := uuid.New()

	w := httptest.NewRecorder()
	f.handler.PutFeedback(w, f.request(http.MethodPut, other, `{"categories":["images"]}`))
	assert.Equal(t, http.StatusForbidden, w.Code)

	w = httptest.NewRecorder()
	f.handler.GetFeedback(w, f.request(http.MethodGet, other, ""))
	assert.Equal(t, http.StatusForbidden, w.Code)

	req := f.request(http.MethodGet, f.userID, "")
	rctx := chi.RouteContext(req.Context())
	rctx.URLParams.Keys[1], rctx.URLParams.Values[1] = "content_id", "not-a-uuid"
	w = httptest.NewRecorder()
	f.handler.GetFeedback(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetFeedback_Success_HasNoSnapshotFields(t *testing.T) {
	f := newFeedbackFixture()
	report := f.savedReport(nil)
	report.Snapshot = &models.FeedbackSnapshot{OriginalURL: "https://example.com/a", CleanedHTML: "<p>x</p>"}
	f.feedbackRepo.On("GetByUserAndContent", mock.Anything, f.userID, f.contentID).Return(report, nil)

	w := httptest.NewRecorder()
	f.handler.GetFeedback(w, f.request(http.MethodGet, f.userID, ""))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	data := resp["data"].(map[string]interface{})
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	assert.ElementsMatch(t, []string{"content_id", "categories", "comment", "status", "created_at", "updated_at"}, keys)
	assert.Nil(t, data["comment"])
	assert.NotContains(t, w.Body.String(), "raw_html")
	assert.NotContains(t, w.Body.String(), "cleaned_html")
}

func TestGetFeedback_NoReport(t *testing.T) {
	f := newFeedbackFixture()
	f.feedbackRepo.On("GetByUserAndContent", mock.Anything, f.userID, f.contentID).Return(nil, nil)

	w := httptest.NewRecorder()
	f.handler.GetFeedback(w, f.request(http.MethodGet, f.userID, ""))

	assert.Equal(t, http.StatusNotFound, w.Code)
}
