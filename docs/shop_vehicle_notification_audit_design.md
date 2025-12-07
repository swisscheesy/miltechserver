# Shop Vehicle Notification Change Tracking - Design Plan

## Overview
This document outlines the design for implementing comprehensive change tracking (audit trail) for shop vehicle notifications. This feature will track all modifications made to vehicle notifications, including who made the change, when it was made, and what was changed.

## Current System Analysis

### Existing Database Schema
The `shop_vehicle_notifications` table currently has:
- **Primary Key**: `id` (text/UUID)
- **Foreign Keys**:
  - `shop_id` → `shops(id)` (CASCADE DELETE/UPDATE)
  - `vehicle_id` → `shop_vehicle(id)` (CASCADE DELETE/UPDATE)
- **Data Fields**: `title`, `description`, `type`, `completed`
- **Timestamps**: `save_time` (creation), `last_updated` (modification)
- **Indexes**: On `shop_id`, `vehicle_id`, `completed`, `type`

### Current Update Flow
1. **Controller** (`shops_controller.go:966-999`): Receives update request from authenticated user
2. **Service** (`shops_service_impl.go`): Validates permissions (shop membership)
3. **Repository** (`shops_repository_impl.go:880-910`): Executes UPDATE statement
   - Updates: `title`, `description`, `type`, `completed`, `last_updated`
   - No tracking of who made the change
   - No tracking of what changed (old vs new values)

### Gap Analysis
**Missing Functionality**:
- ❌ No record of who made changes
- ❌ No record of what was changed (field-level tracking)
- ❌ No audit trail for compliance or dispute resolution
- ❌ No way to see notification history over time
- ❌ No tracking of item additions/removals

## Design Solution

### 1. New Database Table: `shop_vehicle_notification_changes`

This table will store a complete audit trail of all changes made to vehicle notifications.

```sql
CREATE TABLE shop_vehicle_notification_changes (
    -- Primary identification
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,

    -- What was changed
    notification_id TEXT NOT NULL,
    shop_id TEXT NOT NULL,
    vehicle_id TEXT NOT NULL,

    -- Who and when
    changed_by TEXT NOT NULL,  -- user_id of person who made the change
    changed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Type of change
    change_type TEXT NOT NULL,  -- 'create', 'update', 'delete', 'complete', 'reopen', 'items_added', 'items_removed'

    -- What changed (JSONB for flexibility)
    field_changes JSONB NOT NULL,  -- Stores fields that changed

    -- Foreign keys
    CONSTRAINT fk_notification
        FOREIGN KEY (notification_id)
        REFERENCES shop_vehicle_notifications(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_shop
        FOREIGN KEY (shop_id)
        REFERENCES shops(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_vehicle
        FOREIGN KEY (vehicle_id)
        REFERENCES shop_vehicle(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_changed_by
        FOREIGN KEY (changed_by)
        REFERENCES users(uid)
        ON DELETE SET NULL
);

-- Indexes optimized for common query patterns
-- Single notification history (most common query)
CREATE INDEX idx_notification_changes_notification_id
    ON shop_vehicle_notification_changes(notification_id);

-- Shop-wide recent changes (ordered by time)
CREATE INDEX idx_notification_changes_shop_changes
    ON shop_vehicle_notification_changes(shop_id, changed_at DESC);

-- Vehicle-specific changes (ordered by time)
CREATE INDEX idx_notification_changes_vehicle_changes
    ON shop_vehicle_notification_changes(vehicle_id, changed_at DESC);

-- Index for querying field changes (GIN index for JSONB)
CREATE INDEX idx_notification_changes_field_changes
    ON shop_vehicle_notification_changes USING GIN(field_changes);

-- Optional: For filtering specific change types per notification
-- CREATE INDEX idx_notification_changes_notification_type
--     ON shop_vehicle_notification_changes(notification_id, change_type);
```

### 2. JSONB Field Changes Structure

The `field_changes` JSONB column will store a simple array of fields that changed (no old/new values):

```json
{
  "fields_changed": ["title", "description"]
}
```

For completed status change (change_type will be "complete" or "reopen"):
```json
{
  "fields_changed": ["completed"]
}
```

For creation events:
```json
{
  "fields_changed": ["created"]
}
```

For item-related changes:
```json
{
  "fields_changed": ["items"],
  "item_count": 3
}
```

**Note**: The `change_type` field indicates the action (e.g., "items_added" vs "items_removed"), so no need to duplicate that information in the JSONB. The `item_count` field indicates whether it was a single item (1) or bulk operation (>1).

### 3. Item Changes - Integrated Approach

Item additions/removals will be tracked in the same `shop_vehicle_notification_changes` table using these change types:
- `items_added` - When items are added to a notification (single or bulk)
- `items_removed` - When items are removed from a notification (single or bulk)

The `field_changes` JSONB will include the count:
```json
{
  "fields_changed": ["items"],
  "item_count": 2
}
```

The `change_type` differentiates between additions and removals, while `item_count` indicates how many items were affected. This approach:
- Simplifies the change type enum (2 types instead of 4)
- Keeps all notification-related changes in one table for easier querying
- Provides the same information with less complexity

## Implementation Plan

### Phase 1: Database Schema
1. Create migration file: `003_create_notification_audit_tables.sql`
2. Create rollback file: `003_rollback_notification_audit_tables.sql`
3. Add Go model generation for new tables
4. Update `.gen/miltech_ng/public/model/` with new structs

### Phase 2: Repository Layer Updates

**File**: `api/repository/shops_repository.go`

Add new interface methods:
```go
// Notification Change Tracking (includes item changes)
CreateNotificationChange(user *bootstrap.User, change model.ShopVehicleNotificationChanges) error
GetNotificationChanges(user *bootstrap.User, notificationID string) ([]response.NotificationChangeWithUsername, error)
GetNotificationChangesByShop(user *bootstrap.User, shopID string, limit int) ([]response.NotificationChangeWithUsername, error)
GetNotificationChangesByVehicle(user *bootstrap.User, vehicleID string) ([]response.NotificationChangeWithUsername, error)
```

**File**: `api/repository/shops_repository_impl.go`

Implement methods with proper JOINs to include usernames:
```go
func (repo *ShopsRepositoryImpl) CreateNotificationChange(
    user *bootstrap.User,
    change model.ShopVehicleNotificationChanges,
) error {
    // Insert change record with all field tracking
}

func (repo *ShopsRepositoryImpl) GetNotificationChanges(
    user *bootstrap.User,
    notificationID string,
) ([]response.NotificationChangeWithUsername, error) {
    // SELECT with JOIN to users table to get username
    // ORDER BY changed_at DESC
}
```

### Phase 3: Service Layer Updates

**File**: `api/service/shops_service.go`

Add new interface methods:
```go
// Get change history
GetNotificationChangeHistory(user *bootstrap.User, notificationID string) ([]response.NotificationChangeWithUsername, error)
GetShopNotificationChanges(user *bootstrap.User, shopID string, limit int) ([]response.NotificationChangeWithUsername, error)
```

**File**: `api/service/shops_service_impl.go`

Modify existing methods to record changes:

```go
func (service *ShopsServiceImpl) UpdateVehicleNotification(
    user *bootstrap.User,
    notification model.ShopVehicleNotifications,
) error {
    // 1. Get current notification state
    currentNotification, err := service.ShopsRepository.GetVehicleNotificationByID(user, notification.ID)
    if err != nil {
        return err
    }

    // 2. Build field changes JSONB
    fieldChanges, err := buildFieldChanges(currentNotification, &notification)
    if err != nil {
        slog.Warn("Failed to build field changes", "error", err)
        fieldChanges = `{"fields_changed": []}`
    }

    // 3. Update the notification
    err = service.ShopsRepository.UpdateVehicleNotification(user, notification)
    if err != nil {
        return err
    }

    // 4. Record the change (optional - won't fail the update)
    change := model.ShopVehicleNotificationChanges{
        NotificationID: notification.ID,
        ShopID: currentNotification.ShopID,
        VehicleID: currentNotification.VehicleID,
        ChangedBy: user.UserID,
        ChangeType: determineChangeType(currentNotification, &notification),
        FieldChanges: fieldChanges,
    }

    err = service.ShopsRepository.CreateNotificationChange(user, change)
    if err != nil {
        slog.Warn("Failed to record notification change", "error", err)
        // Audit logging is best-effort - don't fail the update if it fails
        // Consider implementing a background job queue for eventual consistency
    }

    return nil
}
```

Add helper functions:
```go
func buildFieldChanges(old, new *model.ShopVehicleNotifications) (string, error) {
    changedFields := []string{}
    changeData := make(map[string]interface{})

    // Handle nullable fields with NULL-safe comparison
    if old.Title != new.Title {
        changedFields = append(changedFields, "title")
    }

    // Check both value changes and NULL transitions for nullable fields
    if old.Description != new.Description {
        changedFields = append(changedFields, "description")
    }

    if old.Type != new.Type {
        changedFields = append(changedFields, "type")
    }

    if old.Completed != new.Completed {
        changedFields = append(changedFields, "completed")
    }

    changeData["fields_changed"] = changedFields

    jsonBytes, err := json.Marshal(changeData)
    if err != nil {
        return "{}", fmt.Errorf("failed to marshal field changes: %w", err)
    }
    return string(jsonBytes), nil
}

func determineChangeType(old, new *model.ShopVehicleNotifications) string {
    if !old.Completed && new.Completed {
        return "complete"
    }
    if old.Completed && !new.Completed {
        return "reopen"
    }
    return "update"
}
```

Similarly update:
- `CreateVehicleNotification` → record "create" change
- `DeleteVehicleNotification` → record "delete" change
- `AddNotificationItem` → record "items_added" change with item_count: 1
- `AddNotificationItemList` → record "items_added" change with item_count: N
- `RemoveNotificationItem` → record "items_removed" change with item_count: 1
- `RemoveNotificationItemList` → record "items_removed" change with item_count: N

### Phase 4: Controller Layer Updates

**File**: `api/controller/shops_controller.go`

Add new endpoints:
```go
// GetNotificationChangeHistory returns change history for a specific notification
func (controller *ShopsController) GetNotificationChangeHistory(c *gin.Context) {
    ctxUser, ok := c.Get("user")
    user, _ := ctxUser.(*bootstrap.User)

    if !ok {
        c.JSON(401, gin.H{"message": "unauthorized"})
        return
    }

    notificationID := c.Param("notification_id")
    if notificationID == "" {
        c.JSON(400, gin.H{"message": "notification_id is required"})
        return
    }

    changes, err := controller.ShopsService.GetNotificationChangeHistory(user, notificationID)
    if err != nil {
        c.Error(err)
        return
    }

    c.JSON(200, response.StandardResponse{
        Status:  200,
        Message: "",
        Data:    changes,
    })
}

// GetShopNotificationChanges returns recent changes for all notifications in a shop
func (controller *ShopsController) GetShopNotificationChanges(c *gin.Context) {
    // Implementation with optional limit query parameter
}
```

### Phase 5: Response Models

**File**: `api/response/notification_changes_response.go` (new file)

```go
package response

import "time"

type NotificationChangeWithUsername struct {
    ID                string                 `json:"id"`
    NotificationID    string                 `json:"notification_id"`
    ShopID            string                 `json:"shop_id"`
    VehicleID         string                 `json:"vehicle_id"`
    ChangedBy         string                 `json:"changed_by"`
    ChangedByUsername string                 `json:"changed_by_username"`
    ChangedAt         time.Time              `json:"changed_at"`
    ChangeType        string                 `json:"change_type"`
    FieldChanges      map[string]interface{} `json:"field_changes"`
}

// Item changes are included in NotificationChangeWithUsername using change_type
// Examples:
// - change_type: "items_added", field_changes: {"fields_changed": ["items"], "item_count": 1}
// - change_type: "items_removed", field_changes: {"fields_changed": ["items"], "item_count": 5}
```

### Phase 6: Route Registration

**File**: `api/route/shops_route.go`

Add new routes:
```go
// Notification change history routes
group.GET("/shops/notifications/:notification_id/changes", pc.GetNotificationChangeHistory)
group.GET("/shops/:shop_id/notifications/changes", pc.GetShopNotificationChanges)
group.GET("/shops/vehicles/:vehicle_id/notifications/changes", pc.GetVehicleNotificationChanges)
```

## Data Flow

### Update Notification Flow (with Audit)
```
1. User submits update via API
   ↓
2. Controller validates request
   ↓
3. Service layer:
   a. Fetches current notification state
   b. Validates user permissions
   c. Builds field changes map
   ↓
4. Repository layer:
   a. UPDATE shop_vehicle_notifications
   b. INSERT INTO shop_vehicle_notification_changes (best-effort)
      - If audit insert fails, log warning but don't fail the update
      - Ensures updates always succeed even if audit system has issues
   ↓
5. Return success response

Note: Audit logging is best-effort to prevent audit system failures from
blocking critical business operations. For strict consistency, wrap both
operations in a transaction and fail both if either fails.
```

### Query Change History Flow
```
1. User requests change history
   ↓
2. Controller validates notification access
   ↓
3. Service validates shop membership
   ↓
4. Repository executes:
   SELECT c.*, u.username
   FROM shop_vehicle_notification_changes c
   JOIN users u ON c.changed_by = u.user_id
   WHERE c.notification_id = $1
   ORDER BY c.changed_at DESC
   ↓
5. Return formatted response with usernames
```

## Security & Access Control

### Permissions Model
- **View change history**: Any shop member can view changes for notifications in their shop
- **Changes are immutable**: Once recorded, audit records cannot be modified or deleted (except via CASCADE when parent notification is deleted)
- **User deletion handling**: Use `ON DELETE SET NULL` for `changed_by` foreign key to preserve audit trail even if user is deleted

### Data Retention
**Default**: Indefinite retention
- Changes are automatically deleted when parent notification is deleted (CASCADE)
- Changes are automatically deleted when parent shop is deleted (CASCADE)
- For compliance, changes persist even if user who made them is deleted (SET NULL)

**Optional Future Enhancement**: Add retention policy
```sql
-- Example: Delete changes older than 2 years
DELETE FROM shop_vehicle_notification_changes
WHERE changed_at < NOW() - INTERVAL '2 years';
```

## Performance Considerations

### Write Performance
- Each notification update adds one INSERT to audit table
- Impact: Minimal (single row insert with indexes)
- Transaction ensures atomicity

### Read Performance
- Indexes on `notification_id`, `shop_id`, `vehicle_id` enable fast lookups
- GIN index on JSONB enables searching within field changes
- Limit query results (default 100) to prevent large data transfers

### Storage Impact
- Estimated row size: ~500 bytes per change record
- 10,000 changes = ~5 MB (negligible)
- JSONB is compressed by PostgreSQL automatically

## Query Examples

### Get all changes for a notification
```sql
SELECT
    c.id,
    c.notification_id,
    c.changed_by,
    u.username as changed_by_username,
    c.changed_at,
    c.change_type,
    c.field_changes
FROM shop_vehicle_notification_changes c
JOIN users u ON c.changed_by = u.user_id
WHERE c.notification_id = 'notification-uuid'
ORDER BY c.changed_at DESC;
```

### Get recent changes for a shop
```sql
SELECT
    c.id,
    c.notification_id,
    svn.title as notification_title,
    sv.model as vehicle_model,
    c.changed_by,
    u.username as changed_by_username,
    c.changed_at,
    c.change_type,
    c.field_changes
FROM shop_vehicle_notification_changes c
JOIN users u ON c.changed_by = u.user_id
JOIN shop_vehicle_notifications svn ON c.notification_id = svn.id
JOIN shop_vehicle sv ON c.vehicle_id = sv.id
WHERE c.shop_id = 'shop-uuid'
ORDER BY c.changed_at DESC
LIMIT 50;
```

### Find who completed a notification
```sql
SELECT
    c.changed_by,
    u.username,
    c.changed_at
FROM shop_vehicle_notification_changes c
JOIN users u ON c.changed_by = u.user_id
WHERE c.notification_id = 'notification-uuid'
  AND c.change_type = 'complete'
ORDER BY c.changed_at DESC
LIMIT 1;
```

### Find when specific fields were changed
```sql
SELECT
    c.id,
    c.notification_id,
    c.changed_by,
    u.username,
    c.changed_at,
    c.field_changes
FROM shop_vehicle_notification_changes c
JOIN users u ON c.changed_by = u.user_id
WHERE c.shop_id = 'shop-uuid'
  AND c.field_changes @> '{"fields_changed": ["title"]}'  -- Title was changed
ORDER BY c.changed_at DESC;
```

### Get all item addition/removal events
```sql
SELECT
    c.id,
    c.notification_id,
    svn.title,
    c.changed_by,
    u.username,
    c.changed_at,
    c.change_type,
    c.field_changes->>'item_count' as item_count
FROM shop_vehicle_notification_changes c
JOIN users u ON c.changed_by = u.user_id
JOIN shop_vehicle_notifications svn ON c.notification_id = svn.id
WHERE c.shop_id = 'shop-uuid'
  AND c.change_type IN ('items_added', 'items_removed')
ORDER BY c.changed_at DESC;
```

## Testing Strategy

### Unit Tests
- Test field change detection logic
- Test JSONB serialization/deserialization
- Test change type determination

### Integration Tests
- Test full update flow with audit creation
- Test querying change history
- Test cascade deletions
- Test transaction rollback scenarios

### Performance Tests
- Benchmark write performance with audit logging
- Test query performance with 10k, 100k change records
- Verify index effectiveness

## Migration Strategy

### Initial Deployment
1. Create tables (no data migration needed - fresh start)
2. Deploy code changes
3. All future changes will be tracked automatically

### Backfilling Historical Data
**Not recommended** - No way to retroactively determine who made changes or when specific fields changed

Alternative: Add a one-time "migration" change record for existing notifications:
```sql
INSERT INTO shop_vehicle_notification_changes (
    notification_id, shop_id, vehicle_id, changed_by, changed_at,
    change_type, field_changes
)
SELECT
    id, shop_id, vehicle_id, 'SYSTEM', save_time,
    'create',
    jsonb_build_object(
        'created', true,
        'note', 'Migrated from legacy system'
    )
FROM shop_vehicle_notifications;
```

## Future Enhancements

### Possible Extensions
1. **Change notifications**: Email/push notifications when high-priority items change
2. **Restore functionality**: Ability to revert to previous versions
3. **Audit reports**: Generate compliance reports showing all changes in date range
4. **Change approval workflow**: Require admin approval for certain changes
5. **Bulk change tracking**: Track multiple related changes as a single "session"
6. **Change comments**: Allow users to add notes explaining why they made changes

## API Endpoints Summary

### New Endpoints
```
GET  /api/v1/shops/notifications/:notification_id/changes
     - Get change history for specific notification
     - Returns: Array of NotificationChangeWithUsername

GET  /api/v1/shops/:shop_id/notifications/changes?limit=100
     - Get recent changes for all notifications in shop
     - Query params: limit (default 100, max 500)
     - Returns: Array of NotificationChangeWithUsername

GET  /api/v1/shops/vehicles/:vehicle_id/notifications/changes
     - Get changes for all notifications on a vehicle
     - Returns: Array of NotificationChangeWithUsername
```

### Modified Endpoints (behavior change)
All existing notification endpoints will continue to work identically, but will now silently record changes:
- `POST /api/v1/shops/notifications` - Records "create" change
- `PUT /api/v1/shops/notifications/:notification_id` - Records "update" or "complete" change
- `DELETE /api/v1/shops/notifications/:notification_id` - Records "delete" change
- `POST /api/v1/shops/notifications/:notification_id/items` - Records "item_added" change
- `DELETE /api/v1/shops/notifications/items/:item_id` - Records "item_removed" change

## Documentation Requirements

### Files to Create/Update
1. **Migration files**:
   - `migrations/003_create_notification_audit_tables.sql`
   - `migrations/003_rollback_notification_audit_tables.sql`

2. **Code documentation**:
   - Update `docs/shop_routes.md` with new endpoints
   - Add inline comments explaining audit logic

3. **API documentation**:
   - Document new response types
   - Document field_changes JSONB structure
   - Provide example responses

## Summary

This design provides:
- ✅ Complete audit trail of all notification changes
- ✅ Tracks who made each change and when
- ✅ Records which fields changed (without storing old/new values for simplicity)
- ✅ Integrated tracking for notification items (additions/removals)
- ✅ Single unified table for all change types
- ✅ Efficient querying with proper indexes
- ✅ Minimal performance impact
- ✅ Follows existing architectural patterns
- ✅ Secure with proper access control (all shop members can view)
- ✅ Indefinite retention (cascade delete with parent records)
- ✅ Scalable JSONB structure for flexibility

The implementation maintains the clean architecture pattern used throughout the codebase, with clear separation of concerns across controller, service, and repository layers.

## Configuration Summary (Based on Requirements)

**Confirmed Settings**:
1. ✅ **Tracking scope**: Both notification field changes AND item additions/removals
2. ✅ **Detail level**: Just "what changed" (field names only, no old/new values)
3. ✅ **Retention**: Forever (cascade delete when parent notification is deleted)
4. ✅ **Access control**: All shop members can view change history
5. ✅ **Item tracking**: Track that items were added/removed with count, but not detailed item information

**Change Types Supported**:
- `create` - Notification created
- `update` - Notification fields updated
- `delete` - Notification deleted
- `complete` - Notification marked complete
- `reopen` - Notification reopened
- `items_added` - Items added (single or bulk, check `item_count` in field_changes)
- `items_removed` - Items removed (single or bulk, check `item_count` in field_changes)
