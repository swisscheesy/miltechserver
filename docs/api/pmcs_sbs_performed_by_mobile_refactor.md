# PMCS SBS Performed-By Mobile Refactor Handoff

## Summary

Every PMCS SBS inspection endpoint that returns inspection data now also identifies who performed that inspection: a user id (`performed_by`) and that user's display username (`performed_by_username`).

This is two different kinds of change depending on the endpoint:

- **Renamed** on the single-inspection save and detail endpoints — the field previously named `created_by` is now `performed_by`, and a new `performed_by_username` field is returned alongside it.
- **Newly added** on the inspection list endpoint and the cross-shop equipment+history aggregate endpoint — these previously returned no performer information at all.

No request body or query parameter changes anywhere in this refactor. `performed_by` is never something mobile submits — it is always derived server-side from the authenticated caller at save time.

## What Changed For Mobile

| Endpoint | Method | Change |
|----------|--------|--------|
| `/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id` | `PUT` (save inspection) | **Renamed:** `created_by` → `performed_by`. **Added:** `performed_by_username`. |
| `/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id` | `GET` (get inspection) | **Renamed:** `created_by` → `performed_by`. **Added:** `performed_by_username`. |
| `/pmcs-sbs/equipment/:equipment_id/pmcs` | `GET` (list inspections) | **Added:** `performed_by`, `performed_by_username` (this endpoint previously returned neither). |
| `/shops/equipment-pmcs-history` | `GET` (equipment + PMCS history aggregate) | **Added:** `performed_by`, `performed_by_username` inside each `historical_pmcs` entry (previously returned neither). |

Endpoints not listed here — delete inspection, and all fault endpoints (list/save/delete/bulk-delete faults) — are unaffected by this refactor.

## Field Semantics

| Field | Type | Notes |
|-------|------|-------|
| `performed_by` | string, nullable | User id of whoever performed the inspection. Always the authenticated caller who first saved this inspection — never re-assigned by a later edit from a different user. |
| `performed_by_username` | string, nullable | Display username for `performed_by`. |

Both fields are **omitted entirely** from the response JSON when there is no value — not `null`, simply absent from the object. This happens if the user account that performed the inspection has since been deleted; the inspection itself is never removed or hidden because of this.

The two fields are always both present or both absent together.

## Save Inspection

`PUT /api/v1/auth/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id`

Request body is unchanged:

```json
{
  "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
  "performed_date": "2026-07-16T14:30:00Z"
}
```

Success response:

```json
{
  "status": 200,
  "data": {
    "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "equipment_id": "550e8400-e29b-41d4-a716-446655440000",
    "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
    "performed_date": "2026-07-16T14:30:00Z",
    "performed_by": "9f1c3a2e-user-uid",
    "performed_by_username": "jsmith",
    "created_at": "2026-07-16T14:31:02.123456Z",
    "updated_at": "2026-07-16T14:31:02.123456Z",
    "faults": []
  },
  "message": "Inspection saved"
}
```

## Get Inspection

`GET /api/v1/auth/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id`

Success response:

```json
{
  "status": 200,
  "data": {
    "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "equipment_id": "550e8400-e29b-41d4-a716-446655440000",
    "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
    "performed_date": "2026-07-16T14:30:00Z",
    "performed_by": "9f1c3a2e-user-uid",
    "performed_by_username": "jsmith",
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

Example of the omitted-field case — the account that performed this inspection has since been deleted:

```json
{
  "status": 200,
  "data": {
    "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "equipment_id": "550e8400-e29b-41d4-a716-446655440000",
    "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
    "performed_date": "2026-06-09T09:00:00Z",
    "created_at": "2026-06-09T09:05:44.001122Z",
    "updated_at": "2026-06-09T09:05:44.001122Z",
    "faults": []
  },
  "message": ""
}
```

## List Inspections

`GET /api/v1/auth/pmcs-sbs/equipment/:equipment_id/pmcs?guide_manual=pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json`

Success response:

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
        "created_at": "2026-07-16T14:31:02.123456Z",
        "performed_by": "9f1c3a2e-user-uid",
        "performed_by_username": "jsmith"
      },
      {
        "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
        "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
        "performed_date": "2026-07-09T09:00:00Z",
        "fault_count": 0,
        "created_at": "2026-07-09T09:05:44.001122Z",
        "performed_by": "b2f1c1b4-user-uid",
        "performed_by_username": "mwright"
      }
    ],
    "count": 2
  },
  "message": ""
}
```

## Equipment + PMCS History Aggregate

`GET /api/v1/auth/shops/equipment-pmcs-history`

Success response (`performed_by`/`performed_by_username` are new fields inside each `historical_pmcs` entry; everything else in this response shape is unchanged):

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
            "created_at": "2026-07-16T14:31:02.123456Z",
            "performed_by": "9f1c3a2e-user-uid",
            "performed_by_username": "jsmith"
          },
          {
            "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
            "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
            "performed_date": "2026-07-09T09:00:00Z",
            "fault_count": 0,
            "created_at": "2026-07-09T09:05:44.001122Z",
            "performed_by": "b2f1c1b4-user-uid",
            "performed_by_username": "mwright"
          }
        ]
      }
    ],
    "count": 1
  },
  "message": ""
}
```

## Mobile Implementation Checklist

- Rename any `created_by` model field to `performed_by` for the save and get-inspection responses.
- Add a `performed_by_username` field alongside it wherever `performed_by` is parsed.
- Add `performed_by` / `performed_by_username` parsing to the list-inspections and equipment-pmcs-history models — these are new fields, not renames, on those two endpoints.
- Treat both fields as optional/nullable in every model that parses them — do not assume presence.
- Display `performed_by_username` to the user rather than `performed_by`; the id has no inherent display meaning.
