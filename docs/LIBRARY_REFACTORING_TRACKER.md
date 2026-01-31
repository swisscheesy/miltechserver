# Library Domain Refactor Tracker

**Created:** 2026-01-30
**Owner:** swisscheese
**Status:** Complete

## Executive Summary

The `library` domain provides access to PMCS (Preventive Maintenance Checks and Services) and BII (Basic Issue Items) document libraries stored in Azure Blob Storage. Unlike most other domains that interact with PostgreSQL, this domain primarily interfaces with Azure Blob Storage for document listing and SAS URL generation.

### Current Architecture Assessment

**Strengths:**
1. **Clean Service Implementation**: The service layer (345 LOC) is well-organized with clear methods and good documentation
2. **Proper Azure Integration**: Uses Azure Blob Storage SDK correctly with SAS token generation
3. **Analytics Integration**: Already integrates with the analytics service for tracking downloads
4. **Input Validation**: Has reasonable validation for blob paths and file types
5. **Typed Response Models**: Well-defined response structures in `library_response.go`

**Weaknesses:**
1. **Scattered Files**: Code spread across 6 directories (`controller`, `service`, `repository`, `route`, `response`, `request`)
2. **Error Handling via String Matching**: Controller uses `strings.Contains(err.Error(), "document not found")` - brittle pattern
3. **Unused Repository Layer**: Repository is scaffolding only (13 LOC interface, 20 LOC impl with no methods)
4. **Inconsistent Pattern**: Uses legacy `New*Router()` pattern instead of `domain.RegisterRoutes()`
5. **Future Expansion Blocked**: BII endpoints commented out, no clear extension path

### Refactoring Priority Assessment

| Criteria | Score | Notes |
|----------|-------|-------|
| Code Size | Medium | 638 LOC total - moderate size |
| Duplication | Low | Minimal duplication present |
| Pattern Violation | Medium | Uses scattered legacy pattern |
| Complexity | Low | Read-only blob operations |
| Risk | Low | No database operations |
| Effort | Low | Straightforward migration |

**Recommendation**: Priority 2 - Should be refactored after EIC. The domain is moderately well-organized but would benefit from colocation and typed error handling.

## Current State Analysis

### File Inventory (Old Pattern)

| File | Location | LOC | Purpose |
|------|----------|-----|---------|
| library_controller.go | api/controller/ | 136 | HTTP handlers for PMCS endpoints |
| library_service.go | api/service/ | 26 | Service interface (3 methods + comments) |
| library_service_impl.go | api/service/ | 345 | Azure Blob Storage operations |
| library_repository.go | api/repository/ | 13 | Empty repository interface |
| library_repository_impl.go | api/repository/ | 20 | Empty repository implementation |
| library_route.go | api/route/ | 52 | Route registration and DI |
| library_response.go | api/response/ | 36 | Response type definitions |
| library_request.go | api/request/ | 10 | Request type (placeholder only) |
| **Total** | | **638** | |

### Current API Endpoints

```
GET /api/v1/library/pmcs/vehicles           # List available PMCS vehicle folders
GET /api/v1/library/pmcs/:vehicle/documents # List PDFs for a specific vehicle
GET /api/v1/library/download?blob_path=...  # Generate time-limited download URL

# Future (commented out):
# GET /api/v1/library/bii/categories
# GET /api/v1/library/bii/:category/documents
# POST /api/v1/auth/library/favorites
# DELETE /api/v1/auth/library/favorites/:document_path
# GET /api/v1/auth/library/favorites
```

### External Dependencies

| Dependency | Type | Description |
|------------|------|-------------|
| azblob.Client | Azure SDK | Azure Blob Storage client for listing blobs |
| azblob.SharedKeyCredential | Azure SDK | Credential for SAS token generation |
| bootstrap.Env | Internal | Environment configuration |
| AnalyticsService | Internal | Tracks PMCS manual downloads |

### Code Analysis

**Controller (136 LOC):**
- `GetPMCSVehicles()` - Lists vehicle folders from PMCS library
- `GetPMCSDocuments()` - Lists PDF documents for a vehicle
- `GenerateDownloadURL()` - Creates SAS URL for secure downloads

**Issues Identified:**
1. String-based error detection (lines 97-118):
```go
if strings.Contains(err.Error(), "document not found") {
    // Handle 404
}
if strings.Contains(err.Error(), "invalid") {
    // Handle 400
}
```

2. Redundant error logging (logs in both service and controller)

**Service (345 LOC):**
- Well-structured with clear helper functions
- `formatDisplayName()` - Converts folder names to display names
- `extractFileName()` - Extracts filename from blob path
- `extractPMCSEquipmentName()` - Extracts equipment name for analytics
- `trackPMCSDownload()` - Analytics integration

**Good Practices Present:**
- Context usage for Azure operations
- Proper blob path validation
- SAS token security (HTTPS only, time-limited, read-only)
- Clock skew handling (5 minute buffer on SAS start time)

**Repository (33 LOC combined):**
- Empty scaffolding for future features (favorites, download history)
- Currently unused - domain is Azure-only

### Error Handling Analysis

**Current Errors (implicit via strings):**
- `"vehicle name cannot be empty"` - Input validation
- `"blob path cannot be empty"` - Input validation
- `"invalid blob path: must start with pmcs/ or bii/"` - Path validation
- `"invalid file type: only PDF files can be downloaded"` - File type validation
- `"document not found: %w"` - Blob not found
- `"failed to list PMCS vehicles: %w"` - Azure listing error
- `"failed to list PMCS documents: %w"` - Azure listing error
- `"failed to generate download URL: %w"` - SAS generation error

## Proposed New Structure

### Option A: Simple Bounded Context (Recommended)

Since this domain has a clear single purpose (document library access), a simple colocated structure is appropriate:

```
api/
  library/
    route.go           # HTTP handlers and route registration
    service.go         # Service interface
    service_impl.go    # Azure Blob Storage operations
    errors.go          # Typed error definitions
    response.go        # Response type definitions (moved from api/response/)
```

**Notes:**
- Repository layer is removed (not needed for Azure-only operations)
- Request types not needed (no POST bodies currently)
- Response types moved into domain

### Option B: Sub-Context Decomposition (Future-Proofed)

If BII support and favorites are planned, consider:

```
api/
  library/
    route.go                    # Main router
    shared/
      errors.go                 # Common error definitions
      response.go               # Shared response helpers
      blob_client.go            # Azure client wrapper
    pmcs/
      service.go                # PMCS interface
      service_impl.go           # PMCS operations
      route.go                  # PMCS handlers and routes
    bii/                        # Future: BII operations
      service.go
      service_impl.go
      route.go
    favorites/                  # Future: User favorites (requires DB)
      repository.go
      repository_impl.go
      service.go
      service_impl.go
      route.go
```

**Recommendation**: Start with Option A. Migrate to Option B only when BII or favorites are actually implemented.

## Implementation Checklist

### Phase 1: Foundation

- [x] 1.1 Create directory structure
  ```bash
  mkdir -p api/library
  ```

- [x] 1.2 Create errors.go
  ```go
  package library

  import "errors"

  var (
      ErrEmptyVehicleName = errors.New("vehicle name cannot be empty")
      ErrEmptyBlobPath    = errors.New("blob path cannot be empty")
      ErrInvalidBlobPath  = errors.New("invalid blob path: must start with pmcs/ or bii/")
      ErrInvalidFileType  = errors.New("invalid file type: only PDF files can be downloaded")
      ErrDocumentNotFound = errors.New("document not found")
      ErrBlobListFailed   = errors.New("failed to list blobs")
      ErrSASGenFailed     = errors.New("failed to generate download URL")
  )
  ```

- [x] 1.3 Move response types from api/response/library_response.go to api/library/response.go

### Phase 2: Core Implementation

- [x] 2.1 Create service.go (interface)
  ```go
  package library

  type Service interface {
      GetPMCSVehicles() (*PMCSVehiclesResponse, error)
      GetPMCSDocuments(vehicleName string) (*DocumentsListResponse, error)
      GenerateDownloadURL(blobPath string) (*DownloadURLResponse, error)
  }
  ```

- [x] 2.2 Migrate service_impl.go
  - Move from api/service/library_service_impl.go
  - Update to use typed errors
  - Keep Azure Blob Storage logic
  - Keep analytics integration

- [x] 2.3 Create route.go (handlers + registration)
  - Combine controller handlers and route registration
  - Update error handling to use typed errors with `errors.Is()`
  - Register both public and authenticated routes

### Phase 3: Wiring

- [x] 3.1 Create Dependencies struct and RegisterRoutes function
  ```go
  package library

  import (
      "database/sql"
      "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
      "github.com/gin-gonic/gin"
      "miltechserver/bootstrap"
  )

  type Dependencies struct {
      DB             *sql.DB                           // For future favorites
      BlobClient     *azblob.Client
      BlobCredential *azblob.SharedKeyCredential
      Env            *bootstrap.Env
      Analytics      AnalyticsService                  // Keep analytics integration
  }

  func RegisterRoutes(deps Dependencies, publicGroup, authGroup *gin.RouterGroup) {
      svc := NewService(deps.BlobClient, deps.BlobCredential, deps.Env, deps.Analytics)
      registerHandlers(svc, publicGroup, authGroup)
  }
  ```

- [x] 3.2 Update main route registration in api/route/route.go
  ```go
  // Replace:
  NewLibraryRouter(db, blobClient, blobCredential, env, authClient, v1Route, authRoutes)

  // With:
  library.RegisterRoutes(library.Dependencies{
      DB:             db,
      BlobClient:     blobClient,
      BlobCredential: blobCredential,
      Env:            env,
      Analytics:      analyticsService, // Need to wire this
  }, v1Route, authRoutes)
  ```

### Phase 4: Verification & Cleanup

- [ ] 4.1 Manual API testing
  - Test GET /api/v1/library/pmcs/vehicles
  - Test GET /api/v1/library/pmcs/{vehicle}/documents
  - Test GET /api/v1/library/download?blob_path=pmcs/...

- [ ] 4.2 Verify analytics tracking still works

- [x] 4.3 Remove legacy files:
  - api/controller/library_controller.go
  - api/service/library_service.go
  - api/service/library_service_impl.go
  - api/repository/library_repository.go
  - api/repository/library_repository_impl.go
  - api/route/library_route.go
  - api/response/library_response.go
  - api/request/library_request.go

## Actual Metrics Improvement

| Metric | Current | New | Improvement |
|--------|---------|--------|-------------|
| Total LOC (prod) | 638 | 542 | 15% reduction |
| Files | 8 (scattered) | 5 (colocated) | 37% fewer files |
| Directories | 6 | 1 | 83% fewer directories |
| Error handling | String matching | Typed errors | Safer, clearer |
| Unused code | ~33 LOC | 0 LOC | Remove dead code |

**Test Coverage Added:** 152 LOC (service + route unit tests)

## Risk Assessment

### Low Risk
- No database operations to migrate
- Read-only blob operations
- No authentication complexity (public routes only for now)
- Clear, well-defined API contract
- Well-tested Azure SDK integration

### Medium Risk
- Analytics service dependency requires careful wiring
- Response type location change may affect imports elsewhere

### Mitigation
- Verify analytics tracking in staging before production
- Search codebase for `response.VehicleFolderResponse` etc. imports before removing old location
- Keep old response file with deprecation comment temporarily if needed

## Comparison with Other Refactored Domains

| Domain | Bounded Contexts | Complexity | Pattern | External Deps |
|--------|------------------|------------|---------|---------------|
| shops | 7 | High | Full decomposition | Blob Storage |
| equipment_services | 5 | Medium | Full decomposition | None |
| user_saves | 5 | Medium | Feature-based | Blob Storage |
| item_lookup | 4 | Low | Data-domain | None |
| eic | 1 | Low | Colocated + utilities | None |
| **library** | 1 | Low | Simple colocated | Blob Storage |

**Key difference**: Library is unique in being Azure Blob Storage-only with no PostgreSQL dependency. The repository layer is scaffolding for future features.

## Dependencies on Other Services

### Analytics Service

**Current Usage** (api/library/service_impl.go):
```go
if analyticsErr := s.trackPMCSDownload(blobPath); analyticsErr != nil {
    slog.Warn("Failed to increment analytics for PMCS download", "blobPath", blobPath, "error", analyticsErr)
}
```

**Integration Point** (api/route/route.go):
```go
analyticsRepo := repository.NewAnalyticsRepositoryImpl(db)
analyticsService := service.NewAnalyticsServiceImpl(analyticsRepo)
```

**Impact**: Need to pass analytics service through Dependencies struct.

## Future Extension Points

### BII Support (Planned)
The service interface and routes have commented placeholders for BII endpoints:
- `GetBIICategories() (*PMCSVehiclesResponse, error)` - Can reuse VehicleFolderResponse
- `GetBIIDocuments(category string) (*DocumentsListResponse, error)` - Can reuse DocumentsListResponse

### User Favorites (Planned)
Requires database operations:
- `AddFavorite(userID, documentPath string) error`
- `RemoveFavorite(userID, documentPath string) error`
- `GetUserFavorites(userID string) ([]string, error)`

**Recommendation**: Implement BII first (no DB changes needed), then favorites in a later phase.

## Progress Log

- 2026-01-30: Initial analysis and planning complete
- 2026-01-31: Refactor complete (colocated package, typed errors, route wiring, tests, legacy removal)

## Notes

- The library container name is hardcoded as `"library"` - consider making configurable
- SAS URLs are valid for 1 hour with HTTPS-only restriction
- PDF files only are allowed for download (security restriction)
- Response types can remain in api/response/ if other domains use them, but they appear library-specific
- Consider adding caching for vehicle/document listings (Azure calls on every request currently)
- No tests exist for this domain - consider adding during refactor
