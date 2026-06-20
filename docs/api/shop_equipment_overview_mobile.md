# Shop Equipment Overview Mobile API Contract

## API Summary

Base URL: `https://<host>/api/v1/auth`

Endpoint: `GET /shops/equipment/overview`

Full path: `GET /api/v1/auth/shops/equipment/overview`

Authentication: Firebase ID token in the `Authorization` header using the Bearer scheme.

Request body: none.

Query parameters: none.

This endpoint returns every shop the authenticated user belongs to, with a compact list of equipment in each shop. It is intended for mobile overview screens that need a current, nested shop-and-equipment snapshot in one request.

## Features

- Returns all shops where the authenticated user has a `shop_members` row.
- Includes the authenticated user's role for each returned shop.
- Includes empty shops with `equipment_count: 0` and `equipment: []`.
- Includes compact equipment identity fields only: `id`, `admin`, `model`, `serial`, and `niin`.
- Filters membership server-side, so equipment from shops the user does not belong to is not returned.
- Uses one current server read per request. There is no pagination, cache contract, conditional response, or sync cursor.
- Supports endpoint-scoped gzip compression when the request includes `Accept-Encoding: gzip`.

## When Mobile Should Use This Endpoint

Use this endpoint when loading a shop equipment overview, search seed, picker, dashboard, or similar screen where the app needs all shops and their equipment grouped together.

Do not use this endpoint as a background sync replacement. It returns the current full overview snapshot and does not provide incremental changes.

## Request Format

| Item | Value |
|------|-------|
| Method | `GET` |
| Path | `/api/v1/auth/shops/equipment/overview` |
| Authentication | Required |
| Request body | None |
| Query parameters | None |
| Content type | Not required because there is no request body |

### Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | Yes | Firebase ID token using the Bearer scheme. |
| `Accept` | No | Use `application/json` if the client sets it. |
| `Accept-Encoding` | No | Include `gzip` if the client can handle gzip-compressed responses. |

## Success Response

Status: `200 OK`

The response uses the standard API envelope:

| Field | Type | Description |
|-------|------|-------------|
| `status` | integer | HTTP status code. For success, `200`. |
| `message` | string | Human-readable result message. |
| `data` | object | Response payload. |
| `data.shops` | array | Shops visible to the authenticated user. Empty array if the user belongs to no shops. |

### Success Response Example

```json
{
  "status": 200,
  "message": "Shop equipment overview retrieved successfully",
  "data": {
    "shops": [
      {
        "id": "9f8a0f30-25c6-43c8-a3ad-6a7c5c5b2b2a",
        "name": "Alpha Maintenance Shop",
        "details": "Motor pool maintenance team",
        "role": "admin",
        "equipment_count": 2,
        "equipment": [
          {
            "id": "1a6e3d8f-61df-48c2-98ac-8a0273f2ef60",
            "admin": "A123",
            "model": "M1097",
            "serial": "SER-10001",
            "niin": "012345678"
          },
          {
            "id": "c9233d31-a80a-40b0-87ef-4f432c4c9807",
            "admin": "B204",
            "model": "M1152A1",
            "serial": "SER-10002",
            "niin": "014567890"
          }
        ]
      },
      {
        "id": "4d8cb36d-891f-4c7d-9e2f-fef33c324f17",
        "name": "Bravo Shop",
        "details": null,
        "role": "member",
        "equipment_count": 0,
        "equipment": []
      }
    ]
  }
}
```

### Empty Result Example

If the authenticated user does not belong to any shops, the request still succeeds:

```json
{
  "status": 200,
  "message": "Shop equipment overview retrieved successfully",
  "data": {
    "shops": []
  }
}
```

## Response Data Fields

### Shop Object

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `id` | string | No | Shop ID. |
| `name` | string | No | Shop display name. |
| `details` | string | Yes | Shop details/description. Can be `null`. |
| `role` | string | No | Authenticated user's role in this shop, such as `admin` or `member`. |
| `equipment_count` | integer | No | Number of equipment records included in `equipment`. |
| `equipment` | array | No | Equipment records in this shop. Empty shops return `[]`, not `null`. |

### Equipment Object

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `id` | string | No | Equipment record ID. |
| `admin` | string | No | Equipment admin number or display identifier. |
| `model` | string | No | Equipment model. |
| `serial` | string | No | Equipment serial number. |
| `niin` | string | No | Equipment NIIN. |

## Ordering

Shops are ordered by newest shop first.

Equipment inside each shop is ordered by newest saved equipment first. If two equipment records have the same save time, equipment ID descending is used as a deterministic tie-breaker.

Mobile should not depend on this ordering for permanent storage semantics. If a screen needs a different order, sort locally for display.

## Compression

This endpoint supports gzip compression only for clients that request it.

To receive gzip, send `Accept-Encoding: gzip`.

When gzip is applied, the response includes `Content-Encoding: gzip`. The JSON body after decompression has the same structure as the uncompressed response.

Compression is scoped to this endpoint. Do not assume other `/shops` endpoints return gzip responses.

## Error Responses

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

### Server Error

Status: `500 Internal Server Error`

```json
{
  "status": 500,
  "message": "failed to retrieve shop equipment overview",
  "data": null
}
```

## Client Handling Notes

- Treat `data.shops` as the source array. It may be empty.
- Treat every `equipment` field as an array. Empty shops return `[]`.
- Do not send a request body.
- Do not add pagination or cursor parameters; this endpoint does not accept them.
- Use gzip when loading large shop/equipment sets to reduce transfer size.
- Cache locally only if the mobile app has its own UX reason to do so. The server response is a current snapshot and does not include cache validators.
- Ignore unknown future fields for forward compatibility.

## Performance Expectations

The endpoint is designed for overview payloads and has been validated against a representative load of 100 shops and 25,000 equipment records. Response size, memory use, and JSON decode cost grow with the number of equipment records returned.

For large memberships, prefer requesting gzip and avoid repeatedly refreshing the endpoint while the user is typing or scrolling unless the screen explicitly needs fresh server data.
