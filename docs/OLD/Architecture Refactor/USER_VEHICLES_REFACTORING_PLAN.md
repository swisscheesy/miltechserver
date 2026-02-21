# User Vehicles Domain Refactoring Plan

## Refactor Progress Tracker

Last updated: 2026-01-29

| Step | Status | Notes |
|------|--------|-------|
| 1. Create directory structure and shared package | Pending | Create `api/user_vehicles/` with `shared/` utilities |
| 2. Extract Vehicles bounded context | Pending | Core vehicle CRUD operations |
| 3. Extract Notifications bounded context | Pending | Vehicle notification management |
| 4. Extract Comments bounded context | Pending | Vehicle comments (currently unused per route comment) |
| 5. Extract Notification Items bounded context | Pending | Items associated with notifications |
| 6. Wire dependencies in central route | Pending | Update `api/route/route.go` to use new structure |
| 7. Testing and verification | Pending | Run existing tests, manual API validation |
| 8. Cleanup legacy files | Pending | Remove old monolithic files |

---

## Executive Summary

The `user_vehicle` domain has grown into a **monolithic structure** spanning ~1,900 lines across six files:

| File | Lines | Responsibility |
|------|-------|----------------|
| `api/repository/user_vehicle_repository_impl.go` | 594 | Data access for all user vehicle entities |
| `api/controller/user_vehicle_controller.go` | 681 | HTTP handlers for 24 endpoints |
| `api/service/user_vehicle_service_impl.go` | 181 | Pass-through business logic |
| `api/repository/user_vehicle_repository.go` | 42 | Repository interface (42 methods) |
| `api/service/user_vehicle_service.go` | 41 | Service interface (25 methods) |
| `api/route/user_vehicle_route.go` | 52 | Route registration |

This plan outlines how to decompose these into **focused bounded contexts** following the same pattern successfully applied in the `shops` and `user_saves` domain refactors.

---

## Current Domain Analysis

### Identified Bounded Contexts

Based on the interface analysis from `api/service/user_vehicle_service.go`, the user vehicles domain contains **4 distinct sub-domains**:

```
user_vehicles/
├── vehicles/            # Core vehicle management
├── notifications/       # Vehicle maintenance notifications
├── comments/            # Vehicle comments (currently unused)
└── notification_items/  # Items linked to notifications
```

### Current Interface Method Count by Context

| Context | Service Methods | Repository Methods | Endpoints |
|---------|-----------------|-------------------|-----------|
| Vehicles | 5 | 6 | 5 |
| Notifications | 6 | 6 | 6 |
| Comments | 7 | 7 | 7 |
| Notification Items | 7 | 7 | 7 |
| **TOTAL** | **25** | **26** | **25** |

### Database Tables

| Context | Table Name | Jet Table Reference |
|---------|------------|---------------------|
| Vehicles | `user_vehicle` | `UserVehicle` |
| Notifications | `user_vehicle_notifications` | `UserVehicleNotifications` |
| Comments | `user_vehicle_comments` | `UserVehicleComments` |
| Notification Items | `user_notification_items` | `UserNotificationItems` |

### Current Route Structure

```go
// User Vehicle Routes (5 endpoints)
group.GET("/user/vehicles", ...)
group.GET("/user/vehicles/:vehicleId", ...)
group.PUT("/user/vehicles", ...)
group.DELETE("/user/vehicles/:vehicleId", ...)
group.DELETE("/user/vehicles", ...)

// User Vehicle Notifications Routes (6 endpoints)
group.GET("/user/vehicle-notifications", ...)
group.GET("/user/vehicle-notifications/vehicle/:vehicleId", ...)
group.GET("/user/vehicle-notifications/:notificationId", ...)
group.PUT("/user/vehicle-notifications", ...)
group.DELETE("/user/vehicle-notifications/:notificationId", ...)
group.DELETE("/user/vehicle-notifications/vehicle/:vehicleId", ...)

// User Vehicle Comments Routes -- Not Used (7 endpoints)
group.GET("/user/vehicle-comments", ...)
group.GET("/user/vehicle-comments/vehicle/:vehicleId", ...)
group.GET("/user/vehicle-comments/notification/:notificationId", ...)
group.GET("/user/vehicle-comments/:commentId", ...)
group.PUT("/user/vehicle-comments", ...)
group.DELETE("/user/vehicle-comments/:commentId", ...)
group.DELETE("/user/vehicle-comments/vehicle/:vehicleId", ...)

// User Notification Items Routes (7 endpoints)
group.GET("/user/notification-items", ...)
group.GET("/user/notification-items/notification/:notificationId", ...)
group.GET("/user/notification-items/:itemId", ...)
group.PUT("/user/notification-items", ...)
group.PUT("/user/notification-items/list", ...)
group.DELETE("/user/notification-items/:itemId", ...)
group.DELETE("/user/notification-items/notification/:notificationId", ...)
```

---

## Architectural Problems Identified

### 1. God Interface Anti-Pattern

The `UserVehicleService` interface contains **25 methods** spanning 4 different concerns:

```go
// api/service/user_vehicle_service.go - CURRENT STATE
type UserVehicleService interface {
    // User Vehicle Operations (5 methods)
    GetUserVehiclesByUser(user *bootstrap.User) ([]model.UserVehicle, error)
    GetUserVehicleById(user *bootstrap.User, vehicleId string) (*model.UserVehicle, error)
    UpsertUserVehicle(user *bootstrap.User, vehicle model.UserVehicle) error
    DeleteUserVehicle(user *bootstrap.User, vehicleId string) error
    DeleteAllUserVehicles(user *bootstrap.User) error

    // User Vehicle Notifications Operations (6 methods)
    GetVehicleNotificationsByUser(user *bootstrap.User) ([]model.UserVehicleNotifications, error)
    GetVehicleNotificationsByVehicle(user *bootstrap.User, vehicleId string) ([]model.UserVehicleNotifications, error)
    GetVehicleNotificationById(user *bootstrap.User, notificationId string) (*model.UserVehicleNotifications, error)
    UpsertVehicleNotification(user *bootstrap.User, notification model.UserVehicleNotifications) error
    DeleteVehicleNotification(user *bootstrap.User, notificationId string) error
    DeleteAllVehicleNotificationsByVehicle(user *bootstrap.User, vehicleId string) error

    // User Vehicle Comments Operations (7 methods)
    // ... 7 more methods

    // User Notification Items Operations (7 methods)
    // ... 7 more methods
}
```

### 2. Single Responsibility Principle Violation

The `UserVehicleRepositoryImpl` struct handles:
- Core vehicle CRUD operations
- Vehicle notification management
- Vehicle comment management
- Notification item management

**Evidence from code:**
```go
// Single struct handling 4 bounded contexts
type UserVehicleRepositoryImpl struct {
    Db *sql.DB
}
```

### 3. Controller Boilerplate Repetition

The `UserVehicleController` repeats the same authentication pattern **24 times**:

```go
ctxUser, ok := c.Get("user")
user, _ := ctxUser.(*bootstrap.User)
if !ok {
    c.JSON(401, gin.H{"message": "unauthorized"})
    slog.Info("Unauthorized request")
    return
}
```

This can be consolidated using a shared helper function like in `user_saves/shared/context.go`.

### 4. Thin Service Layer

The service layer is essentially a pass-through with no business logic:

```go
func (service *UserVehicleServiceImpl) GetUserVehiclesByUser(user *bootstrap.User) ([]model.UserVehicle, error) {
    vehicles, err := service.UserVehicleRepository.GetUserVehiclesByUserId(user)
    if vehicles == nil {
        return []model.UserVehicle{}, nil
    }
    return vehicles, err
}
```

### 5. Testing Difficulty

- Cannot mock individual sub-domains
- Entire monolithic interface must be mocked for any test
- No unit tests currently exist for this domain

### 6. Unused Feature Still Included

The comments feature has a route comment indicating it's "Not Used", but all 7 endpoints and their implementations are still maintained. This adds cognitive load and maintenance burden.

---

## Target Architecture

### Package Structure

```
api/
├── user_vehicles/
│   ├── route.go                    # Central wiring & dependency injection
│   │
│   ├── shared/
│   │   ├── context.go              # User extraction helper (reuse from user_saves)
│   │   └── errors.go               # Sentinel errors
│   │
│   ├── vehicles/
│   │   ├── repository.go           # Interface (5 methods)
│   │   ├── repository_impl.go      # Data access (~150 lines)
│   │   ├── service.go              # Interface
│   │   ├── service_impl.go         # Business logic (~60 lines)
│   │   └── route.go                # 5 endpoints
│   │
│   ├── notifications/
│   │   ├── repository.go           # Interface (6 methods)
│   │   ├── repository_impl.go      # Data access (~150 lines)
│   │   ├── service.go              # Interface
│   │   ├── service_impl.go         # Business logic (~80 lines)
│   │   └── route.go                # 6 endpoints
│   │
│   ├── comments/
│   │   ├── repository.go           # Interface (7 methods)
│   │   ├── repository_impl.go      # Data access (~180 lines)
│   │   ├── service.go              # Interface
│   │   ├── service_impl.go         # Business logic (~90 lines)
│   │   └── route.go                # 7 endpoints
│   │
│   └── notification_items/
│       ├── repository.go           # Interface (7 methods)
│       ├── repository_impl.go      # Data access (~180 lines)
│       ├── service.go              # Interface
│       ├── service_impl.go         # Business logic (~90 lines)
│       └── route.go                # 7 endpoints
```

**Note:** No facade is needed for this domain since:
1. Sub-domains are independent (no cross-domain operations)
2. Simpler than `user_saves` (no shared image management)
3. Follows the `user_saves` pattern where facade is optional for simpler domains

---

## Interface Specifications

### vehicles/service.go

```go
package vehicles

import (
    "miltechserver/.gen/miltech_ng/public/model"
    "miltechserver/bootstrap"
)

type Service interface {
    GetByUser(user *bootstrap.User) ([]model.UserVehicle, error)
    GetByID(user *bootstrap.User, vehicleID string) (*model.UserVehicle, error)
    Upsert(user *bootstrap.User, vehicle model.UserVehicle) error
    Delete(user *bootstrap.User, vehicleID string) error
    DeleteAll(user *bootstrap.User) error
}
```

### vehicles/repository.go

```go
package vehicles

import (
    "miltechserver/.gen/miltech_ng/public/model"
    "miltechserver/bootstrap"
)

type Repository interface {
    GetByUserID(user *bootstrap.User) ([]model.UserVehicle, error)
    GetByID(user *bootstrap.User, vehicleID string) (*model.UserVehicle, error)
    Upsert(user *bootstrap.User, vehicle model.UserVehicle) error
    Delete(user *bootstrap.User, vehicleID string) error
    DeleteAll(user *bootstrap.User) error
}
```

### notifications/service.go

```go
package notifications

import (
    "miltechserver/.gen/miltech_ng/public/model"
    "miltechserver/bootstrap"
)

type Service interface {
    GetByUser(user *bootstrap.User) ([]model.UserVehicleNotifications, error)
    GetByVehicle(user *bootstrap.User, vehicleID string) ([]model.UserVehicleNotifications, error)
    GetByID(user *bootstrap.User, notificationID string) (*model.UserVehicleNotifications, error)
    Upsert(user *bootstrap.User, notification model.UserVehicleNotifications) error
    Delete(user *bootstrap.User, notificationID string) error
    DeleteAllByVehicle(user *bootstrap.User, vehicleID string) error
}
```

### notifications/repository.go

```go
package notifications

import (
    "miltechserver/.gen/miltech_ng/public/model"
    "miltechserver/bootstrap"
)

type Repository interface {
    GetByUserID(user *bootstrap.User) ([]model.UserVehicleNotifications, error)
    GetByVehicleID(user *bootstrap.User, vehicleID string) ([]model.UserVehicleNotifications, error)
    GetByID(user *bootstrap.User, notificationID string) (*model.UserVehicleNotifications, error)
    Upsert(user *bootstrap.User, notification model.UserVehicleNotifications) error
    Delete(user *bootstrap.User, notificationID string) error
    DeleteAllByVehicle(user *bootstrap.User, vehicleID string) error
}
```

### comments/service.go

```go
package comments

import (
    "miltechserver/.gen/miltech_ng/public/model"
    "miltechserver/bootstrap"
)

type Service interface {
    GetByUser(user *bootstrap.User) ([]model.UserVehicleComments, error)
    GetByVehicle(user *bootstrap.User, vehicleID string) ([]model.UserVehicleComments, error)
    GetByNotification(user *bootstrap.User, notificationID string) ([]model.UserVehicleComments, error)
    GetByID(user *bootstrap.User, commentID string) (*model.UserVehicleComments, error)
    Upsert(user *bootstrap.User, comment model.UserVehicleComments) error
    Delete(user *bootstrap.User, commentID string) error
    DeleteAllByVehicle(user *bootstrap.User, vehicleID string) error
}
```

### comments/repository.go

```go
package comments

import (
    "miltechserver/.gen/miltech_ng/public/model"
    "miltechserver/bootstrap"
)

type Repository interface {
    GetByUserID(user *bootstrap.User) ([]model.UserVehicleComments, error)
    GetByVehicleID(user *bootstrap.User, vehicleID string) ([]model.UserVehicleComments, error)
    GetByNotificationID(user *bootstrap.User, notificationID string) ([]model.UserVehicleComments, error)
    GetByID(user *bootstrap.User, commentID string) (*model.UserVehicleComments, error)
    Upsert(user *bootstrap.User, comment model.UserVehicleComments) error
    Delete(user *bootstrap.User, commentID string) error
    DeleteAllByVehicle(user *bootstrap.User, vehicleID string) error
}
```

### notification_items/service.go

```go
package notification_items

import (
    "miltechserver/.gen/miltech_ng/public/model"
    "miltechserver/bootstrap"
)

type Service interface {
    GetByUser(user *bootstrap.User) ([]model.UserNotificationItems, error)
    GetByNotification(user *bootstrap.User, notificationID string) ([]model.UserNotificationItems, error)
    GetByID(user *bootstrap.User, itemID string) (*model.UserNotificationItems, error)
    Upsert(user *bootstrap.User, item model.UserNotificationItems) error
    UpsertBatch(user *bootstrap.User, items []model.UserNotificationItems) error
    Delete(user *bootstrap.User, itemID string) error
    DeleteAllByNotification(user *bootstrap.User, notificationID string) error
}
```

### notification_items/repository.go

```go
package notification_items

import (
    "miltechserver/.gen/miltech_ng/public/model"
    "miltechserver/bootstrap"
)

type Repository interface {
    GetByUserID(user *bootstrap.User) ([]model.UserNotificationItems, error)
    GetByNotificationID(user *bootstrap.User, notificationID string) ([]model.UserNotificationItems, error)
    GetByID(user *bootstrap.User, itemID string) (*model.UserNotificationItems, error)
    Upsert(user *bootstrap.User, item model.UserNotificationItems) error
    UpsertBatch(user *bootstrap.User, items []model.UserNotificationItems) error
    Delete(user *bootstrap.User, itemID string) error
    DeleteAllByNotification(user *bootstrap.User, notificationID string) error
}
```

---

## Central Route Wiring

### user_vehicles/route.go

```go
package user_vehicles

import (
    "database/sql"
    "miltechserver/api/user_vehicles/comments"
    "miltechserver/api/user_vehicles/notifications"
    "miltechserver/api/user_vehicles/notification_items"
    "miltechserver/api/user_vehicles/vehicles"

    "github.com/gin-gonic/gin"
)

type Dependencies struct {
    DB *sql.DB
}

func RegisterRoutes(deps Dependencies, group *gin.RouterGroup) {
    // Create repositories
    vehiclesRepository := vehicles.NewRepository(deps.DB)
    notificationsRepository := notifications.NewRepository(deps.DB)
    commentsRepository := comments.NewRepository(deps.DB)
    notificationItemsRepository := notification_items.NewRepository(deps.DB)

    // Create services
    vehiclesService := vehicles.NewService(vehiclesRepository)
    notificationsService := notifications.NewService(notificationsRepository)
    commentsService := comments.NewService(commentsRepository)
    notificationItemsService := notification_items.NewService(notificationItemsRepository)

    // Register routes for each bounded context
    vehicles.RegisterRoutes(group, vehiclesService)
    notifications.RegisterRoutes(group, notificationsService)
    comments.RegisterRoutes(group, commentsService)
    notification_items.RegisterRoutes(group, notificationItemsService)
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
1. Create `api/user_vehicles/` directory structure
2. Create `shared/context.go` with user extraction helper (can symlink or copy from `user_saves/shared`)
3. Create `shared/errors.go` with sentinel errors
4. Create `route.go` skeleton that delegates to old implementation initially

**New Files**:
```
api/user_vehicles/
├── route.go
└── shared/
    ├── context.go
    └── errors.go
```

**shared/context.go**:
```go
package shared

import (
    "errors"
    "miltechserver/bootstrap"

    "github.com/gin-gonic/gin"
)

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
```

**shared/errors.go**:
```go
package shared

import "errors"

// User errors
var (
    ErrUserNotFound = errors.New("valid user not found")
)

// Vehicle errors
var (
    ErrVehicleNotFound = errors.New("vehicle not found")
)

// Notification errors
var (
    ErrNotificationNotFound = errors.New("notification not found")
)

// Comment errors
var (
    ErrCommentNotFound = errors.New("comment not found")
)

// Item errors
var (
    ErrItemNotFound = errors.New("item not found")
)
```

---

#### Step 2: Extract Vehicles Bounded Context

**Current Location**:
- Repository: `user_vehicle_repository.go:9-15` (5 methods + 1 duplicate)
- Service: `user_vehicle_service.go:9-14` (5 methods)
- Controller: `user_vehicle_controller.go:23-144` (5 handlers)
- Routes: `user_vehicle_route.go:21-25` (5 endpoints)

**New Files**:
```
api/user_vehicles/vehicles/
├── repository.go        # ~20 lines
├── repository_impl.go   # ~150 lines
├── service.go           # ~20 lines
├── service_impl.go      # ~60 lines
└── route.go             # ~80 lines
```

**Migration Steps**:
1. Create new package with interfaces
2. Copy repository implementation from `user_vehicle_repository_impl.go:23-149`
3. Copy service implementation from `user_vehicle_service_impl.go:17-46`
4. Create route handlers using shared context helper
5. Wire up in central route registration
6. Verify endpoints work

**Route Handler Pattern**:
```go
// api/user_vehicles/vehicles/route.go
package vehicles

import (
    "log/slog"
    "miltechserver/.gen/miltech_ng/public/model"
    "miltechserver/api/response"
    "miltechserver/api/user_vehicles/shared"

    "github.com/gin-gonic/gin"
)

type Handler struct {
    service Service
}

func RegisterRoutes(router *gin.RouterGroup, service Service) {
    handler := Handler{service: service}

    router.GET("/user/vehicles", handler.getByUser)
    router.GET("/user/vehicles/:vehicleId", handler.getByID)
    router.PUT("/user/vehicles", handler.upsert)
    router.DELETE("/user/vehicles/:vehicleId", handler.delete)
    router.DELETE("/user/vehicles", handler.deleteAll)
}

func (h *Handler) getByUser(c *gin.Context) {
    user, err := shared.GetUserFromContext(c)
    if err != nil {
        c.JSON(401, gin.H{"message": "unauthorized"})
        slog.Info("Unauthorized request")
        return
    }

    result, err := h.service.GetByUser(user)
    if err != nil {
        c.JSON(404, response.EmptyResponseMessage())
        return
    }

    c.JSON(200, response.StandardResponse{
        Status:  200,
        Message: "",
        Data:    result,
    })
}

// ... additional handlers follow the same pattern
```

---

#### Step 3: Extract Notifications Bounded Context

**Current Location**:
- Repository: `user_vehicle_repository.go:17-23` (6 methods)
- Service: `user_vehicle_service.go:16-22` (6 methods)
- Controller: `user_vehicle_controller.go:146-304` (6 handlers)
- Routes: `user_vehicle_route.go:27-33` (6 endpoints)

**New Files**:
```
api/user_vehicles/notifications/
├── repository.go        # ~25 lines
├── repository_impl.go   # ~150 lines
├── service.go           # ~25 lines
├── service_impl.go      # ~80 lines
└── route.go             # ~100 lines
```

**Migration Steps**:
1. Create new package with interfaces
2. Copy repository implementation from `user_vehicle_repository_impl.go:151-280`
3. Copy service implementation from `user_vehicle_service_impl.go:48-86`
4. Create route handlers using shared context helper
5. Wire up in central route registration
6. Verify endpoints work

---

#### Step 4: Extract Comments Bounded Context

**Note**: The route file indicates these endpoints are "Not Used". Consider:
- Option A: Migrate as-is for completeness
- Option B: Skip migration, mark for deprecation
- **Recommended**: Option A - migrate for API consistency, add deprecation notice

**Current Location**:
- Repository: `user_vehicle_repository.go:25-32` (7 methods)
- Service: `user_vehicle_service.go:24-31` (7 methods)
- Controller: `user_vehicle_controller.go:306-493` (7 handlers)
- Routes: `user_vehicle_route.go:35-42` (7 endpoints)

**New Files**:
```
api/user_vehicles/comments/
├── repository.go        # ~30 lines
├── repository_impl.go   # ~180 lines
├── service.go           # ~30 lines
├── service_impl.go      # ~90 lines
└── route.go             # ~120 lines
```

---

#### Step 5: Extract Notification Items Bounded Context

**Current Location**:
- Repository: `user_vehicle_repository.go:34-41` (7 methods)
- Service: `user_vehicle_service.go:33-40` (7 methods)
- Controller: `user_vehicle_controller.go:495-680` (7 handlers)
- Routes: `user_vehicle_route.go:44-51` (7 endpoints)

**New Files**:
```
api/user_vehicles/notification_items/
├── repository.go        # ~30 lines
├── repository_impl.go   # ~180 lines
├── service.go           # ~30 lines
├── service_impl.go      # ~90 lines
└── route.go             # ~120 lines
```

**Special Consideration**: This bounded context has `UpsertBatch` for bulk operations.

---

#### Step 6: Wire Dependencies and Update Routing

**Update `api/route/route.go`**:

```go
// BEFORE (in NewRouter function)
route.NewUserVehicleRouter(db, authRoutes)

// AFTER
user_vehicles.RegisterRoutes(user_vehicles.Dependencies{DB: db}, authRoutes)
```

**Import Changes**:
```go
import (
    // ... existing imports
    "miltechserver/api/user_vehicles"
)
```

---

#### Step 7: Testing and Verification

**Manual API Testing Checklist**:

| Endpoint | Method | Expected Result |
|----------|--------|-----------------|
| `/api/v1/auth/user/vehicles` | GET | Returns user's vehicles |
| `/api/v1/auth/user/vehicles/:id` | GET | Returns specific vehicle |
| `/api/v1/auth/user/vehicles` | PUT | Creates/updates vehicle |
| `/api/v1/auth/user/vehicles/:id` | DELETE | Deletes specific vehicle |
| `/api/v1/auth/user/vehicles` | DELETE | Deletes all vehicles |
| `/api/v1/auth/user/vehicle-notifications` | GET | Returns notifications |
| `/api/v1/auth/user/vehicle-notifications/vehicle/:id` | GET | Returns vehicle's notifications |
| `/api/v1/auth/user/vehicle-notifications/:id` | GET | Returns specific notification |
| `/api/v1/auth/user/vehicle-notifications` | PUT | Creates/updates notification |
| `/api/v1/auth/user/vehicle-notifications/:id` | DELETE | Deletes notification |
| `/api/v1/auth/user/vehicle-notifications/vehicle/:id` | DELETE | Deletes vehicle's notifications |
| `/api/v1/auth/user/notification-items` | GET | Returns all items |
| `/api/v1/auth/user/notification-items/notification/:id` | GET | Returns notification's items |
| `/api/v1/auth/user/notification-items/:id` | GET | Returns specific item |
| `/api/v1/auth/user/notification-items` | PUT | Creates/updates item |
| `/api/v1/auth/user/notification-items/list` | PUT | Batch upsert items |
| `/api/v1/auth/user/notification-items/:id` | DELETE | Deletes item |
| `/api/v1/auth/user/notification-items/notification/:id` | DELETE | Deletes notification's items |

**Unit Test Requirements**:
- Service layer tests for each bounded context
- Mock repository for isolated testing
- Edge cases: nil user, not found, empty results

---

#### Step 8: Cleanup Legacy Files

**Files to Remove**:
```
api/controller/user_vehicle_controller.go
api/service/user_vehicle_service.go
api/service/user_vehicle_service_impl.go
api/repository/user_vehicle_repository.go
api/repository/user_vehicle_repository_impl.go
api/route/user_vehicle_route.go
```

**Update Imports**: Search and replace all imports pointing to old packages

---

## API Backward Compatibility

### URL Structure Preservation

All existing URLs **MUST** remain unchanged:

| Current URL | New Handler Location |
|-------------|---------------------|
| `GET /api/v1/auth/user/vehicles` | `user_vehicles/vehicles/route.go` |
| `GET /api/v1/auth/user/vehicles/:vehicleId` | `user_vehicles/vehicles/route.go` |
| `PUT /api/v1/auth/user/vehicles` | `user_vehicles/vehicles/route.go` |
| `DELETE /api/v1/auth/user/vehicles/:vehicleId` | `user_vehicles/vehicles/route.go` |
| `DELETE /api/v1/auth/user/vehicles` | `user_vehicles/vehicles/route.go` |
| `GET /api/v1/auth/user/vehicle-notifications` | `user_vehicles/notifications/route.go` |
| `GET /api/v1/auth/user/vehicle-notifications/vehicle/:vehicleId` | `user_vehicles/notifications/route.go` |
| `GET /api/v1/auth/user/vehicle-notifications/:notificationId` | `user_vehicles/notifications/route.go` |
| `PUT /api/v1/auth/user/vehicle-notifications` | `user_vehicles/notifications/route.go` |
| `DELETE /api/v1/auth/user/vehicle-notifications/:notificationId` | `user_vehicles/notifications/route.go` |
| `DELETE /api/v1/auth/user/vehicle-notifications/vehicle/:vehicleId` | `user_vehicles/notifications/route.go` |
| `GET /api/v1/auth/user/vehicle-comments` | `user_vehicles/comments/route.go` |
| `GET /api/v1/auth/user/vehicle-comments/vehicle/:vehicleId` | `user_vehicles/comments/route.go` |
| `GET /api/v1/auth/user/vehicle-comments/notification/:notificationId` | `user_vehicles/comments/route.go` |
| `GET /api/v1/auth/user/vehicle-comments/:commentId` | `user_vehicles/comments/route.go` |
| `PUT /api/v1/auth/user/vehicle-comments` | `user_vehicles/comments/route.go` |
| `DELETE /api/v1/auth/user/vehicle-comments/:commentId` | `user_vehicles/comments/route.go` |
| `DELETE /api/v1/auth/user/vehicle-comments/vehicle/:vehicleId` | `user_vehicles/comments/route.go` |
| `GET /api/v1/auth/user/notification-items` | `user_vehicles/notification_items/route.go` |
| `GET /api/v1/auth/user/notification-items/notification/:notificationId` | `user_vehicles/notification_items/route.go` |
| `GET /api/v1/auth/user/notification-items/:itemId` | `user_vehicles/notification_items/route.go` |
| `PUT /api/v1/auth/user/notification-items` | `user_vehicles/notification_items/route.go` |
| `PUT /api/v1/auth/user/notification-items/list` | `user_vehicles/notification_items/route.go` |
| `DELETE /api/v1/auth/user/notification-items/:itemId` | `user_vehicles/notification_items/route.go` |
| `DELETE /api/v1/auth/user/notification-items/notification/:notificationId` | `user_vehicles/notification_items/route.go` |

---

## Risk Mitigation

### 1. Rollback Plan

Each step can be rolled back independently:

1. **Git branches**: Each step in separate commit
2. **Database**: No schema changes required
3. **Route switching**: Can revert to old `NewUserVehicleRouter` instantly

### 2. Gradual Rollout

1. Deploy with new structure
2. Monitor for errors in logs
3. Verify all endpoints via API testing
4. Cleanup legacy files only after verification

### 3. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Route path changes break clients | Low | High | Keep exact same paths |
| Missing functionality | Medium | Medium | Comprehensive API testing checklist |
| Import cycles | Low | Low | Clean package boundaries |
| Merge conflicts | Low | Medium | Complete before merging shop_refactor |

---

## Success Metrics

### Code Quality

| Metric | Current | Target |
|--------|---------|--------|
| Largest file | 681 lines | < 150 lines |
| Methods per interface | 25 | < 8 |
| Test coverage | 0% | > 80% |
| Domain errors | 0 typed | 5 typed |

### Maintainability

- **Time to understand a context**: 45 minutes → 10 minutes
- **Time to add a new feature**: 2 hours → 30 minutes
- **Number of files changed for a bug fix**: 3+ → 1-2

### Performance

- **No regression** in API response times
- **Same memory footprint**
- **Same database query patterns**

---

## Estimated Effort

| Step | Description | Estimated Effort |
|------|-------------|------------------|
| 1 | Create directory structure and shared package | 1 hour |
| 2 | Extract vehicles bounded context | 2 hours |
| 3 | Extract notifications bounded context | 2 hours |
| 4 | Extract comments bounded context | 2 hours |
| 5 | Extract notification_items bounded context | 2 hours |
| 6 | Wire dependencies and update routing | 1 hour |
| 7 | Testing and verification | 2 hours |
| 8 | Cleanup legacy files | 30 minutes |
| **TOTAL** | | **~12-13 hours** |

---

## Comparison with Previous Refactors

| Aspect | Shops Refactor | User Saves Refactor | User Vehicles Refactor |
|--------|---------------|---------------------|------------------------|
| Bounded contexts | 8 | 5 | 4 |
| Facade pattern | Yes | Yes | No (simpler domain) |
| Shared auth | `shared/authorization.go` | `shared/context.go` | `shared/context.go` |
| Domain errors | `shared/errors.go` | `shared/errors.go` | `shared/errors.go` |
| Central wiring | `route.go` | `route.go` | `route.go` |
| Cross-cutting concerns | Blob storage | Blob storage | None |
| Complexity | High | Medium | Low-Medium |

---

## Appendix A: File Line Counts

### Current State

```
$ wc -l api/repository/user_vehicle_repository_impl.go
     594 api/repository/user_vehicle_repository_impl.go

$ wc -l api/controller/user_vehicle_controller.go
     681 api/controller/user_vehicle_controller.go

$ wc -l api/service/user_vehicle_service_impl.go
     181 api/service/user_vehicle_service_impl.go

$ wc -l api/repository/user_vehicle_repository.go api/service/user_vehicle_service.go api/route/user_vehicle_route.go
      42 api/repository/user_vehicle_repository.go
      41 api/service/user_vehicle_service.go
      52 api/route/user_vehicle_route.go

TOTAL: ~1,591 lines
```

### Target State

```
api/user_vehicles/
├── route.go              (~40 lines)
├── shared/               (~50 lines total)
├── vehicles/             (~250 lines total)
├── notifications/        (~300 lines total)
├── comments/             (~350 lines total)
└── notification_items/   (~350 lines total)

TOTAL: ~1,340 lines (16% reduction through deduplication)
       + Unit test coverage
       + 5 typed domain errors
       + Clean separation of concerns
```

---

## Appendix B: Database Tables

| Package | Tables Accessed | Primary Key |
|---------|-----------------|-------------|
| `user_vehicles/vehicles` | `user_vehicle` | `id` |
| `user_vehicles/notifications` | `user_vehicle_notifications` | `id` |
| `user_vehicles/comments` | `user_vehicle_comments` | `id`, `user_id` (composite) |
| `user_vehicles/notification_items` | `user_notification_items` | `id` |

---

*Document Version: 1.0*
*Created: January 29, 2026*
*Author: Architecture Review*
*Based on: User Saves Domain Refactoring Pattern*
