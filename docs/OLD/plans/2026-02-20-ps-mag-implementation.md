# PS Magazine Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a PS Magazine browsing and download sub-domain (`api/library/ps_mag/`) to the library bounded context, enabling paginated listing with optional filters and 1-hour SAS download URL generation.

**Architecture:** The `ps_mag` package lives as a sub-package of `api/library/`, following the same pattern as `api/item_lookup/lin/` and siblings. All metadata is parsed from blob filenames in memory — no database required. The parent `library/route.go` wires the sub-package in with a single `RegisterHandlers` call.

**Tech Stack:** Go 1.21+, Gin, Azure Blob Storage SDK (`azblob`), `regexp`, `sort`, `strconv`, `testify/require`

---

## Reference

Design document: `docs/plans/2026-02-20-ps-mag-design.md`

Filename convention: `PS_Magazine_Issue_###_Month_Year.pdf`
Example: `PS_Magazine_Issue_495_February_1994.pdf`

Azure container: `library`, prefix: `ps-mag/`

Existing pattern to mirror: `api/library/` (route.go, service.go, service_impl.go, response.go, errors.go, tests)

Run all ps_mag tests: `go test ./api/library/ps_mag/ -v`
Run all library tests: `go test ./api/library/... -v`

---

## Task 1: Create Foundation Files

These are type/error declarations — no logic, no tests needed.

**Files:**
- Create: `api/library/ps_mag/errors.go`
- Create: `api/library/ps_mag/response.go`
- Create: `api/library/ps_mag/service.go`

**Step 1: Create `api/library/ps_mag/errors.go`**

```go
package ps_mag

import "errors"

var (
	ErrEmptyBlobPath   = errors.New("blob path cannot be empty")
	ErrInvalidBlobPath = errors.New("invalid blob path: must start with ps-mag/")
	ErrInvalidFileType = errors.New("invalid file type: only PDF files can be downloaded")
	ErrIssueNotFound   = errors.New("issue not found")
	ErrBlobListFailed  = errors.New("failed to list blobs from Azure")
	ErrSASGenFailed    = errors.New("failed to generate download URL")
	ErrInvalidPage     = errors.New("page must be greater than 0")
	ErrInvalidOrder    = errors.New("order must be 'asc' or 'desc'")
)
```

**Step 2: Create `api/library/ps_mag/response.go`**

```go
package ps_mag

// PSMagIssueResponse represents a single parsed PS Magazine issue.
type PSMagIssueResponse struct {
	Name         string `json:"name"`          // e.g. "PS_Magazine_Issue_495_February_1994.pdf"
	BlobPath     string `json:"blob_path"`     // e.g. "ps-mag/PS_Magazine_Issue_495_February_1994.pdf"
	IssueNumber  int    `json:"issue_number"`  // 495
	Month        string `json:"month"`         // "February"
	Year         int    `json:"year"`          // 1994
	SizeBytes    int64  `json:"size_bytes"`
	LastModified string `json:"last_modified"` // RFC3339
}

// PSMagIssuesResponse is the paginated listing response.
type PSMagIssuesResponse struct {
	Issues     []PSMagIssueResponse `json:"issues"`
	Count      int                  `json:"count"`       // items on this page
	TotalCount int                  `json:"total_count"` // total matching issues across all pages
	Page       int                  `json:"page"`
	TotalPages int                  `json:"total_pages"`
	Order      string               `json:"order"`
}

// DownloadURLResponse contains a time-limited SAS URL for a PS Magazine issue.
type DownloadURLResponse struct {
	BlobPath    string `json:"blob_path"`
	DownloadURL string `json:"download_url"`
	ExpiresAt   string `json:"expires_at"` // RFC3339
}
```

**Step 3: Create `api/library/ps_mag/service.go`**

```go
package ps_mag

// Service provides methods for accessing PS Magazine issues from Azure Blob Storage.
type Service interface {
	// ListIssues returns a paginated, optionally filtered list of PS Magazine issues.
	// page is 1-indexed. order must be "asc" or "desc".
	// year and issueNumber are optional — pass nil to skip the filter.
	ListIssues(page int, order string, year *int, issueNumber *int) (*PSMagIssuesResponse, error)

	// GenerateDownloadURL creates a 1-hour SAS URL for a ps-mag blob.
	// blobPath must start with "ps-mag/" and end with ".pdf".
	GenerateDownloadURL(blobPath string) (*DownloadURLResponse, error)
}
```

**Step 4: Verify the package compiles**

```bash
go build ./api/library/ps_mag/
```

Expected: no output (success).

**Step 5: Commit**

```bash
git add api/library/ps_mag/errors.go api/library/ps_mag/response.go api/library/ps_mag/service.go
git commit -m "feat(ps-mag): add foundation types, errors, and service interface"
```

---

## Task 2: TDD — Filename Parsing Pure Functions

`parseIssueFilename` is a pure function — test it with table-driven tests before writing a single line of implementation.

**Files:**
- Create: `api/library/ps_mag/service_impl_test.go`
- Create: `api/library/ps_mag/service_impl.go`

**Step 1: Write the failing tests in `api/library/ps_mag/service_impl_test.go`**

```go
package ps_mag

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseIssueFilename(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantIssue   int
		wantMonth   string
		wantYear    int
		wantOK      bool
	}{
		{
			name:      "standard issue",
			input:     "PS_Magazine_Issue_495_February_1994.pdf",
			wantIssue: 495,
			wantMonth: "February",
			wantYear:  1994,
			wantOK:    true,
		},
		{
			name:      "low issue number",
			input:     "PS_Magazine_Issue_1_January_1951.pdf",
			wantIssue: 1,
			wantMonth: "January",
			wantYear:  1951,
			wantOK:    true,
		},
		{
			name:   "wrong prefix",
			input:  "Magazine_Issue_495_February_1994.pdf",
			wantOK: false,
		},
		{
			name:   "not a pdf",
			input:  "PS_Magazine_Issue_495_February_1994.txt",
			wantOK: false,
		},
		{
			name:   "missing year",
			input:  "PS_Magazine_Issue_495_February.pdf",
			wantOK: false,
		},
		{
			name:   "empty string",
			input:  "",
			wantOK: false,
		},
		{
			name:   "stray file in prefix",
			input:  "README.md",
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issueNum, month, year, ok := parseIssueFilename(tc.input)
			require.Equal(t, tc.wantOK, ok)
			if tc.wantOK {
				require.Equal(t, tc.wantIssue, issueNum)
				require.Equal(t, tc.wantMonth, month)
				require.Equal(t, tc.wantYear, year)
			}
		})
	}
}
```

**Step 2: Run the test to confirm it fails**

```bash
go test ./api/library/ps_mag/ -v -run TestParseIssueFilename
```

Expected: compile error — `parseIssueFilename undefined`.

**Step 3: Create `api/library/ps_mag/service_impl.go` with just the regex and parseIssueFilename**

```go
package ps_mag

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
)

const (
	PSMagContainerName = "library"
	PSMagPrefix        = "ps-mag/"
	PageSize           = 50
)

var issueRegex = regexp.MustCompile(`^PS_Magazine_Issue_(\d+)_([A-Za-z]+)_(\d{4})\.pdf$`)

type ServiceImpl struct {
	blobClient *azblob.Client
	credential *azblob.SharedKeyCredential
}

func NewService(blobClient *azblob.Client, credential *azblob.SharedKeyCredential) Service {
	return &ServiceImpl{
		blobClient: blobClient,
		credential: credential,
	}
}

// parseIssueFilename extracts issue metadata from a PS Magazine filename.
// Returns false if the name does not match the expected convention.
func parseIssueFilename(name string) (issueNumber int, month string, year int, ok bool) {
	matches := issueRegex.FindStringSubmatch(name)
	if matches == nil {
		return 0, "", 0, false
	}
	issueNumber, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, "", 0, false
	}
	month = matches[2]
	year, err = strconv.Atoi(matches[3])
	if err != nil {
		return 0, "", 0, false
	}
	return issueNumber, month, year, true
}

// filterByYear returns only issues matching the given year.
func filterByYear(issues []PSMagIssueResponse, year int) []PSMagIssueResponse {
	out := issues[:0]
	for _, issue := range issues {
		if issue.Year == year {
			out = append(out, issue)
		}
	}
	return out
}

// filterByIssueNumber returns only issues matching the given issue number.
func filterByIssueNumber(issues []PSMagIssueResponse, issueNumber int) []PSMagIssueResponse {
	out := issues[:0]
	for _, issue := range issues {
		if issue.IssueNumber == issueNumber {
			out = append(out, issue)
		}
	}
	return out
}

// sortIssues sorts issues in-place by IssueNumber. order must be "asc" or "desc".
func sortIssues(issues []PSMagIssueResponse, order string) {
	sort.Slice(issues, func(i, j int) bool {
		if order == "asc" {
			return issues[i].IssueNumber < issues[j].IssueNumber
		}
		return issues[i].IssueNumber > issues[j].IssueNumber
	})
}

// paginateIssues returns the page window and total page count.
// page is 1-indexed. Returns an empty slice (never nil) when page is beyond the end.
func paginateIssues(issues []PSMagIssueResponse, page, pageSize int) (pageItems []PSMagIssueResponse, totalPages int) {
	total := len(issues)
	totalPages = (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}
	start := (page - 1) * pageSize
	if start >= total {
		return []PSMagIssueResponse{}, totalPages
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return issues[start:end], totalPages
}

// listAllIssues fetches every blob under ps-mag/ and parses metadata from filenames.
// Blobs that do not match the filename convention are silently skipped.
func (s *ServiceImpl) listAllIssues() ([]PSMagIssueResponse, error) {
	ctx := context.Background()
	containerClient := s.blobClient.ServiceClient().NewContainerClient(PSMagContainerName)
	prefix := PSMagPrefix
	pager := containerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Prefix: &prefix,
	})

	var issues []PSMagIssueResponse

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBlobListFailed, err)
		}
		for _, blob := range page.Segment.BlobItems {
			if blob.Name == nil {
				continue
			}
			blobPath := *blob.Name
			parts := strings.Split(blobPath, "/")
			fileName := parts[len(parts)-1]

			issueNum, month, year, ok := parseIssueFilename(fileName)
			if !ok {
				slog.Debug("Skipping non-matching ps-mag blob", "blobPath", blobPath)
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

			issues = append(issues, PSMagIssueResponse{
				Name:         fileName,
				BlobPath:     blobPath,
				IssueNumber:  issueNum,
				Month:        month,
				Year:         year,
				SizeBytes:    sizeBytes,
				LastModified: lastModified,
			})
		}
	}
	return issues, nil
}

// ListIssues returns a paginated, optionally filtered list of PS Magazine issues.
func (s *ServiceImpl) ListIssues(page int, order string, year *int, issueNumber *int) (*PSMagIssuesResponse, error) {
	if page < 1 {
		return nil, ErrInvalidPage
	}
	order = strings.ToLower(order)
	if order != "asc" && order != "desc" {
		return nil, ErrInvalidOrder
	}

	issues, err := s.listAllIssues()
	if err != nil {
		return nil, err
	}

	if year != nil {
		issues = filterByYear(issues, *year)
	}
	if issueNumber != nil {
		issues = filterByIssueNumber(issues, *issueNumber)
	}

	sortIssues(issues, order)

	pageItems, totalPages := paginateIssues(issues, page, PageSize)

	return &PSMagIssuesResponse{
		Issues:     pageItems,
		Count:      len(pageItems),
		TotalCount: len(issues),
		Page:       page,
		TotalPages: totalPages,
		Order:      order,
	}, nil
}

// GenerateDownloadURL creates a 1-hour SAS URL for a ps-mag blob.
func (s *ServiceImpl) GenerateDownloadURL(blobPath string) (*DownloadURLResponse, error) {
	if strings.TrimSpace(blobPath) == "" {
		return nil, ErrEmptyBlobPath
	}
	if !strings.HasPrefix(blobPath, PSMagPrefix) {
		return nil, ErrInvalidBlobPath
	}
	if !strings.HasSuffix(strings.ToLower(blobPath), ".pdf") {
		return nil, ErrInvalidFileType
	}

	ctx := context.Background()
	blobClient := s.blobClient.ServiceClient().NewContainerClient(PSMagContainerName).NewBlobClient(blobPath)
	_, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		slog.Error("PS Magazine blob not found", "blobPath", blobPath, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrIssueNotFound, err)
	}

	expiryTime := time.Now().UTC().Add(1 * time.Hour)
	permissions := sas.BlobPermissions{Read: true}

	sasQueryParams, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		StartTime:     time.Now().UTC().Add(-5 * time.Minute),
		ExpiryTime:    expiryTime,
		Permissions:   permissions.String(),
		ContainerName: PSMagContainerName,
		BlobName:      blobPath,
	}.SignWithSharedKey(s.credential)
	if err != nil {
		slog.Error("Failed to generate SAS token for PS Magazine", "blobPath", blobPath, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrSASGenFailed, err)
	}

	downloadURL := fmt.Sprintf("%s?%s", blobClient.URL(), sasQueryParams.Encode())

	slog.Info("Generated PS Magazine download URL",
		"blobPath", blobPath,
		"expiresAt", expiryTime.Format(time.RFC3339))

	return &DownloadURLResponse{
		BlobPath:    blobPath,
		DownloadURL: downloadURL,
		ExpiresAt:   expiryTime.Format(time.RFC3339),
	}, nil
}
```

**Step 4: Run the parsing test to confirm it passes**

```bash
go test ./api/library/ps_mag/ -v -run TestParseIssueFilename
```

Expected: `PASS`

**Step 5: Commit**

```bash
git add api/library/ps_mag/service_impl.go api/library/ps_mag/service_impl_test.go
git commit -m "feat(ps-mag): implement filename parsing with table-driven tests"
```

---

## Task 3: TDD — Filter, Sort, Paginate, and Validation

Add tests for the remaining pure functions and `GenerateDownloadURL` validation (which fails before touching Azure so needs no mock).

**Files:**
- Modify: `api/library/ps_mag/service_impl_test.go`

**Step 1: Add tests — append to `service_impl_test.go`**

```go
// buildTestIssues creates a deterministic slice of test issues for use in filter/sort/paginate tests.
func buildTestIssues() []PSMagIssueResponse {
	return []PSMagIssueResponse{
		{IssueNumber: 100, Month: "January", Year: 1960, Name: "PS_Magazine_Issue_100_January_1960.pdf"},
		{IssueNumber: 200, Month: "June", Year: 1970, Name: "PS_Magazine_Issue_200_June_1970.pdf"},
		{IssueNumber: 300, Month: "March", Year: 1970, Name: "PS_Magazine_Issue_300_March_1970.pdf"},
		{IssueNumber: 400, Month: "August", Year: 1980, Name: "PS_Magazine_Issue_400_August_1980.pdf"},
		{IssueNumber: 495, Month: "February", Year: 1994, Name: "PS_Magazine_Issue_495_February_1994.pdf"},
	}
}

func TestFilterByYear(t *testing.T) {
	issues := buildTestIssues()
	result := filterByYear(issues, 1970)
	require.Len(t, result, 2)
	require.Equal(t, 200, result[0].IssueNumber)
	require.Equal(t, 300, result[1].IssueNumber)
}

func TestFilterByYearNoMatch(t *testing.T) {
	issues := buildTestIssues()
	result := filterByYear(issues, 2099)
	require.Empty(t, result)
}

func TestFilterByIssueNumber(t *testing.T) {
	issues := buildTestIssues()
	result := filterByIssueNumber(issues, 495)
	require.Len(t, result, 1)
	require.Equal(t, "February", result[0].Month)
}

func TestFilterByIssueNumberNoMatch(t *testing.T) {
	issues := buildTestIssues()
	result := filterByIssueNumber(issues, 999)
	require.Empty(t, result)
}

func TestSortIssuesASC(t *testing.T) {
	issues := buildTestIssues()
	// shuffle order first
	issues[0], issues[4] = issues[4], issues[0]
	sortIssues(issues, "asc")
	require.Equal(t, 100, issues[0].IssueNumber)
	require.Equal(t, 495, issues[4].IssueNumber)
}

func TestSortIssuesDESC(t *testing.T) {
	issues := buildTestIssues()
	sortIssues(issues, "desc")
	require.Equal(t, 495, issues[0].IssueNumber)
	require.Equal(t, 100, issues[4].IssueNumber)
}

func TestPaginateIssuesPage1(t *testing.T) {
	// Build 75 issues to test pagination boundary
	issues := make([]PSMagIssueResponse, 75)
	for i := range issues {
		issues[i] = PSMagIssueResponse{IssueNumber: i + 1}
	}
	page, totalPages := paginateIssues(issues, 1, 50)
	require.Len(t, page, 50)
	require.Equal(t, 2, totalPages)
	require.Equal(t, 1, page[0].IssueNumber)
}

func TestPaginateIssuesPage2(t *testing.T) {
	issues := make([]PSMagIssueResponse, 75)
	for i := range issues {
		issues[i] = PSMagIssueResponse{IssueNumber: i + 1}
	}
	page, totalPages := paginateIssues(issues, 2, 50)
	require.Len(t, page, 25)
	require.Equal(t, 2, totalPages)
	require.Equal(t, 51, page[0].IssueNumber)
}

func TestPaginateIssuesBeyondEnd(t *testing.T) {
	issues := make([]PSMagIssueResponse, 10)
	page, totalPages := paginateIssues(issues, 99, 50)
	require.Empty(t, page)
	require.Equal(t, 1, totalPages)
}

func TestPaginateIssuesEmpty(t *testing.T) {
	page, totalPages := paginateIssues([]PSMagIssueResponse{}, 1, 50)
	require.Empty(t, page)
	require.Equal(t, 1, totalPages)
}

func TestGenerateDownloadURLValidation(t *testing.T) {
	svc := NewService(nil, nil)

	_, err := svc.GenerateDownloadURL("")
	require.ErrorIs(t, err, ErrEmptyBlobPath)

	_, err = svc.GenerateDownloadURL("   ")
	require.ErrorIs(t, err, ErrEmptyBlobPath)

	_, err = svc.GenerateDownloadURL("pmcs/some-file.pdf")
	require.ErrorIs(t, err, ErrInvalidBlobPath)

	_, err = svc.GenerateDownloadURL("ps-mag/some-file.txt")
	require.ErrorIs(t, err, ErrInvalidFileType)
}

func TestListIssuesValidation(t *testing.T) {
	svc := NewService(nil, nil)

	_, err := svc.ListIssues(0, "asc", nil, nil)
	require.ErrorIs(t, err, ErrInvalidPage)

	_, err = svc.ListIssues(1, "sideways", nil, nil)
	require.ErrorIs(t, err, ErrInvalidOrder)
}
```

**Step 2: Run the new tests**

```bash
go test ./api/library/ps_mag/ -v -run "TestFilter|TestSort|TestPaginate|TestGenerateDownloadURLValidation|TestListIssuesValidation"
```

Expected: `PASS` for all — the implementations already exist from Task 2.

**Step 3: Commit**

```bash
git add api/library/ps_mag/service_impl_test.go
git commit -m "test(ps-mag): add filter, sort, paginate, and validation unit tests"
```

---

## Task 4: TDD — Route Handlers

Test the HTTP layer with a `serviceStub`. The stub implements the `Service` interface so no Azure dependency is needed.

**Files:**
- Create: `api/library/ps_mag/route.go`
- Create: `api/library/ps_mag/route_test.go`

**Step 1: Write failing tests in `api/library/ps_mag/route_test.go`**

```go
package ps_mag

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type serviceStub struct {
	listResp    *PSMagIssuesResponse
	listErr     error
	downloadResp *DownloadURLResponse
	downloadErr  error
}

func (s *serviceStub) ListIssues(page int, order string, year *int, issueNumber *int) (*PSMagIssuesResponse, error) {
	return s.listResp, s.listErr
}

func (s *serviceStub) GenerateDownloadURL(blobPath string) (*DownloadURLResponse, error) {
	return s.downloadResp, s.downloadErr
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
	stub := &serviceStub{
		downloadResp: &DownloadURLResponse{
			BlobPath:    "ps-mag/PS_Magazine_Issue_495_February_1994.pdf",
			DownloadURL: "https://example.com/sas",
			ExpiresAt:   "2026-02-20T12:00:00Z",
		},
	}
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
```

**Step 2: Run to confirm compile failure**

```bash
go test ./api/library/ps_mag/ -v -run "TestList|TestDownload"
```

Expected: compile error — `registerHandlers undefined`.

**Step 3: Create `api/library/ps_mag/route.go`**

```go
package ps_mag

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"

	"miltechserver/api/response"
)

// Handler holds the ps_mag service dependency.
type Handler struct {
	service Service
}

// RegisterHandlers wires ps_mag routes into the public router group.
// Called from api/library/route.go.
func RegisterHandlers(publicGroup *gin.RouterGroup, blobClient *azblob.Client, credential *azblob.SharedKeyCredential) {
	svc := NewService(blobClient, credential)
	registerHandlers(publicGroup, svc)
}

// registerHandlers is the internal wiring function used directly by tests.
func registerHandlers(publicGroup *gin.RouterGroup, svc Service) {
	handler := Handler{service: svc}
	publicGroup.GET("/library/ps-mag/issues", handler.listIssues)
	publicGroup.GET("/library/ps-mag/download", handler.generateDownloadURL)
}

// listIssues returns a paginated list of PS Magazine issues.
// GET /library/ps-mag/issues?page=1&order=asc&year=1994&issue=495
func (h *Handler) listIssues(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	order := c.DefaultQuery("order", "asc")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": ErrInvalidPage.Error(),
		})
		return
	}

	if o := strings.ToLower(order); o != "asc" && o != "desc" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": ErrInvalidOrder.Error(),
		})
		return
	}

	var year *int
	if yearStr := c.Query("year"); yearStr != "" {
		y, err := strconv.Atoi(yearStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request",
				"details": "year must be a valid integer",
			})
			return
		}
		year = &y
	}

	var issueNumber *int
	if issueStr := c.Query("issue"); issueStr != "" {
		i, err := strconv.Atoi(issueStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request",
				"details": "issue must be a valid integer",
			})
			return
		}
		issueNumber = &i
	}

	slog.Info("ListPSMagIssues endpoint called",
		"page", page, "order", order, "year", year, "issueNumber", issueNumber)

	result, err := h.service.ListIssues(page, order, year, issueNumber)
	if err != nil {
		slog.Error("Failed to list PS Magazine issues", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list issues",
			"details": err.Error(),
		})
		return
	}

	slog.Info("Successfully listed PS Magazine issues",
		"count", result.Count, "totalCount", result.TotalCount, "page", result.Page)

	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: result})
}

// generateDownloadURL returns a time-limited SAS URL for downloading a PS Magazine issue.
// GET /library/ps-mag/download?blob_path=ps-mag/PS_Magazine_Issue_495_February_1994.pdf
func (h *Handler) generateDownloadURL(c *gin.Context) {
	blobPath := c.Query("blob_path")

	slog.Info("GeneratePSMagDownloadURL endpoint called", "blobPath", blobPath)

	result, err := h.service.GenerateDownloadURL(blobPath)
	if err != nil {
		switch {
		case errors.Is(err, ErrIssueNotFound):
			slog.Warn("PS Magazine issue not found", "blobPath", blobPath, "error", err)
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Issue not found",
				"details": "The requested issue does not exist or is not accessible",
			})
		case errors.Is(err, ErrEmptyBlobPath), errors.Is(err, ErrInvalidBlobPath), errors.Is(err, ErrInvalidFileType):
			slog.Warn("Invalid blob path for PS Magazine download", "blobPath", blobPath, "error", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request",
				"details": err.Error(),
			})
		default:
			slog.Error("Failed to generate PS Magazine download URL", "error", err, "blobPath", blobPath)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to generate download URL",
				"details": err.Error(),
			})
		}
		return
	}

	slog.Info("Successfully generated PS Magazine download URL",
		"blobPath", blobPath, "expiresAt", result.ExpiresAt)

	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: result})
}
```

**Step 4: Run all tests to confirm they pass**

```bash
go test ./api/library/ps_mag/ -v
```

Expected: all tests `PASS`.

**Step 5: Commit**

```bash
git add api/library/ps_mag/route.go api/library/ps_mag/route_test.go
git commit -m "feat(ps-mag): implement route handlers with full test coverage"
```

---

## Task 5: Wire ps_mag into library/route.go

**Files:**
- Modify: `api/library/route.go`

**Step 1: Read the current file**

Open `api/library/route.go` and locate the `RegisterRoutes` function (line 30).

**Step 2: Add the ps_mag import and RegisterHandlers call**

The current `RegisterRoutes` function looks like this:

```go
func RegisterRoutes(deps Dependencies, publicGroup, authGroup *gin.RouterGroup) {
	svc := NewService(deps.BlobClient, deps.BlobCredential, deps.Env, deps.Analytics)
	registerHandlers(publicGroup, authGroup, svc)
}
```

Change it to:

```go
func RegisterRoutes(deps Dependencies, publicGroup, authGroup *gin.RouterGroup) {
	svc := NewService(deps.BlobClient, deps.BlobCredential, deps.Env, deps.Analytics)
	registerHandlers(publicGroup, authGroup, svc)
	ps_mag.RegisterHandlers(publicGroup, deps.BlobClient, deps.BlobCredential)
}
```

Also add the import at the top of the file:

```go
import (
	// ... existing imports ...
	"miltechserver/api/library/ps_mag"
)
```

**Step 3: Verify the build**

```bash
go build ./...
```

Expected: no output (success). If there are import errors, verify the package path matches `module/api/library/ps_mag` — check `go.mod` for the module name.

**Step 4: Run all library and ps_mag tests**

```bash
go test ./api/library/... -v
```

Expected: all tests `PASS`.

**Step 5: Commit**

```bash
git add api/library/route.go
git commit -m "feat(ps-mag): wire ps_mag sub-package into library route registration"
```

---

## Verification Checklist

After all tasks are complete, confirm the following:

- [ ] `go build ./...` succeeds with no errors
- [ ] `go test ./api/library/... -v` shows all tests PASS
- [ ] `go vet ./api/library/ps_mag/` reports no issues
- [ ] Routes registered: `GET /library/ps-mag/issues` and `GET /library/ps-mag/download`
- [ ] `api/library/route.go` calls `ps_mag.RegisterHandlers` exactly once
- [ ] No changes to `api/route/route.go` were required

---

## Files Summary

| File | Action |
|------|--------|
| `api/library/ps_mag/errors.go` | Created |
| `api/library/ps_mag/response.go` | Created |
| `api/library/ps_mag/service.go` | Created |
| `api/library/ps_mag/service_impl.go` | Created |
| `api/library/ps_mag/service_impl_test.go` | Created |
| `api/library/ps_mag/route.go` | Created |
| `api/library/ps_mag/route_test.go` | Created |
| `api/library/route.go` | Modified — added `ps_mag.RegisterHandlers` call |
