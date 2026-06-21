# PMCS SBS Faults-Only Server Refactor Design

## Context

The PMCS SBS client has changed before this server feature reached production. Server-side guide progress tracking is no longer needed. The PMCS SBS server should no longer store guide progress, completions, or PMCS-owned equipment records.

Equipment now lives exclusively in `shop_vehicle`. The `pmcs_sbs_equipment` and `pmcs_sbs_completions` tables have been removed. The remaining server persistence is PMCS SBS faults in `pmcs_sbs_faults`.

The live database already has:

- `pmcs_sbs_faults.equipment_id` as `text`;
- `pmcs_sbs_faults.equipment_id` as a foreign key to `shop_vehicle(id)`;
- `ON UPDATE CASCADE ON DELETE CASCADE` on that foreign key.

There is no production migration or old UID compatibility requirement because this feature has not shipped.

## Goals

- Keep the public PMCS SBS library API unchanged.
- Keep Shops behavior unchanged.
- Keep the existing PMCS SBS nested route style for faults:
  - `GET /api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults`
  - `PUT /api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults`
  - `DELETE /api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults`
- Treat `:equipment_id` as `shop_vehicle.id`.
- Allow any member of the vehicle's shop to list, create, update, and delete PMCS SBS faults.
- Remove stale PMCS SBS equipment, completion, and sync code paths.

## Non-Goals

- No PMCS SBS equipment create, update, delete, or list behavior.
- No server-side PMCS guide progress tracking.
- No `pmcs_sbs_completions` support.
- No `POST /pmcs-sbs/sync` support.
- No Shops package behavior changes.
- No data migration path for old PMCS SBS equipment IDs.

## Architecture

The authenticated PMCS SBS server package should become a faults-only API. The current `api/pmcs_sbs_progress` package can be refactored in place to reduce churn, even though its name is now broader than its final behavior. A package rename can be deferred unless a later cleanup explicitly wants that churn.

The package should depend only on:

- Gin handler wiring;
- the authenticated `bootstrap.User`;
- generated Jet `PmcsSbsFaults`, `ShopVehicle`, and `ShopMembers` tables/models;
- the shared response envelope already used by the current handlers.

The PMCS SBS faults code should not call Shops mutation services and should not mutate `shop_vehicle`. Shops remains the owner of vehicle create, update, delete, and notification behavior.

## Authorization

Every fault operation must prove vehicle access before returning or mutating fault rows.

The authorization check should prove:

```text
shop_vehicle.id = :equipment_id
shop_members.shop_id = shop_vehicle.shop_id
shop_members.user_id = authenticated Firebase user id
```

Any matching shop member can list, create, update, and delete PMCS SBS faults for that vehicle.

Missing vehicles and unauthorized vehicles should return the same not-found response so the endpoint does not reveal whether a vehicle exists in another shop.

Recommended response:

```json
{"message":"pmcs sbs equipment not found"}
```

## API Contract

### List Faults

```text
GET /api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults
```

Returns all PMCS SBS faults for an accessible `shop_vehicle` row.

Fault rows should be ordered by:

```text
section_id ASC, item_index ASC
```

An accessible vehicle with no faults returns an empty list.

### Save Fault

```text
PUT /api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults
```

Creates or updates one fault keyed by:

```text
(equipment_id, section_id, item_index)
```

Request body:

```json
{
  "section_id": "before",
  "item_index": 0,
  "item_no": "1",
  "status": "X",
  "fault_text": "Oil leak observed",
  "corrective_action": ""
}
```

Status normalization remains:

- `X` and `x` become `x`;
- `/` and `slash` become `slash`;
- `-` and `dash` become `dash`.

On insert, set `created_at` and `updated_at`. On update, preserve `created_at` and refresh `updated_at`.

### Delete Fault

```text
DELETE /api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults
```

Deletes one fault by `section_id` and `item_index`.

Request body:

```json
{
  "section_id": "before",
  "item_index": 0
}
```

After parent vehicle access is proven, deleting a missing fault should be idempotent and return success.

## Validation And Errors

- Missing auth returns `401`.
- Invalid JSON returns `400 {"message":"invalid request body"}`.
- Blank `equipment_id`, `section_id`, `item_no`, or `fault_text` returns `400`.
- Negative `item_index` returns `400`.
- Invalid `status` returns `400`.
- Missing or unauthorized vehicle returns `404`.
- Unexpected database failures return the existing standard `500` response.

The server should trim leading and trailing whitespace from string inputs before validation and persistence.

## Stale Surface Removal

Remove live code and tests for:

- PMCS SBS equipment list/get/upsert/delete;
- PMCS SBS completion save/batch/delete;
- PMCS SBS sync;
- `equipment_manual` validation and request/response fields;
- `nomenclature`, `model`, and `uic` PMCS equipment metadata handling;
- `PmcsSbsEquipment`, `PmcsSbsCompletions`, `pmcs_sbs_equipment`, and `pmcs_sbs_completions` references.

Docs should be updated so mobile integration guidance describes the faults-only API and no longer describes server-side PMCS guide progress.

## Database Notes

No migration behavior is needed for old PMCS SBS data because the feature has not shipped.

The final schema expectation for PMCS SBS faults is:

- `pmcs_sbs_faults.equipment_id text NOT NULL`;
- primary key on `(equipment_id, section_id, item_index)`;
- foreign key from `pmcs_sbs_faults(equipment_id)` to `shop_vehicle(id)`;
- `ON UPDATE CASCADE ON DELETE CASCADE`;
- indexes sufficient for the list query, covered by the primary key for `equipment_id`, `section_id`, and `item_index`.

## Testing Plan

Update route tests for:

- auth required;
- invalid JSON;
- list faults success;
- save fault success;
- delete fault success;
- service error mapping.

Update service tests for:

- fault validation;
- whitespace trimming;
- status normalization;
- invalid status;
- negative item index;
- blank required fields.

Update repository/integration tests for:

- shop member can list, save, and delete PMCS SBS faults;
- non-member cannot list, save, or delete faults;
- missing vehicle returns not found;
- upsert preserves `created_at` and updates mutable fields;
- delete is idempotent for an accessible vehicle;
- deleting a `shop_vehicle` cascades PMCS SBS faults through the database foreign key.

Add a stale-reference guard in verification:

```text
rg "PmcsSbsEquipment|PmcsSbsCompletions|pmcs_sbs_equipment|pmcs_sbs_completions" api/pmcs_sbs_progress tests/pmcs_sbs_progress docs/api
```

Expected result after implementation: no live PMCS SBS server/API references, except historical design/plan docs if those are intentionally left as archive context.

## Verification

Focused verification should include:

```text
go test ./api/pmcs_sbs_progress
go test ./tests/pmcs_sbs_progress
go test ./api/route
```

If package names or test locations change during implementation, use the equivalent focused PMCS SBS fault suites.

Run `go test ./...` only as a broader health check after the focused tests pass, and report unrelated baseline failures separately if they exist.
