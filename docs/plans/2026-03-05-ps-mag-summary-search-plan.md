# PS Magazine Summary Search Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `GET /library/ps-mag/search?q=phrase&page=1` to search ps_mag_summaries by keyword and return only the matching lines from each file's summary.

**Architecture:** Raw SQL ILIKE query + Go line filtering. A new `Repository` interface + `RepositoryImpl` is added to the `ps_mag` package (mirroring user_suggestions). `ServiceImpl` gains a `repo Repository` field. `RegisterHandlers` and `NewService` are updated to accept `*sql.DB`, threaded through from `api/library/route.go`.

**Tech Stack:** Go, Gin, `database/sql`, raw SQL (ILIKE + LIMIT/OFFSET), testify

---

### Task 1: Foundation — errors, response types, constant

Add the new sentinel error, response types, and page size constant. No tests needed — these are pure type/const declarations.

**Files:**
- Modify: `api/library/ps_mag/errors.go`
- Modify: `api/library/ps_mag/response.go`
- Modify: `api/library/ps_mag/service_impl.go`

**Step 1: Add `ErrQueryTooShort` to `errors.go`**

The file currently contains 8 sentinel errors. Add one more at the end of the `var` block:

```go
ErrQueryTooShort = errors.New("search query must be at least 3 characters")
```

Full updated `errors.go`:
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
	ErrQueryTooShort   = errors.New("search query must be at least 3 characters")
)
```

**Step 2: Add search response types to `response.go`**

Append after `DownloadURLResponse`:

```go
// PSMagSearchResult is a single file with only its matching summary lines.
type PSMagSearchResult struct {
	FileName      string   `json:"file_name"`
	MatchingLines []string `json:"matching_lines"`
}

// PSMagSearchResponse is the paginated summary search response.
type PSMagSearchResponse struct {
	Results    []PSMagSearchResult `json:"results"`
	Count      int                 `json:"count"`
	TotalCount int                 `json:"total_count"`
	Page       int                 `json:"page"`
	TotalPages int                 `json:"total_pages"`
	Query      string              `json:"query"`
}
```

**Step 3: Add `SearchPageSize` constant to `service_impl.go`**

The existing `const` block is at the top of `service_impl.go`:
```go
const (
	PSMagContainerName = "library"
	PSMagPrefix        = "ps-mag/"
	PageSize           = 50
)
```

Add `SearchPageSize`:
```go
const (
	PSMagContainerName = "library"
	PSMagPrefix        = "ps-mag/"
	PageSize           = 50
	SearchPageSize     = 30
)
```

**Step 4: Verify it compiles**

```bash
go build ./api/library/ps_mag/...
```
Expected: no errors.

**Step 5: Commit**

```bash
git add api/library/ps_mag/errors.go api/library/ps_mag/response.go api/library/ps_mag/service_impl.go
git commit -m "feat(ps-mag): add search foundation types, error, and page size constant"
```

---

### Task 2: Repository interface

Define the `Repository` interface and the internal `summaryRow` type that the repository returns. No tests for interfaces.

**Files:**
- Create: `api/library/ps_mag/repository.go`

**Step 1: Create `repository.go`**

```go
package ps_mag

// summaryRow holds a single row from ps_mag_summaries.
// It is an internal type used between the repository and service layers.
type summaryRow struct {
	FileName string
	Summary  string
}

// Repository provides data access for the ps_mag_summaries table.
type Repository interface {
	// SearchSummaries returns paginated rows from ps_mag_summaries whose summary
	// contains phrase (case-insensitive), plus the total count of matching rows.
	// page is 1-indexed. pageSize controls how many rows are returned.
	SearchSummaries(phrase string, page, pageSize int) ([]summaryRow, int, error)
}
```

**Step 2: Verify it compiles**

```bash
go build ./api/library/ps_mag/...
```
Expected: no errors.

**Step 3: Commit**

```bash
git add api/library/ps_mag/repository.go
git commit -m "feat(ps-mag): add Repository interface for summary search"
```

---

### Task 3: Extend Service interface + update existing test stub

Adding `SearchSummaries` to the `Service` interface means the `serviceStub` in `route_test.go` no longer satisfies it. Fix both in one step so existing tests keep passing.

**Files:**
- Modify: `api/library/ps_mag/service.go`
- Modify: `api/library/ps_mag/route_test.go`

**Step 1: Add `SearchSummaries` to `service.go`**

```go
package ps_mag

import "context"

// Service provides methods for accessing PS Magazine issues from Azure Blob Storage.
type Service interface {
	// ListIssues returns a paginated, optionally filtered list of PS Magazine issues.
	// page is 1-indexed. order must be "asc" or "desc".
	// year and issueNumber are optional — pass nil to skip the filter.
	ListIssues(page int, order string, year *int, issueNumber *int) (*PSMagIssuesResponse, error)

	// GenerateDownloadURL creates a 1-hour SAS URL for a ps-mag blob.
	// ctx should be the request context so Azure calls are cancelled on client disconnect.
	// blobPath must start with "ps-mag/" and end with ".pdf".
	GenerateDownloadURL(ctx context.Context, blobPath string) (*DownloadURLResponse, error)

	// SearchSummaries returns a paginated list of PS Magazine issues whose summaries
	// contain query. Only the lines matching query are returned per file.
	// query must be at least 3 characters. page is 1-indexed.
	SearchSummaries(query string, page int) (*PSMagSearchResponse, error)
}
```

**Step 2: Add `SearchSummaries` stub to `serviceStub` in `route_test.go`**

Find the `serviceStub` struct at the top of `route_test.go`. Add two fields and one method:

```go
type serviceStub struct {
	listResp    *PSMagIssuesResponse
	listErr     error
	downloadErr error
	searchResp  *PSMagSearchResponse  // add this
	searchErr   error                 // add this
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

// add this method
func (s *serviceStub) SearchSummaries(query string, page int) (*PSMagSearchResponse, error) {
	return s.searchResp, s.searchErr
}
```

**Step 3: Run existing tests to confirm nothing broke**

```bash
go test ./api/library/ps_mag/... -v
```
Expected: all existing tests PASS.

**Step 4: Commit**

```bash
git add api/library/ps_mag/service.go api/library/ps_mag/route_test.go
git commit -m "feat(ps-mag): extend Service interface with SearchSummaries"
```

---

### Task 4: Route handler — TDD

Write failing handler tests first, then implement the handler.

**Files:**
- Modify: `api/library/ps_mag/route_test.go`
- Modify: `api/library/ps_mag/route.go`

**Step 1: Write the failing handler tests**

Append these tests to `route_test.go`:

```go
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
```

You will also need to add `"errors"` to the imports at the top of `route_test.go` if not already present.

**Step 2: Run tests — expect FAIL (handler not yet registered)**

```bash
go test ./api/library/ps_mag/... -run TestSearchSummaries -v
```
Expected: FAIL — route not found (404) or compile error.

**Step 3: Add `searchSummaries` handler and register the route in `route.go`**

In `registerHandlers`, add:
```go
publicGroup.GET("/library/ps-mag/search", handler.searchSummaries)
```

Add the handler method:
```go
// searchSummaries returns PS Magazine issues whose summary contains the query phrase.
// Only the lines from each summary that contain the phrase are returned.
// GET /library/ps-mag/search?q=phrase&page=1
func (h *Handler) searchSummaries(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if len(q) < 3 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": ErrQueryTooShort.Error(),
		})
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": ErrInvalidPage.Error(),
		})
		return
	}

	slog.Info("SearchPSMagSummaries endpoint called", "query", q, "page", page)

	result, err := h.service.SearchSummaries(q, page)
	if err != nil {
		slog.Error("Failed to search PS Magazine summaries", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to search summaries",
			"details": err.Error(),
		})
		return
	}

	slog.Info("Successfully searched PS Magazine summaries",
		"query", q, "totalCount", result.TotalCount, "page", result.Page)

	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: result})
}
```

**Step 4: Run tests — expect PASS**

```bash
go test ./api/library/ps_mag/... -run TestSearchSummaries -v
```
Expected: all 6 tests PASS.

**Step 5: Run the full package tests to make sure nothing regressed**

```bash
go test ./api/library/ps_mag/... -v
```
Expected: all tests PASS.

**Step 6: Commit**

```bash
git add api/library/ps_mag/route.go api/library/ps_mag/route_test.go
git commit -m "feat(ps-mag): add searchSummaries route handler with validation"
```

---

### Task 5: Service implementation — TDD

TDD for `filterMatchingLines` (pure function) first, then `SearchSummaries` (uses a stubbed repo).

**Files:**
- Modify: `api/library/ps_mag/service_impl_test.go`
- Modify: `api/library/ps_mag/service_impl.go`

**Step 1: Write the failing `filterMatchingLines` tests**

Append to `service_impl_test.go`. You will need to add `"strings"` to imports if not present.

```go
func TestFilterMatchingLines_SingleMatch(t *testing.T) {
	summary := "Line one\nlubrication point A\nLine three"
	got := filterMatchingLines(summary, "lubrication")
	require.Equal(t, []string{"lubrication point A"}, got)
}

func TestFilterMatchingLines_MultipleMatches(t *testing.T) {
	summary := "lubrication A\nno match\nlubrication B"
	got := filterMatchingLines(summary, "lubrication")
	require.Equal(t, []string{"lubrication A", "lubrication B"}, got)
}

func TestFilterMatchingLines_NoMatch(t *testing.T) {
	summary := "Line one\nLine two"
	got := filterMatchingLines(summary, "xyz")
	require.Nil(t, got)
}

func TestFilterMatchingLines_CaseInsensitive(t *testing.T) {
	summary := "Check LUBRICATION schedule"
	got := filterMatchingLines(summary, "lubrication")
	require.Equal(t, []string{"Check LUBRICATION schedule"}, got)
}

func TestFilterMatchingLines_EmptyLinesSkipped(t *testing.T) {
	summary := "\n  \nlubrication note\n\n"
	got := filterMatchingLines(summary, "lubrication")
	require.Equal(t, []string{"lubrication note"}, got)
}

func TestFilterMatchingLines_EmptySummary(t *testing.T) {
	got := filterMatchingLines("", "lubrication")
	require.Nil(t, got)
}
```

**Step 2: Run to verify they FAIL**

```bash
go test ./api/library/ps_mag/... -run TestFilterMatchingLines -v
```
Expected: FAIL — `filterMatchingLines` undefined.

**Step 3: Implement `filterMatchingLines` in `service_impl.go`**

Add after the closing brace of `GenerateDownloadURL`:

```go
// filterMatchingLines splits summary by newline and returns only the trimmed,
// non-empty lines that contain query (case-insensitive).
func filterMatchingLines(summary, query string) []string {
	lowerQuery := strings.ToLower(query)
	var matches []string
	for _, line := range strings.Split(summary, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && strings.Contains(strings.ToLower(trimmed), lowerQuery) {
			matches = append(matches, trimmed)
		}
	}
	return matches
}
```

**Step 4: Run to verify they PASS**

```bash
go test ./api/library/ps_mag/... -run TestFilterMatchingLines -v
```
Expected: all 6 PASS.

**Step 5: Write the failing `SearchSummaries` service tests**

These tests use an in-test `repoStub` to avoid needing a real DB. Append to `service_impl_test.go`:

```go
// repoStub satisfies Repository for unit testing.
type repoStub struct {
	rows  []summaryRow
	total int
	err   error
}

func (r *repoStub) SearchSummaries(_ string, _, _ int) ([]summaryRow, int, error) {
	return r.rows, r.total, r.err
}

func TestServiceSearchSummaries_ReturnsMatchingLines(t *testing.T) {
	stub := &repoStub{
		rows: []summaryRow{
			{FileName: "PS_Magazine_Issue_495_February_1994.pdf", Summary: "Check oil.\nOil level low.\nNo match here."},
		},
		total: 1,
	}
	svc := &ServiceImpl{repo: stub}

	resp, err := svc.SearchSummaries("oil", 1)

	require.NoError(t, err)
	require.Equal(t, 1, resp.TotalCount)
	require.Equal(t, 1, len(resp.Results))
	require.Equal(t, []string{"Check oil.", "Oil level low."}, resp.Results[0].MatchingLines)
	require.Equal(t, "oil", resp.Query)
}

func TestServiceSearchSummaries_EmptyResults(t *testing.T) {
	stub := &repoStub{rows: nil, total: 0}
	svc := &ServiceImpl{repo: stub}

	resp, err := svc.SearchSummaries("oil", 1)

	require.NoError(t, err)
	require.Equal(t, 0, resp.TotalCount)
	require.Empty(t, resp.Results)
}

func TestServiceSearchSummaries_QueryTooShort(t *testing.T) {
	svc := &ServiceImpl{repo: &repoStub{}}

	_, err := svc.SearchSummaries("ab", 1)

	require.ErrorIs(t, err, ErrQueryTooShort)
}

func TestServiceSearchSummaries_InvalidPage(t *testing.T) {
	svc := &ServiceImpl{repo: &repoStub{}}

	_, err := svc.SearchSummaries("oil", 0)

	require.ErrorIs(t, err, ErrInvalidPage)
}

func TestServiceSearchSummaries_RepoError(t *testing.T) {
	stub := &repoStub{err: errors.New("db down")}
	svc := &ServiceImpl{repo: stub}

	_, err := svc.SearchSummaries("oil", 1)

	require.Error(t, err)
}

func TestServiceSearchSummaries_Pagination(t *testing.T) {
	// 35 total results, page size 30 → 2 total pages
	stub := &repoStub{rows: make([]summaryRow, 5), total: 35}
	for i := range stub.rows {
		stub.rows[i] = summaryRow{FileName: "file.pdf", Summary: "oil note"}
	}
	svc := &ServiceImpl{repo: stub}

	resp, err := svc.SearchSummaries("oil", 2)

	require.NoError(t, err)
	require.Equal(t, 35, resp.TotalCount)
	require.Equal(t, 2, resp.TotalPages)
	require.Equal(t, 2, resp.Page)
}
```

You will need `"errors"` in the imports of `service_impl_test.go` if not already present.

**Step 6: Run to verify they FAIL**

```bash
go test ./api/library/ps_mag/... -run TestServiceSearchSummaries -v
```
Expected: FAIL — `SearchSummaries` not implemented on `ServiceImpl`, `repo` field does not exist.

**Step 7: Add `repo` field to `ServiceImpl` and implement `SearchSummaries`**

Update the `ServiceImpl` struct in `service_impl.go`:

```go
type ServiceImpl struct {
	blobClient *azblob.Client
	repo       Repository
}
```

Add the `SearchSummaries` method after `GenerateDownloadURL`:

```go
// SearchSummaries returns a paginated list of PS Magazine issues whose summaries
// contain query. Only the lines matching query are returned per file.
func (s *ServiceImpl) SearchSummaries(query string, page int) (*PSMagSearchResponse, error) {
	if len(strings.TrimSpace(query)) < 3 {
		return nil, ErrQueryTooShort
	}
	if page < 1 {
		return nil, ErrInvalidPage
	}

	rows, totalCount, err := s.repo.SearchSummaries(query, page, SearchPageSize)
	if err != nil {
		return nil, fmt.Errorf("search summaries: %w", err)
	}

	results := make([]PSMagSearchResult, 0, len(rows))
	for _, row := range rows {
		lines := filterMatchingLines(row.Summary, query)
		if len(lines) == 0 {
			continue
		}
		results = append(results, PSMagSearchResult{
			FileName:      row.FileName,
			MatchingLines: lines,
		})
	}

	totalPages := (totalCount + SearchPageSize - 1) / SearchPageSize
	if totalPages == 0 {
		totalPages = 1
	}

	return &PSMagSearchResponse{
		Results:    results,
		Count:      len(results),
		TotalCount: totalCount,
		Page:       page,
		TotalPages: totalPages,
		Query:      query,
	}, nil
}
```

Make sure `"fmt"` is in the imports of `service_impl.go` (it already is).

**Step 8: Run to verify they PASS**

```bash
go test ./api/library/ps_mag/... -run TestServiceSearchSummaries -v
```
Expected: all tests PASS.

**Step 9: Run the full package tests**

```bash
go test ./api/library/ps_mag/... -v
```
Expected: all tests PASS.

**Step 10: Commit**

```bash
git add api/library/ps_mag/service_impl.go api/library/ps_mag/service_impl_test.go
git commit -m "feat(ps-mag): implement SearchSummaries service method and filterMatchingLines"
```

---

### Task 6: Repository implementation — TDD

Write the `RepositoryImpl` backed by `*sql.DB`. The integration test requires a running PostgreSQL instance; it can be skipped in environments without one.

**Files:**
- Create: `api/library/ps_mag/repository_impl.go`
- Create: `api/library/ps_mag/repository_impl_test.go`

**Step 1: Create `repository_impl.go`**

```go
package ps_mag

import (
	"database/sql"
	"fmt"
)

// RepositoryImpl implements Repository using a PostgreSQL database.
type RepositoryImpl struct {
	db *sql.DB
}

// NewRepository constructs a RepositoryImpl backed by the given *sql.DB.
func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

// SearchSummaries returns rows from ps_mag_summaries whose summary contains
// phrase (ILIKE, case-insensitive), paginated by page/pageSize.
// Also returns the total count of matching rows for pagination metadata.
func (r *RepositoryImpl) SearchSummaries(phrase string, page, pageSize int) ([]summaryRow, int, error) {
	pattern := "%" + phrase + "%"
	offset := (page - 1) * pageSize

	var total int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM ps_mag_summaries WHERE summary ILIKE $1`,
		pattern,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count summaries: %w", err)
	}

	rows, err := r.db.Query(
		`SELECT file_name, summary
		 FROM ps_mag_summaries
		 WHERE summary ILIKE $1
		 ORDER BY file_name ASC
		 LIMIT $2 OFFSET $3`,
		pattern, pageSize, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("search summaries query: %w", err)
	}
	defer rows.Close()

	var results []summaryRow
	for rows.Next() {
		var row summaryRow
		var summary sql.NullString
		if err := rows.Scan(&row.FileName, &summary); err != nil {
			return nil, 0, fmt.Errorf("scan summary row: %w", err)
		}
		if summary.Valid {
			row.Summary = summary.String
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate summary rows: %w", err)
	}

	return results, total, nil
}
```

**Step 2: Create `repository_impl_test.go`**

These are integration tests. They read `TEST_DB_URL` from the environment and skip if it is unset.

```go
package ps_mag

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DB_URL")
	if dsn == "" {
		t.Skip("TEST_DB_URL not set — skipping repository integration tests")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestRepositorySearchSummaries_ReturnsResults(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)

	// This test assumes ps_mag_summaries has at least one row.
	// If the table is empty the test will pass with count=0 (not a failure).
	rows, total, err := repo.SearchSummaries("the", 1, 30)

	require.NoError(t, err)
	require.GreaterOrEqual(t, total, 0)
	require.LessOrEqual(t, len(rows), 30)
}

func TestRepositorySearchSummaries_NoMatch(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)

	rows, total, err := repo.SearchSummaries("zzz_no_match_xyz_999", 1, 30)

	require.NoError(t, err)
	require.Equal(t, 0, total)
	require.Empty(t, rows)
}

func TestRepositorySearchSummaries_PageTwo(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)

	// Page 2 with page size 1 — may return empty if only 1 match exists.
	_, _, err := repo.SearchSummaries("the", 2, 1)
	require.NoError(t, err)
}
```

**Step 3: Build to check for compile errors**

```bash
go build ./api/library/ps_mag/...
```
Expected: no errors.

**Step 4: Run integration tests (if TEST_DB_URL is available)**

```bash
TEST_DB_URL="postgres://user:pass@localhost/miltech_ng?sslmode=disable" \
  go test ./api/library/ps_mag/... -run TestRepository -v
```
Expected: PASS (or SKIP if TEST_DB_URL is not set).

**Step 5: Commit**

```bash
git add api/library/ps_mag/repository_impl.go api/library/ps_mag/repository_impl_test.go
git commit -m "feat(ps-mag): add RepositoryImpl for summary search with SQL ILIKE"
```

---

### Task 7: Wiring — thread *sql.DB into ps_mag

Update `NewService` and `RegisterHandlers` to accept `*sql.DB`, create the repository, and pass the DB from `api/library/route.go`.

**Files:**
- Modify: `api/library/ps_mag/service_impl.go`
- Modify: `api/library/ps_mag/route.go`
- Modify: `api/library/route.go`

**Step 1: Update `NewService` in `service_impl.go`**

Change the signature and constructor to accept `*sql.DB` and create the repository:

```go
// NewService creates a Service backed by blobClient for blob operations and
// db for summary search queries.
func NewService(blobClient *azblob.Client, db *sql.DB) Service {
	return &ServiceImpl{
		blobClient: blobClient,
		repo:       NewRepository(db),
	}
}
```

Add `"database/sql"` to the imports in `service_impl.go` if not already present.

**Step 2: Update `RegisterHandlers` in `route.go`**

```go
// RegisterHandlers wires ps_mag routes into the public router group.
// Called from api/library/route.go.
func RegisterHandlers(publicGroup *gin.RouterGroup, blobClient *azblob.Client, db *sql.DB) {
	svc := NewService(blobClient, db)
	registerHandlers(publicGroup, svc)
}
```

Add `"database/sql"` to the imports in `route.go` if not already present.

**Step 3: Update `api/library/route.go` to pass `deps.DB`**

Find the line:
```go
ps_mag.RegisterHandlers(publicGroup, deps.BlobClient)
```

Change it to:
```go
ps_mag.RegisterHandlers(publicGroup, deps.BlobClient, deps.DB)
```

**Step 4: Build the full project**

```bash
go build ./...
```
Expected: no errors.

**Step 5: Run all tests**

```bash
go test ./api/library/... -v
```
Expected: all tests PASS.

**Step 6: Commit**

```bash
git add api/library/ps_mag/service_impl.go api/library/ps_mag/route.go api/library/route.go
git commit -m "feat(ps-mag): wire *sql.DB into NewService and RegisterHandlers for summary search"
```

---

### Task 8: Final verification

**Step 1: Run the full test suite**

```bash
go test ./... 2>&1 | tail -20
```
Expected: all packages PASS, no compile errors.

**Step 2: Build a production binary**

```bash
go build -o /tmp/miltechserver_check ./...
rm /tmp/miltechserver_check
```
Expected: exits cleanly.

**Step 3: Commit if anything was touched in this task**

If nothing changed, no commit needed. Otherwise:

```bash
git add -p
git commit -m "fix(ps-mag): address final compilation issues"
```
