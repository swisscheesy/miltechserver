# User Saves Domain Refactoring Plan

## Refactor Progress Tracker

Last updated: 2026-01-30

| Step | Status | Notes |
|------|--------|-------|
| 1. Create directory structure and shared package | ✅ Complete | Added `api/user_saves/shared` and `api/user_saves/route.go` delegating to legacy |
| 2. Extract Images bounded context | ✅ Complete | Added `api/user_saves/images` and wired image routes to new package |
| 3. Extract Quick Items bounded context | ✅ Complete | Added `api/user_saves/quick` and wired quick routes to new package |
| 4. Extract Serialized Items bounded context | ✅ Complete | Added `api/user_saves/serialized` and wired serialized routes to new package |
| 5. Extract Categories + Items bounded context | ✅ Complete | Added `api/user_saves/categories` and `api/user_saves/categories/items` with route wiring |
| 6. Create Facade service | ✅ Complete | Added `api/user_saves/facade` delegating to sub-services |
| 7. Wire dependencies and update routing | ✅ Complete | `api/route/route.go` now uses `user_saves.RegisterRoutes` |
| 8. Testing and verification | ✅ Complete | Added service-level tests for quick/serialized/categories/items |
| 9. Cleanup legacy files | ✅ Complete | Removed legacy user_saves controller/service/repository/route files |

## Executive Summary

The `user_saves` domain has grown into a **monolithic structure** spanning ~2,000 lines across six files:

| File | Lines | Responsibility |
|------|-------|----------------|
| `api/repository/user_saves_repository_impl.go` | 1,044 | Data access for all user saves entities |
| `api/controller/user_saves_controller.go` | 636 | HTTP handlers for 19 endpoints |
| `api/service/user_saves_service_impl.go` | 151 | Pass-through business logic |
| `api/repository/user_saves_repository.go` | 41 | Repository interface |
| `api/service/user_saves_service.go` | 41 | Service interface |
| `api/route/user_saves_route.go` | 51 | Route registration |

This plan outlines how to decompose these into **focused bounded contexts** following the same pattern successfully applied in the shops domain refactor.

---

## Current Domain Analysis

### Identified Bounded Contexts

Based on the interface analysis from `api/service/user_saves_service.go`, the user saves domain contains **4 distinct sub-domains** plus a cross-cutting concern:

```
user_saves/
├── quick/           # Quick save items (NIIN-based)
├── serialized/      # Serialized items (with serial numbers)
├── categories/      # User-defined categories
│   └── items/       # Items within categories
└── images/          # Cross-cutting image management
```

### Current Interface Method Count by Context

| Context | Service Methods | Repository Methods | Endpoints |
|---------|-----------------|-------------------|-----------|
| Quick Items | 5 | 5 | 5 |
| Serialized Items | 5 | 5 | 5 |
| Categories | 4 | 4 | 4 |
| Categorized Items | 6 | 6 | 6 |
| Image Management | 3 | 3 | 3 |
| **TOTAL** | **23** | **23** | **23** |

### Database Tables

| Context | Table Name |
|---------|------------|
| Quick Items | `user_items_quick` |
| Serialized Items | `user_items_serialized` |
| Categories | `user_item_category` |
| Categorized Items | `user_items_categorized` |

---

## Architectural Problems Identified

### 1. Single Responsibility Principle Violation

The `UserSavesRepositoryImpl` struct handles:
- Quick save item CRUD operations
- Serialized item CRUD operations
- Category management
- Categorized item management
- Image upload/download/delete across all 4 item types
- Bulk image deletion across multiple tables

**Evidence from code:**
```go
// Single struct with 3 dependencies handling 4 bounded contexts
type UserSavesRepositoryImpl struct {
    Db         *sql.DB
    BlobClient *azblob.Client
    Env        *bootstrap.Env
}
```

### 2. Code Duplication

The repository contains significant duplication patterns:

1. **Null user checks repeated** 5+ times:
```go
if user != nil {
    // ... operation
} else {
    return nil, errors.New("valid user not found")
}
```

2. **Bulk upsert logic duplicated** 3 times for different item types
3. **Image-with-items retrieval** duplicated 4 times

### 3. Mixed Abstraction Levels

The repository mixes:
- Low-level database operations
- Business logic (image cleanup coordination)
- External service calls (Azure Blob Storage)

**Example** from `DeleteUserItemCategory`:
```go
func (repo *UserSavesRepositoryImpl) DeleteUserItemCategory(...) error {
    // 1. Query for items with images
    // 2. Delete images from blob storage
    // 3. Delete category image
    // 4. Delete category from database
    // 5. Delete categorized items from database
}
```

This violates separation of concerns - the repository is orchestrating a multi-step business operation.

### 4. Missing Transaction Support

The `DeleteUserItemCategory` method performs multiple database operations without transactions - a data corruption risk:
- Deletes categorized items
- Deletes category
- No rollback on partial failure

### 5. Lack of Domain-Specific Error Types

Current error handling uses generic error strings:
```go
return errors.New("error saving quick item " + err.Error())
return errors.New("user saves not found")
return errors.New("valid user not found")
```

### 6. Controller Boilerplate Repetition

The `UserSavesController` repeats the same authentication pattern 19+ times:
```go
ctxUser, ok := c.Get("user")
user, _ := ctxUser.(*bootstrap.User)
if !ok {
    c.JSON(401, gin.H{"message": "unauthorized"})
    slog.Info("Unauthorized request")
    return
}
```

### 7. Thin Service Layer

The service layer is essentially a pass-through with no business logic:
```go
func (service *UserSavesServiceImpl) UpsertQuickSaveItemByUser(user *bootstrap.User, quick model.UserItemsQuick) error {
    return service.UserSavesRepository.UpsertQuickSaveItemByUser(user, quick)
}
```

---

## Target Architecture

### Phase 1: Package Structure

```
api/
├── user_saves/
│   ├── route.go                    # Central wiring & dependency injection
│   │
│   ├── facade/
│   │   ├── service.go              # Unified interface (backward compat)
│   │   └── service_impl.go         # Delegates to sub-services
│   │
│   ├── shared/
│   │   ├── context.go              # User extraction helper
│   │   └── errors.go               # Sentinel errors
│   │
│   ├── quick/
│   │   ├── repository.go           # Interface (5 methods)
│   │   ├── repository_impl.go      # Data access (~150 lines)
│   │   ├── service.go              # Interface
│   │   ├── service_impl.go         # Business logic (~80 lines)
│   │   └── route.go                # 5 endpoints
│   │
│   ├── serialized/
│   │   ├── repository.go           # Interface (5 methods)
│   │   ├── repository_impl.go      # Data access (~150 lines)
│   │   ├── service.go              # Interface
│   │   ├── service_impl.go         # Business logic (~80 lines)
│   │   └── route.go                # 5 endpoints
│   │
│   ├── categories/
│   │   ├── repository.go           # Interface (4 methods)
│   │   ├── repository_impl.go      # Data access (~120 lines)
│   │   ├── service.go              # Interface
│   │   ├── service_impl.go         # Business logic (~100 lines)
│   │   ├── route.go                # 4 endpoints
│   │   └── items/                  # Categorized items sub-context
│   │       ├── repository.go       # Interface (6 methods)
│   │       ├── repository_impl.go  # Data access (~180 lines)
│   │       ├── service.go          # Interface
│   │       ├── service_impl.go     # Business logic (~100 lines)
│   │       └── route.go            # 6 endpoints
│   │
│   └── images/                     # Cross-cutting image management
│       ├── repository.go           # Interface (3 methods)
│       ├── repository_impl.go      # Blob operations (~150 lines)
│       ├── service.go              # Interface
│       ├── service_impl.go         # Business logic (~80 lines)
│       └── route.go                # 3 endpoints
```

### Phase 2: Shared Utilities

#### User Context Helper

```go
// api/user_saves/shared/context.go
package shared

import (
    "miltechserver/bootstrap"
    "github.com/gin-gonic/gin"
)

// GetUserFromContext extracts the authenticated user from the gin context.
// Returns nil and false if not found or invalid.
func GetUserFromContext(c *gin.Context) (*bootstrap.User, bool) {
    ctxUser, ok := c.Get("user")
    if !ok {
        return nil, false
    }
    user, ok := ctxUser.(*bootstrap.User)
    return user, ok
}

// RequireUser is a convenience wrapper that returns an error response
// if the user is not found in context.
func RequireUser(c *gin.Context) (*bootstrap.User, bool) {
    user, ok := GetUserFromContext(c)
    if !ok {
        c.JSON(401, gin.H{"message": "unauthorized"})
        return nil, false
    }
    return user, true
}
```

#### Sentinel Errors

```go
// api/user_saves/shared/errors.go
package shared

import "errors"

// User errors
var (
    ErrUserNotFound = errors.New("valid user not found")
)

// Item errors
var (
    ErrItemNotFound      = errors.New("item not found")
    ErrItemAlreadyExists = errors.New("item already exists")
    ErrInvalidItemID     = errors.New("invalid item ID")
)

// Category errors
var (
    ErrCategoryNotFound      = errors.New("category not found")
    ErrCategoryAlreadyExists = errors.New("category already exists")
    ErrCategoryNotEmpty      = errors.New("category is not empty")
)

// Image errors
var (
    ErrImageNotFound       = errors.New("image not found")
    ErrImageUploadFailed   = errors.New("failed to upload image")
    ErrImageDeleteFailed   = errors.New("failed to delete image")
    ErrInvalidTableType    = errors.New("invalid table type")
    ErrImageTooLarge       = errors.New("image exceeds maximum size")
)

// Bulk operation errors
var (
    ErrBulkOperationPartialFailure = errors.New("bulk operation partially failed")
)
```

---

## Migration Strategy

### Approach: Strangler Fig Pattern

The **Strangler Fig Pattern** allows incremental migration without breaking existing clients:

1. **Wrap**: New code calls old implementation initially
2. **Replace**: Gradually move logic to new packages
3. **Remove**: Delete old code when fully migrated

### Step-by-Step Implementation

#### Step 1: Create Directory Structure and Shared Package

**Goal**: Establish infrastructure without changing behavior

**Tasks**:
1. Create `api/user_saves/` directory structure
2. Create `shared/context.go` with user extraction helper
3. Create `shared/errors.go` with sentinel errors
4. Create `route.go` skeleton that delegates to old implementation

**Estimated Files Changed**: 4 new files

---

#### Step 2: Extract Images Bounded Context (Low Risk)

**Rationale**: Images is a cross-cutting concern used by all other contexts. Extract first to enable cleaner dependencies.

**Current Location**:
- Repository: `user_saves_repository.go:37-40` (3 methods)
- Service: `user_saves_service.go:37-40` (3 methods)
- Controller: `user_saves_controller.go` (3 handlers)
- Routes: `user_saves_route.go:47-49` (3 endpoints)

**New Files**:
```
api/user_saves/images/
├── repository.go        # ~20 lines
├── repository_impl.go   # ~150 lines
├── service.go           # ~20 lines
├── service_impl.go      # ~80 lines
└── route.go             # ~30 lines
```

**Interface**:
```go
// api/user_saves/images/repository.go
package images

import "miltechserver/bootstrap"

type Repository interface {
    Upload(user *bootstrap.User, itemID string, tableType string, imageData []byte) (string, error)
    Delete(user *bootstrap.User, itemID string, tableType string) error
    Get(user *bootstrap.User, itemID string, tableType string) ([]byte, string, error)
}
```

**Migration Steps**:
1. Create new package with interfaces
2. Copy implementation from `user_saves_repository_impl.go`
3. Wire up in new route registration
4. Update old implementation to delegate to new
5. Add tests for new package

---

#### Step 3: Extract Quick Items Bounded Context (Low Risk)

**Current Location**:
- Repository: `user_saves_repository.go:9-14` (5 methods)
- Service: `user_saves_service.go:9-14` (5 methods)
- Controller: `user_saves_controller.go` (5 handlers)
- Routes: `user_saves_route.go:22-26` (5 endpoints)

**New Files**:
```
api/user_saves/quick/
├── repository.go        # ~25 lines
├── repository_impl.go   # ~150 lines
├── service.go           # ~25 lines
├── service_impl.go      # ~80 lines
└── route.go             # ~35 lines
```

**Interface**:
```go
// api/user_saves/quick/repository.go
package quick

import (
    "miltechserver/.gen/miltech_ng/public/model"
    "miltechserver/bootstrap"
)

type Repository interface {
    GetByUser(user *bootstrap.User) ([]model.UserItemsQuick, error)
    Upsert(user *bootstrap.User, item model.UserItemsQuick) error
    UpsertBatch(user *bootstrap.User, items []model.UserItemsQuick) error
    Delete(user *bootstrap.User, item model.UserItemsQuick) error
    DeleteAll(user *bootstrap.User) error
}
```

**Migration Steps**:
1. Create new package with interfaces
2. Copy implementation from `user_saves_repository_impl.go`
3. Inject images repository for image cleanup
4. Wire up in new route registration
5. Add tests for new package

---

#### Step 4: Extract Serialized Items Bounded Context (Low Risk)

**Current Location**:
- Repository: `user_saves_repository.go:16-21` (5 methods)
- Service: `user_saves_service.go:16-21` (5 methods)
- Controller: `user_saves_controller.go` (5 handlers)
- Routes: `user_saves_route.go:28-32` (5 endpoints)

**New Files**:
```
api/user_saves/serialized/
├── repository.go        # ~25 lines
├── repository_impl.go   # ~150 lines
├── service.go           # ~25 lines
├── service_impl.go      # ~80 lines
└── route.go             # ~35 lines
```

**Interface**:
```go
// api/user_saves/serialized/repository.go
package serialized

import (
    "miltechserver/.gen/miltech_ng/public/model"
    "miltechserver/bootstrap"
)

type Repository interface {
    GetByUser(user *bootstrap.User) ([]model.UserItemsSerialized, error)
    Upsert(user *bootstrap.User, item model.UserItemsSerialized) error
    UpsertBatch(user *bootstrap.User, items []model.UserItemsSerialized) error
    Delete(user *bootstrap.User, item model.UserItemsSerialized) error
    DeleteAll(user *bootstrap.User) error
}
```

---

#### Step 5: Extract Categories Bounded Context (Medium Risk)

**Special Considerations**:
- Has nested sub-context (categorized items)
- Cascading delete requires transaction support
- Image cleanup on delete

**Current Location**:
- Repository: `user_saves_repository.go:23-27` (4 methods)
- Service: `user_saves_service.go:23-27` (4 methods)
- Controller: `user_saves_controller.go` (4 handlers)
- Routes: `user_saves_route.go:34-37` (4 endpoints)

**New Files**:
```
api/user_saves/categories/
├── repository.go        # ~20 lines
├── repository_impl.go   # ~120 lines
├── service.go           # ~20 lines
├── service_impl.go      # ~100 lines
├── route.go             # ~30 lines
└── items/
    ├── repository.go    # ~25 lines
    ├── repository_impl.go # ~180 lines
    ├── service.go       # ~25 lines
    ├── service_impl.go  # ~100 lines
    └── route.go         # ~40 lines
```

**Transaction Support**:
```go
// api/user_saves/categories/repository_impl.go
func (r *RepositoryImpl) Delete(user *bootstrap.User, category model.UserItemCategory) error {
    tx, err := r.Db.Begin()
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback()

    // 1. Get and delete images for categorized items
    // 2. Delete categorized items
    // 3. Delete category image
    // 4. Delete category

    return tx.Commit()
}
```

---

#### Step 6: Create Facade Service (Backward Compatibility)

**Purpose**: Provide a unified interface that maintains backward compatibility with existing consumers.

**New Files**:
```
api/user_saves/facade/
├── service.go           # ~50 lines
└── service_impl.go      # ~150 lines
```

**Implementation**:
```go
// api/user_saves/facade/service.go
package facade

import (
    "miltechserver/.gen/miltech_ng/public/model"
    "miltechserver/bootstrap"
)

// Service provides a unified interface to all user saves sub-domains.
// This maintains backward compatibility with existing code that expects
// a single service interface.
type Service interface {
    // Quick Items
    GetQuickSaveItemsByUser(user *bootstrap.User) ([]model.UserItemsQuick, error)
    UpsertQuickSaveItemByUser(user *bootstrap.User, quick model.UserItemsQuick) error
    UpsertQuickSaveItemListByUser(user *bootstrap.User, quickItems []model.UserItemsQuick) error
    DeleteQuickSaveItemByUser(user *bootstrap.User, quick model.UserItemsQuick) error
    DeleteAllQuickSaveItemsByUser(user *bootstrap.User) error

    // Serialized Items
    GetSerializedItemsByUser(user *bootstrap.User) ([]model.UserItemsSerialized, error)
    UpsertSerializedSaveItemByUser(user *bootstrap.User, serializedItem model.UserItemsSerialized) error
    UpsertSerializedSaveItemListByUser(user *bootstrap.User, serializedItems []model.UserItemsSerialized) error
    DeleteSerializedSaveItemByUser(user *bootstrap.User, serializedItem model.UserItemsSerialized) error
    DeleteAllSerializedItemsByUser(user *bootstrap.User) error

    // Categories
    GetItemCategoriesByUser(user *bootstrap.User) ([]model.UserItemCategory, error)
    UpsertItemCategoryByUser(user *bootstrap.User, itemCategory model.UserItemCategory) error
    DeleteItemCategory(user *bootstrap.User, itemCategory model.UserItemCategory) error
    DeleteAllItemCategories(user *bootstrap.User) error

    // Categorized Items
    GetCategorizedItemsByUser(user *bootstrap.User) ([]model.UserItemsCategorized, error)
    GetCategorizedItemsByCategory(user *bootstrap.User, itemCategory model.UserItemCategory) ([]model.UserItemsCategorized, error)
    UpsertCategorizedItemByUser(user *bootstrap.User, categorizedItem model.UserItemsCategorized) error
    UpsertCategorizedItemListByUser(user *bootstrap.User, categorizedItems []model.UserItemsCategorized) error
    DeleteCategorizedItemByCategoryId(user *bootstrap.User, categorizedItem model.UserItemsCategorized) error
    DeleteAllCategorizedItems(user *bootstrap.User) error

    // Images
    UploadItemImage(user *bootstrap.User, itemID string, tableType string, imageData []byte) (string, error)
    DeleteItemImage(user *bootstrap.User, itemID string, tableType string) error
    GetItemImage(user *bootstrap.User, itemID string, tableType string) ([]byte, string, error)
}
```

---

#### Step 7: Wire Dependencies in Central Route

```go
// api/user_saves/route.go
package user_saves

import (
    "database/sql"
    "miltechserver/api/user_saves/categories"
    "miltechserver/api/user_saves/facade"
    "miltechserver/api/user_saves/images"
    "miltechserver/api/user_saves/quick"
    "miltechserver/api/user_saves/serialized"
    "miltechserver/bootstrap"

    "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
    "github.com/gin-gonic/gin"
)

// Dependencies holds shared dependencies for all user saves sub-domains
type Dependencies struct {
    DB         *sql.DB
    BlobClient *azblob.Client
    Env        *bootstrap.Env
}

// RegisterRoutes sets up all user saves related routes
func RegisterRoutes(deps Dependencies, group *gin.RouterGroup) {
    // Create repositories
    imagesRepo := images.NewRepository(deps.DB, deps.BlobClient, deps.Env)
    quickRepo := quick.NewRepository(deps.DB)
    serializedRepo := serialized.NewRepository(deps.DB)
    categoriesRepo := categories.NewRepository(deps.DB)
    categorizedItemsRepo := categories.NewItemsRepository(deps.DB)

    // Create services with image repository injected
    imagesService := images.NewService(imagesRepo)
    quickService := quick.NewService(quickRepo, imagesRepo)
    serializedService := serialized.NewService(serializedRepo, imagesRepo)
    categoriesService := categories.NewService(categoriesRepo, categorizedItemsRepo, imagesRepo)
    categorizedItemsService := categories.NewItemsService(categorizedItemsRepo, imagesRepo)

    // Create facade for backward compatibility
    facadeService := facade.NewService(
        quickService,
        serializedService,
        categoriesService,
        categorizedItemsService,
        imagesService,
    )

    // Register routes for each sub-domain
    userSavesGroup := group.Group("/user/saves")
    {
        quick.RegisterRoutes(userSavesGroup, facadeService)
        serialized.RegisterRoutes(userSavesGroup, facadeService)
        categories.RegisterRoutes(userSavesGroup, facadeService)
        images.RegisterRoutes(userSavesGroup, facadeService)
    }
}
```

---

#### Step 8: Cleanup Legacy Files

**Files to Remove**:
- `api/controller/user_saves_controller.go`
- `api/service/user_saves_service.go`
- `api/service/user_saves_service_impl.go`
- `api/repository/user_saves_repository.go`
- `api/repository/user_saves_repository_impl.go`
- `api/route/user_saves_route.go`

**Update Imports**: Search and replace all imports pointing to old packages

---

## API Backward Compatibility

### URL Structure Preservation

All existing URLs **MUST** remain unchanged:

| Current URL | New Handler Location |
|-------------|---------------------|
| `GET /api/v1/auth/user/saves/quick_items` | `user_saves/quick/route.go` |
| `PUT /api/v1/auth/user/saves/quick_items/add` | `user_saves/quick/route.go` |
| `PUT /api/v1/auth/user/saves/quick_items/addlist` | `user_saves/quick/route.go` |
| `DELETE /api/v1/auth/user/saves/quick_items` | `user_saves/quick/route.go` |
| `DELETE /api/v1/auth/user/saves/quick_items/all` | `user_saves/quick/route.go` |
| `GET /api/v1/auth/user/saves/serialized_items` | `user_saves/serialized/route.go` |
| `PUT /api/v1/auth/user/saves/serialized_items/add` | `user_saves/serialized/route.go` |
| `PUT /api/v1/auth/user/saves/serialized_items/addlist` | `user_saves/serialized/route.go` |
| `DELETE /api/v1/auth/user/saves/serialized_items` | `user_saves/serialized/route.go` |
| `DELETE /api/v1/auth/user/saves/serialized_items/all` | `user_saves/serialized/route.go` |
| `GET /api/v1/auth/user/saves/item_category` | `user_saves/categories/route.go` |
| `PUT /api/v1/auth/user/saves/item_category` | `user_saves/categories/route.go` |
| `DELETE /api/v1/auth/user/saves/item_category` | `user_saves/categories/route.go` |
| `DELETE /api/v1/auth/user/saves/item_category/all` | `user_saves/categories/route.go` |
| `GET /api/v1/auth/user/saves/categorized_items/category` | `user_saves/categories/items/route.go` |
| `GET /api/v1/auth/user/saves/categorized_items` | `user_saves/categories/items/route.go` |
| `PUT /api/v1/auth/user/saves/categorized_items/add` | `user_saves/categories/items/route.go` |
| `PUT /api/v1/auth/user/saves/categorized_items/addlist` | `user_saves/categories/items/route.go` |
| `DELETE /api/v1/auth/user/saves/categorized_items` | `user_saves/categories/items/route.go` |
| `DELETE /api/v1/auth/user/saves/categorized_items/all` | `user_saves/categories/items/route.go` |
| `POST /api/v1/auth/user/saves/items/image/upload/:table_type` | `user_saves/images/route.go` |
| `DELETE /api/v1/auth/user/saves/items/image/:table_type` | `user_saves/images/route.go` |
| `GET /api/v1/auth/user/saves/items/image/:table_type` | `user_saves/images/route.go` |

---

## Testing Strategy

### Unit Tests Per Package

Each new package must include tests:

```go
// api/user_saves/quick/service_test.go
package quick

func TestGetByUser_Success(t *testing.T) {}
func TestGetByUser_NilUser(t *testing.T) {}
func TestUpsert_NewItem(t *testing.T) {}
func TestUpsert_ExistingItem(t *testing.T) {}
func TestUpsertBatch_Success(t *testing.T) {}
func TestUpsertBatch_EmptyList(t *testing.T) {}
func TestDelete_Success(t *testing.T) {}
func TestDelete_NotFound(t *testing.T) {}
func TestDeleteAll_Success(t *testing.T) {}
func TestDeleteAll_CleanupImages(t *testing.T) {}
```

### Integration Tests

```go
// api/user_saves/integration_test.go
package user_saves

func TestFullUserSavesLifecycle(t *testing.T) {
    // 1. Create quick save item
    // 2. Upload image to quick save item
    // 3. Create category
    // 4. Add item to category
    // 5. Upload image to categorized item
    // 6. Delete category (verify cascade)
    // 7. Verify images cleaned up
}
```

### Minimum Coverage Targets

| Package | Target Coverage |
|---------|-----------------|
| `user_saves/shared` | 90% |
| `user_saves/quick` | 85% |
| `user_saves/serialized` | 85% |
| `user_saves/categories` | 85% |
| `user_saves/categories/items` | 85% |
| `user_saves/images` | 80% |
| `user_saves/facade` | 90% |

---

## Risk Mitigation

### 1. Feature Flags

Use environment variables to toggle between old and new implementations:

```go
// bootstrap/env.go
type Env struct {
    // ... existing fields
    UseModularUserSaves bool  // Toggle new user saves implementation
}

// api/route/route.go
if env.UseModularUserSaves {
    user_saves.RegisterRoutes(deps, authRoutes)
} else {
    route.NewUserSavesRouter(db, blobClient, env, authRoutes)
}
```

### 2. Rollback Plan

Each step can be rolled back independently:

1. **Git branches**: Each step in separate branch
2. **Database**: No schema changes required
3. **Feature flag**: Instant rollback to legacy

### 3. Gradual Rollout

1. Deploy with feature flag disabled
2. Enable for internal testing
3. Enable for subset of users
4. Full rollout

---

## Success Metrics

### Code Quality

| Metric | Current | Target |
|--------|---------|--------|
| Largest file | 1,044 lines | < 200 lines |
| Methods per interface | 23 | < 7 |
| Code duplication | High | Low |
| Test coverage | 0% | > 80% |
| Domain errors | 0 typed | 12 typed |

### Maintainability

- **Time to understand a context**: 1 hour → 15 minutes
- **Time to add a new feature**: 4 hours → 1 hour
- **Number of files changed for a bug fix**: 3+ → 1-2

### Performance

- **No regression** in API response times
- **Same memory footprint**
- **Same database query patterns**

---

## Estimated Effort

| Step | Description | Estimated Effort |
|------|-------------|------------------|
| 1 | Create directory structure and shared package | 2 hours |
| 2 | Extract images bounded context | 3 hours |
| 3 | Extract quick items bounded context | 3 hours |
| 4 | Extract serialized items bounded context | 3 hours |
| 5 | Extract categories bounded context (with items) | 5 hours |
| 6 | Create facade service | 2 hours |
| 7 | Wire dependencies and update bootstrap | 2 hours |
| 8 | Testing and validation | 4 hours |
| 9 | Cleanup legacy files | 1 hour |
| **TOTAL** | | **~25 hours** |

---

## Appendix A: File Line Counts

### Current State

```
$ wc -l api/repository/user_saves_repository_impl.go
    1044 api/repository/user_saves_repository_impl.go

$ wc -l api/controller/user_saves_controller.go
     636 api/controller/user_saves_controller.go

$ wc -l api/service/user_saves_service_impl.go
     151 api/service/user_saves_service_impl.go

$ wc -l api/repository/user_saves_repository.go api/service/user_saves_service.go api/route/user_saves_route.go
      41 api/repository/user_saves_repository.go
      41 api/service/user_saves_service.go
      51 api/route/user_saves_route.go

TOTAL: ~1,964 lines
```

### Target State

```
api/user_saves/
├── route.go              (~60 lines)
├── facade/               (~200 lines total)
├── shared/               (~80 lines total)
├── quick/                (~300 lines total)
├── serialized/           (~300 lines total)
├── categories/           (~220 lines total)
│   └── items/            (~350 lines total)
└── images/               (~280 lines total)

TOTAL: ~1,790 lines (9% reduction through deduplication)
       + 80% test coverage
       + 12 typed domain errors
       + Transaction support
       + Clean separation of concerns
```

---

## Appendix B: Database Tables

| Package | Tables Accessed |
|---------|-----------------|
| `user_saves/quick` | `user_items_quick` |
| `user_saves/serialized` | `user_items_serialized` |
| `user_saves/categories` | `user_item_category` |
| `user_saves/categories/items` | `user_items_categorized` |
| `user_saves/images` | All above (for image URLs) |

---

## Appendix C: Comparison with Shops Pattern

This refactoring follows the same proven pattern from the shops domain:

| Aspect | Shops Refactor | User Saves Refactor |
|--------|---------------|---------------------|
| Bounded contexts | 8 | 5 |
| Facade pattern | Yes | Yes |
| Shared auth | `shared/authorization.go` | `shared/context.go` |
| Domain errors | `shared/errors.go` | `shared/errors.go` |
| Central wiring | `route.go` | `route.go` |
| Nested contexts | `vehicles/notifications/items` | `categories/items` |
| Cross-cutting | Blob storage | Blob storage |

---

*Document Version: 1.0*
*Created: January 2026*
*Author: Architecture Review*
*Based on: Shops Domain Refactoring Pattern*
