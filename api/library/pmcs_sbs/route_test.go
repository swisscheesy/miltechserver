package pmcs_sbs

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// serviceStub implements Service for handler testing.
type serviceStub struct {
	foldersResp *FoldersListResponse
	foldersErr  error
	filesResp   *FilesListResponse
	filesErr    error
	contentResp json.RawMessage
	contentErr  error
}

func (s *serviceStub) GetFolders(_ context.Context) (*FoldersListResponse, error) {
	return s.foldersResp, s.foldersErr
}

func (s *serviceStub) GetFiles(_ context.Context, _ string) (*FilesListResponse, error) {
	return s.filesResp, s.filesErr
}

func (s *serviceStub) GetFileContent(_ context.Context, _ string) (json.RawMessage, error) {
	return s.contentResp, s.contentErr
}

// capturingStub wraps serviceStub and records method call arguments.
type capturingStub struct {
	serviceStub
	capturedFolder   string
	capturedBlobPath string
}

func (s *capturingStub) GetFiles(_ context.Context, folder string) (*FilesListResponse, error) {
	s.capturedFolder = folder
	return s.filesResp, s.filesErr
}

func (s *capturingStub) GetFileContent(_ context.Context, blobPath string) (json.RawMessage, error) {
	s.capturedBlobPath = blobPath
	return s.contentResp, s.contentErr
}

// --- GetFolders ---

func TestGetFoldersSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{
		foldersResp: &FoldersListResponse{
			Folders: []FolderResponse{{Name: "hmmwv", FullPath: "pmcs_sbs/hmmwv/", DisplayName: "HMMWV"}},
			Count:   1,
		},
	}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/pmcs-sbs/folders", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestGetFoldersEmptyResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{foldersResp: &FoldersListResponse{Folders: []FolderResponse{}, Count: 0}}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/pmcs-sbs/folders", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestGetFoldersServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{foldersErr: ErrBlobListFailed}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/pmcs-sbs/folders", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

// --- GetFiles ---

func TestGetFilesSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{
		filesResp: &FilesListResponse{
			FolderName: "hmmwv",
			Files:      []FileResponse{{Name: "hmmwv_up_armor_pmcs.json", BlobPath: "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json"}},
			Count:      1,
		},
	}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/pmcs-sbs/hmmwv/files", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestGetFilesEmptyFolder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{filesResp: &FilesListResponse{FolderName: "empty", Files: []FileResponse{}, Count: 0}}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/pmcs-sbs/empty/files", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestGetFilesServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{filesErr: ErrBlobListFailed}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/pmcs-sbs/hmmwv/files", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

// --- GetFileContent ---

func TestGetFileContentSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{contentResp: json.RawMessage(`{"key":"value"}`)}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/pmcs-sbs/content?blob_path=pmcs_sbs/hmmwv/file.json", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestGetFileContentMissingBlobPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/pmcs-sbs/content", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestGetFileContentNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{contentErr: ErrFileNotFound}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/pmcs-sbs/content?blob_path=pmcs_sbs/hmmwv/missing.json", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestGetFileContentInvalidPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{contentErr: ErrInvalidBlobPath}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/pmcs-sbs/content?blob_path=pmcs/bad.json", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestGetFileContentInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{contentErr: ErrInvalidJSON}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/pmcs-sbs/content?blob_path=pmcs_sbs/hmmwv/corrupt.json", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestGetFileContentReadFailed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{contentErr: errors.New("network error")}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/pmcs-sbs/content?blob_path=pmcs_sbs/hmmwv/file.json", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestGetFilesPassesFolderName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &capturingStub{
		serviceStub: serviceStub{filesResp: &FilesListResponse{FolderName: "hmmwv", Files: []FileResponse{}, Count: 0}},
	}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/pmcs-sbs/hmmwv/files", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "hmmwv", stub.capturedFolder)
}

func TestGetFileContentPassesBlobPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &capturingStub{
		serviceStub: serviceStub{contentResp: json.RawMessage(`{}`)},
	}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/pmcs-sbs/content?blob_path=pmcs_sbs/hmmwv/file.json", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "pmcs_sbs/hmmwv/file.json", stub.capturedBlobPath)
}
