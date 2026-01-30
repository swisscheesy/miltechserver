# Material Images Domain Refactoring Plan

## Refactor Progress Tracker

Last updated: 2026-01-29

| Step | Status | Notes |
|------|--------|-------|
| 1. Create directory structure and shared package | Pending | Create `api/material_images/` with `shared/` utilities |
| 2. Extract ratelimit bounded context | Pending | Rate limiting for uploads |
| 3. Extract images bounded context | Pending | Core image CRUD operations |
| 4. Extract votes bounded context | Pending | Image voting management |
| 5. Extract flags bounded context | Pending | Image flagging/moderation |
| 6. Wire dependencies in central route | Pending | Update `api/route/route.go` to use new structure |
| 7. Testing and verification | Pending | Run tests, manual API validation |
| 8. Cleanup legacy files | Pending | Remove old monolithic files |

---

## Executive Summary

The `material_images` domain has grown into a **monolithic structure** spanning ~1,415 lines across five files:

| File | Lines | Responsibility |
|------|-------|----------------|
| `api/repository/material_images_repository_impl.go` | 599 | Data access for all material image entities |
| `api/controller/material_images_controller.go` | 353 | HTTP handlers for 10 endpoints |
| `api/service/material_images_service_impl.go` | 463 | Business logic with blob storage integration |
| `api/repository/material_images_repository.go` | 41 | Repository interface (15 methods) |
| `api/service/material_images_service.go` | 24 | Service interface (9 methods) |
| `api/route/material_images_route.go` | 48 | Route registration |

This plan outlines how to decompose these into **focused bounded contexts** following the same pattern successfully applied in the `shops`, `user_saves`, and `user_vehicles` domain refactors.

---

## Current Domain Analysis

### Identified Bounded Contexts

Based on the interface analysis from `api/service/material_images_service.go` and `api/repository/material_images_repository.go`, the material images domain contains **4 distinct sub-domains**:

```
material_images/
├── images/      # Core image CRUD and blob storage
├── votes/       # Image voting system
├── flags/       # Image flagging/moderation
└── ratelimit/   # Upload rate limiting
```

### Current Interface Method Count by Context

| Context | Service Methods | Repository Methods | Endpoints |
|---------|-----------------|-------------------|-----------|
| Images | 5 | 6 | 5 |
| Votes | 2 | 4 | 2 |
| Flags | 2 | 2 | 2 |
| Ratelimit | 0 | 3 | 0 (internal) |
| User Lookup | 0 | 1 | 0 (shared) |
| **TOTAL** | **9** | **16** | **9** |

### Database Tables

| Context | Table Name | Jet Table Reference |
|---------|------------|---------------------|
| Images | `material_images` | `MaterialImages` |
| Votes | `material_images_votes` | `MaterialImagesVotes` |
| Flags | `material_images_flags` | `MaterialImagesFlags` |
| Ratelimit | `material_image_upload_limits` | `MaterialImageUploadLimits` |

### Current Route Structure

```go
// Public routes (no auth required)
group.GET("/material-images/niin/:niin", ctrl.GetImagesByNIIN)
group.GET("/material-images/:image_id", ctrl.GetImageByID)

// Protected routes (auth required)
authGroup.POST("/material-images/upload", ctrl.UploadImage)
authGroup.DELETE("/material-images/:image_id", ctrl.DeleteImage)
authGroup.GET("/material-images/user/:user_id", ctrl.GetImagesByUser)

// Voting
authGroup.POST("/material-images/:image_id/vote", ctrl.VoteOnImage)
authGroup.DELETE("/material-images/:image_id/vote", ctrl.RemoveVote)

// Flagging
authGroup.POST("/material-images/:image_id/flag", ctrl.FlagImage)
authGroup.GET("/material-images/:image_id/flags", ctrl.GetImageFlags)
```

---

## Architectural Problems Identified

### 1. God Interface Anti-Pattern

The `MaterialImagesService` interface contains **9 methods** spanning 3 distinct concerns:

```go
// api/service/material_images_service.go - CURRENT STATE
type MaterialImagesService interface {
    // Image operations (5 methods)
    UploadImage(user *bootstrap.User, niin string, imageData []byte, filename string) (*model.MaterialImages, error)
    GetImagesByNIIN(niin string, page int, pageSize int, currentUser *bootstrap.User) ([]response.MaterialImageResponse, int64, error)
    GetImagesByUser(userID string, page int, pageSize int, currentUser *bootstrap.User) ([]response.MaterialImageResponse, int64, error)
    GetImageByID(imageID string, currentUser *bootstrap.User) (*response.MaterialImageResponse, error)
    DeleteImage(user *bootstrap.User, imageID string) error

    // Vote operations (2 methods)
    VoteOnImage(user *bootstrap.User, imageID string, voteType string) error
    RemoveVote(user *bootstrap.User, imageID string) error

    // Flag operations (2 methods)
    FlagImage(user *bootstrap.User, imageID string, reason string, description string) error
    GetImageFlags(imageID string) ([]model.MaterialImagesFlags, error)
}
```

### 2. Single Responsibility Principle Violation

The `MaterialImagesRepositoryImpl` struct handles:
- Core image CRUD with complex JOINs
- Vote management with upsert logic
- Flag management with cascading updates
- Rate limiting with time-based logic
- User lookup for username retrieval

**Evidence from code:**
```go
// Single struct handling 4 bounded contexts + shared concerns
type MaterialImagesRepositoryImpl struct {
    db *sql.DB
}
```

### 3. Controller Boilerplate Repetition

The `MaterialImagesController` repeats the same authentication pattern **8 times**:

```go
user, exists := c.Get("user")
if !exists {
    c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
    return
}
currentUser := user.(*bootstrap.User)
```

This can be consolidated using a shared helper function like in `user_saves/shared/context.go`.

### 4. Mixed Public/Authenticated Routes

The domain handles **both public and authenticated** routes:
- Public: `GET /material-images/niin/:niin`, `GET /material-images/:image_id`
- Authenticated: All other endpoints

This dual-authentication pattern requires careful handling in the refactored structure.

### 5. Blob Storage Integration Tightly Coupled

The service directly manages Azure Blob Storage operations:

```go
// service_impl.go
ctx := context.Background()
_, err = s.blobClient.UploadBuffer(ctx, ContainerName, blobName, imageData, nil)
```

This should be extracted to a shared blob utility.

### 6. Raw SQL Mixed with Jet

The repository uses both raw SQL (for complex JOINs) and go-jet queries:

```go
// Raw SQL for complex JOINs with user data
rawSQL := `
    SELECT
        mi.id, mi.niin, ...
        COALESCE(u.username, 'Unknown') as username
    FROM material_images mi
    LEFT JOIN users u ON mi.user_id = u.uid
    WHERE mi.niin = $1 AND mi.is_active = true
    ORDER BY mi.net_votes DESC, mi.upload_date DESC
    LIMIT $2 OFFSET $3
`
```

### 7. Testing Difficulty

- Cannot mock individual sub-domains
- Entire monolithic interface must be mocked for any test
- No unit tests currently exist for this domain

---

## Target Architecture

### Package Structure

```
api/
├── material_images/
│   ├── route.go                    # Central wiring & dependency injection
│   │
│   ├── shared/
│   │   ├── context.go              # User extraction helper
│   │   ├── errors.go               # Sentinel errors
│   │   └── blob.go                 # Blob storage utilities
│   │
│   ├── images/
│   │   ├── repository.go           # Interface (6 methods)
│   │   ├── repository_impl.go      # Data access (~250 lines)
│   │   ├── service.go              # Interface
│   │   ├── service_impl.go         # Business logic (~200 lines)
│   │   └── route.go                # 5 endpoints (public + auth mixed)
│   │
│   ├── votes/
│   │   ├── repository.go           # Interface (4 methods)
│   │   ├── repository_impl.go      # Data access (~100 lines)
│   │   ├── service.go              # Interface
│   │   ├── service_impl.go         # Business logic (~60 lines)
│   │   └── route.go                # 2 endpoints
│   │
│   ├── flags/
│   │   ├── repository.go           # Interface (2 methods)
│   │   ├── repository_impl.go      # Data access (~80 lines)
│   │   ├── service.go              # Interface
│   │   ├── service_impl.go         # Business logic (~60 lines)
│   │   └── route.go                # 2 endpoints
│   │
│   └── ratelimit/
│       ├── repository.go           # Interface (3 methods)
│       └── repository_impl.go      # Data access (~70 lines)
```

**Note:** No facade is needed for this domain since:
1. Sub-domains are relatively independent
2. Cross-context calls are limited (votes/flags only need image existence check)
3. Simpler than `shops` (no complex authorization chains)

---

## Interface Specifications

### images/service.go

```go
package images

import (
    "miltechserver/.gen/miltech_ng/public/model"
    "miltechserver/api/response"
    "miltechserver/bootstrap"
)

type Service interface {
    Upload(user *bootstrap.User, niin string, imageData []byte, filename string) (*model.MaterialImages, error)
    GetByNIIN(niin string, page int, pageSize int, currentUser *bootstrap.User) ([]response.MaterialImageResponse, int64, error)
    GetByUser(userID string, page int, pageSize int, currentUser *bootstrap.User) ([]response.MaterialImageResponse, int64, error)
    GetByID(imageID string, currentUser *bootstrap.User) (*response.MaterialImageResponse, error)
    Delete(user *bootstrap.User, imageID string) error
}
```

### images/repository.go

```go
package images

import (
    "miltechserver/.gen/miltech_ng/public/model"
    "miltechserver/bootstrap"
)

// ImageWithUser combines image data with username for display
type ImageWithUser struct {
    model.MaterialImages
    Username *string
}

type Repository interface {
    Create(user *bootstrap.User, image model.MaterialImages) (*model.MaterialImages, error)
    GetByID(imageID string) (*model.MaterialImages, error)
    GetByNIIN(niin string, limit int, offset int) ([]ImageWithUser, int64, error)
    GetByUser(userID string, limit int, offset int) ([]ImageWithUser, int64, error)
    UpdateFlags(imageID string, flagCount int, isFlagged bool) error
    Delete(imageID string) error
    GetUsernameByUserID(userID string) (string, error)
}
```

### votes/service.go

```go
package votes

import "miltechserver/bootstrap"

type Service interface {
    Vote(user *bootstrap.User, imageID string, voteType string) error
    RemoveVote(user *bootstrap.User, imageID string) error
}
```

### votes/repository.go

```go
package votes

import "miltechserver/.gen/miltech_ng/public/model"

type Repository interface {
    Upsert(vote model.MaterialImagesVotes) error
    Delete(imageID string, userID string) error
    GetUserVote(imageID string, userID string) (*model.MaterialImagesVotes, error)
    UpdateImageCounts(imageID string) error
}
```

### flags/service.go

```go
package flags

import (
    "miltechserver/.gen/miltech_ng/public/model"
    "miltechserver/bootstrap"
)

type Service interface {
    Flag(user *bootstrap.User, imageID string, reason string, description string) error
    GetByImage(imageID string) ([]model.MaterialImagesFlags, error)
}
```

### flags/repository.go

```go
package flags

import "miltechserver/.gen/miltech_ng/public/model"

type Repository interface {
    Create(flag model.MaterialImagesFlags) error
    GetByImage(imageID string) ([]model.MaterialImagesFlags, error)
}
```

### ratelimit/repository.go

```go
package ratelimit

import "time"

type Repository interface {
    CheckLimit(userID string, niin string) (bool, *time.Time, error)
    UpdateLimit(userID string, niin string) error
    CleanupOld(olderThan time.Time) error
}
```

---

## Central Route Wiring

### material_images/route.go

```go
package material_images

import (
    "database/sql"
    "miltechserver/api/material_images/flags"
    "miltechserver/api/material_images/images"
    "miltechserver/api/material_images/ratelimit"
    "miltechserver/api/material_images/votes"
    "miltechserver/bootstrap"

    "firebase.google.com/go/v4/auth"
    "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
    "github.com/gin-gonic/gin"
)

type Dependencies struct {
    DB         *sql.DB
    BlobClient *azblob.Client
    Env        *bootstrap.Env
    AuthClient *auth.Client
}

func RegisterRoutes(deps Dependencies, publicRouter *gin.RouterGroup, authRouter *gin.RouterGroup) {
    // Create repositories
    rateLimitRepo := ratelimit.NewRepository(deps.DB)
    imagesRepo := images.NewRepository(deps.DB)
    votesRepo := votes.NewRepository(deps.DB)
    flagsRepo := flags.NewRepository(deps.DB)

    // Create services
    imagesService := images.NewService(imagesRepo, rateLimitRepo, deps.BlobClient, deps.Env)
    votesService := votes.NewService(votesRepo, imagesRepo)
    flagsService := flags.NewService(flagsRepo, imagesRepo)

    // Register routes for each bounded context
    images.RegisterRoutes(publicRouter, authRouter, imagesService, deps.AuthClient)
    votes.RegisterRoutes(authRouter, votesService)
    flags.RegisterRoutes(authRouter, flagsService)
}
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
1. Create `api/material_images/` directory structure
2. Create `shared/context.go` with user extraction helper
3. Create `shared/errors.go` with sentinel errors
4. Create `shared/blob.go` with blob storage utilities
5. Create `route.go` skeleton that delegates to old implementation initially

**New Files**:
```
api/material_images/
├── route.go
└── shared/
    ├── context.go
    ├── errors.go
    └── blob.go
```

**shared/context.go**:
```go
package shared

import (
    "errors"
    "miltechserver/bootstrap"

    "github.com/gin-gonic/gin"
)

// GetUserFromContext extracts the authenticated user from gin context
func GetUserFromContext(c *gin.Context) (*bootstrap.User, error) {
    ctxUser, ok := c.Get("user")
    if !ok {
        return nil, errors.New("unauthorized")
    }

    user, ok := ctxUser.(*bootstrap.User)
    if !ok || user == nil {
        return nil, errors.New("unauthorized")
    }

    return user, nil
}

// GetOptionalUserFromContext extracts user if present, returns nil if not authenticated
func GetOptionalUserFromContext(c *gin.Context) *bootstrap.User {
    user, _ := GetUserFromContext(c)
    return user
}
```

**shared/errors.go**:
```go
package shared

import "errors"

// Image errors
var (
    ErrImageNotFound   = errors.New("image not found")
    ErrUnauthorized    = errors.New("unauthorized")
    ErrForbidden       = errors.New("forbidden: you can only modify your own images")
    ErrInvalidNIIN     = errors.New("NIIN must be exactly 9 characters")
    ErrRateLimited     = errors.New("upload rate limit exceeded")
)

// Vote errors
var (
    ErrInvalidVoteType = errors.New("invalid vote type: must be 'upvote' or 'downvote'")
    ErrVoteNotFound    = errors.New("vote not found")
)

// Flag errors
var (
    ErrAlreadyFlagged  = errors.New("you have already flagged this image")
    ErrInvalidReason   = errors.New("invalid flag reason")
)
```

**shared/blob.go**:
```go
package shared

import (
    "context"
    "fmt"

    "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
    "github.com/google/uuid"
)

const ContainerName = "material-images"

// BlobStorage provides utilities for Azure Blob Storage operations
type BlobStorage struct {
    client *azblob.Client
}

func NewBlobStorage(client *azblob.Client) *BlobStorage {
    return &BlobStorage{client: client}
}

// Upload stores image data and returns the blob URL
func (b *BlobStorage) Upload(niin string, imageData []byte) (string, error) {
    if b.client == nil {
        return "", nil // Graceful degradation when blob client unavailable
    }

    blobName := fmt.Sprintf("%s/%s", niin, uuid.New().String())
    ctx := context.Background()

    _, err := b.client.UploadBuffer(ctx, ContainerName, blobName, imageData, nil)
    if err != nil {
        return "", fmt.Errorf("failed to upload to blob storage: %w", err)
    }

    return blobName, nil
}

// Delete removes an image from blob storage
func (b *BlobStorage) Delete(blobName string) error {
    if b.client == nil {
        return nil // Graceful degradation
    }

    ctx := context.Background()
    _, err := b.client.DeleteBlob(ctx, ContainerName, blobName, nil)
    return err
}

// GetURL returns the full URL for a blob
func (b *BlobStorage) GetURL(blobName string, baseURL string) string {
    if blobName == "" {
        return ""
    }
    return fmt.Sprintf("%s/%s/%s", baseURL, ContainerName, blobName)
}
```

---

#### Step 2: Extract Ratelimit Bounded Context

**Current Location**:
- Repository: `material_images_repository.go:34-37` (3 methods)
- Used by: `images` service for upload throttling

**New Files**:
```
api/material_images/ratelimit/
├── repository.go        # ~20 lines
└── repository_impl.go   # ~70 lines
```

**Migration Steps**:
1. Create new package with interface
2. Copy repository implementation for rate limiting methods
3. Inject into images service

---

#### Step 3: Extract Images Bounded Context

**Current Location**:
- Repository: `material_images_repository.go:16-22` (6 methods + user lookup)
- Service: `material_images_service.go:10-15` (5 methods)
- Controller: `material_images_controller.go:25-213` (5 handlers)
- Routes: `material_images_route.go:27-35` (5 endpoints)

**New Files**:
```
api/material_images/images/
├── repository.go        # ~30 lines
├── repository_impl.go   # ~250 lines
├── service.go           # ~25 lines
├── service_impl.go      # ~200 lines
└── route.go             # ~120 lines
```

**Special Considerations**:
- Handle both public and authenticated routes
- Integrate with ratelimit repository for upload throttling
- Use shared blob storage utilities

**Route Handler Pattern**:
```go
// api/material_images/images/route.go
package images

import (
    "net/http"
    "miltechserver/api/material_images/shared"
    "miltechserver/api/request"
    "miltechserver/api/response"

    "firebase.google.com/go/v4/auth"
    "github.com/gin-gonic/gin"
)

type Handler struct {
    service Service
}

func RegisterRoutes(publicRouter *gin.RouterGroup, authRouter *gin.RouterGroup, service Service, authClient *auth.Client) {
    handler := Handler{service: service}

    // Public routes
    publicRouter.GET("/material-images/niin/:niin", handler.getByNIIN)
    publicRouter.GET("/material-images/:image_id", handler.getByID)

    // Authenticated routes
    authRouter.POST("/material-images/upload", handler.upload)
    authRouter.DELETE("/material-images/:image_id", handler.delete)
    authRouter.GET("/material-images/user/:user_id", handler.getByUser)
}

func (h *Handler) upload(c *gin.Context) {
    user, err := shared.GetUserFromContext(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    // ... rest of handler
}

func (h *Handler) getByNIIN(c *gin.Context) {
    // Get optional user for personalized response (e.g., showing user's vote)
    currentUser := shared.GetOptionalUserFromContext(c)

    // ... rest of handler
}

// ... additional handlers follow the same pattern
```

---

#### Step 4: Extract Votes Bounded Context

**Current Location**:
- Repository: `material_images_repository.go:24-28` (4 methods)
- Service: `material_images_service.go:17-19` (2 methods)
- Controller: `material_images_controller.go:215-290` (2 handlers)
- Routes: `material_images_route.go:37-39` (2 endpoints)

**New Files**:
```
api/material_images/votes/
├── repository.go        # ~20 lines
├── repository_impl.go   # ~100 lines
├── service.go           # ~15 lines
├── service_impl.go      # ~60 lines
└── route.go             # ~70 lines
```

**Migration Steps**:
1. Create new package with interfaces
2. Copy repository implementation for vote methods
3. Copy service implementation for vote operations
4. Service needs images repository to verify image exists and get updated counts
5. Create route handlers using shared context helper
6. Wire up in central route registration

---

#### Step 5: Extract Flags Bounded Context

**Current Location**:
- Repository: `material_images_repository.go:30-31` (2 methods)
- Service: `material_images_service.go:21-23` (2 methods)
- Controller: `material_images_controller.go:292-353` (2 handlers)
- Routes: `material_images_route.go:41-45` (2 endpoints)

**New Files**:
```
api/material_images/flags/
├── repository.go        # ~15 lines
├── repository_impl.go   # ~80 lines
├── service.go           # ~15 lines
├── service_impl.go      # ~60 lines
└── route.go             # ~60 lines
```

**Migration Steps**:
1. Create new package with interfaces
2. Copy repository implementation for flag methods
3. Copy service implementation for flag operations
4. Service needs images repository to update flag counts on image
5. Create route handlers using shared context helper
6. Wire up in central route registration

---

#### Step 6: Wire Dependencies and Update Routing

**Update `api/route/route.go`**:

```go
// BEFORE (in NewRouter function)
route.NewMaterialImagesRouter(db, blobClient, env, authClient, v1Route, authRoutes)

// AFTER
material_images.RegisterRoutes(material_images.Dependencies{
    DB:         db,
    BlobClient: blobClient,
    Env:        env,
    AuthClient: authClient,
}, v1Route, authRoutes)
```

**Import Changes**:
```go
import (
    // ... existing imports
    "miltechserver/api/material_images"
)
```

---

#### Step 7: Testing and Verification

**Manual API Testing Checklist**:

| Endpoint | Method | Auth | Expected Result |
|----------|--------|------|-----------------|
| `/api/v1/material-images/niin/:niin` | GET | No | Returns images for NIIN |
| `/api/v1/material-images/:image_id` | GET | No | Returns specific image |
| `/api/v1/auth/material-images/upload` | POST | Yes | Uploads new image |
| `/api/v1/auth/material-images/:image_id` | DELETE | Yes | Deletes user's image |
| `/api/v1/auth/material-images/user/:user_id` | GET | Yes | Returns user's images |
| `/api/v1/auth/material-images/:image_id/vote` | POST | Yes | Records vote |
| `/api/v1/auth/material-images/:image_id/vote` | DELETE | Yes | Removes vote |
| `/api/v1/auth/material-images/:image_id/flag` | POST | Yes | Flags image |
| `/api/v1/auth/material-images/:image_id/flags` | GET | Yes | Returns image flags |

**Unit Test Requirements**:
- Service layer tests for each bounded context
- Mock repository for isolated testing
- Edge cases: nil user, not found, rate limiting, duplicate flags

---

#### Step 8: Cleanup Legacy Files

**Files to Remove**:
```
api/controller/material_images_controller.go
api/service/material_images_service.go
api/service/material_images_service_impl.go
api/repository/material_images_repository.go
api/repository/material_images_repository_impl.go
api/route/material_images_route.go
```

**Update Imports**: Search and replace all imports pointing to old packages

---

## API Backward Compatibility

### URL Structure Preservation

All existing URLs **MUST** remain unchanged:

| Current URL | New Handler Location |
|-------------|---------------------|
| `GET /api/v1/material-images/niin/:niin` | `material_images/images/route.go` |
| `GET /api/v1/material-images/:image_id` | `material_images/images/route.go` |
| `POST /api/v1/auth/material-images/upload` | `material_images/images/route.go` |
| `DELETE /api/v1/auth/material-images/:image_id` | `material_images/images/route.go` |
| `GET /api/v1/auth/material-images/user/:user_id` | `material_images/images/route.go` |
| `POST /api/v1/auth/material-images/:image_id/vote` | `material_images/votes/route.go` |
| `DELETE /api/v1/auth/material-images/:image_id/vote` | `material_images/votes/route.go` |
| `POST /api/v1/auth/material-images/:image_id/flag` | `material_images/flags/route.go` |
| `GET /api/v1/auth/material-images/:image_id/flags` | `material_images/flags/route.go` |

---

## Risk Mitigation

### 1. Rollback Plan

Each step can be rolled back independently:

1. **Git branches**: Each step in separate commit
2. **Database**: No schema changes required
3. **Route switching**: Can revert to old `NewMaterialImagesRouter` instantly

### 2. Gradual Rollout

1. Deploy with new structure
2. Monitor for errors in logs
3. Verify all endpoints via API testing
4. Cleanup legacy files only after verification

### 3. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Route path changes break clients | Low | High | Keep exact same paths |
| Blob storage operations break | Medium | High | Extract shared blob utilities with nil checks |
| Mixed auth routes break | Medium | Medium | Careful router group handling |
| Vote count consistency | Low | Medium | Transaction handling in votes context |
| Rate limiting bypass | Low | High | Verify rate limit integration with images upload |
| Import cycles | Low | Low | Clean package boundaries |

---

## Success Metrics

### Code Quality

| Metric | Current | Target |
|--------|---------|--------|
| Largest file | 599 lines | < 200 lines |
| Methods per interface | 15 | < 7 |
| Test coverage | 0% | > 80% |
| Domain errors | 0 typed | 6 typed |

### Maintainability

- **Time to understand a context**: 30 minutes → 10 minutes
- **Time to add a new feature**: 2 hours → 45 minutes
- **Number of files changed for a bug fix**: 3+ → 1-2

### Performance

- **No regression** in API response times
- **Same memory footprint**
- **Same database query patterns**

---

## Estimated Effort

| Step | Description | Estimated Effort |
|------|-------------|------------------|
| 1 | Create directory structure and shared package | 1.5 hours |
| 2 | Extract ratelimit bounded context | 1 hour |
| 3 | Extract images bounded context | 3 hours |
| 4 | Extract votes bounded context | 1.5 hours |
| 5 | Extract flags bounded context | 1.5 hours |
| 6 | Wire dependencies and update routing | 1 hour |
| 7 | Testing and verification | 2 hours |
| 8 | Cleanup legacy files | 30 minutes |
| **TOTAL** | | **~12 hours** |

---

## Comparison with Previous Refactors

| Aspect | Shops Refactor | User Saves Refactor | User Vehicles Refactor | Material Images Refactor |
|--------|---------------|---------------------|------------------------|--------------------------|
| Bounded contexts | 8 | 5 | 4 | 4 |
| Facade pattern | Yes | Yes | No | No |
| Shared auth | `shared/authorization.go` | `shared/context.go` | `shared/context.go` | `shared/context.go` |
| Domain errors | `shared/errors.go` | `shared/errors.go` | `shared/errors.go` | `shared/errors.go` |
| Central wiring | `route.go` | `route.go` | `route.go` | `route.go` |
| Cross-cutting concerns | Blob storage | Blob storage | None | Blob storage |
| Mixed auth routes | No | No | No | Yes (public + auth) |
| Complexity | High | Medium | Low-Medium | Medium |

---

## Appendix A: File Line Counts

### Current State

```
$ wc -l api/repository/material_images_repository_impl.go
     599 api/repository/material_images_repository_impl.go

$ wc -l api/controller/material_images_controller.go
     353 api/controller/material_images_controller.go

$ wc -l api/service/material_images_service_impl.go
     463 api/service/material_images_service_impl.go

$ wc -l api/repository/material_images_repository.go api/service/material_images_service.go api/route/material_images_route.go
      41 api/repository/material_images_repository.go
      24 api/service/material_images_service.go
      48 api/route/material_images_route.go

TOTAL: ~1,528 lines
```

### Target State

```
api/material_images/
├── route.go              (~40 lines)
├── shared/               (~100 lines total)
├── ratelimit/            (~90 lines total)
├── images/               (~500 lines total)
├── votes/                (~200 lines total)
└── flags/                (~200 lines total)

TOTAL: ~1,130 lines (26% reduction through deduplication)
       + Unit test coverage
       + 6 typed domain errors
       + Clean separation of concerns
```

---

## Appendix B: Database Tables

| Package | Tables Accessed | Primary Key |
|---------|-----------------|-------------|
| `material_images/images` | `material_images`, `users` | `id` |
| `material_images/votes` | `material_images_votes`, `material_images` | `id` |
| `material_images/flags` | `material_images_flags`, `material_images` | `id` |
| `material_images/ratelimit` | `material_image_upload_limits` | `user_id`, `niin` (composite) |

---

## Appendix C: Cross-Context Dependencies

```
┌─────────────────────────────────────────────────────────────┐
│                    material_images/route.go                  │
│                    (Dependency Injection)                    │
└─────────────────────────────────────────────────────────────┘
                              │
         ┌────────────────────┼────────────────────┐
         ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│     images      │  │      votes      │  │      flags      │
│   (Service)     │  │    (Service)    │  │    (Service)    │
└────────┬────────┘  └────────┬────────┘  └────────┬────────┘
         │                    │                    │
         │           ┌────────┴────────┐  ┌────────┴────────┐
         │           │ Needs images    │  │ Needs images    │
         │           │ repo to verify  │  │ repo to update  │
         │           │ existence &     │  │ flag counts     │
         │           │ update counts   │  │                 │
         │           └────────┬────────┘  └────────┬────────┘
         │                    │                    │
┌────────┴────────┐  ┌────────┴────────┐  ┌────────┴────────┐
│   ratelimit     │  │  images repo    │  │  images repo    │
│ (Repository)    │◄─┤  (Repository)   │  │  (Repository)   │
└─────────────────┘  └─────────────────┘  └─────────────────┘
         │
         │ Used by images
         │ service for
         │ upload throttling
         │
         ▼
┌─────────────────┐
│  images service │
└─────────────────┘
```

---

*Document Version: 1.0*
*Created: January 29, 2026*
*Author: Architecture Review*
*Based on: User Saves & User Vehicles Domain Refactoring Pattern*
