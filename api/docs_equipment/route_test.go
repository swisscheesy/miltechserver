package docs_equipment

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// serviceStub implements Service for handler tests.
type serviceStub struct {
	pageResp     EquipmentDetailsPageResponse
	familiesResp FamiliesResponse
	imgFamilies  *ImageFamiliesResponse
	imgList      *FamilyImagesResponse
	imgDownload  *ImageDownloadResponse
	err          error
}

func (s *serviceStub) GetAllPaginated(page int) (EquipmentDetailsPageResponse, error) {
	return s.pageResp, s.err
}
func (s *serviceStub) GetFamilies() (FamiliesResponse, error) {
	return s.familiesResp, s.err
}
func (s *serviceStub) GetByFamilyPaginated(family string, page int) (EquipmentDetailsPageResponse, error) {
	return s.pageResp, s.err
}
func (s *serviceStub) SearchPaginated(query string, page int) (EquipmentDetailsPageResponse, error) {
	return s.pageResp, s.err
}
func (s *serviceStub) ListImageFamilies() (*ImageFamiliesResponse, error) {
	return s.imgFamilies, s.err
}
func (s *serviceStub) ListFamilyImages(family string) (*FamilyImagesResponse, error) {
	return s.imgList, s.err
}
func (s *serviceStub) GenerateImageDownloadURL(_ context.Context, _ string) (*ImageDownloadResponse, error) {
	return s.imgDownload, s.err
}

func newTestRouter(stub *serviceStub) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerHandlers(router.Group("/api/v1"), stub)
	return router
}

func doRequest(router *gin.Engine, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestGetAllPaginatedSuccess(t *testing.T) {
	stub := &serviceStub{pageResp: EquipmentDetailsPageResponse{Count: 40, Page: 1}}
	resp := doRequest(newTestRouter(stub), http.MethodGet, "/api/v1/equipment-details?page=1")
	require.Equal(t, http.StatusOK, resp.Code)
}

func TestGetAllPaginatedInvalidPage(t *testing.T) {
	stub := &serviceStub{}
	resp := doRequest(newTestRouter(stub), http.MethodGet, "/api/v1/equipment-details?page=0")
	require.Equal(t, http.StatusBadRequest, resp.Code)

	resp = doRequest(newTestRouter(stub), http.MethodGet, "/api/v1/equipment-details?page=abc")
	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestGetAllPaginatedError(t *testing.T) {
	stub := &serviceStub{err: errors.New("db down")}
	resp := doRequest(newTestRouter(stub), http.MethodGet, "/api/v1/equipment-details?page=1")
	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestGetAllPaginatedNotFound(t *testing.T) {
	stub := &serviceStub{err: ErrNotFound}
	resp := doRequest(newTestRouter(stub), http.MethodGet, "/api/v1/equipment-details?page=1")
	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestGetFamiliesSuccess(t *testing.T) {
	stub := &serviceStub{familiesResp: FamiliesResponse{Families: []string{"aircraft"}, Count: 1}}
	resp := doRequest(newTestRouter(stub), http.MethodGet, "/api/v1/equipment-details/families")
	require.Equal(t, http.StatusOK, resp.Code)
}

func TestGetFamiliesError(t *testing.T) {
	stub := &serviceStub{err: errors.New("db down")}
	resp := doRequest(newTestRouter(stub), http.MethodGet, "/api/v1/equipment-details/families")
	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestGetByFamilySuccess(t *testing.T) {
	stub := &serviceStub{pageResp: EquipmentDetailsPageResponse{Count: 5, Page: 1}}
	resp := doRequest(newTestRouter(stub), http.MethodGet, "/api/v1/equipment-details/family/aircraft?page=1")
	require.Equal(t, http.StatusOK, resp.Code)
}

func TestGetByFamilyInvalidPage(t *testing.T) {
	stub := &serviceStub{}
	resp := doRequest(newTestRouter(stub), http.MethodGet, "/api/v1/equipment-details/family/aircraft?page=abc")
	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestSearchMissingQuery(t *testing.T) {
	stub := &serviceStub{}
	resp := doRequest(newTestRouter(stub), http.MethodGet, "/api/v1/equipment-details/search")
	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestSearchSuccess(t *testing.T) {
	stub := &serviceStub{pageResp: EquipmentDetailsPageResponse{Count: 2, Page: 1}}
	resp := doRequest(newTestRouter(stub), http.MethodGet, "/api/v1/equipment-details/search?q=AH-64&page=1")
	require.Equal(t, http.StatusOK, resp.Code)
}

func TestSearchNotFound(t *testing.T) {
	stub := &serviceStub{err: ErrNotFound}
	resp := doRequest(newTestRouter(stub), http.MethodGet, "/api/v1/equipment-details/search?q=NOTHING&page=1")
	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestListImageFamiliesSuccess(t *testing.T) {
	stub := &serviceStub{imgFamilies: &ImageFamiliesResponse{Count: 1}}
	resp := doRequest(newTestRouter(stub), http.MethodGet, "/api/v1/equipment-details/images/families")
	require.Equal(t, http.StatusOK, resp.Code)
}

func TestListFamilyImagesSuccess(t *testing.T) {
	stub := &serviceStub{imgList: &FamilyImagesResponse{Family: "aircraft", Count: 3}}
	resp := doRequest(newTestRouter(stub), http.MethodGet, "/api/v1/equipment-details/images/family/aircraft")
	require.Equal(t, http.StatusOK, resp.Code)
}

func TestGenerateImageDownloadBadRequest(t *testing.T) {
	stub := &serviceStub{err: ErrEmptyBlobPath}
	resp := doRequest(newTestRouter(stub), http.MethodGet, "/api/v1/equipment-details/images/download")
	require.Equal(t, http.StatusBadRequest, resp.Code)
}
