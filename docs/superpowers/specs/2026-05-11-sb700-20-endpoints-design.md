# SB 700-20 Endpoints Design

**Date:** 2026-05-11
**Branch:** sb700-20-20

## Overview

Add paginated list and LIN-search endpoints for all 13 SB 700-20 tables. All endpoints are public (no auth). The feature lives in a single `api/sb_700_20/` package and follows the existing TMDE repository/service/handler pattern.

## Tables

| Model | DB Table | PK Type | Search Field | Search Returns |
|---|---|---|---|---|
| `Sb70020AppB` | `sb_700_20_app_b` | composite (lin + nsn) | `lin` | list |
| `Sb70020AppC` | `sb_700_20_app_c` | single (lin) | `lin` | single |
| `Sb70020AppD` | `sb_700_20_app_d` | composite (lin + nsn) | `lin` | list |
| `Sb70020AppE` | `sb_700_20_app_e` | composite (lin + nsn) | `lin` | list |
| `Sb70020AppF` | `sb_700_20_app_f` | single (lin) | `lin` | single |
| `Sb70020AppG` | `sb_700_20_app_g` | single (lin) | `lin` | single |
| `Sb70020AppH1` | `sb_700_20_app_h1` | composite (lin_zmm_lin + lin_zmm_sublin) | `lin_zmm_lin` | list |
| `Sb70020AppH2` | `sb_700_20_app_h2` | composite (lin_zmmsublin + lin_zmm_lin) | `lin_zmm_lin` | list |
| `Sb70020AppI` | `sb_700_20_app_i` | single (lin) | `lin` | single |
| `Sb70020AppJ` | `sb_700_20_app_j` | single (lin) | `lin` | single |
| `Sb70020Chp4` | `sb_700_20_chp_4` | single (lin) | `lin` | single |
| `Sb70020Chp6` | `sb_700_20_chp_6` | composite (lin + current_mcn) | `lin` | list |
| `Sb70020Chp8` | `sb_700_20_chp_8` | composite (lin + current_mcn) | `lin` | list |

## URL Routes (26 total)

All routes are mounted under `/api/v1` (public group).

```
GET /sb700-20/app-b/list?page=1        GET /sb700-20/app-b/search/:lin
GET /sb700-20/app-c/list?page=1        GET /sb700-20/app-c/search/:lin
GET /sb700-20/app-d/list?page=1        GET /sb700-20/app-d/search/:lin
GET /sb700-20/app-e/list?page=1        GET /sb700-20/app-e/search/:lin
GET /sb700-20/app-f/list?page=1        GET /sb700-20/app-f/search/:lin
GET /sb700-20/app-g/list?page=1        GET /sb700-20/app-g/search/:lin
GET /sb700-20/app-h1/list?page=1       GET /sb700-20/app-h1/search/:lin
GET /sb700-20/app-h2/list?page=1       GET /sb700-20/app-h2/search/:lin
GET /sb700-20/app-i/list?page=1        GET /sb700-20/app-i/search/:lin
GET /sb700-20/app-j/list?page=1        GET /sb700-20/app-j/search/:lin
GET /sb700-20/chp-4/list?page=1        GET /sb700-20/chp-4/search/:lin
GET /sb700-20/chp-6/list?page=1        GET /sb700-20/chp-6/search/:lin
GET /sb700-20/chp-8/list?page=1        GET /sb700-20/chp-8/search/:lin
```

The `:lin` path param name is uniform across all tables — for `app_h1` and `app_h2` the repository maps it to the `lin_zmm_lin` column internally.

`page` defaults to `1` if omitted. Page size is fixed at 100.

## Package Structure

```
api/sb_700_20/
├── errors.go              — ErrNotFound, ErrEmptyParam, ErrInvalidPage
├── response.go            — PageResponse[T any] generic struct
├── repository.go          — Repository interface (26 method signatures)
├── repository_impl.go     — repository struct + NewRepository constructor
├── repository_apps.go     — method impls for app_b through app_j (20 methods)
├── repository_chps.go     — method impls for chp_4, chp_6, chp_8 (6 methods)
├── service.go             — Service interface (26 method signatures)
├── service_impl.go        — service struct, NewService, all delegating methods
├── route.go               — Dependencies, Handler, RegisterRoutes, RegisterHandlers
├── handlers_apps.go       — handler methods for app table endpoints
└── handlers_chps.go       — handler methods for chapter table endpoints

tests/sb_700_20/
├── main_test.go
├── helpers_test.go
├── handlers_test.go
└── repository_test.go
```

## Data Layer

### response.go

```go
type PageResponse[T any] struct {
    Items      []T  `json:"items"`
    Count      int  `json:"count"`
    Page       int  `json:"page"`
    TotalPages int  `json:"total_pages"`
    IsLastPage bool `json:"is_last_page"`
}
```

### repository.go

One interface with 26 methods. Naming convention: `Get{Table}ByLIN` / `Get{Table}Paginated`.

- Single-PK tables: `GetAppCByLIN(lin string) (model.Sb70020AppC, error)`
- Composite-PK tables: `GetAppBByLIN(lin string) ([]model.Sb70020AppB, error)`
- Paginated: `GetAppCPaginated(page int) (PageResponse[model.Sb70020AppC], error)`

### repository_impl.go

```go
const pageSize = int64(100)

type repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) Repository { return &repository{db: db} }
```

### repository_apps.go / repository_chps.go

Each method:
1. Validates input (`ErrEmptyParam` for blank string, `ErrInvalidPage` for page < 1)
2. Builds Jet SELECT statement using the generated table package
3. For search: `WHERE col.EQ(String(param))`
4. For paginated: `ORDER_BY(col.ASC()).LIMIT(pageSize).OFFSET(pageSize * int64(page-1))` + separate COUNT query
5. Returns `ErrNotFound` when result slice is empty

Paginated `ORDER_BY` column:
- `app_h1`, `app_h2`: `lin_zmm_lin ASC`
- all others: `lin ASC`

## Service Layer

### service.go

Mirrors Repository interface method-for-method (same 26 signatures).

### service_impl.go

Thin delegation. Only logic: normalize string params with `strings.TrimSpace(strings.ToUpper(lin))` before passing to repository. Page integer is passed through directly.

```go
type service struct{ repository Repository }

func NewService(repo Repository) Service { return &service{repository: repo} }

func (s *service) GetAppCByLIN(lin string) (model.Sb70020AppC, error) {
    return s.repository.GetAppCByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
```

## Handler Layer

### route.go

```go
type Dependencies struct{ DB *sql.DB }
type Handler struct{ service Service }

func RegisterRoutes(deps Dependencies, router *gin.RouterGroup) {
    repo := NewRepository(deps.DB)
    svc := NewService(repo)
    RegisterHandlers(router, svc)
}

func RegisterHandlers(router *gin.RouterGroup, svc Service) {
    handler := Handler{service: svc}
    router.GET("/sb700-20/app-b/list", handler.listAppB)
    router.GET("/sb700-20/app-b/search/:lin", handler.searchAppB)
    // ... all 26 routes ...
}
```

### handlers_apps.go / handlers_chps.go

**Search handler — single-item result:**
```go
func (h *Handler) searchAppC(c *gin.Context) {
    lin := c.Param("lin")
    if strings.TrimSpace(lin) == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
        return
    }
    item, err := h.service.GetAppCByLIN(lin)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
        } else {
            c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
        }
        return
    }
    c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: item})
}
```

**Search handler — list result** (composite-PK tables): identical shape; service returns a slice instead of a single item.

**Paginated list handler** (same for all 13 tables):
```go
func (h *Handler) listAppC(c *gin.Context) {
    pageStr := c.DefaultQuery("page", "1")
    page, err := strconv.Atoi(pageStr)
    if err != nil || page < 1 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
        return
    }
    data, err := h.service.GetAppCPaginated(page)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
        } else {
            c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
        }
        return
    }
    c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: data})
}
```

## Route Registration

One line added to `api/route/route.go` in the public routes block:

```go
import "miltechserver/api/sb_700_20"

sb_700_20.RegisterRoutes(sb_700_20.Dependencies{DB: db}, v1Route)
```

## Tests

### tests/sb_700_20/main_test.go
- `TestMain` opens DB via `TEST_DATABASE_URL` env var (falls back to `.env` file walk)
- Mirrors TMDE `main_test.go` exactly

### tests/sb_700_20/helpers_test.go
- `newTestRouter` / `newTestRouterWithDB` wiring `sb_700_20.RegisterRoutes`
- `doJSONRequest`, `hasRelation`, `countRows`
- One `fetchSample{Table}` helper per table for fetching a real LIN/lin_zmm_lin from the test DB

### tests/sb_700_20/repository_test.go
Per table, with a nil DB:
- blank string param → `ErrEmptyParam`
- whitespace-only param → `ErrEmptyParam`
- page 0 → `ErrInvalidPage`
- page -1 → `ErrInvalidPage`

### tests/sb_700_20/handlers_test.go
Per table (each group skips via `hasRelation` if table absent in test DB):
- `GET …/search/%20%20` → 400
- `GET …/search/:realLIN` → 200 with correct model shape
- `GET …/search/NOTREAL` → 404
- `GET …/list?page=bad` → 400
- `GET …/list?page=0` → 400
- `GET …/list?page=1` → 200 with pagination fields (`items`, `count`, `page`, `total_pages`, `is_last_page`)
- `GET …/list` (no page param) → 200 (defaults to page 1)
- `GET …/list?page=999999` → 404
- Invalid DB → 500

## Error Responses

All error handling uses existing `response` package helpers:
- `response.NoItemFoundResponseMessage()` — 404
- `response.InternalErrorResponseMessage()` — 500
- `gin.H{"error": "..."}` — 400 (validation)

## Constraints

- Page size is hardcoded at 100 (matches TMDE)
- All endpoints are public — registered on `v1Route`, not `authRoutes`
- Jet-generated table/model packages are already present in `.gen/miltech_ng/public/`
- No new DB migrations required
