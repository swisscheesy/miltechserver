package help

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"miltechserver/.gen/miltech_ng/public/model"
)

type serviceStub struct {
	resp model.Help
	err  error
}

func (s *serviceStub) FindByCode(string) (model.Help, error) {
	return s.resp, s.err
}

func TestFindByCodeSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{resp: model.Help{Code: "AB12", Description: "Help text"}}

	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/queries/items/help?code=ab12", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	payload := decodeJSON(t, resp.Body.Bytes())
	require.Equal(t, float64(http.StatusOK), payload["status"])
	require.Equal(t, "", payload["message"])
	data, ok := payload["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "AB12", data["code"])
}

func TestFindByCodeBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{err: ErrInvalidCode}

	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/queries/items/help?code=", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestFindByCodeNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{err: ErrHelpNotFound}

	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/queries/items/help?code=none", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestFindByCodeServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{err: errors.New("boom")}

	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/queries/items/help?code=ab12", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

func decodeJSON(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	return payload
}
