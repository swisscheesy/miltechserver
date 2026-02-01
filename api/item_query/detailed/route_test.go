package detailed

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"miltechserver/api/middleware"
	"miltechserver/api/response"
)

type serviceStub struct {
	resp response.DetailedResponse
	err  error
}

func (s *serviceStub) FindDetailedItem(ctx context.Context, niin string) (response.DetailedResponse, error) {
	return s.resp, s.err
}

func TestFindDetailedSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{resp: response.DetailedResponse{}}

	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/queries/items/detailed?niin=123", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	payload := decodeJSON(t, resp.Body.Bytes())
	require.Equal(t, float64(http.StatusOK), payload["status"])
	require.Equal(t, "", payload["message"])
	_, ok := payload["data"].(map[string]any)
	require.True(t, ok)
}

func TestFindDetailedErrorUsesMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler)
	stub := &serviceStub{err: errBoom{}}

	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/queries/items/detailed?niin=123", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

type errBoom struct{}

func (errBoom) Error() string {
	return "boom"
}

func decodeJSON(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	return payload
}
