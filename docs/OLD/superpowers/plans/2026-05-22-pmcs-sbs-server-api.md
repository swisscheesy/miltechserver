# PMCS SBS Server API — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a new `api/library/pmcs_sbs/` sub-package that exposes three public endpoints for navigating and fetching PMCS Step-by-Step JSON files stored in Azure Blob Storage.

**Architecture:** New sibling sub-package under `api/library/` following the exact `ps_mag` pattern — own Handler, Service interface, ServiceImpl, response types, and errors. One line added to `api/library/route.go` to register routes. No other existing files are touched.

**Tech Stack:** Go 1.21+, Gin, Azure Blob Storage SDK (`azblob`), `encoding/json`, `io.ReadAll`, `path.Clean`, `slog`, `testify/require`

**Spec:** `docs/superpowers/specs/2026-05-22-pmcs-sbs-server-api-design.md`

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `api/library/pmcs_sbs/errors.go` | Create | Sentinel errors |
| `api/library/pmcs_sbs/response.go` | Create | Response types (FolderResponse, FileResponse, etc.) |
| `api/library/pmcs_sbs/service.go` | Create | Service interface |
| `api/library/pmcs_sbs/route_test.go` | Create | HTTP handler tests via httptest + serviceStub |
| `api/library/pmcs_sbs/route.go` | Create | Handler struct, RegisterHandlers, HTTP handlers |
| `api/library/pmcs_sbs/service_impl_test.go` | Create | Pure function and validation unit tests |
| `api/library/pmcs_sbs/service_impl.go` | Create | ServiceImpl — Azure blob calls, content proxy |
| `api/library/route.go` | Modify | Add `pmcs_sbs.RegisterHandlers(publicGroup, deps.BlobClient)` |

---

## Task 1: Foundation — errors, response types, service interface

These are pure type definitions. The handler tests (Task 2) will fail to compile without them, which is the TDD failure gate.

**Files:**
- Create: `api/library/pmcs_sbs/errors.go`
- Create: `api/library/pmcs_sbs/response.go`
- Create: `api/library/pmcs_sbs/service.go`

- [ ] **Step 1: Create `api/library/pmcs_sbs/errors.go`**

```go
package pmcs_sbs

import "errors"

var (
	ErrEmptyFolderName = errors.New("folder name cannot be empty")
	ErrEmptyBlobPath   = errors.New("blob path cannot be empty")
	ErrInvalidBlobPath = errors.New("invalid blob path: must start with pmcs_sbs/")
	ErrInvalidFileType = errors.New("invalid file type: only JSON files are accessible")
	ErrFileNotFound    = errors.New("file not found")
	ErrBlobListFailed  = errors.New("failed to list blobs")
	ErrBlobReadFailed  = errors.New("failed to read blob content")
	ErrInvalidJSON     = errors.New("blob content is not valid JSON")
)
```

- [ ] **Step 2: Create `api/library/pmcs_sbs/response.go`**

```go
package pmcs_sbs

// FolderResponse represents a top-level folder in the PMCS SBS library.
type FolderResponse struct {
	Name        string `json:"name"`
	FullPath    string `json:"full_path"`
	DisplayName string `json:"display_name"`
}

// FoldersListResponse is the response for listing available PMCS SBS folders.
type FoldersListResponse struct {
	Folders []FolderResponse `json:"folders"`
	Count   int              `json:"count"`
}

// FileResponse represents a JSON file in a PMCS SBS folder.
type FileResponse struct {
	Name         string `json:"name"`
	BlobPath     string `json:"blob_path"`
	SizeBytes    int64  `json:"size_bytes"`
	LastModified string `json:"last_modified"`
}

// FilesListResponse is the response for listing files in a PMCS SBS folder.
type FilesListResponse struct {
	FolderName string         `json:"folder_name"`
	Files      []FileResponse `json:"files"`
	Count      int            `json:"count"`
}
```

- [ ] **Step 3: Create `api/library/pmcs_sbs/service.go`**

```go
package pmcs_sbs

import (
	"context"
	"encoding/json"
)

// Service provides access to PMCS Step-by-Step JSON files in Azure Blob Storage.
type Service interface {
	// GetFolders returns all top-level folders under pmcs_sbs/.
	GetFolders() (*FoldersListResponse, error)

	// GetFiles returns all JSON files within the given folder.
	// Returns an empty slice (not an error) if the folder has no JSON files.
	GetFiles(folderName string) (*FilesListResponse, error)

	// GetFileContent fetches a JSON blob and returns its raw content.
	// ctx should be the request context so Azure calls are cancelled on client disconnect.
	GetFileContent(ctx context.Context, blobPath string) (json.RawMessage, error)
}
```

- [ ] **Step 4: Verify the package compiles**

```bash
cd /Users/swisscheese/projects/miltechserver && go build ./api/library/pmcs_sbs/...
```

Expected: clean build (no output).

- [ ] **Step 5: Commit**

```bash
git add api/library/pmcs_sbs/errors.go api/library/pmcs_sbs/response.go api/library/pmcs_sbs/service.go
git commit -m "feat(pmcs-sbs): add foundation types — errors, responses, service interface"
```

---

## Task 2: Write failing handler tests

Write all handler tests before implementing the handler. Tests will fail to compile because `registerHandlers` and `Handler` don't exist yet — that is the expected TDD failure state.

**Files:**
- Create: `api/library/pmcs_sbs/route_test.go`

- [ ] **Step 1: Create `api/library/pmcs_sbs/route_test.go`**

```go
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

func (s *serviceStub) GetFolders() (*FoldersListResponse, error) {
	return s.foldersResp, s.foldersErr
}

func (s *serviceStub) GetFiles(_ string) (*FilesListResponse, error) {
	return s.filesResp, s.filesErr
}

func (s *serviceStub) GetFileContent(_ context.Context, _ string) (json.RawMessage, error) {
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
```

- [ ] **Step 2: Run tests to confirm they fail to compile**

```bash
cd /Users/swisscheese/projects/miltechserver && go test ./api/library/pmcs_sbs/... 2>&1 | head -20
```

Expected: compile error referencing `registerHandlers` or `Handler` undefined. Any other error is unexpected — investigate before proceeding.

---

## Task 3: Implement the handler

Write `route.go` to make the tests from Task 2 pass.

**Files:**
- Create: `api/library/pmcs_sbs/route.go`

- [ ] **Step 1: Create `api/library/pmcs_sbs/route.go`**

```go
package pmcs_sbs

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"

	"miltechserver/api/middleware"
	"miltechserver/api/response"
)

// Handler holds the pmcs_sbs service dependency.
type Handler struct {
	service Service
}

// RegisterHandlers wires pmcs_sbs routes into the public router group.
// Called from api/library/route.go.
func RegisterHandlers(publicGroup *gin.RouterGroup, blobClient *azblob.Client) {
	svc := NewService(blobClient)
	registerHandlers(publicGroup, svc)
}

// registerHandlers is the internal wiring function used directly by tests.
func registerHandlers(publicGroup *gin.RouterGroup, svc Service) {
	h := Handler{service: svc}
	publicGroup.GET("/library/pmcs-sbs/folders", h.getFolders)
	publicGroup.GET("/library/pmcs-sbs/:folder/files", h.getFiles)
	// Rate-limited: each IP is allowed a burst of 10 requests, sustained at 2 req/s.
	publicGroup.GET("/library/pmcs-sbs/content", middleware.RateLimiter(), h.getFileContent)
}

// getFolders returns all top-level folders in the PMCS SBS library.
// GET /library/pmcs-sbs/folders
func (h *Handler) getFolders(c *gin.Context) {
	slog.Info("GetPMCSSBSFolders endpoint called")

	folders, err := h.service.GetFolders()
	if err != nil {
		slog.Error("Failed to retrieve PMCS SBS folders", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve PMCS SBS folders",
			"details": err.Error(),
		})
		return
	}

	slog.Info("Successfully retrieved PMCS SBS folders", "count", folders.Count)
	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: folders})
}

// getFiles returns all JSON files in a specific PMCS SBS folder.
// GET /library/pmcs-sbs/:folder/files
func (h *Handler) getFiles(c *gin.Context) {
	folderName := c.Param("folder")

	slog.Info("GetPMCSSBSFiles endpoint called", "folder", folderName)

	if strings.TrimSpace(folderName) == "" {
		slog.Warn("GetPMCSSBSFiles called with empty folder name")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Folder name is required",
		})
		return
	}

	files, err := h.service.GetFiles(folderName)
	if err != nil {
		slog.Error("Failed to retrieve PMCS SBS files", "error", err, "folder", folderName)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve PMCS SBS files",
			"details": err.Error(),
		})
		return
	}

	slog.Info("Successfully retrieved PMCS SBS files", "count", files.Count, "folder", folderName)
	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: files})
}

// getFileContent fetches a JSON blob from Azure and returns its raw content.
// GET /library/pmcs-sbs/content?blob_path=pmcs_sbs/hmmwv/file.json
func (h *Handler) getFileContent(c *gin.Context) {
	blobPath := c.Query("blob_path")

	slog.Info("GetPMCSSBSFileContent endpoint called", "blobPath", blobPath)

	if strings.TrimSpace(blobPath) == "" {
		slog.Warn("GetPMCSSBSFileContent called with empty blob_path")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "blob_path query parameter is required",
		})
		return
	}

	content, err := h.service.GetFileContent(c.Request.Context(), blobPath)
	if err != nil {
		switch {
		case errors.Is(err, ErrFileNotFound):
			slog.Warn("PMCS SBS file not found", "blobPath", blobPath, "error", err)
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "File not found",
				"details": "The requested file does not exist or is not accessible",
			})
		case errors.Is(err, ErrEmptyBlobPath), errors.Is(err, ErrInvalidBlobPath), errors.Is(err, ErrInvalidFileType):
			slog.Warn("Invalid blob path for PMCS SBS content", "blobPath", blobPath, "error", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request",
				"details": err.Error(),
			})
		default:
			slog.Error("Failed to retrieve PMCS SBS file content", "error", err, "blobPath", blobPath)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to retrieve file content",
				"details": err.Error(),
			})
		}
		return
	}

	slog.Info("Successfully retrieved PMCS SBS file content", "blobPath", blobPath)
	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: content})
}
```

- [ ] **Step 2: Run handler tests and confirm they pass**

```bash
cd /Users/swisscheese/projects/miltechserver && go test ./api/library/pmcs_sbs/... -v -run "TestGet"
```

Expected: all 11 `TestGet*` tests PASS. If any fail, check the error map in `getFileContent` matches the stub errors.

- [ ] **Step 3: Commit**

```bash
git add api/library/pmcs_sbs/route.go api/library/pmcs_sbs/route_test.go
git commit -m "feat(pmcs-sbs): add handler with three public endpoints"
```

---

## Task 4: Write failing service implementation tests

Tests for the two pure helper functions and `GetFileContent` validation. These fail to compile because `service_impl.go` (which defines `NewService`, `formatDisplayName`, `extractFileName`) doesn't exist yet.

**Files:**
- Create: `api/library/pmcs_sbs/service_impl_test.go`

- [ ] **Step 1: Create `api/library/pmcs_sbs/service_impl_test.go`**

```go
package pmcs_sbs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatDisplayName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"lowercase word", "hmmwv", "HMMWV"},
		{"hyphen separator", "m2-bradley", "M2 BRADLEY"},
		{"underscore separator", "m2_bradley", "M2 BRADLEY"},
		{"mixed separators", "m1a1-abrams_tank", "M1A1 ABRAMS TANK"},
		{"already uppercase", "HMMWV", "HMMWV"},
		{"empty string", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, formatDisplayName(tc.input))
		})
	}
}

func TestExtractFileName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"filename only", "file.json", "file.json"},
		{"one folder deep", "hmmwv/file.json", "file.json"},
		{"full blob path", "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json", "hmmwv_up_armor_pmcs.json"},
		{"empty string", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, extractFileName(tc.input))
		})
	}
}

// TestGetFileContentValidation calls NewService(nil) because validation runs before
// any Azure call — a nil blobClient never gets touched.
func TestGetFileContentValidation(t *testing.T) {
	svc := NewService(nil)

	_, err := svc.GetFileContent(context.Background(), "")
	require.ErrorIs(t, err, ErrEmptyBlobPath)

	_, err = svc.GetFileContent(context.Background(), "   ")
	require.ErrorIs(t, err, ErrEmptyBlobPath)

	_, err = svc.GetFileContent(context.Background(), "pmcs/some-file.json")
	require.ErrorIs(t, err, ErrInvalidBlobPath)

	_, err = svc.GetFileContent(context.Background(), "pmcs_sbs/some-file.pdf")
	require.ErrorIs(t, err, ErrInvalidFileType)

	// path.Clean turns "pmcs_sbs/../secret.json" into "secret.json", failing the prefix check.
	_, err = svc.GetFileContent(context.Background(), "pmcs_sbs/../secret.json")
	require.ErrorIs(t, err, ErrInvalidBlobPath)
}
```

- [ ] **Step 2: Run tests to confirm they fail to compile**

```bash
cd /Users/swisscheese/projects/miltechserver && go test ./api/library/pmcs_sbs/... 2>&1 | head -10
```

Expected: compile error referencing `NewService`, `formatDisplayName`, or `extractFileName` undefined.

---

## Task 5: Implement the service

Write `service_impl.go` to make the tests from Task 4 pass.

**Files:**
- Create: `api/library/pmcs_sbs/service_impl.go`

- [ ] **Step 1: Create `api/library/pmcs_sbs/service_impl.go`**

```go
package pmcs_sbs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
)

const (
	LibraryContainerName = "library"
	PMCSSBSPrefix        = "pmcs_sbs/"
)

// ServiceImpl holds the Azure blob client for all blob operations.
type ServiceImpl struct {
	blobClient *azblob.Client
}

// NewService creates a Service backed by the given Azure blob client.
func NewService(blobClient *azblob.Client) Service {
	return &ServiceImpl{blobClient: blobClient}
}

// GetFolders retrieves all top-level folders from pmcs_sbs/ in Azure Blob Storage.
func (s *ServiceImpl) GetFolders() (*FoldersListResponse, error) {
	ctx := context.Background()

	slog.Info("Fetching PMCS SBS folders from Azure Blob Storage",
		"container", LibraryContainerName,
		"prefix", PMCSSBSPrefix)

	containerClient := s.blobClient.ServiceClient().NewContainerClient(LibraryContainerName)
	prefix := PMCSSBSPrefix
	pager := containerClient.NewListBlobsHierarchyPager(
		"/",
		&container.ListBlobsHierarchyOptions{Prefix: &prefix},
	)

	folders := []FolderResponse{}

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			slog.Error("Failed to list PMCS SBS folders from Azure Blob Storage",
				"error", err,
				"container", LibraryContainerName,
				"prefix", PMCSSBSPrefix)
			return nil, fmt.Errorf("%w: %v", ErrBlobListFailed, err)
		}

		for _, p := range page.Segment.BlobPrefixes {
			if p.Name == nil {
				continue
			}
			fullPath := *p.Name
			folderName := strings.TrimSuffix(strings.TrimPrefix(fullPath, PMCSSBSPrefix), "/")
			if folderName == "" {
				continue
			}
			folders = append(folders, FolderResponse{
				Name:        folderName,
				FullPath:    fullPath,
				DisplayName: formatDisplayName(folderName),
			})
		}
	}

	slog.Info("Successfully fetched PMCS SBS folders",
		"count", len(folders),
		"container", LibraryContainerName)

	return &FoldersListResponse{Folders: folders, Count: len(folders)}, nil
}

// GetFiles retrieves all JSON files from a specific folder in pmcs_sbs/.
func (s *ServiceImpl) GetFiles(folderName string) (*FilesListResponse, error) {
	ctx := context.Background()

	if strings.TrimSpace(folderName) == "" {
		return nil, ErrEmptyFolderName
	}

	folderPrefix := fmt.Sprintf("%s%s/", PMCSSBSPrefix, folderName)

	slog.Info("Fetching PMCS SBS files from Azure Blob Storage",
		"container", LibraryContainerName,
		"folderPrefix", folderPrefix)

	containerClient := s.blobClient.ServiceClient().NewContainerClient(LibraryContainerName)
	pager := containerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Prefix: &folderPrefix,
	})

	files := []FileResponse{}

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			slog.Error("Failed to list PMCS SBS files from Azure Blob Storage",
				"error", err,
				"container", LibraryContainerName,
				"folderPrefix", folderPrefix)
			return nil, fmt.Errorf("%w: %v", ErrBlobListFailed, err)
		}

		for _, blob := range page.Segment.BlobItems {
			if blob.Name == nil {
				continue
			}
			blobPath := *blob.Name
			if !strings.HasSuffix(strings.ToLower(blobPath), ".json") {
				slog.Debug("Skipping non-JSON file", "blobPath", blobPath)
				continue
			}

			var sizeBytes int64
			if blob.Properties != nil && blob.Properties.ContentLength != nil {
				sizeBytes = *blob.Properties.ContentLength
			}
			var lastModified string
			if blob.Properties != nil && blob.Properties.LastModified != nil {
				lastModified = blob.Properties.LastModified.Format(time.RFC3339)
			}

			files = append(files, FileResponse{
				Name:         extractFileName(blobPath),
				BlobPath:     blobPath,
				SizeBytes:    sizeBytes,
				LastModified: lastModified,
			})
		}
	}

	slog.Info("Successfully fetched PMCS SBS files",
		"count", len(files),
		"folderName", folderName,
		"container", LibraryContainerName)

	return &FilesListResponse{FolderName: folderName, Files: files, Count: len(files)}, nil
}

// GetFileContent downloads a JSON blob from Azure and returns its raw content.
// ctx should be the request context so the Azure DownloadStream call is cancelled on client disconnect.
func (s *ServiceImpl) GetFileContent(ctx context.Context, blobPath string) (json.RawMessage, error) {
	if strings.TrimSpace(blobPath) == "" {
		return nil, ErrEmptyBlobPath
	}

	// Sanitise path to prevent directory traversal (e.g. "pmcs_sbs/../secret.json").
	blobPath = path.Clean(blobPath)

	if !strings.HasPrefix(blobPath, PMCSSBSPrefix) {
		return nil, ErrInvalidBlobPath
	}
	if !strings.HasSuffix(strings.ToLower(blobPath), ".json") {
		return nil, ErrInvalidFileType
	}

	slog.Info("Downloading PMCS SBS file content from Azure Blob Storage",
		"container", LibraryContainerName,
		"blobPath", blobPath)

	blobClient := s.blobClient.ServiceClient().NewContainerClient(LibraryContainerName).NewBlobClient(blobPath)
	downloadResponse, err := blobClient.DownloadStream(ctx, nil)
	if err != nil {
		slog.Error("Failed to download PMCS SBS file", "error", err, "blobPath", blobPath)
		return nil, fmt.Errorf("%w: %v", ErrFileNotFound, err)
	}
	defer downloadResponse.Body.Close()

	data, err := io.ReadAll(downloadResponse.Body)
	if err != nil {
		slog.Error("Failed to read PMCS SBS file content", "error", err, "blobPath", blobPath)
		return nil, fmt.Errorf("%w: %v", ErrBlobReadFailed, err)
	}

	if !json.Valid(data) {
		slog.Error("PMCS SBS blob contains invalid JSON", "blobPath", blobPath, "size", len(data))
		return nil, ErrInvalidJSON
	}

	slog.Info("Successfully downloaded PMCS SBS file content",
		"blobPath", blobPath,
		"size", len(data))

	return json.RawMessage(data), nil
}

// formatDisplayName converts folder names to human-readable display names.
// Examples: "hmmwv" -> "HMMWV", "m2-bradley" -> "M2 BRADLEY", "m2_bradley" -> "M2 BRADLEY"
func formatDisplayName(name string) string {
	display := strings.ToUpper(name)
	display = strings.ReplaceAll(display, "-", " ")
	display = strings.ReplaceAll(display, "_", " ")
	return display
}

// extractFileName returns the filename from a blob path.
// Example: "pmcs_sbs/hmmwv/file.json" -> "file.json"
func extractFileName(blobPath string) string {
	parts := strings.Split(blobPath, "/")
	if len(parts) == 0 {
		return blobPath
	}
	return parts[len(parts)-1]
}
```

- [ ] **Step 2: Run all tests in the package and confirm they pass**

```bash
cd /Users/swisscheese/projects/miltechserver && go test ./api/library/pmcs_sbs/... -v
```

Expected: all tests PASS. The `TestGetFileContentValidation` tests call `NewService(nil)` — validation fires before any Azure call, so the nil client is never dereferenced.

- [ ] **Step 3: Commit**

```bash
git add api/library/pmcs_sbs/service_impl.go api/library/pmcs_sbs/service_impl_test.go
git commit -m "feat(pmcs-sbs): implement service with Azure blob folder/file listing and content proxy"
```

---

## Task 6: Wire into the library router

Add one import and one call to the existing `api/library/route.go`. No logic changes.

**Files:**
- Modify: `api/library/route.go`

- [ ] **Step 1: Add the import for `pmcs_sbs` to `api/library/route.go`**

In the import block, add `"miltechserver/api/library/pmcs_sbs"` alongside the existing `ps_mag` import:

```go
import (
    "database/sql"
    "errors"
    "log/slog"
    "net/http"
    "strings"

    "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
    "github.com/gin-gonic/gin"

    "miltechserver/api/analytics"
    "miltechserver/api/library/pmcs_sbs"
    "miltechserver/api/library/ps_mag"
    "miltechserver/api/middleware"
    "miltechserver/api/response"
    "miltechserver/bootstrap"
)
```

- [ ] **Step 2: Add the `RegisterHandlers` call in `RegisterRoutes`**

The current `RegisterRoutes` function at line 31 in `api/library/route.go`:

```go
func RegisterRoutes(deps Dependencies, publicGroup, authGroup *gin.RouterGroup) {
    svc := NewService(deps.BlobClient, deps.Env, deps.Analytics)
    registerHandlers(publicGroup, authGroup, svc)
    ps_mag.RegisterHandlers(publicGroup, deps.BlobClient, deps.DB, deps.Analytics)
}
```

Change to:

```go
func RegisterRoutes(deps Dependencies, publicGroup, authGroup *gin.RouterGroup) {
    svc := NewService(deps.BlobClient, deps.Env, deps.Analytics)
    registerHandlers(publicGroup, authGroup, svc)
    ps_mag.RegisterHandlers(publicGroup, deps.BlobClient, deps.DB, deps.Analytics)
    pmcs_sbs.RegisterHandlers(publicGroup, deps.BlobClient)
}
```

- [ ] **Step 3: Build the whole server to confirm no compile errors**

```bash
cd /Users/swisscheese/projects/miltechserver && go build ./...
```

Expected: clean build (no output).

- [ ] **Step 4: Run the full library test suite**

```bash
cd /Users/swisscheese/projects/miltechserver && go test ./api/library/... -v
```

Expected: all tests across `library`, `library/ps_mag`, and `library/pmcs_sbs` PASS.

- [ ] **Step 5: Commit**

```bash
git add api/library/route.go
git commit -m "feat(pmcs-sbs): wire pmcs_sbs routes into library router"
```

---

## Self-Review Checklist (completed inline)

**Spec coverage:**
- [x] `GET /library/pmcs-sbs/folders` → Task 1 (types) + Task 2/3 (handler + tests)
- [x] `GET /library/pmcs-sbs/:folder/files` → Task 1 (types) + Task 2/3 (handler + tests)
- [x] `GET /library/pmcs-sbs/content` (rate-limited) → Task 1 (types) + Task 2/3 (handler + tests)
- [x] `api/library/pmcs_sbs/` sub-package → Tasks 1–5
- [x] Sentinel errors → Task 1
- [x] Response types → Task 1
- [x] Service interface → Task 1
- [x] `formatDisplayName` and `extractFileName` helpers → Task 5 (impl) + Task 4 (tests)
- [x] Path validation and traversal prevention → Task 5 (impl) + Task 4 (tests)
- [x] `json.Valid` check → Task 5 (impl)
- [x] Wire into library router → Task 6

**Type consistency:** All types, method names, and error sentinels defined in Task 1 are used consistently across Tasks 2–6. `serviceStub` in route_test.go implements the `Service` interface defined in service.go. `NewService` is defined in Task 5 and tested in Task 4 — Task 4 fails to compile without Task 5, which is intentional TDD ordering.

**No placeholders:** All steps contain complete, runnable code. All commands include expected output.
