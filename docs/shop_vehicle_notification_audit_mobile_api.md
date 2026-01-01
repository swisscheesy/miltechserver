# Shop Vehicle Notification Change History - Mobile API Documentation

## Overview
This document provides API specifications for implementing the notification change history feature in the Flutter mobile application. The feature allows users to view a complete audit trail of all changes made to shop vehicle notifications.

**Feature Context**: This adds a separate button/action when viewing a notification or its details to show the complete change history.

**Primary Use Case**: Individual notification history accessed via a button on the notification detail screen.

**Target Audience**: Flutter mobile developers

**Related Documentation**: [shop_vehicle_notification_audit_design.md](shop_vehicle_notification_audit_design.md), [shop_vehicle_notification_audit_implementation.md](shop_vehicle_notification_audit_implementation.md)

---

## Table of Contents
- [Authentication](#authentication)
- [Base URL](#base-url)
- [Primary Endpoint](#primary-endpoint-get-notification-change-history)
- [Additional Endpoints](#additional-endpoints)
- [Data Models](#data-models)
- [Change Types](#change-types)
- [Error Responses](#error-responses)

---

## Authentication

All endpoints require authentication via Firebase JWT token in the Authorization header.

```http
Authorization: Bearer <firebase_jwt_token>
```

**Error Response** (401 Unauthorized):
```json
{
  "message": "unauthorized"
}
```

---

## Base URL

All endpoints are prefixed with the API base URL:

```
https://your-api-domain.com/api/v1
```

Replace `your-api-domain.com` with your actual API domain.

---

## Primary Endpoint: Get Notification Change History

Retrieves the complete change history for a specific notification, ordered from newest to oldest.

**Use Case**: Accessed via a button on the notification or notification detail screen to show the complete audit trail.

### Request
```http
GET /shops/notifications/{notification_id}/changes
Authorization: Bearer <token>
```

**Path Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `notification_id` | string (UUID) | Yes | The ID of the notification |

**Query Parameters:** None

### Success Response (200 OK)
```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "notification_id": "660e8400-e29b-41d4-a716-446655440001",
      "shop_id": "770e8400-e29b-41d4-a716-446655440002",
      "vehicle_id": "880e8400-e29b-41d4-a716-446655440003",
      "changed_by": "user-uid-123",
      "changed_by_username": "john_doe",
      "changed_at": "2025-12-06T14:30:25.123Z",
      "change_type": "complete",
      "field_changes": {
        "raw": "{\"fields_changed\": [\"completed\"]}"
      }
    },
    {
      "id": "550e8400-e29b-41d4-a716-446655440004",
      "notification_id": "660e8400-e29b-41d4-a716-446655440001",
      "shop_id": "770e8400-e29b-41d4-a716-446655440002",
      "vehicle_id": "880e8400-e29b-41d4-a716-446655440003",
      "changed_by": "user-uid-456",
      "changed_by_username": "jane_smith",
      "changed_at": "2025-12-06T10:15:00.456Z",
      "change_type": "update",
      "field_changes": {
        "raw": "{\"fields_changed\": [\"title\", \"description\"]}"
      }
    },
    {
      "id": "550e8400-e29b-41d4-a716-446655440005",
      "notification_id": "660e8400-e29b-41d4-a716-446655440001",
      "shop_id": "770e8400-e29b-41d4-a716-446655440002",
      "vehicle_id": "880e8400-e29b-41d4-a716-446655440003",
      "changed_by": "user-uid-789",
      "changed_by_username": "bob_jones",
      "changed_at": "2025-12-05T08:45:30.789Z",
      "change_type": "create",
      "field_changes": {
        "raw": "{\"fields_changed\": [\"created\"]}"
      }
    }
  ]
}
```

### Error Responses

**400 Bad Request** - Missing notification_id:
```json
{
  "message": "notification_id is required"
}
```

**401 Unauthorized** - Invalid or missing authentication:
```json
{
  "message": "unauthorized"
}
```

**403 Forbidden** - User not a member of the shop:
```json
{
  "error": "access denied: user is not a member of this shop"
}
```

**404 Not Found** - Notification doesn't exist:
```json
{
  "error": "failed to get notification: notification not found"
}
```

**500 Internal Server Error** - Server error:
```json
{
  "error": "internal server error"
}
```

---

## Additional Endpoints

These endpoints are available but not the primary mobile use case:

### Get Shop Notification Changes

Retrieves recent notification changes across all notifications in a shop.

```http
GET /shops/{shop_id}/notifications/changes?limit=50
Authorization: Bearer <token>
```

**Path Parameters:**
| Parameter | Type | Required |
|-----------|------|----------|
| `shop_id` | string (UUID) | Yes |

**Query Parameters:**
| Parameter | Type | Required | Default | Max |
|-----------|------|----------|---------|-----|
| `limit` | integer | No | 100 | 500 |

### Get Vehicle Notification Changes

Retrieves all notification changes for a specific vehicle.

```http
GET /shops/vehicles/{vehicle_id}/notifications/changes
Authorization: Bearer <token>
```

**Path Parameters:**
| Parameter | Type | Required |
|-----------|------|----------|
| `vehicle_id` | string (UUID) | Yes |

Both endpoints return the same response format as the primary endpoint.

---

## Data Models

### NotificationChange Model

| Field | Type | Description |
|-------|------|-------------|
| `id` | string (UUID) | Unique identifier for this change record |
| `notification_id` | string (UUID) | ID of the notification that was changed |
| `shop_id` | string (UUID) | ID of the shop |
| `vehicle_id` | string (UUID) | ID of the vehicle |
| `changed_by` | string | UID of the user who made the change |
| `changed_by_username` | string | Username of the user (or "Unknown User" if deleted) |
| `changed_at` | string (ISO 8601) | Timestamp when change was made in UTC format |
| `change_type` | string | Type of change (see [Change Types](#change-types)) |
| `field_changes` | object | Contains a `raw` field with JSON string of what changed |

**Note on Timestamps**: All timestamps are in ISO 8601 format (UTC):
```
2025-12-06T14:30:25.123Z
```

**Note on `field_changes`**: This is an object containing a `raw` field with a JSON string. The mobile app can parse this JSON string as needed:
```json
{
  "raw": "{\"fields_changed\": [\"title\", \"description\"]}"
}
```

### StandardResponse Wrapper

All successful responses are wrapped in this structure:

```json
{
  "status": 200,
  "message": "",
  "data": [...]
}
```

- `status`: HTTP status code (always 200 for success)
- `message`: Empty string for successful responses
- `data`: Array of NotificationChange objects

---

## Change Types

The `change_type` field indicates what kind of change was made:

| Change Type | Description | When It Occurs |
|-------------|-------------|----------------|
| `create` | Notification was created | When a new notification is added |
| `update` | Fields were modified | When notification details are changed |
| `complete` | Notification marked as complete | When notification is marked complete |
| `reopen` | Notification was reopened | When completed notification is reopened |
| `delete` | Notification was deleted | When notification is deleted |
| `items_added` | Items were added | When items are added to the notification |
| `items_removed` | Items were removed | When items are removed from the notification |

---

## Field Changes Structure

The `field_changes` object contains a `raw` field with a JSON string that needs to be parsed.

### Format
```json
{
  "raw": "<JSON_STRING>"
}
```

### Examples After Parsing the Raw String

#### 1. Notification Created
Raw: `"{\"fields_changed\": [\"created\"]}"`

Parsed:
```json
{
  "fields_changed": ["created"]
}
```

#### 2. Fields Updated
Raw: `"{\"fields_changed\": [\"title\", \"description\"]}"`

Parsed:
```json
{
  "fields_changed": ["title", "description"]
}
```

Possible field names:
- `title`
- `description`
- `type`
- `completed`
- `created`
- `deleted`
- `items`

#### 3. Completion Status Changed
Raw: `"{\"fields_changed\": [\"completed\"]}"`

Parsed:
```json
{
  "fields_changed": ["completed"]
}
```

Check `change_type` to determine if it was `complete` or `reopen`.

#### 4. Items Added
Raw: `"{\"fields_changed\": [\"items\"], \"item_count\": 3, \"items_added\": [{\"niin\": \"12345-678-9012\", \"nomenclature\": \"BOLT, MACHINE\", \"quantity\": 25}, {\"niin\": \"98765-432-1098\", \"nomenclature\": \"NUT, HEXAGON\", \"quantity\": 50}, {\"niin\": \"11111-222-3333\", \"nomenclature\": \"WASHER, FLAT\", \"quantity\": 100}]}"`

Parsed:
```json
{
  "fields_changed": ["items"],
  "item_count": 3,
  "items_added": [
    {
      "niin": "12345-678-9012",
      "nomenclature": "BOLT, MACHINE",
      "quantity": 25
    },
    {
      "niin": "98765-432-1098",
      "nomenclature": "NUT, HEXAGON",
      "quantity": 50
    },
    {
      "niin": "11111-222-3333",
      "nomenclature": "WASHER, FLAT",
      "quantity": 100
    }
  ]
}
```

The `item_count` field indicates how many items were added. The `items_added` array contains detailed information about each added item including:
- `niin`: National Item Identification Number
- `nomenclature`: Official item name/description
- `quantity`: Number of units that were added

**Note**: Enhanced detail added December 30, 2025 to match the level of detail provided for item removals.

#### 5. Items Removed
Raw: `\"{\\\"fields_changed\\\": [\\\"items\\\"], \\\"item_count\\\": 2, \\\"items_removed\\\": [{\\\"niin\\\": \\\"12345-678-9012\\\", \\\"nomenclature\\\": \\\"BOLT, MACHINE\\\", \\\"quantity\\\": 25}, {\\\"niin\\\": \\\"98765-432-1098\\\", \\\"nomenclature\\\": \\\"NUT, HEXAGON\\\", \\\"quantity\\\": 50}]}\"`

Parsed:
```json
{
  "fields_changed": ["items"],
  "item_count": 2,
  "items_removed": [
    {
      "niin": "12345-678-9012",
      "nomenclature": "BOLT, MACHINE",
      "quantity": 25
    },
    {
      "niin": "98765-432-1098",
      "nomenclature": "NUT, HEXAGON",
      "quantity": 50
    }
  ]
}
```

The `item_count` field indicates how many items were removed. The `items_removed` array contains detailed information about each removed item including:
- `niin`: National Item Identification Number
- `nomenclature`: Official item name/description
- `quantity`: Number of units that were removed

#### 6. Notification Deleted
Raw: `"{\"fields_changed\": [\"deleted\"]}"`

Parsed:
```json
{
  "fields_changed": ["deleted"]
}
```

---

## Error Responses

### Error Response Formats

Errors use one of two formats:

**Format 1:**
```json
{
  "error": "error message description"
}
```

**Format 2:**
```json
{
  "message": "error message"
}
```

### HTTP Status Codes

| Status Code | Meaning |
|-------------|---------|
| `200` | Success |
| `400` | Bad Request - Invalid or missing parameters |
| `401` | Unauthorized - Invalid/missing authentication token |
| `403` | Forbidden - User lacks permission |
| `404` | Not Found - Resource doesn't exist |
| `500` | Server Error - Internal server error |

### Handling Deleted Users

When a user who made a change is deleted:
- `changed_by` will still contain their UID
- `changed_by_username` will show `"Unknown User"`

---

## Complete Example Response

Here's a complete example showing various change types:

```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "id": "change-uuid-1",
      "notification_id": "notif-123",
      "shop_id": "shop-456",
      "vehicle_id": "vehicle-789",
      "changed_by": "uid-001",
      "changed_by_username": "john_doe",
      "changed_at": "2025-12-06T16:30:00.000Z",
      "change_type": "complete",
      "field_changes": {
        "raw": "{\"fields_changed\": [\"completed\"]}"
      }
    },
    {
      "id": "change-uuid-2",
      "notification_id": "notif-123",
      "shop_id": "shop-456",
      "vehicle_id": "vehicle-789",
      "changed_by": "uid-002",
      "changed_by_username": "jane_smith",
      "changed_at": "2025-12-06T14:15:00.000Z",
      "change_type": "items_added",
      "field_changes": {
        "raw": "{\"fields_changed\": [\"items\"], \"item_count\": 5}"
      }
    },
    {
      "id": "change-uuid-3",
      "notification_id": "notif-123",
      "shop_id": "shop-456",
      "vehicle_id": "vehicle-789",
      "changed_by": "uid-003",
      "changed_by_username": "Unknown User",
      "changed_at": "2025-12-06T10:00:00.000Z",
      "change_type": "update",
      "field_changes": {
        "raw": "{\"fields_changed\": [\"title\", \"description\"]}"
      }
    },
    {
      "id": "change-uuid-4",
      "notification_id": "notif-123",
      "shop_id": "shop-456",
      "vehicle_id": "vehicle-789",
      "changed_by": "uid-001",
      "changed_by_username": "john_doe",
      "changed_at": "2025-12-05T08:30:00.000Z",
      "change_type": "create",
      "field_changes": {
        "raw": "{\"fields_changed\": [\"created\"]}"
      }
    }
  ]
}
```

This example shows a notification's complete history from creation to completion, displayed from newest to oldest.

---

## Implementation Notes

### Authentication
- Use standard Firebase JWT token in Authorization header
- Same authentication mechanism as other shop endpoints
- No token refresh logic needed in this endpoint

### Data Handling
- The `field_changes.raw` field is a JSON string that needs to be parsed by the mobile app
- All timestamps are in ISO 8601 format (UTC)
- Array is ordered newest to oldest (most recent changes first)
- Empty array `[]` is returned if no changes exist

### Error Handling
- Handle all HTTP status codes (400, 401, 403, 404, 500)
- Display appropriate error messages to the user
- "Unknown User" appears when the user who made the change has been deleted

---

## Known Limitations

### ~~Item Removal Tracking Not Implemented~~ ✅ IMPLEMENTED (December 30, 2025)

The `items_removed` change type is **NOW FULLY IMPLEMENTED**.

**What's included**:
- ✅ Complete audit trail for item removals
- ✅ Detailed item information captured (NIIN, nomenclature, quantity)
- ✅ Security verification (shop membership required)
- ✅ Both single and bulk item removal tracking

**Impact**: Users can now see complete audit records when items are removed from notifications, including what specific items were removed.

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2025-12-06 | Initial API documentation for mobile |
| 1.1.0 | 2025-12-30 | Added `items_removed` change type with detailed item information |
| 1.2.0 | 2025-12-30 | Enhanced `items_added` to include detailed item information (NIIN, nomenclature, quantity) |

---

## Support

For questions about this API:
1. Review the design document: [shop_vehicle_notification_audit_design.md](shop_vehicle_notification_audit_design.md)
2. Review the implementation report: [shop_vehicle_notification_audit_implementation.md](shop_vehicle_notification_audit_implementation.md)
3. Contact the backend team
