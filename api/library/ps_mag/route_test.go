package ps_mag

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
	listResp    *PSMagIssuesResponse
	listErr     error
	downloadErr error
	searchResp  *PSMagSearchResponse
	searchErr   error
}

func (s *serviceStub) ListIssues(page int, order string, year *int, issueNumber *int) (*PSMagIssuesResponse, error) {
	return s.listResp, s.listErr
}

func (s *serviceStub) GenerateDownloadURL(_ context.Context, blobPath string) (*DownloadURLResponse, error) {
	if s.downloadErr != nil {
		return nil, s.downloadErr
	}
	return &DownloadURLResponse{BlobPath: blobPath, DownloadURL: "https://example.com/sas", ExpiresAt: "2099-01-01T00:00:00Z"}, nil
}

func (s *serviceStub) SearchSummaries(query string, page int) (*PSMagSearchResponse, error) {
	return s.searchResp, s.searchErr
}

func TestListIssuesSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{
		listResp: &PSMagIssuesResponse{
			Issues:     []PSMagIssueResponse{},
			Count:      0,
			TotalCount: 0,
			Page:       1,
			TotalPages: 1,
			Order:      "asc",
		},
	}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/ps-mag/issues", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestListIssuesDefaultParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{
		listResp: &PSMagIssuesResponse{Issues: []PSMagIssueResponse{}, Count: 0, TotalCount: 0, Page: 1, TotalPages: 1, Order: "asc"},
	}
	registerHandlers(router.Group("/api/v1"), stub)

	// No params — should default to page=1, order=asc
	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/ps-mag/issues", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestListIssuesInvalidPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/ps-mag/issues?page=0", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestListIssuesNonNumericPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/ps-mag/issues?page=abc", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestListIssuesInvalidOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/ps-mag/issues?order=sideways", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestListIssuesInvalidYearParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/ps-mag/issues?year=notanumber", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestListIssuesInvalidIssueParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/ps-mag/issues?issue=notanumber", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestDownloadSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{} // no downloadErr = stub returns success automatically
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/ps-mag/download?blob_path=ps-mag/PS_Magazine_Issue_495_February_1994.pdf", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestDownloadMissingBlobPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{downloadErr: ErrEmptyBlobPath}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/ps-mag/download", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestDownloadNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{downloadErr: ErrIssueNotFound}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/ps-mag/download?blob_path=ps-mag/missing.pdf", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestDownloadInvalidPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{downloadErr: ErrInvalidBlobPath}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/ps-mag/download?blob_path=pmcs/bad.pdf", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestDownloadServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{downloadErr: ErrSASGenFailed}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/ps-mag/download?blob_path=ps-mag/test.pdf", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestSearchSummariesMissingQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/ps-mag/search", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestSearchSummariesQueryTooShort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/ps-mag/search?q=ab", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestSearchSummariesInvalidPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/ps-mag/search?q=oil&page=bad", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestSearchSummariesPageZero(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/ps-mag/search?q=oil&page=0", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestSearchSummariesSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{
		searchResp: &PSMagSearchResponse{
			Results: []PSMagSearchResult{
				{
					FileName:      "PS_Magazine_Issue_495_February_1994.pdf",
					MatchingLines: []string{"Check the oil level."},
				},
			},
			Count:      1,
			TotalCount: 1,
			Page:       1,
			TotalPages: 1,
			Query:      "oil",
		},
	}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/ps-mag/search?q=oil", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestSearchSummariesServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{searchErr: errors.New("db failure")}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/ps-mag/search?q=oil", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusInternalServerError, resp.Code)
}
