# PMCS SBS Inspection History API

This document covers the PMCS SBS API after the introduction of inspection
history. It supersedes `pmcs_sbs_faults_guide_manual_mobile.md` and
`pmcs_sbs_bulk_fault_delete_mobile.md`.

## Summary

Faults are no longer a single overwritable state per `(equipment_id,
guide_manual, section_id, item_index)`. Instead, each PMCS performed on a
vehicle is its own **inspection** (`pmcs_sbs_inspections` row), identified by
a client-generated UUID (`pmcs_id`). Faults belong to one inspection.
Equipment can have many inspections over time, including inspections where
no faults were found ("clean" inspections).

Base URL: `/api/v1/auth`

## Inspection Identity

The mobile client generates a UUID (`pmcs_id`) when the user begins a new
PMCS on a vehicle. That id is sent with every fault save during the session
and identifies the inspection going forward. Starting a new PMCS later means
generating a new `pmcs_id` — there is no server-side session boundary.

`guide_manual` is fixed the first time a `pmcs_id` is used and cannot change
afterward. Sending a different `guide_manual` (or reusing a `pmcs_id` under a
different `equipment_id`) for an existing inspection returns `409`.

## Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `PUT` | `/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id` | Create or update inspection metadata. Use this for a clean (zero-fault) completion, or to correct `performed_date`. |
| `GET` | `/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id` | Get one inspection plus its full faults array. |
| `DELETE` | `/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id` | Delete an inspection and all its faults. |
| `GET` | `/pmcs-sbs/equipment/:equipment_id/pmcs` | List inspection history for a vehicle (most recent first). Optional `guide_manual` filter, `limit`/`offset` pagination (default limit 1000, max 1000). |
| `PUT` | `/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults` | Save (create/update) one fault. Creates the inspection implicitly on first use of a new `pmcs_id`. |
| `DELETE` | `/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults` | Delete one fault. |
| `DELETE` | `/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults/bulk` | Delete up to 100 faults from one inspection. |

`:equipment_id` is the selected Shops vehicle id: `shop_vehicle.id`.
`:pmcs_id` is the client-generated inspection UUID.

## Inspection Object

| Field | Type | Notes |
|-------|------|-------|
| `id` | string (UUID) | The `pmcs_id`. |
| `equipment_id` | string | Returned by the server. |
| `guide_manual` | string | Required on create; immutable after. |
| `performed_date` | string | ISO timestamp; required, client-supplied. |
| `created_by` | string, nullable | User id who first created the inspection. |
| `created_at` / `updated_at` | string | ISO timestamps; response only. |
| `faults` | array | Present on the single-inspection `GET`; omitted from the list endpoint. |

## Inspection Summary Object (list endpoint)

| Field | Type | Notes |
|-------|------|-------|
| `id` | string (UUID) | |
| `guide_manual` | string | |
| `performed_date` | string | ISO timestamp. |
| `fault_count` | integer | Number of faults on this inspection. |
| `created_at` | string | ISO timestamp. |

## Fault Object

| Field | Type | Notes |
|-------|------|-------|
| `pmcs_id` | string (UUID) | The inspection this fault belongs to. |
| `section_id` | string | Required for save/delete. |
| `item_index` | integer | Required for save/delete; must be `0` or greater. |
| `item_no` | string | Required for save. |
| `status` | string | Required for save; returned as normalized server value. |
| `fault_text` | string | Required for save. |
| `corrective_action` | string | Optional; blank string is accepted. |
| `created_at` / `updated_at` | string | ISO timestamps; response only. |

Valid saved `status` response values are `x`, `slash`, and `dash`. Accepted
save inputs: `X`/`x` → `x`, `/`/`slash` → `slash`, `-`/`dash` → `dash`.

## Error Responses

| HTTP | Cause |
|------|-------|
| `400` | Invalid `equipment_id`, malformed `pmcs_id`, invalid `guide_manual`, invalid fault fields, or invalid status. |
| `401` | Missing or invalid authentication. |
| `404` | Vehicle not found or not a shop member; or `pmcs_id` not found for this vehicle. |
| `409` | `guide_manual` (or vehicle ownership) mismatch for an existing `pmcs_id`. |
