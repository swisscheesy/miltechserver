# Shop Settings Unified API Design Document

**Date**: 2026-01-07
**Author**: System Design
**Status**: Design Phase
**Feature**: Unified Shop Settings API

---

## Table of Contents
1. [Executive Summary](#executive-summary)
2. [Current Implementation Analysis](#current-implementation-analysis)
3. [Problem Statement](#problem-statement)
4. [Proposed Solution](#proposed-solution)
5. [Technical Design](#technical-design)
6. [Implementation Plan](#implementation-plan)
7. [API Specification](#api-specification)
8. [Testing Strategy](#testing-strategy)
9. [Migration & Rollout](#migration--rollout)
10. [Future Considerations](#future-considerations)

---

## Executive Summary

This document outlines the design for refactoring the shop settings feature to use a unified API approach that allows easy expansion for future settings while maintaining backward compatibility with existing endpoints.

**Key Goals:**
- Single endpoint to retrieve all shop settings
- Unified endpoint to update multiple settings with partial update support
- Easy addition of new settings in the future
- Maintain backward compatibility with existing endpoints
- Type-safe, maintainable implementation

**Current State:** One setting (`admin_only_lists`) with dedicated GET/PUT endpoints
**Future State:** Unified settings API with extensible architecture

---

## Current Implementation Analysis

### Database Schema
**Table:** `shops`

Current relevant fields:
```sql
CREATE TABLE shops (
    id UUID PRIMARY KEY,
    name VARCHAR NOT NULL,
    details TEXT,
    created_by VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    admin_only_lists BOOLEAN DEFAULT FALSE
);
```

### Current API Endpoints

#### 1. Get Admin Only Lists Setting
- **Endpoint:** `GET /shops/:shop_id/settings/admin-only-lists`
- **Permission:** Any shop member can read
- **Response:**
```json
{
    "status": 200,
    "message": "Shop admin_only_lists setting retrieved successfully",
    "data": {
        "shop_id": "uuid",
        "admin_only_lists": true
    }
}
```

#### 2. Update Admin Only Lists Setting
- **Endpoint:** `PUT /shops/:shop_id/settings/admin-only-lists`
- **Permission:** Shop admins only
- **Request:**
```json
{
    "admin_only_lists": true
}
```
- **Response:**
```json
{
    "status": 200,
    "message": "Shop admin_only_lists setting updated successfully",
    "data": {
        "shop_id": "uuid",
        "admin_only_lists": true
    }
}
```

#### 3. Check Admin Status
- **Endpoint:** `GET /shops/:shop_id/is-admin`
- **Permission:** Any shop member
- **Response:**
```json
{
    "status": 200,
    "message": "",
    "data": {
        "is_admin": true
    }
}
```

### Current Code Architecture

**Layer Structure:**
1. **Route Layer** (`api/route/shops_route.go`): Registers endpoints
2. **Controller Layer** (`api/controller/shops_controller.go`): Handles HTTP requests/responses
3. **Service Layer** (`api/service/shops_service.go`): Business logic and authorization
4. **Repository Layer** (`api/repository/shops_repository.go`): Database operations
5. **Model Layer** (`.gen/miltech_ng/public/model/shops.go`): Auto-generated database models

**Permission Model:**
- **Read Settings:** Any shop member
- **Write Settings:** Shop admins only
- Verified via `IsUserMemberOfShop()` and `IsUserShopAdmin()`

---

## Problem Statement

### Current Limitations

1. **Scalability Issues**
   - Each new setting requires 2 new endpoints (GET and PUT)
   - Routes file grows linearly with settings
   - API becomes fragmented and inconsistent

2. **Client Complexity**
   - Clients must make multiple API calls to fetch all settings
   - No atomic way to view complete shop configuration
   - Higher network overhead for settings management

3. **Maintenance Burden**
   - Duplicated authorization logic across endpoints
   - Each setting needs separate controller methods
   - Inconsistent response structures possible

4. **Developer Experience**
   - Adding a new setting requires changes in 5+ files
   - Easy to forget updating one layer
   - No centralized settings validation

### Example: Adding a New Setting (Current Approach)

To add `allow_guest_view` setting, developer must:
1. Add column to database schema
2. Update `model.Shops` struct (regenerate with go-jet)
3. Add `GetShopAllowGuestViewSetting()` to repository interface
4. Implement repository method
5. Add `GetShopAllowGuestViewSetting()` to service interface
6. Implement service method with auth
7. Add `UpdateShopAllowGuestViewSetting()` to repository interface
8. Implement repository update method
9. Add `UpdateShopAllowGuestViewSetting()` to service interface
10. Implement service update method with auth
11. Add controller method for GET
12. Add controller method for PUT
13. Register 2 new routes
14. Update API documentation

**Total: ~14 steps, changes in 5+ files**

---

## Proposed Solution

### Design Principles

1. **Unified Access**: Single endpoint returns all settings
2. **Partial Updates**: Clients can update one or multiple settings
3. **Backward Compatibility**: Keep existing endpoints for legacy clients
4. **Type Safety**: Use Go structs with validation
5. **Extensibility**: Adding settings requires minimal changes
6. **Consistency**: All settings follow same permission model

### Architecture Overview

```
Client Request
     ↓
[Unified Settings Endpoint]
     ↓
[Controller: ShopsController.GetShopSettings/UpdateShopSettings]
     ↓
[Service: ShopsService (authorization)]
     ↓
[Repository: Database operations]
     ↓
[Database: shops table (typed columns)]
```

### Key Design Decisions

| Decision | Option Chosen | Rationale |
|----------|--------------|-----------|
| Storage | Columns in shops table | Type-safe, queryable, follows existing pattern |
| Backward Compatibility | Keep old endpoints | Safer for existing mobile clients |
| Update Strategy | Partial updates | More flexible for clients |
| Permissions | Same for all settings | Consistent, simple authorization |

---

## Technical Design

### 1. New Request/Response Structures

**File:** `api/request/shops_request.go`

```go
// GetShopSettingsResponse - Response for GET /shops/:shop_id/settings
type ShopSettings struct {
    AdminOnlyLists bool `json:"admin_only_lists"`
    // Future settings will be added here as the shop grows
    // Example: AllowGuestView bool `json:"allow_guest_view"`
}

// UpdateShopSettingsRequest - Request for PUT /shops/:shop_id/settings
// All fields are optional (*bool) to support partial updates
type UpdateShopSettingsRequest struct {
    AdminOnlyLists *bool `json:"admin_only_lists,omitempty"`
    // Future settings will be added here as optional pointers
    // Example: AllowGuestView *bool `json:"allow_guest_view,omitempty"`
}
```

**Validation Rules:**
- At least one field must be provided for updates
- Each field validates independently
- Unknown fields are ignored (forward compatibility)

### 2. Controller Methods

**File:** `api/controller/shops_controller.go`

```go
// GetShopSettings returns all settings for a shop
func (controller *ShopsController) GetShopSettings(c *gin.Context) {
    ctxUser, ok := c.Get("user")
    user, _ := ctxUser.(*bootstrap.User)

    if !ok {
        c.JSON(401, gin.H{"message": "unauthorized"})
        slog.Info("Unauthorized request")
        return
    }

    shopID := c.Param("shop_id")
    if shopID == "" {
        c.JSON(400, gin.H{"message": "shop_id is required"})
        return
    }

    settings, err := controller.ShopsService.GetShopSettings(user, shopID)
    if err != nil {
        c.Error(err)
        return
    }

    c.JSON(200, response.StandardResponse{
        Status:  200,
        Message: "Shop settings retrieved successfully",
        Data:    settings,
    })
}

// UpdateShopSettings updates one or more shop settings (admin only)
func (controller *ShopsController) UpdateShopSettings(c *gin.Context) {
    ctxUser, ok := c.Get("user")
    user, _ := ctxUser.(*bootstrap.User)

    if !ok {
        c.JSON(401, gin.H{"message": "unauthorized"})
        slog.Info("Unauthorized request")
        return
    }

    shopID := c.Param("shop_id")
    if shopID == "" {
        c.JSON(400, gin.H{"message": "shop_id is required"})
        return
    }

    var req request.UpdateShopSettingsRequest
    if err := c.BindJSON(&req); err != nil {
        slog.Info("invalid request", "error", err)
        c.JSON(400, gin.H{"message": "invalid request"})
        return
    }

    // Validate that at least one setting is being updated
    if req.AdminOnlyLists == nil {
        // Add checks for future settings here
        c.JSON(400, gin.H{"message": "at least one setting must be provided"})
        return
    }

    updatedSettings, err := controller.ShopsService.UpdateShopSettings(user, shopID, req)
    if err != nil {
        c.Error(err)
        return
    }

    c.JSON(200, response.StandardResponse{
        Status:  200,
        Message: "Shop settings updated successfully",
        Data:    updatedSettings,
    })
}
```

### 3. Service Layer

**File:** `api/service/shops_service.go` (Interface)

```go
type ShopsService interface {
    // ... existing methods ...

    // Unified Settings Operations
    GetShopSettings(user *bootstrap.User, shopID string) (*request.ShopSettings, error)
    UpdateShopSettings(user *bootstrap.User, shopID string, settings request.UpdateShopSettingsRequest) (*request.ShopSettings, error)

    // Keep existing methods for backward compatibility
    GetShopAdminOnlyListsSetting(user *bootstrap.User, shopID string) (bool, error)
    UpdateShopAdminOnlyListsSetting(user *bootstrap.User, shopID string, adminOnlyLists bool) error
    IsUserShopAdmin(user *bootstrap.User, shopID string) (bool, error)
}
```

**File:** `api/service/shops_service_impl.go` (Implementation)

```go
// GetShopSettings returns all settings for a shop
// Any shop member can read settings
func (service *ShopsServiceImpl) GetShopSettings(user *bootstrap.User, shopID string) (*request.ShopSettings, error) {
    if user == nil {
        return nil, errors.New("unauthorized user")
    }

    // Verify user is a member of the shop
    isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, shopID)
    if err != nil {
        return nil, fmt.Errorf("failed to verify membership: %w", err)
    }

    if !isMember {
        return nil, errors.New("access denied: user is not a member of this shop")
    }

    // Get settings from repository
    settings, err := service.ShopsRepository.GetShopSettings(shopID)
    if err != nil {
        return nil, fmt.Errorf("failed to get shop settings: %w", err)
    }

    slog.Info("Shop settings retrieved", "user_id", user.UserID, "shop_id", shopID)
    return settings, nil
}

// UpdateShopSettings updates one or more shop settings (admin only)
// Supports partial updates - only provided fields are modified
func (service *ShopsServiceImpl) UpdateShopSettings(user *bootstrap.User, shopID string, updates request.UpdateShopSettingsRequest) (*request.ShopSettings, error) {
    if user == nil {
        return nil, errors.New("unauthorized user")
    }

    // Check if user is admin of the shop
    isAdmin, err := service.ShopsRepository.IsUserShopAdmin(user, shopID)
    if err != nil {
        return nil, fmt.Errorf("failed to verify admin status: %w", err)
    }

    if !isAdmin {
        return nil, errors.New("access denied: only shop administrators can modify settings")
    }

    // Update settings in repository (partial update)
    err = service.ShopsRepository.UpdateShopSettings(shopID, updates)
    if err != nil {
        return nil, fmt.Errorf("failed to update shop settings: %w", err)
    }

    // Fetch and return updated settings
    updatedSettings, err := service.ShopsRepository.GetShopSettings(shopID)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch updated settings: %w", err)
    }

    slog.Info("Shop settings updated by admin",
        "user_id", user.UserID,
        "shop_id", shopID,
        "updated_fields", formatUpdatedFields(updates))

    return updatedSettings, nil
}

// Helper function to log which fields were updated
func formatUpdatedFields(updates request.UpdateShopSettingsRequest) string {
    var fields []string
    if updates.AdminOnlyLists != nil {
        fields = append(fields, "admin_only_lists")
    }
    // Add future settings here
    return strings.Join(fields, ", ")
}
```

### 4. Repository Layer

**File:** `api/repository/shops_repository.go` (Interface)

```go
type ShopsRepository interface {
    // ... existing methods ...

    // Unified Settings Operations
    GetShopSettings(shopID string) (*request.ShopSettings, error)
    UpdateShopSettings(shopID string, updates request.UpdateShopSettingsRequest) error

    // Keep existing methods for backward compatibility
    GetShopAdminOnlyListsSetting(shopID string) (bool, error)
    UpdateShopAdminOnlyListsSetting(shopID string, adminOnlyLists bool) error
}
```

**File:** `api/repository/shops_repository_impl.go` (Implementation)

```go
// GetShopSettings retrieves all settings for a shop
func (repo *ShopsRepositoryImpl) GetShopSettings(shopID string) (*request.ShopSettings, error) {
    var settings request.ShopSettings

    query := `
        SELECT
            admin_only_lists
            -- Future settings will be added here
        FROM shops
        WHERE id = $1
    `

    err := repo.DB.QueryRow(query, shopID).Scan(
        &settings.AdminOnlyLists,
        // Future setting scans will be added here
    )

    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("shop not found: %s", shopID)
        }
        return nil, fmt.Errorf("failed to query shop settings: %w", err)
    }

    return &settings, nil
}

// UpdateShopSettings updates shop settings (supports partial updates)
func (repo *ShopsRepositoryImpl) UpdateShopSettings(shopID string, updates request.UpdateShopSettingsRequest) error {
    // Build dynamic UPDATE query based on provided fields
    setClauses := []string{"updated_at = NOW()"}
    args := []interface{}{}
    argPosition := 1

    if updates.AdminOnlyLists != nil {
        setClauses = append(setClauses, fmt.Sprintf("admin_only_lists = $%d", argPosition))
        args = append(args, *updates.AdminOnlyLists)
        argPosition++
    }

    // Future settings will follow the same pattern:
    // if updates.AllowGuestView != nil {
    //     setClauses = append(setClauses, fmt.Sprintf("allow_guest_view = $%d", argPosition))
    //     args = append(args, *updates.AllowGuestView)
    //     argPosition++
    // }

    if len(setClauses) == 1 {
        // Only updated_at would be set, no actual changes
        return errors.New("no settings to update")
    }

    // Add shop_id as final parameter
    args = append(args, shopID)

    query := fmt.Sprintf(`
        UPDATE shops
        SET %s
        WHERE id = $%d
    `, strings.Join(setClauses, ", "), argPosition)

    result, err := repo.DB.Exec(query, args...)
    if err != nil {
        return fmt.Errorf("failed to update shop settings: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to check rows affected: %w", err)
    }

    if rowsAffected == 0 {
        return fmt.Errorf("shop not found: %s", shopID)
    }

    return nil
}
```

### 5. Route Registration

**File:** `api/route/shops_route.go`

```go
func NewShopsRouter(db *sql.DB, blobClient *azblob.Client, env *bootstrap.Env, group *gin.RouterGroup) {
    // ... existing setup ...

    // Shop Settings Operations (Unified)
    group.GET("/shops/:shop_id/settings", pc.GetShopSettings)
    group.PUT("/shops/:shop_id/settings", pc.UpdateShopSettings)

    // Shop Settings Operations (Legacy - Backward Compatibility)
    group.GET("/shops/:shop_id/settings/admin-only-lists", pc.GetShopAdminOnlyListsSetting)
    group.PUT("/shops/:shop_id/settings/admin-only-lists", pc.UpdateShopAdminOnlyListsSetting)
    group.GET("/shops/:shop_id/is-admin", pc.CheckUserIsShopAdmin)

    // ... rest of routes ...
}
```

---

## Implementation Plan

### Phase 1: Foundation (Day 1)
**Goal:** Create new unified endpoint infrastructure

**Steps:**
1. Add `ShopSettings` struct to `api/request/shops_request.go`
2. Add `UpdateShopSettingsRequest` struct to `api/request/shops_request.go`
3. Update repository interface with new methods
4. Implement repository methods
5. Update service interface with new methods
6. Implement service methods
7. Add controller methods
8. Register new routes

**Files Modified:**
- `api/request/shops_request.go`
- `api/repository/shops_repository.go`
- `api/repository/shops_repository_impl.go`
- `api/service/shops_service.go`
- `api/service/shops_service_impl.go`
- `api/controller/shops_controller.go`
- `api/route/shops_route.go`

**Testing:**
- Complete manual testing checklist
- Test backward compatibility with old endpoints

### Phase 2: Documentation (Day 2)
**Goal:** Update documentation and add examples

**Steps:**
1. Update API documentation in `docs/`
2. Add usage examples
3. Create migration guide for client developers
4. Document the process for adding new settings

**Files Created/Modified:**
- Update `docs/OLD/shop_routes.md` with new endpoints
- Create examples directory with sample requests

### Phase 3: Future Enhancement (When Next Setting Needed)
**Goal:** Demonstrate easy addition of new settings

**Example: Adding `allow_guest_view` setting:**

1. **Database Migration** (if not exists)
   ```sql
   ALTER TABLE shops
   ADD COLUMN allow_guest_view BOOLEAN DEFAULT FALSE;
   ```

2. **Update Structs** (3 locations)
   ```go
   // In ShopSettings
   AllowGuestView bool `json:"allow_guest_view"`

   // In UpdateShopSettingsRequest
   AllowGuestView *bool `json:"allow_guest_view,omitempty"`
   ```

3. **Update Repository** (2 locations)
   ```go
   // In GetShopSettings - add to SELECT and Scan
   // In UpdateShopSettings - add to conditional logic
   ```

4. **Update Validation** (1 location)
   ```go
   // In controller - add to validation check
   if req.AdminOnlyLists == nil && req.AllowGuestView == nil {
       // error
   }
   ```

5. **Regenerate go-jet models** (if needed)
   ```bash
   # Regenerate from database schema
   ```

**Total: ~5-6 code changes vs 14 in old approach**

---

## API Specification

### New Endpoints

#### 1. Get All Shop Settings

**Endpoint:** `GET /api/v1/shops/:shop_id/settings`

**Description:** Retrieves all settings for a shop in a single call.

**Authentication:** Required (Firebase JWT)

**Authorization:** User must be a member of the shop

**Path Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| shop_id | string (UUID) | Yes | The shop's unique identifier |

**Success Response (200 OK):**
```json
{
    "status": 200,
    "message": "Shop settings retrieved successfully",
    "data": {
        "admin_only_lists": true
    }
}
```

**Error Responses:**

**401 Unauthorized:**
```json
{
    "message": "unauthorized"
}
```

**400 Bad Request:**
```json
{
    "message": "shop_id is required"
}
```

**403 Forbidden:**
```json
{
    "message": "access denied: user is not a member of this shop"
}
```

**404 Not Found:**
```json
{
    "message": "shop not found"
}
```

**Example cURL:**
```bash
curl -X GET \
  'https://api.example.com/api/v1/shops/550e8400-e29b-41d4-a716-446655440000/settings' \
  -H 'Authorization: Bearer <firebase-jwt-token>'
```

---

#### 2. Update Shop Settings (Partial)

**Endpoint:** `PUT /api/v1/shops/:shop_id/settings`

**Description:** Updates one or more shop settings. Only provided fields are modified. All other settings remain unchanged.

**Authentication:** Required (Firebase JWT)

**Authorization:** User must be a shop administrator

**Path Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| shop_id | string (UUID) | Yes | The shop's unique identifier |

**Request Body:**
All fields are optional. At least one must be provided.

```json
{
    "admin_only_lists": true
}
```

**Request Body Fields:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| admin_only_lists | boolean | No | When true, only admins can create/modify lists |

**Success Response (200 OK):**
Returns the complete updated settings object.

```json
{
    "status": 200,
    "message": "Shop settings updated successfully",
    "data": {
        "admin_only_lists": true
    }
}
```

**Error Responses:**

**401 Unauthorized:**
```json
{
    "message": "unauthorized"
}
```

**400 Bad Request (no fields provided):**
```json
{
    "message": "at least one setting must be provided"
}
```

**403 Forbidden (not admin):**
```json
{
    "message": "access denied: only shop administrators can modify settings"
}
```

**404 Not Found:**
```json
{
    "message": "shop not found"
}
```

**Example cURL:**
```bash
curl -X PUT \
  'https://api.example.com/api/v1/shops/550e8400-e29b-41d4-a716-446655440000/settings' \
  -H 'Authorization: Bearer <firebase-jwt-token>' \
  -H 'Content-Type: application/json' \
  -d '{
    "admin_only_lists": false
  }'
```

---

### Legacy Endpoints (Maintained for Backward Compatibility)

These endpoints remain functional but clients are encouraged to migrate to the unified API.

#### 1. Get Admin Only Lists Setting (Legacy)
- **Endpoint:** `GET /api/v1/shops/:shop_id/settings/admin-only-lists`
- **Status:** Active (backward compatibility)
- **Recommendation:** Use `GET /shops/:shop_id/settings` instead

#### 2. Update Admin Only Lists Setting (Legacy)
- **Endpoint:** `PUT /api/v1/shops/:shop_id/settings/admin-only-lists`
- **Status:** Active (backward compatibility)
- **Recommendation:** Use `PUT /shops/:shop_id/settings` instead

---

## Testing Strategy

**Testing Approach:** All testing will be performed manually. No automated unit or integration tests are required for this feature.

### Manual Testing Checklist

#### Unified Endpoints Testing

**GET /shops/:shop_id/settings**
- [ ] Returns all settings with correct values
- [ ] Returns 200 OK with proper response structure
- [ ] Returns 401 when no authentication token provided
- [ ] Returns 403 when user is not a shop member
- [ ] Returns 404 when shop_id doesn't exist
- [ ] Returns 400 when shop_id parameter is missing

**PUT /shops/:shop_id/settings**
- [ ] Successfully updates `admin_only_lists` setting
- [ ] Returns complete updated settings in response
- [ ] Returns 200 OK with proper response structure
- [ ] Returns 401 when no authentication token provided
- [ ] Returns 403 when user is not a shop admin (only a member)
- [ ] Returns 403 when user is not a shop member
- [ ] Returns 404 when shop_id doesn't exist
- [ ] Returns 400 when shop_id parameter is missing
- [ ] Returns 400 when no settings fields are provided in request body
- [ ] Returns 400 when request body is malformed JSON

#### Partial Update Behavior
- [ ] Updating only `admin_only_lists` doesn't affect other fields
- [ ] Settings not included in request remain unchanged
- [ ] Can update setting to `true`
- [ ] Can update setting to `false`
- [ ] Can toggle setting back and forth multiple times

#### Authorization Testing
- [ ] Shop admin can read settings via GET
- [ ] Shop admin can update settings via PUT
- [ ] Shop member (non-admin) can read settings via GET
- [ ] Shop member (non-admin) cannot update settings via PUT (403)
- [ ] Non-member cannot read settings via GET (403)
- [ ] Non-member cannot update settings via PUT (403)

#### Legacy Endpoints (Backward Compatibility)
- [ ] GET /shops/:shop_id/settings/admin-only-lists still works correctly
- [ ] PUT /shops/:shop_id/settings/admin-only-lists still works correctly
- [ ] Settings updated via legacy PUT are reflected in unified GET
- [ ] Settings updated via unified PUT are reflected in legacy GET
- [ ] Both endpoints return consistent data formats

#### Data Consistency
- [ ] Settings changes via unified endpoint appear in GetUserShops response
- [ ] Default value for `admin_only_lists` is `false` for new shops
- [ ] Setting values persist correctly after update
- [ ] Multiple consecutive updates work correctly
- [ ] Settings are shop-specific (updating one shop doesn't affect others)

#### Edge Cases
- [ ] Creating a shop with `admin_only_lists: true` sets it correctly
- [ ] Creating a shop without specifying `admin_only_lists` defaults to `false`
- [ ] Very long shop_id (invalid UUID format) returns 400 or 404
- [ ] Malformed JSON in PUT request returns 400
- [ ] Extra unknown fields in PUT request are ignored (forward compatibility)

### Testing Tools

**Recommended Tools:**
- **cURL** or **Postman** for API endpoint testing
- **Database client** (psql, pgAdmin, DBeaver) to verify data changes
- **Server logs** to monitor error conditions and debug issues

### Test Data Setup

Before testing, ensure you have:
1. At least 2 test shops created
2. At least 3 test users:
   - User A: Admin of Shop 1
   - User B: Member (non-admin) of Shop 1
   - User C: Not a member of Shop 1
3. Valid Firebase JWT tokens for all test users

### Sample Test Requests

**Get All Settings:**
```bash
curl -X GET \
  'http://localhost:8080/api/v1/shops/550e8400-e29b-41d4-a716-446655440000/settings' \
  -H 'Authorization: Bearer <firebase-jwt-token>'
```

**Update Setting:**
```bash
curl -X PUT \
  'http://localhost:8080/api/v1/shops/550e8400-e29b-41d4-a716-446655440000/settings' \
  -H 'Authorization: Bearer <firebase-jwt-token>' \
  -H 'Content-Type: application/json' \
  -d '{"admin_only_lists": true}'
```

**Test Legacy Endpoint:**
```bash
curl -X GET \
  'http://localhost:8080/api/v1/shops/550e8400-e29b-41d4-a716-446655440000/settings/admin-only-lists' \
  -H 'Authorization: Bearer <firebase-jwt-token>'
```

---

## Migration & Rollout

### Backend Deployment

**Prerequisites:**
- No database migration needed (using existing columns)
- All changes are additive (backward compatible)

**Deployment Steps:**
1. Deploy new code to staging
2. Run manual testing checklist
3. Verify legacy endpoints still function
4. Deploy to production
5. Monitor logs for errors

**Rollback Plan:**
- New endpoints can be disabled by removing route registration
- Legacy endpoints continue to work
- No data migration required for rollback

### Client Migration

**Phase 1: Soft Launch (Week 1-2)**
- New endpoints available
- Update API documentation
- Notify client developers
- Legacy endpoints marked as "discouraged" but not deprecated

**Phase 2: Adoption (Month 1-3)**
- Client teams update to use unified endpoints
- Monitor usage analytics
- Legacy endpoints remain fully functional

**Phase 3: Deprecation (Month 6+)**
- Legacy endpoints marked as deprecated in documentation
- Add deprecation warnings in API responses (custom headers)
- Continue supporting for minimum 6 months

**Phase 4: Sunset (Year 1+)**
- Remove legacy endpoints (breaking change)
- Requires major version bump
- Only after confirming all clients migrated

---

## Future Considerations

### Adding New Settings

**Process Template:**

When adding a new setting (e.g., `allow_public_lists`):

1. **Update Database** (if needed)
   ```sql
   ALTER TABLE shops ADD COLUMN allow_public_lists BOOLEAN DEFAULT FALSE;
   ```

2. **Update `ShopSettings` struct**
   ```go
   type ShopSettings struct {
       AdminOnlyLists   bool `json:"admin_only_lists"`
       AllowPublicLists bool `json:"allow_public_lists"` // NEW
   }
   ```

3. **Update `UpdateShopSettingsRequest` struct**
   ```go
   type UpdateShopSettingsRequest struct {
       AdminOnlyLists   *bool `json:"admin_only_lists,omitempty"`
       AllowPublicLists *bool `json:"allow_public_lists,omitempty"` // NEW
   }
   ```

4. **Update Repository** `GetShopSettings()`
   - Add to SELECT clause
   - Add to Scan() parameters

5. **Update Repository** `UpdateShopSettings()`
   - Add conditional logic for new field

6. **Update Controller Validation**
   - Add to empty check in `UpdateShopSettings()`

7. **Update Documentation**
   - API spec
   - Usage examples

8. **Manual Testing**
   - Run through manual testing checklist
   - Verify all scenarios work correctly

**Estimated Time:** 2-3 hours per new setting (vs 1 day with old approach)

### Setting Categories

If settings grow significantly (10+), consider categorizing:

```json
{
    "permissions": {
        "admin_only_lists": true,
        "allow_guest_view": false
    },
    "notifications": {
        "enable_email": true,
        "enable_push": false
    },
    "features": {
        "enable_advanced_search": true
    }
}
```

This would require:
- Nested structs in `ShopSettings`
- Additional database columns or JSONB
- Updating API spec

### Per-Setting Permissions

If future settings need different permission models:

```go
type SettingPermissions struct {
    Read  string // "member", "admin", "public"
    Write string // "admin", "creator"
}

var settingPermissions = map[string]SettingPermissions{
    "admin_only_lists": {Read: "member", Write: "admin"},
    "shop_logo_url":    {Read: "public", Write: "admin"},
}
```

This would require:
- Permission lookup in service layer
- More complex authorization logic
- Documentation of per-setting permissions

### Performance Optimization

If shops have many settings (20+):
- Consider caching shop settings in Redis
- Cache key: `shop:settings:{shop_id}`
- TTL: 5 minutes
- Invalidate on update

---

## Appendices

### Appendix A: File Checklist

Files that require changes for this implementation:

- [ ] `api/request/shops_request.go` - Add ShopSettings and UpdateShopSettingsRequest
- [ ] `api/repository/shops_repository.go` - Add interface methods
- [ ] `api/repository/shops_repository_impl.go` - Implement methods
- [ ] `api/service/shops_service.go` - Add interface methods
- [ ] `api/service/shops_service_impl.go` - Implement methods
- [ ] `api/controller/shops_controller.go` - Add controller methods
- [ ] `api/route/shops_route.go` - Register routes
- [ ] `docs/OLD/shop_routes.md` - Update documentation

### Appendix B: Example Usage (Client Perspective)

**Flutter Example:**

```dart
// Get all shop settings
Future<ShopSettings> getShopSettings(String shopId) async {
  final response = await http.get(
    Uri.parse('$baseUrl/shops/$shopId/settings'),
    headers: {'Authorization': 'Bearer $token'},
  );

  if (response.statusCode == 200) {
    final data = json.decode(response.body)['data'];
    return ShopSettings.fromJson(data);
  }
  throw Exception('Failed to load settings');
}

// Update a single setting
Future<ShopSettings> toggleAdminOnlyLists(String shopId, bool enabled) async {
  final response = await http.put(
    Uri.parse('$baseUrl/shops/$shopId/settings'),
    headers: {
      'Authorization': 'Bearer $token',
      'Content-Type': 'application/json',
    },
    body: json.encode({'admin_only_lists': enabled}),
  );

  if (response.statusCode == 200) {
    final data = json.decode(response.body)['data'];
    return ShopSettings.fromJson(data);
  }
  throw Exception('Failed to update setting');
}

// Update multiple settings at once
Future<ShopSettings> updateSettings(String shopId, {
  bool? adminOnlyLists,
  bool? allowGuestView, // Future setting
}) async {
  final body = <String, dynamic>{};
  if (adminOnlyLists != null) body['admin_only_lists'] = adminOnlyLists;
  if (allowGuestView != null) body['allow_guest_view'] = allowGuestView;

  final response = await http.put(
    Uri.parse('$baseUrl/shops/$shopId/settings'),
    headers: {
      'Authorization': 'Bearer $token',
      'Content-Type': 'application/json',
    },
    body: json.encode(body),
  );

  if (response.statusCode == 200) {
    final data = json.decode(response.body)['data'];
    return ShopSettings.fromJson(data);
  }
  throw Exception('Failed to update settings');
}
```

### Appendix C: Database Query Examples

**Get all settings:**
```sql
SELECT
    id,
    admin_only_lists
FROM shops
WHERE id = '550e8400-e29b-41d4-a716-446655440000';
```

**Update single setting:**
```sql
UPDATE shops
SET
    admin_only_lists = true,
    updated_at = NOW()
WHERE id = '550e8400-e29b-41d4-a716-446655440000';
```

**Update multiple settings (future):**
```sql
UPDATE shops
SET
    admin_only_lists = false,
    allow_guest_view = true,
    updated_at = NOW()
WHERE id = '550e8400-e29b-41d4-a716-446655440000';
```

---

## Sign-Off

This design document should be reviewed and approved by:
- [ ] Backend Lead
- [ ] Mobile/Client Lead
- [ ] Product Owner
- [ ] DevOps/Infrastructure

**Questions or concerns?** Open an issue or discussion in the project repository.

---

**End of Design Document**
