# Architecture Refactoring Status

**Last Updated:** 2026-01-31
**Owner:** swisscheese

## Overview

This document tracks the progress of refactoring the miltechserver codebase from a monolithic controller/service/repository pattern to a bounded context architecture with colocated domain modules.

## Refactoring Status Summary

| Domain | Status | Complexity | LOC Before | LOC After | Reduction |
|--------|--------|------------|------------|-----------|-----------|
| shops | Complete | High | ~4000 | ~3200 | 20% |
| user_saves | Complete | Medium | ~800 | ~650 | 19% |
| user_vehicles | Complete | Low | ~400 | ~350 | 13% |
| material_images | Complete | Low | ~300 | ~280 | 7% |
| equipment_services | Complete | Medium | ~600 | ~500 | 17% |
| item_lookup | Complete | Low | ~715 | ~550 | 23% |
| eic | Complete | Low | 884 | 555 | 37% |
| **library** | **Complete** | Low | 638 | 542 | 15% |
| **item_comments** | **Complete** | Low-Medium | 729 | 730 | 0% |
| **item_query** | **Complete** | High | 826 | ~700 | ~15% |
| **item_quick_lists** | **Complete** | Low | 245 | 205 | 16% |
| **user_general** | **Complete** | Low | 271 | 270 | 0% |
| **analytics** | **Complete** | Low | 131 | 144 | -10% |

## Completed Refactorings

### 1. shops (January 2026)

**Pattern**: Full bounded context decomposition with shared authorization

**Structure**:
```
api/shops/
  core/           # Shop CRUD operations
  lists/          # Shopping lists
    items/        # List item management
  members/        # Member management
    invites/      # Invite system
  messages/       # Shop messaging
  settings/       # Shop settings
  vehicles/       # Vehicle associations
    notifications/
      items/
      changes/
  facade/         # Orchestration layer
  shared/         # Authorization, context, errors
  route.go        # Main router
```

**Key Improvements**:
- Clear bounded contexts for each feature area
- Shared authorization middleware
- Nested contexts for related features
- Facade pattern for complex operations

### 2. user_saves (January 2026)

**Pattern**: Feature-based decomposition

**Structure**:
```
api/user_saves/
  categories/     # Category management
  images/         # Image handling
  quick/          # Quick saves
  serialized/     # Serialized item saves
  facade/         # Orchestration layer
  route.go        # Main router
```

### 3. user_vehicles (January 2026)

**Pattern**: Simple bounded context

**Structure**:
```
api/user_vehicles/
  repository.go
  repository_impl.go
  service.go
  service_impl.go
  route.go
```

### 4. material_images (January 2026)

**Pattern**: Simple bounded context with mixed routes

**Structure**:
```
api/material_images/
  repository.go
  repository_impl.go
  service.go
  service_impl.go
  route.go
```

### 5. equipment_services (January 2026)

**Pattern**: Full bounded context decomposition

**Structure**:
```
api/equipment_services/
  core/           # Core CRUD
  queries/        # Read operations
  calendar/       # Calendar views
  status/         # Status management
  completion/     # Completion tracking
  route.go
```

### 6. item_lookup (January 2026)

**Pattern**: Data-domain decomposition with shared utilities

**Structure**:
```
api/item_lookup/
  lin/            # LIN lookups
  uoc/            # UOC lookups
  cage/           # CAGE lookups
  substitute/     # Substitute LIN lookups
  shared/         # Errors, pagination, response helpers
  route.go
```

**Key Improvements**:
- Separate bounded contexts for each data domain
- Shared pagination utilities
- Typed error handling

### 7. eic (January 2026)

**Pattern**: Colocated single context with extracted utilities

**Structure**:
```
api/eic/
  route.go           # HTTP handlers and route registration
  service.go         # Service interface
  service_impl.go    # Business logic
  repository.go      # Repository interface
  repository_impl.go # Database queries (deduplicated)
  query_builder.go   # Extracted SQL patterns
  scanner.go         # Extracted row scanning
  errors.go          # Typed errors
```

**Key Improvements**:
- 37% LOC reduction (884 -> 555)
- Query builder pattern eliminates 4x SQL duplication
- Typed error handling
- Colocated domain files (7 files -> 8 files, but better organized)
- Consistent RegisterRoutes() pattern

**Tracker**: [docs/EIC_REFACTORING_TRACKER.md](./EIC_REFACTORING_TRACKER.md)

## Additional Completed Refactorings

### 8. library (Complete - January 2026)

**Pattern**: Simple colocated bounded context

**Current Location**: `api/library/route.go`, `api/library/service.go`, `api/library/service_impl.go`, `api/library/errors.go`, `api/library/response.go`

**Complexity**: Low (638 LOC total - Azure Blob Storage only, no PostgreSQL)

**Structure**:
```
api/library/
  route.go           # HTTP handlers and route registration
  service.go         # Service interface
  service_impl.go    # Azure Blob Storage operations
  errors.go          # Typed error definitions
  response.go        # Response type definitions
```

**Key Improvements**:
- 15% LOC reduction (638 -> 542)
- Remove unused repository scaffolding (33 LOC dead code)
- Typed error handling instead of string matching
- Colocated domain files (8 files -> 5)
- Consistent RegisterRoutes() pattern

**Unique Characteristics**:
- Azure Blob Storage-only domain (no PostgreSQL)
- Generates SAS tokens for secure downloads
- Integrates with analytics service
- Currently public routes only (future: favorites require auth)

**Tracker**: [docs/LIBRARY_REFACTORING_TRACKER.md](./LIBRARY_REFACTORING_TRACKER.md)

### 9. item_comments (Complete)

**Pattern**: Simple colocated bounded context

**Current Location**: `api/item_comments/*`

**Complexity**: Low-Medium (729 LOC total)

**Proposed Structure**:
```
api/item_comments/
  route.go           # HTTP handlers and route registration
  service.go         # Service interface
  service_impl.go    # Business logic with validation
  repository.go      # Repository interface
  repository_impl.go # Database queries
  errors.go          # Typed error definitions
  types.go           # Request/response types
```

**Key Improvements**:
- Colocated domain files (8 files -> 7)
- Consistent RegisterRoutes() pattern
- Errors already typed - best practice model
- Unit + integration tests added for item_comments routes/service

**Unique Characteristics**:
- Mixed routes (public read, authenticated write)
- **Already uses typed errors correctly** - model pattern for other domains
- Threaded comment support (parent_id)
- Soft delete implementation
- Comment flagging for moderation

**Tracker**: [docs/ITEM_COMMENTS_REFACTORING_TRACKER.md](./ITEM_COMMENTS_REFACTORING_TRACKER.md)

## Additional Completed Refactorings (continued)

### 10. item_query (Complete)

**Current Location**: `api/item_query/*`

**Complexity**: High (~826 LOC total - 517 LOC in item_detailed_repository_impl alone)

**Structure**:
```
api/item_query/
  shared/           # typed errors + analytics adapter
  short/            # Short item queries (NIIN/part lookup)
    route.go
    service.go
    service_impl.go
    repository.go
    repository_impl.go
  detailed/         # Detailed item queries (multi-table aggregation)
    route.go
    service.go
    service_impl.go
    repository.go
    repository_impl.go
    queries/         # Extracted query helpers
  route.go          # Main router
```

**Key Improvements**:
- Detailed query failures logged server-side (no longer silently ignored)
- Short query handlers use typed errors instead of string matching
- Endpoints and response shapes preserved
- Query helpers extracted for readability

**Tracker**: [docs/ITEM_QUERY_REFACTORING_TRACKER.md](./ITEM_QUERY_REFACTORING_TRACKER.md)
**Progress**: [docs/ITEM_QUERY_REFACTORING_PROGRESS.md](./ITEM_QUERY_REFACTORING_PROGRESS.md)

### 11-13. Small Domains (Complete - January 2026)

These three small domains were refactored together for consistency and legacy cleanup.

**Tracker**: [docs/SMALL_DOMAINS_REFACTORING_TRACKER.md](./SMALL_DOMAINS_REFACTORING_TRACKER.md)

#### 11. analytics

**Current Location**: `api/analytics/*`

**Complexity**: Low (144 LOC, 4 files)

**Structure**:
```
api/analytics/
├── service.go         # Interface (exported for other domains)
├── service_impl.go    # Implementation
├── repository.go      # Interface
└── repository_impl.go # Database queries
```

**Notes**:
- Internal service, no HTTP routes
- Used by library service and item_query service
- Legacy analytics files removed

#### 12. item_quick_lists

**Current Location**: `api/quick_lists/*`

**Complexity**: Low (205 LOC, 6 core files)

**Structure**:
```
api/quick_lists/
├── route.go           # Handlers + RegisterRoutes()
├── service.go         # Interface
├── service_impl.go    # Implementation
├── repository.go      # Interface
├── repository_impl.go # Database queries
└── response.go        # Consolidated response types
```

**Notes**:
- Legacy quick_lists files removed
- Added route + service tests

#### 13. user_general

**Current Location**: `api/user_general/*`

**Complexity**: Low (270 LOC, 7 core files)

**Structure**:
```
api/user_general/
├── route.go           # Handlers + RegisterRoutes()
├── service.go         # Interface
├── service_impl.go    # Implementation
├── repository.go      # Interface
├── repository_impl.go # Database queries
├── request.go         # Request types
└── errors.go          # Typed errors
```

**Notes**:
- Logging format bug fixed
- Typed errors used for not-found handling
- Legacy user_general files removed

## Prioritization Matrix

| Domain | Size | Duplication | Pattern Violation | Technical Debt | Priority |
|--------|------|-------------|-------------------|----------------|----------|
| ~~eic~~ | ~~High~~ | ~~High~~ | ~~Medium~~ | ~~Medium~~ | ✅ Complete |
| ~~library~~ | ~~Medium~~ | ~~Low~~ | ~~Medium~~ | ~~Low~~ | ✅ Complete |
| ~~**Small Domains**~~ | Low | Low | **High** | Low | ✅ Complete |
| **item_comments** | Medium | Low | Medium | Very Low | **2 - Consider** |
| ~~item_query~~ | ~~High~~ | ~~Medium~~ | ~~High~~ | ~~High~~ | ✅ Complete |

### Priority Rationale

**Priority 1 (Small Domains - Combined)**:
- ✅ Complete (analytics, item_quick_lists, user_general)
- Legacy scattered patterns removed
- user_general logging bug fixed
- analytics now colocated and wired via `analytics.New()`
- **Tracker**: [docs/SMALL_DOMAINS_REFACTORING_TRACKER.md](./SMALL_DOMAINS_REFACTORING_TRACKER.md)

**Priority 2 (item_comments)**:
- Already well-organized with typed errors
- Refactoring is for consistency, not code quality
- Can serve as pattern reference for error handling

**Priority 3 (item_query)**:
- ✅ Complete (see item_query section above)

## Architecture Patterns Reference

### When to Use Full Bounded Context Decomposition
- Multiple distinct feature areas (shops, equipment_services)
- Different authentication/authorization requirements
- High complexity with many operations
- Features that could evolve independently

### When to Use Data-Domain Decomposition
- Multiple distinct data sources/tables (item_lookup)
- Different query patterns for different data
- No shared business logic between data types

### When to Use Simple Colocation
- Single data source (eic, item_comments)
- Related operations on same model
- Focus on code organization and duplication elimination

### When to Skip Refactoring
- Small domains (<300 LOC total)
- Already well-organized
- No significant duplication
- Internal services without HTTP routes
- Simple read-only queries with no business logic

## Technical Debt Tracking

### High Priority (Address in Refactoring)
- [x] ~~EIC repository duplication (484 LOC with 4x repeated SQL)~~ - **COMPLETE**
- [x] ~~String-based error matching in controllers (item_query, user_general)~~ - **COMPLETE**
- [x] ~~Library unused repository layer (33 LOC dead code)~~ - **COMPLETE**
- [x] ~~item_query ignores errors from detail queries (dangerous)~~ - **COMPLETE**

### Medium Priority (Consider During Refactoring)
- [ ] Inconsistent pagination implementations
- [ ] Response type locations (some in domain, some in api/response)
- [x] user_general logging format string bug

### Low Priority (Future Consideration)
- [ ] OpenAPI/Swagger documentation
- [ ] Standardize naming conventions across domains
- [ ] Add integration test coverage
- [ ] user_login_request.go dead code (4 LOC)

## Main Route Registration Status

**File**: `api/route/route.go`

| Domain | Registration Pattern | Status |
|--------|---------------------|--------|
| item_lookup | `domain.RegisterRoutes(deps, router)` | New Pattern |
| user_saves | `domain.RegisterRoutes(deps, router)` | New Pattern |
| user_vehicles | `domain.RegisterRoutes(deps, router)` | New Pattern |
| material_images | `domain.RegisterRoutes(deps, router)` | New Pattern |
| equipment_services | `domain.RegisterRoutes(deps, router)` | New Pattern |
| eic | `eic.RegisterRoutes(eic.Dependencies{DB: db}, router)` | ✅ New Pattern |
| shops | `NewShopsRouter(db, blob, env, router)` | Legacy (uses new internal structure) |
| library | `library.RegisterRoutes(deps, public, auth)` | ✅ New Pattern |
| item_comments | `item_comments.RegisterRoutes(item_comments.Dependencies{DB: db}, v1Route, authRoutes)` | New Pattern |
| item_query | `item_query.RegisterRoutes(item_query.Dependencies{DB: db}, v1Route)` | ✅ New Pattern |
| item_quick_lists | `quick_lists.RegisterRoutes(quick_lists.Dependencies{DB: db}, v1Route)` | ✅ New Pattern |
| user_general | `user_general.RegisterRoutes(user_general.Dependencies{DB: db}, authRoutes)` | ✅ New Pattern |

### Route Registration Pattern

**New Pattern (Recommended)**:
```go
domain.RegisterRoutes(domain.Dependencies{
    DB:         db,
    BlobClient: blobClient,  // if needed
    Env:        env,         // if needed
}, publicGroup, authGroup)   // one or both depending on domain
```

**Dependencies struct per domain**:
- `DB *sql.DB` - Always required for database domains
- `BlobClient *azblob.Client` - For Azure Blob Storage domains
- `BlobCredential *azblob.SharedKeyCredential` - For SAS token generation
- `Env *bootstrap.Env` - For environment configuration
- `AuthClient *auth.Client` - For Firebase auth middleware

## Domain Analysis Summary

### Well-Organized Domains (Low Refactoring Value)
- **item_comments**: Already uses typed errors, good separation
- **item_quick_lists**: Refactored, colocated, with tests
- **user_general**: Refactored, typed errors, logging bug fixed

### Needs Refactoring (High Value)
- ~~**eic**: Massive duplication, large files~~ ✅ **COMPLETE**
- ~~**library**: Dead code, scattered files~~ ✅ **COMPLETE**
- ~~**item_query**: Error handling issues, complex structure~~ ✅ **COMPLETE**

### Best Practice Examples
- **item_comments**: Typed error handling pattern
- **item_lookup**: Data-domain decomposition pattern
- **shops**: Complex domain decomposition pattern
