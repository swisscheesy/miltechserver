package user_suggestions

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type serviceStub struct {
	suggestions   []SuggestionResponse
	suggestionsErr error
	created       *SuggestionResponse
	createErr     error
	updated       *SuggestionResponse
	updateErr     error
	deleteErr     error
	voteErr       error
	removeVoteErr error
}

func (s *serviceStub) GetAllSuggestions(currentUser *bootstrap.User) ([]SuggestionResponse, error) {
	return s.suggestions, s.suggestionsErr
}

func (s *serviceStub) CreateSuggestion(user *bootstrap.User, title, description string) (*SuggestionResponse, error) {
	return s.created, s.createErr
}

func (s *serviceStub) UpdateSuggestion(user *bootstrap.User, suggestionID, title, description string) (*SuggestionResponse, error) {
	return s.updated, s.updateErr
}

func (s *serviceStub) DeleteSuggestion(user *bootstrap.User, suggestionID string) error {
	return s.deleteErr
}

func (s *serviceStub) Vote(user *bootstrap.User, suggestionID string, direction int16) error {
	return s.voteErr
}

func (s *serviceStub) RemoveVote(user *bootstrap.User, suggestionID string) error {
	return s.removeVoteErr
}

func setupRouter(svc Service) *gin.Engine {
	router := gin.New()
	publicGroup := router.Group("/api/v1")
	authGroup := router.Group("/api/v1/auth")
	// Simulate auth middleware setting user in context
	authGroup.Use(func(c *gin.Context) {
		user := &bootstrap.User{UserID: "user-1", Username: "testuser", Email: "test@test.com"}
		c.Set("user", user)
		c.Next()
	})
	registerHandlers(publicGroup, authGroup, nil, svc)
	return router
}

func performRequest(router *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBytes)
	} else {
		reqBody = &bytes.Buffer{}
	}

	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestListSuggestions_200(t *testing.T) {
	svc := &serviceStub{
		suggestions: []SuggestionResponse{
			{ID: "abc-123", Title: "Feature A", Score: 5},
		},
	}
	router := setupRouter(svc)

	w := performRequest(router, "GET", "/api/v1/suggestions", nil)
	require.Equal(t, http.StatusOK, w.Code)

	var resp response.StandardResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Status)
}

func TestCreateSuggestion_201(t *testing.T) {
	svc := &serviceStub{
		created: &SuggestionResponse{
			ID:    "abc-123",
			Title: "New Feature",
		},
	}
	router := setupRouter(svc)

	body := CreateSuggestionRequest{Title: "New Feature", Description: "A great feature"}
	w := performRequest(router, "POST", "/api/v1/auth/suggestions", body)
	require.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateSuggestion_400_Validation(t *testing.T) {
	svc := &serviceStub{createErr: ErrInvalidTitle}
	router := setupRouter(svc)

	body := CreateSuggestionRequest{Title: "", Description: "Desc"}
	w := performRequest(router, "POST", "/api/v1/auth/suggestions", body)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSuggestion_200(t *testing.T) {
	svc := &serviceStub{
		updated: &SuggestionResponse{ID: "abc-123", Title: "Updated"},
	}
	router := setupRouter(svc)

	body := UpdateSuggestionRequest{Title: "Updated", Description: "Updated desc"}
	w := performRequest(router, "PUT", "/api/v1/auth/suggestions/abc-123", body)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateSuggestion_403(t *testing.T) {
	svc := &serviceStub{updateErr: ErrForbidden}
	router := setupRouter(svc)

	body := UpdateSuggestionRequest{Title: "Updated", Description: "Updated desc"}
	w := performRequest(router, "PUT", "/api/v1/auth/suggestions/abc-123", body)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestUpdateSuggestion_404(t *testing.T) {
	svc := &serviceStub{updateErr: ErrSuggestionNotFound}
	router := setupRouter(svc)

	body := UpdateSuggestionRequest{Title: "Updated", Description: "Updated desc"}
	w := performRequest(router, "PUT", "/api/v1/auth/suggestions/abc-123", body)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteSuggestion_200(t *testing.T) {
	svc := &serviceStub{}
	router := setupRouter(svc)

	w := performRequest(router, "DELETE", "/api/v1/auth/suggestions/abc-123", nil)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteSuggestion_403(t *testing.T) {
	svc := &serviceStub{deleteErr: ErrForbidden}
	router := setupRouter(svc)

	w := performRequest(router, "DELETE", "/api/v1/auth/suggestions/abc-123", nil)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestVote_200(t *testing.T) {
	svc := &serviceStub{}
	router := setupRouter(svc)

	body := VoteRequest{Direction: 1}
	w := performRequest(router, "POST", "/api/v1/auth/suggestions/abc-123/vote", body)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestVote_400_InvalidDirection(t *testing.T) {
	svc := &serviceStub{voteErr: ErrInvalidDirection}
	router := setupRouter(svc)

	body := VoteRequest{Direction: 0}
	w := performRequest(router, "POST", "/api/v1/auth/suggestions/abc-123/vote", body)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRemoveVote_200(t *testing.T) {
	svc := &serviceStub{}
	router := setupRouter(svc)

	w := performRequest(router, "DELETE", "/api/v1/auth/suggestions/abc-123/vote", nil)
	require.Equal(t, http.StatusOK, w.Code)
}
