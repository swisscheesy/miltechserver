# Shops Domain Refactoring Plan

## Executive Summary

The shops domain has grown into a **monolithic structure** spanning ~6,745 lines across three files:

| File | Lines | Responsibility |
|------|-------|----------------|
| `api/repository/shops_repository_impl.go` | 2,301 | Data access for all shop-related entities |
| `api/service/shops_service_impl.go` | 2,469 | Business logic for all shop operations |
| `api/controller/shops_controller.go` | 1,975 | HTTP handlers for 50+ endpoints |

This plan outlines how to decompose these god objects into **focused bounded contexts** while maintaining backward compatibility with existing clients.

---

## Current Domain Analysis

### Identified Bounded Contexts

Based on the interface analysis from `api/service/shops_service.go` (lines 10-93), the shops domain contains **8 distinct sub-domains**:

```
shops/
├── core/           # Shop CRUD, settings
├── members/        # Membership, invites, roles
├── messages/       # Chat/messaging with images
├── vehicles/       # Vehicle management
├── notifications/  # Vehicle notifications (maintenance alerts)
├── notification-items/  # Items within notifications
├── lists/          # Shopping/parts lists
└── list-items/     # Items within lists
```

### Current Interface Method Count by Context

| Context | Service Methods | Repository Methods | Endpoints |
|---------|-----------------|-------------------|-----------|
| Core Shop | 6 | 7 | 10 |
| Members | 5 | 6 | 5 |
| Invite Codes | 4 | 5 | 4 |
| Messages | 7 | 11 | 8 |
| Vehicles | 5 | 5 | 5 |
| Notifications | 7 | 7 | 7 |
| Notification Items | 6 | 7 | 6 |
| Lists | 5 | 5 | 5 |
| List Items | 6 | 7 | 7 |
| Settings | 4 | 4 | 4 |
| Change Tracking | 3 | 3 | 3 |
| **TOTAL** | **58** | **67** | **64** |

---

## Target Architecture

### Phase 1: Package Structure

```
api/
├── shops/
│   ├── core/
│   │   ├── controller.go      # Shop CRUD endpoints
│   │   ├── service.go         # Interface
│   │   ├── service_impl.go    # Business logic
│   │   ├── repository.go      # Interface
│   │   └── repository_impl.go # Data access
│   │
│   ├── members/
│   │   ├── controller.go      # Join, leave, promote, remove
│   │   ├── service.go
│   │   ├── service_impl.go
│   │   ├── repository.go
│   │   ├── repository_impl.go
│   │   └── invites/           # Sub-package for invite codes
│   │       ├── service.go
│   │       └── repository.go
│   │
│   ├── messages/
│   │   ├── controller.go      # Chat messages with image upload
│   │   ├── service.go
│   │   ├── service_impl.go
│   │   ├── repository.go
│   │   └── repository_impl.go
│   │
│   ├── vehicles/
│   │   ├── controller.go      # Vehicle CRUD
│   │   ├── service.go
│   │   ├── service_impl.go
│   │   ├── repository.go
│   │   ├── repository_impl.go
│   │   └── notifications/     # Vehicle notifications sub-package
│   │       ├── controller.go
│   │       ├── service.go
│   │       ├── service_impl.go
│   │       ├── repository.go
│   │       ├── repository_impl.go
│   │       └── items/         # Notification items
│   │           ├── service.go
│   │           └── repository.go
│   │
│   ├── lists/
│   │   ├── controller.go      # List CRUD
│   │   ├── service.go
│   │   ├── service_impl.go
│   │   ├── repository.go
│   │   ├── repository_impl.go
│   │   └── items/             # List items sub-package
│   │       ├── service.go
│   │       └── repository.go
│   │
│   ├── settings/
│   │   ├── controller.go      # Shop settings management
│   │   ├── service.go
│   │   └── repository.go
│   │
│   ├── shared/                # Shared types and utilities
│   │   ├── authorization.go   # Centralized auth checks
│   │   ├── errors.go          # Sentinel errors
│   │   └── context.go         # Shop context helpers
│   │
│   └── route.go               # Unified route registration
```

### Phase 2: Shared Authorization Layer

Create a centralized authorization service that all sub-domains can use:

```go
// api/shops/shared/authorization.go
package shared

import "miltechserver/bootstrap"

type ShopAuthorization interface {
    // Core permission checks
    IsUserMemberOfShop(user *bootstrap.User, shopID string) (bool, error)
    IsUserShopAdmin(user *bootstrap.User, shopID string) (bool, error)
    GetUserRoleInShop(user *bootstrap.User, shopID string) (string, error)

    // Ownership checks for specific resources
    CanUserModifyVehicle(user *bootstrap.User, vehicleID string) (bool, error)
    CanUserModifyList(user *bootstrap.User, listID string) (bool, error)
    CanUserModifyNotification(user *bootstrap.User, notificationID string) (bool, error)

    // Enforce - panics/returns error if not authorized
    RequireShopMember(user *bootstrap.User, shopID string) error
    RequireShopAdmin(user *bootstrap.User, shopID string) error
}
```

### Phase 3: Sentinel Errors

```go
// api/shops/shared/errors.go
package shared

import "errors"

// Shop errors
var (
    ErrShopNotFound       = errors.New("shop not found")
    ErrShopAccessDenied   = errors.New("access denied: not a member of this shop")
    ErrShopAdminRequired  = errors.New("access denied: admin privileges required")
    ErrShopCreatorOnly    = errors.New("access denied: only shop creator can perform this action")
)

// Member errors
var (
    ErrMemberNotFound     = errors.New("member not found")
    ErrAlreadyMember      = errors.New("user is already a member of this shop")
    ErrCannotRemoveSelf   = errors.New("cannot remove yourself from shop")
    ErrCannotRemoveCreator = errors.New("cannot remove shop creator")
)

// Invite errors
var (
    ErrInviteCodeInvalid  = errors.New("invalid invite code")
    ErrInviteCodeExpired  = errors.New("invite code has expired")
    ErrInviteCodeUsed     = errors.New("invite code has already been used")
)

// Vehicle errors
var (
    ErrVehicleNotFound    = errors.New("vehicle not found")
    ErrVehicleAccessDenied = errors.New("access denied to vehicle")
)

// List errors
var (
    ErrListNotFound       = errors.New("list not found")
    ErrListAccessDenied   = errors.New("access denied to list")
    ErrAdminOnlyLists     = errors.New("only admins can create lists in this shop")
)

// Notification errors
var (
    ErrNotificationNotFound = errors.New("notification not found")
)
```

---

## Migration Strategy

### Approach: Strangler Fig Pattern

The **Strangler Fig Pattern** allows incremental migration without breaking existing clients:

1. **Wrap**: New code calls old implementation initially
2. **Replace**: Gradually move logic to new packages
3. **Remove**: Delete old code when fully migrated

### Phase 1: Foundation (Week 1-2)

**Goal**: Create shared infrastructure without changing behavior

#### Step 1.1: Create Shared Package
```
api/shops/shared/
├── authorization.go    # Extract from shops_repository_impl.go
├── errors.go           # New sentinel errors
└── context.go          # User/shop context helpers
```

**Files to Extract From**:
- `IsUserShopAdmin` from `shops_repository_impl.go`
- `IsUserMemberOfShop` from `shops_repository_impl.go`
- `GetUserRoleInShop` from `shops_repository_impl.go`

#### Step 1.2: Create Wrapper Interfaces

Create new interfaces that delegate to the existing implementation:

```go
// api/shops/core/service.go
package core

type ShopService interface {
    CreateShop(user *bootstrap.User, shop model.Shops) (*model.Shops, error)
    UpdateShop(user *bootstrap.User, shop model.Shops) (*model.Shops, error)
    DeleteShop(user *bootstrap.User, shopID string) error
    GetShopsByUser(user *bootstrap.User) ([]model.Shops, error)
    GetShopByID(user *bootstrap.User, shopID string) (*response.ShopDetailResponse, error)
    GetUserDataWithShops(user *bootstrap.User) (*response.UserShopsResponse, error)
}

// Initially delegates to existing ShopsService
type shopServiceWrapper struct {
    legacy service.ShopsService
}
```

### Phase 2: Extract Bounded Contexts (Week 3-6)

**Priority Order** (based on complexity and risk):

| Priority | Context | Risk Level | Complexity | Dependencies |
|----------|---------|------------|------------|--------------|
| 1 | Settings | Low | Low | Core only |
| 2 | Invite Codes | Low | Low | Members |
| 3 | Members | Medium | Medium | Core |
| 4 | Lists + Items | Medium | Medium | Core, Members |
| 5 | Messages | Medium | High | Core, Members, Blob Storage |
| 6 | Vehicles | Medium | Medium | Core, Members |
| 7 | Notifications + Items | High | High | Vehicles, Core, Members |
| 8 | Core | High | High | None (foundation) |

#### Step 2.1: Extract Settings (Low Risk)

**Current Location**:
- `shops_service.go:81-88`
- `shops_repository.go:97-103`

**New Files**:
```
api/shops/settings/
├── service.go           # 4 methods
├── service_impl.go      # ~100 lines
├── repository.go        # 4 methods
└── repository_impl.go   # ~80 lines
```

**Migration Steps**:
1. Create new package with interfaces
2. Copy implementation from `shops_*_impl.go`
3. Update `shops_service.go` to delegate to new service
4. Add tests for new package
5. Remove code from monolith

#### Step 2.2: Extract Members + Invites (Medium Risk)

**Current Location**:
- Service: `shops_service.go:20-30`
- Repository: `shops_repository.go:21-37`

**New Files**:
```
api/shops/members/
├── controller.go        # 5 endpoints
├── service.go           # 5 methods
├── service_impl.go      # ~200 lines
├── repository.go        # 6 methods
├── repository_impl.go   # ~250 lines
└── invites/
    ├── service.go       # 4 methods
    ├── service_impl.go  # ~150 lines
    ├── repository.go    # 5 methods
    └── repository_impl.go # ~200 lines
```

#### Step 2.3: Extract Lists Domain (Medium Risk)

**Current Location**:
- Service: `shops_service.go:66-78`
- Repository: `shops_repository.go:78-92`

**New Files**:
```
api/shops/lists/
├── controller.go        # 12 endpoints (list + items)
├── service.go           # 5 methods
├── service_impl.go      # ~200 lines
├── repository.go        # 5 methods
├── repository_impl.go   # ~300 lines
└── items/
    ├── service.go       # 6 methods
    ├── service_impl.go  # ~180 lines
    ├── repository.go    # 7 methods
    └── repository_impl.go # ~250 lines
```

#### Step 2.4: Extract Messages Domain (Medium-High Risk)

**Current Location**:
- Service: `shops_service.go:33-39`
- Repository: `shops_repository.go:39-50`

**Special Considerations**:
- Blob storage integration for images
- Pagination logic
- Message cleanup on shop deletion

**New Files**:
```
api/shops/messages/
├── controller.go        # 8 endpoints
├── service.go           # 7 methods
├── service_impl.go      # ~300 lines
├── repository.go        # 11 methods
├── repository_impl.go   # ~400 lines
└── images/
    ├── service.go       # Image upload/delete
    └── repository.go    # Blob operations
```

#### Step 2.5: Extract Vehicles Domain (Medium Risk)

**Current Location**:
- Service: `shops_service.go:42-46`
- Repository: `shops_repository.go:52-57`

**New Files**:
```
api/shops/vehicles/
├── controller.go        # 5 endpoints
├── service.go           # 5 methods
├── service_impl.go      # ~150 lines
├── repository.go        # 6 methods
└── repository_impl.go   # ~200 lines
```

#### Step 2.6: Extract Notifications Domain (High Risk)

**Current Location**:
- Service: `shops_service.go:49-63`, `shops_service.go:90-92`
- Repository: `shops_repository.go:59-76`, `shops_repository.go:105-109`

**Special Considerations**:
- Complex relationships (Vehicle → Notifications → Items)
- Change tracking/audit trail
- Bulk operations

**New Files**:
```
api/shops/vehicles/notifications/
├── controller.go        # 7 endpoints
├── service.go           # 7 methods
├── service_impl.go      # ~350 lines
├── repository.go        # 7 methods
├── repository_impl.go   # ~400 lines
├── items/
│   ├── service.go       # 6 methods
│   ├── service_impl.go  # ~200 lines
│   ├── repository.go    # 7 methods
│   └── repository_impl.go # ~250 lines
└── changes/
    ├── service.go       # 3 methods (audit trail)
    └── repository.go    # 3 methods
```

### Phase 3: Route Migration (Week 7)

Update route registration to use new modular structure:

```go
// api/shops/route.go
package shops

import (
    "database/sql"
    "miltechserver/api/shops/core"
    "miltechserver/api/shops/members"
    "miltechserver/api/shops/messages"
    "miltechserver/api/shops/vehicles"
    "miltechserver/api/shops/lists"
    "miltechserver/api/shops/settings"
    "miltechserver/bootstrap"

    "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
    "github.com/gin-gonic/gin"
)

// Dependencies holds shared dependencies for all shop sub-domains
type Dependencies struct {
    DB             *sql.DB
    BlobClient     *azblob.Client
    BlobCredential *azblob.SharedKeyCredential
    Env            *bootstrap.Env
}

// RegisterRoutes sets up all shop-related routes
func RegisterRoutes(deps Dependencies, group *gin.RouterGroup) {
    // Shared authorization service
    authService := shared.NewShopAuthorization(deps.DB)

    // Register each sub-domain
    core.RegisterRoutes(deps, authService, group)
    members.RegisterRoutes(deps, authService, group)
    messages.RegisterRoutes(deps, authService, group)
    vehicles.RegisterRoutes(deps, authService, group)
    lists.RegisterRoutes(deps, authService, group)
    settings.RegisterRoutes(deps, authService, group)
}
```

### Phase 4: Cleanup (Week 8)

1. **Remove Legacy Files**:
   - `api/controller/shops_controller.go`
   - `api/service/shops_service.go`
   - `api/service/shops_service_impl.go`
   - `api/repository/shops_repository.go`
   - `api/repository/shops_repository_impl.go`
   - `api/route/shops_route.go`

2. **Update Imports**: Search and replace all imports pointing to old packages

3. **Final Testing**: Run full integration test suite

---

## API Backward Compatibility

### URL Structure Preservation

All existing URLs **MUST** remain unchanged:

| Current URL | New Handler Location |
|-------------|---------------------|
| `POST /api/v1/auth/shops` | `shops/core/controller.go` |
| `GET /api/v1/auth/shops/:shop_id/members` | `shops/members/controller.go` |
| `POST /api/v1/auth/shops/messages` | `shops/messages/controller.go` |
| `GET /api/v1/auth/shops/:shop_id/vehicles` | `shops/vehicles/controller.go` |
| `POST /api/v1/auth/shops/lists` | `shops/lists/controller.go` |

### Response Format Preservation

All response types remain in `api/response/`:
- `user_shops_response.go`
- `vehicle_notifications_with_items_response.go`
- `notification_changes_response.go`

These are NOT moved to avoid breaking client deserialization.

---

## Testing Strategy

### Unit Tests Per Package

Each new package must include:

```go
// api/shops/members/service_test.go
package members

func TestJoinShopViaInviteCode_ValidCode(t *testing.T) {}
func TestJoinShopViaInviteCode_InvalidCode(t *testing.T) {}
func TestJoinShopViaInviteCode_ExpiredCode(t *testing.T) {}
func TestJoinShopViaInviteCode_AlreadyMember(t *testing.T) {}
func TestLeaveShop_Success(t *testing.T) {}
func TestLeaveShop_NotMember(t *testing.T) {}
func TestRemoveMember_AsAdmin(t *testing.T) {}
func TestRemoveMember_NotAdmin(t *testing.T) {}
// ... etc
```

### Integration Tests

```go
// api/shops/integration_test.go
package shops

func TestFullShopLifecycle(t *testing.T) {
    // 1. Create shop
    // 2. Generate invite code
    // 3. Second user joins
    // 4. Create vehicle
    // 5. Create notification
    // 6. Add items
    // 7. View audit trail
    // 8. Leave shop
    // 9. Delete shop
}
```

### Minimum Coverage Targets

| Package | Target Coverage |
|---------|-----------------|
| `shops/shared` | 90% |
| `shops/core` | 80% |
| `shops/members` | 85% |
| `shops/messages` | 75% |
| `shops/vehicles` | 80% |
| `shops/notifications` | 80% |
| `shops/lists` | 80% |
| `shops/settings` | 90% |

---

## Risk Mitigation

### 1. Feature Flags

Use environment variables to toggle between old and new implementations:

```go
// bootstrap/env.go
type Env struct {
    // ... existing fields
    UseModularShops bool  // Toggle new shops implementation
}

// api/route/route.go
if env.UseModularShops {
    shops.RegisterRoutes(deps, authRoutes)
} else {
    route.NewShopsRouter(db, blobClient, env, authRoutes)
}
```

### 2. Parallel Running

During migration, run both implementations and compare results:

```go
func (s *shopServiceWrapper) GetShopByID(user *bootstrap.User, shopID string) (*response.ShopDetailResponse, error) {
    // Get from legacy
    legacyResult, legacyErr := s.legacy.GetShopByID(user, shopID)

    // Get from new
    newResult, newErr := s.new.GetShopByID(user, shopID)

    // Compare (in dev only)
    if os.Getenv("COMPARE_SHOP_IMPLS") == "true" {
        compareResults(legacyResult, newResult)
    }

    // Return legacy for now
    return legacyResult, legacyErr
}
```

### 3. Rollback Plan

Each phase can be rolled back independently:

1. **Git branches**: Each phase in separate branch
2. **Database**: No schema changes required
3. **Feature flag**: Instant rollback to legacy

---

## Success Metrics

### Code Quality

| Metric | Current | Target |
|--------|---------|--------|
| Largest file | 2,469 lines | < 400 lines |
| Methods per interface | 58 | < 10 |
| Cyclomatic complexity | Unknown | < 10 per function |
| Test coverage | 0% | > 80% |

### Maintainability

- **Time to understand a feature**: 2 hours → 30 minutes
- **Time to add a new feature**: 1 day → 2 hours
- **Number of files changed for a bug fix**: 3+ → 1-2

### Performance

- **No regression** in API response times
- **Same memory footprint**
- **Same database query patterns**

---

## Timeline Summary

| Week | Phase | Deliverables |
|------|-------|-------------|
| 1-2 | Foundation | Shared package, wrapper interfaces, feature flag |
| 3 | Settings | Fully extracted settings domain with tests |
| 4 | Members | Members + Invites extracted with tests |
| 5 | Lists | Lists + Items extracted with tests |
| 6 | Messages & Vehicles | Messages and Vehicles extracted with tests |
| 7 | Notifications | Full notifications domain extracted |
| 8 | Cleanup | Remove legacy code, final testing |

---

## Appendix A: File Line Counts

Current state of monolith files:

```
$ wc -l api/repository/shops_repository_impl.go
    2301 api/repository/shops_repository_impl.go

$ wc -l api/service/shops_service_impl.go
    2469 api/service/shops_service_impl.go

$ wc -l api/controller/shops_controller.go
    1975 api/controller/shops_controller.go

TOTAL: 6,745 lines
```

Target state after refactoring:

```
api/shops/
├── core/           (~400 lines total)
├── members/        (~600 lines total)
├── messages/       (~700 lines total)
├── vehicles/       (~350 lines total)
├── notifications/  (~1200 lines total)
├── lists/          (~730 lines total)
├── settings/       (~180 lines total)
├── shared/         (~200 lines total)
└── route.go        (~100 lines)

TOTAL: ~4,460 lines (34% reduction through deduplication)
```

---

## Appendix B: Current Endpoint Mapping

Complete mapping of 64 endpoints to new packages:

### Core Shop (10 endpoints)
- `POST /shops` → `shops/core`
- `GET /shops` → `shops/core`
- `GET /shops/user-data` → `shops/core`
- `GET /shops/:shop_id` → `shops/core`
- `PUT /shops/:shop_id` → `shops/core`
- `DELETE /shops/:shop_id` → `shops/core`
- `GET /shops/:shop_id/settings` → `shops/settings`
- `PUT /shops/:shop_id/settings` → `shops/settings`
- `GET /shops/:shop_id/settings/admin-only-lists` → `shops/settings` (legacy)
- `PUT /shops/:shop_id/settings/admin-only-lists` → `shops/settings` (legacy)
- `GET /shops/:shop_id/is-admin` → `shops/core`

### Members (5 endpoints)
- `POST /shops/join` → `shops/members`
- `DELETE /shops/:shop_id/leave` → `shops/members`
- `DELETE /shops/members/remove` → `shops/members`
- `PUT /shops/members/promote` → `shops/members`
- `GET /shops/:shop_id/members` → `shops/members`

### Invite Codes (4 endpoints)
- `POST /shops/invite-codes` → `shops/members/invites`
- `GET /shops/:shop_id/invite-codes` → `shops/members/invites`
- `DELETE /shops/invite-codes/:code_id` → `shops/members/invites`
- `DELETE /shops/invite-codes/:code_id/delete` → `shops/members/invites`

### Messages (8 endpoints)
- `POST /shops/messages` → `shops/messages`
- `GET /shops/:shop_id/messages` → `shops/messages`
- `GET /shops/:shop_id/messages/paginated` → `shops/messages`
- `PUT /shops/messages` → `shops/messages`
- `DELETE /shops/messages/:message_id` → `shops/messages`
- `POST /shops/messages/image/upload` → `shops/messages`
- `DELETE /shops/messages/image/:message_id` → `shops/messages`

### Vehicles (5 endpoints)
- `POST /shops/vehicles` → `shops/vehicles`
- `GET /shops/:shop_id/vehicles` → `shops/vehicles`
- `GET /shops/vehicles/:vehicle_id` → `shops/vehicles`
- `PUT /shops/vehicles` → `shops/vehicles`
- `DELETE /shops/vehicles/:vehicle_id` → `shops/vehicles`

### Notifications (7 endpoints)
- `POST /shops/vehicles/notifications` → `shops/vehicles/notifications`
- `GET /shops/vehicles/:vehicle_id/notifications` → `shops/vehicles/notifications`
- `GET /shops/vehicles/:vehicle_id/notifications-with-items` → `shops/vehicles/notifications`
- `GET /shops/:shop_id/notifications` → `shops/vehicles/notifications`
- `GET /shops/vehicles/notifications/:notification_id` → `shops/vehicles/notifications`
- `PUT /shops/vehicles/notifications` → `shops/vehicles/notifications`
- `DELETE /shops/vehicles/notifications/:notification_id` → `shops/vehicles/notifications`

### Notification Items (6 endpoints)
- `POST /shops/notifications/items` → `shops/vehicles/notifications/items`
- `GET /shops/notifications/:notification_id/items` → `shops/vehicles/notifications/items`
- `GET /shops/:shop_id/notification-items` → `shops/vehicles/notifications/items`
- `POST /shops/notifications/items/bulk` → `shops/vehicles/notifications/items`
- `DELETE /shops/notifications/items/:item_id` → `shops/vehicles/notifications/items`
- `DELETE /shops/notifications/items/bulk` → `shops/vehicles/notifications/items`

### Lists (5 endpoints)
- `POST /shops/lists` → `shops/lists`
- `GET /shops/:shop_id/lists` → `shops/lists`
- `GET /shops/lists/:list_id` → `shops/lists`
- `PUT /shops/lists` → `shops/lists`
- `DELETE /shops/lists` → `shops/lists`

### List Items (6 endpoints)
- `POST /shops/lists/items` → `shops/lists/items`
- `GET /shops/lists/:list_id/items` → `shops/lists/items`
- `PUT /shops/lists/items` → `shops/lists/items`
- `DELETE /shops/lists/items` → `shops/lists/items`
- `POST /shops/lists/items/bulk` → `shops/lists/items`
- `DELETE /shops/lists/items/bulk` → `shops/lists/items`

### Change Tracking (3 endpoints)
- `GET /shops/notifications/:notification_id/changes` → `shops/vehicles/notifications/changes`
- `GET /shops/:shop_id/notifications/changes` → `shops/vehicles/notifications/changes`
- `GET /shops/vehicles/:vehicle_id/notifications/changes` → `shops/vehicles/notifications/changes`

---

## Appendix C: Database Tables Affected

Tables that will be accessed by each new package:

| Package | Tables |
|---------|--------|
| `shops/core` | `shops` |
| `shops/members` | `shop_members`, `shop_invite_codes` |
| `shops/messages` | `shop_messages` |
| `shops/vehicles` | `shop_vehicle` |
| `shops/notifications` | `shop_vehicle_notifications`, `shop_notification_items`, `shop_vehicle_notification_changes` |
| `shops/lists` | `shop_lists`, `shop_list_items` |
| `shops/settings` | `shops` (settings columns only) |

---

*Document Version: 1.0*
*Created: January 2026*
*Author: Architecture Review*
