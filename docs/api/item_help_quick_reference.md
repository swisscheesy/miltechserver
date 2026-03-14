# Item Help Endpoint Quick Reference

## Purpose
Public endpoint for retrieving help content by `code` within the item query feature.

## Endpoint
- Method: `GET`
- Path: `/api/v1/queries/items/help`
- Authentication: Not required (public)

## Request Structure

### Query Parameters
| Field | Type | Required | Description |
|---|---|---|---|
| `code` | string | Yes | Help lookup code. Server trims whitespace and uppercases before querying. |

### Request Rules
- `code` must be present.
- `code` must not be empty after trimming.

## Response Structure

### Success Response
| Field | Type | Required | Description |
|---|---|---|---|
| `status` | integer | Yes | HTTP status code (`200`). |
| `message` | string | Yes | Empty string on success. |
| `data` | object | Yes | Single help record (first match only). |

### Success `data` Object
| Field | Type | Nullable | Description |
|---|---|---|---|
| `code` | string | No | Help code used for lookup. |
| `literal` | string | Yes | Optional literal/help label text. |
| `description` | string | No | Help description text. |
| `regs` | string | Yes | Optional regulation/reference text. |

## Error Response Structures

### Invalid Request (`400`)
| Field | Type | Required | Description |
|---|---|---|---|
| `message` | string | Yes | Validation error message (`code is required`). |

### Not Found (`404`)
| Field | Type | Required | Description |
|---|---|---|---|
| `status` | integer | Yes | HTTP status code (`404`). |
| `message` | string | Yes | Not-found message. |
| `data` | object | Yes | Empty object. |

### Internal Error (`500`)
| Field | Type | Required | Description |
|---|---|---|---|
| `status` | integer | Yes | HTTP status code (`500`). |
| `message` | string | Yes | Internal error message. |
| `data` | null | Yes | Always null for this response. |

## Multi-Row Matching Behavior
- Database may contain multiple rows for the same `code`.
- The endpoint returns only one row.
- Returned row is deterministic: first row by `description` ascending.
