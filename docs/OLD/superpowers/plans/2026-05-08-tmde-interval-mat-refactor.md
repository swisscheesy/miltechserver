# TMDE Materialized View Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor the TMDE feature to query the `tmde_interval_mat` materialized view instead of the `tmde_requirements` table, exposing the new `item_name` column in all responses.

**Architecture:** All five layers of the TMDE feature (repository interface, repository impl, response type, service interface, service impl) reference `model.TmdeRequirements` — each must be updated to `model.TmdeIntervalMat`. The Jet query source changes from `table.TmdeRequirements` (package `table`) to `view.TmdeIntervalMat` (package `view`). Tests that assert on the shape of the returned data and the DB relation name must be updated to match.

**Tech Stack:** Go, Gin, go-jet/jet v2 (postgres), PostgreSQL materialized view, testify/require

---

## File Map

| File | Action | What changes |
|------|--------|-------------|
| `api/tmde/repository.go` | Modify | Return types: `model.TmdeRequirements` → `model.TmdeIntervalMat` |
| `api/tmde/repository_impl.go` | Modify | Import `view` package; swap `table.TmdeRequirements` → `view.TmdeIntervalMat`; update model types |
| `api/tmde/response.go` | Modify | `Items []model.TmdeRequirements` → `Items []model.TmdeIntervalMat` |
| `api/tmde/service.go` | Modify | Return type: `model.TmdeRequirements` → `model.TmdeIntervalMat` |
| `api/tmde/service_impl.go` | Modify | Return type propagation only; no logic change |
| `tests/tmde/handlers_test.go` | Modify | Unmarshal into `model.TmdeIntervalMat`; assert `*data.Niin`; reference `tmde_interval_mat` relation |
| `tests/tmde/helpers_test.go` | Modify | `fetchTmdeSample` queries `tmde_interval_mat`; `hasRelation` checks `tmde_interval_mat` |

---

## Task 1: Update the repository interface

**Files:**
- Modify: `api/tmde/repository.go`

- [ ] **Step 1: Update the return type in the repository interface**

Replace the file content with:

```go
package tmde

import "miltechserver/.gen/miltech_ng/public/model"

type Repository interface {
	GetByNIIN(niin string) (model.TmdeIntervalMat, error)
	GetAllPaginated(page int) (TmdePageResponse, error)
}
```

- [ ] **Step 2: Verify the file compiles (no test run yet — downstream files still reference the old type)**

```bash
cd /Users/swisscheese/projects/miltechserver && go build ./api/tmde/... 2>&1 | head -40
```

Expected: compile errors about `model.TmdeRequirements` mismatch in `repository_impl.go`, `response.go`, `service.go`, `service_impl.go` — that is correct at this stage.

---

## Task 2: Update the response type

**Files:**
- Modify: `api/tmde/response.go`

- [ ] **Step 1: Swap the Items element type**

Replace the file content with:

```go
package tmde

import "miltechserver/.gen/miltech_ng/public/model"

type TmdePageResponse struct {
	Items      []model.TmdeIntervalMat `json:"items"`
	Count      int                     `json:"count"`
	Page       int                     `json:"page"`
	TotalPages int                     `json:"total_pages"`
	IsLastPage bool                    `json:"is_last_page"`
}
```

---

## Task 3: Update the service interface

**Files:**
- Modify: `api/tmde/service.go`

- [ ] **Step 1: Swap the return type**

Replace the file content with:

```go
package tmde

import "miltechserver/.gen/miltech_ng/public/model"

type Service interface {
	LookupByNIIN(niin string) (model.TmdeIntervalMat, error)
	GetAllPaginated(page int) (TmdePageResponse, error)
}
```

---

## Task 4: Update the service implementation

**Files:**
- Modify: `api/tmde/service_impl.go`

- [ ] **Step 1: Update the return type of LookupByNIIN**

Replace the file content with:

```go
package tmde

import (
	"strings"

	"miltechserver/.gen/miltech_ng/public/model"
)

type service struct {
	repository Repository
}

func NewService(repo Repository) Service {
	return &service{repository: repo}
}

func (s *service) LookupByNIIN(niin string) (model.TmdeIntervalMat, error) {
	normalized := strings.TrimSpace(strings.ToUpper(niin))
	return s.repository.GetByNIIN(normalized)
}

func (s *service) GetAllPaginated(page int) (TmdePageResponse, error) {
	return s.repository.GetAllPaginated(page)
}
```

---

## Task 5: Refactor the repository implementation

This is the core change — swap the Jet query source from `table.TmdeRequirements` to `view.TmdeIntervalMat`.

**Files:**
- Modify: `api/tmde/repository_impl.go`

**Background:** Jet generates two separate packages for database objects:
- `table` — for regular tables (supports INSERT/UPDATE/DELETE)
- `view` — for views and materialized views (SELECT only)

The `view.TmdeIntervalMat` variable provides the same `AllColumns`, `Niin`, `ORDER_BY`-compatible columns as `table.TmdeRequirements` did. The COUNT query also uses `view.TmdeIntervalMat` as its FROM source.

Note: In `model.TmdeIntervalMat`, `Niin` is `*string` (pointer), not `string`. The WHERE clause in `GetByNIIN` uses the Jet column expression `view.TmdeIntervalMat.Niin.EQ(String(niin))` which operates on the column definition, not the Go struct field — so the pointer vs non-pointer difference only matters when reading scan results, not in the query builder.

- [ ] **Step 1: Replace the repository implementation**

Replace the file content with:

```go
package tmde

import (
	"database/sql"
	"math"
	"strings"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/.gen/miltech_ng/public/view"

	. "github.com/go-jet/jet/v2/postgres"
)

const pageSize = int64(100)

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetByNIIN(niin string) (model.TmdeIntervalMat, error) {
	if strings.TrimSpace(niin) == "" {
		return model.TmdeIntervalMat{}, ErrEmptyParam
	}

	var results []model.TmdeIntervalMat
	stmt := SELECT(view.TmdeIntervalMat.AllColumns).
		FROM(view.TmdeIntervalMat).
		WHERE(view.TmdeIntervalMat.Niin.EQ(String(niin)))

	if err := stmt.Query(r.db, &results); err != nil {
		return model.TmdeIntervalMat{}, err
	}

	if len(results) == 0 {
		return model.TmdeIntervalMat{}, ErrNotFound
	}

	return results[0], nil
}

func (r *repository) GetAllPaginated(page int) (TmdePageResponse, error) {
	if page < 1 {
		return TmdePageResponse{}, ErrInvalidPage
	}

	offset := pageSize * int64(page-1)

	var items []model.TmdeIntervalMat
	stmt := SELECT(view.TmdeIntervalMat.AllColumns).
		FROM(view.TmdeIntervalMat).
		ORDER_BY(view.TmdeIntervalMat.Niin.ASC()).
		LIMIT(pageSize).
		OFFSET(offset)

	if err := stmt.Query(r.db, &items); err != nil {
		return TmdePageResponse{}, err
	}

	if len(items) == 0 {
		return TmdePageResponse{}, ErrNotFound
	}

	var dest struct {
		Count int64
	}
	countStmt := SELECT(COUNT(Raw("*")).AS("count")).FROM(view.TmdeIntervalMat)
	if err := countStmt.Query(r.db, &dest); err != nil {
		return TmdePageResponse{}, err
	}
	totalCount := int(dest.Count)

	totalPages := int(math.Ceil(float64(totalCount) / float64(pageSize)))

	return TmdePageResponse{
		Items:      items,
		Count:      len(items),
		Page:       page,
		TotalPages: totalPages,
		IsLastPage: page >= totalPages,
	}, nil
}
```

- [ ] **Step 2: Verify the entire tmde package compiles cleanly**

```bash
cd /Users/swisscheese/projects/miltechserver && go build ./api/tmde/...
```

Expected: no output (clean build).

- [ ] **Step 3: Commit**

```bash
git add api/tmde/repository.go api/tmde/repository_impl.go api/tmde/response.go api/tmde/service.go api/tmde/service_impl.go
git commit -m "refactor(tmde): switch from tmde_requirements table to tmde_interval_mat view

Queries now target the tmde_interval_mat materialized view which adds
the item_name column to all responses. Repository, service, and response
types updated to use model.TmdeIntervalMat throughout."
```

---

## Task 6: Update test helpers to reference the materialized view

**Files:**
- Modify: `tests/tmde/helpers_test.go`

The test helpers call `hasRelation` with `"tmde_requirements"` and `fetchTmdeSample` queries that table directly. Both must reference `tmde_interval_mat` instead.

- [ ] **Step 1: Update the helpers**

Replace the file content with:

```go
package tmde_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"miltechserver/api/middleware"
	"miltechserver/api/tmde"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type standardResponse struct {
	Status  int             `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	return newTestRouterWithDB(t, testDB)
}

func newTestRouterWithDB(t *testing.T, db *sql.DB) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler)
	publicGroup := router.Group("/api/v1")
	tmde.RegisterRoutes(tmde.Dependencies{DB: db}, publicGroup)
	return router
}

func doJSONRequest(t *testing.T, router *gin.Engine, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequest(method, path, strings.NewReader(""))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func hasRelation(t *testing.T, db *sql.DB, relation string) bool {
	t.Helper()
	var exists bool
	err := db.QueryRow(`SELECT to_regclass($1) IS NOT NULL`, "public."+relation).Scan(&exists)
	require.NoError(t, err)
	return exists
}

func countRows(t *testing.T, db *sql.DB, relation string) int {
	t.Helper()
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM " + relation).Scan(&count)
	require.NoError(t, err)
	return count
}

func fetchTmdeSample(t *testing.T, db *sql.DB) (string, bool) {
	t.Helper()
	if !hasRelation(t, db, "tmde_interval_mat") {
		return "", false
	}
	var niin sql.NullString
	err := db.QueryRow("SELECT niin FROM tmde_interval_mat LIMIT 1").Scan(&niin)
	if err == sql.ErrNoRows {
		return "", false
	}
	require.NoError(t, err)
	if !niin.Valid {
		return "", false
	}
	return niin.String, true
}
```

---

## Task 7: Update handler tests to use the new model type

**Files:**
- Modify: `tests/tmde/handlers_test.go`

Three categories of changes:
1. `hasRelation` / `countRows` calls that pass `"tmde_requirements"` → `"tmde_interval_mat"`
2. Unmarshal target in `TestTmdeLookupByNIIN`: `model.TmdeRequirements` → `model.TmdeIntervalMat`
3. NIIN assertion: `data.Niin` was `string`, is now `*string` — must dereference with `*data.Niin`

- [ ] **Step 1: Write the updated handler test file**

Replace the file content with:

```go
package tmde_test

import (
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"testing"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/api/tmde"

	"github.com/stretchr/testify/require"
)

func TestTmdeBlankParams(t *testing.T) {
	router := newTestRouter(t)

	blankNiinResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/niin/%20%20")
	require.Equal(t, http.StatusBadRequest, blankNiinResp.Code)
	var niinErrBody struct {
		Error string `json:"error"`
	}
	require.NoError(t, json.Unmarshal(blankNiinResp.Body.Bytes(), &niinErrBody))
	require.NotEmpty(t, niinErrBody.Error)

	invalidPageResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/requirements?page=bad")
	require.Equal(t, http.StatusBadRequest, invalidPageResp.Code)

	zeroPageResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/requirements?page=0")
	require.Equal(t, http.StatusBadRequest, zeroPageResp.Code)
}

func TestTmdeLookupByNIIN(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "tmde_interval_mat") {
		t.Skip("tmde_interval_mat view missing in test DB")
	}

	rowCount := countRows(t, testDB, "tmde_interval_mat")
	niinValue, ok := fetchTmdeSample(t, testDB)

	if ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/niin/"+niinValue)
		require.Equal(t, http.StatusOK, resp.Code)

		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))

		var data model.TmdeIntervalMat
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotNil(t, data.Niin)
		require.Equal(t, niinValue, *data.Niin)
	} else if rowCount == 0 {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/niin/TEST")
		require.Equal(t, http.StatusNotFound, resp.Code)
	}
}

func TestTmdeNiinNotFound(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "tmde_interval_mat") {
		t.Skip("tmde_interval_mat view missing in test DB")
	}

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/niin/000000000NOTREAL")
	require.Equal(t, http.StatusNotFound, resp.Code)

	var payload response.NoItemFoundResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	require.Equal(t, http.StatusNotFound, payload.Status)
}

func TestTmdeListPaginated(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "tmde_interval_mat") {
		t.Skip("tmde_interval_mat view missing in test DB")
	}

	rowCount := countRows(t, testDB, "tmde_interval_mat")
	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/requirements?page=1")

	if rowCount == 0 {
		require.Equal(t, http.StatusNotFound, resp.Code)
		return
	}

	require.Equal(t, http.StatusOK, resp.Code)

	var payload standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))

	var data tmde.TmdePageResponse
	require.NoError(t, json.Unmarshal(payload.Data, &data))
	require.NotEmpty(t, data.Items)
	require.Equal(t, 1, data.Page)

	expectedTotalPages := int(math.Ceil(float64(rowCount) / 100.0))
	require.Equal(t, expectedTotalPages, data.TotalPages)
	require.Equal(t, 1 >= expectedTotalPages, data.IsLastPage)
}

func TestTmdeListDefaultPage(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "tmde_interval_mat") {
		t.Skip("tmde_interval_mat view missing in test DB")
	}

	rowCount := countRows(t, testDB, "tmde_interval_mat")

	// omitting ?page should default to page 1
	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/requirements")
	if rowCount == 0 {
		require.Equal(t, http.StatusNotFound, resp.Code)
		return
	}
	require.Equal(t, http.StatusOK, resp.Code)
}

func TestTmdeListBeyondLastPage(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "tmde_interval_mat") {
		t.Skip("tmde_interval_mat view missing in test DB")
	}

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/requirements?page=999999")
	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestTmdeInternalError(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://invalid:invalid@localhost:1/invalid?sslmode=disable")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	router := newTestRouterWithDB(t, db)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/requirements?page=1")
	require.Equal(t, http.StatusInternalServerError, resp.Code)

	var payload response.ErrorResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	require.Equal(t, http.StatusInternalServerError, payload.Status)
}
```

- [ ] **Step 2: Run the validation tests (no DB required)**

```bash
cd /Users/swisscheese/projects/miltechserver && go test ./tests/tmde/... -run TestRepositoryValidationErrors -v
```

Expected:
```
--- PASS: TestRepositoryValidationErrors (0.00s)
PASS
```

- [ ] **Step 3: Run blank param tests (no DB required)**

```bash
cd /Users/swisscheese/projects/miltechserver && go test ./tests/tmde/... -run TestTmdeBlankParams -v
```

Expected:
```
--- PASS: TestTmdeBlankParams (0.00s)
PASS
```

- [ ] **Step 4: Run the full test suite**

```bash
cd /Users/swisscheese/projects/miltechserver && go test ./tests/tmde/... -v
```

Expected: all tests pass or skip (tests that require a live DB will skip if the view is absent).

- [ ] **Step 5: Commit**

```bash
git add tests/tmde/handlers_test.go tests/tmde/helpers_test.go
git commit -m "test(tmde): update tests to reference tmde_interval_mat view

Replace all tmde_requirements references in test helpers and handler
tests. Update NIIN assertion to dereference *string Niin field from
the new TmdeIntervalMat model."
```

---

## Self-Review Checklist

- [x] **Spec coverage:** All references to `tmde_requirements` in the application code replaced by `tmde_interval_mat`; new `ItemName` field returned automatically via `AllColumns`; `Niin *string` dereferencing handled in tests
- [x] **No placeholders:** All steps contain exact file content, no TBDs
- [x] **Type consistency:** `model.TmdeIntervalMat` used consistently across all five layers; `view.TmdeIntervalMat` used in all Jet query builder calls
