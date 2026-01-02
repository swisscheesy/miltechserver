# Mobile App Design Document: Audit Trail API Response Changes

**Date:** 2026-01-02
**Version:** 2.0
**Status:** Ready for Implementation
**Target API Version:** Post Audit Trail Deletion Fix

## Executive Summary

The backend API has been updated to fix critical audit trail gaps where notification and vehicle deletions were causing audit data loss. The mobile app must update its data models and parsing logic to handle new nullable fields and additional denormalized data in audit trail responses.

**Impact Level:** Medium - Affects all audit trail / notification history features
**Breaking Changes:** Yes - Several previously non-nullable fields are now nullable
**New Fields:** 3 new fields added to audit responses
**Backwards Compatibility:** NOT REQUIRED - All users will update simultaneously

---

## Overview of Backend Changes

### Problem Fixed
- Previously, when notifications or vehicles were deleted, ALL audit history was CASCADE deleted
- Vehicle deletions had NO audit trail at all
- This resulted in complete loss of historical data

### Solution Implemented
- Database now uses `ON DELETE SET NULL` instead of `ON DELETE CASCADE`
- Audit records are preserved when entities are deleted
- Foreign keys become NULL when referenced entities are deleted
- Denormalized data (title, type, vehicle admin) is stored at the time of each change
- New change type: `vehicle_deleted` tracks vehicle deletions

---

## API Response Model Changes

### Affected Response Model: `NotificationChangeWithUsername`

**Endpoints Using This Model:**
- `GET /api/shops/notifications/{notificationID}/changes` - Get change history for a notification
- `GET /api/shops/{shopID}/changes` - Get recent changes for a shop
- `GET /api/shops/vehicles/{vehicleID}/changes` - Get changes for a vehicle

### Old Structure (Before)

```json
{
  "id": "string",
  "notification_id": "string",                 // ⚠️ NOW NULLABLE
  "shop_id": "string",
  "vehicle_id": "string",                      // ⚠️ NOW NULLABLE
  "changed_by": "string",                      // ⚠️ NOW NULLABLE
  "changed_by_username": "string",
  "changed_at": "2024-01-01T12:00:00Z",
  "change_type": "string",
  "field_changes": {},
  "notification_title": "string"
}
```

### New Structure (After)

```json
{
  "id": "string",
  "notification_id": "string | null",          // ✅ NULLABLE - null when notification deleted
  "shop_id": "string",
  "vehicle_id": "string | null",               // ✅ NULLABLE - null when vehicle deleted
  "changed_by": "string | null",               // ✅ NULLABLE
  "changed_by_username": "string",
  "changed_at": "2024-01-01T12:00:00Z",
  "change_type": "string",
  "field_changes": {},
  "notification_title": "string",              // ALWAYS present (via COALESCE fallback)
  "notification_type": "string | null",        // 🆕 NEW - M1, PM, or MW
  "vehicle_admin": "string | null",            // 🆕 NEW - Vehicle admin number
  "is_deleted": boolean                        // 🆕 NEW - true if entity was deleted
}
```

---

## Field-by-Field Changes

### 1. `notification_id` - **NOW NULLABLE** ⚠️

**Old Type:** `string` (always present)
**New Type:** `string | null`

**When NULL:**
- The notification has been deleted from the system
- The audit record is preserved for historical purposes
- Use denormalized fields (`notification_title`, `notification_type`) to display info

**Mobile App Action Required:**
- Change model property from non-nullable to nullable
- Update all code that accesses this field to handle null case
- Consider using `notification_title` for display instead of fetching by ID

**Example Scenarios:**
```json
// Active notification
"notification_id": "uuid-123",
"notification_title": "Annual PMCS",
"is_deleted": false

// Deleted notification (preserved audit trail)
"notification_id": null,
"notification_title": "Annual PMCS",  // Preserved from when it was created
"is_deleted": true
```

---

### 2. `vehicle_id` - **NOW NULLABLE** ⚠️

**Old Type:** `string` (always present)
**New Type:** `string | null`

**When NULL:**
- The vehicle has been deleted from the system
- The audit record is preserved for historical purposes
- Use denormalized field (`vehicle_admin`) to identify the vehicle

**Mobile App Action Required:**
- Change model property from non-nullable to nullable
- Update all code that accesses this field to handle null case
- Consider using `vehicle_admin` for display instead of fetching by ID

**Example Scenarios:**
```json
// Active vehicle
"vehicle_id": "uuid-456",
"vehicle_admin": "A12345",
"is_deleted": false

// Deleted vehicle (preserved audit trail)
"vehicle_id": null,
"vehicle_admin": "A12345",  // Preserved from when it was created
"is_deleted": true
```

---

### 3. `changed_by` - **NOW NULLABLE** ⚠️

**Old Type:** `string` (always present)
**New Type:** `string | null`

**When NULL:**
- System-generated change or user account deleted
- `changed_by_username` will show "Unknown User" as fallback

**Mobile App Action Required:**
- Change model property from non-nullable to nullable
- Rely on `changed_by_username` for display (always present)

---

### 4. `notification_type` - **NEW FIELD** 🆕

**Type:** `string | null`
**Values:** `"M1"`, `"PM"`, `"MW"`, or `null`

**Purpose:**
- Denormalized field capturing notification type at the time of change
- Preserved even when notification is deleted
- Allows filtering/grouping by type in audit history

**When NULL:**
- Change happened before this field was added (old data)
- Not applicable (e.g., `vehicle_deleted` change type)

**Mobile App Action Required:**
- Add new nullable property to model
- Display in audit history UI if desired
- Can be used for filtering audit records by notification type

**Example Usage:**
```json
{
  "change_type": "create",
  "notification_type": "M1",
  "notification_title": "Engine Maintenance"
}
```

---

### 5. `vehicle_admin` - **NEW FIELD** 🆕

**Type:** `string | null`
**Values:** Vehicle admin number (e.g., `"A12345"`) or `null`

**Purpose:**
- Denormalized field capturing vehicle admin number at the time of change
- Preserved even when vehicle is deleted
- Primary identifier for deleted vehicles

**When NULL:**
- Change happened before this field was added (old data)
- Vehicle information not available

**Mobile App Action Required:**
- Add new nullable property to model
- Use for displaying vehicle identity when `vehicle_id` is null
- Display in audit history: "Vehicle: A12345 (deleted)" when `is_deleted` is true

**Example Usage:**
```json
{
  "vehicle_id": null,
  "vehicle_admin": "A12345",
  "is_deleted": true,
  "notification_title": "Quarterly Inspection (deleted)"
}
```

---

### 6. `is_deleted` - **NEW FIELD** 🆕

**Type:** `boolean`
**Values:** `true` or `false`

**Purpose:**
- Computed flag indicating if the notification OR vehicle has been deleted
- Set to `true` when `notification_id IS NULL OR vehicle_id IS NULL`
- Convenient flag for UI rendering logic

**When TRUE:**
- Either the notification was deleted, or the vehicle was deleted, or both
- Audit record is historical data only
- Display with visual indicator (e.g., strikethrough, "deleted" badge)

**Mobile App Action Required:**
- Add new boolean property to model
- Use for UI rendering decisions:
  - Show "deleted" badge
  - Gray out or strikethrough
  - Disable navigation to detail view
  - Show warning that entity no longer exists

**Example Usage:**
```json
// Active record
{
  "is_deleted": false,
  "notification_title": "Monthly Check"
}

// Deleted record
{
  "is_deleted": true,
  "notification_title": "Monthly Check",
  "notification_id": null
}
```

---

## New Change Type

### `vehicle_deleted` - **NEW CHANGE TYPE** 🆕

**Purpose:**
- Tracks when a vehicle is deleted from the system
- Previously, vehicle deletions had NO audit trail

**Characteristics:**
- `notification_id` will be `null` (not tied to specific notification)
- `vehicle_id` will be `null` after deletion
- `vehicle_admin` will contain the admin number of the deleted vehicle
- `is_deleted` will be `true`
- `change_type` will be `"vehicle_deleted"`

**Field Changes Structure:**
```json
{
  "deleted": true,
  "vehicle_data": {
    "admin": "A12345",
    "niin": "012345678",
    "uoc": "UNK",
    "mileage": 50000,
    "hours": 1200,
    "comment": "Decommissioned due to age"
  }
}
```

**Mobile App Action Required:**
- Add `"vehicle_deleted"` to the list of recognized change types
- Handle UI display: "Vehicle A12345 was deleted"
- Parse and display vehicle data from `field_changes` if needed
- Show this in vehicle history views

---

## Complete Change Type Reference

### All Supported Change Types

The mobile app should handle these change types:

1. **`create`** - Notification was created
2. **`update`** - Notification fields were updated
3. **`complete`** - Notification was marked complete
4. **`reopen`** - Notification was reopened after completion
5. **`delete`** - Notification was deleted (now properly tracked)
6. **`items_added`** - Items were added to notification
7. **`items_removed`** - Items were removed from notification
8. **`vehicle_deleted`** - **NEW** - Vehicle was deleted

---

## Implementation Notes

### Handling Null Values

**Critical Points:**

1. **Never assume non-null** for `notification_id`, `vehicle_id`, or `changed_by`
2. **Fallback to denormalized fields** when IDs are null:
   - `notification_title` is always available (even when `notification_id` is null)
   - `vehicle_admin` is available when vehicle data was captured (even when `vehicle_id` is null)
3. **Handle missing denormalized fields**:
   - `notification_type` may be null
   - `vehicle_admin` may be null

---

## Field Changes Structure Reference

Different change types have different `field_changes` structures:

### 1. Create
```json
{
  "fields_changed": ["created"]
}
```

### 2. Update
```json
{
  "fields_changed": ["title", "type"],
  "old_values": {
    "title": "Old Title",
    "type": "M1"
  },
  "new_values": {
    "title": "New Title",
    "type": "PM"
  }
}
```

**Note:** The update structure contains `old_values` and `new_values` objects showing before/after state. The `fields_changed` array lists which specific fields were modified.

**Example - Title Only Update:**
```json
{
  "fields_changed": ["title"],
  "old_values": { "title": "Engine Check" },
  "new_values": { "title": "Engine Maintenance" }
}
```

**Example - Type Only Update:**
```json
{
  "fields_changed": ["type"],
  "old_values": { "type": "M1" },
  "new_values": { "type": "PM" }
}
```

**Example - Multiple Fields Update:**
```json
{
  "fields_changed": ["title", "type", "completed"],
  "old_values": {
    "title": "Monthly Check",
    "type": "M1",
    "completed": false
  },
  "new_values": {
    "title": "Monthly Inspection",
    "type": "PM",
    "completed": false
  }
}
```

### 3. Complete
```json
{
  "fields_changed": ["completed"],
  "completed": true
}
```

**Note:** When a notification is marked complete, only the `completed` field changes to `true`.

### 4. Reopen
```json
{
  "fields_changed": ["completed"],
  "completed": false
}
```

**Note:** When a notification is reopened after being completed, the `completed` field changes back to `false`.

### 5. Delete
```json
{
  "fields_changed": ["deleted"],
  "deleted": true
}
```

### 6. Items Added
```json
{
  "fields_changed": ["items"],
  "item_count": 3,
  "items_added": [
    {
      "niin": "012345678",
      "nomenclature": "Bolt, Hex",
      "quantity": 10
    },
    {
      "niin": "987654321",
      "nomenclature": "Washer, Lock",
      "quantity": 20
    },
    {
      "niin": "111222333",
      "nomenclature": "Nut, Plain",
      "quantity": 15
    }
  ]
}
```

**Note:** The `items_added` array contains all items that were added in this operation. Each item object includes:
- `niin`: National Item Identification Number
- `nomenclature`: Item description/name
- `quantity`: Number of items added

**Example - Single Item:**
```json
{
  "fields_changed": ["items"],
  "item_count": 1,
  "items_added": [
    {
      "niin": "555666777",
      "nomenclature": "Oil Filter",
      "quantity": 1
    }
  ]
}
```

### 7. Items Removed
```json
{
  "fields_changed": ["items"],
  "item_count": 2,
  "items_removed": [
    {
      "niin": "012345678",
      "nomenclature": "Bolt, Hex",
      "quantity": 10
    },
    {
      "niin": "987654321",
      "nomenclature": "Washer, Lock",
      "quantity": 20
    }
  ]
}
```

**Note:** The `items_removed` array contains all items that were removed in this operation. Structure is identical to `items_added`.

**Example - Single Item Removal:**
```json
{
  "fields_changed": ["items"],
  "item_count": 1,
  "items_removed": [
    {
      "niin": "555666777",
      "nomenclature": "Oil Filter",
      "quantity": 1
    }
  ]
}
```

### 8. Vehicle Deleted (NEW) 🆕
```json
{
  "deleted": true,
  "vehicle_data": {
    "admin": "A12345",
    "niin": "012345678",
    "uoc": "UNK",
    "mileage": 50000,
    "hours": 1200,
    "comment": "Decommissioned due to age"
  }
}
```

**Note:** This is a NEW change type that didn't exist before. It tracks when a vehicle is deleted from the system.

**Field Descriptions:**
- `deleted`: Always `true` for this change type
- `vehicle_data.admin`: Vehicle admin number (e.g., "A12345")
- `vehicle_data.niin`: National Item Identification Number for the vehicle
- `vehicle_data.uoc`: Usable On Code (typically "UNK" for unknown)
- `vehicle_data.mileage`: Total mileage at time of deletion (int32)
- `vehicle_data.hours`: Total operating hours at time of deletion (int32)
- `vehicle_data.comment`: Any comment/reason for deletion

**Example - Vehicle with Minimal Data:**
```json
{
  "deleted": true,
  "vehicle_data": {
    "admin": "B67890",
    "niin": "111222333",
    "uoc": "UNK",
    "mileage": 0,
    "hours": 0,
    "comment": ""
  }
}
```

**Important:**
- All `vehicle_data` fields are always present (not nullable)
- Numeric fields (`mileage`, `hours`) will be `0` if not tracked
- String fields will be empty `""` if not set
- This change type will have `notification_id: null` since it's not tied to a specific notification

---

## Implementation Checklist

**Note:** Backwards compatibility is NOT required. All users will update to the new version simultaneously.

### Data Model Changes

- [ ] Update model to make 3 fields nullable: `notification_id`, `vehicle_id`, `changed_by`
- [ ] Add 3 new fields: `notification_type`, `vehicle_admin`, `is_deleted`
- [ ] Update JSON parsing to handle nullable fields
- [ ] Update JSON serialization to include new fields

### Business Logic Changes

- [ ] Add null-safety checks for `notification_id`, `vehicle_id`, `changed_by`
- [ ] Handle denormalized fields (`notification_title`, `notification_type`, `vehicle_admin`)
- [ ] Handle vehicle deletion records (`vehicle_deleted` change type)
- [ ] Update any caching logic to account for nullable fields
- [ ] Update search/filter logic if needed

### Testing

- [ ] Test with deleted notifications
- [ ] Test with deleted vehicles
- [ ] Test with vehicle deletion audit records
- [ ] Test all 8 change types render correctly
- [ ] Test null handling for all nullable fields
- [ ] Verify `field_changes` parsing for all change types

### Testing Scenarios

**Test Case 1: Deleted Notification**
```json
{
  "notification_id": null,
  "vehicle_id": "uuid-123",
  "notification_title": "Quarterly Inspection",
  "notification_type": "PM",
  "vehicle_admin": "A12345",
  "is_deleted": true,
  "change_type": "delete"
}
```
Expected: Show as deleted, display title and type, disable navigation

**Test Case 2: Deleted Vehicle**
```json
{
  "notification_id": "uuid-456",
  "vehicle_id": null,
  "notification_title": "Monthly Check",
  "notification_type": "M1",
  "vehicle_admin": "B67890",
  "is_deleted": true,
  "change_type": "update"
}
```
Expected: Show vehicle as deleted, display admin number

**Test Case 3: Vehicle Deletion Audit**
```json
{
  "notification_id": null,
  "vehicle_id": null,
  "notification_title": "Deleted Notification",
  "notification_type": null,
  "vehicle_admin": "C11111",
  "is_deleted": true,
  "change_type": "vehicle_deleted",
  "field_changes": {
    "deleted": true,
    "vehicle_data": {
      "admin": "C11111",
      "niin": "012345678"
    }
  }
}
```
Expected: Show vehicle deletion with vehicle data

---

## API Endpoints Reference

### GET /api/shops/notifications/{notificationID}/changes

**Returns:** `NotificationChange[]`

**Use Case:** View complete change history for a specific notification

**Response Example:**
```json
[
  {
    "id": "change-1",
    "notification_id": "notif-123",
    "shop_id": "shop-1",
    "vehicle_id": "vehicle-1",
    "changed_by": "user-1",
    "changed_by_username": "John Doe",
    "changed_at": "2024-01-01T10:00:00Z",
    "change_type": "create",
    "field_changes": { "fields_changed": ["created"] },
    "notification_title": "Annual PMCS",
    "notification_type": "M1",
    "vehicle_admin": "A12345",
    "is_deleted": false
  },
  {
    "id": "change-2",
    "notification_id": null,
    "shop_id": "shop-1",
    "vehicle_id": "vehicle-1",
    "changed_by": "user-2",
    "changed_by_username": "Admin User",
    "changed_at": "2024-01-02T15:30:00Z",
    "change_type": "delete",
    "field_changes": { "fields_changed": ["deleted"], "deleted": true },
    "notification_title": "Annual PMCS",
    "notification_type": "M1",
    "vehicle_admin": "A12345",
    "is_deleted": true
  }
]
```

### GET /api/shops/{shopID}/changes?limit=100

**Returns:** `NotificationChange[]`

**Use Case:** View recent changes across all notifications in a shop

**Query Parameters:**
- `limit` (optional): Maximum number of records (default: 100, max: 500)

### GET /api/shops/vehicles/{vehicleID}/changes

**Returns:** `NotificationChange[]`

**Use Case:** View all changes related to a specific vehicle (including vehicle deletion)

**Special Note:** After vehicle deletion, will include `vehicle_deleted` change type record

---

**Document Version History:**
- v1.0 (2026-01-02): Initial version documenting audit trail API changes
