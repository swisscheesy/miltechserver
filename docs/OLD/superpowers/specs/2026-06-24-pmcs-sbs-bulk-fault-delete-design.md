# PMCS SBS Bulk Fault Delete Design

## Context

PMCS SBS server persistence is faults-only. Fault rows live in `pmcs_sbs_faults` and are attached to Shops equipment by `shop_vehicle.id`, which is still passed as `:equipment_id` in the PMCS SBS fault routes.

The current API lets mobile delete one fault per request:

- `DELETE /api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults`

That route is inefficient when a user clears several faults from the same loaded guide/manual. The new endpoint should delete multiple fault identities in one request without changing the existing single-delete contract.

## Current Contract To Preserve

- Fault identity is `(equipment_id, guide_manual, section_id, item_index)`.
- `equipment_id` is the selected `shop_vehicle.id`.
- `guide_manual` is required for fault list, save, and delete.
- A user can manage faults when they are a member of the shop that owns the vehicle.
- Missing and unauthorized vehicles return the same `404 {"message":"pmcs sbs equipment not found"}`.
- Delete is idempotent after vehicle access passes. Deleting a missing fault returns success.
- The API does not validate that `section_id` and `item_index` exist in the guide JSON.

## Endpoint

Add a new route under the existing authenticated PMCS SBS fault route family:

`DELETE /api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults/bulk`

The existing single-delete route remains unchanged.

## Request

The request is scoped to one guide/manual.

```json
{
  "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
  "faults": [
    { "section_id": "before", "item_index": 0 },
    { "section_id": "before", "item_index": 1 }
  ]
}
```

Validation rules:

- `guide_manual` is required and uses the existing PMCS SBS guide path validation.
- `faults` is required.
- `faults` must contain at least `1` and at most `100` entries.
- Each entry requires nonblank `section_id`.
- Each entry requires `item_index >= 0`.
- Duplicate `(section_id, item_index)` entries are rejected with `ErrInvalidRequest` so `requested_count` stays honest.

## Response

Success response:

```json
{
  "message": "Faults deleted",
  "requested_count": 2,
  "deleted_count": 1
}
```

`requested_count` is the number of validated fault keys in the request.

`deleted_count` is the database affected-row count. Missing fault keys are not errors; they simply do not increment `deleted_count`.

## Architecture

Implement the endpoint in the existing `api/pmcs_sbs_progress` package to follow the current PMCS SBS fault architecture.

Files to update:

- `types.go`: add bulk delete request and response structs.
- `route.go`: register the bulk route, bind JSON once with `ShouldBindJSON`, call the service, and map errors through the existing PMCS SBS error responder.
- `service.go`: add `DeleteFaults` to the service interface.
- `service_impl.go`: validate authenticated user, `equipment_id`, one `guide_manual`, and all fault keys. Reject empty, oversized, invalid, and duplicate key lists.
- `repository.go`: add a bulk delete method that accepts the validated vehicle/manual scope and a `[]FaultKey` key list.
- `repository_impl.go`: keep authorization separate from mutation, then execute one bulk delete.
- `api/route/route_test.go`: assert the new route is registered under auth.

No generated Jet files should be edited by hand.

## Repository Data Flow

The repository should:

1. Call `requireVehicleAccess(user, equipmentID)` once.
2. Execute a single delete for the validated vehicle/manual/key set.
3. Return the affected row count to the service.

Target SQL shape:

```sql
DELETE FROM pmcs_sbs_faults
WHERE equipment_id = $1
  AND guide_manual = $2
  AND ROW(section_id, item_index) IN (...)
```

Use Jet for the delete predicate. Jet supports `ROW(...).IN(...)`, so the implementation can remain inside the current type-safe query style.

## Database And Performance

No schema change is needed.

The existing primary key supports the bulk delete access pattern:

`(equipment_id, guide_manual, section_id, item_index)`

Do not add a new index for this feature. The endpoint deletes at most `100` keys for one vehicle and one guide/manual, and the existing primary key already covers the filter columns.

## Error Handling

Match the existing PMCS SBS fault API error style:

| Condition | HTTP status | Response |
|-----------|-------------|----------|
| Invalid JSON body | `400` | `{"message":"invalid request body"}` |
| Blank `equipment_id` | `400` | `{"message":"invalid id"}` |
| Missing or invalid `guide_manual` | `400` | `{"message":"invalid guide manual"}` |
| Empty `faults` | `400` | `{"message":"invalid request"}` |
| More than `100` faults | `400` | `{"message":"invalid request"}` |
| Blank `section_id` | `400` | `{"message":"invalid request"}` |
| Negative `item_index` | `400` | `{"message":"invalid request"}` |
| Duplicate fault keys | `400` | `{"message":"invalid request"}` |
| Missing or unauthorized vehicle | `404` | `{"message":"pmcs sbs equipment not found"}` |
| Unexpected server error | `500` | Generic internal error response |

## Testing

Add focused tests that match the existing package style:

- Route registration for `DELETE /api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults/bulk`.
- Handler success response with `requested_count` and `deleted_count`.
- Handler invalid JSON response.
- Service validation for empty `faults`, more than `100` faults, duplicate keys, blank `section_id`, negative `item_index`, invalid `guide_manual`, and blank `equipment_id`.
- Service passes normalized vehicle/manual/key data to the repository.
- Repository deletes multiple faults in one request.
- Repository returns accurate affected-row count when some requested keys are missing.
- Repository preserves faults for other guide manuals.
- Repository preserves faults for other vehicles.
- Repository denies non-members and missing vehicles with `ErrNotFound`.

Focused verification target:

```sh
go test ./api/pmcs_sbs_progress ./tests/pmcs_sbs_progress ./api/route -count=1
```

If the implementation touches broader route setup unexpectedly, include:

```sh
go test ./api/library/pmcs_sbs -count=1
```

## Out Of Scope

- Changing the existing single-delete route.
- Reintroducing PMCS SBS equipment, progress, completion, or sync persistence.
- Validating guide/manual existence in Azure Blob storage.
- Validating whether section and item identities exist in the guide JSON.
- Adding database indexes or schema migrations.
- Changing Shops authorization semantics.
