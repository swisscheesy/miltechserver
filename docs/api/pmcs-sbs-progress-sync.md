# PMCS SBS Faults API

Base URL: `/api/v1/auth`

The server no longer stores PMCS SBS guide progress, completions, or PMCS-owned equipment. PMCS SBS tracking remains client-side. The authenticated server API stores only PMCS SBS faults for existing `shop_vehicle` equipment.

The public PMCS SBS library API remains separate and continues to serve guide JSON from Azure Blob Storage.

## Authorization

All endpoints require Firebase authentication. The authenticated user must be a member of the shop that owns the target `shop_vehicle`.

Missing vehicles and vehicles in shops the user cannot access both return:

```json
{"message":"pmcs sbs equipment not found"}
```

## List Faults

`GET /pmcs-sbs/equipment/:equipment_id/faults`

Returns all PMCS SBS faults for the shop vehicle.

## Save Fault

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

Accepted status inputs are `X`, `x`, `/`, `slash`, `-`, and `dash`. Responses normalize to `x`, `slash`, or `dash`.

## Delete Fault

`DELETE /pmcs-sbs/equipment/:equipment_id/faults`

```json
{
  "section_id": "before",
  "item_index": 0
}
```

Deletes are idempotent after the user has access to the parent vehicle.
