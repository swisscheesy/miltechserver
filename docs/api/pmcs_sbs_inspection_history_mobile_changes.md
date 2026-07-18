# PMCS SBS Inspection History — Mobile API Change Reference

This document explains what changed in the PMCS SBS server API, why it changed, and the complete new request/response contract. It is written for the mobile team integrating against these changes. It supersedes `pmcs_sbs_faults_guide_manual_mobile.md` and `pmcs_sbs_bulk_fault_delete_mobile.md`, both of which now carry a pointer to this document (and to `pmcs_sbs_inspections_mobile.md`, the terse endpoint reference).

Base URL for every endpoint below: `/api/v1/auth`

---

## Why This Changed

Previously, a PMCS SBS fault was identified by `(equipment_id, guide_manual, section_id, item_index)`. Saving a fault for a given checklist item always overwrote whatever was previously stored for that same item — there was no concept of "this PMCS, performed on this date" as a distinct record. Two consequences followed directly from that model:

- There was no way to see a vehicle's PMCS history. The server only ever held "the current state of each checklist item," never a record of past inspections.
- A "clean" PMCS — one where the technician found zero faults — left **no trace at all** on the server, because there was nothing to overwrite and nothing new to insert.

The product requirement is a historical view: every PMCS performed on a vehicle, the date it was performed, and the faults found during it — including inspections where nothing was found. That requires a concept of an inspection *event*, not just a per-item state.

**No fault history was migrated.** The old fault rows were current-state-only snapshots with no performed-date concept, so there was nothing meaningful to carry forward into the new model. The `pmcs_sbs_faults` table was dropped and recreated with a new shape. Any fault data visible in the old API is gone from the server; this is a clean cutover, not a backward-compatible migration.

---

## What Changed, Conceptually

**Before:** A fault belonged directly to `(equipment_id, guide_manual)`. There was no server-side grouping of faults by "this specific PMCS session."

**After:** Every PMCS performed on a vehicle is its own **inspection** — a `pmcs_sbs_inspections` row identified by a UUID (`pmcs_id`). Faults belong to exactly one inspection, not directly to the vehicle. A vehicle can have any number of inspections over time, including inspections with zero faults.

**Inspection identity is client-generated.** The mobile client generates the `pmcs_id` UUID when a PMCS begins on a vehicle. That same `pmcs_id` is sent with every fault save during that PMCS, and identifies which inspection those faults belong to. Starting a new PMCS later means generating a new `pmcs_id` — the server has no independent concept of a "session" or a time-based boundary between one PMCS and the next. Whatever `pmcs_id` is sent is the inspection.

**`guide_manual` is now fixed per inspection.** The first request (of any kind — inspection save or fault save) that uses a given `pmcs_id` sets that inspection's `guide_manual`. Every subsequent request using that same `pmcs_id` must send the same `guide_manual`, or it fails (see Error Responses). This also means a `pmcs_id` cannot be reused across two different `equipment_id`s — that also produces the same conflict failure.

**`performed_date` is always client-supplied.** The server never stamps `performed_date` from its own clock. This exists because a PMCS may be performed offline and synced later, so the "true" performed date can only come from the client.

---

## Endpoint Changes: Old vs. New

| Old endpoint | Status | New endpoint |
|---|---|---|
| `GET /pmcs-sbs/equipment/:equipment_id/faults?guide_manual=...` | **Removed** | No direct replacement — faults are read per-inspection via `GET .../pmcs/:pmcs_id`, or inspection history (with fault counts, not full fault bodies) via `GET .../pmcs` |
| `PUT /pmcs-sbs/equipment/:equipment_id/faults` | **Moved** | `PUT /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults` |
| `DELETE /pmcs-sbs/equipment/:equipment_id/faults` | **Moved** | `DELETE /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults` |
| `DELETE /pmcs-sbs/equipment/:equipment_id/faults/bulk` | **Moved, body changed** | `DELETE /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults/bulk` (`guide_manual` removed from the request body — it's implied by `pmcs_id` now) |
| *(none)* | **New** | `PUT /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id` — create/update inspection metadata |
| *(none)* | **New** | `GET /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id` — get one inspection with its full faults array |
| *(none)* | **New** | `DELETE /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id` — delete an inspection and all its faults |
| *(none)* | **New** | `GET /pmcs-sbs/equipment/:equipment_id/pmcs` — list inspection history for a vehicle |

Every moved/new endpoint now carries `:pmcs_id` as a path parameter. `:equipment_id` is unchanged — it is still the Shops vehicle id (`shop_vehicle.id`).

---

## Full Endpoint Reference

### 1. `PUT /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id`

Creates the inspection if `:pmcs_id` doesn't exist yet, or updates `performed_date` on an existing one. This is the endpoint to call for a "clean" PMCS (zero faults found) — with no faults ever saved against this `pmcs_id`, this is the only way that inspection gets recorded at all. It's also how `performed_date` is corrected after the fact.

Request body:

```json
{
  "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
  "performed_date": "2026-07-16T14:30:00Z"
}
```

Success response (`200`):

```json
{
  "status": 200,
  "data": {
    "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "equipment_id": "550e8400-e29b-41d4-a716-446655440000",
    "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
    "performed_date": "2026-07-16T14:30:00Z",
    "created_by": "9f1c3a2e-user-uid",
    "created_at": "2026-07-16T14:31:02.123456Z",
    "updated_at": "2026-07-16T14:31:02.123456Z",
    "faults": []
  },
  "message": "Inspection saved"
}
```

### 2. `GET /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id`

Returns one inspection with its complete faults array.

Success response (`200`):

```json
{
  "status": 200,
  "data": {
    "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "equipment_id": "550e8400-e29b-41d4-a716-446655440000",
    "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
    "performed_date": "2026-07-16T14:30:00Z",
    "created_by": "9f1c3a2e-user-uid",
    "created_at": "2026-07-16T14:31:02.123456Z",
    "updated_at": "2026-07-16T14:45:18.987654Z",
    "faults": [
      {
        "pmcs_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
        "section_id": "before",
        "item_index": 0,
        "item_no": "1",
        "status": "x",
        "fault_text": "Oil leak observed",
        "corrective_action": "",
        "created_at": "2026-07-16T14:44:02.123456Z",
        "updated_at": "2026-07-16T14:44:02.123456Z"
      }
    ]
  },
  "message": ""
}
```

A clean inspection (no faults saved) returns `"faults": []`.

### 3. `DELETE /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id`

Deletes an inspection and all faults belonging to it. No request body.

Success response (`200`):

```json
{
  "message": "Inspection deleted"
}
```

### 4. `GET /pmcs-sbs/equipment/:equipment_id/pmcs`

Lists inspection history for a vehicle, most recent `performed_date` first. Does **not** include each inspection's full faults array — only a count. To see a specific inspection's faults, call endpoint 2 with its `id`.

Query parameters:

| Parameter | Required | Default | Notes |
|---|---|---|---|
| `guide_manual` | No | — | Filters to inspections for one specific guide/manual. Omit to get all guide/manuals for this vehicle. |
| `limit` | No | `1000` | Min `1`, max `1000`. |
| `offset` | No | `0` | Min `0`. |

Example request:

`GET /api/v1/auth/pmcs-sbs/equipment/550e8400-e29b-41d4-a716-446655440000/pmcs?limit=50&offset=0`

Success response (`200`):

```json
{
  "status": 200,
  "data": {
    "inspections": [
      {
        "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
        "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
        "performed_date": "2026-07-16T14:30:00Z",
        "fault_count": 3,
        "created_at": "2026-07-16T14:31:02.123456Z"
      },
      {
        "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
        "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
        "performed_date": "2026-07-09T09:00:00Z",
        "fault_count": 0,
        "created_at": "2026-07-09T09:05:44.001122Z"
      }
    ],
    "count": 2
  },
  "message": ""
}
```

`count` is the number of inspections in this response page, not the total across all pages.

### 5. `PUT /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults`

Saves (creates or updates) one fault under the given inspection. If this is the first request of any kind to use this `pmcs_id`, the inspection is created implicitly using the `guide_manual` and `performed_date` in this same request body — there is no need to call endpoint 1 first for a PMCS that has faults.

Request body:

```json
{
  "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
  "performed_date": "2026-07-16T14:30:00Z",
  "section_id": "before",
  "item_index": 0,
  "item_no": "1",
  "status": "X",
  "fault_text": "Oil leak observed",
  "corrective_action": ""
}
```

`guide_manual` and `performed_date` are required on **every** fault save, not just the first — every save re-applies the same create-or-update-inspection step described in endpoint 1.

Success response (`200`):

```json
{
  "status": 200,
  "data": {
    "pmcs_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "section_id": "before",
    "item_index": 0,
    "item_no": "1",
    "status": "x",
    "fault_text": "Oil leak observed",
    "corrective_action": "",
    "created_at": "2026-07-16T14:44:02.123456Z",
    "updated_at": "2026-07-16T14:44:02.123456Z"
  },
  "message": "Fault saved"
}
```

Saving the same `(pmcs_id, section_id, item_index)` again updates the existing fault row rather than creating a new one.

### 6. `DELETE /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults`

Deletes one fault.

Request body:

```json
{
  "section_id": "before",
  "item_index": 0
}
```

Success response (`200`):

```json
{
  "message": "Fault deleted"
}
```

Deleting a fault key that doesn't exist still returns success, as long as the caller can access the vehicle.

### 7. `DELETE /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults/bulk`

Deletes up to 100 faults from one inspection in a single request. Unlike the old bulk-delete endpoint, `guide_manual` is not part of the request body — the inspection is already identified by `:pmcs_id` in the path.

Request body:

```json
{
  "faults": [
    { "section_id": "before", "item_index": 0 },
    { "section_id": "during", "item_index": 3 }
  ]
}
```

Success response (`200`):

```json
{
  "message": "Faults deleted",
  "requested_count": 2,
  "deleted_count": 2
}
```

`requested_count` is the number of validated, unique fault keys sent in the request. `deleted_count` is the number of rows that actually existed and were removed. A fault key that doesn't exist does not fail the request — it's counted in `requested_count` but not `deleted_count`.

---

### 8. `GET /shops/equipment-pmcs-history`

Not part of the PMCS SBS route group (it lives under `/shops`, alongside the other Shops aggregate endpoints), but returns PMCS inspection history batched across every vehicle the user can access, so it's documented here for convenience.

Returns every vehicle the authenticated user has access to, across every shop they belong to, each with its PMCS inspection history (summary form — same fields as the list endpoint above, no nested faults) nested underneath. Equipment with no PMCS history yet is included with an empty `historical_pmcs` array. There are no query parameters, no pagination, and no per-vehicle cap — every accessible vehicle and its complete inspection history is always returned in one response.

Success response (`200`):

```json
{
  "status": 200,
  "data": {
    "equipment": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "shop_id": "b2f1c1b4-1111-4a2b-9c3d-000000000001",
        "admin": "B1",
        "model": "M1152A1",
        "serial": "SER-B1",
        "niin": "",
        "historical_pmcs": [
          {
            "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
            "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
            "performed_date": "2026-07-16T14:30:00Z",
            "fault_count": 3,
            "created_at": "2026-07-16T14:31:02.123456Z"
          },
          {
            "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
            "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
            "performed_date": "2026-07-09T09:00:00Z",
            "fault_count": 0,
            "created_at": "2026-07-09T09:05:44.001122Z"
          }
        ]
      },
      {
        "id": "9d8e7f6a-2222-4b3c-8d4e-000000000002",
        "shop_id": "b2f1c1b4-1111-4a2b-9c3d-000000000001",
        "admin": "B2",
        "model": "M998",
        "serial": "SER-B2",
        "niin": "",
        "historical_pmcs": []
      }
    ],
    "count": 2
  },
  "message": ""
}
```

`historical_pmcs` is ordered most-recent-`performed_date`-first within each vehicle, same as the inspection list endpoint. `count` is the number of equipment entries in this response (there is no pagination, so this is always the caller's full accessible equipment count). `shop_id` identifies which shop each vehicle belongs to — the equipment list itself is flat, not grouped by shop.

To see full fault detail (not just a count) for one specific inspection, call `GET /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id` (endpoint 2 above) with that inspection's `id`.

Error responses: `401` (`{"message":"unauthorized"}`) for missing/invalid authentication; `500` for an unexpected server error. There is no `400`/`404`/`409` case for this endpoint — it takes no parameters and its equipment list is inherently scoped to what the caller can access.

---

## Object Reference

### Inspection Object

| Field | Type | Notes |
|---|---|---|
| `id` | string (UUID) | The `pmcs_id`. Client-generated; never generated by the server. |
| `equipment_id` | string | The vehicle this inspection belongs to. |
| `guide_manual` | string | Set on first use of this `pmcs_id`; immutable after. |
| `performed_date` | string (ISO 8601 timestamp) | Always client-supplied. |
| `created_by` | string (nullable) | User id who first created the inspection. This field is **omitted entirely** from the response JSON when there is no value (not `null`, simply absent). |
| `created_at` / `updated_at` | string (ISO 8601 timestamp) | Server-assigned, response only. |
| `faults` | array of Fault Object | Present on the single-inspection `GET`. Not present on the list endpoint. |

### Inspection Summary Object (list endpoint only)

| Field | Type | Notes |
|---|---|---|
| `id` | string (UUID) | |
| `guide_manual` | string | |
| `performed_date` | string (ISO 8601 timestamp) | |
| `fault_count` | integer | Number of faults on this inspection. `0` for a clean inspection. |
| `created_at` | string (ISO 8601 timestamp) | |

### Fault Object

| Field | Type | Notes |
|---|---|---|
| `pmcs_id` | string (UUID) | The inspection this fault belongs to. |
| `section_id` | string | Required for save/delete. |
| `item_index` | integer | Required for save/delete. Must be `0` or greater. |
| `item_no` | string | Required for save. |
| `status` | string | Required for save. Returned as the normalized server value (see below). |
| `fault_text` | string | Required for save. |
| `corrective_action` | string | Optional; blank string is accepted. |
| `created_at` / `updated_at` | string (ISO 8601 timestamp) | Server-assigned, response only. |

`status` values, saved/returned form and accepted input forms (unchanged from before this change):

| Accepted input | Stored/returned |
|---|---|
| `X` or `x` | `x` |
| `/` or `slash` | `slash` |
| `-` or `dash` | `dash` |

Any other input value is rejected.

---

## Validation Rules

- `equipment_id` must be non-blank.
- `pmcs_id` (path parameter) must parse as a valid UUID. A blank or malformed value is rejected.
- `guide_manual` must be non-blank, must not contain a backslash, must start with `pmcs_sbs/`, must end with `.json`, and must not contain path traversal segments (e.g. `..`) or redundant separators.
- `performed_date` must be present (a zero/unset timestamp is rejected).
- `section_id` and `item_no` must be non-blank; `item_index` must be `0` or greater; `fault_text` must be non-blank.
- Bulk delete's `faults` array must contain between `1` and `100` entries. Duplicate `(section_id, item_index)` entries within the same request are rejected.
- All string fields are trimmed of surrounding whitespace before validation.

---

## Error Responses

| HTTP status | Response body | Cause |
|---|---|---|
| `400` | `{"message":"invalid request body"}` | Malformed JSON body. |
| `400` | `{"message":"invalid id"}` | Blank `equipment_id`. |
| `400` | `{"message":"invalid pmcs id"}` | Blank or malformed `pmcs_id`. |
| `400` | `{"message":"invalid guide manual"}` | `guide_manual` fails the format rules above. |
| `400` | `{"message":"invalid request"}` | A required field is blank/missing, `item_index` is negative, `performed_date` is missing, or bulk-delete's `faults` array is empty, over 100 entries, or contains a duplicate key. |
| `400` | `{"message":"invalid fault status"}` | `status` is not one of the accepted values. |
| `400` | `{"message":"invalid query parameters"}` | The list endpoint's query string doesn't bind (e.g. `limit`/`offset` out of range). |
| `401` | `{"message":"unauthorized"}` | Missing or invalid authentication. |
| `404` | `{"message":"pmcs sbs equipment not found"}` | The vehicle doesn't exist, or the caller isn't a member of the shop that owns it. |
| `404` | `{"message":"pmcs sbs inspection not found"}` | `pmcs_id` doesn't exist, or exists but belongs to a different `equipment_id` than the one in the path. |
| `409` | `{"message":"pmcs sbs inspection conflict"}` | The request's `guide_manual` doesn't match what this `pmcs_id` was already created with, **or** this `pmcs_id` already belongs to a different vehicle. |
| `500` | `{"status":500,"data":null,"message":"internal Server Error"}` | Unexpected server error. |

A mismatched `pmcs_id` (belonging to another vehicle) always reads as `404` on read/delete paths and `409` on write paths that would otherwise mutate it — it never silently succeeds and never exposes data across vehicles.

---

## Behavioral Notes

- There is no server-side session or time boundary between inspections. Whatever `pmcs_id` a request uses is the inspection it operates on — a new PMCS on the same vehicle the same day requires a new `pmcs_id`.
- `guide_manual` cannot change once a `pmcs_id` has been used, and a `pmcs_id` cannot move to a different vehicle. Both produce a `409`.
- A clean (zero-fault) PMCS produces no record at all unless `PUT .../pmcs/:pmcs_id` (endpoint 1) is called for it.
- Deleting a fault or fault key that doesn't exist is not an error, as long as the vehicle is accessible to the caller.
- All old fault history was discarded during this change and cannot be recovered from the server.
