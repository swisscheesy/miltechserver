# Item Lookup Refactor Tracker

**Created:** 2026-01-30
**Owner:** swisscheese
**Status:** Implemented (Pending Verification)

## Executive Summary

The `item_lookup` domain is a read-only data lookup service that currently combines multiple unrelated data sources into a single monolithic controller/service/repository. This design violates the Single Responsibility Principle and creates unnecessary coupling between distinct bounded contexts.

### Current Architecture Issues

1. **Mixed Concerns**: LIN lookups, UOC lookups, CAGE lookups, and Substitute LIN lookups are all bundled together despite being entirely separate data domains
2. **Large Interface**: The `ItemLookupRepository` interface has 8 methods spanning 4 different data tables/views
3. **No Clear Boundaries**: A change to LIN lookup logic could inadvertently affect UOC lookups due to shared code paths
4. **Poor Cohesion**: Each lookup type uses different database tables/views but shares controller and service layers

### Proposed Solution

Decompose the monolithic `item_lookup` domain into 4 distinct bounded contexts:
- **lin/** - LIN (Line Item Number) lookups by page and NIIN
- **uoc/** - UOC (Unit of Consumption) lookups by page, specific code, and model
- **cage/** - CAGE (Commercial and Government Entity) address lookups
- **substitute/** - Army Substitute LIN lookups

## Current State Analysis

### File Inventory (Old Pattern)

| File | Location | LOC | Purpose |
|------|----------|-----|---------|
| item_lookup_controller.go | api/controller/ | 243 | HTTP handlers for all lookup types |
| item_lookup_service.go | api/service/ | 19 | Interface definition (8 methods) |
| item_lookup_service_impl.go | api/service/ | 127 | Service implementation |
| item_lookup_repository.go | api/repository/ | 19 | Repository interface (8 methods) |
| item_lookup_repository_impl.go | api/repository/ | 278 | Database queries |
| item_lookup_route.go | api/route/ | 29 | Route registration |
| **Total** | | **715** | |

Legacy files removed on 2026-01-30; current implementation lives under `api/item_lookup/`.

### Current API Endpoints

```
GET /api/v1/lookup/lin                    # Paginated LIN list
GET /api/v1/lookup/lin/by-niin/:niin      # LIN by NIIN
GET /api/v1/lookup/niin/by-lin/:lin       # NIIN by LIN
GET /api/v1/lookup/lin/lin/:niin          # Legacy LIN by NIIN
GET /api/v1/lookup/lin/niin/:lin          # Legacy NIIN by LIN
GET /api/v1/lookup/substitute-lin         # All substitute LIN records
GET /api/v1/lookup/cage/:cage             # CAGE address by code
GET /api/v1/lookup/uoc                    # Paginated UOC list
GET /api/v1/lookup/uoc/:uoc               # Specific UOC
GET /api/v1/lookup/uoc/by-model/:model    # UOC by vehicle model
GET /api/v1/lookup/uoc/model/:model       # Legacy UOC by vehicle model
```

### Database Dependencies

| Context | Table/View | Description |
|---------|------------|-------------|
| lin | lookup_lin_niin (view) | LIN to NIIN mapping view |
| uoc | lookup_uoc (table) | Unit of Consumption table |
| cage | cage_address (table) | CAGE code addresses |
| substitute | army_substitute_lin (table) | Army substitute LIN data |

### Code Duplication Analysis

**Repeated Patterns in Repository:**
- Pagination logic (identical in `SearchLINByPage` and `SearchUOCByPage`)
- Empty string validation
- "No items found" error handling
- Count queries for pagination

**Repeated Patterns in Controller:**
- Error handling (checking for "no item" in error message - brittle)
- Parameter extraction and validation
- Response wrapping

## Proposed New Structure

```
api/
  item_lookup/
    route.go                    # Main router that composes all contexts
    shared/
      errors.go                 # Common error definitions
      pagination.go             # Shared pagination utilities
      response.go               # Common response helpers
    lin/
      repository.go             # LIN repository interface
      repository_impl.go        # LIN database queries
      service.go                # LIN service interface
      service_impl.go           # LIN business logic
      route.go                  # LIN HTTP handlers and routes
    uoc/
      repository.go             # UOC repository interface
      repository_impl.go        # UOC database queries
      service.go                # UOC service interface
      service_impl.go           # UOC business logic
      route.go                  # UOC HTTP handlers and routes
    cage/
      repository.go             # CAGE repository interface
      repository_impl.go        # CAGE database queries
      service.go                # CAGE service interface
      service_impl.go           # CAGE business logic
      route.go                  # CAGE HTTP handlers and routes
    substitute/
      repository.go             # Substitute LIN repository interface
      repository_impl.go        # Substitute LIN database queries
      service.go                # Substitute LIN service interface
      service_impl.go           # Substitute LIN business logic
      route.go                  # Substitute LIN HTTP handlers and routes
```

### API Endpoint Improvements

Consider improving route clarity:

```
# Previous (legacy, still supported)
GET /api/v1/lookup/lin/lin/:niin    # "lin/lin" is confusing

# Proposed (clearer)
GET /api/v1/lookup/lin              # Paginated LIN list
GET /api/v1/lookup/lin/by-niin/:niin    # Find LIN by NIIN
GET /api/v1/lookup/niin/by-lin/:lin     # Find NIIN by LIN
GET /api/v1/lookup/substitute-lin       # All substitute LIN records
GET /api/v1/lookup/cage/:cage           # CAGE address by code
GET /api/v1/lookup/uoc                  # Paginated UOC list
GET /api/v1/lookup/uoc/:uoc             # Specific UOC
GET /api/v1/lookup/uoc/by-model/:model  # UOC by vehicle model
```

Note: Backward-compatible legacy routes are retained; clients can migrate to new routes without breaking.

## Implementation Checklist

### Phase 1: Foundation

- [x] 1.1 Create directory structure
  ```bash
  mkdir -p api/item_lookup/shared
  mkdir -p api/item_lookup/lin
  mkdir -p api/item_lookup/uoc
  mkdir -p api/item_lookup/cage
  mkdir -p api/item_lookup/substitute
  ```

- [x] 1.2 Create shared/errors.go
  ```go
  package shared

  import "errors"

  var (
      ErrNotFound      = errors.New("no items found")
      ErrEmptyParam    = errors.New("required parameter is empty")
      ErrInvalidPage   = errors.New("page number must be greater than 0")
  )
  ```

- [x] 1.3 Create shared/pagination.go
  ```go
  package shared

  const DefaultPageSize = int64(20)

  type PagedResponse struct {
      Count      int  `json:"count"`
      Page       int  `json:"page"`
      TotalPages int  `json:"total_pages"`
      IsLastPage bool `json:"is_last_page"`
  }

  func CalculateTotalPages(totalCount int, pageSize int64) int {
      return int(math.Ceil(float64(totalCount) / float64(pageSize)))
  }

  func CalculateOffset(page int, pageSize int64) int64 {
      return pageSize * int64(page-1)
  }
  ```

- [x] 1.4 Create shared/response.go
  ```go
  package shared

  import (
      "net/http"
      "strings"
      "github.com/gin-gonic/gin"
      "miltechserver/api/response"
  )

  func HandleError(c *gin.Context, err error) {
      if errors.Is(err, ErrNotFound) || strings.Contains(err.Error(), "no items") {
          c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
          return
      }
      if errors.Is(err, ErrEmptyParam) || errors.Is(err, ErrInvalidPage) {
          c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
          return
      }
      c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
  }
  ```

- [x] 1.5 Create central route.go skeleton

### Phase 2: Bounded Contexts

- [x] 2.1 Extract LIN context (api/item_lookup/lin/)
  - Move LIN-related methods from repository
  - Move LIN-related service methods
  - Create LIN-specific handlers
  - Register LIN routes

- [x] 2.2 Extract UOC context (api/item_lookup/uoc/)
  - Move UOC-related methods from repository
  - Move UOC-related service methods
  - Create UOC-specific handlers
  - Register UOC routes

- [x] 2.3 Extract CAGE context (api/item_lookup/cage/)
  - Move CAGE-related methods from repository
  - Move CAGE-related service methods
  - Create CAGE-specific handlers
  - Register CAGE routes

- [x] 2.4 Extract Substitute context (api/item_lookup/substitute/)
  - Move substitute LIN methods from repository
  - Move substitute LIN service methods
  - Create substitute-specific handlers
  - Register substitute routes

### Phase 3: Wiring

- [x] 3.1 Wire dependencies in item_lookup/route.go
  ```go
  package item_lookup

  import (
      "database/sql"
      "github.com/gin-gonic/gin"
      "miltechserver/api/item_lookup/lin"
      "miltechserver/api/item_lookup/uoc"
      "miltechserver/api/item_lookup/cage"
      "miltechserver/api/item_lookup/substitute"
  )

  type Dependencies struct {
      DB *sql.DB
  }

  func RegisterRoutes(deps Dependencies, router *gin.RouterGroup) {
      linRepo := lin.NewRepository(deps.DB)
      linService := lin.NewService(linRepo)
      lin.RegisterRoutes(router, linService)

      uocRepo := uoc.NewRepository(deps.DB)
      uocService := uoc.NewService(uocRepo)
      uoc.RegisterRoutes(router, uocService)

      cageRepo := cage.NewRepository(deps.DB)
      cageService := cage.NewService(cageRepo)
      cage.RegisterRoutes(router, cageService)

      substituteRepo := substitute.NewRepository(deps.DB)
      substituteService := substitute.NewService(substituteRepo)
      substitute.RegisterRoutes(router, substituteService)
  }
  ```

- [x] 3.2 Update main route registration in api/route/route.go

- [x] 3.3 Remove legacy controller/service/repository/routes

### Phase 4: Verification & Cleanup

- [x] 4.1 Create integration tests for each bounded context
- [ ] 4.2 Manual API testing (verify all endpoints work)
- [x] 4.3 Remove legacy files (after verification)
- [x] 4.4 Update any documentation referencing old structure

## File-by-File Migration Plan

### Step 1: Create LIN Bounded Context

**api/item_lookup/lin/repository.go**
```go
package lin

import (
    "miltechserver/.gen/miltech_ng/public/model"
    "miltechserver/api/response"
)

type Repository interface {
    SearchByPage(page int) (response.LINPageResponse, error)
    SearchByNIIN(niin string) ([]model.LookupLinNiin, error)
    SearchNIINByLIN(lin string) ([]model.LookupLinNiin, error)
}
```

**api/item_lookup/lin/repository_impl.go**
- Extract lines 30-127 from item_lookup_repository_impl.go
- Refactor to use shared pagination utilities

**api/item_lookup/lin/service.go**
```go
package lin

import (
    "miltechserver/.gen/miltech_ng/public/model"
    "miltechserver/api/response"
)

type Service interface {
    LookupByPage(page int) (response.LINPageResponse, error)
    LookupByNIIN(niin string) ([]model.LookupLinNiin, error)
    LookupNIINByLIN(lin string) ([]model.LookupLinNiin, error)
}
```

**api/item_lookup/lin/service_impl.go**
- Extract lines 22-60 from item_lookup_service_impl.go

**api/item_lookup/lin/route.go**
- Extract handlers from lines 25-110 of item_lookup_controller.go
- Combine controller and route into single file per bounded context pattern

### Step 2: Create UOC Bounded Context

**api/item_lookup/uoc/repository.go**
```go
package uoc

import (
    "miltechserver/.gen/miltech_ng/public/model"
    "miltechserver/api/response"
)

type Repository interface {
    SearchByPage(page int) (response.UOCPageResponse, error)
    SearchSpecific(uoc string) ([]model.LookupUoc, error)
    SearchByModel(model string) ([]model.LookupUoc, error)
}
```

**api/item_lookup/uoc/repository_impl.go**
- Extract lines 177-278 from item_lookup_repository_impl.go

**api/item_lookup/uoc/service.go & service_impl.go**
- Extract lines 91-127 from item_lookup_service_impl.go

**api/item_lookup/uoc/route.go**
- Extract handlers from lines 159-243 of item_lookup_controller.go

### Step 3: Create CAGE Bounded Context

**api/item_lookup/cage/repository.go**
```go
package cage

import "miltechserver/.gen/miltech_ng/public/model"

type Repository interface {
    SearchByCode(cage string) ([]model.CageAddress, error)
}
```

**api/item_lookup/cage/repository_impl.go**
- Extract lines 150-175 from item_lookup_repository_impl.go

**api/item_lookup/cage/service.go & service_impl.go**
- Extract lines 74-85 from item_lookup_service_impl.go

**api/item_lookup/cage/route.go**
- Extract handler from lines 132-156 of item_lookup_controller.go

### Step 4: Create Substitute Bounded Context

**api/item_lookup/substitute/repository.go**
```go
package substitute

import "miltechserver/.gen/miltech_ng/public/model"

type Repository interface {
    SearchAll() ([]model.ArmySubstituteLin, error)
}
```

**api/item_lookup/substitute/repository_impl.go**
- Extract lines 129-148 from item_lookup_repository_impl.go

**api/item_lookup/substitute/service.go & service_impl.go**
- Extract lines 62-72 from item_lookup_service_impl.go

**api/item_lookup/substitute/route.go**
- Extract handler from lines 112-130 of item_lookup_controller.go

## Metrics

| Metric | Current | Target | Notes |
|--------|---------|--------|-------|
| Largest file (LOC) | 113 (lin/uoc repository_impl) | < 100 | Slightly above target |
| Methods per interface | 1-3 | 2-3 | Per context |
| Code duplication | Low | Low | Shared utilities in shared/ |
| Bounded contexts | 4 | 4 | Clear separation |
| Testability | High | High | Integration tests added |

## Risk Assessment

### Low Risk
- Read-only operations (no data modification)
- Clear separation between data sources
- No cross-context dependencies

### Medium Risk
- Route changes may affect API clients
- Need to verify all endpoints work after migration

### Mitigation
- Coordinate client updates for new route names
- Comprehensive endpoint testing before deployment

## Progress Log

- 2026-01-30: Initial analysis and planning complete
- 2026-01-30: Refactor implementation complete (pending manual verification)
- 2026-01-30: Integration tests passing (`go test ./tests/item_lookup -v`)

## Response Types Inventory

The following response types in `api/response/` are used by this domain:

| File | Type | Used By |
|------|------|---------|
| lin_page_response.go | LINPageResponse | LIN page + search responses |
| uoc_page_response.go | UOCPageResponse | UOC page + search responses |

## Dependencies on Main Route Registration

**Current Registration** (in `api/route/route.go`):
```go
item_lookup.RegisterRoutes(item_lookup.Dependencies{DB: db}, v1Route)
```

## Comparison with Other Refactored Domains

| Domain | Bounded Contexts | Complexity | Pattern |
|--------|------------------|------------|---------|
| shops | 7 (core, lists, members, messages, settings, vehicles, facade) | High | Full decomposition with shared auth |
| equipment_services | 5 (core, queries, calendar, status, completion) | Medium | Full decomposition with shared auth |
| user_saves | 5 (categories, images, quick, serialized, facade) | Medium | Feature-based decomposition |
| **item_lookup** | 4 (lin, uoc, cage, substitute) | Low | Read-only lookups |

Key difference: `item_lookup` has no authentication requirements (all public endpoints), making it simpler than previous refactors.

## Notes

- Consider adding OpenAPI/Swagger documentation during refactor
- Pagination logic shared via `api/item_lookup/shared`
- Error handling uses typed errors
- Route naming clarified (`/lin/by-niin/:niin`, `/niin/by-lin/:lin`) with legacy routes still supported
- Response types consolidated to 2
- No authentication middleware required (all public routes)
- Legacy routes retained for backward compatibility
