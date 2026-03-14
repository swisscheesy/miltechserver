# Item Help API - Mobile Integration Guide

**Base URL:** `https://<host>/api/v1`  
**Authentication:** None (public endpoint)  
**Content-Type:** `application/json`

## Endpoint

`GET /queries/items/help`

## Query Parameters

| Parameter | Type | Required | Description |
|---|---|---|---|
| `code` | string | Yes | Help code to search for. Input is normalized by trimming whitespace and converting to uppercase before database lookup. |

## Behavior

- The server normalizes `code` as uppercase before querying the `help` table.
- If multiple rows match a code, only the first row is returned.
- First row is deterministic: ordered by `description ASC`.

## Success Response (`200`)

```json
{
  "status": 200,
  "message": "",
  "data": {
    "code": "AB12",
    "literal": "Literal Value",
    "description": "Help description",
    "regs": "AR 123"
  }
}
```

## Error Responses

### Missing or empty code (`400`)

```json
{
  "message": "code is required"
}
```

### Code not found (`404`)

```json
{
  "status": 404,
  "message": "No item found",
  "data": {}
}
```

### Internal server error (`500`)

```json
{
  "status": 500,
  "message": "internal Server Error",
  "data": null
}
```
