# Small Domains Refactoring Tracker

**Created:** 2026-01-31
**Owner:** swisscheese
**Status:** Complete
**Completed:** 2026-01-31

## Overview

This document covers the refactoring of three small domains that were initially marked as "Skip" but need to be updated to match the new bounded context architecture for codebase consistency.

| Domain | Current LOC | Files | Effort | Priority |
|--------|-------------|-------|--------|----------|
| item_quick_lists | ~245 | 9 | ~30 min | 1 |
| user_general | ~271 | 6 | ~45 min | 2 |
| analytics | ~131 | 4 | ~20 min | 3 |
| **Total** | **~647** | **19** | **~1.5 hr** | - |

## Rationale for Refactoring

While these domains are small and functional, they violate the new architecture pattern:

1. **Inconsistent codebase** - Developers see mixed patterns in `api/` directory
2. **Legacy scattered files** - Files spread across controller/, service/, repository/, route/, response/
3. **Missing typed errors** - String matching or generic 500 responses
4. **Old route registration** - `NewXxxRouter()` instead of `RegisterRoutes()`
5. **Known bugs** - user_general has a logging format bug

After refactoring, the legacy directories can be cleaned up once item_comments and item_query are also migrated.

---

## Domain 1: item_quick_lists

### Current State

**Files (9 total, ~245 LOC):**
```
api/controller/quick_lists_controller.go      # 59 LOC
api/service/item_quick_lists_service.go       # 10 LOC
api/service/item_quick_lists_service_impl.go  # 39 LOC
api/repository/item_quick_lists_repository.go # 10 LOC
api/repository/item_quick_lists_repository_impl.go # 74 LOC
api/route/item_quick_lists_route.go           # 25 LOC
api/response/quick_lists_battery_response.go  # 10 LOC
api/response/quick_lists_clothing_response.go # 10 LOC
api/response/quick_lists_wheels_response.go   # 10 LOC
```

**Current API Endpoints:**
```
GET /api/v1/quick-lists/clothing   # Public - clothing items
GET /api/v1/quick-lists/wheels     # Public - wheel/tire items
GET /api/v1/quick-lists/batteries  # Public - battery items
```

**Issues:**
- Scattered across 4 directories
- No typed errors (only generic 500 responses)
- Uses legacy `NewItemQuickListsRouter()` pattern
- 3 separate response files for simple structs

### Proposed Structure

```
api/quick_lists/
├── route.go           # HTTP handlers + RegisterRoutes()
├── service.go         # Service interface
├── service_impl.go    # Service implementation
├── repository.go      # Repository interface
├── repository_impl.go # Database queries
├── response.go        # All response types (consolidated)
└── errors.go          # Typed errors (optional - simple domain)
```

### Implementation Checklist

- [x] Create `api/quick_lists/` directory
- [x] Create `response.go` - consolidate 3 response files
- [x] Create `repository.go` - interface
- [x] Create `repository_impl.go` - move from legacy
- [x] Create `service.go` - interface
- [x] Create `service_impl.go` - move from legacy
- [x] Create `route.go` with:
  - [x] `Dependencies` struct
  - [x] `Handler` struct
  - [x] `RegisterRoutes()` function
  - [x] HTTP handlers (inline, no separate controller)
- [x] Update `api/route/route.go` to use new registration
- [x] Delete legacy files (9 files)
- [x] Test endpoints (route + service tests added)

### Expected Result

| Metric | Before | After | Change |
|--------|--------|-------|--------|
| Files | 9 | 6 | -33% |
| Directories | 4 | 1 | -75% |
| LOC | ~245 | ~180 | -27% |

---

## Domain 2: user_general

### Current State

**Files (6 total, ~271 LOC):**
```
api/controller/user_general_controller.go     # 95 LOC
api/service/user_general_service.go           # 10 LOC
api/service/user_general_service_impl.go      # 29 LOC
api/repository/user_general_repository.go     # 10 LOC
api/repository/user_general_repository_impl.go # 88 LOC
api/route/user_general_route.go               # 25 LOC
```

**Note:** Also uses `api/request/user_delete_request.go` and `api/request/user_display_name_change_request.go`

**Current API Endpoints:**
```
POST   /api/v1/user/general/refresh      # Auth - upsert user on login
DELETE /api/v1/user/general/delete_user  # Auth - delete user account
POST   /api/v1/user/general/dn_change    # Auth - update display name
```

**Issues:**
1. **Bug on line 34**: `slog.Info("Unauthorized request %s")` - format placeholder with no argument
2. **String-based error matching**: `err.Error() == "user not found"` instead of typed errors
3. Scattered across 4 directories
4. Uses legacy `NewUserGeneralRouter()` pattern

### Proposed Structure

```
api/user_general/
├── route.go           # HTTP handlers + RegisterRoutes()
├── service.go         # Service interface
├── service_impl.go    # Service implementation
├── repository.go      # Repository interface
├── repository_impl.go # Database queries
├── request.go         # Request types (consolidated)
└── errors.go          # Typed errors (ErrUserNotFound, etc.)
```

### Implementation Checklist

- [x] Create `api/user_general/` directory
- [x] Create `errors.go`:
  ```go
  var (
      ErrUserNotFound = errors.New("user not found")
  )
  ```
- [x] Create `request.go` - consolidate request types
- [x] Create `repository.go` - interface
- [x] Create `repository_impl.go` - use typed errors
- [x] Create `service.go` - interface
- [x] Create `service_impl.go` - move from legacy
- [x] Create `route.go` with:
  - [x] `Dependencies` struct
  - [x] `Handler` struct
  - [x] `RegisterRoutes()` function
  - [x] HTTP handlers with typed error handling
  - [x] **Fix logging bug** (remove format placeholder)
- [x] Update `api/route/route.go` to use new registration
- [x] Delete legacy files (6 files + 2 request files)
- [x] Test endpoints (route + service tests added)

### Expected Result

| Metric | Before | After | Change |
|--------|--------|-------|--------|
| Files | 8 | 7 | -12% |
| Directories | 5 | 1 | -80% |
| LOC | ~271 | ~220 | -19% |
| Bugs | 1 | 0 | Fixed |

---

## Domain 3: analytics

### Current State

**Files (4 total, ~131 LOC):**
```
api/service/analytics_service.go           # 10 LOC
api/service/analytics_service_impl.go      # 72 LOC
api/repository/analytics_repository.go     # 10 LOC
api/repository/analytics_repository_impl.go # 59 LOC
```

**Characteristics:**
- Internal service only (no HTTP routes)
- Used as dependency by: `library`, `item_query`
- Well-written code (proper error wrapping, constants)
- Provides: `IncrementItemSearchSuccess()`, `IncrementPMCSManualDownload()`

**Issues:**
- Scattered across 2 directories
- Inconsistent with colocated domain pattern

### Proposed Structure

```
api/analytics/
├── service.go         # Service interface (exported for other domains)
├── service_impl.go    # Service implementation
├── repository.go      # Repository interface
└── repository_impl.go # Database queries
```

### Implementation Checklist

- [x] Create `api/analytics/` directory
- [x] Create `repository.go` - interface
- [x] Create `repository_impl.go` - move from legacy
- [x] Create `service.go` - interface (AnalyticsService)
- [x] Create `service_impl.go` - move from legacy
- [x] Create `New()` constructor function for dependency injection
- [x] Update `api/library/` imports to use new location
- [x] Update `api/route/route.go` or bootstrap to wire analytics
- [x] Delete legacy files (4 files)
- [x] Test library and item_query still work (unit tests)

### Dependency Update Required

Library now imports analytics directly and expects `analytics.Service` in its dependencies.

### Expected Result

| Metric | Before | After | Change |
|--------|--------|-------|--------|
| Files | 4 | 4 | 0% |
| Directories | 2 | 1 | -50% |
| LOC | ~131 | ~131 | 0% |

---

## Combined Implementation Order

### Phase 1: analytics (Dependency - Do First)
Since analytics is used by other domains, refactor it first to establish the import path.

1. Create `api/analytics/` with all files
2. Update library to import from new location
3. Delete legacy analytics files
4. Verify library tests pass

### Phase 2: item_quick_lists (Simplest)
Simple read-only domain with no dependencies.

1. Create `api/quick_lists/` with all files
2. Update route registration
3. Delete legacy files
4. Test endpoints

### Phase 3: user_general (Has Bug Fix)
Authenticated endpoints with bug fix.

1. Create `api/user_general/` with all files
2. Implement typed errors
3. Fix logging bug
4. Update route registration
5. Delete legacy files
6. Test endpoints

---

## Route Registration Updates

### Before (api/route/route.go)
```go
NewItemQuickListsRouter(db, router)
NewUserGeneralRouter(db, authRoutes)
// analytics has no routes
```

### After (api/route/route.go)
```go
// Analytics (no routes, but needs to be created for DI)
analyticsSvc := analytics.New(db)

// Quick Lists
quick_lists.RegisterRoutes(quick_lists.Dependencies{DB: db}, publicGroup)

// User General
user_general.RegisterRoutes(user_general.Dependencies{DB: db}, authGroup)

// Update library to use analytics service
library.RegisterRoutes(library.Dependencies{
    // ...existing deps...
    Analytics: analyticsSvc,
}, publicGroup, authGroup)
```

---

## Post-Refactoring Cleanup

After all three domains are refactored, check if legacy directories can be cleaned up:

### Remaining Legacy Files (after this refactoring)
```
api/controller/
  - item_comments_controller.go  # Pending item_comments refactor
  - item_query_controller.go     # Pending item_query refactor

api/service/
  - item_comments_service*.go    # Pending
  - item_detailed_service*.go    # Pending (item_query)
  - item_short_service*.go       # Pending (item_query)

api/repository/
  - item_comments_repository*.go # Pending
  - item_detailed_repository*.go # Pending
  - item_query_repository*.go    # Pending

api/route/
  - item_comments_route.go       # Pending
  - item_query_route.go          # Pending
```

Once `item_comments` and `item_query` are refactored, the legacy directories can be removed entirely.

---

## Success Criteria

- [x] All 3 domains use `RegisterRoutes()` pattern
- [x] All 3 domains colocated in `api/<domain>/` directories
- [x] user_general logging bug fixed
- [x] user_general uses typed errors
- [x] analytics properly injected into library
- [ ] All existing tests pass
- [x] All endpoints return expected responses (route tests cover quick_lists + user_general)
- [x] Legacy files deleted (19 files total)

---

## Completion Metrics (Actual)

- item_quick_lists: 6 core files, 205 LOC (excluding tests), 2 test files added
- user_general: 7 core files, 270 LOC (excluding tests), 2 test files added
- analytics: 4 core files, 144 LOC (excluding tests)
- Legacy files deleted: 19 files total
- Tests: `go test ./api/...` passed; `go test ./...` failed due to existing issues in `tests/equipment_services`, `tests/material_images`, `tests/shops` (unrelated to this refactor)

---

## Summary

| Domain | Files Before | Files After | Key Changes |
|--------|--------------|-------------|-------------|
| item_quick_lists | 9 | 6 | Consolidate responses, colocate |
| user_general | 8 | 7 | Fix bug, typed errors, colocate |
| analytics | 4 | 4 | Colocate, update imports |
| **Total** | **21** | **17** | **-19% files, 100% consistency** |

**Total Estimated Effort:** ~1.5 hours

**Risk Level:** Low - small domains, no complex business logic
