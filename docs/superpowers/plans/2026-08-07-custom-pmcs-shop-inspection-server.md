# Custom PMCS Shop Inspection Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make custom-checklist PMCS inspections first-class authenticated Shop inspection records without breaking guide-backed clients.

**Architecture:** Extend `pmcs_sbs_inspections` with a database-enforced guide/custom source union and retain the existing routes, vehicle membership authorization, transactional fault upsert, notes, comments, and history aggregation. Custom provenance is immutable snapshot metadata, not a foreign key to mutable User-PMCS content.

**Tech Stack:** Go, Gin, PostgreSQL, go-jet v2, `database/sql`, `github.com/google/uuid`, `github.com/clipperhouse/uax29/v2/graphemes`, Testify.

## Global Constraints

- Read `docs/superpowers/specs/2026-08-07-custom-pmcs-shop-inspection-server-design.md` before starting.
- Existing guide requests that provide `guide_manual` and omit `source_type` remain valid and decode as `guide`.
- Every response explicitly returns `source_type`; inapplicable source fields use `omitempty`, never empty sentinels.
- A `pmcs_id` is immutable across equipment ID and the complete source tuple; mismatch returns the existing inspection conflict response.
- Vehicle membership authorization and the `pmcs_id` plus `equipment_id` boundary remain mandatory on every inspection/fault/comment operation.
- Custom checklist name and fault data are Shop-visible; authored checklist trees are not copied into this schema.
- Checklist name and section title use the User-PMCS short-field limits: 200 grapheme clusters and 8 KiB UTF-8.
- Keep transactions short. Validation and authorization execute before the inspection/fault transaction.
- Never log authored text, names, notes, fault text, corrective action, request bodies, credentials, or Firebase claims.
- Rehearse forward -> rollback -> forward only on `miltech_ng_test` through `TEST_DATABASE_URL`. Apply forward once to non-production `miltech_ng`; never roll it back for rehearsal and never target production.
- Regenerate Jet from migrated `miltech_ng` only after both non-production forward migrations succeed. Never hand-edit `.gen`.
- Use one fresh implementer and one independent reviewer per task. Record task commit, tests, findings, and worktree state in a progress ledger.
- No deployment, production migration, push, merge, or mobile implementation is authorized by this plan.

---

### Task 1: Source-Union Migration and Jet Regeneration

**Files:**
- Create: `migrations/011_extend_pmcs_inspection_sources.sql`
- Create: `migrations/011_rollback_extend_pmcs_inspection_sources.sql`
- Regenerate, do not hand-edit: `.gen/miltech_ng/public/model/pmcs_sbs_inspections.go`
- Regenerate, do not hand-edit: `.gen/miltech_ng/public/model/pmcs_sbs_faults.go`
- Regenerate, do not hand-edit: `.gen/miltech_ng/public/table/pmcs_sbs_inspections.go`
- Regenerate, do not hand-edit: `.gen/miltech_ng/public/table/pmcs_sbs_faults.go`
- Test: `tests/pmcs_sbs_progress/repository_test.go`

**Interfaces:**
- Produces: `PmcsSbsInspections.SourceType string`, nullable `GuideManual`, nullable custom provenance fields, and `PmcsSbsFaults.SectionTitle *string` in generated Jet models.
- Preserves: existing inspection and fault primary/foreign keys plus `idx_pmcs_sbs_inspections_equipment_performed`.

- [ ] **Step 1: Add a failing schema contract test**

Add a repository integration test that executes direct inserts for one guide row, one valid custom row, and invalid mixed/incomplete rows. The custom insert shape must be:

```sql
INSERT INTO pmcs_sbs_inspections
  (id, equipment_id, source_type, guide_manual,
   custom_checklist_id, custom_revision_id,
   custom_revision_number, custom_checklist_name,
   performed_date, performed_by)
VALUES ($1, $2, 'custom', NULL, $3, $4, 0, 'Device Checklist', $5, $6)
```

Run: `rtk go test ./tests/pmcs_sbs_progress -run TestInspectionSourceConstraint -count=1`

Expected: FAIL because `source_type` and custom columns do not exist.

- [ ] **Step 2: Write the forward and guarded rollback migrations**

The forward migration must backfill guide rows before installing the union:

```sql
BEGIN;
ALTER TABLE pmcs_sbs_inspections
  ADD COLUMN source_type TEXT NOT NULL DEFAULT 'guide',
  ADD COLUMN custom_checklist_id UUID,
  ADD COLUMN custom_revision_id UUID,
  ADD COLUMN custom_revision_number INTEGER,
  ADD COLUMN custom_checklist_name TEXT;
ALTER TABLE pmcs_sbs_inspections ALTER COLUMN guide_manual DROP NOT NULL;
ALTER TABLE pmcs_sbs_faults ADD COLUMN section_title TEXT;
ALTER TABLE pmcs_sbs_inspections
  DROP CONSTRAINT pmcs_sbs_inspections_nonblank_check,
  DROP CONSTRAINT pmcs_sbs_inspections_guide_manual_format_check;
ALTER TABLE pmcs_sbs_inspections
  ADD CONSTRAINT pmcs_sbs_inspections_source_type_check
    CHECK (source_type IN ('guide', 'custom')),
  ADD CONSTRAINT pmcs_sbs_inspections_source_shape_check CHECK (
    (source_type = 'guide' AND guide_manual IS NOT NULL
      AND guide_manual = btrim(guide_manual)
      AND guide_manual LIKE 'pmcs_sbs/%'
      AND right(guide_manual, 5) = '.json'
      AND custom_checklist_id IS NULL AND custom_revision_id IS NULL
      AND custom_revision_number IS NULL AND custom_checklist_name IS NULL)
    OR
    (source_type = 'custom' AND guide_manual IS NULL
      AND custom_checklist_id IS NOT NULL AND custom_revision_id IS NOT NULL
      AND custom_revision_number >= 0
      AND custom_checklist_name = btrim(custom_checklist_name)
      AND btrim(custom_checklist_name) <> '')
  );
ALTER TABLE pmcs_sbs_inspections ALTER COLUMN source_type DROP DEFAULT;
COMMIT;
```

The rollback begins with a `DO` block that raises if any custom row exists. It then removes new constraints/columns and restores the original guide constraints and `guide_manual NOT NULL`.

- [ ] **Step 3: Verify database targets before applying anything**

Run:

```bash
test "$(psql "$TEST_DATABASE_URL" -Atc 'select current_database()')" = "miltech_ng_test"
```

Expected: exit 0 and no output. Stop immediately on any other database name.

- [ ] **Step 4: Rehearse test migration and verify preservation**

Run forward, focused tests, rollback, and forward again against
`TEST_DATABASE_URL`. Verify an existing guide fixture retains its UUID,
manual, performed date, note, faults, and comments. Expected: every SQL command
and focused test succeeds; rollback refusal is separately proven after a
temporary custom row is inserted inside a rolled-back test transaction.

- [ ] **Step 5: Apply forward once to development and regenerate Jet**

Verify development reports `miltech_ng`, apply the forward migration once,
then run:

```bash
jet -dsn="$JET_DATABASE_URL" -schema=public -path=./.gen
gofmt -w .gen/miltech_ng/public/model .gen/miltech_ng/public/table
```

Expected: generated inspection/fault models match the interfaces above. Audit
the full `.gen` diff and stop if unrelated schema drift appears.

- [ ] **Step 6: Run and commit the migration task**

Run: `rtk go test ./tests/pmcs_sbs_progress -run 'TestInspectionSourceConstraint|TestRepositoryVehicleDeleteCascades' -count=1`

Expected: PASS.

Commit: `feat(pmcs): add custom inspection provenance schema`

---

### Task 2: Source DTOs and Validation

**Files:**
- Modify: `api/pmcs_sbs_progress/types.go`
- Modify: `api/pmcs_sbs_progress/errors.go`
- Modify: `api/pmcs_sbs_progress/service_impl.go`
- Modify: `api/pmcs_sbs_progress/service_impl_test.go`

**Interfaces:**
- Produces: `InspectionSourceRequest`, source fields embedded in `InspectionRequest` and `FaultRequest`, and source fields on `InspectionResponse` and `InspectionSummaryResponse`.
- Produces: `normalizeInspectionSource(InspectionSourceRequest) (ValidatedInspectionSource, error)`.

- [ ] **Step 1: Write table-driven failing validation tests**

Cover legacy guide omission, explicit guide, custom revision zero, custom
published revision, mixed fields, missing UUID/name/number, zero UUID,
negative number, invalid source type, 201 graphemes, and 8193 UTF-8 bytes.

Run: `rtk go test ./api/pmcs_sbs_progress -run 'TestValidateInspectionSource|TestUpsertFaultAcceptsCustomSource' -count=1`

Expected: FAIL because source DTOs and validation do not exist.

- [ ] **Step 2: Add request and response source types**

Use a pointer for revision number so omitted differs from valid zero:

```go
type InspectionSourceRequest struct {
    SourceType           string `json:"source_type"`
    GuideManual          string `json:"guide_manual"`
    CustomChecklistID    string `json:"custom_checklist_id"`
    CustomRevisionID     string `json:"custom_revision_id"`
    CustomRevisionNumber *int32 `json:"custom_revision_number"`
    CustomChecklistName  string `json:"custom_checklist_name"`
}

type ValidatedInspectionSource struct {
    SourceType           string
    GuideManual          *string
    CustomChecklistID    *uuid.UUID
    CustomRevisionID     *uuid.UUID
    CustomRevisionNumber *int32
    CustomChecklistName  *string
}
```

Add `SectionTitle string` to `FaultRequest` and `SectionTitle *string
omitempty` to `FaultResponse`.

- [ ] **Step 3: Implement fail-closed normalization**

Infer legacy guide only when `source_type` is blank, `guide_manual` is
nonblank, and every custom field is absent. Validate custom UUIDs with
`uuid.Parse`, reject `uuid.Nil`, and count short text with UAX29 grapheme
segmentation plus byte length. Return existing invalid-request errors without
including field contents.

- [ ] **Step 4: Map source values into Jet models and responses**

Update inspection validation and mapping so source pointers remain nullable
and response JSON omits inapplicable fields. Preserve all existing performer,
note, fault, and comment behavior.

- [ ] **Step 5: Verify and commit**

Run: `rtk go test ./api/pmcs_sbs_progress -count=1`

Expected: all package tests pass, including legacy route fixtures.

Commit: `feat(pmcs): validate guide and custom inspection sources`

---

### Task 3: Transactional Custom Fault Persistence

**Files:**
- Modify: `api/pmcs_sbs_progress/repository.go`
- Modify: `api/pmcs_sbs_progress/repository_impl.go`
- Modify: `tests/pmcs_sbs_progress/helpers_test.go`
- Modify: `tests/pmcs_sbs_progress/repository_test.go`

**Interfaces:**
- Consumes: generated nullable source fields from Task 1 and validated models from Task 2.
- Produces: idempotent `EnsureInspection` and `UpsertFault` for either source while preserving existing method signatures.

- [ ] **Step 1: Write failing repository tests**

Test implicit custom creation, explicit clean creation, same-source retry,
source mutation conflict, equipment mutation conflict, nullable guide scan,
section-title persistence, member access, nonmember not-found, and cascades.

Run: `rtk go test ./tests/pmcs_sbs_progress -run 'TestRepository(Custom|EnsureInspection|UpsertFault)' -count=1`

Expected: FAIL against the guide-only repository SQL.

- [ ] **Step 2: Extend `ensureInspection`**

Insert all source columns and make the conflict predicate null-safe:

```sql
WHERE pmcs_sbs_inspections.equipment_id = EXCLUDED.equipment_id
  AND pmcs_sbs_inspections.source_type = EXCLUDED.source_type
  AND pmcs_sbs_inspections.guide_manual IS NOT DISTINCT FROM EXCLUDED.guide_manual
  AND pmcs_sbs_inspections.custom_checklist_id IS NOT DISTINCT FROM EXCLUDED.custom_checklist_id
  AND pmcs_sbs_inspections.custom_revision_id IS NOT DISTINCT FROM EXCLUDED.custom_revision_id
  AND pmcs_sbs_inspections.custom_revision_number IS NOT DISTINCT FROM EXCLUDED.custom_revision_number
  AND pmcs_sbs_inspections.custom_checklist_name IS NOT DISTINCT FROM EXCLUDED.custom_checklist_name
```

Keep only performed date, notes, and updated timestamp mutable.

- [ ] **Step 3: Extend fault persistence and reads**

Insert/update/return `section_title`. Keep the transaction limited to ensure
inspection, upsert fault, and commit. Preserve `requireVehicleAccess` before
opening it.

- [ ] **Step 4: Update list/detail scans**

Return nullable guide and custom fields without coalescing them. Preserve
ordering and count queries.

- [ ] **Step 5: Verify and commit**

Run:

```bash
rtk go test ./tests/pmcs_sbs_progress -count=1
rtk go test ./api/pmcs_sbs_progress -count=1
```

Expected: both packages pass.

Commit: `feat(pmcs): persist custom shop inspection faults`

---

### Task 4: Route Compatibility and Mobile Contract Handoff

**Files:**
- Modify: `api/pmcs_sbs_progress/route.go`
- Modify: `api/pmcs_sbs_progress/route_test.go`
- Modify: `docs/api/pmcs_sbs_inspections_mobile.md`
- Create: `docs/client/2026-08-07-custom-pmcs-shop-inspection-mobile-contract.md`

**Interfaces:**
- Produces: backward-compatible guide JSON and documented custom request/response fixtures for the mobile plan.

- [ ] **Step 1: Add failing route contract tests**

Assert a legacy guide request reaches the service as guide-shaped, a custom
fault carries all provenance and section title, response JSON omits
`guide_manual` for custom, and source validation failures retain existing
status/envelope conventions.

Run: `rtk go test ./api/pmcs_sbs_progress -run 'TestRoute.*Custom|TestUpsertFaultSuccess' -count=1`

Expected: FAIL until handler DTOs and fixtures use the new contract.

- [ ] **Step 2: Update handlers without adding routes**

Keep strict JSON decoding and existing route registration. Do not add a
parallel `/custom-pmcs` endpoint. Ensure response source fields come only from
validated/repository values.

- [ ] **Step 3: Write both documentation updates**

Document exact guide/custom JSON, nullable/omitted fields, error codes,
authorization, lazy materialization, idempotency, limits, reset batching,
clean completion, notes/comments, and the no-outbox mobile expectation. The
standalone client handoff must explicitly state that cursors/ETags from
User-PMCS content sync are unrelated to Shop inspection mutations.

- [ ] **Step 4: Verify and commit**

Run: `rtk go test ./api/pmcs_sbs_progress -count=1`

Expected: PASS.

Commit: `docs(pmcs): publish custom inspection mobile contract`

---

### Task 5: Shop Aggregate History Integration

**Files:**
- Modify: `api/response/user_shops_response.go`
- Modify: `api/shops/aggregates/repository_impl.go`
- Modify: `api/shops/aggregates/service_impl_test.go`
- Modify: `tests/shops/shops_equipment_pmcs_history_test.go`

**Interfaces:**
- Consumes: source-discriminated inspection rows from Tasks 1–3.
- Produces: `historical_pmcs[]` summaries with explicit source and custom provenance.

- [ ] **Step 1: Write failing aggregate tests**

Insert one guide and one custom inspection for the same vehicle. Assert both
are ordered by performed date, carry the correct mutually exclusive source
fields, and have independent fault/comment counts and performer display.

Run: `rtk go test ./tests/shops -run 'Test.*EquipmentPmcsHistory.*Custom' -count=1`

Expected: FAIL because `PmcsHistorySummary` is guide-only.

- [ ] **Step 2: Extend the response and mapping**

Add source fields with `omitempty`. Map nullable Jet values directly; never
derive custom identity from a guide string. Leave the existing batched count
queries and equipment query count unchanged.

- [ ] **Step 3: Guard query performance**

Assert the repository still performs one inspection query plus batched fault
and comment count queries, not one query per inspection. Do not add an index
without representative `EXPLAIN (ANALYZE, BUFFERS)` evidence.

- [ ] **Step 4: Verify and commit**

Run:

```bash
rtk go test ./api/shops/aggregates -count=1
rtk go test ./tests/shops -run 'Test(GetEquipmentPmcsHistory|EquipmentPmcsHistory)' -count=1
```

Expected: PASS.

Commit: `feat(shops): include custom PMCS inspection history`

---

### Task 6: Server Lifecycle Review and Delivery Gate

**Files:**
- Modify: `docs/project_notes/decisions.md`
- Create: `.superpowers/sdd/custom-pmcs-shop-inspection-server-progress.md`
- Modify only if findings require it: files from Tasks 1–5

**Interfaces:**
- Produces: reviewed server HEAD and exact contract commit for mobile implementation.

- [ ] **Step 1: Record the architectural decision**

Add an ADR covering the source union, immutable snapshot provenance, lack of
User-PMCS foreign keys, backward-compatible guide inference, and Shop-wide
visibility boundary.

- [ ] **Step 2: Run focused verification**

```bash
rtk go test ./api/pmcs_sbs_progress -count=1
rtk go test ./tests/pmcs_sbs_progress -count=1
rtk go test ./api/shops/aggregates -count=1
rtk go test ./tests/shops -run 'Test(GetEquipmentPmcsHistory|EquipmentPmcsHistory)' -count=1
```

Expected: all pass.

- [ ] **Step 3: Run whole-repository and race verification**

```bash
rtk go test ./... -count=1
rtk go test -race ./api/pmcs_sbs_progress ./api/shops/aggregates -count=1
```

Report exact passes and any accepted pre-existing failures separately.

- [ ] **Step 4: Repeat the migration safety gate**

Prove test forward/rollback/forward, development forward-only, current schema
constraints, and generated Jet consistency. Confirm neither connection names
production.

- [ ] **Step 5: Perform independent whole-branch review**

Audit authorization, route registration, null handling, conflict predicates,
transaction boundaries, logs, history response compatibility, rollback data
safety, docs, and test depth. Resolve every finding before proceeding.

- [ ] **Step 6: Commit the reviewed handoff**

Commit: `docs(pmcs): finalize custom inspection server handoff`

Record final HEAD, upstream status, worktree state, migration evidence, exact
test totals, and the mobile contract path. Mobile Task 5 must pin this server
HEAD before consuming the contract.
