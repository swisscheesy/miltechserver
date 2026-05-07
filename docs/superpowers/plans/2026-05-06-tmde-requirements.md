# TMDE Requirements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expose the `tmde_requirements` table via two public, unauthenticated endpoints — a NIIN lookup and a paginated list — following the existing `eic`/`pol_products` module pattern.

**Architecture:** Flat package `api/tmde/` with 7 files (errors, repository interface, repository impl, service interface, service impl, response type, route/handlers). Registered on `v1Route` alongside `eic` and `pol_products`. Tests live in `tests/tmde/` mirroring `tests/eic/`.

**Tech Stack:** Go, Gin, go-jet/v2 (Jet), PostgreSQL, testify/require

---

## File Map

| Action | Path | Responsibility |
|--------|------|----------------|
| Create | `api/tmde/errors.go` | Sentinel errors: ErrNotFound, ErrEmptyParam, ErrInvalidPage |
| Create | `api/tmde/response.go` | TmdePageResponse struct (package-local) |
| Create | `api/tmde/repository.go` | Repository interface |
| Create | `api/tmde/repository_impl.go` | Jet-based SQL queries |
| Create | `api/tmde/service.go` | Service interface |
| Create | `api/tmde/service_impl.go` | Input normalization, delegates to repository |
| Create | `api/tmde/route.go` | Dependencies, Handler, RegisterRoutes, RegisterHandlers, two handlers |
| Modify | `api/route/route.go` | Register tmde routes in public block |
| Create | `tests/tmde/main_test.go` | TestMain, DB setup, loadEnv |
| Create | `tests/tmde/helpers_test.go` | newTestRouter, doJSONRequest, hasRelation, countRows, fetchTmdeSample |
| Create | `tests/tmde/repository_test.go` | Repository validation error tests (nil DB) |
| Create | `tests/tmde/handlers_test.go` | Handler integration tests against test DB |

---

## Task 1: Package foundation — errors, interfaces, response type

**Files:**
- Create: `api/tmde/errors.go`
- Create: `api/tmde/response.go`
- Create: `api/tmde/repository.go`
- Create: `api/tmde/service.go`

- [ ] **Step 1: Create errors.go**

```go
// api/tmde/errors.go
package tmde

import "errors"

var (
	ErrNotFound    = errors.New("no TMDE requirements found")
	ErrEmptyParam  = errors.New("required parameter is empty")
	ErrInvalidPage = errors.New("page number must be greater than 0")
)
```

- [ ] **Step 2: Create response.go**

```go
// api/tmde/response.go
package tmde

import "miltechserver/.gen/miltech_ng/public/model"

type TmdePageResponse struct {
	Items      []model.TmdeRequirements `json:"items"`
	Count      int                      `json:"count"`
	Page       int                      `json:"page"`
	TotalPages int                      `json:"total_pages"`
	IsLastPage bool                     `json:"is_last_page"`
}
```

- [ ] **Step 3: Create repository.go**

```go
// api/tmde/repository.go
package tmde

import "miltechserver/.gen/miltech_ng/public/model"

type Repository interface {
	GetByNIIN(niin string) (model.TmdeRequirements, error)
	GetAllPaginated(page int) (TmdePageResponse, error)
}
```

- [ ] **Step 4: Create service.go**

```go
// api/tmde/service.go
package tmde

import "miltechserver/.gen/miltech_ng/public/model"

type Service interface {
	LookupByNIIN(niin string) (model.TmdeRequirements, error)
	GetAllPaginated(page int) (TmdePageResponse, error)
}
```

- [ ] **Step 5: Verify the package compiles (no impl yet, just types)**

```bash
cd /Users/swisscheese/projects/miltechserver && go build ./api/tmde/...
```

Expected: compile error — interfaces are defined but no implementations yet. That's fine; this verifies the types themselves are correct before adding implementations.

- [ ] **Step 6: Commit**

```bash
git add api/tmde/errors.go api/tmde/response.go api/tmde/repository.go api/tmde/service.go
git commit -m "feat(tmde): add package foundation - errors, interfaces, response type"
```

---

## Task 2: Repository implementation

**Files:**
- Create: `api/tmde/repository_impl.go`

- [ ] **Step 1: Create repository_impl.go**

```go
// api/tmde/repository_impl.go
package tmde

import (
	"database/sql"
	"math"
	"strings"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/.gen/miltech_ng/public/table"

	. "github.com/go-jet/jet/v2/postgres"
)

const pageSize = int64(100)

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetByNIIN(niin string) (model.TmdeRequirements, error) {
	if strings.TrimSpace(niin) == "" {
		return model.TmdeRequirements{}, ErrEmptyParam
	}

	var results []model.TmdeRequirements
	stmt := SELECT(table.TmdeRequirements.AllColumns).
		FROM(table.TmdeRequirements).
		WHERE(table.TmdeRequirements.Niin.EQ(String(niin)))

	if err := stmt.Query(r.db, &results); err != nil {
		return model.TmdeRequirements{}, err
	}

	if len(results) == 0 {
		return model.TmdeRequirements{}, ErrNotFound
	}

	return results[0], nil
}

func (r *repository) GetAllPaginated(page int) (TmdePageResponse, error) {
	if page < 1 {
		return TmdePageResponse{}, ErrInvalidPage
	}

	offset := pageSize * int64(page-1)

	var items []model.TmdeRequirements
	stmt := SELECT(table.TmdeRequirements.AllColumns).
		FROM(table.TmdeRequirements).
		LIMIT(pageSize).
		OFFSET(offset)

	if err := stmt.Query(r.db, &items); err != nil {
		return TmdePageResponse{}, err
	}

	if len(items) == 0 {
		return TmdePageResponse{}, ErrNotFound
	}

	var totalCount int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM tmde_requirements").Scan(&totalCount); err != nil {
		return TmdePageResponse{}, err
	}

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

- [ ] **Step 2: Verify the package compiles**

```bash
cd /Users/swisscheese/projects/miltechserver && go build ./api/tmde/...
```

Expected: compile error — service and route don't exist yet but the repository should be fine. If there are errors only in repository_impl.go, fix them now.

- [ ] **Step 3: Commit**

```bash
git add api/tmde/repository_impl.go
git commit -m "feat(tmde): implement repository with Jet queries"
```

---

## Task 3: Service implementation

**Files:**
- Create: `api/tmde/service_impl.go`

- [ ] **Step 1: Create service_impl.go**

```go
// api/tmde/service_impl.go
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

func (s *service) LookupByNIIN(niin string) (model.TmdeRequirements, error) {
	normalized := strings.TrimSpace(strings.ToUpper(niin))
	return s.repository.GetByNIIN(normalized)
}

func (s *service) GetAllPaginated(page int) (TmdePageResponse, error) {
	return s.repository.GetAllPaginated(page)
}
```

- [ ] **Step 2: Commit**

```bash
git add api/tmde/service_impl.go
git commit -m "feat(tmde): implement service"
```

---

## Task 4: Route and handlers

**Files:**
- Create: `api/tmde/route.go`

- [ ] **Step 1: Create route.go**

```go
// api/tmde/route.go
package tmde

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"miltechserver/api/response"

	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	DB *sql.DB
}

type Handler struct {
	service Service
}

func RegisterRoutes(deps Dependencies, router *gin.RouterGroup) {
	repo := NewRepository(deps.DB)
	svc := NewService(repo)
	RegisterHandlers(router, svc)
}

func RegisterHandlers(router *gin.RouterGroup, svc Service) {
	handler := Handler{service: svc}
	router.GET("/tmde/niin/:niin", handler.lookupByNIIN)
	router.GET("/tmde/requirements", handler.listAllPaginated)
}

func (h *Handler) lookupByNIIN(c *gin.Context) {
	niin := c.Param("niin")

	if strings.TrimSpace(niin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "NIIN parameter is required"})
		return
	}

	item, err := h.service.LookupByNIIN(niin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "",
		Data:    item,
	})
}

func (h *Handler) listAllPaginated(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}

	data, err := h.service.GetAllPaginated(page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "",
		Data:    data,
	})
}
```

- [ ] **Step 2: Verify the full package compiles cleanly**

```bash
cd /Users/swisscheese/projects/miltechserver && go build ./api/tmde/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add api/tmde/route.go
git commit -m "feat(tmde): implement route handlers"
```

---

## Task 5: Test infrastructure

**Files:**
- Create: `tests/tmde/main_test.go`
- Create: `tests/tmde/helpers_test.go`

- [ ] **Step 1: Create tests/tmde/main_test.go**

```go
// tests/tmde/main_test.go
package tmde_test

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	_ = loadEnv()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		log.Fatal("TEST_DATABASE_URL is not set")
	}

	var err error
	testDB, err = sql.Open("postgres", "postgres://postgres:potato123@192.168.20.70/miltech_ng_test?sslmode=disable")
	if err != nil {
		log.Fatalf("failed to open test database: %v", err)
	}

	if err := testDB.Ping(); err != nil {
		log.Fatalf("failed to ping test database: %v", err)
	}

	exitCode := m.Run()

	if err := testDB.Close(); err != nil {
		log.Printf("failed to close test database: %v", err)
	}

	os.Exit(exitCode)
}

func loadEnv() error {
	if os.Getenv("TEST_DATABASE_URL") != "" {
		return nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	current := wd
	for {
		envPath := filepath.Join(current, ".env")
		if _, statErr := os.Stat(envPath); statErr == nil {
			return godotenv.Load(envPath)
		}

		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}
```

- [ ] **Step 2: Create tests/tmde/helpers_test.go**

```go
// tests/tmde/helpers_test.go
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
	if !hasRelation(t, db, "tmde_requirements") {
		return "", false
	}
	var niin sql.NullString
	err := db.QueryRow("SELECT niin FROM tmde_requirements LIMIT 1").Scan(&niin)
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

- [ ] **Step 3: Verify the test package compiles**

```bash
cd /Users/swisscheese/projects/miltechserver && go test ./tests/tmde/... -run=^$ -v
```

Expected: `ok` with no tests run (there are no test functions yet). If it fails to compile, fix the error before continuing.

- [ ] **Step 4: Commit**

```bash
git add tests/tmde/main_test.go tests/tmde/helpers_test.go
git commit -m "test(tmde): add test infrastructure"
```

---

## Task 6: Repository validation tests

**Files:**
- Create: `tests/tmde/repository_test.go`

- [ ] **Step 1: Write the failing test**

```go
// tests/tmde/repository_test.go
package tmde_test

import (
	"database/sql"
	"testing"

	"miltechserver/api/tmde"

	"github.com/stretchr/testify/require"
)

func TestRepositoryValidationErrors(t *testing.T) {
	repo := tmde.NewRepository((*sql.DB)(nil))

	_, err := repo.GetByNIIN("  ")
	require.ErrorIs(t, err, tmde.ErrEmptyParam)

	_, err = repo.GetByNIIN("\t")
	require.ErrorIs(t, err, tmde.ErrEmptyParam)

	_, err = repo.GetAllPaginated(0)
	require.ErrorIs(t, err, tmde.ErrInvalidPage)

	_, err = repo.GetAllPaginated(-1)
	require.ErrorIs(t, err, tmde.ErrInvalidPage)
}
```

- [ ] **Step 2: Run the test**

```bash
cd /Users/swisscheese/projects/miltechserver && go test ./tests/tmde/... -run=TestRepositoryValidationErrors -v
```

Expected: `PASS` — validation checks fire before any DB call so the nil `*sql.DB` is never touched.

- [ ] **Step 3: Commit**

```bash
git add tests/tmde/repository_test.go
git commit -m "test(tmde): add repository validation tests"
```

---

## Task 7: Handler integration tests

**Files:**
- Create: `tests/tmde/handlers_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// tests/tmde/handlers_test.go
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

	invalidPageResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/requirements?page=bad")
	require.Equal(t, http.StatusBadRequest, invalidPageResp.Code)

	zeroPageResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/requirements?page=0")
	require.Equal(t, http.StatusBadRequest, zeroPageResp.Code)
}

func TestTmdeLookupByNIIN(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "tmde_requirements") {
		t.Skip("tmde_requirements table missing in test DB")
	}

	rowCount := countRows(t, testDB, "tmde_requirements")
	niinValue, ok := fetchTmdeSample(t, testDB)

	if ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/niin/"+niinValue)
		require.Equal(t, http.StatusOK, resp.Code)

		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))

		var data model.TmdeRequirements
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.Equal(t, niinValue, data.Niin)
	} else if rowCount == 0 {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/niin/TEST")
		require.Equal(t, http.StatusNotFound, resp.Code)
	}
}

func TestTmdeNiinNotFound(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "tmde_requirements") {
		t.Skip("tmde_requirements table missing in test DB")
	}

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/niin/000000000NOTREAL")
	require.Equal(t, http.StatusNotFound, resp.Code)

	var payload response.NoItemFoundResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	require.Equal(t, http.StatusNotFound, payload.Status)
}

func TestTmdeListPaginated(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "tmde_requirements") {
		t.Skip("tmde_requirements table missing in test DB")
	}

	rowCount := countRows(t, testDB, "tmde_requirements")
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

	if !hasRelation(t, testDB, "tmde_requirements") {
		t.Skip("tmde_requirements table missing in test DB")
	}

	rowCount := countRows(t, testDB, "tmde_requirements")

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

	if !hasRelation(t, testDB, "tmde_requirements") {
		t.Skip("tmde_requirements table missing in test DB")
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

- [ ] **Step 2: Run all tests to confirm they compile and pass**

```bash
cd /Users/swisscheese/projects/miltechserver && go test ./tests/tmde/... -v
```

Expected: All tests pass. Tests that require the DB table will skip gracefully with `t.Skip` if the table is absent. `TestTmdeBlankParams` and `TestRepositoryValidationErrors` should always pass (no DB needed). `TestTmdeInternalError` should pass (uses an invalid DSN intentionally).

If any test fails, fix the production code before committing.

- [ ] **Step 3: Commit**

```bash
git add tests/tmde/handlers_test.go
git commit -m "test(tmde): add handler integration tests"
```

---

## Task 8: Wire into main router

**Files:**
- Modify: `api/route/route.go`

- [ ] **Step 1: Add tmde import and registration**

Open `api/route/route.go`. Add `"miltechserver/api/tmde"` to the import block alongside `eic`, `pol_products`, etc.

In the `Setup` function, add the following line in the public routes block (after `eic.RegisterRoutes` is a natural fit):

```go
tmde.RegisterRoutes(tmde.Dependencies{DB: db}, v1Route)
```

The updated public routes block will look like:

```go
// All Public Routes
NewGeneralRouter(v1Route, env)
NewGeneralQueriesRouter(v1Route, env)
item_query.RegisterRoutes(item_query.Dependencies{DB: db}, v1Route)
item_lookup.RegisterRoutes(item_lookup.Dependencies{DB: db}, v1Route)
quick_lists.RegisterRoutes(quick_lists.Dependencies{DB: db}, v1Route)
pol_products.RegisterRoutes(pol_products.Dependencies{DB: db}, v1Route)
eic.RegisterRoutes(eic.Dependencies{DB: db}, v1Route)
tmde.RegisterRoutes(tmde.Dependencies{DB: db}, v1Route)
docs_equipment.RegisterRoutes(docs_equipment.Dependencies{DB: db, BlobClient: blobClient}, v1Route)
```

- [ ] **Step 2: Verify the full project builds**

```bash
cd /Users/swisscheese/projects/miltechserver && go build ./...
```

Expected: no errors.

- [ ] **Step 3: Run the full tmde test suite one final time**

```bash
cd /Users/swisscheese/projects/miltechserver && go test ./tests/tmde/... -v
```

Expected: all tests pass (or skip gracefully if test DB table is absent).

- [ ] **Step 4: Commit**

```bash
git add api/route/route.go
git commit -m "feat(tmde): register TMDE requirements routes in main router"
```
