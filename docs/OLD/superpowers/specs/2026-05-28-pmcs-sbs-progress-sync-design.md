# PMCS SBS Progress Sync Design

Date: 2026-05-28
Status: Design approved by user; awaiting implementation plan

## Goal

Add authenticated server-side sync for PMCS Step-by-Step user progress so logged-in users can move between devices and keep equipment, completed steps, and annotated faults in sync.

The existing PMCS SBS library API stays public and blob-backed. This design adds a separate authenticated database-backed API for user-owned progress state.

## Decisions

- Use a hybrid sync model:
  - action endpoints for normal online use
  - one batch change-set endpoint for offline replay
  - aggregate read endpoints for device bootstrap
- Use last-write-wins by server processing time.
- Use hard deletes. The API does not use `deleted_at` for this pass.
- Equipment IDs are client-generated UUIDs.
- `pmcs_sbs_equipment.equipment_manual` stores the PMCS SBS JSON `blob_path`, for example `pmcs_sbs/hmmwv/hmmwv_basic_pmcs.json`.
- Batch sync requests use explicit change sets, not full replacement snapshots.
- Faults are one row per PMCS item, keyed by `(equipment_id, section_id, item_index)`.
- Fault `status` must be one of `X`, `/`, or `-`.
- Completion rows represent only completed steps. An unchecked step is represented by no row.

## Current Context

The current PMCS SBS content endpoints live under `api/library/pmcs_sbs` and are registered on the public `/api/v1` route group:

- `GET /library/pmcs-sbs/folders`
- `GET /library/pmcs-sbs/:folder/files`
- `GET /library/pmcs-sbs/content?blob_path=...`

Those endpoints serve JSON from Azure Blob Storage and do not require authentication.

The new progress API will write to these already-generated Postgres tables:

- `pmcs_sbs_equipment`
- `pmcs_sbs_completions`
- `pmcs_sbs_faults`

`pmcs_sbs_equipment` stores `user_uid`, so it is the ownership anchor. `pmcs_sbs_completions` and `pmcs_sbs_faults` are scoped by `equipment_id`, so every completion and fault operation must first prove that the target equipment belongs to the authenticated Firebase user.

## Package Architecture

Create a new authenticated package, `api/pmcs_sbs_progress`, separate from `api/library/pmcs_sbs`.

Files:

- `route.go`: HTTP handlers, auth context extraction, request binding, and response mapping.
- `service.go`: service interface.
- `service_impl.go`: validation and business rules.
- `repository.go`: repository interface.
- `repository_impl.go`: Jet/Postgres operations.
- `types.go`: request and response DTOs.
- `errors.go`: sentinel errors for predictable HTTP mapping.

Register the package from `api/route/route.go` under the existing authenticated route group:

```go
authRoutes := router.Group("/api/v1/auth")
authRoutes.Use(middleware.AuthenticationMiddleware(authClient))
```

## Endpoint Contract

All routes below are relative to `/api/v1/auth`.

### Equipment

#### `GET /pmcs-sbs/equipment`

Returns all PMCS SBS equipment rows for the logged-in user.

Response data:

```json
{
  "equipment": [
    {
      "id": "uuid",
      "equipment_manual": "pmcs_sbs/hmmwv/hmmwv_basic_pmcs.json",
      "admin": "A12",
      "serial": "SER123",
      "uic": "WABC01",
      "created_at": "2026-05-28T12:00:00Z",
      "updated_at": "2026-05-28T12:00:00Z"
    }
  ],
  "count": 1
}
```

#### `GET /pmcs-sbs/equipment/:equipment_id`

Returns one equipment row plus all completions and faults for that equipment. This is the primary device-switch/bootstrap endpoint.

Response data:

```json
{
  "equipment": {
    "id": "uuid",
    "equipment_manual": "pmcs_sbs/hmmwv/hmmwv_basic_pmcs.json",
    "admin": "A12",
    "serial": "SER123",
    "uic": "WABC01",
    "created_at": "2026-05-28T12:00:00Z",
    "updated_at": "2026-05-28T12:00:00Z"
  },
  "completions": [
    {
      "equipment_id": "uuid",
      "section_id": "before",
      "item_index": 0,
      "item_no": "1",
      "step_id": "1-a",
      "is_complete": true,
      "updated_at": "2026-05-28T12:05:00Z"
    }
  ],
  "faults": [
    {
      "equipment_id": "uuid",
      "section_id": "before",
      "item_index": 0,
      "item_no": "1",
      "status": "X",
      "fault_text": "Oil leak observed",
      "corrective_action": "",
      "created_at": "2026-05-28T12:06:00Z",
      "updated_at": "2026-05-28T12:06:00Z"
    }
  ]
}
```

#### `PUT /pmcs-sbs/equipment/:equipment_id`

Creates or updates one equipment row. The server ignores any `user_uid` in the body and uses the authenticated Firebase UID.

Request body:

```json
{
  "equipment_manual": "pmcs_sbs/hmmwv/hmmwv_basic_pmcs.json",
  "admin": "A12",
  "serial": "SER123",
  "uic": "WABC01"
}
```

Response: `200 OK` with the saved equipment row.

#### `DELETE /pmcs-sbs/equipment/:equipment_id`

Hard deletes the equipment row for the logged-in user and removes child completions and faults for that equipment.

Response: `200 OK`.

### Completions

#### `PUT /pmcs-sbs/equipment/:equipment_id/completions`

Upserts one completed step. The server stores `is_complete = true`.

Request body:

```json
{
  "section_id": "before",
  "item_index": 0,
  "item_no": "1",
  "step_id": "1-a"
}
```

Response: `200 OK` with the saved completion row.

#### `DELETE /pmcs-sbs/equipment/:equipment_id/completions`

Deletes one completion row. This is how the client represents an unchecked step.

Request body:

```json
{
  "section_id": "before",
  "item_index": 0,
  "step_id": "1-a"
}
```

Response: `200 OK`.

### Faults

#### `PUT /pmcs-sbs/equipment/:equipment_id/faults`

Upserts one fault for `(equipment_id, section_id, item_index)`.

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

Response: `200 OK` with the saved fault row.

#### `DELETE /pmcs-sbs/equipment/:equipment_id/faults`

Deletes one fault by `(equipment_id, section_id, item_index)`.

Request body:

```json
{
  "section_id": "before",
  "item_index": 0
}
```

Response: `200 OK`.

### Batch Sync

#### `POST /pmcs-sbs/sync`

Applies an explicit offline replay change set in one database transaction.

Request body:

```json
{
  "upsert_equipment": [
    {
      "id": "uuid",
      "equipment_manual": "pmcs_sbs/hmmwv/hmmwv_basic_pmcs.json",
      "admin": "A12",
      "serial": "SER123",
      "uic": "WABC01"
    }
  ],
  "delete_equipment_ids": ["uuid"],
  "upsert_completions": [
    {
      "equipment_id": "uuid",
      "section_id": "before",
      "item_index": 0,
      "item_no": "1",
      "step_id": "1-a"
    }
  ],
  "delete_completions": [
    {
      "equipment_id": "uuid",
      "section_id": "before",
      "item_index": 0,
      "step_id": "1-a"
    }
  ],
  "upsert_faults": [
    {
      "equipment_id": "uuid",
      "section_id": "before",
      "item_index": 0,
      "item_no": "1",
      "status": "X",
      "fault_text": "Oil leak observed",
      "corrective_action": ""
    }
  ],
  "delete_faults": [
    {
      "equipment_id": "uuid",
      "section_id": "before",
      "item_index": 0
    }
  ]
}
```

Response data contains authoritative state for every touched equipment ID that still exists after sync. Equipment IDs deleted by the request are echoed separately.

```json
{
  "equipment": [
    {
      "equipment": {
        "id": "uuid",
        "equipment_manual": "pmcs_sbs/hmmwv/hmmwv_basic_pmcs.json",
        "admin": "A12",
        "serial": "SER123",
        "uic": "WABC01",
        "created_at": "2026-05-28T12:00:00Z",
        "updated_at": "2026-05-28T12:00:00Z"
      },
      "completions": [],
      "faults": []
    }
  ],
  "deleted_equipment_ids": ["uuid"]
}
```

## Sync Rules

Normal online flow:

1. App creates equipment locally with a UUID.
2. App calls `PUT /pmcs-sbs/equipment/:equipment_id`.
3. Each checked step calls completion `PUT`.
4. Each unchecked step calls completion `DELETE`.
5. Each fault save calls fault `PUT`.
6. Each fault removal calls fault `DELETE`.
7. Another device calls `GET /pmcs-sbs/equipment` and `GET /pmcs-sbs/equipment/:equipment_id` to rebuild state.

Offline flow:

1. App queues explicit operations while offline.
2. App sends queued operations to `POST /pmcs-sbs/sync`.
3. Server validates the whole request.
4. Server applies changes in one transaction.
5. Server returns authoritative rows with server timestamps.

Conflict handling:

- Last write wins by server processing time.
- Server assigns `created_at` and `updated_at`.
- Client timestamps do not decide conflicts.
- A sync request that deletes an equipment ID and also upserts completions or faults for the same equipment ID is rejected with `400`.
- A sync request that upserts and deletes the same completion or fault key is rejected with `400`.
- Child rows for an equipment ID may only be changed if that equipment belongs to the current user, or if the same sync request creates that equipment for the current user.

## Validation

Equipment:

- `equipment_id` must be a valid UUID.
- `equipment_manual` is required, must start with `pmcs_sbs/`, must end with `.json`, and must still pass the prefix check after path cleaning.
- `admin` is required and trimmed.
- `serial` is optional and trimmed.
- `uic` is optional and trimmed.
- Request `user_uid`, if present, is ignored.

Completions:

- `section_id` is required and trimmed.
- `item_index` must be `>= 0`.
- `item_no` is required and trimmed.
- `step_id` is required and trimmed.
- The server stores `is_complete = true`.

Faults:

- `section_id` is required and trimmed.
- `item_index` must be `>= 0`.
- `item_no` is required and trimmed.
- `status` must be exactly `X`, `/`, or `-`.
- `fault_text` is required and trimmed.
- `corrective_action` is optional and trimmed.

## Error Handling

- `401 Unauthorized`: auth context is missing or invalid.
- `400 Bad Request`: invalid UUID, invalid JSON, invalid blob path, invalid status, empty required field, or contradictory sync request.
- `404 Not Found`: equipment does not exist for the current user.
- `500 Internal Server Error`: unexpected database or server failure.

Internal database details are logged with `slog` and are not returned to clients.

## Repository Behavior

Use Jet for Postgres operations, consistent with most user-owned data modules in the repo.

Required repository capabilities:

- List equipment by `user_uid`.
- Get one equipment aggregate by `user_uid` and `equipment_id`.
- Upsert equipment using client UUID and authenticated `user_uid`.
- Hard delete equipment and child rows.
- Upsert one completion after ownership validation.
- Delete one completion after ownership validation.
- Upsert one fault after ownership validation.
- Delete one fault after ownership validation.
- Apply batch sync in a transaction.

The batch sync transaction should validate ownership before mutating child rows. If a sync request creates an equipment row and child rows for that same ID, child validation can treat that equipment ID as owned by the current request.

## Testing

Handler tests:

- Auth missing returns `401`.
- Invalid JSON returns `400`.
- Invalid UUID path returns `400`.
- Service errors map to `400`, `404`, and `500`.
- Route registration uses `/api/v1/auth`.

Service tests:

- Equipment validation trims fields and rejects invalid `equipment_manual`.
- Completion validation rejects empty identifiers and negative item indexes.
- Fault validation rejects statuses outside `X`, `/`, and `-`.
- Contradictory sync requests return validation errors.
- Child writes require owned equipment.

Repository or integration tests:

- Equipment upsert/list/get/delete.
- Equipment delete removes child completions and faults.
- Completion upsert/delete.
- Fault upsert/delete.
- User isolation: user A cannot read, update, or delete user B equipment or child rows.
- Batch sync commits valid change sets.
- Batch sync rolls back all writes on any invalid child operation.

## Out Of Scope

- Migrating or changing the existing table schema.
- Soft-delete sync tombstones.
- Client-clock conflict resolution.
- `409 Conflict` stale-write protection.
- Multiple fault rows per PMCS item.
- Changing the public PMCS SBS content API.
