# Shops API Efficiency Mobile Refactor Guide

## Purpose

This document explains the Shops API efficiency changes added for mobile clients. It is intended as a refactor guide for replacing multi-request screen loads with additive aggregate read endpoints where that improves client performance.

The existing Shops endpoints remain active. These changes do not remove, deprecate, or reshape existing endpoint contracts. Mobile can adopt the new aggregate endpoints screen by screen.

Base URL: `https://<host>/api/v1/auth`

Authentication: Firebase ID token in the `Authorization` header using the Bearer scheme.

Request body: none for all endpoints in this document.

## What Changed

The server added four authenticated aggregate read endpoints:

| New endpoint | Main use |
|--------------|----------|
| `GET /shops/bootstrap` | Load the Shops landing screen across all shops visible to the user. |
| `GET /shops/:shop_id/snapshot` | Load one shop detail/dashboard screen with selected related sections. |
| `GET /shops/:shop_id/lists-with-items` | Load shop lists and list items without one request per list. |
| `GET /shops/vehicles/:vehicle_id/maintenance-snapshot` | Load a vehicle maintenance screen with notifications, changes, and services. |

The server also fixed existing query shapes for current Shops/equipment-service endpoints. Those fixes preserve the existing response contracts and are not new mobile contracts.

## Why These Endpoints Were Added

Several Shops screens could require chained calls:

- Load shops, then per-shop details or counts.
- Load shop lists, then load list items one list at a time.
- Load a vehicle, then notifications, notification items, change history, and equipment services separately.
- Load a shop detail screen by separately requesting vehicles, lists, notifications, services, and message preview data.

The new endpoints let mobile request screen-shaped snapshots from the server. This reduces round trips and avoids client-side N+1 request patterns.

## Replacement Map

Use the new endpoints when the screen needs the combined payload. Keep existing endpoints when the screen only needs one narrow resource or when mutation flows already use a specific legacy endpoint.

| Screen or workflow | New endpoint | Can replace or reduce calls to |
|--------------------|--------------|--------------------------------|
| Shops landing screen | `GET /shops/bootstrap` | `GET /shops/user-data`, repeated per-shop setup calls, and some uses of `GET /shops/equipment/overview` when bounded equipment summaries are enough. |
| Shop detail/dashboard | `GET /shops/:shop_id/snapshot` | `GET /shops/:shop_id`, `GET /shops/:shop_id/vehicles`, `GET /shops/:shop_id/lists`, `GET /shops/:shop_id/notifications`, `GET /shops/:shop_id/equipment-services`, and optional message/change preview calls. |
| Shop list tree | `GET /shops/:shop_id/lists-with-items` | `GET /shops/:shop_id/lists` plus repeated `GET /shops/lists/:list_id/items`. |
| Vehicle maintenance screen | `GET /shops/vehicles/:vehicle_id/maintenance-snapshot` | `GET /shops/vehicles/:vehicle_id`, `GET /shops/vehicles/:vehicle_id/notifications-with-items`, `GET /shops/vehicles/:vehicle_id/notifications/changes`, and relevant equipment-service list calls. |

Do not use these aggregate reads for create, update, delete, complete, or batch item mutations. Existing mutation endpoints remain the contract for writes.

## Common Response Envelope

Successful aggregate responses use the standard response envelope:

| Field | Type | Description |
|-------|------|-------------|
| `status` | integer | HTTP status code. |
| `message` | string | Human-readable result message. |
| `data` | object | Endpoint-specific payload. |

Example:

```json
{
  "status": 200,
  "message": "Shops bootstrap retrieved successfully",
  "data": {}
}
```

## Limits And Counts

The aggregate endpoints intentionally bound high-cardinality arrays by default. Limits protect mobile payload size and server query size.

Rules:

- Limit query parameters are optional.
- Missing limit parameters use server defaults.
- Values above the maximum are capped to the maximum.
- Empty, non-integer, zero, or negative limit values return `400 Bad Request`.
- `limits` objects show the applied limits after defaulting and max-cap clamping.
- Returned-section `counts` count records included in that response.
- `shop.counts` and bootstrap `counts` are shop-wide totals, not bounded array lengths.

## Endpoint: `GET /shops/bootstrap`

Purpose: load the first Shops screen for the authenticated user in one request.

Full path: `GET /api/v1/auth/shops/bootstrap`

Authentication: required.

Request body: none.

Returns only shops where the authenticated user is a member.

### Query Parameters

| Parameter | Type | Required | Default | Maximum | Description |
|-----------|------|----------|---------|---------|-------------|
| `equipment_limit_per_shop` | integer | No | `50` | `250` | Maximum compact equipment records returned per shop. |

### Use This To Replace

Use this for the Shops landing screen when mobile needs shop identity, role, settings, counts, and a bounded equipment preview in one call.

It can reduce or replace loading `GET /shops/user-data` plus follow-up per-shop setup calls. It can also replace some uses of `GET /shops/equipment/overview` when the screen only needs bounded equipment previews.

Keep using `GET /shops/equipment/overview` when the screen specifically needs the existing unbounded compact equipment overview contract.

### Success Data Shape

| Field | Type | Description |
|-------|------|-------------|
| `shops` | array | Shops visible to the authenticated user. Empty if the user belongs to no shops. |
| `shops[].id` | string | Shop ID. |
| `shops[].name` | string | Shop name. |
| `shops[].details` | string or null | Shop details. |
| `shops[].role` | string | Authenticated user's role in this shop. |
| `shops[].is_admin` | boolean | Whether the user is an admin in this shop. |
| `shops[].settings.admin_only_lists` | boolean | Whether only admins can manage lists. |
| `shops[].counts` | object | Shop-wide totals. |
| `shops[].equipment` | array | Bounded compact equipment records for this shop. |

### Bootstrap Success Example

```json
{
  "status": 200,
  "message": "Shops bootstrap retrieved successfully",
  "data": {
    "shops": [
      {
        "id": "shop-123",
        "name": "Alpha Maintenance Shop",
        "details": "Motor pool maintenance team",
        "role": "admin",
        "is_admin": true,
        "settings": {
          "admin_only_lists": false
        },
        "counts": {
          "members": 8,
          "vehicles": 24,
          "lists": 3,
          "messages": 42,
          "notifications": 12,
          "notification_items": 18,
          "open_services": 5,
          "services": 14,
          "recent_changes": 9
        },
        "equipment": [
          {
            "id": "vehicle-123",
            "admin": "A12",
            "model": "M1152A1",
            "serial": "SER-0001",
            "niin": "013456789"
          }
        ]
      },
      {
        "id": "shop-456",
        "name": "Bravo Shop",
        "details": null,
        "role": "member",
        "is_admin": false,
        "settings": {
          "admin_only_lists": true
        },
        "counts": {
          "members": 3,
          "vehicles": 0,
          "lists": 0,
          "messages": 0,
          "notifications": 0,
          "open_services": 0
        },
        "equipment": []
      }
    ]
  }
}
```

### Bootstrap Empty Example

```json
{
  "status": 200,
  "message": "Shops bootstrap retrieved successfully",
  "data": {
    "shops": []
  }
}
```

## Endpoint: `GET /shops/:shop_id/snapshot`

Purpose: load one shop detail/dashboard screen in one request.

Full path: `GET /api/v1/auth/shops/:shop_id/snapshot`

Authentication: required. The authenticated user must be a member of the shop.

Request body: none.

### Query Parameters

| Parameter | Type | Required | Default | Maximum | Description |
|-----------|------|----------|---------|---------|-------------|
| `include` | comma-separated string | No | `vehicles,lists,notifications,services` | Not applicable | Controls which related arrays are populated. Allowed values are `vehicles`, `lists`, `notifications`, `messages`, `services`, `changes`. |
| `vehicles_limit` | integer | No | `50` | `200` | Maximum vehicles returned when `vehicles` is included. |
| `lists_limit` | integer | No | `50` | `200` | Maximum lists returned when `lists` is included. |
| `items_limit` | integer | No | `50` | `200` | Maximum list items returned per returned list when `lists` is included. |
| `notification_limit` | integer | No | `50` | `200` | Maximum notifications returned when `notifications` is included. |
| `notification_items_limit` | integer | No | `25` | `100` | Maximum notification items returned per returned notification. |
| `message_limit` | integer | No | `20` | `100` | Maximum messages returned when `messages` is included. |
| `services_limit` | integer | No | `50` | `200` | Maximum services returned when `services` is included. |
| `changes_limit` | integer | No | `50` | `200` | Maximum recent changes returned when `changes` is included. |

### Use This To Replace

Use this when the shop detail/dashboard screen needs several related sections at once.

It can reduce or replace separate calls to:

| Existing call | Replacement note |
|---------------|------------------|
| `GET /shops/:shop_id` | `data.shop` contains shop identity, role, settings, and counts. |
| `GET /shops/:shop_id/vehicles` | Include `vehicles`. |
| `GET /shops/:shop_id/lists` plus item calls | Include `lists`. |
| `GET /shops/:shop_id/notifications` | Include `notifications`. |
| `GET /shops/:shop_id/equipment-services` | Include `services`. |
| `GET /shops/:shop_id/notifications/changes` | Include `changes`. |

### Success Data Shape

| Field | Type | Description |
|-------|------|-------------|
| `shop` | object | Shop summary, role, settings, and shop-wide counts. |
| `vehicles` | array | Bounded vehicles if included; otherwise `[]`. |
| `lists` | array | Bounded lists with bounded items if included; otherwise `[]`. |
| `notifications` | array | Bounded notifications with bounded items if included; otherwise `[]`. |
| `messages` | array | Bounded messages if included; otherwise `[]`. Not included by default. |
| `services` | array | Bounded services if included; otherwise `[]`. |
| `recent_changes` | array | Bounded recent changes if included; otherwise `[]`. Not included by default. |
| `limits` | object | Applied limits. |

### Shop Snapshot Success Example

```json
{
  "status": 200,
  "message": "Shop snapshot retrieved successfully",
  "data": {
    "shop": {
      "id": "shop-123",
      "name": "Alpha Maintenance Shop",
      "details": "Motor pool maintenance team",
      "role": "admin",
      "is_admin": true,
      "settings": {
        "admin_only_lists": false
      },
      "counts": {
        "members": 8,
        "vehicles": 24,
        "lists": 3,
        "messages": 42,
        "notifications": 12,
        "open_services": 5
      }
    },
    "limits": {
      "vehicles": 50,
      "lists": 50,
      "items_per_list": 50,
      "notifications": 50,
      "notification_items_per_notification": 25,
      "messages": 20,
      "services": 50,
      "recent_changes": 50
    },
    "vehicles": [
      {
        "id": "vehicle-123",
        "creator_id": "firebase-user-1",
        "niin": "013456789",
        "admin": "A12",
        "model": "M1152A1",
        "serial": "SER-0001",
        "uoc": "UNK",
        "mileage": 12345,
        "hours": 420,
        "comment": "Ready",
        "save_time": "2026-07-01T18:00:00Z",
        "last_updated": "2026-07-03T15:00:00Z",
        "shop_id": "shop-123",
        "tracked_mileage": 12345,
        "tracked_hours": 420
      }
    ],
    "lists": [
      {
        "id": "list-123",
        "shop_id": "shop-123",
        "created_by": "firebase-user-1",
        "created_by_username": "Garcia",
        "description": "Monday service parts",
        "created_at": "2026-07-03T15:10:00Z",
        "updated_at": "2026-07-03T15:12:00Z",
        "items": [
          {
            "id": "item-123",
            "list_id": "list-123",
            "niin": "012345678",
            "nomenclature": "Oil filter",
            "quantity": 4,
            "added_by": "firebase-user-1",
            "added_by_username": "Garcia",
            "created_at": "2026-07-03T15:12:00Z",
            "updated_at": "2026-07-03T15:12:00Z",
            "nickname": "Engine filter",
            "unit_of_measure": "EA"
          }
        ]
      }
    ],
    "notifications": [
      {
        "notification": {
          "id": "notification-123",
          "shop_id": "shop-123",
          "vehicle_id": "vehicle-123",
          "title": "Quarterly PM",
          "description": "Quarterly service due",
          "type": "PM",
          "completed": false,
          "save_time": "2026-07-02T16:00:00Z",
          "last_updated": "2026-07-02T16:00:00Z"
        },
        "items": []
      }
    ],
    "messages": [],
    "services": [
      {
        "id": "service-123",
        "shop_id": "shop-123",
        "equipment_id": "vehicle-123",
        "list_id": "list-123",
        "description": "Quarterly service",
        "service_type": "inspection",
        "created_by": "firebase-user-1",
        "created_by_username": "Garcia",
        "is_completed": false,
        "created_at": "2026-07-01T12:00:00Z",
        "updated_at": "2026-07-01T12:00:00Z",
        "service_date": "2026-07-08T12:00:00Z",
        "service_hours": 450,
        "completion_date": null
      }
    ],
    "recent_changes": []
  }
}
```

### Include Messages Example

Full path example: `GET /api/v1/auth/shops/shop-123/snapshot?include=messages&message_limit=1`

```json
{
  "status": 200,
  "message": "Shop snapshot retrieved successfully",
  "data": {
    "shop": {
      "id": "shop-123",
      "name": "Alpha Maintenance Shop",
      "details": "Motor pool maintenance team",
      "role": "member",
      "is_admin": false,
      "settings": {
        "admin_only_lists": false
      },
      "counts": {
        "members": 8,
        "vehicles": 24,
        "lists": 3,
        "messages": 42,
        "notifications": 12,
        "open_services": 5
      }
    },
    "limits": {
      "vehicles": 50,
      "lists": 50,
      "items_per_list": 50,
      "notifications": 50,
      "notification_items_per_notification": 25,
      "messages": 1,
      "services": 50,
      "recent_changes": 50
    },
    "vehicles": [],
    "lists": [],
    "notifications": [],
    "messages": [
      {
        "id": "message-123",
        "shop_id": "shop-123",
        "user_id": "firebase-user-1",
        "message": "Parts are staged.",
        "created_at": "2026-07-03T15:30:00Z",
        "updated_at": "2026-07-03T15:30:00Z",
        "is_edited": false,
        "parent_id": null
      }
    ],
    "services": [],
    "recent_changes": []
  }
}
```

## Endpoint: `GET /shops/:shop_id/lists-with-items`

Purpose: load a bounded shop list tree in one request.

Full path: `GET /api/v1/auth/shops/:shop_id/lists-with-items`

Authentication: required. The authenticated user must be a member of the shop.

Request body: none.

### Query Parameters

| Parameter | Type | Required | Default | Maximum | Description |
|-----------|------|----------|---------|---------|-------------|
| `lists_limit` | integer | No | `50` | `200` | Maximum lists returned. |
| `items_limit` | integer | No | `50` | `200` | Maximum items returned per returned list. |

### Use This To Replace

Use this for list views that need list headers and their items together.

It can replace:

- `GET /shops/:shop_id/lists`
- Follow-up calls to `GET /shops/lists/:list_id/items` for each list

Keep using the narrow list and list-item endpoints for list create, update, delete, item create, item update, item delete, and batch item mutations.

### Success Data Shape

| Field | Type | Description |
|-------|------|-------------|
| `lists` | array | Bounded list objects. |
| `counts.lists` | integer | Number of list objects returned. |
| `counts.items` | integer | Number of nested list item objects returned. |
| `limits.lists` | integer | Applied maximum list count. |
| `limits.items_per_list` | integer | Applied maximum item count per returned list. |

### Lists With Items Success Example

```json
{
  "status": 200,
  "message": "Shop lists with items retrieved successfully",
  "data": {
    "counts": {
      "lists": 2,
      "items": 1
    },
    "limits": {
      "lists": 50,
      "items_per_list": 50
    },
    "lists": [
      {
        "id": "list-123",
        "shop_id": "shop-123",
        "created_by": "firebase-user-1",
        "created_by_username": "Garcia",
        "description": "Monday service parts",
        "created_at": "2026-07-03T15:10:00Z",
        "updated_at": "2026-07-03T15:12:00Z",
        "items": [
          {
            "id": "item-123",
            "list_id": "list-123",
            "niin": "012345678",
            "nomenclature": "Oil filter",
            "quantity": 4,
            "added_by": "firebase-user-1",
            "added_by_username": "Garcia",
            "created_at": "2026-07-03T15:12:00Z",
            "updated_at": "2026-07-03T15:12:00Z",
            "nickname": "Engine filter",
            "unit_of_measure": "EA"
          }
        ]
      },
      {
        "id": "list-456",
        "shop_id": "shop-123",
        "created_by": "firebase-user-2",
        "created_by_username": null,
        "description": "Empty list",
        "created_at": "2026-07-02T20:00:00Z",
        "updated_at": "2026-07-02T20:00:00Z",
        "items": []
      }
    ]
  }
}
```

## Endpoint: `GET /shops/vehicles/:vehicle_id/maintenance-snapshot`

Purpose: load one vehicle maintenance screen in one request.

Full path: `GET /api/v1/auth/shops/vehicles/:vehicle_id/maintenance-snapshot`

Authentication: required. The authenticated user must be a member of the vehicle's shop.

Request body: none.

### Query Parameters

| Parameter | Type | Required | Default | Maximum | Description |
|-----------|------|----------|---------|---------|-------------|
| `notification_limit` | integer | No | `50` | `200` | Maximum notifications returned. |
| `notification_items_limit` | integer | No | `25` | `100` | Maximum items returned per returned notification. |
| `services_limit` | integer | No | `50` | `200` | Maximum equipment services returned. |
| `changes_limit` | integer | No | `50` | `200` | Maximum recent notification changes returned. |

### Use This To Replace

Use this when the vehicle maintenance screen needs the vehicle plus maintenance context.

It can reduce or replace separate calls to:

- `GET /shops/vehicles/:vehicle_id`
- `GET /shops/vehicles/:vehicle_id/notifications-with-items`
- `GET /shops/vehicles/:vehicle_id/notifications/changes`
- equipment-service list calls filtered to the selected vehicle

Keep using existing vehicle, notification, notification item, and equipment-service mutation endpoints for writes.

### Success Data Shape

| Field | Type | Description |
|-------|------|-------------|
| `vehicle` | object | Vehicle record. |
| `notifications` | array | Bounded notifications with bounded items. |
| `recent_changes` | array | Bounded notification changes for the vehicle. |
| `services` | array | Bounded equipment service records for the vehicle. |
| `counts` | object | Returned-section counts. |
| `limits` | object | Applied limits. |

### Notification Change Shape

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `id` | string | No | Notification change ID. |
| `notification_id` | string | Yes | Notification ID. `null` when the notification has been deleted. |
| `shop_id` | string | No | Shop ID. |
| `vehicle_id` | string | Yes | Vehicle ID. `null` when the vehicle has been deleted. |
| `changed_by` | string | Yes | User ID that made the change, when available. |
| `changed_by_username` | string | No | Username or server fallback value. |
| `changed_at` | string | No | Change timestamp. |
| `change_type` | string | No | Change type. |
| `field_changes.raw` | string | Yes | Encoded JSON string from the database when present. |
| `notification_title` | string | No | Notification title or server fallback value. |
| `notification_type` | string | Yes | Notification type when available. |
| `vehicle_admin` | string | Yes | Vehicle admin number when available. |
| `is_deleted` | boolean | No | `true` when the notification or vehicle reference is deleted. |

### Vehicle Maintenance Snapshot Success Example

```json
{
  "status": 200,
  "message": "Vehicle maintenance snapshot retrieved successfully",
  "data": {
    "vehicle": {
      "id": "vehicle-123",
      "creator_id": "firebase-user-1",
      "niin": "013456789",
      "admin": "A12",
      "model": "M1152A1",
      "serial": "SER-0001",
      "uoc": "UNK",
      "mileage": 12345,
      "hours": 420,
      "comment": "Ready",
      "save_time": "2026-07-01T18:00:00Z",
      "last_updated": "2026-07-03T15:00:00Z",
      "shop_id": "shop-123",
      "tracked_mileage": 12345,
      "tracked_hours": 420
    },
    "notifications": [
      {
        "notification": {
          "id": "notification-123",
          "shop_id": "shop-123",
          "vehicle_id": "vehicle-123",
          "title": "Quarterly PM",
          "description": "Quarterly service due",
          "type": "PM",
          "completed": false,
          "save_time": "2026-07-02T16:00:00Z",
          "last_updated": "2026-07-02T16:00:00Z"
        },
        "items": [
          {
            "id": "notification-item-123",
            "shop_id": "shop-123",
            "notification_id": "notification-123",
            "niin": "012345678",
            "nomenclature": "Oil filter",
            "quantity": 1,
            "save_time": "2026-07-02T16:05:00Z"
          }
        ]
      }
    ],
    "recent_changes": [
      {
        "id": "change-123",
        "notification_id": "notification-123",
        "shop_id": "shop-123",
        "vehicle_id": "vehicle-123",
        "changed_by": "firebase-user-1",
        "changed_by_username": "Garcia",
        "changed_at": "2026-07-03T15:20:00Z",
        "change_type": "updated",
        "field_changes": {
          "raw": "{\"completed\":{\"old\":false,\"new\":true}}"
        },
        "notification_title": "Quarterly PM",
        "notification_type": "PM",
        "vehicle_admin": "A12",
        "is_deleted": false
      }
    ],
    "services": [
      {
        "id": "service-123",
        "shop_id": "shop-123",
        "equipment_id": "vehicle-123",
        "list_id": "list-123",
        "description": "Quarterly service",
        "service_type": "inspection",
        "created_by": "firebase-user-1",
        "created_by_username": "Garcia",
        "is_completed": false,
        "created_at": "2026-07-01T12:00:00Z",
        "updated_at": "2026-07-01T12:00:00Z",
        "service_date": "2026-07-08T12:00:00Z",
        "service_hours": 450,
        "completion_date": null
      }
    ],
    "counts": {
      "notifications": 1,
      "notification_items": 1,
      "recent_changes": 1,
      "services": 1
    },
    "limits": {
      "notifications": 50,
      "notification_items_per_notification": 25,
      "services": 50,
      "recent_changes": 50
    }
  }
}
```

## Error Responses

Authentication errors use the existing auth middleware shape. Aggregate validation and access errors use the shapes below.

### Missing Authorization Header

Status: `401 Unauthorized`

```json
{
  "message": "No Authorization header found"
}
```

### Invalid Authorization Header

Status: `401 Unauthorized`

```json
{
  "message": "Invalid Authorization header"
}
```

### Invalid Firebase Token

Status: `401 Unauthorized`

```json
{
  "message": "Invalid token"
}
```

### Authenticated User Missing From Handler Context

Status: `401 Unauthorized`

```json
{
  "message": "unauthorized"
}
```

### Shop Or Vehicle Access Denied

Status: `403 Forbidden`

```json
{
  "message": "access denied"
}
```

### Invalid Include

Status: `400 Bad Request`

```json
{
  "message": "invalid include"
}
```

### Invalid Limit

Status: `400 Bad Request`

```json
{
  "message": "invalid limit"
}
```

### Aggregate Server Error

Status: `500 Internal Server Error`

```json
{
  "status": 500,
  "message": "failed to retrieve shops aggregate",
  "data": null
}
```

## Compression

All four new aggregate endpoints support endpoint-scoped gzip compression when the request includes `Accept-Encoding: gzip`.

When gzip is applied, the response includes `Content-Encoding: gzip`. The decompressed JSON body has the same structure as the uncompressed response.

This does not change compression behavior for existing Shops endpoints.

## Mobile Adoption Notes

- Adopt these endpoints by screen, not all at once.
- Keep existing narrow endpoints where a screen only needs one resource.
- Keep existing mutation endpoints for writes.
- Treat empty arrays as valid data. The aggregate service normalizes empty sections to `[]`.
- Treat unknown future fields as forward-compatible additions.
- Use the `limits` object to understand what was applied after server defaulting and max-cap clamping.
- Do not assume returned array length equals total records. Use shop-wide `counts` where documented, and returned-section `counts` only as response counts.
- Request gzip for large aggregate screens to reduce transfer size.

## Existing Endpoints That Remain Unchanged

These existing endpoints still use their prior contracts:

| Existing endpoint | Notes |
|-------------------|-------|
| `GET /shops/equipment/overview` | Still returns the existing unbounded compact equipment overview. Use it when the client needs every compact equipment record across visible shops. |
| `GET /shops/user-data` | Still available for existing client flows. |
| `GET /shops/:shop_id` | Still available for narrow shop detail reads. |
| `GET /shops/:shop_id/lists` | Still available for list-only reads. |
| `GET /shops/lists/:list_id/items` | Still available for item-only reads. |
| `GET /shops/:shop_id/vehicles` | Still available for vehicle-only reads. |
| `GET /shops/vehicles/:vehicle_id/notifications-with-items` | Still available for vehicle notification reads. |
| `GET /shops/:shop_id/equipment-services` | Still available for equipment-service list reads. |

