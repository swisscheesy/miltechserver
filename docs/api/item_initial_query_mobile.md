# Item Initial Query API — Mobile Integration Guide

**Base URL:** `https://<host>/api/v1`  
**Authentication:** Bearer token required  
**Content-Type:** `application/json`

---

## Overview

The Item Initial Query endpoint provides a fast, lightweight item lookup designed for mobile search. It supports three lookup strategies:

| Strategy | `method` param | Description |
|---|---|---|
| NIIN lookup | `niin` | Exact match on a NIIN against the `niin_lookup` table |
| NIIN lookup with cancelled fallback | `niin` + `cancelled=true` | Same as above, but if no match is found it searches for the NIIN in the historical `cancelled_niin` data and returns the canonical replacement item(s) |
| Part number lookup | `part` | Looks up all items associated with a given part number |

> [!IMPORTANT]
> The `cancelled=true` parameter is a new opt-in feature. All existing requests that do **not** include `cancelled=true` behave exactly as before — no client changes are required for backwards compatibility.

---

## Endpoint

**`GET /queries/items/initial`**

---

## Query Parameters

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `method` | string | Yes | — | Lookup strategy. Must be `"niin"` or `"part"`. |
| `value` | string | Yes | — | The NIIN or part number to search for. |
| `cancelled` | boolean | No | `false` | **NIIN only.** When `"true"`, enables the cancelled NIIN fallback. Ignored when `method=part`. |

### Parameter Notes

- `cancelled` is only evaluated when `method=niin`. Pass it as the literal string `"true"` — any other value (`"false"`, `"0"`, blank, or absent) disables the fallback and uses the standard NIIN lookup.
- NIINs should be passed without hyphens (e.g. `013469317`, not `01-346-9317`).
- Part numbers are matched case-insensitively on the server.

---

## Response Envelope

All responses share the same outer structure:

```json
{
  "status": 200,
  "message": "",
  "data": { ... }
}
```

| Field | Type | Description |
|---|---|---|
| `status` | integer | HTTP status code mirrored in the body |
| `message` | string | Empty string on success; may contain a human-readable message on errors |
| `data` | object or array | The item result(s). Shape depends on the lookup strategy — see sections below. |

---

## Strategy 1 — Standard NIIN Lookup

**Request:** `method=niin`  
**Returns:** A single item object when found.

### Example Request

```
GET /api/v1/queries/items/initial?method=niin&value=013469317
```

### Success Response — `200 OK`

`data` is a **single object**.

```json
{
  "status": 200,
  "message": "",
  "data": {
    "niin": "013469317",
    "fsc": "1005",
    "item_name": "RIFLE,5.56 MILLIMETER",
    "has_amdf": true,
    "has_flis": true
  }
}
```

### Error Responses

| Condition | Status | Body |
|---|---|---|
| NIIN not found in `niin_lookup` | `404` | `{"status": 404, "message": ""}` |
| Database unavailable | `500` | `{"status": 500, "message": "internal server error"}` |

---

## Strategy 2 — Cancelled NIIN Fallback

**Request:** `method=niin&cancelled=true`  
**Returns:** An **array** of item objects. The array always contains at least one element on success.

### How It Works

1. The server first performs the standard NIIN lookup against `niin_lookup`.
2. **If a match is found**, it is returned immediately as a single-element array.
3. **If no match is found**, the server searches the historical cancelled NIIN data for the given NIIN. A NIIN may appear as cancelled in multiple records.
4. For each unique canonical NIIN discovered in step 3, the server re-queries `niin_lookup` and collects the results.
5. All resolved items are returned together in one array.

This allows users to search for old, superseded, or invalidated NIINs and be automatically directed to the current replacement item(s).

### Example Request

```
GET /api/v1/queries/items/initial?method=niin&value=007354295&cancelled=true
```

### Scenario A — Primary Hit (NIIN exists in niin_lookup)

The cancelled fallback is **not triggered**. The item is found directly and returned as a single-element array.

```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "niin": "007354295",
      "fsc": "2530",
      "item_name": "WHEEL ASSEMBLY,VEHICULAR",
      "has_amdf": true,
      "has_flis": false
    }
  ]
}
```

### Scenario B — Cancelled NIIN Fallback Hit

The NIIN was not found in `niin_lookup`, but the server found it in the historical cancelled NIIN data and resolved one or more canonical replacements.

```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "niin": "013578921",
      "fsc": "2530",
      "item_name": "WHEEL ASSEMBLY,VEHICULAR",
      "has_amdf": true,
      "has_flis": true
    }
  ]
}
```

### Scenario C — Multiple Canonical NIINs Resolved

The cancelled NIIN matched multiple records, each referencing a different canonical NIIN. All resolved items are returned.

```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "niin": "013578921",
      "fsc": "2530",
      "item_name": "WHEEL ASSEMBLY,VEHICULAR",
      "has_amdf": true,
      "has_flis": true
    },
    {
      "niin": "014112885",
      "fsc": "2530",
      "item_name": "WHEEL ASSEMBLY,HEAVY DUTY",
      "has_amdf": false,
      "has_flis": true
    }
  ]
}
```

### Error Responses

| Condition | Status | Body |
|---|---|---|
| NIIN not in `niin_lookup` **and** not found in cancelled records | `404` | `{"status": 404, "message": ""}` |
| Cancelled NIINs found but none resolve to an active `niin_lookup` entry | `404` | `{"status": 404, "message": ""}` |
| Database unavailable | `500` | `{"status": 500, "message": "internal server error"}` |

---

## Strategy 3 — Part Number Lookup

**Request:** `method=part`  
**Returns:** An **array** of item objects. Multiple items may share the same part number.

### Example Request

```
GET /api/v1/queries/items/initial?method=part&value=12345678
```

### Success Response — `200 OK`

`data` is an **array**.

```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "niin": "013469317",
      "fsc": "1005",
      "item_name": "RIFLE,5.56 MILLIMETER",
      "has_amdf": true,
      "has_flis": true
    },
    {
      "niin": "014882310",
      "fsc": "1005",
      "item_name": "RIFLE,5.56 MILLIMETER,CARBINE",
      "has_amdf": true,
      "has_flis": false
    }
  ]
}
```

### Error Responses

| Condition | Status | Body |
|---|---|---|
| Part number not found | `404` | `{"status": 404, "message": "no item found"}` |
| Database unavailable | `500` | `{"status": 500, "message": "internal server error"}` |

---

## Item Object Fields

All three strategies return items using the same object shape:

| Field | Type | Nullable | Description |
|---|---|---|---|
| `niin` | string | Yes | National Item Identification Number (9 digits, no hyphens) |
| `fsc` | string | Yes | Federal Supply Class (4-digit code) |
| `item_name` | string | Yes | Official item nomenclature |
| `has_amdf` | boolean | Yes | Whether Army Master Data File data is available for this item |
| `has_flis` | boolean | Yes | Whether Federal Logistics Information System data is available for this item |

> [!NOTE]
> All item fields are nullable. A field being `null` means the data was not available in the source system for that item — it does not indicate an error.

---

## `data` Shape Reference

| Strategy | `cancelled` param | `data` shape |
|---|---|---|
| `method=niin` | absent or `false` | Single object `{}` |
| `method=niin` | `true` | Array `[{}, ...]` — always an array, even for a single result |
| `method=part` | (ignored) | Array `[{}, ...]` |

> [!IMPORTANT]
> When `cancelled=true` is used with `method=niin`, the `data` field is **always an array**, including when the primary NIIN is found directly. Design your parser to handle an array for this path.

---

## Error Handling Guidance

| Status | Recommended Behaviour |
|---|---|
| `404` | The NIIN or part number was not found (and could not be resolved via cancelled records when applicable). Inform the user. |
| `500` | A server-side error occurred. Show a generic retry message; do not display raw body content to end users. |

---

## Recommended Mobile Workflow

### Basic NIIN search

1. Call `GET /queries/items/initial?method=niin&value=<niin>`.
2. On `200`, navigate to the item detail screen using the single object in `data`.
3. On `404`, inform the user the NIIN was not found, and offer the option to retry with the cancelled fallback.

### NIIN search with cancelled fallback

1. Call `GET /queries/items/initial?method=niin&value=<niin>&cancelled=true`.
2. On `200`, parse `data` as an **array**.
   - If the array contains one item, navigate to the item detail screen.
   - If the array contains multiple items, present the user with a selection list indicating the NIIN was replaced by multiple items.
3. On `404`, inform the user the NIIN could not be found in any record (active or historical).

### Part number search

1. Call `GET /queries/items/initial?method=part&value=<part>`.
2. On `200`, parse `data` as an **array**.
   - If one item is returned, navigate directly to the item detail screen.
   - If multiple items are returned, present a disambiguation list.
3. On `404`, inform the user no items are associated with that part number.

---

## Quick Reference

| Goal | Request |
|---|---|
| Look up NIIN 013469317 | `?method=niin&value=013469317` |
| Look up NIIN, including cancelled history | `?method=niin&value=013469317&cancelled=true` |
| Look up part number 12345678 | `?method=part&value=12345678` |
