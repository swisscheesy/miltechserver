# Custom PMCS Shop Inspection Integration — Server Design

Date: 2026-08-07
Status: Approved for implementation planning

## Problem

The Shop PMCS inspection API models every inspection as guide-backed. The
`pmcs_sbs_inspections.guide_manual` column is required and constrained to a
`pmcs_sbs/*.json` path, and every inspection response and equipment-history
summary assumes that identity.

The mobile application now supports user-created PMCS checklists. Users must
be able to run those checklists against Shop equipment and have the resulting
faults participate in the same Shop inspection history, detail, comments,
notes, clean-completion, and DA Form 2404 workflows as guide-backed PMCS.
Encoding custom checklist identity in a fabricated guide path would corrupt
inspection provenance and break guide loading and display behavior.

## Goal

Generalize the existing authenticated Shop inspection record so one
inspection has exactly one immutable source:

- `guide`: identified by `guide_manual`; or
- `custom`: identified by checklist/revision UUIDs plus an immutable revision
  number and checklist display name.

Existing guide clients and routes remain compatible. Custom inspections reuse
the current Shop membership authorization, transactional fault upsert,
history aggregation, notes, comments, deletion, and fault-count behavior.

## Scope

This design covers `miltechserver` only:

- Postgres migration and rollback;
- regenerated Jet models;
- `api/pmcs_sbs_progress` request, validation, service, and repository changes;
- Shop equipment-history aggregate response changes;
- tests and mobile-facing contract documentation.

Mobile selection, Drift persistence, synchronization, and UI are defined in
`miltech/docs/superpowers/specs/2026-08-07-custom-pmcs-shop-integration-mobile-design.md`.

## Decisions

### One inspection table with an explicit source union

`pmcs_sbs_inspections` remains the parent of `pmcs_sbs_faults`. It gains an
explicit `source_type` and nullable source-specific columns. A database CHECK
constraint enforces the union so an inspection can never be both guide and
custom or neither.

Rejected alternatives:

- Synthetic `guide_manual` values such as `custom/<uuid>.json`: violates the
  guide-path contract and makes history, conflict handling, and content
  loading dishonest.
- Separate custom inspection/fault tables and routes: duplicates vehicle
  authorization, comments, history aggregation, lifecycle, and deletion.

### Snapshot provenance, not a live checklist foreign key

Custom source metadata is copied onto the inspection and has no foreign key to
`user_pmcs_checklists` or `user_pmcs_revisions`. This is required because a
device-local checklist may never exist on the server and historical records
must survive checklist retirement, unsubscribe, owner deletion, or revision
garbage collection.

The server does not copy the authored checklist tree. Shop history detail is
fault-oriented today, and the immutable source name/revision plus fault
location labels are sufficient for history, comments, and 2404 export.

### Backward-compatible routes and guide JSON

All routes stay under `/api/v1/auth/pmcs-sbs/equipment/:equipment_id/pmcs`.
Guide requests may omit `source_type`; the server interprets a nonblank
`guide_manual` with no custom fields as `guide`. Responses always include
`source_type`. Existing `guide_manual` remains a string for guide records and
is omitted for custom records.

### Existing authorization remains authoritative

Every operation first validates that the authenticated user belongs to the
Shop owning `equipment_id`. Every inspection lookup also filters by both
`pmcs_id` and `equipment_id`. Custom checklist ownership is deliberately not
an authorization condition: the shared Shop record is the inspection and
fault data, not access to the original private checklist content.

### Write-through lifecycle, no server queue

The client continues the existing autosave contract. Saving the first custom
fault implicitly creates the inspection in the same transaction. Finishing a
clean inspection explicitly upserts the inspection. The server adds no job,
outbox, draft status, or background reconciliation mechanism.

## Database design

Migration `011_extend_pmcs_inspection_sources.sql` alters existing tables in
place and preserves all guide-backed history.

### `pmcs_sbs_inspections`

Add:

```sql
source_type             TEXT NOT NULL DEFAULT 'guide',
custom_checklist_id     UUID,
custom_revision_id      UUID,
custom_revision_number  INTEGER,
custom_checklist_name   TEXT
```

Change `guide_manual` from `TEXT NOT NULL` to nullable `TEXT`.

Replace the current guide-only constraints with:

```sql
CHECK (source_type IN ('guide', 'custom'))

CHECK (
  (
    source_type = 'guide'
    AND guide_manual IS NOT NULL
    AND guide_manual = btrim(guide_manual)
    AND guide_manual LIKE 'pmcs_sbs/%'
    AND right(guide_manual, 5) = '.json'
    AND custom_checklist_id IS NULL
    AND custom_revision_id IS NULL
    AND custom_revision_number IS NULL
    AND custom_checklist_name IS NULL
  )
  OR
  (
    source_type = 'custom'
    AND guide_manual IS NULL
    AND custom_checklist_id IS NOT NULL
    AND custom_revision_id IS NOT NULL
    AND custom_revision_number >= 0
    AND custom_checklist_name = btrim(custom_checklist_name)
    AND btrim(custom_checklist_name) <> ''
  )
)
```

The existing `(equipment_id, performed_date DESC)` index remains the correct
history index. No source index is added because current queries start with an
equipment UUID and do not filter by custom checklist identity. New indexes
require measured query evidence.

### `pmcs_sbs_faults`

Add nullable `section_title TEXT`. `section_id` remains the opaque identity
used in the primary key `(pmcs_id, section_id, item_index)`; custom clients
send the immutable snapshot-section UUID. `section_title` is presentation
metadata and does not participate in identity. Guide clients may omit it and
guide detail continues falling back to `section_id`.

### Migration and rollback order

Forward migration:

1. add source columns with `source_type DEFAULT 'guide'`;
2. make `guide_manual` nullable;
3. add `section_title`;
4. drop guide-only CHECK constraints;
5. add the source-union constraints;
6. remove the `source_type` default so all future inserts are explicit.

Rollback is permitted only in the test rehearsal. It must first prove that no
`source_type = 'custom'` rows exist, then remove the new constraints/columns,
restore `guide_manual NOT NULL`, and restore the original guide constraints.
It must fail rather than discard custom inspection history.

## API contract

### Source fields

The following additive fields appear on inspection create/fault requests and
inspection/history responses:

```json
{
  "source_type": "custom",
  "custom_checklist_id": "0f6e54e4-8143-47a8-a6c7-f969eae5f8b6",
  "custom_revision_id": "ed058e1a-cfec-4e81-9769-69fa76b2c365",
  "custom_revision_number": 3,
  "custom_checklist_name": "Weekly Generator PMCS"
}
```

Guide requests remain:

```json
{
  "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json"
}
```

Guide responses add `"source_type": "guide"`. Source-inapplicable fields are
omitted, not emitted as empty strings or zero UUIDs.

### Save a custom fault

`PUT /api/v1/auth/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults`

```json
{
  "source_type": "custom",
  "custom_checklist_id": "0f6e54e4-8143-47a8-a6c7-f969eae5f8b6",
  "custom_revision_id": "ed058e1a-cfec-4e81-9769-69fa76b2c365",
  "custom_revision_number": 3,
  "custom_checklist_name": "Weekly Generator PMCS",
  "performed_date": "2026-08-07T16:00:00Z",
  "section_id": "8c554475-97a2-459d-bf63-d7ce40ee909c",
  "section_title": "Before Operation",
  "item_index": 1,
  "item_no": "1",
  "status": "x",
  "fault_text": "Oil level below operating range",
  "corrective_action": ""
}
```

The response retains the existing fault shape and adds `section_title`.

### Explicit completion

`PUT /api/v1/auth/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id`

The body carries the same source fields plus `performed_date` and nullable
`notes`. This materializes clean custom inspections and finalizes the
performed date/note for inspections already created by fault autosave.

### History and detail

Both `GET .../pmcs` and the Shop aggregate
`GET /api/v1/auth/shops/equipment-pmcs-history` return source-discriminated
summaries. Custom summaries include checklist/revision metadata, performer,
fault count, and comment count. `GET .../pmcs/:pmcs_id` returns the same source
metadata with faults, notes, and comments.

Existing optional `guide_manual` history filtering remains guide-only. No
custom checklist filter is added in this scope.

## Validation

- `source_type` accepts only `guide` or `custom`.
- Omitted `source_type` is accepted only for a guide-shaped legacy request.
- Guide requests use the existing canonical path validation.
- Custom checklist and revision identifiers must parse as nonzero UUIDs.
- Custom revision number is `>= 0`; zero represents a device-local working
  revision snapshot.
- Checklist name and `section_title`, when present, use the existing User-PMCS
  short-field limits: at most 200 grapheme clusters and 8 KiB UTF-8.
- Mixed guide/custom fields, blank custom names, invalid UTF-8, and unknown
  source types return the existing invalid-request response without echoing
  authored text.
- A reused `pmcs_id` must match equipment ID and the complete immutable source
  tuple. Any mismatch returns `409 pmcs sbs inspection conflict`.

## Repository and transaction behavior

`ensureInspection` remains one atomic `INSERT ... ON CONFLICT ... DO UPDATE
... WHERE ... RETURNING` operation. The conflict WHERE clause compares:

- `equipment_id`;
- `source_type`;
- `guide_manual IS NOT DISTINCT FROM ...`;
- custom checklist/revision UUIDs;
- custom revision number; and
- custom checklist name.

Only `performed_date`, `notes`, and `updated_at` are mutable. External calls,
JSON work, Unicode validation, and authorization queries stay outside the
fault-upsert transaction. The transaction contains only inspection ensure,
fault upsert, and commit.

## Error handling and privacy

- Vehicle membership failures remain not-found to avoid cross-Shop leakage.
- Missing/mismatched inspection IDs preserve existing 404/409 behavior.
- Database errors are wrapped with operation context but never include
  checklist names, notes, fault text, corrective action, or request bodies.
- Custom checklist name and fault data become visible to every member who can
  access the Shop equipment inspection history. The authored checklist tree
  remains private and is not copied into Shop inspection storage.

## Testing

### Migration

- Existing guide rows survive forward migration unchanged.
- Guide and custom source CHECK constraints accept valid rows and reject
  mixed/incomplete rows.
- `section_title` is nullable for existing guide faults.
- Rollback succeeds with guide-only data and refuses when custom rows exist.
- Forward/rollback/forward is rehearsed only against `miltech_ng_test`.

### Service and route

- Legacy guide request compatibility.
- Valid custom inspection and fault decoding.
- All invalid source-union cases.
- UUID, revision-number, name, and section-title boundaries.
- Correct status mapping and response omission rules.
- Route error mapping remains stable.

### Repository and aggregate

- Implicit custom inspection creation with fault upsert.
- Explicit zero-fault custom inspection creation.
- Idempotent retry with identical provenance.
- Conflict on equipment or source mutation.
- Vehicle membership authorization for every operation.
- Custom source metadata in list, detail, and Shop aggregate history.
- Fault and comment counts for custom records.
- Vehicle deletion cascades custom inspections, faults, and comments.

## Delivery boundaries

- No production migration, deployment, push, or merge is authorized by this
  design or its implementation plan.
- Apply forward/rollback/forward only to `miltech_ng_test`; apply forward once
  to non-production `miltech_ng`; never roll back `miltech_ng`.
- Regenerate Jet only after both non-production forward migrations succeed.
- Do not hand-edit `.gen` files.
- Update the existing PMCS inspection API documentation and add a standalone
  mobile contract handoff before mobile implementation begins.
