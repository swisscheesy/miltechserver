# Shops Feature Performance Optimization Design

## Overview

This document outlines performance improvements for the `shops` feature, encompassing all shop-related endpoints including core operations, members, vehicles, notifications, lists, and messages.

## Progress

- [x] Design analysis complete (2026-02-01)
- [x] Clarifications confirmed (2026-02-01)
- [x] Refactor implementation in codebase (2026-02-01)
- [ ] Manual verification complete
- [x] ADR created and added to `docs/project_notes/decisions.md` (2026-02-01)

## Current State Analysis

### Architecture Summary

The shops feature follows a clean architecture with bounded contexts:

| Layer | Purpose | Files |
|-------|---------|-------|
| Routes | HTTP routing and middleware | `api/shops/*/route.go` |
| Controllers | Request/response handling | `api/controller/shops_controller_*.go` |
| Services | Business logic | `api/shops/*/service_impl.go` |
| Repositories | Data access | `api/shops/*/repository_impl.go` |
| Authorization | Permission checks | `api/shops/shared/authorization.go` |

### Sub-domains

| Domain | Path Prefix | Key Operations |
|--------|-------------|----------------|
| Core | `/shops` | CRUD, list user shops |
| Members | `/shops/:id/members` | Join, leave, promote, list |
| Invites | `/shops/:id/invites` | Generate/validate invite codes |
| Settings | `/shops/:id/settings` | Shop configuration |
| Messages | `/shops/:id/messages` | Paginated messaging with images |
| Vehicles | `/shops/:id/vehicles` | Vehicle management |
| Notifications | `/shops/:id/vehicles/:vid/notifications` | Maintenance tracking |
| Notification Items | `.../notifications/:nid/items` | Equipment/supplies per notification |
| Lists | `/shops/:id/lists` | Procurement/inventory lists |
| List Items | `/shops/:id/lists/:lid/items` | Items within lists |

---

## Critical Performance Issues Identified

### Issue Summary

| Priority | Issue | Impact | Location |
|----------|-------|--------|----------|
| **P0** | Missing database indexes | Very High | Database schema |
| **P0** | N+1 query in notifications | High | `notifications/repository_impl.go` |
| **P0** | Authorization not cached per-request | High | `shared/authorization.go` |
| **P1** | COUNT(*) vs EXISTS for boolean checks | Medium-High | `shared/authorization.go` |
| **P1** | Inefficient admin_check subquery | Medium | `core/repository_impl.go` |
| **P1** | No context timeouts for blob operations | Medium | `core/repository_impl.go` |
| **P2** | Sequential blob deletion | Medium | `core/repository_impl.go` |
| **P2** | OFFSET-based pagination | Medium | `messages/repository_impl.go` |
| **P2** | Slice allocations without pre-sizing | Low-Medium | Multiple files |
| **P3** | Redundant SELECT after INSERT | Low | `lists/repository_impl.go` |
| **P3** | No-op field assignments | Very Low | `vehicles/service_impl.go` |

---

## Proposed Optimizations

### Priority 0: Database Indexes (Critical)

**Impact**: Very High | **Effort**: Low | **Risk**: Low

**Problem**: Authorization queries run on every request without supporting indexes. The `shop_members` table is queried with conditions on `(shop_id, user_id)` and `(shop_id, user_id, role)` but likely has no composite index.

**Evidence**: Every protected endpoint calls:
- `IsUserMemberOfShop(user, shopID)`
- `IsUserShopAdmin(user, shopID)`

Without indexes, these are full table scans.

**Solution**: Create composite indexes on all frequently queried tables.

**Migration File**: `migrations/XXX_create_shop_indexes.sql`

```sql
-- ============================================================
-- CRITICAL: Authorization queries (run on EVERY request)
-- ============================================================

-- Most common authorization pattern: check membership
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_shop_members_shop_user
    ON shop_members(shop_id, user_id);

-- Admin check pattern: verify role
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_shop_members_shop_user_role
    ON shop_members(shop_id, user_id, role);

-- Member listing: fetch all members of a shop
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_shop_members_shop_joined
    ON shop_members(shop_id, joined_at);

-- ============================================================
-- Vehicle queries
-- ============================================================

-- List vehicles by shop
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_shop_vehicle_shop_id
    ON shop_vehicle(shop_id);

-- Ordered vehicle listing
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_shop_vehicle_shop_save_time
    ON shop_vehicle(shop_id, save_time DESC);

-- ============================================================
-- Message queries
-- ============================================================

-- List messages by shop (common)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_shop_messages_shop_id
    ON shop_messages(shop_id);

-- Paginated message listing with ordering
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_shop_messages_shop_created
    ON shop_messages(shop_id, created_at DESC);

-- ============================================================
-- Notification queries
-- ============================================================

-- List notifications by vehicle
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_shop_vehicle_notifications_vehicle_id
    ON shop_vehicle_notifications(vehicle_id);

-- List notifications by shop (for shop-wide views)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_shop_vehicle_notifications_shop_id
    ON shop_vehicle_notifications(shop_id);

-- ============================================================
-- Notification item queries
-- ============================================================

-- Items by notification (critical for N+1 fix)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_shop_notification_items_notification_id
    ON shop_notification_items(notification_id);

-- Items by shop
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_shop_notification_items_shop_id
    ON shop_notification_items(shop_id);

-- ============================================================
-- List queries
-- ============================================================

-- Lists by shop
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_shop_lists_shop_id
    ON shop_lists(shop_id);

-- List items by list
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_shop_list_items_list_id
    ON shop_list_items(list_id);

```

**Note**: The invite code index is intentionally omitted (not implemented).

**Verification Query**:
```sql
-- Run this to verify indexes exist after migration
SELECT tablename, indexname, indexdef
FROM pg_indexes
WHERE tablename LIKE 'shop%'
ORDER BY tablename, indexname;
```

---

### Priority 0: Fix N+1 Query in GetVehicleNotificationsWithItems

**Impact**: High | **Effort**: Medium | **Risk**: Medium

**Problem**: For N notifications, the current code executes N+1 database queries.

**File**: [api/shops/vehicles/notifications/repository_impl.go](../../api/shops/vehicles/notifications/repository_impl.go)

**Current Code** (lines 60-84):
```go
func (repo *RepositoryImpl) GetVehicleNotificationsWithItems(user *bootstrap.User, vehicleID string) ([]response.VehicleNotificationWithItems, error) {
    notifications, err := repo.GetVehicleNotifications(user, vehicleID)
    if err != nil {
        return nil, err
    }

    var result []response.VehicleNotificationWithItems
    for _, notification := range notifications {
        items, err := repo.GetNotificationItems(user, notification.ID)  // N+1 QUERY!
        if err != nil {
            return nil, err
        }
        result = append(result, response.VehicleNotificationWithItems{
            Notification: notification,
            Items:        items,
        })
    }
    return result, nil
}
```

**Solution Option A**: Two queries with in-memory grouping (recommended for simplicity)

```go
func (repo *RepositoryImpl) GetVehicleNotificationsWithItems(user *bootstrap.User, vehicleID string) ([]response.VehicleNotificationWithItems, error) {
    // Query 1: Get all notifications for the vehicle
    notifications, err := repo.GetVehicleNotifications(user, vehicleID)
    if err != nil {
        return nil, err
    }

    if len(notifications) == 0 {
        return []response.VehicleNotificationWithItems{}, nil
    }

    // Extract notification IDs
    notificationIDs := make([]string, len(notifications))
    for i, n := range notifications {
        notificationIDs[i] = n.ID
    }

    // Query 2: Get ALL items for ALL notifications in one query
    allItems, err := repo.GetItemsByNotificationIDs(notificationIDs)
    if err != nil {
        return nil, err
    }

    // Group items by notification ID in memory
    itemsByNotification := make(map[string][]model.ShopVehicleNotificationItems)
    for _, item := range allItems {
        itemsByNotification[item.NotificationID] = append(
            itemsByNotification[item.NotificationID],
            item,
        )
    }

    // Build result
    result := make([]response.VehicleNotificationWithItems, len(notifications))
    for i, notification := range notifications {
        result[i] = response.VehicleNotificationWithItems{
            Notification: notification,
            Items:        itemsByNotification[notification.ID],
        }
    }
    return result, nil
}

// New helper function
func (repo *RepositoryImpl) GetItemsByNotificationIDs(notificationIDs []string) ([]model.ShopVehicleNotificationItems, error) {
    if len(notificationIDs) == 0 {
        return []model.ShopVehicleNotificationItems{}, nil
    }

    // Build IN clause expressions
    expressions := make([]jet.Expression, len(notificationIDs))
    for i, id := range notificationIDs {
        expressions[i] = jet.String(id)
    }

    stmt := SELECT(ShopVehicleNotificationItems.AllColumns).
        FROM(ShopVehicleNotificationItems).
        WHERE(ShopVehicleNotificationItems.NotificationID.IN(expressions...)).
        ORDER_BY(ShopVehicleNotificationItems.CreatedAt.ASC())

    var items []model.ShopVehicleNotificationItems
    err := stmt.Query(repo.db, &items)
    return items, err
}
```

**Solution Option B**: Single query with JSON aggregation (more complex but single round-trip)

```go
func (repo *RepositoryImpl) GetVehicleNotificationsWithItems(user *bootstrap.User, vehicleID string) ([]response.VehicleNotificationWithItems, error) {
    // Single query using PostgreSQL JSON aggregation
    rawSQL := `
        SELECT
            n.id, n.vehicle_id, n.shop_id, n.title, n.description,
            n.created_by, n.created_at, n.updated_at, n.save_time,
            COALESCE(
                json_agg(
                    json_build_object(
                        'id', i.id,
                        'notification_id', i.notification_id,
                        'equipment_id', i.equipment_id,
                        'supply_id', i.supply_id,
                        'quantity', i.quantity,
                        'notes', i.notes,
                        'created_at', i.created_at
                    )
                ) FILTER (WHERE i.id IS NOT NULL),
                '[]'
            ) as items
        FROM shop_vehicle_notifications n
        LEFT JOIN shop_notification_items i ON i.notification_id = n.id
        WHERE n.vehicle_id = $1
        GROUP BY n.id
        ORDER BY n.save_time DESC
    `
    // Execute and parse...
}
```

**Expected Improvement**: From N+1 queries to 2 queries (or 1 with Option B). For 20 notifications, this is a 10x reduction in database round-trips.

---

### Priority 0: Request-Scoped Authorization Caching

**Impact**: High | **Effort**: Medium | **Risk**: Low

**Problem**: Authorization checks (`IsUserMemberOfShop`, `IsUserShopAdmin`) hit the database on every call, but the same check may run multiple times per request.

**Example**: A single request to update a vehicle notification may check:
1. `IsUserMemberOfShop` in middleware
2. `IsUserShopAdmin` in the handler
3. `CanUserModifyVehicle` which calls both again
4. `CanUserModifyNotification` which calls them again

This results in 4-8 identical database queries per request.

**File**: [api/shops/shared/authorization.go](../../api/shops/shared/authorization.go)

**Solution**: Implement request-scoped caching using Gin context.

**New File**: `api/shops/shared/cached_authorization.go`

```go
package shared

import (
    "fmt"
    "sync"

    "github.com/gin-gonic/gin"
    "miltechserver/bootstrap"
)

// CachedAuthorization wraps ShopAuthorization with per-request caching
type CachedAuthorization struct {
    inner ShopAuthorization
    mu    sync.RWMutex
    cache map[string]bool
}

// NewCachedAuthorization creates a cached wrapper around the given authorization
func NewCachedAuthorization(inner ShopAuthorization) *CachedAuthorization {
    return &CachedAuthorization{
        inner: inner,
        cache: make(map[string]bool),
    }
}

// FromContext retrieves or creates a cached authorization for this request
func CachedAuthorizationFromContext(c *gin.Context, factory func() ShopAuthorization) *CachedAuthorization {
    const key = "cached_authorization"

    if cached, exists := c.Get(key); exists {
        return cached.(*CachedAuthorization)
    }

    auth := NewCachedAuthorization(factory())
    c.Set(key, auth)
    return auth
}

func (ca *CachedAuthorization) cacheKey(operation, shopID, userID string) string {
    return fmt.Sprintf("%s:%s:%s", operation, shopID, userID)
}

func (ca *CachedAuthorization) getCached(key string) (bool, bool) {
    ca.mu.RLock()
    defer ca.mu.RUnlock()
    val, ok := ca.cache[key]
    return val, ok
}

func (ca *CachedAuthorization) setCached(key string, val bool) {
    ca.mu.Lock()
    defer ca.mu.Unlock()
    ca.cache[key] = val
}

func (ca *CachedAuthorization) IsUserMemberOfShop(user *bootstrap.User, shopID string) (bool, error) {
    key := ca.cacheKey("member", shopID, user.UserID)

    if val, ok := ca.getCached(key); ok {
        return val, nil
    }

    val, err := ca.inner.IsUserMemberOfShop(user, shopID)
    if err != nil {
        return false, err
    }

    ca.setCached(key, val)
    return val, nil
}

func (ca *CachedAuthorization) IsUserShopAdmin(user *bootstrap.User, shopID string) (bool, error) {
    key := ca.cacheKey("admin", shopID, user.UserID)

    if val, ok := ca.getCached(key); ok {
        return val, nil
    }

    val, err := ca.inner.IsUserShopAdmin(user, shopID)
    if err != nil {
        return false, err
    }

    ca.setCached(key, val)
    return val, nil
}

// Implement remaining ShopAuthorization interface methods similarly...
func (ca *CachedAuthorization) CanUserModifyVehicle(user *bootstrap.User, shopID, vehicleID string) (bool, error) {
    // First check membership (will use cache)
    isMember, err := ca.IsUserMemberOfShop(user, shopID)
    if err != nil || !isMember {
        return false, err
    }

    // Delegate vehicle-specific check to inner
    return ca.inner.CanUserModifyVehicle(user, shopID, vehicleID)
}

// ... implement other methods
```

**Usage in Routes**:
```go
func (handler *Handler) updateNotification(c *gin.Context) {
    user := shared.GetUserFromContext(c)
    shopID := c.Param("shopId")

    // Get cached authorization for this request
    auth := shared.CachedAuthorizationFromContext(c, func() shared.ShopAuthorization {
        return handler.authorization
    })

    // This call will be cached for subsequent checks in this request
    isAdmin, err := auth.IsUserShopAdmin(user, shopID)
    // ...
}
```

**Expected Improvement**: 2-8x reduction in authorization queries per request.

---

### Priority 1: Use EXISTS Instead of COUNT for Boolean Checks

**Impact**: Medium-High | **Effort**: Low | **Risk**: Low

**Problem**: Authorization queries use `COUNT(*)` then check if > 0, but PostgreSQL must count ALL matching rows.

**File**: [api/shops/shared/authorization.go](../../api/shops/shared/authorization.go)

**Current Code**:
```go
func (auth *ShopAuthorizationImpl) IsUserMemberOfShop(user *bootstrap.User, shopID string) (bool, error) {
    var result struct{ Count int64 }

    stmt := SELECT(COUNT(ShopMembers.ID).AS("count")).
        FROM(ShopMembers).
        WHERE(
            ShopMembers.ShopID.EQ(String(shopID)).
            AND(ShopMembers.UserID.EQ(String(user.UserID))),
        )

    err := stmt.Query(auth.db, &result)
    return result.Count > 0, err  // Counted ALL rows just to check > 0
}
```

**Solution**: Use `LIMIT 1` with existence check.

```go
func (auth *ShopAuthorizationImpl) IsUserMemberOfShop(user *bootstrap.User, shopID string) (bool, error) {
    var result []struct{ Exists int }

    stmt := SELECT(Int(1).AS("exists")).
        FROM(ShopMembers).
        WHERE(
            ShopMembers.ShopID.EQ(String(shopID)).
            AND(ShopMembers.UserID.EQ(String(user.UserID))),
        ).
        LIMIT(1)

    err := stmt.Query(auth.db, &result)
    if err != nil {
        return false, err
    }
    return len(result) > 0, nil
}
```

**Alternative using raw EXISTS**:
```go
func (auth *ShopAuthorizationImpl) IsUserMemberOfShop(user *bootstrap.User, shopID string) (bool, error) {
    var result struct{ Exists bool }

    rawSQL := `SELECT EXISTS(
        SELECT 1 FROM shop_members
        WHERE shop_id = $1 AND user_id = $2
    ) as exists`

    row := auth.db.QueryRow(rawSQL, shopID, user.UserID)
    err := row.Scan(&result.Exists)
    return result.Exists, err
}
```

**Expected Improvement**: Query stops at first match instead of counting all. Especially beneficial for shops with many members.

---

### Priority 1: Optimize admin_check Subquery

**Impact**: Medium | **Effort**: Low | **Risk**: Low

**Problem**: The `GetShopsWithStatsForUser` query joins ALL admin records, then filters by user.

**File**: [api/shops/core/repository_impl.go](../../api/shops/core/repository_impl.go) (lines 217-231)

**Current Query Logic**:
```sql
LEFT JOIN (
    SELECT shop_id, user_id
    FROM shop_members
    WHERE role = 'admin'  -- Scans ALL admins in system
) admin_check ON s.id = admin_check.shop_id AND admin_check.user_id = $1
```

**Solution**: Filter by user_id early in the subquery.

```sql
LEFT JOIN (
    SELECT shop_id
    FROM shop_members
    WHERE role = 'admin' AND user_id = $1  -- Filter early, scan much smaller set
) admin_check ON s.id = admin_check.shop_id
```

**Code Change**:
```go
// Before
adminCheckSubquery := SELECT(ShopMembers.ShopID, ShopMembers.UserID).
    FROM(ShopMembers).
    WHERE(ShopMembers.Role.EQ(String("admin"))).
    AsTable("admin_check")

// After
adminCheckSubquery := SELECT(ShopMembers.ShopID).
    FROM(ShopMembers).
    WHERE(
        ShopMembers.Role.EQ(String("admin")).
        AND(ShopMembers.UserID.EQ(String(user.UserID))),
    ).
    AsTable("admin_check")
```

---

### Priority 1: Add Context Timeouts for Blob Operations

**Impact**: Medium | **Effort**: Low | **Risk**: Low

**Problem**: Blob storage operations use `context.Background()` without timeout, which can hang indefinitely.

**File**: [api/shops/core/repository_impl.go](../../api/shops/core/repository_impl.go) (line 315)

**Current Code**:
```go
ctx := context.Background()  // No timeout - can hang forever
pager := repo.blobClient.NewListBlobsFlatPager(containerName, &azblob.ListBlobsFlatOptions{
    Prefix: &prefix,
})
```

**Solution**: Add timeout context.

```go
const blobOperationTimeout = 30 * time.Second

func (repo *RepositoryImpl) deleteShopBlobs(shopID string) error {
    ctx, cancel := context.WithTimeout(context.Background(), blobOperationTimeout)
    defer cancel()

    containerName := "shop-" + shopID
    pager := repo.blobClient.NewListBlobsFlatPager(containerName, &azblob.ListBlobsFlatOptions{
        Prefix: &prefix,
    })

    for pager.More() {
        page, err := pager.NextPage(ctx)
        if err != nil {
            if errors.Is(err, context.DeadlineExceeded) {
                return fmt.Errorf("blob listing timed out after %v: %w", blobOperationTimeout, err)
            }
            return err
        }
        // ...
    }
}
```

---

### Priority 1: Select Only Required Columns

**Impact**: Low-Medium | **Effort**: Low | **Risk**: Low

**Problem**: Several queries fetch all columns when only 1-2 are needed.

**File**: [api/shops/shared/authorization.go](../../api/shops/shared/authorization.go) (lines 188-203)

**Current Code**:
```go
stmt := SELECT(Shops.AllColumns).  // Fetches 10+ columns
    FROM(Shops).
    WHERE(Shops.ID.EQ(String(shopID)))

// But only uses:
return shop.AdminOnlyLists, nil  // 1 boolean field
```

**Solution**: Select only needed columns.

```go
stmt := SELECT(Shops.AdminOnlyLists).
    FROM(Shops).
    WHERE(Shops.ID.EQ(String(shopID)))

var result struct{ AdminOnlyLists bool }
err := stmt.Query(auth.db, &result)
return result.AdminOnlyLists, err
```

---

### Priority 2: Parallel Blob Deletion

**Impact**: Medium | **Effort**: Medium | **Risk**: Medium

**Problem**: When deleting a shop, blobs are deleted sequentially.

**File**: [api/shops/core/repository_impl.go](../../api/shops/core/repository_impl.go) (lines 310-358)

**Current Code**:
```go
for _, blob := range page.Segment.BlobItems {
    _, err := repo.blobClient.DeleteBlob(ctx, containerName, *blob.Name, nil)  // Sequential
    if err != nil {
        slog.Warn("Failed to delete blob", ...)
    }
}
```

**Solution**: Use bounded concurrency with errgroup.

```go
import (
    "golang.org/x/sync/errgroup"
    "sync/atomic"
)

const maxConcurrentBlobDeletes = 10

func (repo *RepositoryImpl) deleteShopBlobs(ctx context.Context, shopID string) error {
    containerName := "shop-" + shopID

    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(maxConcurrentBlobDeletes)

    var deleteCount int64
    var errorCount int64

    pager := repo.blobClient.NewListBlobsFlatPager(containerName, nil)

    for pager.More() {
        page, err := pager.NextPage(ctx)
        if err != nil {
            return fmt.Errorf("listing blobs: %w", err)
        }

        for _, blob := range page.Segment.BlobItems {
            blobName := *blob.Name  // Capture for goroutine

            g.Go(func() error {
                _, err := repo.blobClient.DeleteBlob(ctx, containerName, blobName, nil)
                if err != nil {
                    atomic.AddInt64(&errorCount, 1)
                    slog.Warn("Failed to delete blob", "blob", blobName, "error", err)
                    return nil  // Continue with other deletions
                }
                atomic.AddInt64(&deleteCount, 1)
                return nil
            })
        }
    }

    if err := g.Wait(); err != nil {
        return err
    }

    slog.Info("Blob deletion complete",
        "shop_id", shopID,
        "deleted", deleteCount,
        "errors", errorCount)
    return nil
}
```

**Expected Improvement**: 5-10x faster shop deletion for shops with many images.

---

### Priority 2: Cursor-Based Pagination for Messages

**Impact**: Medium | **Effort**: Medium | **Risk**: Low

**Problem**: OFFSET-based pagination degrades as offset increases. PostgreSQL must scan and discard `offset` rows before returning `limit` rows.

**File**: [api/shops/messages/repository_impl.go](../../api/shops/messages/repository_impl.go) (lines 74-89)

**Current Code**:
```go
stmt := SELECT(ShopMessages.AllColumns).
    FROM(ShopMessages).
    WHERE(ShopMessages.ShopID.EQ(String(shopID))).
    ORDER_BY(ShopMessages.CreatedAt.DESC()).
    OFFSET(int64(offset)).  // Gets slower as offset increases
    LIMIT(int64(limit))
```

**Solution**: Implement cursor-based (keyset) pagination as an **opt-in feature** while maintaining full backwards compatibility.

#### Backwards Compatibility Guarantee

This change is **additive only** - existing clients will continue to work without modification:

| Client Type | Sends | Receives | Behavior |
|-------------|-------|----------|----------|
| **Existing** | `page` + `limit` | Same response as today | No change required |
| **New** | `before_id` + `limit` | Same response + optional `next_cursor` | Opt-in to cursor pagination |

**Updated Request Type** (additive fields only):
```go
type GetShopMessagesPaginatedRequest struct {
    Page     int     `form:"page,default=1" binding:"omitempty,min=1"`
    Limit    int     `form:"limit,default=20" binding:"omitempty,min=1,max=100"`
    BeforeID *string `form:"before_id" binding:"omitempty"` // Load messages older than this ID
    AfterID  *string `form:"after_id" binding:"omitempty"`  // Load messages newer than this ID
}
```

**Updated Response Type** (additive field only):
```go
type PaginatedShopMessagesResponse struct {
    // EXISTING fields - unchanged
    Messages   []ShopMessage      `json:"messages"`
    Pagination *PaginationMetadata `json:"pagination,omitempty"` // Only set for offset pagination

    // NEW optional field - only populated when cursor pagination is used
    NextCursor *string `json:"next_cursor,omitempty"`
}
```

**Repository Implementation**:
```go
func (repo *RepositoryImpl) GetShopMessagesPaginated(req GetShopMessagesRequest) (*PaginatedShopMessagesResponse, error) {
    // Route to appropriate implementation based on request parameters
    if req.BeforeID != nil || req.AfterID != nil {
        // New cursor-based path (opt-in)
        return repo.getMessagesByCursor(req)
    }

    // Legacy page/offset-based path (unchanged behavior for existing clients)
    return repo.getMessagesByOffset(req)
}

func (repo *RepositoryImpl) getMessagesByCursor(req GetShopMessagesRequest) (*PaginatedShopMessagesResponse, error) {
    conditions := []jet.BoolExpression{
        ShopMessages.ShopID.EQ(String(req.ShopID)),
    }

    // Cursor-based filtering
    if req.BeforeID != nil {
        // Get the created_at timestamp for the cursor message
        cursorMsg, err := repo.GetMessageByID(*req.BeforeID)
        if err != nil {
            return nil, err
        }
        conditions = append(conditions,
            ShopMessages.CreatedAt.LT(TimestampzT(cursorMsg.CreatedAt)),
        )
    }

    stmt := SELECT(ShopMessages.AllColumns).
        FROM(ShopMessages).
        WHERE(jet.AND(conditions...)).
        ORDER_BY(ShopMessages.CreatedAt.DESC()).
        LIMIT(int64(req.Limit + 1))  // Fetch one extra to check if more exist

    var messages []model.ShopMessages
    if err := stmt.Query(repo.db, &messages); err != nil {
        return nil, err
    }

    // Check if there are more messages
    hasMore := len(messages) > req.Limit
    if hasMore {
        messages = messages[:req.Limit]
    }

    // Build response with backwards-compatible pagination metadata
    response := &PaginatedShopMessagesResponse{
        Messages: messages,
        Pagination: PaginationMetadata{
            Limit:   req.Limit,
            HasMore: hasMore,
            // Note: Total and Offset not applicable for cursor pagination
            // but fields remain in struct for compatibility
        },
    }

    // Add cursor for new clients
    if hasMore && len(messages) > 0 {
        lastMsg := messages[len(messages)-1]
        response.NextCursor = &lastMsg.ID
    }

    return response, nil
}

// getMessagesByOffset - UNCHANGED from current implementation
func (repo *RepositoryImpl) getMessagesByOffset(req GetShopMessagesRequest) (*PaginatedShopMessagesResponse, error) {
    // Existing implementation remains exactly the same
    // ...
}
```

**Expected Improvement**: Consistent O(1) pagination performance regardless of how deep into the message history the user scrolls, while maintaining full backwards compatibility for existing mobile clients.

---

### Priority 2: Pre-allocate Slices

**Impact**: Low-Medium | **Effort**: Low | **Risk**: Very Low

**Problem**: Multiple locations use `append` to grow slices without pre-allocation, causing repeated memory allocations.

**Files Affected**:
- [api/shops/core/repository_impl.go:264](../../api/shops/core/repository_impl.go)
- [api/shops/members/repository_impl.go:186](../../api/shops/members/repository_impl.go)
- [api/shops/vehicles/notifications/repository_impl.go:77](../../api/shops/vehicles/notifications/repository_impl.go)
- [api/shops/lists/items/repository_impl.go:287-290](../../api/shops/lists/items/repository_impl.go)

**Current Pattern**:
```go
var results []response.ShopWithStats
for rows.Next() {
    results = append(results, ...)  // Causes O(log n) reallocations
}
```

**Solution**: Pre-allocate when size is known or estimable.

```go
// When building from query results with known count
count, _ := getCount()  // If you have a count query
results := make([]response.ShopWithStats, 0, count)

// When transforming fixed-size input
expressions := make([]jet.Expression, len(itemIDs))
for i, id := range itemIDs {
    expressions[i] = jet.String(id)
}

// When size is unknown but bounded
results := make([]response.ShopWithStats, 0, 100)  // Reasonable initial capacity
```

---

### Priority 3: Remove No-Op Field Assignments

**Impact**: Very Low | **Effort**: Very Low | **Risk**: Very Low

**Problem**: Dead code that assigns empty string to itself.

**File**: [api/shops/vehicles/service_impl.go](../../api/shops/vehicles/service_impl.go) (lines 48-72, 153-177)

**Current Code**:
```go
if vehicle.Niin == "" {
    vehicle.Niin = ""  // Completely useless
}
if vehicle.Model == "" {
    vehicle.Model = ""  // Does nothing
}
// ... repeated for 20+ fields
```

**Solution**: Remove entirely or convert to actual default logic if intended.

```go
// If defaults were intended:
if vehicle.Niin == "" {
    vehicle.Niin = "UNKNOWN"  // Actual default value
}

// If validation was intended:
// Just remove - empty string is already empty

// If nil-safety was intended (for pointers):
if vehicle.Niin == nil {
    empty := ""
    vehicle.Niin = &empty
}
```

---

## Implementation Summary

| Priority | Change | Effort | Impact | Risk |
|----------|--------|--------|--------|------|
| **P0** | Database indexes | 30 min | Very High | Low |
| **P0** | Fix N+1 query | 2 hr | High | Medium |
| **P0** | Authorization caching | 2 hr | High | Low |
| **P1** | EXISTS vs COUNT | 1 hr | Medium-High | Low |
| **P1** | Optimize admin_check | 30 min | Medium | Low |
| **P1** | Context timeouts | 30 min | Medium | Low |
| **P1** | Select only needed columns | 30 min | Low-Medium | Low |
| **P2** | Parallel blob deletion | 1.5 hr | Medium | Medium |
| **P2** | Cursor pagination | 2 hr | Medium | Low |
| **P2** | Pre-allocate slices | 1 hr | Low-Medium | Very Low |
| **P3** | Remove no-op code | 15 min | Very Low | Very Low |

**Total Estimated Effort**: ~12 hours

---

## Expected Performance Improvement

| Metric | Current | After Optimization |
|--------|---------|-------------------|
| Authorization queries per request | 4-8 | 1-2 (cached) |
| GetNotificationsWithItems queries | N+1 | 2 |
| Message pagination at offset 1000 | Scans 1000+ rows | Scans limit rows |
| Shop deletion (100 images) | ~10s sequential | ~1-2s parallel |
| Index-supported query performance | Full table scans | Index seeks |

**Estimated overall improvement**: **3-10x faster** for typical shop operations.

---

## Testing Recommendations

### Load Testing Scenarios

1. **Authorization stress test**: 100 concurrent requests checking membership
2. **Deep pagination test**: Request page 100 of messages
3. **Large shop test**: Shop with 50 vehicles, 200 notifications, 1000 items
4. **Blob deletion test**: Delete shop with 500 images

### Query Performance Verification

```sql
-- Verify index usage with EXPLAIN ANALYZE
EXPLAIN ANALYZE
SELECT * FROM shop_members
WHERE shop_id = 'test-shop-id' AND user_id = 'test-user-id';

-- Should show: Index Scan using idx_shop_members_shop_user
-- Should NOT show: Seq Scan
```

---

## API Backwards Compatibility

### Compatibility Guarantee

**All optimizations in this document are backwards compatible.** Existing mobile and web clients will continue to work without any changes.

| Optimization | API Impact | Client Action Required |
|--------------|------------|------------------------|
| Database indexes | None | None |
| N+1 query fix | None | None |
| Authorization caching | None | None |
| EXISTS vs COUNT | None | None |
| Optimize admin_check | None | None |
| Context timeouts | None | None |
| Select fewer columns | None | None |
| Parallel blob deletion | None | None |
| **Cursor pagination** | **Additive only** | **None (opt-in for new clients)** |
| Pre-allocate slices | None | None |
| Remove no-op code | None | None |

### Cursor Pagination Compatibility Details

The cursor-based pagination is implemented as an **opt-in feature**:

1. **Existing clients** continue sending `page` and receive the exact same response structure
2. **New clients** can opt-in by sending `before_id` instead of `page`
3. **Response structure** adds one optional field (`next_cursor`) that existing clients can safely ignore
4. **No breaking changes** to any existing endpoint contracts

---

## Questions for Implementation

1. **Index Creation Timing**: Should indexes be created during a maintenance window, or is `CONCURRENTLY` sufficient for production?

2. **Cache TTL**: For request-scoped caching, the cache is automatically cleared per-request. Do we also need cross-request caching (Redis) for high-traffic shops?

3. **Monitoring**: Should we add query timing metrics to identify slow queries post-optimization?

4. **Blob Deletion Strategy**: For very large shops, should blob deletion be moved to a background job instead of blocking the DELETE request?

---

## References

- [PostgreSQL Index Types](https://www.postgresql.org/docs/current/indexes-types.html)
- [Go errgroup documentation](https://pkg.go.dev/golang.org/x/sync/errgroup)
- [Keyset Pagination Explained](https://use-the-index-luke.com/no-offset)
- [Azure Blob Storage Go SDK](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/storage/azblob)
