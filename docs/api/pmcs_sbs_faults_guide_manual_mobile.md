# PMCS SBS Faults Guide Scope API Changes

> **Superseded** — faults are now scoped to a `pmcs_id` (inspection), not a bare `(equipment_id, guide_manual)` pair. See `docs/api/pmcs_sbs_inspections_mobile.md` for the current contract.

This document covers the PMCS SBS fault API changes for mobile clients.

## Summary

PMCS SBS server persistence is now faults-only.

Faults are attached to existing Shops equipment records by `shop_vehicle.id`. Faults are also scoped to the loaded PMCS SBS guide/manual by `guide_manual`.

The server no longer stores PMCS SBS equipment records, guide progress, completed steps, or sync batches.

## Current Fault Endpoints

Base URL: `/api/v1/auth`

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/pmcs-sbs/equipment/:equipment_id/faults?guide_manual=...` | List faults for one Shops vehicle and one PMCS SBS guide/manual. |
| `PUT` | `/pmcs-sbs/equipment/:equipment_id/faults` | Create or update one fault. |
| `DELETE` | `/pmcs-sbs/equipment/:equipment_id/faults` | Delete one fault. |
| `DELETE` | `/pmcs-sbs/equipment/:equipment_id/faults/bulk` | Delete up to 100 faults for one Shops vehicle and one PMCS SBS guide/manual. |

`:equipment_id` is the selected Shops vehicle id: `shop_vehicle.id`.

## Fault Identity

A fault is uniquely identified by:

| Field | Source |
|-------|--------|
| `equipment_id` | Route parameter; the selected `shop_vehicle.id`. |
| `guide_manual` | PMCS SBS guide blob path, for example `pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json`. |
| `section_id` | Section id from the loaded guide. |
| `item_index` | Zero-based item index in the section. |

Saving the same identity updates the existing fault.

## Fault Object

| Field | Type | Notes |
|-------|------|-------|
| `equipment_id` | string | Returned by the server. |
| `guide_manual` | string | Required for list/save/delete; returned by the server. |
| `section_id` | string | Required for save/delete. |
| `item_index` | integer | Required for save/delete; must be `0` or greater. |
| `item_no` | string | Required for save. |
| `status` | string | Required for save; returned as normalized server value. |
| `fault_text` | string | Required for save. |
| `corrective_action` | string | Optional; blank string is accepted. |
| `created_at` | string | ISO timestamp; response only. |
| `updated_at` | string | ISO timestamp; response only. |

Valid saved `status` response values are `x`, `slash`, and `dash`.

Accepted save inputs for `status`:

| Input | Stored/returned |
|-------|-----------------|
| `X` | `x` |
| `x` | `x` |
| `/` | `slash` |
| `slash` | `slash` |
| `-` | `dash` |
| `dash` | `dash` |

## List Faults

Request:

`GET /api/v1/auth/pmcs-sbs/equipment/550e8400-e29b-41d4-a716-446655440000/faults?guide_manual=pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json`

Success response:

```json
{
  "status": 200,
  "data": {
    "faults": [
      {
        "equipment_id": "550e8400-e29b-41d4-a716-446655440000",
        "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
        "section_id": "before",
        "item_index": 0,
        "item_no": "1",
        "status": "x",
        "fault_text": "Oil leak observed",
        "corrective_action": "",
        "created_at": "2026-06-21T18:44:12.123456Z",
        "updated_at": "2026-06-21T19:12:03.654321Z"
      }
    ],
    "count": 1
  },
  "message": ""
}
```

No faults response:

```json
{
  "status": 200,
  "data": {
    "faults": [],
    "count": 0
  },
  "message": ""
}
```

## Save Fault

Request:

`PUT /api/v1/auth/pmcs-sbs/equipment/550e8400-e29b-41d4-a716-446655440000/faults`

```json
{
  "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
  "section_id": "before",
  "item_index": 0,
  "item_no": "1",
  "status": "X",
  "fault_text": "Oil leak observed",
  "corrective_action": ""
}
```

Success response:

```json
{
  "status": 200,
  "data": {
    "equipment_id": "550e8400-e29b-41d4-a716-446655440000",
    "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
    "section_id": "before",
    "item_index": 0,
    "item_no": "1",
    "status": "x",
    "fault_text": "Oil leak observed",
    "corrective_action": "",
    "created_at": "2026-06-21T18:44:12.123456Z",
    "updated_at": "2026-06-21T19:12:03.654321Z"
  },
  "message": "Fault saved"
}
```

## Delete Fault

Request:

`DELETE /api/v1/auth/pmcs-sbs/equipment/550e8400-e29b-41d4-a716-446655440000/faults`

```json
{
  "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
  "section_id": "before",
  "item_index": 0
}
```

Success response:

```json
{
  "message": "Fault deleted"
}
```

Deleting a missing fault still returns success when the user can access the parent vehicle.

## Bulk Delete Faults

Request:

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
      "section_id": "before",
      "item_index": 1
    }
  ]
}
```

Success response:

```json
{
  "message": "Faults deleted",
  "requested_count": 2,
  "deleted_count": 1
}
```

Rules:

- `guide_manual` is required once per request.
- `faults` must contain `1` to `100` entries.
- Each fault key requires `section_id` and `item_index`.
- Duplicate `(section_id, item_index)` entries are rejected.
- Missing fault keys do not fail the request when the vehicle is accessible. They are included in `requested_count` but not `deleted_count`.

## Error Responses

| Condition | HTTP status | Response |
|-----------|-------------|----------|
| Invalid JSON body | `400` | `{"message":"invalid request body"}` |
| Blank `equipment_id` | `400` | `{"message":"invalid id"}` |
| Missing or invalid `guide_manual` | `400` | `{"message":"invalid guide manual"}` |
| Blank required field or negative `item_index` | `400` | `{"message":"invalid request"}` |
| Invalid `status` | `400` | `{"message":"invalid fault status"}` |
| Empty `faults` on bulk delete | `400` | `{"message":"invalid request"}` |
| More than 100 faults on bulk delete | `400` | `{"message":"invalid request"}` |
| Duplicate bulk delete fault keys | `400` | `{"message":"invalid request"}` |
| Missing or unauthorized vehicle | `404` | `{"message":"pmcs sbs equipment not found"}` |
| Unexpected server error | `500` | `{"status":500,"data":null,"message":"internal Server Error"}` |

## Removed PMCS SBS Server Behaviors

These PMCS SBS server behaviors are no longer available:

| Removed behavior | Current behavior |
|------------------|------------------|
| PMCS SBS equipment create/update/delete/list/detail | Use Shops equipment records. |
| PMCS SBS completion endpoints | Keep guide progress and completed steps client-side. |
| PMCS SBS batch sync endpoint | Use direct fault list/save/delete endpoints only. |
