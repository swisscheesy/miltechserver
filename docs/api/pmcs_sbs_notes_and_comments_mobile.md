# PMCS SBS Inspection Notes + Comments Mobile Handoff

## Summary

PMCS SBS inspections now support two new pieces of user-generated context, on top of the existing inspection/fault data described in `pmcs_sbs_inspections_mobile.md`:

1. **Notes** — a single free-text field on the inspection itself, editable by any shop member with access to the vehicle (same access rule as `guide_manual`/`performed_date` today). Set or cleared through the existing save-inspection endpoint — no new endpoint for this.
2. **Comments** — a flat, chronological discussion thread on an inspection. Any shop member with access to the vehicle can post a comment; only the comment's author can edit or delete it. Three new endpoints. Comments are always returned embedded in the single-inspection `GET`, never fetched separately.

Why: field techs and shop members reviewing an inspection had no way to leave context ("torque spec was off, re-checked and it's fine") or discuss a finding with each other after the fact. Everything before this was either a fixed checklist item (`fault_text`/`corrective_action` per line item) or nothing at all.

This is purely additive — no existing field, endpoint, or response shape changes or moves. See ADR-018 in `docs/project_notes/decisions.md` for the full design rationale.

## What Changed For Mobile

| Endpoint | Method | Change |
|----------|--------|--------|
| `/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id` | `PUT` (save inspection) | **Added:** optional `notes` request field. **Added:** `notes` response field. |
| `/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id` | `GET` (get inspection) | **Added:** `notes` response field. **Added:** `comments` array (full comment objects). |
| `/pmcs-sbs/equipment/:equipment_id/pmcs` | `GET` (list inspections) | **Added:** `comment_count` response field. `notes` is intentionally **not** included here — full note text is a detail-view concern, same as `faults`. |
| `/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/comments` | `POST` (new) | Add a comment to an inspection. |
| `/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/comments/:comment_id` | `PUT` (new) | Edit a comment you authored. |
| `/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/comments/:comment_id` | `DELETE` (new) | Soft-delete a comment you authored. |

Endpoints not listed here — delete inspection, and all fault endpoints — are unaffected.

## Notes

### Field Semantics

| Field | Type | Notes |
|-------|------|-------|
| `notes` | string, nullable | Free-text, max 4000 characters. Trimmed server-side; an all-whitespace or omitted value clears the note (stored/returned as absent, not an empty string). |

`notes` is edited the same way as every other mutable inspection field: resend the full `InspectionRequest` body on `PUT .../pmcs/:pmcs_id`. There is no partial-update — omitting `notes` from a save request clears it, it does not leave the previous value untouched.

### Save Inspection With a Note

`PUT /api/v1/auth/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id`

Request:

```json
{
  "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
  "performed_date": "2026-07-21T14:30:00Z",
  "notes": "Battery terminals showed light corrosion, cleaned during PMCS."
}
```

Response:

```json
{
  "status": 200,
  "data": {
    "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "equipment_id": "550e8400-e29b-41d4-a716-446655440000",
    "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
    "performed_date": "2026-07-21T14:30:00Z",
    "performed_by": "9f1c3a2e-user-uid",
    "performed_by_username": "jsmith",
    "notes": "Battery terminals showed light corrosion, cleaned during PMCS.",
    "created_at": "2026-07-21T14:31:02.123456Z",
    "updated_at": "2026-07-21T14:31:02.123456Z",
    "faults": [],
    "comments": []
  },
  "message": "Inspection saved"
}
```

To clear an existing note, resend the save request with `"notes": null` (or omit the field entirely):

```json
{
  "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
  "performed_date": "2026-07-21T14:30:00Z"
}
```

## Comments

### Field Semantics

| Field | Type | Notes |
|-------|------|-------|
| `id` | string (UUID) | Server-generated — never send one on create. |
| `pmcs_id` | string (UUID) | The inspection this comment belongs to. |
| `author_id` | string | User id of whoever posted the comment. |
| `author_username` | string, nullable | Display username for `author_id`. Omitted when there is no value (e.g. the author account was later deleted). |
| `text` | string | 1–2000 characters, trimmed server-side. |
| `created_at` | string | ISO timestamp. Never changes after creation. |
| `updated_at` | string, nullable | ISO timestamp of the last edit or delete. **Omitted** on a comment that has never been edited — its presence is how you distinguish an edited/deleted comment from a fresh one. |

Comments are returned oldest-first inside the inspection's `comments` array — there is no separate "fetch comments" endpoint, no pagination, and no `parent_id`/threading.

**Deletion is soft.** `DELETE` does not remove the comment row — it rewrites `text` to the literal string `"Deleted by user"` and stamps `updated_at`. The comment stays in the array (and counts toward `comment_count`) after deletion; render it accordingly rather than filtering it out, unless product wants client-side filtering on that sentinel text.

### Add a Comment

`POST /api/v1/auth/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/comments`

Request:

```json
{
  "text": "Re-checked the leak after cleaning — looks resolved."
}
```

Response:

```json
{
  "status": 201,
  "data": {
    "id": "a1b2c3d4-e5f6-4789-a0b1-c2d3e4f5a6b7",
    "pmcs_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "author_id": "9f1c3a2e-user-uid",
    "author_username": "jsmith",
    "text": "Re-checked the leak after cleaning — looks resolved.",
    "created_at": "2026-07-21T15:02:11.456789Z"
  },
  "message": "Comment created"
}
```

### Edit a Comment

`PUT /api/v1/auth/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/comments/:comment_id`

Only the original author may call this — any other caller gets `403`.

Request:

```json
{
  "text": "Re-checked the leak after cleaning — confirmed resolved, no further action."
}
```

Response:

```json
{
  "status": 200,
  "data": {
    "id": "a1b2c3d4-e5f6-4789-a0b1-c2d3e4f5a6b7",
    "pmcs_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "author_id": "9f1c3a2e-user-uid",
    "author_username": "jsmith",
    "text": "Re-checked the leak after cleaning — confirmed resolved, no further action.",
    "created_at": "2026-07-21T15:02:11.456789Z",
    "updated_at": "2026-07-21T15:05:47.998877Z"
  },
  "message": "Comment updated"
}
```

### Delete a Comment

`DELETE /api/v1/auth/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/comments/:comment_id`

No request body. Only the original author may call this — any other caller gets `403`.

Response:

```json
{
  "status": 200,
  "data": {
    "id": "a1b2c3d4-e5f6-4789-a0b1-c2d3e4f5a6b7",
    "pmcs_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "author_id": "9f1c3a2e-user-uid",
    "author_username": "jsmith",
    "text": "Deleted by user",
    "created_at": "2026-07-21T15:02:11.456789Z",
    "updated_at": "2026-07-21T15:08:03.112233Z"
  },
  "message": "Comment deleted"
}
```

### Get Inspection — Notes and Comments Together

`GET /api/v1/auth/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id`

```json
{
  "status": 200,
  "data": {
    "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "equipment_id": "550e8400-e29b-41d4-a716-446655440000",
    "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
    "performed_date": "2026-07-21T14:30:00Z",
    "performed_by": "9f1c3a2e-user-uid",
    "performed_by_username": "jsmith",
    "notes": "Battery terminals showed light corrosion, cleaned during PMCS.",
    "created_at": "2026-07-21T14:31:02.123456Z",
    "updated_at": "2026-07-21T14:31:02.123456Z",
    "faults": [
      {
        "pmcs_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
        "section_id": "before",
        "item_index": 0,
        "item_no": "1",
        "status": "x",
        "fault_text": "Oil leak observed",
        "corrective_action": "",
        "created_at": "2026-07-21T14:44:02.123456Z",
        "updated_at": "2026-07-21T14:44:02.123456Z"
      }
    ],
    "comments": [
      {
        "id": "a1b2c3d4-e5f6-4789-a0b1-c2d3e4f5a6b7",
        "pmcs_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
        "author_id": "9f1c3a2e-user-uid",
        "author_username": "jsmith",
        "text": "Re-checked the leak after cleaning — confirmed resolved, no further action.",
        "created_at": "2026-07-21T15:02:11.456789Z",
        "updated_at": "2026-07-21T15:05:47.998877Z"
      },
      {
        "id": "b2c3d4e5-f6a7-4890-b1c2-d3e4f5a6b7c8",
        "pmcs_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
        "author_id": "b2f1c1b4-user-uid",
        "author_username": "mwright",
        "text": "Good catch, thanks for confirming.",
        "created_at": "2026-07-21T15:10:00.000000Z"
      }
    ]
  },
  "message": ""
}
```

Note the second comment has no `updated_at` — it has never been edited.

### List Inspections — Comment Count Only

`GET /api/v1/auth/pmcs-sbs/equipment/:equipment_id/pmcs`

```json
{
  "status": 200,
  "data": {
    "inspections": [
      {
        "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
        "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
        "performed_date": "2026-07-21T14:30:00Z",
        "fault_count": 1,
        "comment_count": 2,
        "created_at": "2026-07-21T14:31:02.123456Z",
        "performed_by": "9f1c3a2e-user-uid",
        "performed_by_username": "jsmith"
      }
    ],
    "count": 1
  },
  "message": ""
}
```

Note `notes` is absent here — fetch the single inspection via `GET .../pmcs/:pmcs_id` to read its note text.

## Error Responses

New/changed cases on top of what `pmcs_sbs_inspections_mobile.md` already documents:

| HTTP | Cause |
|------|-------|
| `400` | (comment endpoints) `text` is blank or exceeds 2000 characters. (save endpoint) `notes` exceeds 4000 characters. |
| `403` | Editing or deleting a comment authored by a different user. |
| `404` | (comment endpoints) `comment_id` not found, or the inspection's `pmcs_id` does not belong to the URL's `equipment_id` (same cross-vehicle boundary rule that already applies to faults). |

## Mobile Implementation Checklist

- Add an optional `notes` string field to the save-inspection request model and send it (or `null`) on every save — there is no partial-update, so a client that doesn't round-trip the field will silently clear notes on the next save.
- Add `notes` (nullable) to the inspection detail response model. Do **not** add it to the inspection summary/list model — it is intentionally absent there.
- Add `comments` (array, detail response only) and `comment_count` (integer, list response only) to their respective models.
- Add a `Comment` model: `id`, `pmcs_id`, `author_id`, `author_username` (nullable), `text`, `created_at`, `updated_at` (nullable — its absence means "never edited").
- Implement the three new comment endpoints (create/edit/delete) and wire them to an inspection-detail UI, gating edit/delete controls to `author_id == current_user_id` client-side (the server enforces this too and returns `403`, but hiding the control avoids a round-trip for the common case).
- Render soft-deleted comments (`text == "Deleted by user"`) rather than filtering them client-side, to match server behavior — or explicitly decide to filter them and confirm that's the desired UX, since the server does not do this filtering for you.
- Treat `notes`, `author_username`, and `updated_at` (on comments) as optional/nullable in every model that parses them.
