# PMCS SBS Progress Sync API

Base URL: `https://<host>/api/v1/auth`
Authentication: Firebase bearer token required.
Content-Type: `application/json`

## Overview

This API syncs PMCS SBS equipment, completed steps, and faults for logged-in users. The public PMCS SBS library API still serves document JSON from Azure Blob Storage. This API stores user progress for a selected `equipment_manual` blob path.

## Rules

- Equipment IDs are client-generated UUIDs.
- `equipment_manual` stores the PMCS SBS JSON blob path, for example `pmcs_sbs/hmmwv/basic.json`.
- `equipment_manual` must be a clean path under `pmcs_sbs/` and must end with `.json`.
- `admin` is required when saving equipment.
- Completion rows represent completed steps only.
- `section_id`, `item_no`, and `step_id` are required for completion saves.
- `item_index` must be zero or greater.
- Fault status must be `X`, `/`, or `-`.
- `fault_text` is required when saving a fault.
- Deletes are hard deletes.
- Last write wins by server processing time.
- Batch sync rejects contradictory changes in one request, such as upserting and deleting the same equipment, completion, or fault.

## Endpoints

### List Equipment

`GET /pmcs-sbs/equipment`

Returns all PMCS SBS equipment for the authenticated user.

### Get Equipment Progress

`GET /pmcs-sbs/equipment/:equipment_id`

Returns one equipment row with its completions and faults.

### Save Equipment

`PUT /pmcs-sbs/equipment/:equipment_id`

```json
{
  "equipment_manual": "pmcs_sbs/hmmwv/basic.json",
  "admin": "A12",
  "serial": "SER123",
  "uic": "WABC01"
}
```

### Delete Equipment

`DELETE /pmcs-sbs/equipment/:equipment_id`

Deletes equipment and child completion/fault rows.

### Save Completion

`PUT /pmcs-sbs/equipment/:equipment_id/completions`

```json
{
  "section_id": "before",
  "item_index": 0,
  "item_no": "1",
  "step_id": "1-a"
}
```

### Delete Completion

`DELETE /pmcs-sbs/equipment/:equipment_id/completions`

```json
{
  "section_id": "before",
  "item_index": 0,
  "step_id": "1-a"
}
```

### Save Fault

`PUT /pmcs-sbs/equipment/:equipment_id/faults`

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

### Delete Fault

`DELETE /pmcs-sbs/equipment/:equipment_id/faults`

```json
{
  "section_id": "before",
  "item_index": 0
}
```

### Batch Sync

`POST /pmcs-sbs/sync`

Sends explicit offline replay changes in one request.

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
