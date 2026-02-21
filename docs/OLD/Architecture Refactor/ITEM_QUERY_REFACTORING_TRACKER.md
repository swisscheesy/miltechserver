# Item Query Domain Refactor Tracker

**Created:** 2026-01-30
**Owner:** swisscheese
**Status:** Planning (Not Started)

## Executive Summary

The `item_query` domain is a complex data retrieval service that combines two distinct functionalities: "short" item queries (basic NIIN/part lookups) and "detailed" item queries (multi-table aggregation with 10+ database tables). This domain has **critical technical debt** that needs to be addressed, including a dangerous pattern of ignoring errors from detail query methods.

### Current Architecture Issues

**Critical:**
1. **Ignored Errors (Dangerous)**: The `GetDetailedItemData` method in repository ignores errors from 9 out of 10 helper methods using `_, _ := repo.getXxxData(niin)` pattern. This silently swallows database errors.
2. **String-Based Error Matching**: Controller uses `strings.Contains(err.Error(), "no item")` - brittle pattern

**Major:**
3. **Massive Repository File**: `item_detailed_repository_impl.go` is 517 LOC with 10 separate query methods
4. **Mixed Concerns**: Combines "short" queries (lightweight) with "detailed" queries (heavy multi-table joins)
5. **Tight Analytics Coupling**: Analytics tracking is embedded directly in service layer

**Minor:**
6. **Scattered Files**: Code spread across 6 directories
7. **Inconsistent Naming**: `item_query_controller.go` uses `ItemShortService` and `ItemDetailedService` - mixed naming

### Refactoring Priority Assessment

| Criteria | Score | Notes |
|----------|-------|-------|
| Code Size | High | 826 LOC total, 517 in one file |
| Duplication | Medium | Repeated query patterns |
| Pattern Violation | High | Ignores errors, string matching |
| Complexity | High | 10+ database tables, analytics |
| Risk | Medium | Error handling changes needed |
| Effort | High | Requires careful decomposition |

**Recommendation**: Priority 4 - Complex refactoring that requires careful planning. The error-ignoring pattern should be fixed regardless of full refactoring decision.

## Current State Analysis

### File Inventory (Old Pattern)

| File | Location | LOC | Purpose |
|------|----------|-----|---------|
| item_query_controller.go | api/controller/ | 77 | HTTP handlers for short/detailed queries |
| item_short_service.go | api/service/ | 10 | Short query interface |
| item_short_service_impl.go | api/service/ | 86 | Short query implementation + analytics |
| item_detailed_service.go | api/service/ | 7 | Detailed query interface |
| item_detailed_service_impl.go | api/service/ | 18 | Detailed query implementation (thin wrapper) |
| item_query_repository.go | api/repository/ | 10 | Short query repository interface |
| item_query_repository_impl.go | api/repository/ | 64 | Short query database operations |
| item_detailed_repository.go | api/repository/ | 7 | Detailed query repository interface |
| item_detailed_repository_impl.go | api/repository/ | 517 | Detailed query database operations |
| item_query_route.go | api/route/ | 30 | Route registration and DI |
| **Total** | | **826** | |

### Current API Endpoints

```
GET /api/v1/queries/items/initial?method=niin&value=123456789  # Short query by NIIN
GET /api/v1/queries/items/initial?method=part&value=ABC123     # Short query by part number
GET /api/v1/queries/items/detailed?niin=123456789              # Detailed item data
```

### Database Dependencies

| Context | Table/View | Description |
|---------|------------|-------------|
| short | niin_lookup (view) | Basic item lookup view |
| short | part_number | Part number to NIIN mapping |
| detailed | army_master_data_file | Core AMDF data |
| detailed | amdf_management | AMDF management info |
| detailed | amdf_credit | AMDF credit data |
| detailed | amdf_billing | AMDF billing data |
| detailed | amdf_matcat | AMDF material category |
| detailed | amdf_phrase | AMDF phrases |
| detailed | amdf_i_and_s | AMDF I&S data |
| detailed | army_line_item_number | LIN data |
| detailed | army_packaging_and_freight | Packaging data |
| detailed | army_packaging_1, _2 | Additional packaging |
| detailed | army_freight | Freight data |
| detailed | army_sarsscat | SARSS-CAT data |
| detailed | flis_* (multiple tables) | FLIS data (10+ tables) |
| detailed | cage_address | CAGE addresses |
| detailed | cage_status_and_type | CAGE status |
| detailed | dss_weight_and_cube | DSS data |
| detailed | disposition | Disposition data |
| detailed | component_end_item | Component/end item data |

### Critical Issue: Ignored Errors

**Location**: `api/repository/item_detailed_repository_impl.go` lines 25-44

```go
func (repo *ItemDetailedRepositoryImpl) GetDetailedItemData(niin string) (response.DetailedResponse, error) {
    amdfData, _ := repo.getAmdfData(niin)           // ERROR IGNORED
    armyPackData, _ := repo.getArmyPackagingAndFreight(niin)  // ERROR IGNORED
    sarsscatData, _ := repo.getSarsscat(niin)       // ERROR IGNORED
    identificationData, _ := repo.getIdentification(niin)     // ERROR IGNORED
    managementData, _ := repo.getManagement(niin)   // ERROR IGNORED
    referenceData, _ := repo.getReference(niin)     // ERROR IGNORED
    freightData, _ := repo.getFreight(niin)         // ERROR IGNORED
    packagingData, _ := repo.getPackaging(niin)     // ERROR IGNORED
    characteristicsData, _ := repo.getCharacteristics(niin)   // ERROR IGNORED
    dispositionData, _ := repo.getDisposition(niin) // ERROR IGNORED

    fullDetailedItem := response.DetailedResponse{...}
    return fullDetailedItem, nil  // ALWAYS returns nil error
}
```

**Impact**:
- Database connection failures are silently ignored
- Query errors result in partial/empty data without indication
- Debugging production issues becomes very difficult
- Users see incomplete data with no error indication

**Recommended Fix**:
1. Collect all errors in a slice
2. Return partial data with error summary
3. Or fail fast on first critical error
4. Log all errors with structured logging

### Code Duplication Analysis

**Repeated Query Patterns**:
Each helper method in `item_detailed_repository_impl.go` follows identical pattern:
```go
func (repo *ItemDetailedRepositoryImpl) getXxxData(niin string) (details.Xxx, error) {
    xxx := details.Xxx{}
    stmt := SELECT(table.Xxx.AllColumns).FROM(table.Xxx).WHERE(table.Xxx.Niin.EQ(String(niin)))
    err := stmt.Query(repo.Db, &xxx)
    if err != nil {
        return details.Xxx{}, err
    }
    return xxx, nil
}
```

This pattern is repeated ~25 times with minor variations.

### Analytics Integration

**Current Pattern** (in `item_short_service_impl.go`):
```go
func (service *ItemShortServiceImpl) FindShortByNiin(niin string) (model.NiinLookup, error) {
    val, err := service.ItemQueryRepository.ShortItemSearchNiin(niin)
    if err != nil {
        return model.NiinLookup{}, err
    }
    // Analytics call embedded in service
    service.trackItemSearchSuccess(normalizedNiin, nomenclature)
    return val, nil
}
```

**Issue**: Analytics is tightly coupled to business logic.

## Proposed New Structure

### Decomposition Strategy

Split into two bounded contexts based on distinct use cases:

```
api/
  item_query/
    route.go                    # Main router
    shared/
      errors.go                 # Common error definitions
      analytics.go              # Analytics wrapper (decoupled)
    short/
      route.go                  # HTTP handlers
      service.go                # Service interface
      service_impl.go           # Business logic
      repository.go             # Repository interface
      repository_impl.go        # Database queries
    detailed/
      route.go                  # HTTP handlers
      service.go                # Service interface
      service_impl.go           # Business logic
      repository.go             # Repository interface
      repository_impl.go        # Database queries (refactored)
      queries/                  # Extracted query helpers
        amdf.go                 # AMDF-related queries
        packaging.go            # Packaging-related queries
        flis.go                 # FLIS-related queries
        reference.go            # Reference/CAGE queries
```

### Error Handling Strategy

**For detailed queries**, implement partial success pattern:

```go
type DetailedQueryResult struct {
    Data     DetailedResponse
    Errors   []QueryError
    Complete bool
}

type QueryError struct {
    Source  string  // "amdf", "packaging", etc.
    Message string
    Fatal   bool    // If true, entire query should fail
}

func (repo *Repository) GetDetailedItemData(niin string) (DetailedQueryResult, error) {
    result := DetailedQueryResult{Complete: true}

    amdfData, err := repo.getAmdfData(niin)
    if err != nil {
        if isCritical(err) {
            return DetailedQueryResult{}, fmt.Errorf("critical AMDF error: %w", err)
        }
        result.Errors = append(result.Errors, QueryError{Source: "amdf", Message: err.Error()})
        result.Complete = false
    }
    result.Data.Amdf = amdfData

    // ... repeat for other queries

    return result, nil
}
```

## Implementation Checklist

### Phase 0: Critical Fix (Do First)

- [ ] 0.1 **Fix ignored errors in item_detailed_repository_impl.go**
  - Add error collection and reporting
  - Log all query errors
  - Return error summary to caller

- [ ] 0.2 **Replace string-based error matching in controller**
  - Define typed errors
  - Use `errors.Is()` pattern

### Phase 1: Foundation

- [ ] 1.1 Create directory structure
  ```bash
  mkdir -p api/item_query/shared
  mkdir -p api/item_query/short
  mkdir -p api/item_query/detailed
  mkdir -p api/item_query/detailed/queries
  ```

- [ ] 1.2 Create shared/errors.go
  ```go
  package shared

  import "errors"

  var (
      ErrNotFound     = errors.New("no items found")
      ErrInvalidNiin  = errors.New("invalid NIIN format")
      ErrInvalidPart  = errors.New("invalid part number")
      ErrPartialData  = errors.New("partial data returned due to query errors")
  )
  ```

- [ ] 1.3 Create shared/analytics.go (decoupled)
  ```go
  package shared

  type AnalyticsTracker interface {
      TrackItemSearch(niin, nomenclature string)
  }

  type NoOpTracker struct{}
  func (NoOpTracker) TrackItemSearch(string, string) {}
  ```

### Phase 2: Short Query Context

- [ ] 2.1 Create short/repository.go and repository_impl.go
- [ ] 2.2 Create short/service.go and service_impl.go
- [ ] 2.3 Create short/route.go with handlers
- [ ] 2.4 Wire analytics through interface

### Phase 3: Detailed Query Context

- [ ] 3.1 Extract query helpers into detailed/queries/*.go
- [ ] 3.2 Implement error collection in repository
- [ ] 3.3 Create detailed/service.go and service_impl.go
- [ ] 3.4 Create detailed/route.go with handlers
- [ ] 3.5 Update response type to include error info

### Phase 4: Wiring

- [ ] 4.1 Create item_query/route.go to compose both contexts
- [ ] 4.2 Update api/route/route.go registration
- [ ] 4.3 Remove legacy files after verification

### Phase 5: Verification

- [ ] 5.1 Test short query endpoints
- [ ] 5.2 Test detailed query endpoints
- [ ] 5.3 Verify analytics still tracks correctly
- [ ] 5.4 Verify error logging works
- [ ] 5.5 Remove legacy files

## Expected Metrics Improvement

| Metric | Current | Target | Improvement |
|--------|---------|--------|-------------|
| Total LOC | 826 | ~700 | 15% reduction |
| Largest file | 517 LOC | <150 LOC | 71% reduction |
| Ignored errors | 10 | 0 | 100% fix |
| String matching | Yes | No | Fixed |
| Files | 10 (scattered) | 12 (organized) | Better structure |

## Risk Assessment

### High Risk
- **Error handling changes may affect client behavior**
  - Currently returns empty/partial data silently
  - Fix will return errors or partial data indication
  - May need client-side updates

### Medium Risk
- Analytics decoupling requires careful testing
- Large number of database tables means many failure points
- Response type changes for partial data support

### Mitigation
- Add feature flag for new error handling behavior
- Comprehensive logging during transition
- Staged rollout with monitoring
- Keep legacy endpoint available during transition
- Document API changes for clients

## Dependencies

### External Services
- Analytics service (internal)

### Shared Code (api/details/)
10 files defining detailed response structures:
- `amdf.go` (AMDF data structures)
- `army_packaging_and_freight.go`
- `characteristics.go`
- `disposition.go`
- `freight.go`
- `identification.go`
- `management.go`
- `packaging.go`
- `reference.go`
- `sarsscat.go`

Total: 120 LOC in api/details/

These should remain in place as they define the API response contract.

## Progress Log

- 2026-01-30: Initial analysis and planning complete

## Notes

- This is the most complex remaining domain due to multi-table aggregation
- The error-ignoring pattern is a critical bug that should be fixed even without full refactoring
- Consider implementing caching for detailed queries (expensive multi-table joins)
- The detailed query hits 20+ database tables - consider query optimization
- Response types in api/details/ and api/response/ are part of API contract - handle with care
- Analytics decoupling is recommended but not required for basic refactoring
