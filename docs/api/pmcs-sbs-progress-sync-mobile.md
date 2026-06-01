# PMCS SBS Progress Sync Mobile API Contract

## API Summary

Base URL: `https://<host>/api/v1/auth`

Authentication: Firebase ID token in `Authorization: Bearer <token>`.

Content type: request bodies are JSON and should use `Content-Type: application/json`.

This API stores authenticated user-owned PMCS Step-by-Step progress. It is separate from the public PMCS SBS library API that serves document JSON from Azure Blob Storage. The library API provides PMCS SBS content; this progress API stores the user's equipment selections, completed steps, and annotated faults for a selected PMCS SBS document.

All records are scoped to the Firebase user id resolved by the authentication middleware. A user cannot read, update, or delete another user's PMCS SBS progress. Attempts to access another user's equipment return the same not-found response as missing equipment.

## Resource Model

### Equipment

An equipment row is the sync anchor for one user-owned PMCS SBS session.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string UUID | Yes | Client-generated equipment id. Sent as `:equipment_id` for single-equipment endpoints and as `id` in batch equipment objects. |
| `equipment_manual` | string | Yes | PMCS SBS JSON blob path. Must be a clean path starting with `pmcs_sbs/` and ending with `.json`. |
| `admin` | string | Yes | Administrative number or user-facing equipment label. Blank values are rejected. |
| `serial` | string | No | Serial number. Blank is accepted. |
| `uic` | string | No | Unit identification code. Blank is accepted. |
| `created_at` | RFC 3339 timestamp | Response only | Server timestamp from initial insert. |
| `updated_at` | RFC 3339 timestamp | Response only | Server timestamp from latest accepted write. |

### Completion

A completion row represents one completed PMCS step. There are no stored incomplete rows. Removing a completed step deletes the row.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `equipment_id` | string UUID | Response and batch requests | Parent equipment id. In single-equipment endpoints this comes from `:equipment_id`. |
| `section_id` | string | Yes | PMCS section id. Blank values are rejected. |
| `item_index` | integer | Yes | Zero-based item index in the section. Must be `0` or greater. |
| `item_no` | string | Save only | Display item number from the source PMCS item. Required for save requests. |
| `step_id` | string | Yes | Stable PMCS step id. Blank values are rejected. |
| `is_complete` | boolean | Response only | Always `true` for rows returned by this API. |
| `updated_at` | RFC 3339 timestamp | Response only | Server timestamp from latest accepted write. |

### Fault

A fault row represents the user's annotated status for one PMCS item. Faults are keyed by equipment, section, and item index.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `equipment_id` | string UUID | Response and batch requests | Parent equipment id. In single-equipment endpoints this comes from `:equipment_id`. |
| `section_id` | string | Yes | PMCS section id. Blank values are rejected. |
| `item_index` | integer | Yes | Zero-based item index in the section. Must be `0` or greater. |
| `item_no` | string | Save only | Display item number from the source PMCS item. Required for save requests. |
| `status` | string | Save only | Accepted input values are `X`, `x`, `/`, `slash`, `-`, and `dash`. Responses use normalized values: `x`, `slash`, or `dash`. |
| `fault_text` | string | Save only | User-entered deficiency text. Blank values are rejected. |
| `corrective_action` | string | No | User-entered corrective action. Blank is accepted. |
| `created_at` | RFC 3339 timestamp | Response only | Server timestamp from initial insert. Preserved when a fault is updated. |
| `updated_at` | RFC 3339 timestamp | Response only | Server timestamp from latest accepted write. |

## Shared Response Shapes

Most successful non-delete responses use this envelope:

```json
{
  "status": 200,
  "data": {},
  "message": ""
}
```

Delete endpoints return a plain message object:

```json
{
  "message": "Equipment deleted"
}
```

Common error shapes:

| Condition | HTTP status | Response body |
|-----------|-------------|---------------|
| Missing authorization header | `401` | `{"message": "No Authorization header found"}` |
| Malformed authorization header | `401` | `{"message": "Invalid Authorization header"}` |
| Invalid Firebase token | `401` | `{"message": "Invalid token"}` |
| Authenticated user missing from handler context | `401` | `{"message": "unauthorized"}` |
| Invalid JSON body | `400` | `{"message": "invalid request body"}` |
| Invalid UUID path or request id | `400` | `{"message": "invalid id"}` |
| Invalid required fields | `400` | `{"message": "invalid request"}` |
| Invalid `equipment_manual` path | `400` | `{"message": "invalid equipment manual blob path"}` |
| Invalid fault status | `400` | `{"message": "invalid fault status"}` |
| Contradictory batch sync request | `400` | `{"message": "invalid sync request"}` |
| Missing or unauthorized equipment | `404` | `{"message": "pmcs sbs equipment not found"}` |
| Unexpected server error | `500` | `{"status": 500, "data": null, "message": "internal Server Error"}` |

## Endpoint Index

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/pmcs-sbs/equipment` | List all PMCS SBS equipment saved by the authenticated user. |
| `GET` | `/pmcs-sbs/equipment/:equipment_id` | Read one equipment row with all saved completions and faults. |
| `PUT` | `/pmcs-sbs/equipment/:equipment_id` | Create or update one equipment row. |
| `DELETE` | `/pmcs-sbs/equipment/:equipment_id` | Delete one equipment row and all child progress rows. |
| `PUT` | `/pmcs-sbs/equipment/:equipment_id/completions` | Mark one PMCS step complete. |
| `PATCH` | `/pmcs-sbs/equipment/:equipment_id/completions/batch` | Apply up to 100 completed-step changes for one equipment row. |
| `DELETE` | `/pmcs-sbs/equipment/:equipment_id/completions` | Remove one completed-step row. |
| `PUT` | `/pmcs-sbs/equipment/:equipment_id/faults` | Create or update one PMCS item fault. |
| `DELETE` | `/pmcs-sbs/equipment/:equipment_id/faults` | Delete one PMCS item fault. |
| `POST` | `/pmcs-sbs/sync` | Apply a transaction-backed batch of explicit progress changes. |

## Endpoints

### List Equipment

`GET /pmcs-sbs/equipment`

Returns all PMCS SBS equipment rows owned by the authenticated user, ordered by `updated_at` descending.

Request body: none.

Success response: `200 OK`

```json
{
  "status": 200,
  "data": {
    "equipment": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "equipment_manual": "pmcs_sbs/hmmwv/basic.json",
        "admin": "A12",
        "serial": "SER123",
        "uic": "WABC01",
        "created_at": "2026-05-28T18:44:12.123456Z",
        "updated_at": "2026-05-28T19:12:03.654321Z"
      }
    ],
    "count": 1
  },
  "message": ""
}
```

An empty result is still a success:

```json
{
  "status": 200,
  "data": {
    "equipment": [],
    "count": 0
  },
  "message": ""
}
```

### Get Equipment Progress

`GET /pmcs-sbs/equipment/:equipment_id`

Returns one equipment row with all saved completion and fault rows. Completion rows are ordered by section id, item index, and step id. Fault rows are ordered by section id and item index.

Path parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `equipment_id` | string UUID | Yes | Equipment id owned by the authenticated user. |

Request body: none.

Success response: `200 OK`

```json
{
  "status": 200,
  "data": {
    "equipment": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "equipment_manual": "pmcs_sbs/hmmwv/basic.json",
      "admin": "A12",
      "serial": "SER123",
      "uic": "WABC01",
      "created_at": "2026-05-28T18:44:12.123456Z",
      "updated_at": "2026-05-28T19:12:03.654321Z"
    },
    "completions": [
      {
        "equipment_id": "550e8400-e29b-41d4-a716-446655440000",
        "section_id": "before",
        "item_index": 0,
        "item_no": "1",
        "step_id": "1-a",
        "is_complete": true,
        "updated_at": "2026-05-28T19:15:01.111111Z"
      }
    ],
    "faults": [
      {
        "equipment_id": "550e8400-e29b-41d4-a716-446655440000",
        "section_id": "before",
        "item_index": 0,
        "item_no": "1",
        "status": "x",
        "fault_text": "Oil leak observed",
        "corrective_action": "",
        "created_at": "2026-05-28T19:16:02.222222Z",
        "updated_at": "2026-05-28T19:16:02.222222Z"
      }
    ]
  },
  "message": ""
}
```

### Save Equipment

`PUT /pmcs-sbs/equipment/:equipment_id`

Creates or updates the authenticated user's equipment row for the supplied id. The id is client-generated and must be a UUID. Writes are last-write-wins by server processing time.

Path parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `equipment_id` | string UUID | Yes | Client-generated equipment id. |

Request body:

```json
{
  "equipment_manual": "pmcs_sbs/hmmwv/basic.json",
  "admin": "A12",
  "serial": "SER123",
  "uic": "WABC01"
}
```

Success response: `200 OK`

```json
{
  "status": 200,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "equipment_manual": "pmcs_sbs/hmmwv/basic.json",
    "admin": "A12",
    "serial": "SER123",
    "uic": "WABC01",
    "created_at": "2026-05-28T18:44:12.123456Z",
    "updated_at": "2026-05-28T19:12:03.654321Z"
  },
  "message": "Equipment saved"
}
```

### Delete Equipment

`DELETE /pmcs-sbs/equipment/:equipment_id`

Deletes one equipment row owned by the authenticated user. Completion and fault rows for the same equipment id are deleted in the same transaction.

Path parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `equipment_id` | string UUID | Yes | Equipment id owned by the authenticated user. |

Request body: none.

Success response: `200 OK`

```json
{
  "message": "Equipment deleted"
}
```

### Save Completion

`PUT /pmcs-sbs/equipment/:equipment_id/completions`

Creates or refreshes one completed-step row for an existing equipment row owned by the authenticated user. The stored value of `is_complete` is always `true`.

Path parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `equipment_id` | string UUID | Yes | Equipment id owned by the authenticated user. |

Request body:

```json
{
  "section_id": "before",
  "item_index": 0,
  "item_no": "1",
  "step_id": "1-a"
}
```

Success response: `200 OK`

```json
{
  "status": 200,
  "data": {
    "equipment_id": "550e8400-e29b-41d4-a716-446655440000",
    "section_id": "before",
    "item_index": 0,
    "item_no": "1",
    "step_id": "1-a",
    "is_complete": true,
    "updated_at": "2026-05-28T19:15:01.111111Z"
  },
  "message": "Completion saved"
}
```

### Batch Completions

`PATCH /pmcs-sbs/equipment/:equipment_id/completions/batch`

Applies multiple completed-step changes for one existing equipment row owned by the authenticated user. This endpoint is intended for screen-level flushes where the app collects step toggles locally and sends them when the user navigates away. The request is transaction-backed; if any change is invalid, unauthorized, or references missing equipment, no changes from the request are committed.

The request may include upserts, deletes, or both. Empty arrays and omitted arrays are accepted as a no-op. A request may contain at most 100 total changes across `upsert_completions` and `delete_completions`.

Path parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `equipment_id` | string UUID | Yes | Equipment id owned by the authenticated user. |

Request body:

```json
{
  "upsert_completions": [
    {
      "section_id": "during",
      "item_index": 0,
      "item_no": "1",
      "step_id": "c"
    }
  ],
  "delete_completions": [
    {
      "section_id": "during",
      "item_index": 0,
      "step_id": "d"
    }
  ]
}
```

Success response: `200 OK`

```json
{
  "status": 200,
  "data": {
    "upserted_count": 1,
    "deleted_count": 1
  },
  "message": "Completions synced"
}
```

No-op request:

```json
{}
```

### Delete Completion

`DELETE /pmcs-sbs/equipment/:equipment_id/completions`

Deletes one completed-step row for an existing equipment row owned by the authenticated user. Missing completion rows are treated as a successful delete when the parent equipment exists.

Path parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `equipment_id` | string UUID | Yes | Equipment id owned by the authenticated user. |

Request body:

```json
{
  "section_id": "before",
  "item_index": 0,
  "step_id": "1-a"
}
```

Success response: `200 OK`

```json
{
  "message": "Completion deleted"
}
```

### Save Fault

`PUT /pmcs-sbs/equipment/:equipment_id/faults`

Creates or updates one fault row for an existing equipment row owned by the authenticated user. Faults are unique by equipment id, section id, and item index.

Path parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `equipment_id` | string UUID | Yes | Equipment id owned by the authenticated user. |

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

Success response: `200 OK`

```json
{
  "status": 200,
  "data": {
    "equipment_id": "550e8400-e29b-41d4-a716-446655440000",
    "section_id": "before",
    "item_index": 0,
    "item_no": "1",
    "status": "x",
    "fault_text": "Oil leak observed",
    "corrective_action": "",
    "created_at": "2026-05-28T19:16:02.222222Z",
    "updated_at": "2026-05-28T19:16:02.222222Z"
  },
  "message": "Fault saved"
}
```

### Delete Fault

`DELETE /pmcs-sbs/equipment/:equipment_id/faults`

Deletes one fault row for an existing equipment row owned by the authenticated user. Missing fault rows are treated as a successful delete when the parent equipment exists.

Path parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `equipment_id` | string UUID | Yes | Equipment id owned by the authenticated user. |

Request body:

```json
{
  "section_id": "before",
  "item_index": 0
}
```

Success response: `200 OK`

```json
{
  "message": "Fault deleted"
}
```

### Batch Sync

`POST /pmcs-sbs/sync`

Applies an explicit set of equipment, completion, and fault changes in one database transaction. If any change in the request is invalid, unauthorized, or references missing equipment, no changes from that request are committed.

The endpoint returns aggregate state for equipment ids touched by non-equipment-delete operations, plus the ids deleted by the request. Deleted equipment ids are not returned in the `equipment` array.

Request body:

```json
{
  "upsert_equipment": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "equipment_manual": "pmcs_sbs/hmmwv/basic.json",
      "admin": "A12",
      "serial": "SER123",
      "uic": "WABC01"
    }
  ],
  "delete_equipment_ids": [],
  "upsert_completions": [
    {
      "equipment_id": "550e8400-e29b-41d4-a716-446655440000",
      "section_id": "before",
      "item_index": 0,
      "item_no": "1",
      "step_id": "1-a"
    }
  ],
  "delete_completions": [],
  "upsert_faults": [
    {
      "equipment_id": "550e8400-e29b-41d4-a716-446655440000",
      "section_id": "before",
      "item_index": 0,
      "item_no": "1",
      "status": "X",
      "fault_text": "Oil leak observed",
      "corrective_action": ""
    }
  ],
  "delete_faults": []
}
```

Success response: `200 OK`

```json
{
  "status": 200,
  "data": {
    "equipment": [
      {
        "equipment": {
          "id": "550e8400-e29b-41d4-a716-446655440000",
          "equipment_manual": "pmcs_sbs/hmmwv/basic.json",
          "admin": "A12",
          "serial": "SER123",
          "uic": "WABC01",
          "created_at": "2026-05-28T18:44:12.123456Z",
          "updated_at": "2026-05-28T19:20:03.333333Z"
        },
        "completions": [
          {
            "equipment_id": "550e8400-e29b-41d4-a716-446655440000",
            "section_id": "before",
            "item_index": 0,
            "item_no": "1",
            "step_id": "1-a",
            "is_complete": true,
            "updated_at": "2026-05-28T19:20:03.333333Z"
          }
        ],
        "faults": [
          {
            "equipment_id": "550e8400-e29b-41d4-a716-446655440000",
            "section_id": "before",
            "item_index": 0,
            "item_no": "1",
            "status": "x",
            "fault_text": "Oil leak observed",
            "corrective_action": "",
            "created_at": "2026-05-28T19:20:03.333333Z",
            "updated_at": "2026-05-28T19:20:03.333333Z"
          }
        ]
      }
    ],
    "deleted_equipment_ids": []
  },
  "message": "Sync complete"
}
```

An empty batch is valid:

```json
{
  "upsert_equipment": [],
  "delete_equipment_ids": [],
  "upsert_completions": [],
  "delete_completions": [],
  "upsert_faults": [],
  "delete_faults": []
}
```

Empty or omitted arrays both decode as empty change sets.

## Batch Sync Validation

The sync endpoint rejects contradictory changes within the same request.

| Contradiction | Result |
|---------------|--------|
| Same equipment id in `upsert_equipment` and `delete_equipment_ids` | `400 {"message": "invalid sync request"}` |
| Completion upsert references equipment also listed in `delete_equipment_ids` | `400 {"message": "invalid sync request"}` |
| Fault upsert references equipment also listed in `delete_equipment_ids` | `400 {"message": "invalid sync request"}` |
| Same completion key in `upsert_completions` and `delete_completions` | `400 {"message": "invalid sync request"}` |
| Same fault key in `upsert_faults` and `delete_faults` | `400 {"message": "invalid sync request"}` |

Contradiction checks canonicalize UUID casing and trim whitespace. Completion keys use `equipment_id`, trimmed `section_id`, `item_index`, and trimmed `step_id`. Fault keys use `equipment_id`, trimmed `section_id`, and `item_index`.

## Persistence Semantics

- Equipment, completion, and fault writes trim leading and trailing whitespace from string fields before persistence.
- Equipment writes preserve `created_at` on update and refresh `updated_at`.
- Completion writes always store `is_complete: true` and refresh `updated_at`.
- Batch completion requests accept at most 100 total upsert and delete changes.
- Fault writes preserve `created_at` on update and refresh `updated_at`.
- Single completion and fault delete endpoints validate ownership through the parent equipment row.
- Single completion and fault delete endpoints do not require the child row to exist.
- Equipment deletes are hard deletes and remove child completion and fault rows.
- Batch sync runs in a single transaction and returns the post-commit shape that the transaction produced.
- Conflict resolution is last-write-wins by server processing time. The API does not accept client timestamps as conflict-resolution inputs.
