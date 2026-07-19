# PMCS SBS Performed-By Tracking — Design

**Goal:** Whenever a client is given information about a PMCS SBS inspection — single-inspection detail, per-equipment inspection history, or the cross-shop equipment+history aggregate — it should also know who performed that PMCS: a user id and that user's username.

**Context:** The PMCS SBS inspection history feature (`docs/superpowers/specs/2026-07-16-pmcs-sbs-inspection-history-design.md`) introduced `pmcs_sbs_inspections` with a `created_by` column: a nullable `TEXT` FK to `users(uid)` (`ON DELETE SET NULL`), auto-populated from the authenticated caller's token whenever an inspection is saved — never client-submitted. It is currently returned only on the single-inspection detail endpoint (`InspectionResponse.CreatedBy`, `omitempty`), with no join to `users` anywhere, so no username is ever surfaced. Two other read paths — `ListInspections` (per-equipment history) and `shops/aggregates.GetEquipmentPmcsHistory` (cross-shop history) — don't return it at all.

## Decisions

- **No new column.** `created_by` already is "who performed this PMCS" in every way that matters: it's set once, from the authenticated caller, at save time. Rather than adding a parallel `performed_by` field, this rename repurposes the existing column end-to-end (DB column, FK constraint, generated Go model, JSON field): `created_by` → `performed_by`.
- **Sticky-on-conflict behavior is unchanged.** `ensureInspection`'s `ON CONFLICT DO UPDATE` (repository_impl.go:329-370) already excludes this column from its `SET` list, so re-saving an existing inspection (e.g. a later fault edit by a different shop member, or a `performed_date` correction) never reassigns who performed it — it stays pinned to whoever first created the row. This is preserved exactly.
- **`performed_by` is still never client-submitted.** It continues to be derived server-side from the authenticated caller, consistent with the existing save flow. This design only concerns exposing it (with a username) everywhere inspection data is read, not changing who's allowed to set it.
- **Breaking API change, handled as a clean rename.** `created_by` is documented today in `docs/api/pmcs_sbs_inspections_mobile.md` and `docs/api/pmcs_sbs_inspection_history_mobile_changes.md`. Both are updated in the same change to `performed_by` / `performed_by_username`, coordinated with the mobile team rather than kept as a deprecated duplicate field.

## Database

New migration `007_rename_pmcs_sbs_inspections_created_by_to_performed_by.sql`:

```sql
ALTER TABLE pmcs_sbs_inspections RENAME COLUMN created_by TO performed_by;
ALTER TABLE pmcs_sbs_inspections RENAME CONSTRAINT fk_pmcs_sbs_inspections_created_by TO fk_pmcs_sbs_inspections_performed_by;
```

Rollback (`007_rollback_rename_pmcs_sbs_inspections_created_by_to_performed_by.sql`):

```sql
ALTER TABLE pmcs_sbs_inspections RENAME CONSTRAINT fk_pmcs_sbs_inspections_performed_by TO fk_pmcs_sbs_inspections_created_by;
ALTER TABLE pmcs_sbs_inspections RENAME COLUMN performed_by TO created_by;
```

Pure rename — no data migration, existing values carry over unchanged. No new index: the join added below is always `users.uid = pmcs_sbs_inspections.performed_by`, driven against `users`' primary key, which is already indexed.

After the migration runs, go-jet models are regenerated (`jet gen` against the updated schema) so `.gen/miltech_ng/public/model/pmcs_sbs_inspections.go` and `.gen/miltech_ng/public/table/pmcs_sbs_inspections.go` reflect `PerformedBy *string` in place of `CreatedBy *string`.

## Backend Changes

### Query pattern

All three read paths add a `LEFT JOIN` to `users` and select `Users.Username.AS("performed_by_username")`, following the exact precedent already established in `api/shops/lists/repository_impl.go:38-56`:

```go
stmt := SELECT(
    PmcsSbsInspections.AllColumns,
    Users.Username.AS("performed_by_username"),
).FROM(
    PmcsSbsInspections.LEFT_JOIN(Users, Users.UID.EQ(PmcsSbsInspections.PerformedBy)),
).WHERE(...)

var row struct {
    model.PmcsSbsInspections
    PerformedByUsername *string `sql:"performed_by_username"`
}
```

`LEFT JOIN`, not `INNER JOIN`, is required: if the user account referenced by `performed_by` is later deleted, the FK's `ON DELETE SET NULL` nulls the column, and the inspection must still be returned (`performed_by: null, performed_by_username: null`) rather than disappearing from history.

### 1. `pmcs_sbs_progress.GetInspection` (single inspection detail)

`repository_impl.go:33-64` — add the join to the existing single-row query. `InspectionResponse` (types.go:57-66): rename `CreatedBy *string` → `PerformedBy *string`, add `PerformedByUsername *string \`json:"performed_by_username,omitempty"\``. `mapInspection` (repository_impl.go:334-349) updated accordingly.

### 2. `pmcs_sbs_progress.ListInspections` (per-equipment inspection history)

`repository_impl.go:66-127` — add the join to the existing inspections query (the one that runs before the separate batched fault-count query; unaffected by the join since it's many-to-one). `InspectionSummary` (repository.go:29-35) gains `PerformedBy *string` + `PerformedByUsername *string` (internal type, no JSON tags). `InspectionSummaryResponse` (types.go:68-74) gains `PerformedBy *string \`json:"performed_by,omitempty"\`` + `PerformedByUsername *string \`json:"performed_by_username,omitempty"\`` — same `omitempty` convention as `InspectionResponse`, since the two fields are always both-nil or both-set together. `ServiceImpl.ListInspections` (service_impl.go:94-103) maps the two new fields through.

### 3. `shops/aggregates.GetEquipmentPmcsHistory` (cross-shop equipment + history aggregate)

`repository_impl.go:1155-1260` — add the join to the existing inspections query (the one that runs after the equipment query and before the fault-count batching). `response.PmcsHistorySummary` (`api/response/user_shops_response.go:229-235`) gains `PerformedBy *string \`json:"performed_by,omitempty"\`` + `PerformedByUsername *string \`json:"performed_by_username,omitempty"\`` — same `omitempty` convention as the other two response types; the merge loop that builds `historyByEquipmentID` (lines 1238-1247) passes them through.

### 4. Save path (`EnsureInspection`) — optimized, not blindly joined

`repo.EnsureInspection` (repository_impl.go:26-31) performs an `INSERT ... ON CONFLICT ... RETURNING`, which cannot join. Rather than adding a second `SELECT ... LEFT JOIN users` after every save, the service layer exploits a fact already available for free: `bootstrap.User.Username` carries the caller's own username from the auth token, with no DB call.

In `ServiceImpl.EnsureInspection`, after the upsert returns:
- If `saved.PerformedBy != nil && *saved.PerformedBy == user.UserID` (the common case — caller is saving their own inspection, whether newly created or re-confirming one they already own) — use `user.Username` directly. Zero extra DB calls.
- Otherwise (the sticky-conflict edge case — the row already existed under a *different* original performer) — issue one lookup query (`SELECT username FROM users WHERE uid = $1`) for that single user id.

Net effect: the normal save flow costs zero additional round trips; only the rare cross-user re-save edge case costs one extra single-row lookup.

`UpsertFault` / `FaultResponse` are untouched — fault responses carry no inspection-level performer field, so no change is needed there.

## Error Handling / Access Control

No changes. `requireVehicleAccess` and `requireInspectionOwnership` already gate every read/write by shop membership (`shop_members.user_id`); adding a column rename and a `LEFT JOIN` doesn't alter authorization in any of the three read paths or the save path.

## API Contract Changes (breaking)

Both mobile-facing docs are updated in this change:
- `docs/api/pmcs_sbs_inspections_mobile.md` — the `created_by` row in its field table becomes two rows: `performed_by` and `performed_by_username`.
- `docs/api/pmcs_sbs_inspection_history_mobile_changes.md` — same field-table update, plus its two JSON examples (lines 78, 101) updated to show `performed_by`/`performed_by_username` in place of `created_by`.

`InspectionSummaryResponse` and `PmcsHistorySummary` previously had no performer field at all — this is additive for those two response shapes, not a rename.

## Testing

- **Migration:** apply + rollback round-trip against a scratch DB.
- **`pmcs_sbs_progress`:** update `service_impl_test.go`'s existing `CreatedBy` assertions (lines 149, 161-162) to `PerformedBy`. Add a repository-level real-Postgres case asserting `performed_by_username` resolves correctly for `GetInspection` and `ListInspections`, and a case where the referenced user account has been deleted (`performed_by`/`performed_by_username` both nil, row still present, not dropped by the join).
- **`shops/aggregates`:** extend `shops_equipment_pmcs_history_test.go` with a case asserting `performed_by`/`performed_by_username` per history entry, including the zero-inspection-equipment case (unaffected, still `historical_pmcs: []`).
- **Save-path optimization:** unit test (mocked repository) confirming zero extra DB calls when the caller matches the sticky owner, and correct username resolution via the fallback lookup when they don't.

## Out of Scope

- No change to who is *allowed* to set `performed_by` — it remains always the authenticated caller at save time (or the original sticky owner on conflict), never client-submitted. A distinct "record who physically performed the PMCS, independent of who's syncing the data" concept was considered and explicitly rejected in favor of reusing the existing caller-derived field.
- No historical backfill logic needed — the rename preserves existing `created_by` values as-is.
- No change to `pmcs_sbs_faults` — faults have no independent performer concept; they inherit it from their parent inspection.
- No change to the "latest writer wins" alternative for conflict handling — sticky/first-writer semantics were explicitly chosen and match existing behavior.
