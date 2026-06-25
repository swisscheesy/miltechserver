# PMCS SBS Faults Mobile Refactor Handoff

## Summary

The PMCS SBS authenticated server API has been refactored to store faults only.

Mobile should no longer use the server for PMCS SBS equipment records, guide progress, completed steps, or sync batches. Equipment now comes from Shops, and PMCS SBS fault records are attached to existing `shop_vehicle.id` values.

The public PMCS SBS library/content API is unchanged. Guide JSON loading remains separate from this faults API, but fault records now include the selected guide/manual path so the app only shows faults for the loaded manual.

## What Changed For Mobile

| Area | Previous server behavior | Current server behavior |
|------|--------------------------|-------------------------|
| PMCS SBS equipment | Server-owned PMCS equipment records | Removed. Use Shops equipment, specifically `shop_vehicle.id`. |
| PMCS SBS progress/completions | Server-side completed-step tracking | Removed. Keep guide progress and completed steps client-side. |
| PMCS SBS sync | Batch sync endpoint for equipment, completions, and faults | Removed. Use direct fault list/save/delete endpoints only. |
| PMCS SBS faults | Child records under PMCS SBS equipment ids | Faults are now keyed to Shops equipment ids and the selected PMCS SBS guide/manual path. |

## Required Mobile Refactor

1. Replace any PMCS SBS server equipment id usage with the selected Shops equipment id.
2. Stop calling PMCS SBS equipment create, update, delete, list, or detail endpoints.
3. Stop calling PMCS SBS completion endpoints.
4. Stop calling `POST /pmcs-sbs/sync`.
5. Keep PMCS SBS guide progress and completed-step state in local/client storage.
6. Send the current guide/manual path as `guide_manual` for every fault list, save, and delete operation.
7. Use the faults endpoints below for only PMCS SBS fault persistence.
8. Treat `404 {"message":"pmcs sbs equipment not found"}` as either a missing vehicle or a vehicle the user is not authorized to access.

## Equipment Source

PMCS SBS faults now use Shops equipment as their parent record.

| Mobile concept | Server value to use |
|----------------|---------------------|
| Selected PMCS SBS equipment id | `shop_vehicle.id` |
| Equipment list/source | Shops equipment APIs |
| Fault parent id in PMCS routes | `:equipment_id`, where the value is `shop_vehicle.id` |

The server does not create or modify Shops equipment through the PMCS SBS faults API.

## Guide/Manual Scope

PMCS SBS faults are scoped to the selected guide/manual.

| Mobile concept | Server value to use |
|----------------|---------------------|
| Loaded PMCS SBS guide | The guide blob path, for example `pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json`. |
| Fault guide/manual field | `guide_manual` |

The server requires `guide_manual` on every fault operation. It must be a clean `pmcs_sbs/...*.json` path. Missing, non-PMCS, non-JSON, or traversal paths are rejected with `400 {"message":"invalid guide manual"}`.

When the user changes manuals for the same vehicle, mobile should call the list endpoint again with the new `guide_manual` value and replace the visible fault state with that response.

## Authorization Behavior

Every fault operation requires Firebase authentication.

The authenticated user must be a member of the shop that owns the target `shop_vehicle`.

Any shop member can list, save, and delete PMCS SBS faults for that vehicle. There is no admin-only fault restriction in the PMCS SBS faults API.

Missing vehicles and unauthorized vehicles intentionally return the same response:

```json
{"message":"pmcs sbs equipment not found"}
```

## Current Endpoints

Base URL: `https://<host>/api/v1/auth`

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/pmcs-sbs/equipment/:equipment_id/faults?guide_manual=...` | List PMCS SBS faults for one Shops vehicle and one guide/manual. |
| `PUT` | `/pmcs-sbs/equipment/:equipment_id/faults` | Create or update one PMCS SBS fault. |
| `DELETE` | `/pmcs-sbs/equipment/:equipment_id/faults` | Delete one PMCS SBS fault. |

The route keeps the word `equipment`, but `:equipment_id` now means `shop_vehicle.id`.

## Fault Identity

A fault is uniquely identified by:

| Field | Source |
|-------|--------|
| `equipment_id` | Route parameter; must be `shop_vehicle.id`. |
| `guide_manual` | Current PMCS SBS guide blob path. |
| `section_id` | PMCS section id from the guide. |
| `item_index` | Zero-based PMCS item index in the section. |

Saving the same `equipment_id`, `guide_manual`, `section_id`, and `item_index` updates the existing fault.

## Fault Fields

| Field | Type | Request behavior | Response behavior |
|-------|------|------------------|-------------------|
| `equipment_id` | string | Route parameter only | Returned in fault objects. |
| `guide_manual` | string | Required for save/delete body and list query; blank/invalid rejected | Returned in fault objects. |
| `section_id` | string | Required for save/delete; blank rejected | Returned. |
| `item_index` | integer | Required for save/delete; must be `0` or greater | Returned. |
| `item_no` | string | Required for save; blank rejected | Returned. |
| `status` | string | Required for save; normalized by server | Returned as normalized value. |
| `fault_text` | string | Required for save; blank rejected | Returned. |
| `corrective_action` | string | Optional; blank accepted | Returned. |
| `created_at` | timestamp | Response only | Preserved across updates. |
| `updated_at` | timestamp | Response only | Changes on accepted updates. |

String request fields are trimmed by the server before validation and storage.

## Fault Status Values

The save endpoint accepts these input values:

| Mobile input | Stored/returned value |
|--------------|-----------------------|
| `X` | `x` |
| `x` | `x` |
| `/` | `slash` |
| `slash` | `slash` |
| `-` | `dash` |
| `dash` | `dash` |

Any other value returns:

```json
{"message":"invalid fault status"}
```

## List Faults

Request:

`GET /api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults?guide_manual=pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json`

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

An accessible vehicle with no faults returns `faults: []` and `count: 0`.

Faults are filtered to the requested `guide_manual` and ordered by `section_id` ascending, then `item_index` ascending.

## Save Fault

Request:

`PUT /api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults`

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

Mobile should use the response body as the canonical saved fault, because `status`, timestamps, and trimmed text may differ from the submitted request.

## Delete Fault

Request:

`DELETE /api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults`

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

After the user has access to the parent vehicle, deleting a missing fault is still a success.

## Error Handling

| Condition | HTTP status | Response body |
|-----------|-------------|---------------|
| Missing authorization header | `401` | `{"message":"No Authorization header found"}` |
| Malformed authorization header | `401` | `{"message":"Invalid Authorization header"}` |
| Invalid Firebase token | `401` | `{"message":"Invalid token"}` |
| Firebase token missing email | `401` | `{"message":"Email not found in token"}` |
| Authenticated user missing from handler context | `401` | `{"message":"unauthorized"}` |
| Invalid JSON body | `400` | `{"message":"invalid request body"}` |
| Blank `equipment_id` | `400` | `{"message":"invalid id"}` |
| Missing or invalid `guide_manual` | `400` | `{"message":"invalid guide manual"}` |
| Blank required field or negative `item_index` | `400` | `{"message":"invalid request"}` |
| Invalid `status` | `400` | `{"message":"invalid fault status"}` |
| Missing or unauthorized vehicle | `404` | `{"message":"pmcs sbs equipment not found"}` |
| Unexpected server error | `500` | `{"status":500,"data":null,"message":"internal Server Error"}` |

## Removed Endpoints

Mobile should remove usage of these PMCS SBS server endpoints:

| Removed behavior | Replacement |
|------------------|-------------|
| `GET /pmcs-sbs/equipment` | Use Shops equipment APIs. |
| `GET /pmcs-sbs/equipment/:equipment_id` | Use Shops equipment APIs plus `GET /pmcs-sbs/equipment/:equipment_id/faults?guide_manual=...`. |
| `PUT /pmcs-sbs/equipment/:equipment_id` | Use Shops equipment APIs for equipment changes. |
| `DELETE /pmcs-sbs/equipment/:equipment_id` | Use Shops equipment APIs for equipment deletion. |
| `PUT /pmcs-sbs/equipment/:equipment_id/completions` | Keep completed-step state client-side. |
| `PATCH /pmcs-sbs/equipment/:equipment_id/completions/batch` | Keep completed-step state client-side. |
| `DELETE /pmcs-sbs/equipment/:equipment_id/completions` | Keep completed-step state client-side. |
| `POST /pmcs-sbs/sync` | Use direct fault list/save/delete calls only. |

## Mobile Implementation Checklist

- Use `shop_vehicle.id` when constructing PMCS SBS fault URLs.
- Use the loaded PMCS SBS guide blob path as `guide_manual`.
- Fetch/list Shops equipment before opening or syncing PMCS SBS faults.
- Remove server-backed PMCS SBS equipment models if they only represented old PMCS-owned equipment.
- Remove PMCS SBS completion API models and calls.
- Remove PMCS SBS sync request/response models and calls.
- Keep local guide progress independent from server fault persistence, but include `guide_manual` with server fault persistence.
- Store returned fault `status`, `created_at`, and `updated_at` from the server response.
- Treat `404 pmcs sbs equipment not found` as non-revealing: do not tell the user whether the vehicle exists elsewhere.
- Refresh or clear local fault state when the selected Shops vehicle or guide/manual changes.
- Expect a vehicle delete in Shops to remove associated server faults through cascade behavior.
