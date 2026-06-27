package pmcs_sbs

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
	imageResp   *ImageDownload
	imageErr    error
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
	capturedFolder        string
	capturedBlobPath      string
	capturedImageBlobPath string
	capturedImageName     string
}

func (s *capturingStub) GetFiles(_ context.Context, folder string) (*FilesListResponse, error) {
	s.capturedFolder = folder
	return s.filesResp, s.filesErr
}

func (s *capturingStub) GetFileContent(_ context.Context, blobPath string) (json.RawMessage, error) {
	s.capturedBlobPath = blobPath
	return s.contentResp, s.contentErr
}

func (s *capturingStub) GetImage(_ context.Context, blobPath string, imageName string) (*ImageDownload, error) {
	s.capturedImageBlobPath = blobPath
	s.capturedImageName = imageName
	return s.imageResp, s.imageErr
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

// --- GetImage ---

func newImageRequest(target string, remoteAddr string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.RemoteAddr = remoteAddr
	return req
}

func TestGetImageSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &capturingStub{
		serviceStub: serviceStub{
			imageResp: &ImageDownload{
				Body:          io.NopCloser(strings.NewReader("png-bytes")),
				ContentLength: int64(len("png-bytes")),
				ContentType:   "image/png",
				FileName:      "Before_12.png",
				BlobPath:      "pmcs_sbs/hmmwv/images/file/Before_12.png",
			},
		},
	}
	registerHandlers(router.Group("/api/v1"), stub)

	req := newImageRequest("/api/v1/library/pmcs-sbs/image?blob_path=pmcs_sbs/hmmwv/file.json&image_name=Before_12", "198.51.100.10:1234")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "image/png", resp.Header().Get("Content-Type"))
	require.Equal(t, `inline; filename="Before_12.png"`, resp.Header().Get("Content-Disposition"))
	require.Equal(t, "public, max-age=86400", resp.Header().Get("Cache-Control"))
	require.Equal(t, "9", resp.Header().Get("Content-Length"))
	require.Equal(t, "png-bytes", resp.Body.String())
}

func TestGetImageMissingBlobPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &capturingStub{}
	registerHandlers(router.Group("/api/v1"), stub)

	req := newImageRequest("/api/v1/library/pmcs-sbs/image?image_name=Before_12", "198.51.100.11:1234")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
	require.JSONEq(t, `{"error":"blob_path query parameter is required"}`, resp.Body.String())
}

func TestGetImageMissingImageName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &capturingStub{}
	registerHandlers(router.Group("/api/v1"), stub)

	req := newImageRequest("/api/v1/library/pmcs-sbs/image?blob_path=pmcs_sbs/hmmwv/file.json", "198.51.100.12:1234")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
	require.JSONEq(t, `{"error":"image_name query parameter is required"}`, resp.Body.String())
}

func TestGetImageInvalidRequestErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "invalid blob path", err: ErrInvalidBlobPath},
		{name: "invalid image name", err: ErrInvalidImageName},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			stub := &capturingStub{
				serviceStub: serviceStub{imageErr: tc.err},
			}
			registerHandlers(router.Group("/api/v1"), stub)

			req := newImageRequest("/api/v1/library/pmcs-sbs/image?blob_path=pmcs_sbs/hmmwv/file.json&image_name=Before_12", "198.51.100.13:1234")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			require.Equal(t, http.StatusBadRequest, resp.Code)
			require.JSONEq(t, `{"error":"Invalid request","details":"`+tc.err.Error()+`"}`, resp.Body.String())
		})
	}
}

func TestGetImageNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &capturingStub{
		serviceStub: serviceStub{imageErr: ErrFileNotFound},
	}
	registerHandlers(router.Group("/api/v1"), stub)

	req := newImageRequest("/api/v1/library/pmcs-sbs/image?blob_path=pmcs_sbs/hmmwv/file.json&image_name=Before_12", "198.51.100.14:1234")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusNotFound, resp.Code)
	require.JSONEq(t, `{"error":"Image not found","details":"The requested image does not exist or is not accessible"}`, resp.Body.String())
}

func TestGetImageGenericError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &capturingStub{
		serviceStub: serviceStub{imageErr: errors.New("network error")},
	}
	registerHandlers(router.Group("/api/v1"), stub)

	req := newImageRequest("/api/v1/library/pmcs-sbs/image?blob_path=pmcs_sbs/hmmwv/file.json&image_name=Before_12", "198.51.100.15:1234")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusInternalServerError, resp.Code)
	require.JSONEq(t, `{"error":"Failed to retrieve image"}`, resp.Body.String())
}

func TestGetImageNilDownloadResponseReturnsGenericError(t *testing.T) {
	tests := []struct {
		name      string
		imageResp *ImageDownload
	}{
		{name: "nil image response", imageResp: nil},
		{name: "nil image body", imageResp: &ImageDownload{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			stub := &capturingStub{
				serviceStub: serviceStub{imageResp: tc.imageResp},
			}
			registerHandlers(router.Group("/api/v1"), stub)

			req := newImageRequest("/api/v1/library/pmcs-sbs/image?blob_path=pmcs_sbs/hmmwv/file.json&image_name=Before_12", "198.51.100.16:1234")
			resp := httptest.NewRecorder()

			require.NotPanics(t, func() {
				router.ServeHTTP(resp, req)
			})
			require.Equal(t, http.StatusInternalServerError, resp.Code)
			require.JSONEq(t, `{"error":"Failed to retrieve image"}`, resp.Body.String())
		})
	}
}

func TestGetImagePassesBlobPathAndImageName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &capturingStub{
		serviceStub: serviceStub{
			imageResp: &ImageDownload{
				Body:          io.NopCloser(strings.NewReader("png-bytes")),
				ContentLength: int64(len("png-bytes")),
				ContentType:   "image/png",
				FileName:      "Before_12.png",
			},
		},
	}
	registerHandlers(router.Group("/api/v1"), stub)

	req := newImageRequest("/api/v1/library/pmcs-sbs/image?blob_path=%20pmcs_sbs%2Fhmmwv%2Ffile.json%20&image_name=%20Before_12%20", "198.51.100.16:1234")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, " pmcs_sbs/hmmwv/file.json ", stub.capturedImageBlobPath)
	require.Equal(t, " Before_12 ", stub.capturedImageName)
}
