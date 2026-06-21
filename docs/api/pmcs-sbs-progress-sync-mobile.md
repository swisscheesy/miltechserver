# PMCS SBS Faults Mobile API Contract

## API Summary

Base URL: `https://<host>/api/v1/auth`

Authentication: Firebase ID token in `Authorization: Bearer <token>`.

Content type: request bodies are JSON and should use `Content-Type: application/json`.

The server stores PMCS SBS faults only. PMCS SBS guide progress and completed-step tracking are client-side only. Equipment is owned by Shops and lives in `shop_vehicle`.

## Resource Model

### Fault

Faults are keyed by `equipment_id`, `section_id`, and `item_index`. The `equipment_id` value is the `shop_vehicle.id`.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `equipment_id` | string | Response only | `shop_vehicle.id`. In requests this comes from `:equipment_id`. |
| `section_id` | string | Yes | PMCS section id. Blank values are rejected. |
| `item_index` | integer | Yes | Zero-based item index in the section. Must be `0` or greater. |
| `item_no` | string | Required for save; returned in responses | Display item number from the source PMCS item. |
| `status` | string | Required for save; returned in responses | Accepted input values are `X`, `x`, `/`, `slash`, `-`, and `dash`. Responses use `x`, `slash`, or `dash`. |
| `fault_text` | string | Required for save; returned in responses | User-entered deficiency text. Blank values are rejected. |
| `corrective_action` | string | No | User-entered corrective action. Blank is accepted. |
| `created_at` | RFC 3339 timestamp | Response only | Server timestamp from initial insert. Preserved on update. |
| `updated_at` | RFC 3339 timestamp | Response only | Server timestamp from latest accepted write. |

## Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/pmcs-sbs/equipment/:equipment_id/faults` | List all PMCS SBS faults for a shop vehicle. |
| `PUT` | `/pmcs-sbs/equipment/:equipment_id/faults` | Create or update one PMCS SBS fault. |
| `DELETE` | `/pmcs-sbs/equipment/:equipment_id/faults` | Delete one PMCS SBS fault. |

## Common Errors

| Condition | HTTP status | Response body |
|-----------|-------------|---------------|
| Missing authorization header | `401` | `{"message":"No Authorization header found"}` |
| Malformed authorization header | `401` | `{"message":"Invalid Authorization header"}` |
| Invalid Firebase token | `401` | `{"message":"Invalid token"}` |
| Firebase token missing email | `401` | `{"message":"Email not found in token"}` |
| Authenticated user missing from handler context | `401` | `{"message":"unauthorized"}` |
| Invalid JSON body | `400` | `{"message":"invalid request body"}` |
| Blank equipment id | `400` | `{"message":"invalid id"}` |
| Invalid required fields | `400` | `{"message":"invalid request"}` |
| Invalid fault status | `400` | `{"message":"invalid fault status"}` |
| Missing or unauthorized vehicle | `404` | `{"message":"pmcs sbs equipment not found"}` |
| Unexpected server error | `500` | `{"status":500,"data":null,"message":"internal Server Error"}` |

## List Faults

`GET /pmcs-sbs/equipment/:equipment_id/faults`

Success response:

```json
{
  "status": 200,
  "data": {
    "faults": [
      {
        "equipment_id": "550e8400-e29b-41d4-a716-446655440000",
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

An accessible vehicle with no faults returns:

```json
{
  "status": 200,
  "data": {
    "faults": [],
    "count": 0
  },
  "message": ""
}
```

## Save Fault

`PUT /pmcs-sbs/equipment/:equipment_id/faults`

Request:

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

Success response:

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
    "created_at": "2026-06-21T18:44:12.123456Z",
    "updated_at": "2026-06-21T19:12:03.654321Z"
  },
  "message": "Fault saved"
}
```

## Delete Fault

`DELETE /pmcs-sbs/equipment/:equipment_id/faults`

Request:

```json
{
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

## Removed Server Behavior

The server no longer exposes PMCS SBS equipment create/update/delete/list endpoints, completion endpoints, or `POST /pmcs-sbs/sync`.
