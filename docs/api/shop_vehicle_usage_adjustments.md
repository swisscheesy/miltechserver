# Shop Vehicle Usage Adjustments API

## Endpoint

`PATCH /api/v1/auth/shops/vehicles/:vehicle_id/usage`

Replace `:vehicle_id` with the Shop vehicle UUID. This endpoint adjusts the
persisted tracked mileage and/or tracked hours atomically; it does not replace
the vehicle's metadata or base `mileage` and `hours` readings.

## Authentication and authorization

The route is authenticated. The authenticated user must be a member of the
Shop that owns the vehicle. The vehicle creator or Shop administrator role is
not required for this usage-only operation.

## Request structure

Send one JSON object with `Content-Type: application/json`:

| Field | Type | Required | Rules |
| --- | --- | --- | --- |
| `operation` | string | No | `add` or `subtract`. Omitted, empty, or whitespace-only values default to add. |
| `mileage_adjustment` | integer | One adjustment must be positive | An unsigned magnitude from 0 through 10,000. |
| `hours_adjustment` | integer | One adjustment must be positive | An unsigned magnitude from 0 through 10,000. |

Values are magnitudes, not signed deltas: never send a negative number. A
single `operation` applies to both supplied fields; a request cannot add one
field while subtracting the other. Supply at least one positive adjustment.
Both fields may be supplied, or either field may be omitted. A zero field is
permitted when the other field is positive, but a request with both fields
omitted or with no positive value is invalid. Unknown fields, malformed JSON,
and trailing JSON values are rejected.

The resulting tracked value may be exactly zero: subtraction to zero is valid.
The operation is rejected atomically if either resulting value would be below
zero or above the supported `int32` range.

## Add examples

Omitting `operation` defaults to add:

```json
{
  "mileage_adjustment": 25
}
```

To add both tracked values explicitly:

```json
{
  "operation": "add",
  "mileage_adjustment": 25,
  "hours_adjustment": 5
}
```

## Subtract examples

One subtract operation applies to both supplied magnitudes:

```json
{
  "operation": "subtract",
  "mileage_adjustment": 10,
  "hours_adjustment": 5
}
```

For example, subtracting `10` from a tracked mileage of `10` succeeds and
persists `tracked_mileage: 0`.

## Success response

On `200 OK`, the server returns the full persisted Shop vehicle in the standard
success envelope. Treat the returned vehicle, including `tracked_mileage` and
`tracked_hours`, as authoritative rather than calculating local final values.

```json
{
  "status": 200,
  "data": {
    "id": "a7e4abf1-18a3-4fa5-bd31-ef46b4e09b1b",
    "creator_id": "firebase-user-id",
    "niin": "1234-00-123-4567",
    "admin": "ADMIN-001",
    "model": "M1123",
    "serial": "SERIAL-001",
    "uoc": "ABC",
    "mileage": 100,
    "hours": 50,
    "comment": "owner-managed metadata",
    "save_time": "2026-08-22T12:00:00Z",
    "last_updated": "2026-08-22T12:30:00Z",
    "shop_id": "c1d9b0b7-8a88-43f2-9d55-42c3716dfa12",
    "tracked_mileage": 125,
    "tracked_hours": 55
  },
  "message": "Equipment usage adjusted successfully"
}
```

## Validation and error responses

The endpoint uses the standard `{status, data, message}` envelope for every
error. `data` is `null`. Clients must use the HTTP status for control flow and
may display `message` as appropriate; exact validation detail can vary.

`400 Bad Request` — invalid request shape or values, such as a negative
magnitude, an unsupported operation, no positive adjustment, unknown fields,
or malformed JSON:

```json
{
  "status": 400,
  "data": null,
  "message": "invalid usage adjustment: adjustments must be between 0 and 10000"
}
```

`401 Unauthorized` — no authenticated user:

```json
{
  "status": 401,
  "data": null,
  "message": "unauthorized"
}
```

`403 Forbidden` — authenticated user is not a member of the vehicle's Shop:

```json
{
  "status": 403,
  "data": null,
  "message": "shop access denied"
}
```

`404 Not Found` — vehicle does not exist:

```json
{
  "status": 404,
  "data": null,
  "message": "shop vehicle not found"
}
```

`409 Conflict` — the requested adjustment would make tracked usage negative or
would overflow the supported range. Neither field is persisted when this occurs:

```json
{
  "status": 409,
  "data": null,
  "message": "usage adjustment would move tracked usage outside the supported range"
}
```

`500 Internal Server Error` — an unexpected server failure:

```json
{
  "status": 500,
  "data": null,
  "message": "internal Server Error"
}
```

## Backward compatibility

The existing absolute-update endpoint remains available for valid legacy
clients:

| Client operation | Endpoint | Behavior |
| --- | --- | --- |
| Usage adjustment client | `PATCH /api/v1/auth/shops/vehicles/:vehicle_id/usage` | Membership-authorized atomic add/subtract of tracked usage; returns the persisted full vehicle. |
| Legacy absolute update client | `PUT /api/v1/auth/shops/vehicles` | Unchanged for valid absolute vehicle updates. Negative `tracked_mileage` or `tracked_hours` is rejected with `400`. |

New usage-adjustment clients must use PATCH. PATCH must not be used to update
vehicle metadata, and the legacy PUT endpoint is not a fallback transport for
PATCH usage adjustments.

## Retry and reconciliation guidance

This request is not safely replayable after an ambiguous failure because it is
an increment/decrement operation. Do not automatically retry it, and do not
fall back to legacy PUT after a timeout, connection failure, or any other
ambiguous result.

If the client does not receive a definitive response, fetch the vehicle through
the normal authenticated vehicle read path and reconcile its displayed state
from the authoritative persisted `tracked_mileage` and `tracked_hours`. Only
after reconciliation and explicit user direction may the client submit a new
adjustment. A definitive `400`, `401`, `403`, `404`, or `409` did not apply the
requested usage adjustment; correct the request, access, or user decision
before another attempt.

## Deployment order

Deploy in this order:

1. Apply database migration `015` (Shop vehicle tracked-usage constraints).
2. Deploy the server version that exposes this PATCH endpoint and legacy PUT
   hardening.
3. Deploy Flutter clients that use the PATCH transport and reconciliation rules.

Do not ship the Flutter PATCH client before the migration and server are live.
Coordinate any rollback across the migration, server, and Flutter rollout: do
not leave deployed Flutter PATCH clients targeting a server or migration state
that has been rolled back below this endpoint's contract.
