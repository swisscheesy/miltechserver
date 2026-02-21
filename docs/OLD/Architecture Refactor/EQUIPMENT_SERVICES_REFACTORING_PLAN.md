# Equipment Services Refactoring Plan

**Created:** 2026-01-30
**Status:** Planning
**Priority:** High
**Estimated Complexity:** Large (1,126+ LOC across repository and service)

---

## Executive Summary

This document outlines the plan to refactor the `equipment_services` domain from a monolithic structure into bounded contexts, following the patterns established in the `shops`, `user_saves`, `user_vehicles`, and `material_images` refactoring efforts.

The equipment_services domain currently manages maintenance/service scheduling for shop equipment (vehicles), with functionality spanning CRUD operations, query/filtering, calendar views, status tracking (overdue/due-soon), and completion workflows.

---

## Current State Analysis

### File Inventory

| File | Lines | Methods | Responsibility |
|------|-------|---------|----------------|
| [equipment_services_repository.go](../api/repository/equipment_services_repository.go) | 34 | 14 | Interface definition |
| [equipment_services_repository_impl.go](../api/repository/equipment_services_repository_impl.go) | 575 | 14 | Data access layer |
| [equipment_services_service.go](../api/service/equipment_services_service.go) | 24 | 10 | Service interface |
| [equipment_services_service_impl.go](../api/service/equipment_services_service_impl.go) | 553 | 12 | Business logic |
| [equipment_services_controller.go](../api/controller/equipment_services_controller.go) | 424 | 11 | HTTP handlers |
| [equipment_services_request.go](../api/request/equipment_services_request.go) | 66 | - | Request DTOs |
| [equipment_services_response.go](../api/response/equipment_services_response.go) | 62 | - | Response DTOs |
| **Total** | **~1,738** | **~47** | |

### Current Repository Interface (14 methods)

```go
type EquipmentServicesRepository interface {
    // CRUD Operations (5)
    CreateEquipmentService(...)
    GetEquipmentServiceByID(...)
    UpdateEquipmentService(...)
    DeleteEquipmentService(...)
    CompleteEquipmentService(...)

    // Query Operations (5)
    GetEquipmentServices(...)
    GetServicesByEquipment(...)
    GetServicesInDateRange(...)
    GetOverdueServices(...)
    GetServicesDueSoon(...)

    // Validation Helpers (4)
    ValidateServiceOwnership(...)
    ValidateServiceAccess(...)
    ValidateEquipmentAccess(...)
    ValidateListAccess(...)

    // Username Lookup (1)
    GetUsernameByUserID(...)
}
```

### Current Service Interface (10 methods)

```go
type EquipmentServicesService interface {
    // CRUD Operations (5)
    CreateEquipmentService(...)
    GetEquipmentServiceByID(...)
    UpdateEquipmentService(...)
    DeleteEquipmentService(...)
    CompleteEquipmentService(...)

    // Query Operations (5)
    GetEquipmentServices(...)
    GetServicesByEquipment(...)
    GetServicesInDateRange(...)
    GetOverdueServices(...)
    GetServicesDueSoon(...)
}
```

### Identified Issues

1. **Response Mapping Duplication** - The same `EquipmentServiceResponse` mapping code is repeated 8 times across the service layer
2. **N+1 Query Pattern** - Each response construction makes a separate `GetUsernameByUserID` call
3. **Authorization Mixed with Business Logic** - Shop membership checks scattered throughout methods
4. **Validation in Wrong Layer** - Ownership/access validation belongs in shared authorization, not repository
5. **Large Interface Surface** - 14 repository methods exceeds target of <7 per interface
6. **God Object Anti-pattern** - Single repository handles too many distinct responsibilities

---

## Target Architecture

### Directory Structure

```
api/equipment_services/
├── route.go                           # Central route registration & DI
├── shared/
│   ├── context.go                     # Request context helpers
│   ├── errors.go                      # Domain-specific typed errors
│   ├── authorization.go               # Shared authorization logic
│   └── mappers.go                     # Response mapping utilities
├── core/
│   ├── repository.go                  # CRUD operations interface (~4 methods)
│   ├── repository_impl.go             # Create, Read, Update, Delete
│   ├── service.go                     # Core service interface
│   ├── service_impl.go                # Business logic for CRUD
│   └── route.go                       # Core CRUD routes
├── queries/
│   ├── repository.go                  # Query operations interface (~3 methods)
│   ├── repository_impl.go             # GetEquipmentServices, GetServicesByEquipment
│   ├── service.go                     # Query service interface
│   ├── service_impl.go                # Filtering, pagination logic
│   └── route.go                       # Query routes
├── calendar/
│   ├── repository.go                  # Date-range queries interface (~1 method)
│   ├── repository_impl.go             # GetServicesInDateRange
│   ├── service.go                     # Calendar service interface
│   ├── service_impl.go                # Date parsing, calendar logic
│   └── route.go                       # /calendar endpoint
├── status/
│   ├── repository.go                  # Status queries interface (~2 methods)
│   ├── repository_impl.go             # GetOverdueServices, GetServicesDueSoon
│   ├── service.go                     # Status service interface
│   ├── service_impl.go                # Overdue/due-soon business logic
│   └── route.go                       # /overdue, /due-soon endpoints
└── completion/
    ├── repository.go                  # Completion interface (~1 method)
    ├── repository_impl.go             # CompleteEquipmentService
    ├── service.go                     # Completion service interface
    ├── service_impl.go                # Completion workflow logic
    └── route.go                       # /complete endpoint
```

### Bounded Context Breakdown

#### 1. `shared/` - Cross-Cutting Concerns

**Purpose:** Centralize authorization, error handling, and response mapping

**Files:**
- `context.go` - Request context extraction helpers
- `errors.go` - Typed domain errors
- `authorization.go` - Shop membership/ownership validation
- `mappers.go` - Centralized `model.EquipmentServices` to `response.EquipmentServiceResponse` conversion

**Key Responsibilities:**
- Eliminate 8x duplicated response mapping code
- Centralize username lookup with batching capability
- Provide reusable authorization checks

```go
// shared/errors.go
package shared

import "errors"

var (
    ErrServiceNotFound     = errors.New("equipment service not found")
    ErrUnauthorized        = errors.New("unauthorized")
    ErrAccessDenied        = errors.New("access denied: user is not a member of this shop")
    ErrModifyDenied        = errors.New("access denied: only service creators or shop admins can modify")
    ErrDeleteDenied        = errors.New("access denied: only service creators or shop admins can delete")
    ErrInvalidServiceHours = errors.New("service_hours must be non-negative")
    ErrInvalidDateFormat   = errors.New("invalid date format, expected RFC3339")
    ErrEquipmentNotFound   = errors.New("equipment not found or access denied")
    ErrListNotFound        = errors.New("list not found or access denied")
    ErrShopMismatch        = errors.New("equipment and list must belong to the same shop")
)
```

```go
// shared/mappers.go
package shared

import (
    "miltechserver/.gen/miltech_ng/public/model"
    "miltechserver/api/response"
)

type UsernameResolver interface {
    GetUsernameByUserID(userID string) (string, error)
}

func MapServiceToResponse(svc model.EquipmentServices, username string) response.EquipmentServiceResponse {
    return response.EquipmentServiceResponse{
        ID:                svc.ID,
        ShopID:            svc.ShopID,
        EquipmentID:       svc.EquipmentID,
        ListID:            svc.ListID,
        Description:       svc.Description,
        ServiceType:       svc.ServiceType,
        CreatedBy:         svc.CreatedBy,
        CreatedByUsername: username,
        IsCompleted:       svc.IsCompleted,
        CreatedAt:         svc.CreatedAt,
        UpdatedAt:         svc.UpdatedAt,
        ServiceDate:       svc.ServiceDate,
        ServiceHours:      svc.ServiceHours,
        CompletionDate:    svc.CompletionDate,
    }
}

func MapServicesToResponses(services []model.EquipmentServices, resolver UsernameResolver) []response.EquipmentServiceResponse {
    responses := make([]response.EquipmentServiceResponse, len(services))
    for i, svc := range services {
        username, _ := resolver.GetUsernameByUserID(svc.CreatedBy)
        if username == "" {
            username = "Unknown User"
        }
        responses[i] = MapServiceToResponse(svc, username)
    }
    return responses
}
```

#### 2. `core/` - CRUD Operations

**Purpose:** Basic Create, Read, Update, Delete operations

**Repository Interface (~4 methods):**
```go
type Repository interface {
    Create(user *bootstrap.User, service model.EquipmentServices) (*model.EquipmentServices, error)
    GetByID(user *bootstrap.User, serviceID string) (*model.EquipmentServices, error)
    Update(user *bootstrap.User, service model.EquipmentServices) (*model.EquipmentServices, error)
    Delete(user *bootstrap.User, serviceID string) error
}
```

**Service Interface:**
```go
type Service interface {
    Create(user *bootstrap.User, req request.CreateEquipmentServiceRequest) (*response.EquipmentServiceResponse, error)
    GetByID(user *bootstrap.User, shopID, serviceID string) (*response.EquipmentServiceResponse, error)
    Update(user *bootstrap.User, shopID string, req request.UpdateEquipmentServiceRequest) (*response.EquipmentServiceResponse, error)
    Delete(user *bootstrap.User, shopID, serviceID string) error
}
```

**Routes:**
- `POST /shops/:shop_id/equipment-services` - Create service
- `GET /shops/:shop_id/equipment-services/:service_id` - Get service by ID
- `PUT /shops/:shop_id/equipment-services/:service_id` - Update service
- `DELETE /shops/:shop_id/equipment-services/:service_id` - Delete service

#### 3. `queries/` - List & Filter Operations

**Purpose:** Paginated queries with filtering

**Repository Interface (~3 methods):**
```go
type Repository interface {
    GetByShop(user *bootstrap.User, shopID string, filters request.GetEquipmentServicesRequest) ([]model.EquipmentServices, int64, error)
    GetByEquipment(user *bootstrap.User, equipmentID string, limit, offset int, startDate, endDate *time.Time) ([]model.EquipmentServices, int64, error)
    GetUsernameByUserID(userID string) (string, error)
}
```

**Service Interface:**
```go
type Service interface {
    GetByShop(user *bootstrap.User, shopID string, req request.GetEquipmentServicesRequest) (*response.PaginatedEquipmentServicesResponse, error)
    GetByEquipment(user *bootstrap.User, equipmentID string, limit, offset int, startDate, endDate *time.Time) (*response.PaginatedEquipmentServicesResponse, error)
}
```

**Routes:**
- `GET /shops/:shop_id/equipment-services` - List services with filters
- `GET /equipment/:equipment_id/services` - Services by equipment

#### 4. `calendar/` - Date Range Operations

**Purpose:** Calendar view functionality

**Repository Interface (~1 method):**
```go
type Repository interface {
    GetInDateRange(user *bootstrap.User, shopID string, startDate, endDate time.Time, equipmentID *string) ([]model.EquipmentServices, error)
}
```

**Service Interface:**
```go
type Service interface {
    GetCalendarServices(user *bootstrap.User, shopID string, req request.GetCalendarServicesRequest) (*response.CalendarServicesResponse, error)
}
```

**Routes:**
- `GET /shops/:shop_id/equipment-services/calendar` - Calendar view

#### 5. `status/` - Overdue & Due Soon Tracking

**Purpose:** Service status monitoring

**Repository Interface (~2 methods):**
```go
type Repository interface {
    GetOverdue(user *bootstrap.User, shopID string, equipmentID *string, limit int) ([]ServiceWithDays, error)
    GetDueSoon(user *bootstrap.User, shopID string, daysAhead int, equipmentID *string, limit int) ([]ServiceWithDays, error)
}

type ServiceWithDays struct {
    model.EquipmentServices
    DaysCount int // DaysOverdue or DaysUntilDue
}
```

**Service Interface:**
```go
type Service interface {
    GetOverdue(user *bootstrap.User, shopID string, req request.GetOverdueServicesRequest) (*response.OverdueServicesResponse, error)
    GetDueSoon(user *bootstrap.User, shopID string, req request.GetDueSoonServicesRequest) (*response.DueSoonServicesResponse, error)
}
```

**Routes:**
- `GET /shops/:shop_id/equipment-services/overdue` - Overdue services
- `GET /shops/:shop_id/equipment-services/due-soon` - Due soon services

#### 6. `completion/` - Service Completion Workflow

**Purpose:** Mark services as completed

**Repository Interface (~1 method):**
```go
type Repository interface {
    Complete(user *bootstrap.User, serviceID string, completionDate *time.Time) (*model.EquipmentServices, error)
}
```

**Service Interface:**
```go
type Service interface {
    Complete(user *bootstrap.User, shopID, serviceID string, req request.CompleteEquipmentServiceRequest) (*response.EquipmentServiceResponse, error)
}
```

**Routes:**
- `POST /shops/:shop_id/equipment-services/:service_id/complete` - Complete service

---

## Implementation Steps

### Phase 1: Setup Foundation

| Step | Description | Files Created |
|------|-------------|---------------|
| 1.1 | Create directory structure | `api/equipment_services/` tree |
| 1.2 | Create shared errors | `shared/errors.go` |
| 1.3 | Create shared context helpers | `shared/context.go` |
| 1.4 | Create shared authorization | `shared/authorization.go` |
| 1.5 | Create shared mappers | `shared/mappers.go` |
| 1.6 | Create central route.go skeleton | `route.go` |

### Phase 2: Extract Bounded Contexts

| Step | Description | Files Created |
|------|-------------|---------------|
| 2.1 | Extract core CRUD context | `core/*.go` |
| 2.2 | Extract queries context | `queries/*.go` |
| 2.3 | Extract calendar context | `calendar/*.go` |
| 2.4 | Extract status context | `status/*.go` |
| 2.5 | Extract completion context | `completion/*.go` |

### Phase 3: Wire Dependencies

| Step | Description | Files Modified |
|------|-------------|----------------|
| 3.1 | Wire all contexts in central route.go | `route.go` |
| 3.2 | Update main route registration | `api/route/route.go` |
| 3.3 | Retain legacy controller during transition | (keep existing) |

### Phase 4: Verification & Cleanup

| Step | Description | Notes |
|------|-------------|-------|
| 4.1 | Create integration tests | `tests/equipment_services/` |
| 4.2 | Manual API testing | Verify all endpoints |
| 4.3 | Remove legacy files | After verification |

---

## Central Route.go Pattern

Following the established pattern from `material_images/route.go`:

```go
package equipment_services

import (
    "database/sql"

    "github.com/gin-gonic/gin"

    "miltechserver/api/equipment_services/calendar"
    "miltechserver/api/equipment_services/completion"
    "miltechserver/api/equipment_services/core"
    "miltechserver/api/equipment_services/queries"
    "miltechserver/api/equipment_services/shared"
    "miltechserver/api/equipment_services/status"
    shopsShared "miltechserver/api/shops/shared"
)

type Dependencies struct {
    DB *sql.DB
}

func RegisterRoutes(deps Dependencies, router *gin.RouterGroup) {
    // Shared dependencies
    shopAuth := shopsShared.NewShopAuthorization(deps.DB)
    authorization := shared.NewAuthorization(deps.DB, shopAuth)

    // Repositories
    coreRepo := core.NewRepository(deps.DB)
    queriesRepo := queries.NewRepository(deps.DB)
    calendarRepo := calendar.NewRepository(deps.DB)
    statusRepo := status.NewRepository(deps.DB)
    completionRepo := completion.NewRepository(deps.DB)

    // Services
    coreService := core.NewService(coreRepo, authorization)
    queriesService := queries.NewService(queriesRepo, shopAuth)
    calendarService := calendar.NewService(calendarRepo, shopAuth)
    statusService := status.NewService(statusRepo, shopAuth)
    completionService := completion.NewService(completionRepo, authorization)

    // Register routes for each context
    core.RegisterRoutes(router, coreService)
    queries.RegisterRoutes(router, queriesService)
    calendar.RegisterRoutes(router, calendarService)
    status.RegisterRoutes(router, statusService)
    completion.RegisterRoutes(router, completionService)
}
```

---

## Migration Strategy

### Backward Compatibility

1. **Keep legacy controller during transition** - Allow both old and new routes to coexist
2. **Use the same API contract** - No changes to request/response DTOs
3. **Feature flag option** - Environment variable to toggle new vs legacy routes

### Cutover Plan

1. Deploy new bounded context routes alongside legacy
2. Update API consumers to use new routes (if paths change)
3. Monitor for errors in new implementation
4. Remove legacy files after 1-2 sprint verification period

---

## Metrics & Targets

| Metric | Current | Target | Notes |
|--------|---------|--------|-------|
| Largest file (LOC) | 575 | < 200 | Repository impl is largest |
| Methods per interface | 14 (repo) | < 7 | Target <=5 per bounded context |
| Total interfaces | 2 | 12 | 2 per context (repo + service) |
| Code duplication | 8 mappings | 1 shared | Centralize in mappers.go |
| Test coverage | 0% | > 80% | Add tests per context |
| Cyclomatic complexity | High | Low | Single-responsibility methods |

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Breaking existing API | Low | High | Keep same request/response DTOs |
| Authorization gaps | Medium | High | Comprehensive test coverage |
| Performance regression | Low | Medium | Monitor query performance |
| Incomplete migration | Medium | Low | Keep legacy until verified |

---

## Dependencies

### External Dependencies
- `miltechserver/api/shops/shared` - ShopAuthorization interface
- `miltechserver/.gen/miltech_ng/public/model` - Generated Jet models
- `miltechserver/api/request` - Request DTOs (keep existing)
- `miltechserver/api/response` - Response DTOs (keep existing)

### Internal Dependencies (between bounded contexts)
- `completion/` depends on `shared/authorization`
- `core/` depends on `shared/authorization`, `shared/mappers`
- All contexts depend on `shared/errors`

---

## Status Tracker

| Step | Status | Notes |
|------|--------|-------|
| 1.1 Create directory structure | Pending | |
| 1.2 Create shared/errors.go | Pending | |
| 1.3 Create shared/context.go | Pending | |
| 1.4 Create shared/authorization.go | Pending | |
| 1.5 Create shared/mappers.go | Pending | |
| 1.6 Create central route.go skeleton | Pending | |
| 2.1 Extract core CRUD context | Pending | |
| 2.2 Extract queries context | Pending | |
| 2.3 Extract calendar context | Pending | |
| 2.4 Extract status context | Pending | |
| 2.5 Extract completion context | Pending | |
| 3.1 Wire dependencies in route.go | Pending | |
| 3.2 Update main route registration | Pending | |
| 4.1 Create integration tests | Pending | |
| 4.2 Manual API testing | Pending | |
| 4.3 Remove legacy files | Pending | |

---

## Appendix A: Current API Endpoints

| Method | Path | Handler | Context |
|--------|------|---------|---------|
| POST | `/shops/:shop_id/equipment-services` | CreateEquipmentService | core |
| GET | `/shops/:shop_id/equipment-services` | GetEquipmentServices | queries |
| GET | `/shops/:shop_id/equipment-services/:service_id` | GetEquipmentServiceByID | core |
| PUT | `/shops/:shop_id/equipment-services/:service_id` | UpdateEquipmentService | core |
| DELETE | `/shops/:shop_id/equipment-services/:service_id` | DeleteEquipmentService | core |
| GET | `/equipment/:equipment_id/services` | GetServicesByEquipment | queries |
| GET | `/shops/:shop_id/equipment-services/calendar` | GetServicesInDateRange | calendar |
| GET | `/shops/:shop_id/equipment-services/overdue` | GetOverdueServices | status |
| GET | `/shops/:shop_id/equipment-services/due-soon` | GetServicesDueSoon | status |
| POST | `/shops/:shop_id/equipment-services/:service_id/complete` | CompleteEquipmentService | completion |

---

## Appendix B: Database Tables

The equipment_services domain interacts with these tables:

- `equipment_services` - Primary table for service records
- `shop_members` - For authorization (shop membership)
- `shop_vehicle` - Equipment (vehicles) linked to services
- `shop_lists` - Optional list association
- `users` - Username lookup

---

## Next Steps

1. Review and approve this plan
2. Begin Phase 1 implementation
3. Create tracking issue for progress
