package quick_lists

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type serviceStub struct {
	clothingResp  QuickListsClothingResponse
	clothingErr   error
	wheelsResp    QuickListsWheelsResponse
	wheelsErr     error
	batteriesResp QuickListsBatteryResponse
	batteriesErr  error
}

func (s *serviceStub) GetQuickListClothing() (QuickListsClothingResponse, error) {
	return s.clothingResp, s.clothingErr
}

func (s *serviceStub) GetQuickListWheels() (QuickListsWheelsResponse, error) {
	return s.wheelsResp, s.wheelsErr
}

func (s *serviceStub) GetQuickListBatteries() (QuickListsBatteryResponse, error) {
	return s.batteriesResp, s.batteriesErr
}

func TestQuickListsClothingSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{clothingResp: QuickListsClothingResponse{Count: 0}}

	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/quick-lists/clothing", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestQuickListsWheelsError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{wheelsErr: errors.New("db error")}

	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/quick-lists/wheels", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestQuickListsBatteriesSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{batteriesResp: QuickListsBatteryResponse{Count: 2}}

	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/quick-lists/batteries", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
}
