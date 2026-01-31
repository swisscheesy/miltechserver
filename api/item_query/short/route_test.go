package short

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/item_query/shared"
)

type serviceStub struct {
	niinResp model.NiinLookup
	niinErr  error
	partResp []model.NiinLookup
	partErr  error
}

func (s *serviceStub) FindShortByNiin(string) (model.NiinLookup, error) {
	return s.niinResp, s.niinErr
}

func (s *serviceStub) FindShortByPart(string) ([]model.NiinLookup, error) {
	return s.partResp, s.partErr
}

func TestFindShortNiinNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{niinErr: shared.ErrNoItemsFound}

	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/queries/items/initial?method=niin&value=123", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestFindShortPartNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{partErr: shared.ErrNoItemsFound}

	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/queries/items/initial?method=part&value=ABC", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestFindShortNiinServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{niinErr: errors.New("boom")}

	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/queries/items/initial?method=niin&value=123", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestFindShortPartSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	niin := "013469317"
	stub := &serviceStub{partResp: []model.NiinLookup{{Niin: &niin}}}

	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/queries/items/initial?method=part&value=54321", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	payload := decodeJSON(t, resp.Body.Bytes())
	require.Equal(t, float64(http.StatusOK), payload["status"])
	require.Equal(t, "", payload["message"])
	_, ok := payload["data"].([]any)
	require.True(t, ok)
}

func TestFindShortNiinSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	niin := "013469317"
	stub := &serviceStub{niinResp: model.NiinLookup{Niin: &niin}}

	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/queries/items/initial?method=niin&value=013469317", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	payload := decodeJSON(t, resp.Body.Bytes())
	require.Equal(t, float64(http.StatusOK), payload["status"])
	require.Equal(t, "", payload["message"])
	_, ok := payload["data"].(map[string]any)
	require.True(t, ok)
}

func TestFindShortUnknownMethodNoResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{}

	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/queries/items/initial?method=unknown&value=123", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestFindShortNiinMissingValueUsesService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{niinErr: shared.ErrNoItemsFound}

	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/queries/items/initial?method=niin", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusNotFound, resp.Code)
}

func decodeJSON(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	return payload
}
