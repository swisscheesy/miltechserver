# Shop Equipment Usage Adjustments Design

Date: 2026-08-21
Status: Approved for implementation planning

## Problem

Shop equipment usage currently moves in only one practical direction. The
Flutter dialog accepts positive digits, the Cubit calculates an absolute
tracked total, and the legacy server vehicle-update endpoint writes that total.
Users cannot correct an overstatement by subtracting mileage or hours.

The legacy endpoint is also the full equipment-edit endpoint. Its
`tracked_mileage` and `tracked_hours` fields are absolute totals, so changing
their meaning to deltas would break installed clients. A usage adjustment must
therefore use a separate contract.

The client currently collapses a remote `null` tracked value to `0`, then treats
all tracked values at or below zero as uninitialized. That makes an explicit
subtraction result of zero display the original base reading. Supporting zero
requires preserving the distinction between an absent tracked value and an
explicit tracked value of zero.

## Goals

- Allow any current Shop member to add or subtract equipment mileage and/or
  hours.
- Use one operation selection for both supplied fields.
- Permit subtraction to exactly zero and reject any result below zero.
- Make the server authoritative for authenticated Shop adjustments.
- Apply concurrent adjustments without lost updates.
- Preserve the legacy `PUT /api/v1/auth/shops/vehicles` request and response
  contract for existing clients.
- Keep Flutter Drift at schema version 6 because v6 has not shipped.
- Return the persisted equipment record from the new endpoint.
- Document the new endpoint for the client team with request, response, error,
  compatibility, and deployment details.

## Non-goals

- Reinterpreting legacy absolute tracked fields as deltas.
- Allowing clients to send negative adjustment magnitudes.
- Supporting different operations for mileage and hours in one request.
- Adding an equipment-usage audit-history table.
- Adding an idempotency-key table or automatic mutation retries.
- Changing creator/admin permissions for equipment metadata, deletion, or
  other Shop operations.
- Incrementing the Flutter database to schema version 7.

## Approved API contract

### Route

`PATCH /api/v1/auth/shops/vehicles/:vehicle_id/usage`

### Request

```json
{
  "operation": "subtract",
  "mileage_adjustment": 10,
  "hours_adjustment": 5
}
```

Rules:

- `operation` accepts only `add` and `subtract`.
- An omitted or blank `operation` defaults to `add`.
- The single operation applies to every supplied adjustment field.
- `mileage_adjustment` and `hours_adjustment` are optional integer magnitudes.
- At least one adjustment must be greater than zero.
- A supplied adjustment must be between 0 and 10,000 inclusive; because the
  request must contain at least one positive value, two zeros are invalid.
- Negative magnitudes are invalid. The operation, not the sign, chooses the
  direction.
- Unknown JSON fields and multiple JSON values in the body are rejected.
- A computed result may equal zero but may not be negative or exceed the
  PostgreSQL/Go `int32` range used by the current schema and generated model.

### Success response

```json
{
  "status": 200,
  "message": "Equipment usage adjusted successfully",
  "data": {
    "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "creator_id": "user-1",
    "niin": "012345678",
    "admin": "A123",
    "model": "M1097",
    "serial": "SER-10001",
    "uoc": "ABC",
    "mileage": 100,
    "hours": 50,
    "comment": "",
    "save_time": "2026-08-21T18:00:00Z",
    "last_updated": "2026-08-21T19:30:00Z",
    "shop_id": "shop-1",
    "tracked_mileage": 90,
    "tracked_hours": 45
  }
}
```

The response must be the row returned by the persistence operation. An HTTP 2xx
status without a decodable equipment record is not client success.

### Errors

Errors use the existing standard envelope with `data: null`:

```json
{
  "status": 409,
  "message": "usage adjustment would move tracked usage outside the supported range",
  "data": null
}
```

- `400 Bad Request`: malformed JSON, unknown field, invalid operation, negative
  magnitude, magnitude over 10,000, or no positive adjustment.
- `401 Unauthorized`: authenticated user context is missing.
- `403 Forbidden`: the caller is not a current member of the equipment's Shop.
- `404 Not Found`: the equipment does not exist.
- `409 Conflict`: the authoritative result would be below zero or above the
  supported `int32` maximum.
- `500 Internal Server Error`: unexpected authorization or persistence error;
  the response does not expose database details.

## Server architecture

The new handler performs strict JSON decoding and maps typed usage errors
locally instead of sending expected client errors through the current global
middleware, which otherwise converts most errors to HTTP 500.

The service validates request shape, defaults the operation to `add`, loads the
equipment to obtain its Shop, verifies current Shop membership, sets a UTC
`last_updated`, and asks the repository to apply the adjustment.

The repository uses a short, context-aware PostgreSQL transaction:

1. Select the equipment row by ID with `FOR UPDATE`.
2. Derive each current usage value as `tracked` when non-null, otherwise the
   base `mileage` or `hours` value.
3. Calculate supplied fields using `int64` intermediates.
4. Reject the whole request if either result is outside `0..2147483647`.
5. Update only the tracked fields supplied by the request plus `last_updated`.
6. Return all columns from the update and commit.

No network call or authorization query occurs while the row lock is held. This
keeps the lock scope to the read/calculate/update sequence. Concurrent calls to
the new endpoint serialize on the equipment row and cannot overwrite one
another from stale client snapshots.

The legacy absolute PUT remains available. Valid request and response behavior
is unchanged, but negative `tracked_mileage` or `tracked_hours` values are
rejected. Legacy absolute writes can still overwrite a concurrent adjustment;
the new Flutter client therefore never falls back to the legacy PUT.

## PostgreSQL invariant

Migration 015 adds and validates two named constraints:

```sql
tracked_mileage IS NULL OR tracked_mileage >= 0
tracked_hours IS NULL OR tracked_hours >= 0
```

The migration uses a transaction, a bounded lock timeout, `NOT VALID`, and then
`VALIDATE CONSTRAINT`. Existing negative rows make migration validation fail
atomically instead of being silently rewritten. The rollback drops only the two
named constraints.

The constraints do not change table columns or generated Jet types, so Jet
generation is not required for this migration.

## Flutter architecture

### Operation model

Create one shared enum with wire values `add` and `subtract` and an `apply`
helper. The dialog, Cubit, remote repository, and local repository all consume
that enum. The UI always sends an explicit operation even though the server
defaults an omitted operation to `add` for compatibility.

### Remote authority

Authenticated Shops call the PATCH endpoint with magnitudes. The Cubit does not
calculate or send absolute totals. The API parses the response `data` as
`ShopVehicleRM`; the repository verifies the returned vehicle ID and converts
it to the domain model; the Cubit stores the returned vehicle in success state.

There is no optimistic local write for authenticated Shops, no fallback to the
legacy PUT, and no automatic retry. Because the endpoint is not idempotent, an
automatic retry after an ambiguous network failure could apply an adjustment
twice. The UI tells the user to refresh current usage before trying again when
the outcome cannot be confirmed.

### Local authority

Logged-out visible Shops apply the same operation semantics inside a Drift
transaction. The repository reads the current row, computes both results,
rejects the whole operation if either result is negative, updates only supplied
tracked fields, and returns the persisted domain vehicle. Hidden PMCS local
equipment remains excluded by the existing guard.

### Null versus zero

`ShopVehicle.trackedMileage` and `trackedHours` become nullable. Effective usage
uses null-coalescing:

```dart
int get effectiveMileage => trackedMileage ?? mileage;
int get effectiveHours => trackedHours ?? hours;
```

Remote `null` stays null. Explicit remote or local zero stays zero.

The Drift table columns become nullable. Schema version remains exactly 6:

- Fresh v6 databases create nullable tracked columns.
- Historical schema snapshots v1-v5 remain historical.
- The existing `from5To6` step alters `shop_vehicles` and transforms legacy
  zero sentinels with `NULLIF(tracked_mileage, 0)` and
  `NULLIF(tracked_hours, 0)`.
- The v6 schema dump, generated v6 Dart schema, `database.steps.dart`, and
  `database.g.dart` are regenerated from source. They are never hand-edited.

Zero is safe to transform during the v5-to-v6 migration because subtraction to
zero was not previously supported and existing client logic treated all zero
tracked values as the uninitialized sentinel.

## User experience

The existing update dialog gets one single-select Material `SegmentedButton`:

```text
[ Add ] [ Subtract ]
```

- Add is selected by default.
- The selection controls both fields.
- Inputs remain positive digits only.
- Labels and instructions reflect the selected operation.
- Current values use effective usage.
- A live preview shows current and resulting mileage/hours.
- Either field may be empty.
- The update button is disabled when both fields are empty/zero, either field
  exceeds 10,000, or subtraction would make a result negative.
- A result of zero remains enabled.
- The dialog stays open on failure and displays a refresh-before-retry message
  for an unconfirmed remote outcome.

## Compatibility and rollout

- Old client with new server: legacy PUT continues to work.
- New client with new server: PATCH provides atomic adjustments and persisted
  responses.
- New client with old server: PATCH returns 404; the client reports failure and
  does not fall back to PUT.
- Deployment order is migration 015, server endpoint, then Flutter release.
- Server rollback after the Flutter release makes adjustments unavailable, so
  rollback must be coordinated with the client release state.

## Acceptance criteria

- Any current Shop member can add or subtract one or both usage fields.
- One operation applies to both fields.
- Missing operation adds.
- Explicit zero is displayed as zero.
- Subtraction to zero succeeds.
- A below-zero result returns 409 and changes neither field.
- Concurrent adjustments through the new endpoint are cumulative.
- Equipment metadata cannot be changed through the usage route.
- A negative legacy absolute tracked value is rejected.
- Existing valid legacy PUT calls remain compatible.
- Flutter database version remains 6 and no v7 migration artifacts exist.
- The client never reports success without a valid persisted equipment record.
- Client API documentation contains exact request, response, error, and rollout
  examples.
