# PMCS SBS Performed-By Tracking Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename `pmcs_sbs_inspections.created_by` to `performed_by` end-to-end (DB, generated models, Go types, JSON) and surface a joined `performed_by_username` alongside it in every inspection read path: single-inspection detail, per-equipment inspection list, and the cross-shop equipment+history aggregate.

**Architecture:** No new column — the existing caller-derived, FK-backed `created_by` column already is "who performed this PMCS." Three read paths each gain a `LEFT JOIN users` (against `users.uid`, the primary key) to resolve a username alongside the id. The save path (`EnsureInspection`) avoids the extra join entirely in the common case by reusing the caller's own username already present on the auth token, falling back to a single-row lookup only when the sticky first-writer differs from the current caller.

**Tech Stack:** Go, Gin, go-jet (Postgres query builder), `database/sql` + `lib/pq`, testify, real-Postgres integration tests (no mocked DB).

## Global Constraints

- Test DB connection: `postgres://postgres:potato123@192.168.20.70/miltech_ng_test?sslmode=disable` (from `tests/pmcs_sbs_progress/main_test.go` and `tests/shops/main_test.go` — both `_test` packages open this DSN in `TestMain`). All integration tests in this plan require this DB reachable.
- `performed_by` is never client-submitted — it is always derived server-side from the authenticated caller (`bootstrap.User.UserID`), consistent with the current `created_by` behavior. No task in this plan adds a request field for it.
- Sticky-on-conflict semantics are preserved exactly: `ensureInspection`'s `ON CONFLICT DO UPDATE` must continue to exclude this column from its `SET` list.
- This is a breaking API rename (`created_by` → `performed_by`, JSON field), coordinated by updating the two mobile-facing docs (`docs/api/pmcs_sbs_inspections_mobile.md`, `docs/api/pmcs_sbs_inspection_history_mobile_changes.md`) in the same tasks that change the corresponding response shape — not deferred.
- Follow existing code conventions exactly: go-jet dot-imports (`. "miltechserver/.gen/miltech_ng/public/table"`), `LEFT_JOIN(Users, Users.UID.EQ(X.PerformedBy))` + `Users.Username.AS("performed_by_username")` scanned into a destination struct that embeds the model type (precedent: `api/shops/lists/repository_impl.go:38-56`).
- Conventional commit messages per task; never mention AI tooling in commit text.

---

### Task 1: Migration — rename `created_by` to `performed_by`

**Files:**
- Create: `migrations/007_rename_pmcs_sbs_inspections_created_by_to_performed_by.sql`
- Create: `migrations/007_rollback_rename_pmcs_sbs_inspections_created_by_to_performed_by.sql`

**Interfaces:**
- Produces: DB column `pmcs_sbs_inspections.performed_by` (was `created_by`), FK constraint `fk_pmcs_sbs_inspections_performed_by` (was `fk_pmcs_sbs_inspections_created_by`). Task 2 depends on this being applied to the test DB before regenerating models.

- [ ] **Step 1: Write the migration**

```sql
-- PMCS SBS Performed-By Rename
-- Migration: 007_rename_pmcs_sbs_inspections_created_by_to_performed_by.sql
--
-- Repurposes the existing caller-derived created_by column as performed_by:
-- see docs/superpowers/specs/2026-07-18-pmcs-sbs-performed-by-design.md.
-- Pure rename, no data change — existing values carry over unchanged.

ALTER TABLE pmcs_sbs_inspections RENAME COLUMN created_by TO performed_by;
ALTER TABLE pmcs_sbs_inspections RENAME CONSTRAINT fk_pmcs_sbs_inspections_created_by TO fk_pmcs_sbs_inspections_performed_by;
```

- [ ] **Step 2: Write the rollback**

```sql
-- Rollback: 007_rollback_rename_pmcs_sbs_inspections_created_by_to_performed_by.sql

ALTER TABLE pmcs_sbs_inspections RENAME CONSTRAINT fk_pmcs_sbs_inspections_performed_by TO fk_pmcs_sbs_inspections_created_by;
ALTER TABLE pmcs_sbs_inspections RENAME COLUMN performed_by TO created_by;
```

- [ ] **Step 3: Apply the migration to the test DB**

Run:
```bash
PGPASSWORD=potato123 psql -h 192.168.20.70 -U postgres -d miltech_ng_test -f migrations/007_rename_pmcs_sbs_inspections_created_by_to_performed_by.sql
```
Expected: `ALTER TABLE` printed twice, no errors.

- [ ] **Step 4: Verify the rename**

Run:
```bash
PGPASSWORD=potato123 psql -h 192.168.20.70 -U postgres -d miltech_ng_test -c "\d pmcs_sbs_inspections"
```
Expected: column list shows `performed_by` (not `created_by`), foreign-key constraint list shows `fk_pmcs_sbs_inspections_performed_by`.

- [ ] **Step 5: Verify the rollback round-trips cleanly**

Run:
```bash
PGPASSWORD=potato123 psql -h 192.168.20.70 -U postgres -d miltech_ng_test -f migrations/007_rollback_rename_pmcs_sbs_inspections_created_by_to_performed_by.sql
PGPASSWORD=potato123 psql -h 192.168.20.70 -U postgres -d miltech_ng_test -c "\d pmcs_sbs_inspections" | grep created_by
```
Expected: rollback succeeds, `created_by` is back. Then re-apply the forward migration (Step 3 command again) so the test DB ends this task in the renamed state — every later task assumes `performed_by` exists.

- [ ] **Step 6: Commit**

```bash
git add migrations/007_rename_pmcs_sbs_inspections_created_by_to_performed_by.sql migrations/007_rollback_rename_pmcs_sbs_inspections_created_by_to_performed_by.sql
git commit -m "feat(pmcs-sbs): rename inspections.created_by to performed_by"
```

---

### Task 2: Regenerate go-jet models for `pmcs_sbs_inspections`

**Files:**
- Modify: `.gen/miltech_ng/public/model/pmcs_sbs_inspections.go`
- Modify: `.gen/miltech_ng/public/table/pmcs_sbs_inspections.go`

**Interfaces:**
- Consumes: Task 1's renamed column (must already be applied to the DB used for generation).
- Produces: `model.PmcsSbsInspections.PerformedBy *string`, `table.PmcsSbsInspections.PerformedBy postgres.ColumnString`. Every later task's Go code references these exact names.

- [ ] **Step 1: Regenerate via go-jet CLI, or hand-edit if the CLI isn't available**

Preferred (if the `jet` binary is installed and `go-jet/jet/v2/cmd/jet` is on `PATH`):
```bash
jet -dsn="postgres://postgres:potato123@192.168.20.70/miltech_ng_test?sslmode=disable" -schema=public -path=./.gen
```
This regenerates all `.gen/miltech_ng/public/**` files against the live (already-migrated) schema — confirm with `git diff --stat .gen/` that only `pmcs_sbs_inspections.go` (model and table) changed, and only in the `CreatedBy`→`PerformedBy` rename. If anything else in `.gen/` changed unexpectedly, stop and investigate before continuing (it means the schema has drifted from what the rest of the codebase expects).

If `jet` isn't available in this environment, hand-edit the two files to the exact output that command would produce:

`.gen/miltech_ng/public/model/pmcs_sbs_inspections.go` — replace the `CreatedBy` field:
```go
type PmcsSbsInspections struct {
	ID            uuid.UUID `sql:"primary_key" json:"id"`
	EquipmentID   string    `json:"equipment_id"`
	GuideManual   string    `json:"guide_manual"`
	PerformedDate time.Time `json:"performed_date"`
	PerformedBy   *string   `json:"performed_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
```

`.gen/miltech_ng/public/table/pmcs_sbs_inspections.go` — replace every `CreatedBy`/`created_by` occurrence:
```go
type pmcsSbsInspectionsTable struct {
	postgres.Table

	// Columns
	ID            postgres.ColumnString
	EquipmentID   postgres.ColumnString
	GuideManual   postgres.ColumnString
	PerformedDate postgres.ColumnTimestampz
	PerformedBy   postgres.ColumnString
	CreatedAt     postgres.ColumnTimestampz
	UpdatedAt     postgres.ColumnTimestampz

	AllColumns     postgres.ColumnList
	MutableColumns postgres.ColumnList
	DefaultColumns postgres.ColumnList
}
```
and in `newPmcsSbsInspectionsTableImpl`:
```go
func newPmcsSbsInspectionsTableImpl(schemaName, tableName, alias string) pmcsSbsInspectionsTable {
	var (
		IDColumn            = postgres.StringColumn("id")
		EquipmentIDColumn   = postgres.StringColumn("equipment_id")
		GuideManualColumn   = postgres.StringColumn("guide_manual")
		PerformedDateColumn = postgres.TimestampzColumn("performed_date")
		PerformedByColumn   = postgres.StringColumn("performed_by")
		CreatedAtColumn     = postgres.TimestampzColumn("created_at")
		UpdatedAtColumn     = postgres.TimestampzColumn("updated_at")
		allColumns          = postgres.ColumnList{IDColumn, EquipmentIDColumn, GuideManualColumn, PerformedDateColumn, PerformedByColumn, CreatedAtColumn, UpdatedAtColumn}
		mutableColumns      = postgres.ColumnList{EquipmentIDColumn, GuideManualColumn, PerformedDateColumn, PerformedByColumn, CreatedAtColumn, UpdatedAtColumn}
		defaultColumns      = postgres.ColumnList{CreatedAtColumn, UpdatedAtColumn}
	)

	return pmcsSbsInspectionsTable{
		Table: postgres.NewTable(schemaName, tableName, alias, allColumns...),

		//Columns
		ID:            IDColumn,
		EquipmentID:   EquipmentIDColumn,
		GuideManual:   GuideManualColumn,
		PerformedDate: PerformedDateColumn,
		PerformedBy:   PerformedByColumn,
		CreatedAt:     CreatedAtColumn,
		UpdatedAt:     UpdatedAtColumn,

		AllColumns:     allColumns,
		MutableColumns: mutableColumns,
		DefaultColumns: defaultColumns,
	}
}
```
Leave every other line in both files untouched (header comments, `AS`/`FromSchema`/`WithPrefix`/`WithSuffix` methods, `newPmcsSbsInspectionsTable`).

- [ ] **Step 2: Confirm the codebase still compiles (rename ripples through in Task 3, so full green isn't expected yet)**

Run:
```bash
go build ./.gen/... 2>&1 | head -20
```
Expected: no errors (this only builds the generated package in isolation; `miltechserver/api/...` won't build again until Task 3 renames its own references — that's expected and fine here).

- [ ] **Step 3: Commit**

```bash
git add .gen/miltech_ng/public/model/pmcs_sbs_inspections.go .gen/miltech_ng/public/table/pmcs_sbs_inspections.go
git commit -m "chore(pmcs-sbs): regenerate jet models for performed_by rename"
```

---

### Task 3: Propagate the rename through `pmcs_sbs_progress` (no new behavior)

**Files:**
- Modify: `api/pmcs_sbs_progress/types.go`
- Modify: `api/pmcs_sbs_progress/repository_impl.go`
- Modify: `api/pmcs_sbs_progress/service_impl.go`
- Modify: `tests/pmcs_sbs_progress/helpers_test.go`
- Modify: `tests/pmcs_sbs_progress/repository_test.go`
- Modify: `api/pmcs_sbs_progress/service_impl_test.go`

**Interfaces:**
- Consumes: Task 2's `model.PmcsSbsInspections.PerformedBy *string`.
- Produces: `InspectionResponse.PerformedBy *string` (JSON `performed_by`). No behavior change — this task only fixes every reference to the renamed field so the package compiles and existing tests pass under the new name.

- [ ] **Step 1: Rename in `types.go`**

In `InspectionResponse` (was `CreatedBy *string \`json:"created_by,omitempty"\``):
```go
type InspectionResponse struct {
	ID            uuid.UUID       `json:"id"`
	EquipmentID   string          `json:"equipment_id"`
	GuideManual   string          `json:"guide_manual"`
	PerformedDate time.Time       `json:"performed_date"`
	PerformedBy   *string         `json:"performed_by,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	Faults        []FaultResponse `json:"faults"`
}
```

- [ ] **Step 2: Rename in `repository_impl.go`'s `ensureInspection`**

Replace:
```go
	now := time.Now().UTC()
	var createdByExpr Expression = NULL
	if inspection.CreatedBy != nil {
		createdByExpr = String(*inspection.CreatedBy)
	}

	stmt := PmcsSbsInspections.INSERT(
		PmcsSbsInspections.ID,
		PmcsSbsInspections.EquipmentID,
		PmcsSbsInspections.GuideManual,
		PmcsSbsInspections.PerformedDate,
		PmcsSbsInspections.CreatedBy,
		PmcsSbsInspections.CreatedAt,
		PmcsSbsInspections.UpdatedAt,
	).VALUES(
		UUID(inspection.ID),
		String(inspection.EquipmentID),
		String(inspection.GuideManual),
		TimestampzT(inspection.PerformedDate),
		createdByExpr,
		TimestampzT(now),
		TimestampzT(now),
	).ON_CONFLICT(PmcsSbsInspections.ID).DO_UPDATE(
```

with:
```go
	now := time.Now().UTC()
	var performedByExpr Expression = NULL
	if inspection.PerformedBy != nil {
		performedByExpr = String(*inspection.PerformedBy)
	}

	stmt := PmcsSbsInspections.INSERT(
		PmcsSbsInspections.ID,
		PmcsSbsInspections.EquipmentID,
		PmcsSbsInspections.GuideManual,
		PmcsSbsInspections.PerformedDate,
		PmcsSbsInspections.PerformedBy,
		PmcsSbsInspections.CreatedAt,
		PmcsSbsInspections.UpdatedAt,
	).VALUES(
		UUID(inspection.ID),
		String(inspection.EquipmentID),
		String(inspection.GuideManual),
		TimestampzT(inspection.PerformedDate),
		performedByExpr,
		TimestampzT(now),
		TimestampzT(now),
	).ON_CONFLICT(PmcsSbsInspections.ID).DO_UPDATE(
```
The `SET(...)` clause immediately below (which sets only `PerformedDate` and `UpdatedAt`) is unchanged — this is exactly what preserves sticky-first-writer semantics.

Replace `mapInspection`:
```go
func mapInspection(row model.PmcsSbsInspections, faultRows []model.PmcsSbsFaults) InspectionResponse {
	faults := make([]FaultResponse, 0, len(faultRows))
	for _, faultRow := range faultRows {
		faults = append(faults, mapFault(faultRow))
	}
	return InspectionResponse{
		ID:            row.ID,
		EquipmentID:   row.EquipmentID,
		GuideManual:   row.GuideManual,
		PerformedDate: row.PerformedDate,
		PerformedBy:   row.PerformedBy,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
		Faults:        faults,
	}
}
```
(Task 4 changes this function's signature again to add a username parameter — this step only fixes the field name.)

- [ ] **Step 3: Rename in `service_impl.go`'s `validateInspectionRequest`**

Replace:
```go
	createdBy := strings.TrimSpace(userID)
	return model.PmcsSbsInspections{
		ID:            parsedPmcsID,
		EquipmentID:   trimmedEquipmentID,
		GuideManual:   guideManual,
		PerformedDate: req.PerformedDate.UTC(),
		CreatedBy:     &createdBy,
	}, nil
```
with:
```go
	performedBy := strings.TrimSpace(userID)
	return model.PmcsSbsInspections{
		ID:            parsedPmcsID,
		EquipmentID:   trimmedEquipmentID,
		GuideManual:   guideManual,
		PerformedDate: req.PerformedDate.UTC(),
		PerformedBy:   &performedBy,
	}, nil
```

- [ ] **Step 4: Rename in `tests/pmcs_sbs_progress/helpers_test.go`**

Replace:
```go
func sampleInspection(equipmentID string, createdBy string) model.PmcsSbsInspections {
	createdByCopy := createdBy
	return model.PmcsSbsInspections{
		ID:            uuid.New(),
		EquipmentID:   equipmentID,
		GuideManual:   "pmcs_sbs/hmmwv/file.json",
		PerformedDate: time.Now().UTC(),
		CreatedBy:     &createdByCopy,
	}
}
```
with:
```go
func sampleInspection(equipmentID string, performedBy string) model.PmcsSbsInspections {
	performedByCopy := performedBy
	return model.PmcsSbsInspections{
		ID:            uuid.New(),
		EquipmentID:   equipmentID,
		GuideManual:   "pmcs_sbs/hmmwv/file.json",
		PerformedDate: time.Now().UTC(),
		PerformedBy:   &performedByCopy,
	}
}
```

- [ ] **Step 5: Rename in `tests/pmcs_sbs_progress/repository_test.go`**

In `TestRepositoryEnsureInspectionCreatesRecord`, replace:
```go
	require.NotNil(t, saved.CreatedBy)
	require.Equal(t, user.UserID, *saved.CreatedBy)
```
with:
```go
	require.NotNil(t, saved.PerformedBy)
	require.Equal(t, user.UserID, *saved.PerformedBy)
```

- [ ] **Step 6: Rename in `api/pmcs_sbs_progress/service_impl_test.go`**

In `TestEnsureInspectionMapsResponse`, replace:
```go
func TestEnsureInspectionMapsResponse(t *testing.T) {
	createdBy := "user-1"
	stub := &repoStub{inspection: &model.PmcsSbsInspections{
		ID:            samplePmcsID(),
		EquipmentID:   "vehicle-1",
		GuideManual:   "pmcs_sbs/hmmwv/file.json",
		PerformedDate: time.Now().UTC(),
		CreatedBy:     &createdBy,
	}}
	svc := NewService(stub)

	resp, err := svc.EnsureInspection(requireUser(), "vehicle-1", samplePmcsIDStr, InspectionRequest{
		GuideManual:   "pmcs_sbs/hmmwv/file.json",
		PerformedDate: time.Now(),
	})

	require.NoError(t, err)
	require.Equal(t, samplePmcsID(), resp.ID)
	require.Equal(t, "vehicle-1", stub.capturedInspection.EquipmentID)
	require.NotNil(t, stub.capturedInspection.CreatedBy)
	require.Equal(t, "user-1", *stub.capturedInspection.CreatedBy)
	require.Empty(t, resp.Faults)
}
```
with:
```go
func TestEnsureInspectionMapsResponse(t *testing.T) {
	performedBy := "user-1"
	stub := &repoStub{inspection: &model.PmcsSbsInspections{
		ID:            samplePmcsID(),
		EquipmentID:   "vehicle-1",
		GuideManual:   "pmcs_sbs/hmmwv/file.json",
		PerformedDate: time.Now().UTC(),
		PerformedBy:   &performedBy,
	}}
	svc := NewService(stub)

	resp, err := svc.EnsureInspection(requireUser(), "vehicle-1", samplePmcsIDStr, InspectionRequest{
		GuideManual:   "pmcs_sbs/hmmwv/file.json",
		PerformedDate: time.Now(),
	})

	require.NoError(t, err)
	require.Equal(t, samplePmcsID(), resp.ID)
	require.Equal(t, "vehicle-1", stub.capturedInspection.EquipmentID)
	require.NotNil(t, stub.capturedInspection.PerformedBy)
	require.Equal(t, "user-1", *stub.capturedInspection.PerformedBy)
	require.Empty(t, resp.Faults)
}
```

- [ ] **Step 7: Build and run the full package test suite**

Run:
```bash
go build ./... 2>&1 | head -40
go test ./api/pmcs_sbs_progress/... -v 2>&1 | tail -60
go test ./tests/pmcs_sbs_progress/... -v 2>&1 | tail -80
```
Expected: clean build, all tests pass (this is a pure rename — no assertions about new behavior yet).

- [ ] **Step 8: Commit**

```bash
git add api/pmcs_sbs_progress/types.go api/pmcs_sbs_progress/repository_impl.go api/pmcs_sbs_progress/service_impl.go tests/pmcs_sbs_progress/helpers_test.go tests/pmcs_sbs_progress/repository_test.go api/pmcs_sbs_progress/service_impl_test.go
git commit -m "refactor(pmcs-sbs): rename CreatedBy to PerformedBy throughout pmcs_sbs_progress"
```

---

### Task 4: Single-inspection `performed_by_username` (detail + save)

**Files:**
- Modify: `api/pmcs_sbs_progress/repository.go`
- Modify: `api/pmcs_sbs_progress/repository_impl.go`
- Modify: `api/pmcs_sbs_progress/service_impl.go`
- Modify: `api/pmcs_sbs_progress/types.go`
- Modify: `api/pmcs_sbs_progress/service_impl_test.go`
- Modify: `tests/pmcs_sbs_progress/helpers_test.go`
- Modify: `tests/pmcs_sbs_progress/repository_test.go`
- Modify: `docs/api/pmcs_sbs_inspections_mobile.md`
- Modify: `docs/api/pmcs_sbs_inspection_history_mobile_changes.md`

**Interfaces:**
- Consumes: Task 3's `InspectionResponse.PerformedBy`, `mapInspection(row, faultRows)`.
- Produces: `Repository.GetInspection(...) (*InspectionDetail, []model.PmcsSbsFaults, error)`, `InspectionDetail{model.PmcsSbsInspections; PerformedByUsername *string}`, `Repository.LookupUsername(userID string) (*string, error)`, `mapInspection(row model.PmcsSbsInspections, performedByUsername *string, faultRows []model.PmcsSbsFaults) InspectionResponse`, `InspectionResponse.PerformedByUsername *string`. Tasks 5 and 6 reuse the same join pattern but not these exact types.

- [ ] **Step 1: Write the failing repository test — username resolves on `GetInspection`**

Add to `tests/pmcs_sbs_progress/repository_test.go`:
```go
func TestRepositoryGetInspectionIncludesPerformedByUsername(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-get-username")
	user.Username = "jsmith"
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B17")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	_, err := repo.EnsureInspection(user, inspection)
	require.NoError(t, err)

	detail, _, err := repo.GetInspection(user, vehicleID, inspection.ID)
	require.NoError(t, err)
	require.NotNil(t, detail.PerformedBy)
	require.Equal(t, user.UserID, *detail.PerformedBy)
	require.NotNil(t, detail.PerformedByUsername)
	require.Equal(t, "jsmith", *detail.PerformedByUsername)
}

func TestRepositoryGetInspectionReturnsNilUsernameWhenPerformerDeleted(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	performer := testUser("pmcs-get-deleted-performer")
	viewer := testUser("pmcs-get-deleted-viewer")
	ensureUser(t, testDB, performer)
	ensureUser(t, testDB, viewer)
	shopID := createShopWithMember(t, testDB, performer, "member")
	addShopMember(t, testDB, shopID, viewer, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, performer, "B18")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, performer.UserID)
	_, err := repo.EnsureInspection(performer, inspection)
	require.NoError(t, err)

	_, err = testDB.Exec(`DELETE FROM users WHERE uid=$1`, performer.UserID)
	require.NoError(t, err)

	detail, _, err := repo.GetInspection(viewer, vehicleID, inspection.ID)
	require.NoError(t, err)
	require.Nil(t, detail.PerformedBy)
	require.Nil(t, detail.PerformedByUsername)
}
```
The second test relies on `shop_members.user_id` being `ON DELETE CASCADE` to `users(uid)` (verified against the live schema) and `pmcs_sbs_inspections.performed_by` being `ON DELETE SET NULL` (Task 1) — deleting the performer's user row nulls their `performed_by` but doesn't remove the inspection, and a *different*, still-a-member `viewer` can still fetch it.

Add `addShopMember` to `tests/pmcs_sbs_progress/helpers_test.go` (this package has no equivalent yet — `tests/shops/helpers_test.go` has one but it's a separate Go package):
```go
func addShopMember(t *testing.T, db *sql.DB, shopID string, user *bootstrap.User, role string) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO shop_members (id, shop_id, user_id, role, joined_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		uuid.New().String(),
		shopID,
		user.UserID,
		role,
		time.Now().UTC(),
	)
	require.NoError(t, err)
}
```

- [ ] **Step 2: Run the new tests to verify they fail**

Run:
```bash
go test ./tests/pmcs_sbs_progress/... -run TestRepositoryGetInspection -v
```
Expected: FAIL — compile error, since `GetInspection` doesn't yet return an object with `.PerformedByUsername`, and `addShopMember` doesn't exist as a new symbol until this step's helper is added (it will compile once the helper exists, but the *behavioral* assertions will fail until Step 3's repository change lands). Confirm the failure is about the missing `PerformedByUsername` field, not an unrelated typo.

- [ ] **Step 3: Add `InspectionDetail` and `LookupUsername` to the repository interface**

In `api/pmcs_sbs_progress/repository.go`, change the interface and add the new type:
```go
type Repository interface {
	EnsureInspection(user *bootstrap.User, inspection model.PmcsSbsInspections) (*model.PmcsSbsInspections, error)
	GetInspection(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID) (*InspectionDetail, []model.PmcsSbsFaults, error)
	ListInspections(user *bootstrap.User, equipmentID string, guideManual string, limit int, offset int) ([]InspectionSummary, error)
	DeleteInspection(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID) error
	LookupUsername(userID string) (*string, error)

	UpsertFault(user *bootstrap.User, inspection model.PmcsSbsInspections, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error)
	DeleteFault(user *bootstrap.User, equipmentID string, key FaultKey) error
	DeleteFaults(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID, keys []FaultKey) (int64, error)
}

type InspectionDetail struct {
	model.PmcsSbsInspections
	PerformedByUsername *string
}
```

- [ ] **Step 4: Implement the join in `GetInspection` and add `LookupUsername`**

In `api/pmcs_sbs_progress/repository_impl.go`, replace `GetInspection`:
```go
func (repo *RepositoryImpl) GetInspection(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID) (*InspectionDetail, []model.PmcsSbsFaults, error) {
	if err := repo.requireVehicleAccess(user, equipmentID); err != nil {
		return nil, nil, err
	}

	var row struct {
		model.PmcsSbsInspections
		PerformedByUsername *string `sql:"performed_by_username"`
	}
	stmt := SELECT(
		PmcsSbsInspections.AllColumns,
		Users.Username.AS("performed_by_username"),
	).
		FROM(PmcsSbsInspections.LEFT_JOIN(Users, Users.UID.EQ(PmcsSbsInspections.PerformedBy))).
		WHERE(
			PmcsSbsInspections.ID.EQ(UUID(pmcsID)).
				AND(PmcsSbsInspections.EquipmentID.EQ(String(equipmentID))),
		)

	if err := stmt.Query(repo.db, &row); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
			return nil, nil, ErrInspectionNotFound
		}
		return nil, nil, fmt.Errorf("get pmcs sbs inspection: %w", err)
	}

	var faults []model.PmcsSbsFaults
	faultsStmt := SELECT(PmcsSbsFaults.AllColumns).
		FROM(PmcsSbsFaults).
		WHERE(PmcsSbsFaults.PmcsID.EQ(UUID(pmcsID))).
		ORDER_BY(PmcsSbsFaults.SectionID.ASC(), PmcsSbsFaults.ItemIndex.ASC())

	if err := faultsStmt.Query(repo.db, &faults); err != nil {
		return nil, nil, fmt.Errorf("list pmcs sbs inspection faults: %w", err)
	}

	return &InspectionDetail{PmcsSbsInspections: row.PmcsSbsInspections, PerformedByUsername: row.PerformedByUsername}, faults, nil
}
```

Add `LookupUsername` (place it near `requireInspectionOwnership`, before `ensureInspection`):
```go
func (repo *RepositoryImpl) LookupUsername(userID string) (*string, error) {
	var row struct {
		Username *string `sql:"username"`
	}
	stmt := SELECT(Users.Username.AS("username")).
		FROM(Users).
		WHERE(Users.UID.EQ(String(userID))).
		LIMIT(1)

	if err := stmt.Query(repo.db, &row); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("lookup username: %w", err)
	}
	return row.Username, nil
}
```
`Users` is already available via this file's existing dot-import of `miltechserver/.gen/miltech_ng/public/table` — no new import needed.

- [ ] **Step 5: Run the repository tests to verify they pass**

Run:
```bash
go test ./tests/pmcs_sbs_progress/... -run TestRepositoryGetInspection -v
```
Expected: PASS for both new tests. This step only touched the repository layer, so `service_impl.go` won't compile yet — that's expected; move to the next step.

- [ ] **Step 6: Update `types.go`, `mapInspection`, and the service layer**

In `api/pmcs_sbs_progress/types.go`, add the field to `InspectionResponse`:
```go
type InspectionResponse struct {
	ID                  uuid.UUID       `json:"id"`
	EquipmentID         string          `json:"equipment_id"`
	GuideManual         string          `json:"guide_manual"`
	PerformedDate       time.Time       `json:"performed_date"`
	PerformedBy         *string         `json:"performed_by,omitempty"`
	PerformedByUsername *string         `json:"performed_by_username,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	Faults              []FaultResponse `json:"faults"`
}
```

In `api/pmcs_sbs_progress/repository_impl.go`, change `mapInspection`'s signature:
```go
func mapInspection(row model.PmcsSbsInspections, performedByUsername *string, faultRows []model.PmcsSbsFaults) InspectionResponse {
	faults := make([]FaultResponse, 0, len(faultRows))
	for _, faultRow := range faultRows {
		faults = append(faults, mapFault(faultRow))
	}
	return InspectionResponse{
		ID:                  row.ID,
		EquipmentID:         row.EquipmentID,
		GuideManual:         row.GuideManual,
		PerformedDate:       row.PerformedDate,
		PerformedBy:         row.PerformedBy,
		PerformedByUsername: performedByUsername,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
		Faults:              faults,
	}
}
```

In `api/pmcs_sbs_progress/service_impl.go`, replace `EnsureInspection` and `GetInspection`:
```go
func (service *ServiceImpl) EnsureInspection(user *bootstrap.User, equipmentID string, pmcsID string, req InspectionRequest) (*InspectionResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	inspection, err := service.validateInspectionRequest(equipmentID, pmcsID, user.UserID, req)
	if err != nil {
		return nil, err
	}
	saved, err := service.repository.EnsureInspection(user, inspection)
	if err != nil {
		return nil, err
	}
	performedByUsername, err := service.resolvePerformedByUsername(user, saved.PerformedBy)
	if err != nil {
		return nil, err
	}
	resp := mapInspection(*saved, performedByUsername, nil)
	return &resp, nil
}

// resolvePerformedByUsername avoids a DB round trip in the common case: when
// the sticky performed_by owner is the caller themselves, their username is
// already on the auth token (bootstrap.User.Username). Only when a save
// touches an inspection whose sticky owner is a *different* user does this
// fall back to a single-row lookup.
func (service *ServiceImpl) resolvePerformedByUsername(user *bootstrap.User, performedBy *string) (*string, error) {
	if performedBy == nil {
		return nil, nil
	}
	if *performedBy == user.UserID {
		username := user.Username
		return &username, nil
	}
	return service.repository.LookupUsername(*performedBy)
}

func (service *ServiceImpl) GetInspection(user *bootstrap.User, equipmentID string, pmcsID string) (*InspectionResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return nil, err
	}
	parsedPmcsID, err := validatePmcsID(pmcsID)
	if err != nil {
		return nil, err
	}

	detail, faults, err := service.repository.GetInspection(user, trimmedEquipmentID, parsedPmcsID)
	if err != nil {
		return nil, err
	}
	resp := mapInspection(detail.PmcsSbsInspections, detail.PerformedByUsername, faults)
	return &resp, nil
}
```

- [ ] **Step 7: Update the `repoStub` test double and existing service tests**

In `api/pmcs_sbs_progress/service_impl_test.go`, add fields to `repoStub` and change its `GetInspection`:
```go
type repoStub struct {
	inspection    *model.PmcsSbsInspections
	detailUsername *string
	faults        []model.PmcsSbsFaults
	summaries     []InspectionSummary
	savedFault    *model.PmcsSbsFaults
	deletedCount  int64
	err           error

	lookupUsernameResult *string
	lookupUsernameErr    error
	lookupUsernameCalls  int

	capturedUser            *bootstrap.User
	capturedEquipmentID     string
	capturedPmcsID          uuid.UUID
	capturedGuideManual     string
	capturedLimit           int
	capturedOffset          int
	capturedInspection      model.PmcsSbsInspections
	capturedFault           model.PmcsSbsFaults
	capturedDelete          FaultKey
	capturedBulkKeys        []FaultKey
	capturedLookupUsernameID string
}
```
Replace `GetInspection`:
```go
func (repo *repoStub) GetInspection(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID) (*InspectionDetail, []model.PmcsSbsFaults, error) {
	repo.capturedUser = user
	repo.capturedEquipmentID = equipmentID
	repo.capturedPmcsID = pmcsID
	if repo.inspection == nil {
		return nil, repo.faults, repo.err
	}
	return &InspectionDetail{PmcsSbsInspections: *repo.inspection, PerformedByUsername: repo.detailUsername}, repo.faults, repo.err
}
```
Add `LookupUsername`:
```go
func (repo *repoStub) LookupUsername(userID string) (*string, error) {
	repo.lookupUsernameCalls++
	repo.capturedLookupUsernameID = userID
	return repo.lookupUsernameResult, repo.lookupUsernameErr
}
```

Add two new tests covering the save-path optimization (this is the behavior described in the design's "Efficiency" section — worth a dedicated test since it's a deliberate perf choice, not incidental):
```go
func TestEnsureInspectionResolvesPerformedByUsernameFromCallerWithoutLookup(t *testing.T) {
	performedBy := "user-1"
	stub := &repoStub{inspection: &model.PmcsSbsInspections{
		ID:            samplePmcsID(),
		EquipmentID:   "vehicle-1",
		GuideManual:   "pmcs_sbs/hmmwv/file.json",
		PerformedDate: time.Now().UTC(),
		PerformedBy:   &performedBy,
	}}
	svc := NewService(stub)
	user := &bootstrap.User{UserID: "user-1", Username: "jsmith"}

	resp, err := svc.EnsureInspection(user, "vehicle-1", samplePmcsIDStr, InspectionRequest{
		GuideManual:   "pmcs_sbs/hmmwv/file.json",
		PerformedDate: time.Now(),
	})

	require.NoError(t, err)
	require.NotNil(t, resp.PerformedByUsername)
	require.Equal(t, "jsmith", *resp.PerformedByUsername)
	require.Equal(t, 0, stub.lookupUsernameCalls)
}

func TestEnsureInspectionResolvesPerformedByUsernameViaLookupWhenStickyOwnerDiffers(t *testing.T) {
	performedBy := "original-user"
	lookupResult := "original-username"
	stub := &repoStub{
		inspection: &model.PmcsSbsInspections{
			ID:            samplePmcsID(),
			EquipmentID:   "vehicle-1",
			GuideManual:   "pmcs_sbs/hmmwv/file.json",
			PerformedDate: time.Now().UTC(),
			PerformedBy:   &performedBy,
		},
		lookupUsernameResult: &lookupResult,
	}
	svc := NewService(stub)
	user := &bootstrap.User{UserID: "editor-user", Username: "editor"}

	resp, err := svc.EnsureInspection(user, "vehicle-1", samplePmcsIDStr, InspectionRequest{
		GuideManual:   "pmcs_sbs/hmmwv/file.json",
		PerformedDate: time.Now(),
	})

	require.NoError(t, err)
	require.NotNil(t, resp.PerformedByUsername)
	require.Equal(t, "original-username", *resp.PerformedByUsername)
	require.Equal(t, 1, stub.lookupUsernameCalls)
	require.Equal(t, "original-user", stub.capturedLookupUsernameID)
}
```

- [ ] **Step 8: Run the full package test suite to verify everything passes**

Run:
```bash
go build ./... 2>&1 | head -40
go test ./api/pmcs_sbs_progress/... -v 2>&1 | tail -80
go test ./tests/pmcs_sbs_progress/... -v 2>&1 | tail -100
```
Expected: clean build, all tests pass, including the four new tests from this task.

- [ ] **Step 9: Update mobile-facing docs**

In `docs/api/pmcs_sbs_inspections_mobile.md`, replace the `created_by` row in the "Inspection Object" table:
```markdown
| `performed_by` | string, nullable | User id who performed the inspection. |
| `performed_by_username` | string, nullable | Username of `performed_by`. Omitted (not `null`) alongside `performed_by` when there is no value. |
```

In `docs/api/pmcs_sbs_inspection_history_mobile_changes.md`, update both JSON examples (the `EnsureInspection` success response and the `GetInspection` success response) by replacing:
```json
    "created_by": "9f1c3a2e-user-uid",
```
with:
```json
    "performed_by": "9f1c3a2e-user-uid",
    "performed_by_username": "jsmith",
```
And update the field table row:
```markdown
| `performed_by` | string (nullable) | User id who performed the inspection. This field is **omitted entirely** from the response JSON when there is no value (not `null`, simply absent). |
| `performed_by_username` | string (nullable) | Username of `performed_by`. Follows the same omitted-when-nil rule. |
```

- [ ] **Step 10: Commit**

```bash
git add api/pmcs_sbs_progress/repository.go api/pmcs_sbs_progress/repository_impl.go api/pmcs_sbs_progress/service_impl.go api/pmcs_sbs_progress/types.go api/pmcs_sbs_progress/service_impl_test.go tests/pmcs_sbs_progress/helpers_test.go tests/pmcs_sbs_progress/repository_test.go docs/api/pmcs_sbs_inspections_mobile.md docs/api/pmcs_sbs_inspection_history_mobile_changes.md
git commit -m "feat(pmcs-sbs): return performed_by_username on single-inspection endpoints"
```

---

### Task 5: `performed_by_username` on the per-equipment inspection list

**Files:**
- Modify: `api/pmcs_sbs_progress/repository.go`
- Modify: `api/pmcs_sbs_progress/repository_impl.go`
- Modify: `api/pmcs_sbs_progress/service_impl.go`
- Modify: `api/pmcs_sbs_progress/types.go`
- Modify: `api/pmcs_sbs_progress/service_impl_test.go`
- Modify: `tests/pmcs_sbs_progress/repository_test.go`
- Modify: `docs/api/pmcs_sbs_inspections_mobile.md`

**Interfaces:**
- Consumes: Task 3's renamed fields. Independent of Task 4 (touches `ListInspections`, not `GetInspection`/`EnsureInspection`) — same join pattern, different call site.
- Produces: `InspectionSummary.PerformedBy *string`, `InspectionSummary.PerformedByUsername *string`, `InspectionSummaryResponse.PerformedBy *string` (JSON `performed_by`), `InspectionSummaryResponse.PerformedByUsername *string` (JSON `performed_by_username`).

- [ ] **Step 1: Write the failing repository test**

Add to `tests/pmcs_sbs_progress/repository_test.go`:
```go
func TestRepositoryListInspectionsIncludesPerformedByUsername(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-list-username")
	user.Username = "jsmith"
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B19")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	_, err := repo.EnsureInspection(user, inspection)
	require.NoError(t, err)

	summaries, err := repo.ListInspections(user, vehicleID, "", 10, 0)
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.NotNil(t, summaries[0].PerformedBy)
	require.Equal(t, user.UserID, *summaries[0].PerformedBy)
	require.NotNil(t, summaries[0].PerformedByUsername)
	require.Equal(t, "jsmith", *summaries[0].PerformedByUsername)
}
```

- [ ] **Step 2: Run it to verify it fails**

Run:
```bash
go test ./tests/pmcs_sbs_progress/... -run TestRepositoryListInspectionsIncludesPerformedByUsername -v
```
Expected: FAIL — compile error, `InspectionSummary` has no `PerformedByUsername` field yet.

- [ ] **Step 3: Add fields to `InspectionSummary` in `repository.go`**

```go
type InspectionSummary struct {
	ID                  uuid.UUID
	GuideManual         string
	PerformedDate       time.Time
	FaultCount          int
	CreatedAt           time.Time
	PerformedBy         *string
	PerformedByUsername *string
}
```

- [ ] **Step 4: Add the join to `ListInspections` in `repository_impl.go`**

Replace the whole function:
```go
func (repo *RepositoryImpl) ListInspections(user *bootstrap.User, equipmentID string, guideManual string, limit int, offset int) ([]InspectionSummary, error) {
	if err := repo.requireVehicleAccess(user, equipmentID); err != nil {
		return nil, err
	}

	condition := PmcsSbsInspections.EquipmentID.EQ(String(equipmentID))
	if guideManual != "" {
		condition = condition.AND(PmcsSbsInspections.GuideManual.EQ(String(guideManual)))
	}

	var inspections []struct {
		model.PmcsSbsInspections
		PerformedByUsername *string `sql:"performed_by_username"`
	}
	stmt := SELECT(
		PmcsSbsInspections.AllColumns,
		Users.Username.AS("performed_by_username"),
	).
		FROM(PmcsSbsInspections.LEFT_JOIN(Users, Users.UID.EQ(PmcsSbsInspections.PerformedBy))).
		WHERE(condition).
		ORDER_BY(PmcsSbsInspections.PerformedDate.DESC()).
		LIMIT(int64(limit)).
		OFFSET(int64(offset))

	if err := stmt.Query(repo.db, &inspections); err != nil {
		return nil, fmt.Errorf("list pmcs sbs inspections: %w", err)
	}
	if len(inspections) == 0 {
		return []InspectionSummary{}, nil
	}

	ids := make([]Expression, 0, len(inspections))
	for _, inspection := range inspections {
		ids = append(ids, UUID(inspection.ID))
	}

	var counts []struct {
		PmcsID uuid.UUID `sql:"pmcs_id"`
		Total  int32     `sql:"total"`
	}
	countStmt := SELECT(
		PmcsSbsFaults.PmcsID.AS("pmcs_id"),
		COUNT(PmcsSbsFaults.PmcsID).AS("total"),
	).FROM(PmcsSbsFaults).
		WHERE(PmcsSbsFaults.PmcsID.IN(ids...)).
		GROUP_BY(PmcsSbsFaults.PmcsID)

	if err := countStmt.Query(repo.db, &counts); err != nil {
		return nil, fmt.Errorf("count pmcs sbs faults: %w", err)
	}

	countByID := make(map[uuid.UUID]int, len(counts))
	for _, c := range counts {
		countByID[c.PmcsID] = int(c.Total)
	}

	summaries := make([]InspectionSummary, 0, len(inspections))
	for _, inspection := range inspections {
		summaries = append(summaries, InspectionSummary{
			ID:                  inspection.ID,
			GuideManual:         inspection.GuideManual,
			PerformedDate:       inspection.PerformedDate,
			FaultCount:          countByID[inspection.ID],
			CreatedAt:           inspection.CreatedAt,
			PerformedBy:         inspection.PerformedBy,
			PerformedByUsername: inspection.PerformedByUsername,
		})
	}
	return summaries, nil
}
```

- [ ] **Step 5: Run the repository test to verify it passes**

Run:
```bash
go test ./tests/pmcs_sbs_progress/... -run TestRepositoryListInspectionsIncludesPerformedByUsername -v
```
Expected: PASS.

- [ ] **Step 6: Add fields to `InspectionSummaryResponse` and map them through in the service**

In `api/pmcs_sbs_progress/types.go`:
```go
type InspectionSummaryResponse struct {
	ID                  uuid.UUID `json:"id"`
	GuideManual         string    `json:"guide_manual"`
	PerformedDate       time.Time `json:"performed_date"`
	FaultCount          int       `json:"fault_count"`
	CreatedAt           time.Time `json:"created_at"`
	PerformedBy         *string   `json:"performed_by,omitempty"`
	PerformedByUsername *string   `json:"performed_by_username,omitempty"`
}
```

In `api/pmcs_sbs_progress/service_impl.go`'s `ListInspections`:
```go
	responses := make([]InspectionSummaryResponse, 0, len(summaries))
	for _, summary := range summaries {
		responses = append(responses, InspectionSummaryResponse{
			ID:                  summary.ID,
			GuideManual:         summary.GuideManual,
			PerformedDate:       summary.PerformedDate,
			FaultCount:          summary.FaultCount,
			CreatedAt:           summary.CreatedAt,
			PerformedBy:         summary.PerformedBy,
			PerformedByUsername: summary.PerformedByUsername,
		})
	}
	return &InspectionListResponse{Inspections: responses, Count: len(responses)}, nil
```

- [ ] **Step 7: Extend the existing service-level test**

In `api/pmcs_sbs_progress/service_impl_test.go`, replace `TestListInspectionsMapsSummaries`:
```go
func TestListInspectionsMapsSummaries(t *testing.T) {
	now := time.Now().UTC()
	performedBy := "user-1"
	performedByUsername := "jsmith"
	stub := &repoStub{summaries: []InspectionSummary{
		{ID: samplePmcsID(), GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: now, FaultCount: 2, CreatedAt: now, PerformedBy: &performedBy, PerformedByUsername: &performedByUsername},
	}}
	svc := NewService(stub)

	resp, err := svc.ListInspections(requireUser(), "vehicle-1", ListInspectionsRequest{Limit: 10, Offset: 0})

	require.NoError(t, err)
	require.Equal(t, 1, resp.Count)
	require.Equal(t, 2, resp.Inspections[0].FaultCount)
	require.NotNil(t, resp.Inspections[0].PerformedBy)
	require.Equal(t, "user-1", *resp.Inspections[0].PerformedBy)
	require.NotNil(t, resp.Inspections[0].PerformedByUsername)
	require.Equal(t, "jsmith", *resp.Inspections[0].PerformedByUsername)
}
```

- [ ] **Step 8: Run the full package test suite**

Run:
```bash
go build ./... 2>&1 | head -40
go test ./api/pmcs_sbs_progress/... -v 2>&1 | tail -80
go test ./tests/pmcs_sbs_progress/... -v 2>&1 | tail -100
```
Expected: clean build, all tests pass.

- [ ] **Step 9: Update mobile-facing docs**

In `docs/api/pmcs_sbs_inspections_mobile.md`, add two rows to the "Inspection Summary Object (list endpoint)" table:
```markdown
| `performed_by` | string, nullable | User id who performed the inspection. |
| `performed_by_username` | string, nullable | Username of `performed_by`. |
```

- [ ] **Step 10: Commit**

```bash
git add api/pmcs_sbs_progress/repository.go api/pmcs_sbs_progress/repository_impl.go api/pmcs_sbs_progress/service_impl.go api/pmcs_sbs_progress/types.go api/pmcs_sbs_progress/service_impl_test.go tests/pmcs_sbs_progress/repository_test.go docs/api/pmcs_sbs_inspections_mobile.md
git commit -m "feat(pmcs-sbs): return performed_by_username on the inspection list endpoint"
```

---

### Task 6: `performed_by_username` on the cross-shop equipment+history aggregate

**Files:**
- Modify: `api/response/user_shops_response.go`
- Modify: `api/shops/aggregates/repository_impl.go`
- Modify: `tests/shops/helpers_test.go`
- Modify: `tests/shops/shops_equipment_pmcs_history_test.go`

**Interfaces:**
- Consumes: Task 1's renamed column. Independent of Tasks 4 and 5 (different package, `shops/aggregates` rather than `pmcs_sbs_progress`) — same join pattern, different call site and response type.
- Produces: `response.PmcsHistorySummary.PerformedBy *string` (JSON `performed_by`), `response.PmcsHistorySummary.PerformedByUsername *string` (JSON `performed_by_username`).

- [ ] **Step 1: Add a `performedBy` parameter to the `createPmcsInspection` test helper**

In `tests/shops/helpers_test.go`, replace:
```go
func createPmcsInspection(t *testing.T, db *sql.DB, equipmentID string, guideManual string, performedDate time.Time) string {
	t.Helper()

	id := uuid.New().String()
	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO pmcs_sbs_inspections (id, equipment_id, guide_manual, performed_date, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $5)`,
		id, equipmentID, guideManual, performedDate, now,
	)
	require.NoError(t, err)
	return id
}
```
with:
```go
func createPmcsInspection(t *testing.T, db *sql.DB, equipmentID string, guideManual string, performedDate time.Time, performedBy string) string {
	t.Helper()

	id := uuid.New().String()
	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO pmcs_sbs_inspections (id, equipment_id, guide_manual, performed_date, performed_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $6)`,
		id, equipmentID, guideManual, performedDate, performedBy, now,
	)
	require.NoError(t, err)
	return id
}
```

Update the three existing call sites in `tests/shops/shops_equipment_pmcs_history_test.go` (all three tests already `ensureUser(t, testDB, "history-user")` before calling this helper, so `performedBy: "history-user"` satisfies the FK):
```go
	newerInspectionID := createPmcsInspection(t, testDB, vehicleWithHistoryID, "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json", newerTime, "history-user")
	createPmcsFault(t, testDB, newerInspectionID, "before", 0)
	createPmcsFault(t, testDB, newerInspectionID, "during", 1)
	olderInspectionID := createPmcsInspection(t, testDB, vehicleWithHistoryID, "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json", olderTime, "history-user")
```
and
```go
	inspectionID := createPmcsInspection(t, testDB, vehicleID, "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json", performedDate, "history-user")
```

- [ ] **Step 2: Write the failing repository test**

Add to `tests/shops/shops_equipment_pmcs_history_test.go`:
```go
func TestGetEquipmentPmcsHistoryRepositoryIncludesPerformedByUsername(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "history-user")
	router := newTestRouter(t)

	shopID := createShop(t, router, "history-user", "History Shop")
	vehicleID := createVehicle(t, router, "history-user", shopID)

	performedDate := time.Date(2026, time.July, 16, 14, 30, 0, 0, time.UTC)
	inspectionID := createPmcsInspection(t, testDB, vehicleID, "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json", performedDate, "history-user")

	repository := aggregates.NewRepository(testDB)
	equipment, err := repository.GetEquipmentPmcsHistory(context.Background(), &bootstrap.User{UserID: "history-user"})

	require.NoError(t, err)
	require.Len(t, equipment, 1)
	require.Len(t, equipment[0].HistoricalPmcs, 1)
	entry := equipment[0].HistoricalPmcs[0]
	require.Equal(t, inspectionID, entry.ID.String())
	require.NotNil(t, entry.PerformedBy)
	require.Equal(t, "history-user", *entry.PerformedBy)
	require.NotNil(t, entry.PerformedByUsername)
	require.Equal(t, "test-user", *entry.PerformedByUsername)
}
```
(`"test-user"` is the fixed username `ensureUser` in this package always inserts — see `tests/shops/helpers_test.go:139`.)

- [ ] **Step 3: Run it to verify it fails**

Run:
```bash
go test ./tests/shops/... -run TestGetEquipmentPmcsHistoryRepositoryIncludesPerformedByUsername -v
```
Expected: FAIL — compile error, `PmcsHistorySummary` has no `PerformedBy`/`PerformedByUsername` fields yet.

- [ ] **Step 4: Add fields to `response.PmcsHistorySummary`**

In `api/response/user_shops_response.go`:
```go
type PmcsHistorySummary struct {
	ID                  uuid.UUID `json:"id"`
	GuideManual         string    `json:"guide_manual"`
	PerformedDate       time.Time `json:"performed_date"`
	FaultCount          int       `json:"fault_count"`
	CreatedAt           time.Time `json:"created_at"`
	PerformedBy         *string   `json:"performed_by,omitempty"`
	PerformedByUsername *string   `json:"performed_by_username,omitempty"`
}
```

- [ ] **Step 5: Add the join to `GetEquipmentPmcsHistory` in `api/shops/aggregates/repository_impl.go`**

Replace the inspections query and its two consuming loops:
```go
	var inspections []struct {
		model.PmcsSbsInspections
		PerformedByUsername *string `sql:"performed_by_username"`
	}
	inspectionsStmt := SELECT(
		PmcsSbsInspections.AllColumns,
		Users.Username.AS("performed_by_username"),
	).
		FROM(PmcsSbsInspections.LEFT_JOIN(Users, Users.UID.EQ(PmcsSbsInspections.PerformedBy))).
		WHERE(PmcsSbsInspections.EquipmentID.IN(equipmentIDs...)).
		ORDER_BY(PmcsSbsInspections.EquipmentID.ASC(), PmcsSbsInspections.PerformedDate.DESC())

	if err := inspectionsStmt.QueryContext(ctx, repo.db, &inspections); err != nil {
		return nil, fmt.Errorf("failed to query pmcs inspections for equipment history: %w", err)
	}
```
(the `faultCountByInspectionID` block right after is unchanged — `inspection.ID` still resolves via the embedded `model.PmcsSbsInspections`).

Replace the merge loop:
```go
	historyByEquipmentID := make(map[string][]response.PmcsHistorySummary, len(equipmentRows))
	for _, inspection := range inspections {
		historyByEquipmentID[inspection.EquipmentID] = append(historyByEquipmentID[inspection.EquipmentID], response.PmcsHistorySummary{
			ID:                  inspection.ID,
			GuideManual:         inspection.GuideManual,
			PerformedDate:       inspection.PerformedDate,
			FaultCount:          faultCountByInspectionID[inspection.ID],
			CreatedAt:           inspection.CreatedAt,
			PerformedBy:         inspection.PerformedBy,
			PerformedByUsername: inspection.PerformedByUsername,
		})
	}
```
`Users` is already available via this file's existing dot-import of `miltechserver/.gen/miltech_ng/public/table` — no new import needed.

- [ ] **Step 6: Run the repository test to verify it passes**

Run:
```bash
go test ./tests/shops/... -run TestGetEquipmentPmcsHistoryRepositoryIncludesPerformedByUsername -v
```
Expected: PASS.

- [ ] **Step 7: Extend the existing handler-level test**

In `tests/shops/shops_equipment_pmcs_history_test.go`'s `TestEquipmentPmcsHistoryEndpoint`, add after the existing `fault_count` assertion:
```go
	require.Equal(t, "history-user", inspection["performed_by"])
	require.Equal(t, "test-user", inspection["performed_by_username"])
```

- [ ] **Step 8: Run the full `shops` test suite**

Run:
```bash
go build ./... 2>&1 | head -40
go test ./tests/shops/... -run PmcsHistory -v 2>&1 | tail -100
go test ./tests/shops/... -v 2>&1 | tail -150
```
Expected: clean build, all `shops` tests pass (the last command catches any other test in the package relying on the old `createPmcsInspection` 4-arg signature — there should be none per Task 6 Step 1's grep, but this confirms it).

- [ ] **Step 9: Commit**

```bash
git add api/response/user_shops_response.go api/shops/aggregates/repository_impl.go tests/shops/helpers_test.go tests/shops/shops_equipment_pmcs_history_test.go
git commit -m "feat(shops): return performed_by_username on the equipment pmcs history aggregate"
```

---

## Self-Review Notes

**Spec coverage:** Task 1 covers the migration decision. Task 2 covers model regeneration. Task 3 covers the pure rename (preserving sticky-on-conflict, verified by unchanged `ON CONFLICT` `SET` clause). Task 4 covers the single-inspection detail endpoint, the save-path optimization, and its two docs. Task 5 covers the per-equipment list endpoint and its doc. Task 6 covers the cross-shop aggregate. Every response shape named in the spec (`InspectionResponse`, `InspectionSummaryResponse`, `PmcsHistorySummary`) is covered by exactly one task.

**Type consistency:** `PerformedBy *string` and `PerformedByUsername *string` are named identically across every touched type (`model.PmcsSbsInspections`, `InspectionDetail`, `InspectionSummary`, `InspectionSummaryResponse`, `InspectionResponse`, `response.PmcsHistorySummary`) — no drift between tasks. `mapInspection`'s new three-argument signature (`row, performedByUsername, faultRows`) is used consistently in both its Task 4 call sites (`EnsureInspection`, `GetInspection`).

**Scope:** Each task after Task 3 produces an independently testable, independently revertible deliverable (one endpoint's worth of behavior). Tasks 4, 5, and 6 have no dependency on each other and can be executed in any order, or in parallel by separate subagents, once Task 3 is merged.
