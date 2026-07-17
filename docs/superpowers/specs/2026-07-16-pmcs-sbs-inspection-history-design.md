# PMCS SBS Inspection History — Design

Date: 2026-07-16
Status: Approved for implementation planning

## Problem

`pmcs_sbs_faults` today has no inspection-event dimension. Its primary key is
`(equipment_id, guide_manual, section_id, item_index)`, so saving a fault for
the same vehicle/guide/section/item always overwrites whatever was saved
before (`api/pmcs_sbs_progress/repository_impl.go:81-92`, an
`ON_CONFLICT ... DO_UPDATE` on that exact key). There is no table recording
that a PMCS was performed — only the latest fault state per checklist item.

This means:
- A vehicle can only ever show its *current* set of open faults, never a
  history of past inspections.
- A clean inspection (zero faults found) leaves no trace at all, since only
  fault rows persist.

This reverses part of a deliberate prior simplification: a 2026-06-21 design
(`docs/OLD/superpowers/plans/2026-06-21-pmcs-sbs-faults-only.md`) removed a
prior `pmcs_sbs_equipment` + `pmcs_sbs_completions` pair of tables specifically
to reduce the feature to "faults only." That decision's tradeoffs were
intentional, not oversights — this design reintroduces an inspection-event
concept because the product requirement (historical view of PMCS's performed
per vehicle) now requires it.

## Goal

Give users the ability to:
1. Perform multiple PMCS's on the same equipment over different time periods,
   each independently tracking its own faults.
2. Query the server for a list of all PMCS's performed on a piece of
   equipment, the date each was performed, and the faults found during it —
   including PMCS's where no faults were found.

## Scope

Server-only (`miltechserver`): Postgres schema, Jet models, and the Go API in
`api/pmcs_sbs_progress/`. The Flutter client's local Drift mirror
(`PmcsSbsFaultsTable` in the `miltech` repo) and its sync/UI logic are a **hard
downstream dependency** for shipping this end-to-end, but are designed
separately, out of scope here.

## Key decisions

These were confirmed during design review; each has a real alternative that
was considered and rejected — see rationale inline.

1. **Clean inspections are recorded.** A PMCS record is created whenever a
   user performs one, independent of whether any faults are found, so history
   shows every inspection including clean passes. (Alternative rejected: only
   recording inspections that have at least one fault, which would preserve
   today's blind spot.)

2. **Inspection creation is a hybrid of implicit and explicit.** The existing
   live-autosave-per-fault behavior is preserved — saving a fault implicitly
   creates its parent inspection record if one doesn't exist yet. A separate,
   explicit endpoint exists to create an inspection with zero faults, for the
   clean-completion case. (Alternatives rejected: a full explicit
   start/finish lifecycle with an in_progress/completed status — bigger
   behavior change than needed; a single atomic submit-everything-at-the-end
   call — drops today's live-autosave contract.)

3. **The client generates the inspection id (UUID) and sends it with every
   fault save in that session.** This is what tells the server "this is a new
   inspection" versus "continue the most recent one" — there is no
   server-side heuristic (like date-bucketing) to get this wrong.
   (Alternative rejected: one-inspection-per-calendar-day bucketing, which
   would incorrectly merge two real inspections done on the same vehicle on
   the same day.)

4. **`performed_date` is client-supplied, not a server write timestamp.**
   Field techs may inspect a vehicle offline and not sync until later; a
   server-assigned timestamp would record sync time, not inspection time.

5. **No data migration.** The existing `pmcs_sbs_faults` rows represent only
   "current state, no date performed" and are not preserved — the migration
   creates the new schema empty.

## Database schema

### New table: `pmcs_sbs_inspections`

```sql
CREATE TABLE pmcs_sbs_inspections (
    id              uuid NOT NULL PRIMARY KEY,     -- client-generated
    equipment_id    text NOT NULL,
    guide_manual    text NOT NULL,
    performed_date  timestamptz NOT NULL,           -- client-supplied
    created_by      text,                            -- nullable: preserved if creating user is later removed
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT fk_pmcs_sbs_inspections_equipment_id
        FOREIGN KEY (equipment_id) REFERENCES shop_vehicle(id)
        ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT fk_pmcs_sbs_inspections_created_by
        FOREIGN KEY (created_by) REFERENCES users(uid)
        ON UPDATE CASCADE ON DELETE SET NULL,
    CONSTRAINT pmcs_sbs_inspections_nonblank_check
        CHECK (btrim(equipment_id) <> '' AND btrim(guide_manual) <> ''),
    CONSTRAINT pmcs_sbs_inspections_guide_manual_format_check
        CHECK (guide_manual = btrim(guide_manual) AND guide_manual LIKE 'pmcs_sbs/%' AND right(guide_manual, 5) = '.json')
);

CREATE INDEX idx_pmcs_sbs_inspections_equipment_performed
    ON pmcs_sbs_inspections (equipment_id, performed_date DESC);
```

`created_by` is nullable with `ON DELETE SET NULL`, unlike the existing
`shop_vehicle.creator_id` pattern (`NOT NULL DEFAULT ''` combined with an
`ON DELETE SET NULL` FK, which would raise a NOT NULL violation if that
cascade path ever actually fired). Since this is a new column, there's no
reason to copy that inconsistency forward.

### Modified table: `pmcs_sbs_faults`

```sql
CREATE TABLE pmcs_sbs_faults (
    pmcs_id            uuid NOT NULL,
    section_id         text NOT NULL,
    item_index         integer NOT NULL,
    item_no            text NOT NULL,
    status             text NOT NULL,
    fault_text         text NOT NULL,
    corrective_action  text NOT NULL DEFAULT '',
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),

    PRIMARY KEY (pmcs_id, section_id, item_index),
    CONSTRAINT fk_pmcs_sbs_faults_pmcs_id
        FOREIGN KEY (pmcs_id) REFERENCES pmcs_sbs_inspections(id)
        ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT pmcs_sbs_faults_item_index_check CHECK (item_index >= 0),
    CONSTRAINT pmcs_sbs_faults_status_check CHECK (status = ANY (ARRAY['x','slash','dash'])),
    CONSTRAINT pmcs_sbs_faults_nonblank_fields_check
        CHECK (btrim(section_id) <> '' AND btrim(item_no) <> '' AND btrim(fault_text) <> '')
);
```

`equipment_id` and `guide_manual` move to the parent table — a fault's
vehicle/guide is reached via `pmcs_id → pmcs_sbs_inspections`, not duplicated
on every fault row. The PK's leading column (`pmcs_id`) already covers "all
faults for this inspection" lookups; no additional index is needed.

## API surface

All routes stay under `/api/v1/auth/pmcs-sbs/equipment/:equipment_id/...`,
with `pmcs_id` inserted as a new path segment.

| Method | Path | Purpose |
|---|---|---|
| `PUT` | `.../pmcs/:pmcs_id` | Create or update inspection metadata. Body: `{guide_manual, performed_date}`. Idempotent — used explicitly for a clean (zero-fault) completion, or optionally called proactively. `guide_manual` is immutable after first creation; `performed_date` can be corrected. |
| `PUT` | `.../pmcs/:pmcs_id/faults` | Save (create/update) one fault. Body adds `guide_manual` + `performed_date` alongside the existing fault fields (`section_id`, `item_index`, `item_no`, `status`, `fault_text`, `corrective_action`) — same pattern as today's contract, which already requires `guide_manual` on every save. If `pmcs_id` doesn't exist yet, it's created in the same transaction (implicit-creation path). |
| `DELETE` | `.../pmcs/:pmcs_id/faults` | Delete one fault. Body: `{section_id, item_index}` — `guide_manual` no longer needed since `pmcs_id` implies it. |
| `DELETE` | `.../pmcs/:pmcs_id/faults/bulk` | Delete up to 100 faults for one inspection — same shape as today, scoped by `pmcs_id`. |
| `GET` | `.../pmcs` | New. List inspection history for the vehicle: `{id, guide_manual, performed_date, fault_count, created_at}[]`, ordered `performed_date DESC`, optional `guide_manual` filter, paginated (`limit`/`offset`, matching the `equipment_services` convention: default 1000, max 1000). |
| `GET` | `.../pmcs/:pmcs_id` | New (replaces today's `GET .../faults?guide_manual=...`). Returns the inspection plus its full faults array in one response: `{id, guide_manual, performed_date, created_by, faults: [...]}`. |
| `DELETE` | `.../pmcs/:pmcs_id` | New. Deletes an entire inspection and cascades its faults — for removing an accidental or duplicate entry. |

### Guide manual immutability

Since `guide_manual` is fixed at creation, both the explicit `PUT .../pmcs/:pmcs_id`
and the implicit-creation path in `PUT .../pmcs/:pmcs_id/faults` must compare
the request's `guide_manual` against an existing row's stored value when the
inspection already exists. A mismatch is a client bug (or an id collision)
and must be rejected with a 409, not silently ignored — silently accepting a
mismatched value would let a fault get attached to an inspection under the
wrong guide, corrupting the "faults found during this PMCS" grouping the
whole feature exists to provide.

### Cross-vehicle boundary rule

`pmcs_id` is a client-generated UUID, so a request could reference a
`pmcs_id` that exists but belongs to a *different* vehicle than the URL's
`:equipment_id` — by bug or by a client probing across shops. Every query
that takes both `:equipment_id` and `pmcs_id` must filter by
`pmcs_sbs_inspections.equipment_id = :equipment_id` in addition to the
existing shop-membership check (`requireVehicleAccess`), so a mismatched
`pmcs_id` reads as "not found" rather than leaking or corrupting another
vehicle's inspection.

## Migration plan

One new migration (plus rollback), following the existing
`migrations/00N_*.sql` pattern:
- Creates `pmcs_sbs_inspections`.
- Drops the existing `pmcs_sbs_faults` table and recreates it in its new
  shape (simpler and safer than an in-place `ALTER` + backfill, given no data
  needs to be preserved — see Key decision 5).
- Rollback reverses both steps.

Jet models under `.gen/miltech_ng/public/{model,table}/` are regenerated
afterward to pick up both tables.

## Code impact

Every file in `api/pmcs_sbs_progress/` needs updating:
- `types.go` — new `Inspection` DTOs, updated `FaultKey`.
- `repository.go` / `repository_impl.go` — new queries (`EnsureInspection`,
  `ListInspections`, `GetInspection`, `DeleteInspection`), re-keyed fault
  queries, and the equipment-boundary check described above.
- `service.go` / `service_impl.go` — add `performed_date` validation
  (required; no future-date restriction by default).
- `route.go` — three new handlers (`GET .../pmcs`, `GET .../pmcs/:pmcs_id`,
  `DELETE .../pmcs/:pmcs_id`) plus the updated existing ones.
- `errors.go` — add `ErrInspectionNotFound`.

## Test impact

Existing tests need rework, not just additions:
- `TestRepositoryVehicleDeleteCascadesFaults`
  (`tests/pmcs_sbs_progress/repository_test.go:183`) must be extended to
  verify the two-level cascade: `shop_vehicle` → `pmcs_sbs_inspections` →
  `pmcs_sbs_faults`.
- `TestRepositoryFaultsAreScopedByGuideManual` (line 98) becomes "scoped by
  `pmcs_id`" instead.

New tests needed:
- Implicit inspection creation on first fault save.
- Explicit clean-completion `PUT .../pmcs/:pmcs_id`.
- Cross-vehicle `pmcs_id` boundary rejection.
- Inspection-delete cascades its faults but leaves sibling inspections for
  the same vehicle untouched.
- List-endpoint pagination and ordering.

## Documentation impact

- Rewrite `docs/api/pmcs_sbs_faults_guide_manual_mobile.md` and
  `docs/api/pmcs_sbs_bulk_fault_delete_mobile.md` to reflect the new
  contract.
- Add an ADR to `docs/project_notes/decisions.md` recording that this
  reintroduces an inspection-event concept, since it reverses part of the
  2026-06-21 "faults-only" design.

## Out of scope / follow-up required before ship

The Flutter client (`miltech` repo) mirrors `pmcs_sbs_faults` locally in
Drift (`lib/_data/database/tables.dart:492-517`) and autosaves faults with no
current concept of an inspection id. Shipping this change requires a
corresponding Drift schema migration (add `pmcs_id`), and updates to
`PmcsSbsProgressApi` / `PmcsSbsProgressRepository` /
`PmcsSbsStepViewerCubit` to generate and track the client-side inspection
UUID per session. This is designed separately in the `miltech` repo.
