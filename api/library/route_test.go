package library

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type serviceStub struct {
	vehiclesResp *PMCSVehiclesResponse
	vehiclesErr  error
	docsResp     *DocumentsListResponse
	docsErr      error
	downloadResp *DownloadURLResponse
	downloadErr  error
}

func (s *serviceStub) GetPMCSVehicles() (*PMCSVehiclesResponse, error) {
	return s.vehiclesResp, s.vehiclesErr
}

func (s *serviceStub) GetPMCSDocuments(vehicleName string) (*DocumentsListResponse, error) {
	return s.docsResp, s.docsErr
}

func (s *serviceStub) GenerateDownloadURL(_ context.Context, blobPath string) (*DownloadURLResponse, error) {
	return s.downloadResp, s.downloadErr
}

func TestGetPMCSVehiclesSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{vehiclesResp: &PMCSVehiclesResponse{Vehicles: []VehicleFolderResponse{}, Count: 0}}

	registerHandlers(router.Group("/api/v1"), router.Group("/api/v1/auth"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/pmcs/vehicles", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestGetPMCSDocumentsRequiresVehicle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{}

	registerHandlers(router.Group("/api/v1"), router.Group("/api/v1/auth"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/pmcs/%20/documents", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestGenerateDownloadURLNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{downloadErr: ErrDocumentNotFound}

	registerHandlers(router.Group("/api/v1"), router.Group("/api/v1/auth"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/download?blob_path=pmcs/test.pdf", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestGenerateDownloadURLInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{downloadErr: ErrInvalidBlobPath}

	registerHandlers(router.Group("/api/v1"), router.Group("/api/v1/auth"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/download?blob_path=bad", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestGenerateDownloadURLServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{downloadErr: errors.New("boom")}

	registerHandlers(router.Group("/api/v1"), router.Group("/api/v1/auth"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/download?blob_path=pmcs/test.pdf", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusInternalServerError, resp.Code)
}
