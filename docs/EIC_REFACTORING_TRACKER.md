# EIC (End Item Code) Refactor Tracker

**Created:** 2026-01-30
**Completed:** 2026-01-30
**Owner:** swisscheese
**Status:** ✅ Complete

## Completion Summary

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| **Total LOC** | 884 | 555 | 37% reduction |
| **Files** | 7 | 8 | Better organization |
| **Directories** | 4 | 1 | 75% consolidation |
| **SQL Duplication** | 4x repeated | 1x shared | Eliminated |
| **Error Handling** | String matching | Typed errors | Improved |

**Final Structure:**
```
api/eic/
├── route.go           # HTTP handlers and route registration
├── service.go         # Service interface
├── service_impl.go    # Business logic
├── repository.go      # Repository interface
├── repository_impl.go # Database queries (deduplicated)
├── query_builder.go   # Extracted SQL patterns
├── scanner.go         # Row scanning utilities
└── errors.go          # Typed domain errors
```

---

## Executive Summary

The `eic` (End Item Code) domain is a read-only data lookup service that currently follows the legacy monolithic controller/service/repository pattern spread across the traditional `api/controller`, `api/service`, `api/repository`, and `api/route` directories. This represents one of the largest remaining monolithic domains in the codebase with the repository implementation alone spanning 484 lines of code.

### Current Architecture Issues

1. **Massive Code Duplication**: The repository has 4 nearly identical SQL queries (each ~40 lines of column selection and GROUP BY clauses) repeated for GetByNIIN, GetByLIN, GetByFSCPaginated, and GetAllPaginated
2. **Large File Size**: `eic_repository_impl.go` is 484 LOC - the largest repository file in the codebase
3. **Scattered Files**: Code is spread across 4 separate directories (`controller`, `service`, `repository`, `route`) instead of being colocated
4. **Brittle Error Handling**: Controller uses string matching (`strings.Contains(err.Error(), "no EIC items found")`) instead of typed errors
5. **No Shared Utilities**: Pagination logic is duplicated instead of using shared utilities like `item_lookup/shared`
6. **Single Responsibility Violation**: While all queries relate to EIC data, the massive SQL duplication could be decomposed

### Refactoring Priority Assessment

| Criteria | Score | Notes |
|----------|-------|-------|
| Code Size | High | 484 LOC repository - largest in codebase |
| Duplication | High | Same 80-line SQL query repeated 4x |
| Pattern Violation | Medium | Uses old scattered pattern |
| Complexity | Low | Read-only, no authentication |
| Risk | Low | Simple data lookups, no state changes |
| Effort | Medium | Significant but straightforward |

**Recommendation**: This should be the NEXT domain refactored after `item_lookup` due to its size and code quality issues.

## Current State Analysis

### File Inventory (Old Pattern)

| File | Location | LOC | Purpose |
|------|----------|-----|---------|
| eic_controller.go | api/controller/ | 145 | HTTP handlers for all EIC lookups |
| eic_service.go | api/service/ | 12 | Interface definition (4 methods) |
| eic_service_impl.go | api/service/ | 80 | Service implementation |
| eic_repository.go | api/repository/ | 12 | Repository interface (4 methods) |
| eic_repository_impl.go | api/repository/ | 484 | Database queries (massive duplication) |
| eic_route.go | api/route/ | 23 | Route registration |
| eic_response.go | api/response/ | 128 | Response type definitions |
| **Total** | | **884** | |

### Current API Endpoints

```
GET /api/v1/eic/niin/:niin     # EIC records by NIIN
GET /api/v1/eic/lin/:lin       # EIC records by LIN
GET /api/v1/eic/fsc/:fsc       # EIC records by FSC (paginated)
GET /api/v1/eic/items          # All EIC records (paginated with optional search)
```

### Database Dependencies

| Context | Table | Description |
|---------|-------|-------------|
| eic | eic | End Item Code master table with 90+ columns |

### Code Duplication Analysis

**Critical Issue - Repeated SQL Patterns in Repository:**

The following column list and GROUP BY clause appear **4 times** (lines 38-67, 128-157, 225-254, 369-398):

```sql
SELECT
    inc, fsc, niin, eic, lin, nomen, model, eicc, ecc, cmdtycd, reported, dahr,
    publvl1, pubno1, pubdate1, pubchg1, pubcgdt1,
    -- ... 60+ more columns ...
    array_agg(DISTINCT uoeic ORDER BY uoeic) as uoeic_array,
    array_agg(DISTINCT mrc ORDER BY mrc) as mrc_array,
    COUNT(*) as variant_count
FROM eic
WHERE [condition]
GROUP BY inc, fsc, niin, eic, lin, nomen, model, ...
```

**Similarly, the scan function is repeated 4 times** (lines 77-96, 167-186, 265-284, 410-429):

```go
err := rows.Scan(
    &item.Inc, &item.Fsc, &item.Niin, &item.Eic, &item.Lin, &item.Nomen, &item.Model,
    // ... 80+ field scans ...
    pq.Array(&item.UoeicArray), pq.Array(&item.MrcArray), &item.VariantCount,
)
```

**Repeated Patterns in Controller:**
- Error handling with string matching (lines 36-40, 66-70, 103-107, 131-135)
- Response wrapping with StandardResponse

### Interface Analysis

**Current Repository Interface** (4 methods):
```go
type EICRepository interface {
    GetByNIIN(niin string) ([]response.EICConsolidatedItem, error)
    GetByLIN(lin string) ([]response.EICConsolidatedItem, error)
    GetByFSCPaginated(fsc string, page int) (response.EICPageResponse, error)
    GetAllPaginated(page int, search string) (response.EICPageResponse, error)
}
```

**Current Service Interface** (4 methods - thin wrapper):
```go
type EICService interface {
    LookupByNIIN(niin string) ([]response.EICConsolidatedItem, error)
    LookupByLIN(lin string) ([]response.EICConsolidatedItem, error)
    LookupByFSCPaginated(fsc string, page int) (response.EICPageResponse, error)
    LookupAllPaginated(page int, search string) (response.EICPageResponse, error)
}
```

## Proposed New Structure

### Option A: Simple Bounded Context (Recommended)

Since all operations are on the same `eic` table and serve the same purpose (EIC data lookup), decomposition by bounded context is not appropriate. Instead, focus on:

1. **Colocation**: Move all files into `api/eic/` directory
2. **Shared SQL Builder**: Extract common query building logic
3. **Typed Errors**: Replace string matching with typed errors
4. **Shared Utilities**: Use pagination helpers similar to `item_lookup/shared`

```
api/
  eic/
    route.go                    # HTTP handlers and route registration
    service.go                  # Service interface
    service_impl.go             # Service implementation
    repository.go               # Repository interface
    repository_impl.go          # Database queries
    query_builder.go            # Shared SQL query building (NEW)
    scanner.go                  # Shared row scanning (NEW)
    errors.go                   # Typed error definitions (NEW)
```

### Option B: Query Type Decomposition

If desired, could split by query type:

```
api/
  eic/
    route.go                    # Main router
    shared/
      errors.go                 # Common error definitions
      query.go                  # Shared SQL components
      scanner.go                # Row scanning utilities
    lookup/                     # NIIN/LIN direct lookups
      repository.go
      repository_impl.go
      service.go
      service_impl.go
      route.go
    browse/                     # Paginated browsing (FSC, all)
      repository.go
      repository_impl.go
      service.go
      service_impl.go
      route.go
```

**Recommendation**: Option A is simpler and more appropriate since all operations share the same data model and table.

## Implementation Checklist

### Phase 1: Foundation

- [ ] 1.1 Create directory structure
  ```bash
  mkdir -p api/eic
  ```

- [ ] 1.2 Create errors.go
  ```go
  package eic

  import "errors"

  var (
      ErrNotFound    = errors.New("no EIC items found")
      ErrEmptyParam  = errors.New("required parameter is empty")
      ErrInvalidPage = errors.New("page number must be greater than 0")
  )
  ```

- [ ] 1.3 Create query_builder.go
  ```go
  package eic

  // selectColumns returns the standard SELECT clause for EIC queries
  func selectColumns() string {
      return `
      SELECT
          inc, fsc, niin, eic, lin, nomen, model, eicc, ecc, cmdtycd, reported, dahr,
          publvl1, pubno1, pubdate1, pubchg1, pubcgdt1,
          publcl2, pubno2, pubdate2, pubchg2, pubcgdt2,
          publvl3, pubno3, pubdate3, pubchg3, pubcgdt3,
          publvl4, pubno4, pubdate4, pubchg4, pubcgdt4,
          publvl5, pubno5, pubdate5, pubchg5, pubcgdt5,
          publvl6, pubno6, pubdate6, pubchg6, pubcgdt6,
          publvl7, pubno7, pubdate7, pubchg7, pubcgdt7,
          pubremks, eqpmcsa, eqpmcsb, eqpmcsc, eqpmcsd, eqpmcse, eqpmcsf,
          eqpmcsg, eqpmcsh, eqpmcsi, eqpmcsj, eqpmcsk, eqpmcsl,
          wpnrec, sernotrk, orf, aoap, gainloss, usage, urm1, urm2,
          uom1, uom2, uom3, mau1, uom4, mau2,
          warranty, rbm, sos, erc, eslvl, oslin, lcc, nounabb,
          curfmc, prevfmc, bstat1, bstat2, matcat, itemmgr, eos, sorts, status, lst_updt,
          array_agg(DISTINCT uoeic ORDER BY uoeic) as uoeic_array,
          array_agg(DISTINCT mrc ORDER BY mrc) as mrc_array,
          COUNT(*) as variant_count
      FROM eic
      `
  }

  // groupByColumns returns the standard GROUP BY clause for EIC queries
  func groupByColumns() string {
      return `
      GROUP BY inc, fsc, niin, eic, lin, nomen, model, eicc, ecc, cmdtycd, reported, dahr,
          publvl1, pubno1, pubdate1, pubchg1, pubcgdt1,
          publcl2, pubno2, pubdate2, pubchg2, pubcgdt2,
          publvl3, pubno3, pubdate3, pubchg3, pubcgdt3,
          publvl4, pubno4, pubdate4, pubchg4, pubcgdt4,
          publvl5, pubno5, pubdate5, pubchg5, pubcgdt5,
          publvl6, pubno6, pubdate6, pubchg6, pubcgdt6,
          publvl7, pubno7, pubdate7, pubchg7, pubcgdt7,
          pubremks, eqpmcsa, eqpmcsb, eqpmcsc, eqpmcsd, eqpmcse, eqpmcsf,
          eqpmcsg, eqpmcsh, eqpmcsi, eqpmcsj, eqpmcsk, eqpmcsl,
          wpnrec, sernotrk, orf, aoap, gainloss, usage, urm1, urm2,
          uom1, uom2, uom3, mau1, uom4, mau2,
          warranty, rbm, sos, erc, eslvl, oslin, lcc, nounabb,
          curfmc, prevfmc, bstat1, bstat2, matcat, itemmgr, eos, sorts, status, lst_updt
      `
  }
  ```

- [ ] 1.4 Create scanner.go
  ```go
  package eic

  import (
      "database/sql"
      "github.com/lib/pq"
      "miltechserver/api/response"
  )

  // scanConsolidatedItem scans a row into an EICConsolidatedItem
  func scanConsolidatedItem(rows *sql.Rows) (response.EICConsolidatedItem, error) {
      var item response.EICConsolidatedItem
      err := rows.Scan(
          &item.Inc, &item.Fsc, &item.Niin, &item.Eic, &item.Lin, &item.Nomen, &item.Model,
          &item.Eicc, &item.Ecc, &item.Cmdtycd, &item.Reported, &item.Dahr,
          &item.Publvl1, &item.Pubno1, &item.Pubdate1, &item.Pubchg1, &item.Pubcgdt1,
          &item.Publcl2, &item.Pubno2, &item.Pubdate2, &item.Pubchg2, &item.Pubcgdt2,
          &item.Publvl3, &item.Pubno3, &item.Pubdate3, &item.Pubchg3, &item.Pubcgdt3,
          &item.Publvl4, &item.Pubno4, &item.Pubdate4, &item.Pubchg4, &item.Pubcgdt4,
          &item.Publvl5, &item.Pubno5, &item.Pubdate5, &item.Pubchg5, &item.Pubcgdt5,
          &item.Publvl6, &item.Pubno6, &item.Pubdate6, &item.Pubchg6, &item.Pubcgdt6,
          &item.Publvl7, &item.Pubno7, &item.Pubdate7, &item.Pubchg7, &item.Pubcgdt7,
          &item.Pubremks, &item.Eqpmcsa, &item.Eqpmcsb, &item.Eqpmcsc, &item.Eqpmcsd,
          &item.Eqpmcse, &item.Eqpmcsf, &item.Eqpmcsg, &item.Eqpmcsh, &item.Eqpmcsi,
          &item.Eqpmcsj, &item.Eqpmcsk, &item.Eqpmcsl, &item.Wpnrec, &item.Sernotrk,
          &item.Orf, &item.Aoap, &item.Gainloss, &item.Usage, &item.Urm1, &item.Urm2,
          &item.Uom1, &item.Uom2, &item.Uom3, &item.Mau1, &item.Uom4, &item.Mau2,
          &item.Warranty, &item.Rbm, &item.Sos, &item.Erc, &item.Eslvl, &item.Oslin,
          &item.Lcc, &item.Nounabb, &item.Curfmc, &item.Prevfmc, &item.Bstat1, &item.Bstat2,
          &item.Matcat, &item.Itemmgr, &item.Eos, &item.Sorts, &item.Status, &item.LstUpdt,
          pq.Array(&item.UoeicArray), pq.Array(&item.MrcArray), &item.VariantCount,
      )
      return item, err
  }
  ```

### Phase 2: Core Implementation

- [ ] 2.1 Create repository.go (interface)
- [ ] 2.2 Create repository_impl.go (refactored with query builder)
- [ ] 2.3 Create service.go (interface)
- [ ] 2.4 Create service_impl.go
- [ ] 2.5 Create route.go (handlers + route registration)

### Phase 3: Wiring

- [ ] 3.1 Create Dependencies struct and RegisterRoutes function
  ```go
  package eic

  import (
      "database/sql"
      "github.com/gin-gonic/gin"
  )

  type Dependencies struct {
      DB *sql.DB
  }

  func RegisterRoutes(deps Dependencies, router *gin.RouterGroup) {
      repo := NewRepository(deps.DB)
      svc := NewService(repo)
      RegisterHandlers(router, svc)
  }
  ```

- [ ] 3.2 Update main route registration in api/route/route.go
  ```go
  // Replace:
  NewEICRouter(db, v1Route)

  // With:
  eic.RegisterRoutes(eic.Dependencies{DB: db}, v1Route)
  ```

- [ ] 3.3 Remove legacy files after verification

### Phase 4: Verification & Cleanup

- [ ] 4.1 Create integration tests for EIC endpoints
- [ ] 4.2 Manual API testing (verify all 4 endpoints work)
- [ ] 4.3 Remove legacy files:
  - api/controller/eic_controller.go
  - api/service/eic_service.go
  - api/service/eic_service_impl.go
  - api/repository/eic_repository.go
  - api/repository/eic_repository_impl.go
  - api/route/eic_route.go
- [ ] 4.4 Keep response types in api/response/eic_response.go (shared location)

## Expected Metrics Improvement

| Metric | Current | Target | Improvement |
|--------|---------|--------|-------------|
| Total LOC | 884 | ~400 | 55% reduction |
| Repository LOC | 484 | ~150 | 69% reduction |
| Duplicated SQL | 4x | 1x | 75% reduction |
| Files | 7 (scattered) | 7 (colocated) | Better organization |
| Largest file | 484 LOC | <150 LOC | Clean separation |

## Risk Assessment

### Low Risk
- Read-only operations (no data modification)
- No authentication requirements
- Single table dependency
- No cross-domain dependencies
- Clear, well-defined API contract

### Medium Risk
- Large SQL queries require careful extraction
- Response type is complex (90+ fields)

### Mitigation
- Extract SQL incrementally, test after each change
- Keep response type in shared location (api/response/)
- Comprehensive endpoint testing before removing legacy code

## Comparison with Other Refactored Domains

| Domain | Bounded Contexts | Complexity | Pattern |
|--------|------------------|------------|---------|
| shops | 7 (core, lists, members, messages, settings, vehicles, facade) | High | Full decomposition with shared auth |
| equipment_services | 5 (core, queries, calendar, status, completion) | Medium | Full decomposition with shared auth |
| user_saves | 5 (categories, images, quick, serialized, facade) | Medium | Feature-based decomposition |
| item_lookup | 4 (lin, uoc, cage, substitute) | Low | Read-only lookups with shared utils |
| **eic** (proposed) | 1 (single context with shared utilities) | Low | Colocated with extracted utilities |

Key difference: `eic` is a single bounded context - all queries operate on the same table with the same response model. The refactoring focus is on **eliminating duplication** rather than **decomposing boundaries**.

## Progress Log

- 2026-01-30: Initial analysis and planning complete
- 2026-01-31: Implemented new `api/eic` module and wired routes; legacy files retained pending verification
- 2026-01-31: Removed legacy EIC files; added EIC tests and confirmed `go test ./tests/eic` passes

## Dependencies on Main Route Registration

**Current Registration** (in `api/route/route.go` line 35):
```go
NewEICRouter(db, v1Route)
```

**Proposed Registration**:
```go
eic.RegisterRoutes(eic.Dependencies{DB: db}, v1Route)
```

## Notes

- Response types should remain in `api/response/eic_response.go` since they define the API contract
- The query builder pattern can potentially be reused by other domains with similar SQL patterns
- Consider adding OpenAPI/Swagger documentation during refactor
- No authentication middleware required (all public routes)
