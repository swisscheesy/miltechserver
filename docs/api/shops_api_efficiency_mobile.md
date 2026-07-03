# Shops Aggregate API Mobile Contract

## API Summary

Base URL: `https://<host>/api/v1/auth`

Authentication: Firebase ID token in the `Authorization` header using the Bearer scheme.

Request body: none for all endpoints in this document.

These endpoints are additive read aggregates for mobile screens that currently need several Shops calls to build one view. They do not replace or change existing Shops endpoints.

## Endpoints

| Endpoint | Purpose |
|----------|---------|
| `GET /shops/:shop_id/lists-with-items` | Return all lists for one shop with each list's items nested inline. |
| `GET /shops/vehicles/:vehicle_id/maintenance-snapshot` | Return one vehicle with bounded maintenance notifications, recent changes, and services. |
| `GET /shops/:shop_id/snapshot` | Return a shop summary with selected related sections for a detail screen. |
| `GET /shops/bootstrap` | Return all shops visible to the authenticated user with settings, counts, and bounded equipment summaries. |

## Backward Compatibility Guarantee

All routes in this document are new aggregate read endpoints. Existing Shops endpoints remain active and keep their existing request and response contracts, including `GET /shops/equipment/overview`.

Mobile clients can adopt these aggregate endpoints screen by screen. No existing client has to migrate immediately, and no existing endpoint should be treated as deprecated because of this document.

## Common Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | Yes | Firebase ID token using the Bearer scheme. |
| `Accept` | No | Use `application/json` if the client sets it. |
| `Accept-Encoding` | No | Include `gzip` if the client can handle gzip-compressed responses. |

## Common Response Envelope

Successful responses use the standard API envelope:

| Field | Type | Description |
|-------|------|-------------|
| `status` | integer | HTTP status code. For success, `200`. |
| `message` | string | Human-readable result message. |
| `data` | object | Endpoint-specific payload. |

## `GET /shops/:shop_id/lists-with-items`

Full path: `GET /api/v1/auth/shops/:shop_id/lists-with-items`

Purpose: load all shop lists and their items in one request.

Authorization: authenticated user must be a member of the shop.

Request body: none.

Query parameters: none.

### Path Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `shop_id` | string | Yes | Shop ID. |

### Success Response

Status: `200 OK`

Message: `Shop lists with items retrieved successfully`

| Field | Type | Description |
|-------|------|-------------|
| `data.lists` | array | Lists in the shop. Empty array if the shop has no lists. |
| `data.lists[].id` | string | List ID. |
| `data.lists[].shop_id` | string | Shop ID that owns the list. |
| `data.lists[].created_by` | string | User ID that created the list. |
| `data.lists[].created_by_username` | string or null | Username for `created_by` when available. |
| `data.lists[].description` | string | List description. |
| `data.lists[].created_at` | string or null | List creation timestamp. |
| `data.lists[].updated_at` | string or null | List update timestamp. |
| `data.lists[].items` | array | Items nested under the list. Empty lists return `[]`, not `null`. |
| `data.lists[].items[].id` | string | List item ID. |
| `data.lists[].items[].list_id` | string | Parent list ID. |
| `data.lists[].items[].niin` | string | Item NIIN. |
| `data.lists[].items[].nomenclature` | string | Item name or description. |
| `data.lists[].items[].quantity` | integer | Requested quantity. |
| `data.lists[].items[].added_by` | string | User ID that added the item. |
| `data.lists[].items[].added_by_username` | string or null | Username for `added_by` when available. |
| `data.lists[].items[].created_at` | string or null | Item creation timestamp. |
| `data.lists[].items[].updated_at` | string or null | Item update timestamp. |
| `data.lists[].items[].nickname` | string or null | Optional item nickname. |
| `data.lists[].items[].unit_of_measure` | string or null | Optional unit of measure. |

### Success Response Example

```json
{
  "status": 200,
  "message": "Shop lists with items retrieved successfully",
  "data": {
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

## `GET /shops/vehicles/:vehicle_id/maintenance-snapshot`

Full path: `GET /api/v1/auth/shops/vehicles/:vehicle_id/maintenance-snapshot`

Purpose: load one vehicle and its maintenance-related sections in one request.

Authorization: authenticated user must be a member of the vehicle's shop.

Request body: none.

### Path Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `vehicle_id` | string | Yes | Shop vehicle ID. |

### Query Parameters

| Parameter | Type | Required | Default | Maximum | Invalid Values | Description |
|-----------|------|----------|---------|---------|----------------|-------------|
| `services_limit` | integer | No | `50` | `200` | Empty value, non-integer, `0`, negative | Maximum services returned. Values above `200` are capped to `200`. |
| `changes_limit` | integer | No | `50` | `200` | Empty value, non-integer, `0`, negative | Maximum recent notification changes returned. Values above `200` are capped to `200`. |

Bounded fixed sections:

| Section | Limit |
|---------|-------|
| `notifications` | Most recent `50` notifications for the vehicle. |
| `notifications[].items` | All items for the returned notifications. |

### Success Response

Status: `200 OK`

Message: `Vehicle maintenance snapshot retrieved successfully`

| Field | Type | Description |
|-------|------|-------------|
| `data.vehicle` | object | Vehicle record. |
| `data.notifications` | array | Bounded vehicle notifications, each with nested `items`. |
| `data.recent_changes` | array | Bounded notification changes for the vehicle. |
| `data.services` | array | Bounded equipment service records for the vehicle. |
| `data.counts` | object | Counts for the returned aggregate sections. These are returned-section counts, not total table counts. |

### `counts` Shape

| Field | Type | Description |
|-------|------|-------------|
| `notifications` | integer | Number of notification objects returned. |
| `notification_items` | integer | Number of nested notification items returned. |
| `recent_changes` | integer | Number of recent change objects returned. |
| `services` | integer | Number of service objects returned. |

### Notification Change Object

The `recent_changes` arrays on this endpoint and `GET /shops/:shop_id/snapshot` use this object shape.

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `id` | string | No | Notification change ID. |
| `notification_id` | string | Yes | Notification ID. `null` when the notification has been deleted. |
| `shop_id` | string | No | Shop ID. |
| `vehicle_id` | string | Yes | Vehicle ID. `null` when the vehicle has been deleted. |
| `changed_by` | string | Yes | User ID that made the change, when available. |
| `changed_by_username` | string | No | Username for the user that made the change, or a server fallback value. |
| `changed_at` | string | No | Change timestamp. |
| `change_type` | string | No | Change type. |
| `field_changes` | object | No | Contains `raw` when database change JSON is present. `raw` is a string containing the encoded change JSON from the database. |
| `notification_title` | string | No | Notification title or server fallback value. |
| `notification_type` | string | Yes | Notification type when available. |
| `vehicle_admin` | string | Yes | Vehicle admin number when available. |
| `is_deleted` | boolean | No | `true` when the notification or vehicle reference is deleted. |

### Success Response Example

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
    }
  }
}
```

## `GET /shops/:shop_id/snapshot`

Full path: `GET /api/v1/auth/shops/:shop_id/snapshot`

Purpose: load one shop summary and selected related sections in one request.

Authorization: authenticated user must be a member of the shop.

Request body: none.

### Path Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `shop_id` | string | Yes | Shop ID. |

### Query Parameters

| Parameter | Type | Required | Default | Maximum | Invalid Values | Description |
|-----------|------|----------|---------|---------|----------------|-------------|
| `include` | comma-separated string | No | `vehicles,lists,notifications,services` | Not applicable | Empty value or any value outside the allowed set | Controls which related arrays are populated. Allowed values: `vehicles`, `lists`, `notifications`, `messages`, `services`, `changes`. When supplied, only listed sections are populated; omitted sections return empty arrays. |
| `message_limit` | integer | No | `20` | `100` | Empty value, non-integer, `0`, negative | Maximum messages returned when `messages` is included. Values above `100` are capped to `100`. |
| `services_limit` | integer | No | `50` | `200` | Empty value, non-integer, `0`, negative | Maximum services returned when `services` is included. Values above `200` are capped to `200`. |
| `changes_limit` | integer | No | `50` | `200` | Empty value, non-integer, `0`, negative | Maximum recent changes returned when `changes` is included. Values above `200` are capped to `200`. |

Bounded fixed sections:

| Section | Limit |
|---------|-------|
| `notifications` | Most recent `50` shop notifications. |
| `notifications[].items` | Up to `25` items per returned notification. |

Unbounded included sections:

| Section | Notes |
|---------|-------|
| `vehicles` | Returns all vehicles in the shop. |
| `lists` | Returns all lists in the shop with all list items. |

### Success Response

Status: `200 OK`

Message: `Shop snapshot retrieved successfully`

| Field | Type | Description |
|-------|------|-------------|
| `data.shop` | object | Shop summary, role, settings, and shop-wide counts. |
| `data.vehicles` | array | Vehicles if included; otherwise `[]`. |
| `data.lists` | array | Lists with items if included; otherwise `[]`. |
| `data.notifications` | array | Notifications with bounded items if included; otherwise `[]`. |
| `data.messages` | array | Messages if included; otherwise `[]`. Messages are not part of the default include set. |
| `data.services` | array | Services if included; otherwise `[]`. |
| `data.recent_changes` | array | Recent changes if included; otherwise `[]`. Changes are not part of the default include set. |

### `shop.counts` Shape

These counts are shop-wide counts, not bounded section lengths.

| Field | Type | Description |
|-------|------|-------------|
| `members` | integer | Total shop members. |
| `vehicles` | integer | Total shop vehicles. |
| `lists` | integer | Total shop lists. |
| `messages` | integer | Total shop messages. |
| `notifications` | integer | Total shop notifications. |
| `open_services` | integer | Total open equipment services. |

### Success Response Example

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
        "items": []
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

`GET /api/v1/auth/shops/shop-123/snapshot?include=messages&message_limit=1`

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

## `GET /shops/bootstrap`

Full path: `GET /api/v1/auth/shops/bootstrap`

Purpose: load the authenticated user's Shops landing data in one request.

Authorization: returns only shops where the authenticated user has a `shop_members` row.

Request body: none.

### Query Parameters

| Parameter | Type | Required | Default | Maximum | Invalid Values | Description |
|-----------|------|----------|---------|---------|----------------|-------------|
| `equipment_limit_per_shop` | integer | No | `50` | `250` | Empty value, non-integer, `0`, negative | Maximum compact equipment records returned per shop. Values above `250` are capped to `250`. |

The endpoint returns all visible shops. The `equipment` array is bounded per shop; `counts.vehicles` is the total vehicle count for the shop and may be larger than `equipment.length`.

### Success Response

Status: `200 OK`

Message: `Shops bootstrap retrieved successfully`

| Field | Type | Description |
|-------|------|-------------|
| `data.shops` | array | Shops visible to the authenticated user. Empty array if the user belongs to no shops. |
| `data.shops[].id` | string | Shop ID. |
| `data.shops[].name` | string | Shop display name. |
| `data.shops[].details` | string or null | Shop details. |
| `data.shops[].role` | string | Authenticated user's role in the shop. |
| `data.shops[].is_admin` | boolean | Whether `role` is `admin`. |
| `data.shops[].settings.admin_only_lists` | boolean | Whether only admins can manage lists for the shop. |
| `data.shops[].counts` | object | Shop-wide aggregate counts. |
| `data.shops[].equipment` | array | Bounded compact equipment summaries for the shop. |

### `counts` Shape

These counts are shop-wide counts. Fields marked optional are omitted by JSON when their value is zero.

| Field | Type | Optional When Zero | Description |
|-------|------|--------------------|-------------|
| `members` | integer | No | Total shop members. |
| `vehicles` | integer | No | Total shop vehicles. |
| `lists` | integer | No | Total shop lists. |
| `messages` | integer | No | Total shop messages. |
| `notifications` | integer | No | Total shop notifications. |
| `notification_items` | integer | Yes | Total shop notification items. |
| `open_services` | integer | No | Total open equipment services. |
| `services` | integer | Yes | Total equipment services. |
| `recent_changes` | integer | Yes | Total notification change records. |

### Success Response Example

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

### Empty Result Example

```json
{
  "status": 200,
  "message": "Shops bootstrap retrieved successfully",
  "data": {
    "shops": []
  }
}
```

## Error Responses

Authentication middleware errors use the existing auth response shape.

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

## Compression Notes

All aggregate endpoints in this document apply endpoint-scoped gzip compression when the request includes `Accept-Encoding: gzip`.

When gzip is applied, the response includes `Content-Encoding: gzip`. The JSON body after decompression has the same structure as the uncompressed response.

Compression support on these aggregate endpoints does not change compression behavior for existing Shops endpoints. For the existing equipment overview endpoint, continue using the contract in `docs/api/shop_equipment_overview_mobile.md`.

## Client Migration Guidance

- Use these endpoints only for screens that benefit from reducing several reads into one current snapshot.
- Keep existing narrow endpoint integrations where the screen already has exactly the data it needs.
- Treat unknown future fields as forward-compatible additions.
- Treat arrays as arrays even when empty. The aggregate service normalizes empty sections to `[]`.
- Do not send request bodies to these endpoints.
- Do not assume bounded section counts are total table counts unless the field is documented as a shop-wide count.
- Keep existing `GET /shops/equipment/overview` usage when the screen needs the unbounded compact shop/equipment overview.

## Performance Notes

- Aggregate payload size grows with the number of included sections and returned records.
- High-cardinality sections are bounded where documented: vehicle maintenance notifications, shop snapshot notifications, shop snapshot notification items, messages, services, recent changes, and bootstrap equipment per shop.
- `GET /shops/:shop_id/lists-with-items` returns all lists and all list items for the shop.
- `GET /shops/:shop_id/snapshot` returns all vehicles and all lists with items when those sections are included.
- `GET /shops/bootstrap` returns all shops visible to the authenticated user, with equipment bounded per shop.
- Request gzip for large aggregate screens to reduce transfer size. Gzip reduces bandwidth, but clients still need to decode the full JSON payload after decompression.
