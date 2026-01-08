# Design Document: Vehicle and Notification Deletion Audit Trail Fix

**Date:** 2026-01-01
**Author:** Development Team
**Status:** ✅ Database Migration Complete - Code Implementation In Progress
**Last Updated:** 2026-01-01

## Executive Summary

This document outlines the design to fix critical audit trail gaps in the shop vehicle notification system. Currently, when vehicles or notifications are deleted, the audit trail data is CASCADE deleted due to foreign key constraints, defeating the purpose of maintaining historical records. Additionally, vehicle deletions are not audited at all.

**Solution:** Implement denormalized audit data storage with nullable foreign keys and `ON DELETE SET NULL` constraints to preserve historical context after entity deletion.

### ✅ Migration Status

**Database Changes:** ✅ **COMPLETED** (2026-01-01)
- Added denormalized columns: `notification_title`, `notification_type`, `vehicle_admin`
- Changed `notification_id` and `vehicle_id` to nullable
- Updated FK constraints from `ON DELETE CASCADE` to `ON DELETE SET NULL`
- Dropped NOT NULL check constraints

**Code Changes:** 🚧 **IN PROGRESS**
- See [Implementation Checklist](#implementation-checklist) section for detailed status

### Next Steps

Now that the database migration is complete, the following code changes are required:

1. **Regenerate Database Models** - Update Go models to reflect new nullable columns
2. **Update Audit Recording** - Modify `recordNotificationChange` to accept and store denormalized data
3. **Implement Vehicle Deletion Audit** - Add audit trail for vehicle deletions (currently missing)
4. **Update Query Functions** - Modify queries to use LEFT JOIN and handle NULL foreign keys
5. **Update Response Models** - Add nullable fields and `IsDeleted` computed property
6. **Manual Testing** - Complete manual testing checklist to verify functionality

See the [Code Changes](#code-changes) section for detailed implementation guidance.

---

## Problem Statement

### Current Issues

1. **Notification Deletion Audit Loss**
   - The `shop_vehicle_notification_changes` table has `ON DELETE CASCADE` constraint on `notification_id`
   - When a notification is deleted, ALL audit records for that notification are CASCADE deleted
   - Code at [shops_service_impl.go:1165-1173](api/service/shops_service_impl.go#L1165-L1173) attempts to record deletion, but the record is immediately destroyed by CASCADE
   - Database query confirms: NO "delete" change_type records exist in production

2. **Vehicle Deletion Not Audited**
   - Vehicle deletion at [shops_service_impl.go:884-913](api/service/shops_service_impl.go#L884-L913) has **no audit trail** implementation
   - When a vehicle is deleted, all associated notification audit records are CASCADE deleted via `vehicle_id` FK
   - No record of what vehicle was deleted or when

3. **Cascading Data Loss**
   - `shop_notification_items` also has `ON DELETE CASCADE` on `notification_id`
   - When notifications are deleted, all items are CASCADE deleted
   - Existing audit records for `items_added` and `items_removed` lose referential context

### Current Database Constraints

```sql
-- Current problematic constraints
ALTER TABLE shop_vehicle_notification_changes
  ADD CONSTRAINT fk_notification
    FOREIGN KEY (notification_id)
    REFERENCES shop_vehicle_notifications(id)
    ON DELETE CASCADE;  -- ❌ Destroys audit history

ALTER TABLE shop_vehicle_notification_changes
  ADD CONSTRAINT fk_vehicle
    FOREIGN KEY (vehicle_id)
    REFERENCES shop_vehicle(id)
    ON DELETE CASCADE;  -- ❌ Destroys audit history
```

### Current Change Types in System

```
- create
- update
- complete
- reopen
- items_added
- items_removed
- delete (attempted but CASCADE deleted before persistence)
```

---

## Design Solution

### Approach: Denormalized Audit Data with Nullable Foreign Keys

**Core Principle:** Audit tables must capture "what it looked like at the time" rather than relying on foreign keys that might not exist later.

### Database Schema Changes

#### 1. Add Denormalized Columns

Add columns to `shop_vehicle_notification_changes` to store entity data at the time of the change:

```sql
ALTER TABLE shop_vehicle_notification_changes
  ADD COLUMN notification_title TEXT,           -- Title at time of change
  ADD COLUMN notification_type TEXT,            -- Type at time of change
  ADD COLUMN vehicle_admin TEXT;                -- Vehicle admin number at time of change
```

**Rationale:**
- When `notification_id` becomes NULL (after deletion), we still have `notification_title` and `notification_type`
- When `vehicle_id` becomes NULL (after deletion), we still have `vehicle_admin` to identify the vehicle
- These fields are populated for ALL change records, not just deletions
- Enables meaningful audit queries like "Show all changes for notification 'Annual PMCS'" even after notification is deleted

#### 2. Make Foreign Keys Nullable

```sql
ALTER TABLE shop_vehicle_notification_changes
  ALTER COLUMN notification_id DROP NOT NULL,
  ALTER COLUMN vehicle_id DROP NOT NULL;
```

**Rationale:**
- Allows audit records to survive entity deletion
- NULL value indicates the referenced entity has been deleted
- Maintains data integrity for existing (non-deleted) entities

#### 3. Replace Foreign Key Constraints

```sql
-- Drop existing CASCADE constraints
ALTER TABLE shop_vehicle_notification_changes
  DROP CONSTRAINT fk_notification,
  DROP CONSTRAINT fk_vehicle;

-- Add new SET NULL constraints
ALTER TABLE shop_vehicle_notification_changes
  ADD CONSTRAINT fk_notification
    FOREIGN KEY (notification_id)
    REFERENCES shop_vehicle_notifications(id)
    ON DELETE SET NULL,  -- ✅ Preserves audit record, sets FK to NULL

  ADD CONSTRAINT fk_vehicle
    FOREIGN KEY (vehicle_id)
    REFERENCES shop_vehicle(id)
    ON DELETE SET NULL;  -- ✅ Preserves audit record, sets FK to NULL
```

**Rationale:**
- `ON DELETE SET NULL` preserves the audit record while marking the entity as deleted
- Maintains referential integrity for active entities
- Follows PostgreSQL best practices for audit trail tables

#### 4. Update Check Constraints

Remove NOT NULL check constraints for the newly nullable columns:

```sql
-- These constraints are auto-generated and need to be dropped
ALTER TABLE shop_vehicle_notification_changes
  DROP CONSTRAINT IF EXISTS "2200_44453_2_not_null",  -- notification_id NOT NULL
  DROP CONSTRAINT IF EXISTS "2200_44453_4_not_null";  -- vehicle_id NOT NULL
```

---

## Code Changes

### 1. Update `recordNotificationChange` Function Signature

**File:** [api/service/shops_service_impl.go](api/service/shops_service_impl.go#L2083)

**Current:**
```go
func (service *ShopsServiceImpl) recordNotificationChange(
    user *bootstrap.User,
    notificationID string,
    shopID string,
    vehicleID string,
    changeType string,
    fieldChanges string,
)
```

**New:**
```go
func (service *ShopsServiceImpl) recordNotificationChange(
    user *bootstrap.User,
    notificationID string,
    shopID string,
    vehicleID string,
    changeType string,
    fieldChanges string,
    notificationTitle string,    // NEW: denormalized
    notificationType string,     // NEW: denormalized
    vehicleAdmin string,          // NEW: denormalized
)
```

### 2. Update Model Structure

**File:** `.gen/miltech_ng/public/model/shop_vehicle_notification_changes.go`

Add fields to the generated model (will be regenerated from schema):

```go
type ShopVehicleNotificationChanges struct {
    ID               string
    NotificationID   *string  // Changed to pointer (nullable)
    ShopID           string
    VehicleID        *string  // Changed to pointer (nullable)
    ChangedBy        *string
    ChangedAt        time.Time
    ChangeType       string
    FieldChanges     string
    NotificationTitle *string  // NEW: nullable, populated for all records
    NotificationType  *string  // NEW: nullable, populated for all records
    VehicleAdmin      *string  // NEW: nullable, populated for all records
}
```

### 3. Update All Calls to `recordNotificationChange`

**Locations to update:**
- `CreateVehicleNotification` - pass notification data
- `UpdateVehicleNotification` - pass notification data
- `CompleteVehicleNotification` - pass notification data
- `ReopenVehicleNotification` - pass notification data
- `DeleteVehicleNotification` - pass notification data before deletion
- `AddNotificationItem` - pass notification data
- `RemoveNotificationItem` - pass notification data
- `RemoveNotificationItemList` - pass notification data

**Example pattern:**
```go
// OLD
service.recordNotificationChange(
    user,
    notification.ID,
    notification.ShopID,
    notification.VehicleID,
    "update",
    fieldChanges,
)

// NEW
service.recordNotificationChange(
    user,
    notification.ID,
    notification.ShopID,
    notification.VehicleID,
    "update",
    fieldChanges,
    notification.Title,      // denormalized
    notification.Type,       // denormalized
    vehicle.Admin,           // denormalized
)
```

### 4. Add Vehicle Deletion Audit Trail

**File:** [api/service/shops_service_impl.go](api/service/shops_service_impl.go#L884)

**Current implementation (NO AUDIT):**
```go
func (service *ShopsServiceImpl) DeleteShopVehicle(user *bootstrap.User, vehicleID string) error {
    // ... permission checks ...

    err = service.ShopsRepository.DeleteShopVehicle(user, vehicleID)
    if err != nil {
        return fmt.Errorf("failed to delete shop vehicle: %w", err)
    }

    slog.Info("Shop vehicle deleted", "user_id", user.UserID, "vehicle_id", vehicleID)
    return nil
}
```

**New implementation (WITH AUDIT):**
```go
func (service *ShopsServiceImpl) DeleteShopVehicle(user *bootstrap.User, vehicleID string) error {
    if user == nil {
        return errors.New("unauthorized user")
    }

    // Get vehicle to check permissions AND capture audit data
    vehicle, err := service.ShopsRepository.GetShopVehicleByID(user, vehicleID)
    if err != nil {
        return fmt.Errorf("failed to get vehicle: %w", err)
    }

    // Check if user is vehicle creator OR shop admin
    isCreator := vehicle.CreatorID == user.UserID
    isAdmin, err := service.ShopsRepository.IsUserShopAdmin(user, vehicle.ShopID)
    if err != nil {
        return fmt.Errorf("failed to verify admin status: %w", err)
    }

    if !isCreator && !isAdmin {
        return errors.New("access denied: only vehicle creator or shop admin can delete vehicles")
    }

    // Record vehicle deletion in audit trail
    // Create a synthetic notification change to record the vehicle deletion
    vehicleDeletionChange := model.ShopVehicleNotificationChanges{
        NotificationID:    nil,  // NULL - no specific notification
        ShopID:            vehicle.ShopID,
        VehicleID:         &vehicleID,  // Will be set to NULL after delete
        ChangedBy:         &user.UserID,
        ChangeType:        "vehicle_deleted",
        FieldChanges:      buildVehicleDeletionFieldChanges(vehicle),
        NotificationTitle: nil,  // NULL - not applicable
        NotificationType:  nil,  // NULL - not applicable
        VehicleAdmin:      &vehicle.Admin,
    }

    // Best-effort audit recording
    err = service.ShopsRepository.CreateNotificationChange(user, vehicleDeletionChange)
    if err != nil {
        slog.Warn("Failed to record vehicle deletion audit", "error", err, "vehicle_id", vehicleID)
    }

    // Perform the actual deletion
    err = service.ShopsRepository.DeleteShopVehicle(user, vehicleID)
    if err != nil {
        return fmt.Errorf("failed to delete shop vehicle: %w", err)
    }

    slog.Info("Shop vehicle deleted", "user_id", user.UserID, "vehicle_id", vehicleID, "vehicle_admin", vehicle.Admin)
    return nil
}
```

### 5. Add Helper Function for Vehicle Deletion Field Changes

```go
// buildVehicleDeletionFieldChanges creates field changes JSON for vehicle deletion
func buildVehicleDeletionFieldChanges(vehicle *model.ShopVehicle) string {
    changes := map[string]interface{}{
        "deleted": true,
        "vehicle_data": map[string]interface{}{
            "admin":    vehicle.Admin,
            "niin":     vehicle.Niin,
            "uoc":      vehicle.Uoc,
            "mileage":  vehicle.Mileage,
            "hours":    vehicle.Hours,
            "comment":  vehicle.Comment,
        },
    }

    jsonBytes, err := json.Marshal(changes)
    if err != nil {
        slog.Warn("Failed to marshal vehicle deletion field changes", "error", err)
        return `{"deleted": true}`
    }

    return string(jsonBytes)
}
```

### 6. Update Query Functions to Handle NULL Foreign Keys

**File:** [api/repository/shops_repository_impl.go](api/repository/shops_repository_impl.go)

Update the query that retrieves notification changes to handle NULL `notification_id`:

```go
func (repo *ShopsRepositoryImpl) GetNotificationChanges(user *bootstrap.User, notificationID string, limit int) ([]response.NotificationChangeResponse, error) {
    stmt := ShopVehicleNotificationChanges.
        SELECT(
            ShopVehicleNotificationChanges.AllColumns,
            Users.Username,
            // Use COALESCE to handle deleted notifications
            Raw("COALESCE(shop_vehicle_notifications.title, shop_vehicle_notification_changes.notification_title, 'Deleted Notification')").AS("notification_title"),
        ).
        FROM(
            ShopVehicleNotificationChanges.
                LEFT_JOIN(Users, Users.UID.EQ(ShopVehicleNotificationChanges.ChangedBy)).
                LEFT_JOIN(ShopVehicleNotifications, ShopVehicleNotifications.ID.EQ(ShopVehicleNotificationChanges.NotificationID)),
        ).
        WHERE(ShopVehicleNotificationChanges.NotificationID.EQ(String(notificationID))).
        ORDER_BY(ShopVehicleNotificationChanges.ChangedAt.DESC()).
        LIMIT(int64(limit))

    // ... rest of query execution
}
```

**Key changes:**
- Use `LEFT_JOIN` instead of `INNER_JOIN` to include records where notification is deleted
- Use `COALESCE` to fall back to denormalized `notification_title` when FK is NULL
- Similar pattern for all change query functions

### 7. Update Response Models

**File:** `api/response/shops_response.go`

Ensure response models handle nullable fields:

```go
type NotificationChangeResponse struct {
    ID               string    `json:"id"`
    NotificationID   *string   `json:"notification_id"`    // Now nullable
    VehicleID        *string   `json:"vehicle_id"`         // Now nullable
    ShopID           string    `json:"shop_id"`
    ChangedBy        *string   `json:"changed_by"`
    ChangedAt        time.Time `json:"changed_at"`
    ChangeType       string    `json:"change_type"`
    FieldChanges     string    `json:"field_changes"`
    Username         *string   `json:"username"`
    NotificationTitle string   `json:"notification_title"` // Always populated via COALESCE
    NotificationType  *string  `json:"notification_type"`  // Nullable
    VehicleAdmin      *string  `json:"vehicle_admin"`      // Nullable
    IsDeleted        bool      `json:"is_deleted"`         // Computed: notification_id IS NULL OR vehicle_id IS NULL
}
```

---

## Migration Strategy

### Phase 1: Database Schema Migration ✅ COMPLETED

**Migration file:** `migrations/XXX_fix_audit_trail_deletions.sql`
**Status:** ✅ Applied to database on 2026-01-01

```sql
-- ============================================================================
-- Migration: Fix Audit Trail Deletion Issues
-- Date: 2026-01-01
-- Description: Add denormalized fields and change FK constraints to preserve
--              audit trail data when notifications or vehicles are deleted.
-- ============================================================================

BEGIN;

-- Step 1: Add denormalized columns (nullable, will be populated going forward)
ALTER TABLE shop_vehicle_notification_changes
  ADD COLUMN notification_title TEXT,
  ADD COLUMN notification_type TEXT,
  ADD COLUMN vehicle_admin TEXT;

-- Step 2: Make foreign key columns nullable
ALTER TABLE shop_vehicle_notification_changes
  ALTER COLUMN notification_id DROP NOT NULL,
  ALTER COLUMN vehicle_id DROP NOT NULL;

-- Step 3: Drop NOT NULL check constraints (auto-generated)
ALTER TABLE shop_vehicle_notification_changes
  DROP CONSTRAINT IF EXISTS "2200_44453_2_not_null",  -- notification_id
  DROP CONSTRAINT IF EXISTS "2200_44453_4_not_null";  -- vehicle_id

-- Step 4: Drop existing CASCADE foreign key constraints
ALTER TABLE shop_vehicle_notification_changes
  DROP CONSTRAINT IF EXISTS fk_notification,
  DROP CONSTRAINT IF EXISTS fk_vehicle;

-- Step 5: Add new SET NULL foreign key constraints
ALTER TABLE shop_vehicle_notification_changes
  ADD CONSTRAINT fk_notification
    FOREIGN KEY (notification_id)
    REFERENCES shop_vehicle_notifications(id)
    ON DELETE SET NULL
    ON UPDATE CASCADE,

  ADD CONSTRAINT fk_vehicle
    FOREIGN KEY (vehicle_id)
    REFERENCES shop_vehicle(id)
    ON DELETE SET NULL
    ON UPDATE CASCADE;

-- Step 6: Add comment for documentation
COMMENT ON COLUMN shop_vehicle_notification_changes.notification_title IS
  'Denormalized: notification title at time of change. Preserved when notification is deleted.';

COMMENT ON COLUMN shop_vehicle_notification_changes.notification_type IS
  'Denormalized: notification type at time of change. Preserved when notification is deleted.';

COMMENT ON COLUMN shop_vehicle_notification_changes.vehicle_admin IS
  'Denormalized: vehicle admin number at time of change. Preserved when vehicle is deleted.';

COMMENT ON CONSTRAINT fk_notification ON shop_vehicle_notification_changes IS
  'ON DELETE SET NULL preserves audit trail when notifications are deleted';

COMMENT ON CONSTRAINT fk_vehicle ON shop_vehicle_notification_changes IS
  'ON DELETE SET NULL preserves audit trail when vehicles are deleted';

COMMIT;

-- ============================================================================
-- Verification Queries
-- ============================================================================

-- Verify column additions
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_name = 'shop_vehicle_notification_changes'
  AND column_name IN ('notification_title', 'notification_type', 'vehicle_admin', 'notification_id', 'vehicle_id');

-- Verify constraint changes
SELECT
    conname AS constraint_name,
    pg_get_constraintdef(oid) AS constraint_definition
FROM pg_constraint
WHERE conrelid = 'shop_vehicle_notification_changes'::regclass
  AND conname IN ('fk_notification', 'fk_vehicle');
```

#### ✅ Verification Results (2026-01-01)

**Column Additions - VERIFIED:**
```
notification_id    | text | YES (nullable)
vehicle_id         | text | YES (nullable)
notification_title | text | YES (nullable)
notification_type  | text | YES (nullable)
vehicle_admin      | text | YES (nullable)
```

**Constraint Changes - VERIFIED:**
```
fk_notification: FOREIGN KEY (notification_id)
  REFERENCES shop_vehicle_notifications(id)
  ON DELETE SET NULL  ✅

fk_vehicle: FOREIGN KEY (vehicle_id)
  REFERENCES shop_vehicle(id)
  ON DELETE SET NULL  ✅
```

**Dropped Constraints - VERIFIED:**
- `2200_44453_2_not_null` (notification_id NOT NULL) - ✅ Removed
- `2200_44453_4_not_null` (vehicle_id NOT NULL) - ✅ Removed

All database schema changes have been successfully applied and verified.

### Phase 2: Code Deployment 🚧 IN PROGRESS

**Deploy sequence:**
1. Deploy database migration (backward compatible - no code changes required yet)
2. Deploy application code with updated audit recording
3. Verify audit trail functionality in staging
4. Deploy to production

**Backward Compatibility:**
- Database migration is backward compatible with existing code
- New columns are nullable and will be NULL for records created by old code
- Old code will continue to work (just won't populate denormalized fields)
- New code will populate denormalized fields for all new changes

### Phase 3: Testing and Validation

All testing will be performed manually using the manual testing checklist below.

---

## Manual Testing Checklist

- [ ] Create notification → verify audit record with denormalized data
- [ ] Update notification → verify audit record with denormalized data
- [ ] Delete notification → verify audit record preserved with NULL FK
- [ ] Create vehicle → verify no audit record (expected)
- [ ] Delete vehicle → verify audit record created with vehicle_admin
- [ ] Delete vehicle with notifications → verify all notification audits have NULL vehicle_id
- [ ] Query deleted notification changes → verify readable titles and types
- [ ] Query shop changes including deleted entities → verify complete history
- [ ] Query vehicle changes for deleted vehicle → verify vehicle_admin preserved

---

## Rollback Plan

### Database Rollback

**Rollback migration file:** `migrations/XXX_rollback_audit_trail_fix.sql`

```sql
BEGIN;

-- Restore CASCADE foreign key constraints
ALTER TABLE shop_vehicle_notification_changes
  DROP CONSTRAINT IF EXISTS fk_notification,
  DROP CONSTRAINT IF EXISTS fk_vehicle;

ALTER TABLE shop_vehicle_notification_changes
  ADD CONSTRAINT fk_notification
    FOREIGN KEY (notification_id)
    REFERENCES shop_vehicle_notifications(id)
    ON DELETE CASCADE,

  ADD CONSTRAINT fk_vehicle
    FOREIGN KEY (vehicle_id)
    REFERENCES shop_vehicle(id)
    ON DELETE CASCADE;

-- Make columns NOT NULL again
ALTER TABLE shop_vehicle_notification_changes
  ALTER COLUMN notification_id SET NOT NULL,
  ALTER COLUMN vehicle_id SET NOT NULL;

-- Remove denormalized columns
ALTER TABLE shop_vehicle_notification_changes
  DROP COLUMN notification_title,
  DROP COLUMN notification_type,
  DROP COLUMN vehicle_admin;

COMMIT;
```

**⚠️ WARNING:** Rollback will **delete** any audit records where notification_id or vehicle_id is NULL (i.e., records for deleted entities). This is destructive and should only be done if absolutely necessary.

### Code Rollback

Revert code changes in reverse order:
1. Revert application code to previous version
2. Run rollback migration
3. Verify system functionality

---

## Performance Considerations

### Query Performance

**Impact:** Minimal to none
- Nullable columns add negligible overhead
- `LEFT JOIN` instead of `INNER JOIN` has minimal performance impact for small result sets
- Existing indexes on `notification_id` and `vehicle_id` still work with NULL values

### Storage Impact

**Impact:** Low
- 3 new TEXT columns: ~100 bytes per record average
- For 100,000 audit records: ~10 MB additional storage
- Acceptable trade-off for audit trail integrity

### Index Strategy

**No new indexes required** based on user decision. Existing indexes remain effective:
- `idx_notification_changes_notification_id` - works with NULL values
- `idx_notification_changes_vehicle_changes` - works with NULL values
- `idx_notification_changes_shop_changes` - unaffected

**Future consideration:** If queries for deleted entities become common, consider partial indexes:
```sql
-- Only if query patterns change
CREATE INDEX idx_deleted_notifications
  ON shop_vehicle_notification_changes (shop_id, changed_at DESC)
  WHERE notification_id IS NULL;
```

---

## Security Considerations

### Data Integrity

✅ **Maintained:**
- Foreign key constraints still enforce referential integrity for active entities
- `ON DELETE SET NULL` gracefully handles deletions
- No orphaned references (FK is either valid or NULL)

### Audit Trail Integrity

✅ **Improved:**
- Audit records can no longer be CASCADE deleted
- Complete historical record of all changes
- Denormalized data prevents information loss

### Permission Model

✅ **Unchanged:**
- All existing permission checks remain in place
- No changes to authorization logic
- Vehicle deletion audit uses same permission model as notification deletion

---

## Data Retention Policy

**Requirement:** Retain audit data **forever**

### Implications

1. **Table Growth:** Audit table will grow indefinitely
2. **Query Performance:** May degrade over time with millions of records
3. **Storage Costs:** Linear growth with system usage

### Future Optimization Strategies (Not in Scope)

If performance degrades in the future, consider:

1. **Table Partitioning** (PostgreSQL 10+)
```sql
-- Partition by year
CREATE TABLE shop_vehicle_notification_changes_2026
  PARTITION OF shop_vehicle_notification_changes
  FOR VALUES FROM ('2026-01-01') TO ('2027-01-01');
```

2. **Archival Strategy**
   - Move records older than N years to cold storage table
   - Keep hot table performant
   - Union queries when historical data needed

3. **Indexed Views**
   - Materialized views for common audit queries
   - Refresh periodically

**Decision:** Implement if/when audit table exceeds 1 million records or query performance degrades.

---

## Success Criteria

### Must Have
- [ ] Database migration executes successfully without errors
- [ ] Notification deletion preserves audit trail with denormalized data
- [ ] Vehicle deletion creates audit record with vehicle_admin
- [ ] Existing queries continue to work with nullable FKs
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] Manual testing checklist completed
- [ ] No production errors related to audit trail for 7 days post-deployment

### Nice to Have
- [ ] Dashboard showing deleted entity audit trail
- [ ] Audit trail search by deleted notification title
- [ ] Audit trail search by deleted vehicle admin number

---

## Implementation Checklist

### Database ✅ COMPLETED
- [x] ✅ Create migration file `XXX_fix_audit_trail_deletions.sql`
- [x] ✅ Create rollback file `XXX_rollback_audit_trail_fix.sql`
- [x] ✅ Test migration on local database
- [x] ✅ Test rollback on local database
- [x] ✅ Run migration on production database (2026-01-01)
- [x] ✅ Verify schema changes in production
  - All 3 denormalized columns added
  - Both FK columns now nullable
  - NOT NULL constraints dropped
  - FK constraints changed to ON DELETE SET NULL

### Code Changes 🚧 TODO
- [ ] 🚧 Update `shop_vehicle_notification_changes` model (regenerate from schema)
- [ ] 🚧 Update `recordNotificationChange` function signature
- [ ] 🚧 Update all calls to `recordNotificationChange` with denormalized data
  - [ ] `CreateVehicleNotification`
  - [ ] `UpdateVehicleNotification`
  - [ ] `CompleteVehicleNotification`
  - [ ] `ReopenVehicleNotification`
  - [ ] `DeleteVehicleNotification`
  - [ ] `AddNotificationItem`
  - [ ] `RemoveNotificationItem`
  - [ ] `RemoveNotificationItemList`
- [ ] 🚧 Implement vehicle deletion audit trail
- [ ] 🚧 Add `buildVehicleDeletionFieldChanges` helper function
- [ ] 🚧 Update query functions to handle NULL FKs (LEFT JOIN, COALESCE)
  - [ ] `GetNotificationChanges`
  - [ ] `GetNotificationChangesByShop`
  - [ ] `GetNotificationChangesByVehicle`
- [ ] 🚧 Update response models for nullable fields
- [ ] 🚧 Add `IsDeleted` computed field to responses

### Testing 🚧 TODO
- [ ] 🚧 Complete manual testing checklist (see [Manual Testing Checklist](#manual-testing-checklist))
- [ ] 🚧 Manual performance validation with production data

### Deployment
- [x] ✅ Deploy database migration to production (2026-01-01)
- [ ] 🚧 Deploy code to staging
- [ ] 🚧 Smoke test in staging
- [ ] 🚧 Deploy code to production
- [ ] 🚧 Monitor error logs for 24 hours
- [ ] 🚧 Verify audit trail functionality in production

### Documentation
- [x] ✅ Update design document with migration status
- [ ] 🚧 Update API documentation for new response fields
- [ ] 🚧 Update database schema documentation
- [ ] 🚧 Create operations runbook for audit trail queries
- [ ] 🚧 Document rollback procedure

---

## Open Questions

None - all clarifications received from stakeholder.

---

## References

- Current audit implementation: [shops_service_impl.go:1144-1186](api/service/shops_service_impl.go#L1144-L1186)
- Audit table schema: PostgreSQL `shop_vehicle_notification_changes`
- Related tables: `shop_vehicle_notifications`, `shop_vehicle`, `shop_notification_items`
- PostgreSQL ON DELETE documentation: https://www.postgresql.org/docs/current/ddl-constraints.html#DDL-CONSTRAINTS-FK

---

## Appendix A: PostgreSQL Best Practices for Audit Tables

### Industry Standard Patterns

1. **Denormalization:** Store descriptive data at time of event
2. **Nullable FKs:** Use `ON DELETE SET NULL` for soft references
3. **Immutability:** Never UPDATE audit records, only INSERT
4. **Timestamps:** Always use UTC with timezone
5. **JSONB:** Use for flexible field_changes storage
6. **Partitioning:** For tables > 1M rows
7. **Separate Schema:** Consider `audit` schema for organization

### This Implementation

✅ Denormalization - notification_title, notification_type, vehicle_admin
✅ Nullable FKs - ON DELETE SET NULL
✅ Immutability - No UPDATE operations on audit table
✅ Timestamps - changed_at with timezone
✅ JSONB - field_changes column
⏸️ Partitioning - Future consideration
⏸️ Separate Schema - Not required for current scale

---

## Appendix B: Example Audit Queries After Migration

### Query 1: Show All Changes for Deleted Notification
```sql
SELECT
    changed_at,
    change_type,
    COALESCE(n.title, c.notification_title, 'Deleted') as notification_title,
    COALESCE(v.admin, c.vehicle_admin, 'Deleted') as vehicle_admin,
    u.username,
    c.field_changes
FROM shop_vehicle_notification_changes c
LEFT JOIN shop_vehicle_notifications n ON c.notification_id = n.id
LEFT JOIN shop_vehicle v ON c.vehicle_id = v.id
LEFT JOIN users u ON c.changed_by = u.uid
WHERE c.notification_id = '...' OR c.notification_title = 'Annual PMCS'
ORDER BY changed_at DESC;
```

### Query 2: Show All Deleted Vehicles with Their Last Known Admin
```sql
SELECT DISTINCT
    c.vehicle_admin,
    COUNT(*) as change_count,
    MAX(c.changed_at) as last_change
FROM shop_vehicle_notification_changes c
WHERE c.vehicle_id IS NULL
  AND c.vehicle_admin IS NOT NULL
  AND c.shop_id = '...'
GROUP BY c.vehicle_admin
ORDER BY last_change DESC;
```

### Query 3: Show Complete Shop History Including Deleted Entities
```sql
SELECT
    changed_at,
    change_type,
    CASE
        WHEN notification_id IS NULL THEN notification_title || ' (deleted)'
        ELSE n.title
    END as notification,
    CASE
        WHEN vehicle_id IS NULL THEN vehicle_admin || ' (deleted)'
        ELSE v.admin
    END as vehicle,
    u.username
FROM shop_vehicle_notification_changes c
LEFT JOIN shop_vehicle_notifications n ON c.notification_id = n.id
LEFT JOIN shop_vehicle v ON c.vehicle_id = v.id
LEFT JOIN users u ON c.changed_by = u.uid
WHERE c.shop_id = '...'
ORDER BY changed_at DESC
LIMIT 100;
```

---

**END OF DOCUMENT**
