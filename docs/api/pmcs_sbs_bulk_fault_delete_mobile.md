# PMCS SBS Bulk Fault Delete Mobile API

> **Superseded** — faults are now scoped to a `pmcs_id` (inspection), not a bare `(equipment_id, guide_manual)` pair. See `docs/api/pmcs_sbs_inspections_mobile.md` for the current contract.

This document covers the PMCS SBS bulk fault delete endpoint for mobile clients.

## Endpoint

Base URL: `/api/v1/auth`

| Method | Path | Purpose |
|--------|------|---------|
| `DELETE` | `/pmcs-sbs/equipment/:equipment_id/faults/bulk` | Delete up to 100 faults for one Shops vehicle and one PMCS SBS guide/manual. |

`equipment_id` is the selected Shops vehicle id from `shop_vehicle.id`.

## Authentication

This endpoint requires the same authenticated request context as the existing PMCS SBS fault endpoints.

The authenticated user must be a member of the Shop that owns the selected vehicle. Missing vehicles and unauthorized vehicles both return `404`.

## Request Parameters

### Path Parameters

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `equipment_id` | string | Yes | Shops vehicle id. Blank ids are rejected. |

### Headers

| Header | Required | Notes |
|--------|----------|-------|
| `Authorization` | Yes | Existing authenticated API token header. |
| `Content-Type: application/json` | Yes | Required because the `DELETE` request has a JSON body. |

## Request Body

```json
{
  "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
  "faults": [
    {
      "section_id": "before",
      "item_index": 0
    },
    {
      "section_id": "before",
      "item_index": 1
    }
  ]
}
```

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `guide_manual` | string | Yes | PMCS SBS guide blob path. Must be a clean `pmcs_sbs/...json` path. |
| `faults` | array | Yes | Must contain `1` to `100` fault keys. |
| `faults[].section_id` | string | Yes | Section id from the loaded guide. Blank values are rejected. |
| `faults[].item_index` | integer | Yes | Zero-based item index in the section. Negative values are rejected. |

## Fault Identity

Each fault key is scoped by:

| Field | Source |
|-------|--------|
| `equipment_id` | Route parameter. |
| `guide_manual` | Request body, once per bulk request. |
| `section_id` | Each `faults` entry. |
| `item_index` | Each `faults` entry. |

The server trims string inputs before validation. Duplicate `(section_id, item_index)` entries in the same request are rejected after trimming.

## Success Response

Status: `200`

```json
{
  "message": "Faults deleted",
  "requested_count": 2,
  "deleted_count": 1
}
```

| Field | Type | Notes |
|-------|------|-------|
| `message` | string | Always `Faults deleted` on success. |
| `requested_count` | integer | Number of validated unique fault keys sent in the request. |
| `deleted_count` | integer | Number of existing rows actually deleted. |

Missing fault keys do not fail the request when the user can access the parent vehicle. They are included in `requested_count` but not `deleted_count`.

## Example Requests

### Delete Two Fault Keys

`DELETE /api/v1/auth/pmcs-sbs/equipment/550e8400-e29b-41d4-a716-446655440000/faults/bulk`

```json
{
  "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
  "faults": [
    {
      "section_id": "before",
      "item_index": 0
    },
    {
      "section_id": "during",
      "item_index": 3
    }
  ]
}
```

Response:

```json
{
  "message": "Faults deleted",
  "requested_count": 2,
  "deleted_count": 2
}
```

### Idempotent Delete With Missing Key

```json
{
  "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
  "faults": [
    {
      "section_id": "before",
      "item_index": 0
    },
    {
      "section_id": "missing",
      "item_index": 99
    }
  ]
}
```

Response when only the first key existed:

```json
{
  "message": "Faults deleted",
  "requested_count": 2,
  "deleted_count": 1
}
```

## Error Responses

| Condition | HTTP status | Response |
|-----------|-------------|----------|
| Invalid JSON body | `400` | `{"message":"invalid request body"}` |
| Blank `equipment_id` | `400` | `{"message":"invalid id"}` |
| Missing or invalid `guide_manual` | `400` | `{"message":"invalid guide manual"}` |
| Empty `faults` array | `400` | `{"message":"invalid request"}` |
| More than 100 fault keys | `400` | `{"message":"invalid request"}` |
| Blank `section_id` | `400` | `{"message":"invalid request"}` |
| Negative `item_index` | `400` | `{"message":"invalid request"}` |
| Duplicate `(section_id, item_index)` entries | `400` | `{"message":"invalid request"}` |
| Missing or unauthorized vehicle | `404` | `{"message":"pmcs sbs equipment not found"}` |
| Unexpected server error | `500` | `{"status":500,"data":null,"message":"internal Server Error"}` |

## Mobile Behavior Notes

- Send one request per selected guide/manual.
- Do not mix fault keys from different guide manuals in the same request.
- Use the same `guide_manual` value that was used to load/list the current guide's faults.
- Treat `deleted_count` lower than `requested_count` as a successful idempotent delete, not a partial failure.
- Refresh or locally remove only faults for the selected vehicle and selected `guide_manual`.
