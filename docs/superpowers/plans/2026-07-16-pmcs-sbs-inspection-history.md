# PMCS SBS Inspection History Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give equipment multiple dated PMCS inspection records over time, each grouping its own faults, replacing today's single-state-per-checklist-item model.

**Architecture:** A new `pmcs_sbs_inspections` table (client-generated UUID PK, FK to `shop_vehicle`) becomes the parent of a re-keyed `pmcs_sbs_faults` table (`pmcs_id` FK replaces the old `equipment_id`+`guide_manual` composite key). The Go package `api/pmcs_sbs_progress/` is rewritten layer by layer (repository → service → route) to match, following its existing three-layer structure. No data migration — existing fault rows are current-state-only and are discarded.

**Tech Stack:** Go, Gin, go-jet v2.13.0 (query builder + code generator), PostgreSQL, `github.com/google/uuid`, `testify/require`.

## Global Constraints

- Spec: `docs/superpowers/specs/2026-07-16-pmcs-sbs-inspection-history-design.md` — every task below implements a section of it; read it first if anything here is ambiguous.
- `pmcs_sbs_inspections.id` and `pmcs_sbs_faults.pmcs_id` are client-generated UUIDs (Postgres `UUID` type, no server-side default) — the server never generates an inspection id.
- `guide_manual` is immutable after an inspection is first created; a request that supplies a different `guide_manual` for an existing `pmcs_id` (or a `pmcs_id` that belongs to a different `equipment_id`) must fail with `ErrInspectionConflict` (HTTP 409), never silently succeed.
- Every query that takes both `:equipment_id` and a `pmcs_id` must filter by the inspection's `equipment_id` in addition to the existing shop-membership check, so a mismatched `pmcs_id` reads as not-found rather than crossing vehicle boundaries.
- `performed_date` is always client-supplied, never a server write-timestamp.
- No existing `pmcs_sbs_faults` data is preserved across the migration — the migration drops and recreates the table.
- Follow existing project conventions throughout: go-jet dot-imported from `miltechserver/.gen/miltech_ng/public/table`, `model` package imported normally, `bootstrap.User` for the authenticated caller, `response.StandardResponse{Status, Data, Message}` for success JSON bodies, `gin.H{"message": ...}` for error bodies.

---

### Task 1: Database Migration and Jet Model Regeneration

**Files:**
- Create: `migrations/006_create_pmcs_sbs_inspections.sql`
- Create: `migrations/006_rollback_pmcs_sbs_inspections.sql`
- Modify (generated, not hand-edited): `.gen/miltech_ng/public/model/pmcs_sbs_inspections.go`, `.gen/miltech_ng/public/table/pmcs_sbs_inspections.go`, `.gen/miltech_ng/public/model/pmcs_sbs_faults.go`, `.gen/miltech_ng/public/table/pmcs_sbs_faults.go`

**Interfaces:**
- Produces: `model.PmcsSbsInspections{ID uuid.UUID, EquipmentID string, GuideManual string, PerformedDate time.Time, CreatedBy *string, CreatedAt time.Time, UpdatedAt time.Time}` and `model.PmcsSbsFaults{PmcsID uuid.UUID, SectionID string, ItemIndex int32, ItemNo string, Status string, FaultText string, CorrectiveAction string, CreatedAt time.Time, UpdatedAt time.Time}` — these exact field names and types are what every later task's Go code is written against.
- Produces: table variables `PmcsSbsInspections` and `PmcsSbsFaults` (dot-importable from `miltechserver/.gen/miltech_ng/public/table`) with columns matching the model fields.

- [ ] **Step 1: Write the migration**

```sql
-- PMCS SBS Inspection History
-- Migration: 006_create_pmcs_sbs_inspections.sql
--
-- Introduces pmcs_sbs_inspections as the parent of pmcs_sbs_faults so a
-- vehicle can have many dated PMCS inspections over time, each with its own
-- faults, instead of one overwritable fault state per checklist item.
--
-- Existing pmcs_sbs_faults rows are current-state-only snapshots with no
-- date-performed concept and are not preserved; see
-- docs/superpowers/specs/2026-07-16-pmcs-sbs-inspection-history-design.md.

DROP TABLE IF EXISTS pmcs_sbs_faults;

CREATE TABLE pmcs_sbs_inspections (
    id              UUID NOT NULL PRIMARY KEY,
    equipment_id    TEXT NOT NULL,
    guide_manual    TEXT NOT NULL,
    performed_date  TIMESTAMPTZ NOT NULL,
    created_by      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

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

CREATE TABLE pmcs_sbs_faults (
    pmcs_id            UUID NOT NULL,
    section_id         TEXT NOT NULL,
    item_index         INTEGER NOT NULL,
    item_no            TEXT NOT NULL,
    status             TEXT NOT NULL,
    fault_text         TEXT NOT NULL,
    corrective_action  TEXT NOT NULL DEFAULT '',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),

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

- [ ] **Step 2: Write the rollback**

```sql
-- Rollback: 006_rollback_pmcs_sbs_inspections.sql

DROP TABLE IF EXISTS pmcs_sbs_faults;
DROP INDEX IF EXISTS idx_pmcs_sbs_inspections_equipment_performed;
DROP TABLE IF EXISTS pmcs_sbs_inspections;

CREATE TABLE pmcs_sbs_faults (
    equipment_id       text NOT NULL,
    guide_manual       text NOT NULL,
    section_id         text NOT NULL,
    item_index         integer NOT NULL,
    item_no            text NOT NULL,
    status             text NOT NULL,
    fault_text         text NOT NULL,
    corrective_action  text NOT NULL DEFAULT '',
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),

    PRIMARY KEY (equipment_id, guide_manual, section_id, item_index),
    CONSTRAINT fk_pmcs_sbs_faults_equipment_id
        FOREIGN KEY (equipment_id) REFERENCES shop_vehicle(id)
        ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT pmcs_sbs_faults_item_index_check CHECK (item_index >= 0),
    CONSTRAINT pmcs_sbs_faults_status_check CHECK (status = ANY (ARRAY['x','slash','dash'])),
    CONSTRAINT pmcs_sbs_faults_nonblank_fields_check CHECK (
        btrim(equipment_id) <> '' AND btrim(guide_manual) <> '' AND
        btrim(section_id) <> '' AND btrim(item_no) <> '' AND btrim(fault_text) <> ''
    ),
    CONSTRAINT pmcs_sbs_faults_guide_manual_format_check CHECK (
        guide_manual = btrim(guide_manual) AND guide_manual LIKE 'pmcs_sbs/%' AND right(guide_manual, 5) = '.json'
    )
);
```

- [ ] **Step 3: Apply the migration to both the dev and test databases**

Run (connection details from `.env`; the test database is a separate database on the same server, used by `tests/pmcs_sbs_progress/main_test.go`):

```bash
PGPASSWORD=potato123 psql -h 192.168.20.70 -U postgres -d miltech_ng -f migrations/006_create_pmcs_sbs_inspections.sql
PGPASSWORD=potato123 psql -h 192.168.20.70 -U postgres -d miltech_ng_test -f migrations/006_create_pmcs_sbs_inspections.sql
```

Expected: both commands print `DROP TABLE`, `CREATE TABLE`, `CREATE INDEX`, `CREATE TABLE` with no errors.

- [ ] **Step 4: Verify the schema with a direct query**

```bash
PGPASSWORD=potato123 psql -h 192.168.20.70 -U postgres -d miltech_ng -c "\d pmcs_sbs_inspections" -c "\d pmcs_sbs_faults"
```

Expected: `pmcs_sbs_inspections` shows columns `id, equipment_id, guide_manual, performed_date, created_by, created_at, updated_at` with `id` as primary key; `pmcs_sbs_faults` shows `pmcs_id, section_id, item_index, item_no, status, fault_text, corrective_action, created_at, updated_at` with primary key `(pmcs_id, section_id, item_index)` and a foreign key on `pmcs_id`.

- [ ] **Step 5: Regenerate Jet models by starting the server against the dev database**

```bash
go run . &
GO_PID=$!
sleep 8
kill $GO_PID 2>/dev/null
wait $GO_PID 2>/dev/null
```

`timeout` is not used here since it's a GNU coreutils command not available by default on macOS; `$!` (last background PID) is POSIX-portable and, unlike `%1`, doesn't require interactive job control. Expected: the process either exits on its own (e.g. missing Firebase/Azure credentials — fine, schema generation runs first, before those are needed) or is killed after 8 seconds. Either way, `.gen/miltech_ng/public/model/pmcs_sbs_inspections.go` and `.gen/miltech_ng/public/table/pmcs_sbs_inspections.go` now exist, and `.gen/miltech_ng/public/model/pmcs_sbs_faults.go` / `.gen/miltech_ng/public/table/pmcs_sbs_faults.go` reflect the new columns. Confirm with:

```bash
git status --short .gen/
cat .gen/miltech_ng/public/model/pmcs_sbs_inspections.go
```

Expected: `model.PmcsSbsInspections` struct has fields `ID uuid.UUID`, `EquipmentID string`, `GuideManual string`, `PerformedDate time.Time`, `CreatedBy *string`, `CreatedAt time.Time`, `UpdatedAt time.Time` (nullable `created_by` becomes a pointer, matching the existing `NetVotes *int32` pattern for other nullable columns in this codebase). `model.PmcsSbsFaults` has `PmcsID uuid.UUID` in place of the old `EquipmentID`/`GuideManual` string fields.

- [ ] **Step 6: Confirm the whole module still builds**

```bash
go build ./...
```

Expected: FAILS at this point — `api/pmcs_sbs_progress/*.go` and `tests/pmcs_sbs_progress/*.go` still reference the old `model.PmcsSbsFaults.EquipmentID`/`GuideManual` fields that no longer exist. This confirms the generated models actually changed shape; Task 2 fixes the build.

- [ ] **Step 7: Commit**

```bash
git add migrations/006_create_pmcs_sbs_inspections.sql migrations/006_rollback_pmcs_sbs_inspections.sql .gen/
git commit -m "feat(pmcs-sbs): add pmcs_sbs_inspections table and re-key faults under it"
```

---

### Task 2: Repository Layer — Inspections and Re-keyed Faults

**Files:**
- Modify: `api/pmcs_sbs_progress/types.go`
- Modify: `api/pmcs_sbs_progress/errors.go`
- Modify: `api/pmcs_sbs_progress/repository.go`
- Modify: `api/pmcs_sbs_progress/repository_impl.go`
- Modify: `tests/pmcs_sbs_progress/helpers_test.go`
- Modify: `tests/pmcs_sbs_progress/repository_test.go`

**Interfaces:**
- Consumes: `model.PmcsSbsInspections`, `model.PmcsSbsFaults`, `PmcsSbsInspections`/`PmcsSbsFaults` table vars from Task 1.
- Produces: `Repository` interface (`EnsureInspection`, `GetInspection`, `ListInspections`, `DeleteInspection`, `UpsertFault`, `DeleteFault`, `DeleteFaults`), `FaultKey{PmcsID uuid.UUID, SectionID string, ItemIndex int32}`, `InspectionSummary{ID uuid.UUID, GuideManual string, PerformedDate time.Time, FaultCount int, CreatedAt time.Time}`, errors `ErrInvalidPmcsID`, `ErrInspectionNotFound`, `ErrInspectionConflict` — Task 3 (service layer) is written directly against these names and signatures.

- [ ] **Step 1: Update `types.go` request/response DTOs**

Replace the entire file:

```go
package pmcs_sbs_progress

import (
	"time"

	"github.com/google/uuid"
)

type InspectionRequest struct {
	GuideManual   string    `json:"guide_manual"`
	PerformedDate time.Time `json:"performed_date"`
}

type ListInspectionsRequest struct {
	GuideManual string `form:"guide_manual"`
	Limit       int    `form:"limit,default=1000" binding:"omitempty,min=1,max=1000"`
	Offset      int    `form:"offset,default=0" binding:"omitempty,min=0"`
}

type FaultRequest struct {
	GuideManual      string    `json:"guide_manual"`
	PerformedDate    time.Time `json:"performed_date"`
	SectionID        string    `json:"section_id"`
	ItemIndex        int32     `json:"item_index"`
	ItemNo           string    `json:"item_no"`
	Status           string    `json:"status"`
	FaultText        string    `json:"fault_text"`
	CorrectiveAction string    `json:"corrective_action"`
}

type DeleteFaultRequest struct {
	SectionID string `json:"section_id"`
	ItemIndex int32  `json:"item_index"`
}

type BulkDeleteFaultRequest struct {
	Faults []BulkDeleteFaultItemRequest `json:"faults"`
}

type BulkDeleteFaultItemRequest struct {
	SectionID string `json:"section_id"`
	ItemIndex int32  `json:"item_index"`
}

type FaultResponse struct {
	PmcsID           uuid.UUID `json:"pmcs_id"`
	SectionID        string    `json:"section_id"`
	ItemIndex        int32     `json:"item_index"`
	ItemNo           string    `json:"item_no"`
	Status           string    `json:"status"`
	FaultText        string    `json:"fault_text"`
	CorrectiveAction string    `json:"corrective_action"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type InspectionResponse struct {
	ID            uuid.UUID       `json:"id"`
	EquipmentID   string          `json:"equipment_id"`
	GuideManual   string          `json:"guide_manual"`
	PerformedDate time.Time       `json:"performed_date"`
	CreatedBy     *string         `json:"created_by,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	Faults        []FaultResponse `json:"faults"`
}

type InspectionSummaryResponse struct {
	ID            uuid.UUID `json:"id"`
	GuideManual   string    `json:"guide_manual"`
	PerformedDate time.Time `json:"performed_date"`
	FaultCount    int       `json:"fault_count"`
	CreatedAt     time.Time `json:"created_at"`
}

type InspectionListResponse struct {
	Inspections []InspectionSummaryResponse `json:"inspections"`
	Count       int                          `json:"count"`
}

type BulkDeleteFaultResponse struct {
	RequestedCount int `json:"requested_count"`
	DeletedCount   int `json:"deleted_count"`
}
```

- [ ] **Step 2: Update `errors.go`**

Replace the entire file:

```go
package pmcs_sbs_progress

import "errors"

var (
	ErrInvalidID          = errors.New("invalid id")
	ErrInvalidPmcsID      = errors.New("invalid pmcs id")
	ErrInvalidGuideManual = errors.New("invalid guide manual")
	ErrInvalidRequest     = errors.New("invalid request")
	ErrInvalidStatus      = errors.New("invalid fault status")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrNotFound           = errors.New("pmcs sbs equipment not found")
	ErrInspectionNotFound = errors.New("pmcs sbs inspection not found")
	ErrInspectionConflict = errors.New("pmcs sbs inspection conflict")
)
```

- [ ] **Step 3: Update `repository.go` interface**

Replace the entire file:

```go
package pmcs_sbs_progress

import (
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"

	"github.com/google/uuid"
)

type Repository interface {
	EnsureInspection(user *bootstrap.User, inspection model.PmcsSbsInspections) (*model.PmcsSbsInspections, error)
	GetInspection(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID) (*model.PmcsSbsInspections, []model.PmcsSbsFaults, error)
	ListInspections(user *bootstrap.User, equipmentID string, guideManual string, limit int, offset int) ([]InspectionSummary, error)
	DeleteInspection(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID) error

	UpsertFault(user *bootstrap.User, inspection model.PmcsSbsInspections, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error)
	DeleteFault(user *bootstrap.User, equipmentID string, key FaultKey) error
	DeleteFaults(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID, keys []FaultKey) (int64, error)
}

type FaultKey struct {
	PmcsID    uuid.UUID
	SectionID string
	ItemIndex int32
}

type InspectionSummary struct {
	ID            uuid.UUID
	GuideManual   string
	PerformedDate time.Time
	FaultCount    int
	CreatedAt     time.Time
}
```

- [ ] **Step 4: Update `tests/pmcs_sbs_progress/helpers_test.go`**

Replace `clearPmcsSbsTables`, `sampleFault`, and add `sampleInspection`. Full file:

```go
package pmcs_sbs_progress_test

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func testUser(id string) *bootstrap.User {
	return &bootstrap.User{UserID: id, Username: id, Email: id + "@example.com"}
}

func ensureUser(t *testing.T, db *sql.DB, user *bootstrap.User) {
	t.Helper()

	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO users (uid, email, username, created_at, is_enabled)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (uid) DO NOTHING`,
		user.UserID,
		user.Email,
		user.Username,
		now,
		true,
	)
	require.NoError(t, err)
}

func clearPmcsSbsTables(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(
		`TRUNCATE TABLE
			pmcs_sbs_faults,
			pmcs_sbs_inspections,
			shop_vehicle_notification_changes,
			shop_vehicle_notifications,
			shop_vehicle,
			shop_members,
			shops
		RESTART IDENTITY CASCADE`,
	)
	require.NoError(t, err)
}

func createShopWithMember(t *testing.T, db *sql.DB, user *bootstrap.User, role string) string {
	t.Helper()

	shopID := uuid.New().String()
	memberID := uuid.New().String()
	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO shops (id, name, details, created_by, created_at, updated_at, admin_only_lists)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		shopID,
		"PMCS Shop",
		"Details",
		user.UserID,
		now,
		now,
		false,
	)
	require.NoError(t, err)

	_, err = db.Exec(
		`INSERT INTO shop_members (id, shop_id, user_id, role, joined_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		memberID,
		shopID,
		user.UserID,
		role,
		now,
	)
	require.NoError(t, err)
	return shopID
}

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

func createShopVehicle(t *testing.T, db *sql.DB, shopID string, creator *bootstrap.User, admin string) string {
	t.Helper()

	vehicleID := uuid.New().String()
	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO shop_vehicle (
			id, creator_id, niin, admin, model, serial, uoc, mileage, hours, comment,
			save_time, last_updated, shop_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		vehicleID,
		creator.UserID,
		"",
		admin,
		"M1152A1",
		fmt.Sprintf("SER-%s", admin),
		"UNK",
		0,
		0,
		"",
		now,
		now,
		shopID,
	)
	require.NoError(t, err)
	return vehicleID
}

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

func sampleFault(pmcsID uuid.UUID) model.PmcsSbsFaults {
	now := time.Now().UTC()
	return model.PmcsSbsFaults{
		PmcsID:           pmcsID,
		SectionID:        "before",
		ItemIndex:        0,
		ItemNo:           "1",
		Status:           "x",
		FaultText:        "leak",
		CorrectiveAction: "",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}
```

- [ ] **Step 5: Write the new `tests/pmcs_sbs_progress/repository_test.go`**

Replace the entire file:

```go
package pmcs_sbs_progress_test

import (
	"testing"
	"time"

	"miltechserver/api/pmcs_sbs_progress"

	"github.com/stretchr/testify/require"
)

func TestRepositoryEnsureInspectionCreatesRecord(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-insp-create")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B1")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	saved, err := repo.EnsureInspection(user, inspection)

	require.NoError(t, err)
	require.Equal(t, inspection.ID, saved.ID)
	require.Equal(t, vehicleID, saved.EquipmentID)
	require.Equal(t, "pmcs_sbs/hmmwv/file.json", saved.GuideManual)
	require.NotNil(t, saved.CreatedBy)
	require.Equal(t, user.UserID, *saved.CreatedBy)
}

func TestRepositoryEnsureInspectionIsIdempotentAndUpdatesPerformedDate(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-insp-idem")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B2")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	first, err := repo.EnsureInspection(user, inspection)
	require.NoError(t, err)

	corrected := inspection
	corrected.PerformedDate = first.PerformedDate.Add(-24 * time.Hour)
	second, err := repo.EnsureInspection(user, corrected)

	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.True(t, second.PerformedDate.Before(first.PerformedDate))
}

func TestRepositoryEnsureInspectionRejectsGuideManualMismatch(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-insp-conflict")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B3")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	_, err := repo.EnsureInspection(user, inspection)
	require.NoError(t, err)

	mismatched := inspection
	mismatched.GuideManual = "pmcs_sbs/hmmwv/other.json"
	_, err = repo.EnsureInspection(user, mismatched)

	require.ErrorIs(t, err, pmcs_sbs_progress.ErrInspectionConflict)
}

func TestRepositoryEnsureInspectionDeniesNonMember(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	owner := testUser("pmcs-insp-owner")
	other := testUser("pmcs-insp-other")
	ensureUser(t, testDB, owner)
	ensureUser(t, testDB, other)
	shopID := createShopWithMember(t, testDB, owner, "admin")
	vehicleID := createShopVehicle(t, testDB, shopID, owner, "B4")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	_, err := repo.EnsureInspection(other, sampleInspection(vehicleID, other.UserID))

	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)
}

func TestRepositoryUpsertFaultCreatesInspectionImplicitly(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-implicit")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B5")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	fault := sampleFault(inspection.ID)

	saved, err := repo.UpsertFault(user, inspection, fault)
	require.NoError(t, err)
	require.Equal(t, inspection.ID, saved.PmcsID)

	fetched, faults, err := repo.GetInspection(user, vehicleID, inspection.ID)
	require.NoError(t, err)
	require.Equal(t, inspection.GuideManual, fetched.GuideManual)
	require.Len(t, faults, 1)
	require.Equal(t, "leak", faults[0].FaultText)
}

func TestRepositoryUpsertFaultReusesExistingInspectionForSamePmcsID(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-reuse")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B6")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	first := sampleFault(inspection.ID)
	_, err := repo.UpsertFault(user, inspection, first)
	require.NoError(t, err)

	second := sampleFault(inspection.ID)
	second.SectionID = "during"
	second.ItemIndex = 1
	second.FaultText = "second fault"
	_, err = repo.UpsertFault(user, inspection, second)
	require.NoError(t, err)

	_, faults, err := repo.GetInspection(user, vehicleID, inspection.ID)
	require.NoError(t, err)
	require.Len(t, faults, 2)
}

func TestRepositoryUpsertFaultRejectsGuideManualMismatch(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-conflict")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B7")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	_, err := repo.UpsertFault(user, inspection, sampleFault(inspection.ID))
	require.NoError(t, err)

	mismatched := inspection
	mismatched.GuideManual = "pmcs_sbs/hmmwv/other.json"
	_, err = repo.UpsertFault(user, mismatched, sampleFault(inspection.ID))

	require.ErrorIs(t, err, pmcs_sbs_progress.ErrInspectionConflict)
}

func TestRepositoryGetInspectionReturnsFaultsOrderedBySectionAndItem(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-get-order")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B8")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	late := sampleFault(inspection.ID)
	late.SectionID = "during"
	late.ItemIndex = 2
	_, err := repo.UpsertFault(user, inspection, late)
	require.NoError(t, err)

	early := sampleFault(inspection.ID)
	early.SectionID = "before"
	early.ItemIndex = 0
	_, err = repo.UpsertFault(user, inspection, early)
	require.NoError(t, err)

	_, faults, err := repo.GetInspection(user, vehicleID, inspection.ID)
	require.NoError(t, err)
	require.Len(t, faults, 2)
	require.Equal(t, "before", faults[0].SectionID)
	require.Equal(t, "during", faults[1].SectionID)
}

func TestRepositoryGetInspectionReturnsCleanInspectionWithEmptyFaults(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-get-clean")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B9")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	_, err := repo.EnsureInspection(user, inspection)
	require.NoError(t, err)

	fetched, faults, err := repo.GetInspection(user, vehicleID, inspection.ID)
	require.NoError(t, err)
	require.Equal(t, inspection.ID, fetched.ID)
	require.Empty(t, faults)
}

func TestRepositoryGetInspectionRejectsCrossVehiclePmcsID(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-get-crossveh")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B10")
	otherVehicleID := createShopVehicle(t, testDB, shopID, user, "B11")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	_, err := repo.EnsureInspection(user, inspection)
	require.NoError(t, err)

	_, _, err = repo.GetInspection(user, otherVehicleID, inspection.ID)

	require.ErrorIs(t, err, pmcs_sbs_progress.ErrInspectionNotFound)
}

func TestRepositoryListInspectionsOrdersByPerformedDateDescWithFaultCounts(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-list-order")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B12")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	older := sampleInspection(vehicleID, user.UserID)
	older.PerformedDate = time.Now().UTC().Add(-48 * time.Hour)
	_, err := repo.EnsureInspection(user, older)
	require.NoError(t, err)

	newer := sampleInspection(vehicleID, user.UserID)
	newer.PerformedDate = time.Now().UTC()
	_, err = repo.UpsertFault(user, newer, sampleFault(newer.ID))
	require.NoError(t, err)

	summaries, err := repo.ListInspections(user, vehicleID, "", 10, 0)
	require.NoError(t, err)
	require.Len(t, summaries, 2)
	require.Equal(t, newer.ID, summaries[0].ID)
	require.Equal(t, 1, summaries[0].FaultCount)
	require.Equal(t, older.ID, summaries[1].ID)
	require.Equal(t, 0, summaries[1].FaultCount)
}

func TestRepositoryListInspectionsFiltersByGuideManual(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-list-filter")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B13")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	first := sampleInspection(vehicleID, user.UserID)
	first.GuideManual = "pmcs_sbs/hmmwv/first.json"
	_, err := repo.EnsureInspection(user, first)
	require.NoError(t, err)

	second := sampleInspection(vehicleID, user.UserID)
	second.GuideManual = "pmcs_sbs/hmmwv/second.json"
	_, err = repo.EnsureInspection(user, second)
	require.NoError(t, err)

	summaries, err := repo.ListInspections(user, vehicleID, "pmcs_sbs/hmmwv/first.json", 10, 0)
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.Equal(t, first.ID, summaries[0].ID)
}

func TestRepositoryDeleteInspectionCascadesFaultsButNotSiblings(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-delete-cascade")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B14")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	toDelete := sampleInspection(vehicleID, user.UserID)
	_, err := repo.UpsertFault(user, toDelete, sampleFault(toDelete.ID))
	require.NoError(t, err)

	sibling := sampleInspection(vehicleID, user.UserID)
	_, err = repo.UpsertFault(user, sibling, sampleFault(sibling.ID))
	require.NoError(t, err)

	err = repo.DeleteInspection(user, vehicleID, toDelete.ID)
	require.NoError(t, err)

	var faultCount int
	err = testDB.QueryRow(`SELECT COUNT(*) FROM pmcs_sbs_faults WHERE pmcs_id=$1`, toDelete.ID).Scan(&faultCount)
	require.NoError(t, err)
	require.Equal(t, 0, faultCount)

	_, siblingFaults, err := repo.GetInspection(user, vehicleID, sibling.ID)
	require.NoError(t, err)
	require.Len(t, siblingFaults, 1)
}

func TestRepositoryDeleteFaultAndBulkDeleteFaultsScopedToInspection(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-delete-fault")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B15")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	first := sampleFault(inspection.ID)
	first.SectionID = "before"
	first.ItemIndex = 0
	_, err := repo.UpsertFault(user, inspection, first)
	require.NoError(t, err)

	second := sampleFault(inspection.ID)
	second.SectionID = "during"
	second.ItemIndex = 1
	_, err = repo.UpsertFault(user, inspection, second)
	require.NoError(t, err)

	err = repo.DeleteFault(user, vehicleID, pmcs_sbs_progress.FaultKey{PmcsID: inspection.ID, SectionID: "before", ItemIndex: 0})
	require.NoError(t, err)

	deletedCount, err := repo.DeleteFaults(user, vehicleID, inspection.ID, []pmcs_sbs_progress.FaultKey{
		{PmcsID: inspection.ID, SectionID: "during", ItemIndex: 1},
		{PmcsID: inspection.ID, SectionID: "missing", ItemIndex: 99},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), deletedCount)

	_, faults, err := repo.GetInspection(user, vehicleID, inspection.ID)
	require.NoError(t, err)
	require.Empty(t, faults)
}

func TestRepositoryVehicleDeleteCascadesInspectionsAndFaults(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-vehicle-cascade")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B16")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	_, err := repo.UpsertFault(user, inspection, sampleFault(inspection.ID))
	require.NoError(t, err)

	_, err = testDB.Exec(`DELETE FROM shop_vehicle WHERE id=$1`, vehicleID)
	require.NoError(t, err)

	var inspectionCount, faultCount int
	err = testDB.QueryRow(`SELECT COUNT(*) FROM pmcs_sbs_inspections WHERE equipment_id=$1`, vehicleID).Scan(&inspectionCount)
	require.NoError(t, err)
	require.Equal(t, 0, inspectionCount)

	err = testDB.QueryRow(`SELECT COUNT(*) FROM pmcs_sbs_faults WHERE pmcs_id=$1`, inspection.ID).Scan(&faultCount)
	require.NoError(t, err)
	require.Equal(t, 0, faultCount)
}
```

- [ ] **Step 6: Run the repository tests to confirm they fail to compile**

```bash
go test ./tests/pmcs_sbs_progress/... -run TestRepository -v
```

Expected: FAIL — build error, `pmcs_sbs_progress.NewRepository(testDB)` does not yet implement `EnsureInspection`/`GetInspection`/`ListInspections`/`DeleteInspection`, and `UpsertFault`/`DeleteFault`/`DeleteFaults` still have the old signatures.

- [ ] **Step 7: Rewrite `repository_impl.go`**

Replace the entire file:

```go
package pmcs_sbs_progress

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/bootstrap"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/google/uuid"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) EnsureInspection(user *bootstrap.User, inspection model.PmcsSbsInspections) (*model.PmcsSbsInspections, error) {
	if err := repo.requireVehicleAccess(user, inspection.EquipmentID); err != nil {
		return nil, err
	}
	return ensureInspection(repo.db, inspection)
}

func (repo *RepositoryImpl) GetInspection(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID) (*model.PmcsSbsInspections, []model.PmcsSbsFaults, error) {
	if err := repo.requireVehicleAccess(user, equipmentID); err != nil {
		return nil, nil, err
	}

	var inspection model.PmcsSbsInspections
	stmt := SELECT(PmcsSbsInspections.AllColumns).
		FROM(PmcsSbsInspections).
		WHERE(
			PmcsSbsInspections.ID.EQ(UUID(pmcsID)).
				AND(PmcsSbsInspections.EquipmentID.EQ(String(equipmentID))),
		)

	if err := stmt.Query(repo.db, &inspection); err != nil {
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

	return &inspection, faults, nil
}

func (repo *RepositoryImpl) ListInspections(user *bootstrap.User, equipmentID string, guideManual string, limit int, offset int) ([]InspectionSummary, error) {
	if err := repo.requireVehicleAccess(user, equipmentID); err != nil {
		return nil, err
	}

	condition := PmcsSbsInspections.EquipmentID.EQ(String(equipmentID))
	if guideManual != "" {
		condition = condition.AND(PmcsSbsInspections.GuideManual.EQ(String(guideManual)))
	}

	var inspections []model.PmcsSbsInspections
	stmt := SELECT(PmcsSbsInspections.AllColumns).
		FROM(PmcsSbsInspections).
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
		PmcsSbsFaults.PmcsID,
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
			ID:            inspection.ID,
			GuideManual:   inspection.GuideManual,
			PerformedDate: inspection.PerformedDate,
			FaultCount:    countByID[inspection.ID],
			CreatedAt:     inspection.CreatedAt,
		})
	}
	return summaries, nil
}

func (repo *RepositoryImpl) DeleteInspection(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID) error {
	if err := repo.requireVehicleAccess(user, equipmentID); err != nil {
		return err
	}

	result, err := PmcsSbsInspections.DELETE().
		WHERE(
			PmcsSbsInspections.ID.EQ(UUID(pmcsID)).
				AND(PmcsSbsInspections.EquipmentID.EQ(String(equipmentID))),
		).
		Exec(repo.db)
	if err != nil {
		return fmt.Errorf("delete pmcs sbs inspection: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete pmcs sbs inspection rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrInspectionNotFound
	}
	return nil
}

func (repo *RepositoryImpl) UpsertFault(user *bootstrap.User, inspection model.PmcsSbsInspections, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error) {
	if err := repo.requireVehicleAccess(user, inspection.EquipmentID); err != nil {
		return nil, err
	}

	tx, err := repo.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin upsert pmcs sbs fault transaction: %w", err)
	}
	defer tx.Rollback()

	savedInspection, err := ensureInspection(tx, inspection)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	fault.PmcsID = savedInspection.ID
	if fault.CreatedAt.IsZero() {
		fault.CreatedAt = now
	}
	fault.UpdatedAt = now

	stmt := PmcsSbsFaults.INSERT(
		PmcsSbsFaults.PmcsID,
		PmcsSbsFaults.SectionID,
		PmcsSbsFaults.ItemIndex,
		PmcsSbsFaults.ItemNo,
		PmcsSbsFaults.Status,
		PmcsSbsFaults.FaultText,
		PmcsSbsFaults.CorrectiveAction,
		PmcsSbsFaults.CreatedAt,
		PmcsSbsFaults.UpdatedAt,
	).VALUES(
		UUID(fault.PmcsID),
		String(fault.SectionID),
		Int32(fault.ItemIndex),
		String(fault.ItemNo),
		String(fault.Status),
		String(fault.FaultText),
		String(fault.CorrectiveAction),
		TimestampzT(fault.CreatedAt),
		TimestampzT(now),
	).ON_CONFLICT(
		PmcsSbsFaults.PmcsID,
		PmcsSbsFaults.SectionID,
		PmcsSbsFaults.ItemIndex,
	).DO_UPDATE(SET(
		PmcsSbsFaults.ItemNo.SET(String(fault.ItemNo)),
		PmcsSbsFaults.Status.SET(String(fault.Status)),
		PmcsSbsFaults.FaultText.SET(String(fault.FaultText)),
		PmcsSbsFaults.CorrectiveAction.SET(String(fault.CorrectiveAction)),
		PmcsSbsFaults.UpdatedAt.SET(TimestampzT(now)),
	)).RETURNING(PmcsSbsFaults.AllColumns)

	var saved model.PmcsSbsFaults
	if err := stmt.Query(tx, &saved); err != nil {
		return nil, fmt.Errorf("upsert pmcs sbs fault: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit upsert pmcs sbs fault transaction: %w", err)
	}
	return &saved, nil
}

func (repo *RepositoryImpl) DeleteFault(user *bootstrap.User, equipmentID string, key FaultKey) error {
	if err := repo.requireVehicleAccess(user, equipmentID); err != nil {
		return err
	}
	if err := repo.requireInspectionOwnership(repo.db, equipmentID, key.PmcsID); err != nil {
		return err
	}

	if _, err := PmcsSbsFaults.DELETE().
		WHERE(
			PmcsSbsFaults.PmcsID.EQ(UUID(key.PmcsID)).
				AND(PmcsSbsFaults.SectionID.EQ(String(key.SectionID))).
				AND(PmcsSbsFaults.ItemIndex.EQ(Int32(key.ItemIndex))),
		).
		Exec(repo.db); err != nil {
		return fmt.Errorf("delete pmcs sbs fault: %w", err)
	}
	return nil
}

func (repo *RepositoryImpl) DeleteFaults(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID, keys []FaultKey) (int64, error) {
	if err := repo.requireVehicleAccess(user, equipmentID); err != nil {
		return 0, err
	}
	if err := repo.requireInspectionOwnership(repo.db, equipmentID, pmcsID); err != nil {
		return 0, err
	}
	if len(keys) == 0 {
		return 0, nil
	}

	keyRows := make([]Expression, 0, len(keys))
	for _, key := range keys {
		keyRows = append(keyRows, ROW(String(key.SectionID), Int32(key.ItemIndex)))
	}

	result, err := PmcsSbsFaults.DELETE().
		WHERE(
			PmcsSbsFaults.PmcsID.EQ(UUID(pmcsID)).
				AND(ROW(PmcsSbsFaults.SectionID, PmcsSbsFaults.ItemIndex).IN(keyRows...)),
		).
		Exec(repo.db)
	if err != nil {
		return 0, fmt.Errorf("bulk delete pmcs sbs faults: %w", err)
	}
	deletedCount, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("bulk delete pmcs sbs faults rows affected: %w", err)
	}
	return deletedCount, nil
}

func (repo *RepositoryImpl) requireVehicleAccess(user *bootstrap.User, equipmentID string) error {
	if user == nil {
		return ErrUnauthorized
	}

	stmt := SELECT(Int(1).AS("exists")).
		FROM(
			ShopVehicle.
				INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(ShopVehicle.ShopID)),
		).
		WHERE(
			ShopVehicle.ID.EQ(String(equipmentID)).
				AND(ShopMembers.UserID.EQ(String(user.UserID))),
		).
		LIMIT(1)

	var rows []struct {
		Exists int `sql:"exists"`
	}
	if err := stmt.Query(repo.db, &rows); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("authorize pmcs sbs vehicle fault access: %w", err)
	}
	if len(rows) == 0 {
		return ErrNotFound
	}
	return nil
}

func (repo *RepositoryImpl) requireInspectionOwnership(queryable qrm.Queryable, equipmentID string, pmcsID uuid.UUID) error {
	stmt := SELECT(Int(1).AS("exists")).
		FROM(PmcsSbsInspections).
		WHERE(
			PmcsSbsInspections.ID.EQ(UUID(pmcsID)).
				AND(PmcsSbsInspections.EquipmentID.EQ(String(equipmentID))),
		).
		LIMIT(1)

	var rows []struct {
		Exists int `sql:"exists"`
	}
	if err := stmt.Query(queryable, &rows); err != nil {
		return fmt.Errorf("authorize pmcs sbs inspection access: %w", err)
	}
	if len(rows) == 0 {
		return ErrInspectionNotFound
	}
	return nil
}

// ensureInspection inserts the inspection if it doesn't exist yet, or, if a
// row with this id already exists, verifies equipment_id and guide_manual
// match and updates performed_date. A mismatch on either field returns
// ErrInspectionConflict. queryable is either *sql.DB (standalone calls) or
// *sql.Tx (the implicit-creation path inside UpsertFault) — both satisfy
// qrm.Queryable.
func ensureInspection(queryable qrm.Queryable, inspection model.PmcsSbsInspections) (*model.PmcsSbsInspections, error) {
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
		SET(
			PmcsSbsInspections.PerformedDate.SET(TimestampzT(inspection.PerformedDate)),
			PmcsSbsInspections.UpdatedAt.SET(TimestampzT(now)),
		).WHERE(
			PmcsSbsInspections.EquipmentID.EQ(String(inspection.EquipmentID)).
				AND(PmcsSbsInspections.GuideManual.EQ(String(inspection.GuideManual))),
		),
	).RETURNING(PmcsSbsInspections.AllColumns)

	var saved model.PmcsSbsInspections
	if err := stmt.Query(queryable, &saved); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrInspectionConflict
		}
		return nil, fmt.Errorf("ensure pmcs sbs inspection: %w", err)
	}
	return &saved, nil
}
```

- [ ] **Step 8: Run the repository tests again**

```bash
go test ./tests/pmcs_sbs_progress/... -v
```

Expected: PASS — all tests listed in Step 5.

- [ ] **Step 9: Commit**

```bash
git add api/pmcs_sbs_progress/types.go api/pmcs_sbs_progress/errors.go api/pmcs_sbs_progress/repository.go api/pmcs_sbs_progress/repository_impl.go tests/pmcs_sbs_progress/helpers_test.go tests/pmcs_sbs_progress/repository_test.go
git commit -m "feat(pmcs-sbs): rewrite repository layer around pmcs_sbs_inspections"
```

---

### Task 3: Service Layer — Validation and Mapping

**Files:**
- Modify: `api/pmcs_sbs_progress/service.go`
- Modify: `api/pmcs_sbs_progress/service_impl.go`
- Modify: `api/pmcs_sbs_progress/service_impl_test.go`

**Interfaces:**
- Consumes: `Repository` interface, `FaultKey`, `InspectionSummary`, error vars from Task 2; `model.PmcsSbsInspections`, `model.PmcsSbsFaults` from Task 1.
- Produces: `Service` interface (`EnsureInspection`, `GetInspection`, `ListInspections`, `DeleteInspection`, `UpsertFault`, `DeleteFault`, `DeleteFaults`) with signatures `(user *bootstrap.User, equipmentID string, pmcsID string, ...)` (pmcs id as a raw string — parsed/validated inside the service) — Task 4 (route layer) calls these directly with `c.Param("pmcs_id")` unparsed.

- [ ] **Step 1: Write the new `api/pmcs_sbs_progress/service_impl_test.go`**

Replace the entire file:

```go
package pmcs_sbs_progress

import (
	"errors"
	"testing"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type repoStub struct {
	inspection    *model.PmcsSbsInspections
	faults        []model.PmcsSbsFaults
	summaries     []InspectionSummary
	savedFault    *model.PmcsSbsFaults
	deletedCount  int64
	err           error

	capturedUser        *bootstrap.User
	capturedEquipmentID string
	capturedPmcsID      uuid.UUID
	capturedGuideManual string
	capturedLimit       int
	capturedOffset      int
	capturedInspection  model.PmcsSbsInspections
	capturedFault       model.PmcsSbsFaults
	capturedDelete      FaultKey
	capturedBulkKeys    []FaultKey
}

func (repo *repoStub) EnsureInspection(user *bootstrap.User, inspection model.PmcsSbsInspections) (*model.PmcsSbsInspections, error) {
	repo.capturedUser = user
	repo.capturedInspection = inspection
	if repo.inspection != nil {
		return repo.inspection, repo.err
	}
	return &inspection, repo.err
}

func (repo *repoStub) GetInspection(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID) (*model.PmcsSbsInspections, []model.PmcsSbsFaults, error) {
	repo.capturedUser = user
	repo.capturedEquipmentID = equipmentID
	repo.capturedPmcsID = pmcsID
	return repo.inspection, repo.faults, repo.err
}

func (repo *repoStub) ListInspections(user *bootstrap.User, equipmentID string, guideManual string, limit int, offset int) ([]InspectionSummary, error) {
	repo.capturedUser = user
	repo.capturedEquipmentID = equipmentID
	repo.capturedGuideManual = guideManual
	repo.capturedLimit = limit
	repo.capturedOffset = offset
	return repo.summaries, repo.err
}

func (repo *repoStub) DeleteInspection(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID) error {
	repo.capturedUser = user
	repo.capturedEquipmentID = equipmentID
	repo.capturedPmcsID = pmcsID
	return repo.err
}

func (repo *repoStub) UpsertFault(user *bootstrap.User, inspection model.PmcsSbsInspections, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error) {
	repo.capturedUser = user
	repo.capturedInspection = inspection
	repo.capturedFault = fault
	if repo.savedFault != nil {
		return repo.savedFault, repo.err
	}
	return &fault, repo.err
}

func (repo *repoStub) DeleteFault(user *bootstrap.User, equipmentID string, key FaultKey) error {
	repo.capturedUser = user
	repo.capturedEquipmentID = equipmentID
	repo.capturedDelete = key
	return repo.err
}

func (repo *repoStub) DeleteFaults(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID, keys []FaultKey) (int64, error) {
	repo.capturedUser = user
	repo.capturedEquipmentID = equipmentID
	repo.capturedPmcsID = pmcsID
	repo.capturedBulkKeys = keys
	return repo.deletedCount, repo.err
}

func requireUser() *bootstrap.User {
	return &bootstrap.User{UserID: "user-1", Email: "user-1@example.com", Username: "user-1"}
}

func requireServiceError(t *testing.T, err error, target error) {
	t.Helper()
	require.Error(t, err)
	require.Truef(t, errors.Is(err, target), "expected %v, got %v", target, err)
}

const samplePmcsIDStr = "11111111-1111-1111-1111-111111111111"

func samplePmcsID() uuid.UUID {
	return uuid.MustParse(samplePmcsIDStr)
}

func TestEnsureInspectionRequiresAuth(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.EnsureInspection(nil, "vehicle-1", samplePmcsIDStr, InspectionRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: time.Now()})

	requireServiceError(t, err, ErrUnauthorized)
}

func TestEnsureInspectionRejectsInvalidValues(t *testing.T) {
	svc := NewService(&repoStub{})
	now := time.Now()

	cases := []struct {
		name        string
		equipmentID string
		pmcsID      string
		req         InspectionRequest
		want        error
	}{
		{name: "blank equipment", equipmentID: " ", pmcsID: samplePmcsIDStr, req: InspectionRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: now}, want: ErrInvalidID},
		{name: "malformed pmcs id", equipmentID: "vehicle-1", pmcsID: "not-a-uuid", req: InspectionRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: now}, want: ErrInvalidPmcsID},
		{name: "blank pmcs id", equipmentID: "vehicle-1", pmcsID: " ", req: InspectionRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: now}, want: ErrInvalidPmcsID},
		{name: "invalid guide manual", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, req: InspectionRequest{GuideManual: "pmcs/hmmwv/file.json", PerformedDate: now}, want: ErrInvalidGuideManual},
		{name: "zero performed date", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, req: InspectionRequest{GuideManual: "pmcs_sbs/hmmwv/file.json"}, want: ErrInvalidRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.EnsureInspection(requireUser(), tc.equipmentID, tc.pmcsID, tc.req)
			requireServiceError(t, err, tc.want)
		})
	}
}

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

func TestGetInspectionRejectsInvalidPmcsID(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.GetInspection(requireUser(), "vehicle-1", "not-a-uuid")

	requireServiceError(t, err, ErrInvalidPmcsID)
}

func TestGetInspectionMapsFaults(t *testing.T) {
	now := time.Now().UTC()
	stub := &repoStub{
		inspection: &model.PmcsSbsInspections{ID: samplePmcsID(), EquipmentID: "vehicle-1", GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: now},
		faults: []model.PmcsSbsFaults{{
			PmcsID: samplePmcsID(), SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "x", FaultText: "leak", CreatedAt: now, UpdatedAt: now,
		}},
	}
	svc := NewService(stub)

	resp, err := svc.GetInspection(requireUser(), "vehicle-1", samplePmcsIDStr)

	require.NoError(t, err)
	require.Equal(t, samplePmcsID(), stub.capturedPmcsID)
	require.Len(t, resp.Faults, 1)
	require.Equal(t, "leak", resp.Faults[0].FaultText)
}

func TestListInspectionsAppliesDefaultLimitAndOffset(t *testing.T) {
	stub := &repoStub{summaries: []InspectionSummary{}}
	svc := NewService(stub)

	_, err := svc.ListInspections(requireUser(), "vehicle-1", ListInspectionsRequest{})

	require.NoError(t, err)
	require.Equal(t, 1000, stub.capturedLimit)
	require.Equal(t, 0, stub.capturedOffset)
}

func TestListInspectionsValidatesGuideManualFilterWhenProvided(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.ListInspections(requireUser(), "vehicle-1", ListInspectionsRequest{GuideManual: "pmcs/hmmwv/file.json"})

	requireServiceError(t, err, ErrInvalidGuideManual)
}

func TestListInspectionsMapsSummaries(t *testing.T) {
	now := time.Now().UTC()
	stub := &repoStub{summaries: []InspectionSummary{
		{ID: samplePmcsID(), GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: now, FaultCount: 2, CreatedAt: now},
	}}
	svc := NewService(stub)

	resp, err := svc.ListInspections(requireUser(), "vehicle-1", ListInspectionsRequest{Limit: 10, Offset: 0})

	require.NoError(t, err)
	require.Equal(t, 1, resp.Count)
	require.Equal(t, 2, resp.Inspections[0].FaultCount)
}

func TestDeleteInspectionValidatesPmcsID(t *testing.T) {
	svc := NewService(&repoStub{})

	err := svc.DeleteInspection(requireUser(), "vehicle-1", "not-a-uuid")

	requireServiceError(t, err, ErrInvalidPmcsID)
}

func TestDeleteInspectionPassesParsedID(t *testing.T) {
	stub := &repoStub{}
	svc := NewService(stub)

	err := svc.DeleteInspection(requireUser(), " vehicle-1 ", samplePmcsIDStr)

	require.NoError(t, err)
	require.Equal(t, "vehicle-1", stub.capturedEquipmentID)
	require.Equal(t, samplePmcsID(), stub.capturedPmcsID)
}

func TestUpsertFaultRejectsInvalidValues(t *testing.T) {
	svc := NewService(&repoStub{})
	baseReq := func() FaultRequest {
		return FaultRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: time.Now(), SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "X", FaultText: "leak"}
	}

	cases := []struct {
		name        string
		equipmentID string
		pmcsID      string
		mutate      func(FaultRequest) FaultRequest
		want        error
	}{
		{name: "blank equipment", equipmentID: " ", pmcsID: samplePmcsIDStr, mutate: func(r FaultRequest) FaultRequest { return r }, want: ErrInvalidID},
		{name: "malformed pmcs id", equipmentID: "vehicle-1", pmcsID: "bad", mutate: func(r FaultRequest) FaultRequest { return r }, want: ErrInvalidPmcsID},
		{name: "invalid guide manual", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, mutate: func(r FaultRequest) FaultRequest { r.GuideManual = "pmcs_sbs/../file.json"; return r }, want: ErrInvalidGuideManual},
		{name: "blank section", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, mutate: func(r FaultRequest) FaultRequest { r.SectionID = " "; return r }, want: ErrInvalidRequest},
		{name: "negative item index", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, mutate: func(r FaultRequest) FaultRequest { r.ItemIndex = -1; return r }, want: ErrInvalidRequest},
		{name: "blank item no", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, mutate: func(r FaultRequest) FaultRequest { r.ItemNo = " "; return r }, want: ErrInvalidRequest},
		{name: "invalid status", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, mutate: func(r FaultRequest) FaultRequest { r.Status = "BAD"; return r }, want: ErrInvalidStatus},
		{name: "blank fault text", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, mutate: func(r FaultRequest) FaultRequest { r.FaultText = " "; return r }, want: ErrInvalidRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.UpsertFault(requireUser(), tc.equipmentID, tc.pmcsID, tc.mutate(baseReq()))
			requireServiceError(t, err, tc.want)
		})
	}
}

func TestUpsertFaultAcceptsAllowedStatuses(t *testing.T) {
	stub := &repoStub{}
	svc := NewService(stub)
	cases := []struct {
		input string
		want  string
	}{
		{input: "X", want: "x"},
		{input: "x", want: "x"},
		{input: "/", want: "slash"},
		{input: "slash", want: "slash"},
		{input: "-", want: "dash"},
		{input: "dash", want: "dash"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			_, err := svc.UpsertFault(requireUser(), "vehicle-1", samplePmcsIDStr, FaultRequest{
				GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: time.Now(), SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: tc.input, FaultText: "leak",
			})
			require.NoError(t, err)
			require.Equal(t, tc.want, stub.capturedFault.Status)
		})
	}
}

func TestUpsertFaultReturnsMappedResponse(t *testing.T) {
	now := time.Now().UTC()
	stub := &repoStub{savedFault: &model.PmcsSbsFaults{
		PmcsID: samplePmcsID(), SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "x", FaultText: "leak", CreatedAt: now, UpdatedAt: now,
	}}
	svc := NewService(stub)

	resp, err := svc.UpsertFault(requireUser(), "vehicle-1", samplePmcsIDStr, FaultRequest{
		GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: now, SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "X", FaultText: "leak",
	})

	require.NoError(t, err)
	require.Equal(t, "vehicle-1", stub.capturedInspection.EquipmentID)
	require.Equal(t, samplePmcsID(), stub.capturedInspection.ID)
	require.Equal(t, "x", stub.capturedFault.Status)
	require.Equal(t, samplePmcsID(), resp.PmcsID)
}

func TestDeleteFaultPassesValidatedKey(t *testing.T) {
	stub := &repoStub{}
	svc := NewService(stub)

	err := svc.DeleteFault(requireUser(), " vehicle-1 ", samplePmcsIDStr, DeleteFaultRequest{SectionID: " before ", ItemIndex: 0})

	require.NoError(t, err)
	require.Equal(t, "vehicle-1", stub.capturedEquipmentID)
	require.Equal(t, samplePmcsID(), stub.capturedDelete.PmcsID)
	require.Equal(t, "before", stub.capturedDelete.SectionID)
}

func TestDeleteFaultsPassesValidatedKeysAndCounts(t *testing.T) {
	stub := &repoStub{deletedCount: 1}
	svc := NewService(stub)

	resp, err := svc.DeleteFaults(requireUser(), " vehicle-1 ", samplePmcsIDStr, BulkDeleteFaultRequest{
		Faults: []BulkDeleteFaultItemRequest{
			{SectionID: " before ", ItemIndex: 0},
			{SectionID: " after ", ItemIndex: 2},
		},
	})

	require.NoError(t, err)
	require.Equal(t, "vehicle-1", stub.capturedEquipmentID)
	require.Equal(t, []FaultKey{
		{PmcsID: samplePmcsID(), SectionID: "before", ItemIndex: 0},
		{PmcsID: samplePmcsID(), SectionID: "after", ItemIndex: 2},
	}, stub.capturedBulkKeys)
	require.Equal(t, 2, resp.RequestedCount)
	require.Equal(t, 1, resp.DeletedCount)
}

func TestDeleteFaultsRequiresAuth(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.DeleteFaults(nil, "vehicle-1", samplePmcsIDStr, BulkDeleteFaultRequest{
		Faults: []BulkDeleteFaultItemRequest{{SectionID: "before", ItemIndex: 0}},
	})

	requireServiceError(t, err, ErrUnauthorized)
}

func TestValidateBulkDeleteFaultRequestRejectsInvalidValues(t *testing.T) {
	svc := NewService(&repoStub{})
	validFaults := []BulkDeleteFaultItemRequest{{SectionID: "before", ItemIndex: 0}}
	tooManyFaults := make([]BulkDeleteFaultItemRequest, maxBulkDeleteFaults+1)
	for i := range tooManyFaults {
		tooManyFaults[i] = BulkDeleteFaultItemRequest{SectionID: "before", ItemIndex: int32(i)}
	}

	cases := []struct {
		name        string
		equipmentID string
		pmcsID      string
		req         BulkDeleteFaultRequest
		want        error
	}{
		{name: "blank equipment", equipmentID: " ", pmcsID: samplePmcsIDStr, req: BulkDeleteFaultRequest{Faults: validFaults}, want: ErrInvalidID},
		{name: "malformed pmcs id", equipmentID: "vehicle-1", pmcsID: "bad", req: BulkDeleteFaultRequest{Faults: validFaults}, want: ErrInvalidPmcsID},
		{name: "empty faults", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, req: BulkDeleteFaultRequest{Faults: []BulkDeleteFaultItemRequest{}}, want: ErrInvalidRequest},
		{name: "too many faults", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, req: BulkDeleteFaultRequest{Faults: tooManyFaults}, want: ErrInvalidRequest},
		{name: "blank section", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, req: BulkDeleteFaultRequest{Faults: []BulkDeleteFaultItemRequest{{SectionID: " ", ItemIndex: 0}}}, want: ErrInvalidRequest},
		{name: "negative item index", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, req: BulkDeleteFaultRequest{Faults: []BulkDeleteFaultItemRequest{{SectionID: "before", ItemIndex: -1}}}, want: ErrInvalidRequest},
		{name: "duplicate key", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, req: BulkDeleteFaultRequest{Faults: []BulkDeleteFaultItemRequest{{SectionID: " before ", ItemIndex: 0}, {SectionID: "before", ItemIndex: 0}}}, want: ErrInvalidRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := svc.validateBulkDeleteFaultRequest(tc.equipmentID, tc.pmcsID, tc.req)
			requireServiceError(t, err, tc.want)
		})
	}
}
```

- [ ] **Step 2: Run the service tests to confirm they fail to compile**

```bash
go test ./api/pmcs_sbs_progress/... -run "TestEnsureInspection|TestGetInspection|TestListInspections|TestDeleteInspection|TestUpsertFault|TestDeleteFault|TestValidateBulkDeleteFaultRequest" -v
```

Expected: FAIL — build error, `Service`/`NewService` do not yet have the new methods.

- [ ] **Step 3: Update `service.go`**

Replace the entire file:

```go
package pmcs_sbs_progress

import "miltechserver/bootstrap"

type Service interface {
	EnsureInspection(user *bootstrap.User, equipmentID string, pmcsID string, req InspectionRequest) (*InspectionResponse, error)
	GetInspection(user *bootstrap.User, equipmentID string, pmcsID string) (*InspectionResponse, error)
	ListInspections(user *bootstrap.User, equipmentID string, req ListInspectionsRequest) (*InspectionListResponse, error)
	DeleteInspection(user *bootstrap.User, equipmentID string, pmcsID string) error

	UpsertFault(user *bootstrap.User, equipmentID string, pmcsID string, req FaultRequest) (*FaultResponse, error)
	DeleteFault(user *bootstrap.User, equipmentID string, pmcsID string, req DeleteFaultRequest) error
	DeleteFaults(user *bootstrap.User, equipmentID string, pmcsID string, req BulkDeleteFaultRequest) (*BulkDeleteFaultResponse, error)
}
```

- [ ] **Step 4: Rewrite `service_impl.go`**

Replace the entire file:

```go
package pmcs_sbs_progress

import (
	"fmt"
	"path"
	"strings"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"

	"github.com/google/uuid"
)

type ServiceImpl struct {
	repository Repository
}

func NewService(repository Repository) *ServiceImpl {
	return &ServiceImpl{repository: repository}
}

const maxBulkDeleteFaults = 100
const defaultListInspectionsLimit = 1000

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
	resp := mapInspection(*saved, nil)
	return &resp, nil
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

	inspection, faults, err := service.repository.GetInspection(user, trimmedEquipmentID, parsedPmcsID)
	if err != nil {
		return nil, err
	}
	resp := mapInspection(*inspection, faults)
	return &resp, nil
}

func (service *ServiceImpl) ListInspections(user *bootstrap.User, equipmentID string, req ListInspectionsRequest) (*InspectionListResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return nil, err
	}

	guideManual := strings.TrimSpace(req.GuideManual)
	if guideManual != "" {
		guideManual, err = validateGuideManual(guideManual)
		if err != nil {
			return nil, err
		}
	}

	limit := req.Limit
	if limit <= 0 {
		limit = defaultListInspectionsLimit
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	summaries, err := service.repository.ListInspections(user, trimmedEquipmentID, guideManual, limit, offset)
	if err != nil {
		return nil, err
	}

	responses := make([]InspectionSummaryResponse, 0, len(summaries))
	for _, summary := range summaries {
		responses = append(responses, InspectionSummaryResponse{
			ID:            summary.ID,
			GuideManual:   summary.GuideManual,
			PerformedDate: summary.PerformedDate,
			FaultCount:    summary.FaultCount,
			CreatedAt:     summary.CreatedAt,
		})
	}
	return &InspectionListResponse{Inspections: responses, Count: len(responses)}, nil
}

func (service *ServiceImpl) DeleteInspection(user *bootstrap.User, equipmentID string, pmcsID string) error {
	if !hasAuthenticatedUser(user) {
		return ErrUnauthorized
	}
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return err
	}
	parsedPmcsID, err := validatePmcsID(pmcsID)
	if err != nil {
		return err
	}
	return service.repository.DeleteInspection(user, trimmedEquipmentID, parsedPmcsID)
}

func (service *ServiceImpl) UpsertFault(user *bootstrap.User, equipmentID string, pmcsID string, req FaultRequest) (*FaultResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	inspection, fault, err := service.validateFaultRequest(equipmentID, pmcsID, user.UserID, req)
	if err != nil {
		return nil, err
	}
	saved, err := service.repository.UpsertFault(user, inspection, fault)
	if err != nil {
		return nil, err
	}
	resp := mapFault(*saved)
	return &resp, nil
}

func (service *ServiceImpl) DeleteFault(user *bootstrap.User, equipmentID string, pmcsID string, req DeleteFaultRequest) error {
	if !hasAuthenticatedUser(user) {
		return ErrUnauthorized
	}
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return err
	}
	key, err := service.validateDeleteFaultRequest(pmcsID, req)
	if err != nil {
		return err
	}
	return service.repository.DeleteFault(user, trimmedEquipmentID, key)
}

func (service *ServiceImpl) DeleteFaults(user *bootstrap.User, equipmentID string, pmcsID string, req BulkDeleteFaultRequest) (*BulkDeleteFaultResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	trimmedEquipmentID, parsedPmcsID, keys, err := service.validateBulkDeleteFaultRequest(equipmentID, pmcsID, req)
	if err != nil {
		return nil, err
	}
	deletedCount, err := service.repository.DeleteFaults(user, trimmedEquipmentID, parsedPmcsID, keys)
	if err != nil {
		return nil, err
	}
	return &BulkDeleteFaultResponse{RequestedCount: len(keys), DeletedCount: int(deletedCount)}, nil
}

func (service *ServiceImpl) validateInspectionRequest(equipmentID string, pmcsID string, userID string, req InspectionRequest) (model.PmcsSbsInspections, error) {
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return model.PmcsSbsInspections{}, err
	}
	parsedPmcsID, err := validatePmcsID(pmcsID)
	if err != nil {
		return model.PmcsSbsInspections{}, err
	}
	guideManual, err := validateGuideManual(req.GuideManual)
	if err != nil {
		return model.PmcsSbsInspections{}, err
	}
	if req.PerformedDate.IsZero() {
		return model.PmcsSbsInspections{}, ErrInvalidRequest
	}

	createdBy := strings.TrimSpace(userID)
	return model.PmcsSbsInspections{
		ID:            parsedPmcsID,
		EquipmentID:   trimmedEquipmentID,
		GuideManual:   guideManual,
		PerformedDate: req.PerformedDate.UTC(),
		CreatedBy:     &createdBy,
	}, nil
}

func (service *ServiceImpl) validateFaultRequest(equipmentID string, pmcsID string, userID string, req FaultRequest) (model.PmcsSbsInspections, model.PmcsSbsFaults, error) {
	inspection, err := service.validateInspectionRequest(equipmentID, pmcsID, userID, InspectionRequest{
		GuideManual:   req.GuideManual,
		PerformedDate: req.PerformedDate,
	})
	if err != nil {
		return model.PmcsSbsInspections{}, model.PmcsSbsFaults{}, err
	}

	sectionID := strings.TrimSpace(req.SectionID)
	itemNo := strings.TrimSpace(req.ItemNo)
	status, validStatus := normalizeFaultStatus(req.Status)
	faultText := strings.TrimSpace(req.FaultText)
	if sectionID == "" || itemNo == "" || req.ItemIndex < 0 || faultText == "" {
		return model.PmcsSbsInspections{}, model.PmcsSbsFaults{}, ErrInvalidRequest
	}
	if !validStatus {
		return model.PmcsSbsInspections{}, model.PmcsSbsFaults{}, ErrInvalidStatus
	}

	now := time.Now().UTC()
	fault := model.PmcsSbsFaults{
		PmcsID:           inspection.ID,
		SectionID:        sectionID,
		ItemIndex:        req.ItemIndex,
		ItemNo:           itemNo,
		Status:           status,
		FaultText:        faultText,
		CorrectiveAction: strings.TrimSpace(req.CorrectiveAction),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	return inspection, fault, nil
}

func (service *ServiceImpl) validateDeleteFaultRequest(pmcsID string, req DeleteFaultRequest) (FaultKey, error) {
	parsedPmcsID, err := validatePmcsID(pmcsID)
	if err != nil {
		return FaultKey{}, err
	}
	sectionID := strings.TrimSpace(req.SectionID)
	if sectionID == "" || req.ItemIndex < 0 {
		return FaultKey{}, ErrInvalidRequest
	}
	return FaultKey{PmcsID: parsedPmcsID, SectionID: sectionID, ItemIndex: req.ItemIndex}, nil
}

func (service *ServiceImpl) validateBulkDeleteFaultRequest(equipmentID string, pmcsID string, req BulkDeleteFaultRequest) (string, uuid.UUID, []FaultKey, error) {
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return "", uuid.UUID{}, nil, err
	}
	parsedPmcsID, err := validatePmcsID(pmcsID)
	if err != nil {
		return "", uuid.UUID{}, nil, err
	}
	if len(req.Faults) == 0 || len(req.Faults) > maxBulkDeleteFaults {
		return "", uuid.UUID{}, nil, ErrInvalidRequest
	}
	keys := make([]FaultKey, 0, len(req.Faults))
	seen := make(map[string]struct{}, len(req.Faults))
	for _, fault := range req.Faults {
		sectionID := strings.TrimSpace(fault.SectionID)
		if sectionID == "" || fault.ItemIndex < 0 {
			return "", uuid.UUID{}, nil, ErrInvalidRequest
		}
		duplicateKey := fmt.Sprintf("%s\x00%d", sectionID, fault.ItemIndex)
		if _, exists := seen[duplicateKey]; exists {
			return "", uuid.UUID{}, nil, ErrInvalidRequest
		}
		seen[duplicateKey] = struct{}{}
		keys = append(keys, FaultKey{PmcsID: parsedPmcsID, SectionID: sectionID, ItemIndex: fault.ItemIndex})
	}
	return trimmedEquipmentID, parsedPmcsID, keys, nil
}

func validateEquipmentID(equipmentID string) (string, error) {
	trimmedEquipmentID := strings.TrimSpace(equipmentID)
	if trimmedEquipmentID == "" {
		return "", ErrInvalidID
	}
	return trimmedEquipmentID, nil
}

func validatePmcsID(pmcsID string) (uuid.UUID, error) {
	trimmed := strings.TrimSpace(pmcsID)
	if trimmed == "" {
		return uuid.UUID{}, ErrInvalidPmcsID
	}
	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return uuid.UUID{}, ErrInvalidPmcsID
	}
	return parsed, nil
}

func validateGuideManual(guideManual string) (string, error) {
	trimmedGuideManual := strings.TrimSpace(guideManual)
	if trimmedGuideManual == "" ||
		strings.Contains(trimmedGuideManual, "\\") ||
		!strings.HasPrefix(trimmedGuideManual, "pmcs_sbs/") ||
		!strings.HasSuffix(trimmedGuideManual, ".json") ||
		path.Clean(trimmedGuideManual) != trimmedGuideManual {
		return "", ErrInvalidGuideManual
	}
	return trimmedGuideManual, nil
}

func hasAuthenticatedUser(user *bootstrap.User) bool {
	return user != nil && strings.TrimSpace(user.UserID) != ""
}

func normalizeFaultStatus(status string) (string, bool) {
	switch strings.TrimSpace(status) {
	case "X", "x":
		return "x", true
	case "/", "slash":
		return "slash", true
	case "-", "dash":
		return "dash", true
	default:
		return "", false
	}
}

func mapFault(row model.PmcsSbsFaults) FaultResponse {
	return FaultResponse{
		PmcsID:           row.PmcsID,
		SectionID:        row.SectionID,
		ItemIndex:        row.ItemIndex,
		ItemNo:           row.ItemNo,
		Status:           row.Status,
		FaultText:        row.FaultText,
		CorrectiveAction: row.CorrectiveAction,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

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
		CreatedBy:     row.CreatedBy,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
		Faults:        faults,
	}
}
```

- [ ] **Step 5: Run the service tests again**

```bash
go test ./api/pmcs_sbs_progress/... -run "TestEnsureInspection|TestGetInspection|TestListInspections|TestDeleteInspection|TestUpsertFault|TestDeleteFault|TestValidateBulkDeleteFaultRequest" -v
```

Expected: PASS — all tests from Step 1. (Route-layer tests in the same package will still fail to compile at this point — that's Task 4.)

- [ ] **Step 6: Commit**

```bash
git add api/pmcs_sbs_progress/service.go api/pmcs_sbs_progress/service_impl.go api/pmcs_sbs_progress/service_impl_test.go
git commit -m "feat(pmcs-sbs): rewrite service layer for inspection-scoped faults"
```

---

### Task 4: Route Layer — HTTP Handlers

**Files:**
- Modify: `api/pmcs_sbs_progress/route.go`
- Modify: `api/pmcs_sbs_progress/route_test.go`

**Interfaces:**
- Consumes: `Service` interface from Task 3.
- Produces: seven registered routes under `/pmcs-sbs/equipment/:equipment_id/...` as listed in the design spec's API surface table.

- [ ] **Step 1: Write the new `api/pmcs_sbs_progress/route_test.go`**

Replace the entire file:

```go
package pmcs_sbs_progress

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type serviceStub struct {
	inspectionResp *InspectionResponse
	listResp       *InspectionListResponse
	faultResp      *FaultResponse
	bulkDeleteResp *BulkDeleteFaultResponse
	err            error

	capturedUser        *bootstrap.User
	capturedEquipmentID string
	capturedPmcsID      string
	capturedRequest     interface{}
}

func (s *serviceStub) EnsureInspection(user *bootstrap.User, equipmentID string, pmcsID string, req InspectionRequest) (*InspectionResponse, error) {
	s.capturedUser = user
	s.capturedEquipmentID = equipmentID
	s.capturedPmcsID = pmcsID
	s.capturedRequest = req
	return s.inspectionResp, s.err
}

func (s *serviceStub) GetInspection(user *bootstrap.User, equipmentID string, pmcsID string) (*InspectionResponse, error) {
	s.capturedUser = user
	s.capturedEquipmentID = equipmentID
	s.capturedPmcsID = pmcsID
	return s.inspectionResp, s.err
}

func (s *serviceStub) ListInspections(user *bootstrap.User, equipmentID string, req ListInspectionsRequest) (*InspectionListResponse, error) {
	s.capturedUser = user
	s.capturedEquipmentID = equipmentID
	s.capturedRequest = req
	return s.listResp, s.err
}

func (s *serviceStub) DeleteInspection(user *bootstrap.User, equipmentID string, pmcsID string) error {
	s.capturedUser = user
	s.capturedEquipmentID = equipmentID
	s.capturedPmcsID = pmcsID
	return s.err
}

func (s *serviceStub) UpsertFault(user *bootstrap.User, equipmentID string, pmcsID string, req FaultRequest) (*FaultResponse, error) {
	s.capturedUser = user
	s.capturedEquipmentID = equipmentID
	s.capturedPmcsID = pmcsID
	s.capturedRequest = req
	return s.faultResp, s.err
}

func (s *serviceStub) DeleteFault(user *bootstrap.User, equipmentID string, pmcsID string, req DeleteFaultRequest) error {
	s.capturedUser = user
	s.capturedEquipmentID = equipmentID
	s.capturedPmcsID = pmcsID
	s.capturedRequest = req
	return s.err
}

func (s *serviceStub) DeleteFaults(user *bootstrap.User, equipmentID string, pmcsID string, req BulkDeleteFaultRequest) (*BulkDeleteFaultResponse, error) {
	s.capturedUser = user
	s.capturedEquipmentID = equipmentID
	s.capturedPmcsID = pmcsID
	s.capturedRequest = req
	return s.bulkDeleteResp, s.err
}

const routeTestPmcsID = "11111111-1111-1111-1111-111111111111"

func TestHandlersRequireAuth(t *testing.T) {
	router := newRouteTestRouter(&serviceStub{})

	resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID, nil, nil)

	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestHandlersRejectInvalidAuthContext(t *testing.T) {
	cases := []struct {
		name  string
		value interface{}
	}{
		{name: "wrong type", value: "user-1"},
		{name: "nil user", value: (*bootstrap.User)(nil)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			group := router.Group("/api/v1/auth")
			group.Use(func(c *gin.Context) {
				c.Set("user", tc.value)
				c.Next()
			})
			registerHandlers(group, &serviceStub{})

			resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID, nil, routeUser())

			require.Equal(t, http.StatusUnauthorized, resp.Code)
		})
	}
}

func TestUpsertInspectionSuccess(t *testing.T) {
	now := time.Now().UTC()
	stub := &serviceStub{inspectionResp: &InspectionResponse{
		ID: uuid.MustParse(routeTestPmcsID), EquipmentID: "vehicle-1", GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: now, Faults: []FaultResponse{},
	}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID, InspectionRequest{
		GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: now,
	}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "vehicle-1", stub.capturedEquipmentID)
	require.Equal(t, routeTestPmcsID, stub.capturedPmcsID)
}

func TestGetInspectionSuccess(t *testing.T) {
	now := time.Now().UTC()
	stub := &serviceStub{inspectionResp: &InspectionResponse{
		ID: uuid.MustParse(routeTestPmcsID), EquipmentID: "vehicle-1", GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: now,
		Faults: []FaultResponse{{PmcsID: uuid.MustParse(routeTestPmcsID), SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "x", FaultText: "leak", CreatedAt: now, UpdatedAt: now}},
	}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID, nil, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	var body struct {
		Data InspectionResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Len(t, body.Data.Faults, 1)
}

func TestListInspectionsSuccess(t *testing.T) {
	stub := &serviceStub{listResp: &InspectionListResponse{Inspections: []InspectionSummaryResponse{}, Count: 0}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs", nil, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "vehicle-1", stub.capturedEquipmentID)
}

func TestDeleteInspectionSuccess(t *testing.T) {
	stub := &serviceStub{}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodDelete, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID, nil, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, routeTestPmcsID, stub.capturedPmcsID)
}

func TestUpsertFaultSuccess(t *testing.T) {
	now := time.Now().UTC()
	stub := &serviceStub{faultResp: &FaultResponse{
		PmcsID: uuid.MustParse(routeTestPmcsID), SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "x", FaultText: "leak", CreatedAt: now, UpdatedAt: now,
	}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID+"/faults", FaultRequest{
		GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: now, SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "X", FaultText: "leak",
	}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	captured, ok := stub.capturedRequest.(FaultRequest)
	require.True(t, ok)
	require.Equal(t, "X", captured.Status)

	var body struct {
		Data FaultResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Equal(t, "x", body.Data.Status)
}

func TestInvalidJSONReturnsBadRequest(t *testing.T) {
	router := newRouteTestRouter(&serviceStub{})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID+"/faults", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-1")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestDeleteFaultSuccess(t *testing.T) {
	stub := &serviceStub{}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodDelete, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID+"/faults", DeleteFaultRequest{
		SectionID: "before", ItemIndex: 0,
	}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	captured, ok := stub.capturedRequest.(DeleteFaultRequest)
	require.True(t, ok)
	require.Equal(t, "before", captured.SectionID)
}

func TestBulkDeleteFaultsSuccess(t *testing.T) {
	stub := &serviceStub{bulkDeleteResp: &BulkDeleteFaultResponse{RequestedCount: 2, DeletedCount: 1}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodDelete, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID+"/faults/bulk", BulkDeleteFaultRequest{
		Faults: []BulkDeleteFaultItemRequest{
			{SectionID: "before", ItemIndex: 0},
			{SectionID: "after", ItemIndex: 2},
		},
	}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)

	var body struct {
		Message        string `json:"message"`
		RequestedCount int    `json:"requested_count"`
		DeletedCount   int    `json:"deleted_count"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Equal(t, "Faults deleted", body.Message)
	require.Equal(t, 2, body.RequestedCount)
	require.Equal(t, 1, body.DeletedCount)
}

func TestServiceErrorMapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{name: "unauthorized", err: ErrUnauthorized, want: http.StatusUnauthorized},
		{name: "invalid id", err: ErrInvalidID, want: http.StatusBadRequest},
		{name: "invalid pmcs id", err: ErrInvalidPmcsID, want: http.StatusBadRequest},
		{name: "invalid guide manual", err: ErrInvalidGuideManual, want: http.StatusBadRequest},
		{name: "invalid request", err: ErrInvalidRequest, want: http.StatusBadRequest},
		{name: "invalid status", err: ErrInvalidStatus, want: http.StatusBadRequest},
		{name: "inspection conflict", err: ErrInspectionConflict, want: http.StatusConflict},
		{name: "not found", err: ErrNotFound, want: http.StatusNotFound},
		{name: "inspection not found", err: ErrInspectionNotFound, want: http.StatusNotFound},
		{name: "internal", err: errors.New("boom"), want: http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := newRouteTestRouter(&serviceStub{err: tc.err})
			resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID, nil, routeUser())
			require.Equal(t, tc.want, resp.Code)
		})
	}
}

func newRouteTestRouter(stub Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/auth")
	group.Use(func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID != "" {
			c.Set("user", &bootstrap.User{UserID: userID, Email: userID + "@example.com", Username: userID})
		}
		c.Next()
	})
	registerHandlers(group, stub)
	return router
}

func routeUser() *bootstrap.User {
	return &bootstrap.User{UserID: "user-1", Email: "user-1@example.com", Username: "user-1"}
}

func doRouteJSON(router *gin.Engine, method string, path string, body interface{}, user *bootstrap.User) *httptest.ResponseRecorder {
	var payload bytes.Buffer
	if body != nil {
		err := json.NewEncoder(&payload).Encode(body)
		if err != nil {
			panic(err)
		}
	}
	req := httptest.NewRequest(method, path, &payload)
	req.Header.Set("Content-Type", "application/json")
	if user != nil {
		req.Header.Set("X-User-ID", user.UserID)
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}
```

- [ ] **Step 2: Run the route tests to confirm they fail to compile**

```bash
go test ./api/pmcs_sbs_progress/... -run "TestHandlers|TestUpsertInspection|TestGetInspection|TestListInspections|TestDeleteInspection|TestUpsertFault|TestInvalidJSON|TestDeleteFault|TestBulkDeleteFaults|TestServiceErrorMapping" -v
```

Expected: FAIL — build error, `registerHandlers` doesn't register the new routes yet and `Handler` is missing the new methods the stub/tests reference.

- [ ] **Step 3: Rewrite `route.go`**

Replace the entire file:

```go
package pmcs_sbs_progress

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	DB *sql.DB
}

type Handler struct {
	service Service
}

func RegisterRoutes(deps Dependencies, group *gin.RouterGroup) {
	repo := NewRepository(deps.DB)
	svc := NewService(repo)
	registerHandlers(group, svc)
}

func registerHandlers(group *gin.RouterGroup, svc Service) {
	handler := Handler{service: svc}

	group.PUT("/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id", handler.upsertInspection)
	group.GET("/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id", handler.getInspection)
	group.DELETE("/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id", handler.deleteInspection)
	group.GET("/pmcs-sbs/equipment/:equipment_id/pmcs", handler.listInspections)
	group.PUT("/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults", handler.upsertFault)
	group.DELETE("/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults", handler.deleteFault)
	group.DELETE("/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults/bulk", handler.deleteFaults)
}

func (handler Handler) upsertInspection(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req InspectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	result, err := handler.service.EnsureInspection(user, c.Param("equipment_id"), c.Param("pmcs_id"), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Message: "Inspection saved", Data: result})
}

func (handler Handler) getInspection(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	result, err := handler.service.GetInspection(user, c.Param("equipment_id"), c.Param("pmcs_id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Message: "", Data: result})
}

func (handler Handler) deleteInspection(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	if err := handler.service.DeleteInspection(user, c.Param("equipment_id"), c.Param("pmcs_id")); err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Inspection deleted"})
}

func (handler Handler) listInspections(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req ListInspectionsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid query parameters"})
		return
	}

	result, err := handler.service.ListInspections(user, c.Param("equipment_id"), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Message: "", Data: result})
}

func (handler Handler) upsertFault(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req FaultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	result, err := handler.service.UpsertFault(user, c.Param("equipment_id"), c.Param("pmcs_id"), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Message: "Fault saved", Data: result})
}

func (handler Handler) deleteFault(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req DeleteFaultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	if err := handler.service.DeleteFault(user, c.Param("equipment_id"), c.Param("pmcs_id"), req); err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Fault deleted"})
}

func (handler Handler) deleteFaults(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req BulkDeleteFaultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	result, err := handler.service.DeleteFaults(user, c.Param("equipment_id"), c.Param("pmcs_id"), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Faults deleted",
		"requested_count": result.RequestedCount,
		"deleted_count":   result.DeletedCount,
	})
}

func getUser(c *gin.Context) (*bootstrap.User, bool) {
	value, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return nil, false
	}

	user, ok := value.(*bootstrap.User)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return nil, false
	}

	return user, true
}

func respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
	case errors.Is(err, ErrInvalidID),
		errors.Is(err, ErrInvalidPmcsID),
		errors.Is(err, ErrInvalidGuideManual),
		errors.Is(err, ErrInvalidRequest),
		errors.Is(err, ErrInvalidStatus):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	case errors.Is(err, ErrInspectionConflict):
		c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "pmcs sbs equipment not found"})
	case errors.Is(err, ErrInspectionNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "pmcs sbs inspection not found"})
	default:
		slog.Error("PMCS SBS fault handler failed", "error", err)
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
	}
}
```

- [ ] **Step 4: Run the full package test suite**

```bash
go vet ./...
go build ./...
go test ./api/pmcs_sbs_progress/... ./tests/pmcs_sbs_progress/... -v
```

Expected: `go build ./...` succeeds (the whole module compiles again — this is what was broken since Task 1 Step 6). All tests in both packages PASS.

- [ ] **Step 5: Commit**

```bash
git add api/pmcs_sbs_progress/route.go api/pmcs_sbs_progress/route_test.go
git commit -m "feat(pmcs-sbs): add inspection endpoints and re-scope fault routes by pmcs_id"
```

---

### Task 5: Documentation Updates

**Files:**
- Create: `docs/api/pmcs_sbs_inspections_mobile.md`
- Modify: `docs/api/pmcs_sbs_faults_guide_manual_mobile.md` (superseded pointer only)
- Modify: `docs/api/pmcs_sbs_bulk_fault_delete_mobile.md` (superseded pointer only)
- Modify: `docs/project_notes/decisions.md`

**Interfaces:**
- None — documentation only, no code dependencies. This task can run any time after Task 4 merges.

- [ ] **Step 1: Add a superseded pointer to the top of the old fault docs**

In `docs/api/pmcs_sbs_faults_guide_manual_mobile.md`, insert immediately after the `# PMCS SBS Faults Guide Scope API Changes` title:

```markdown
> **Superseded** — faults are now scoped to a `pmcs_id` (inspection), not a bare `(equipment_id, guide_manual)` pair. See `docs/api/pmcs_sbs_inspections_mobile.md` for the current contract.
```

In `docs/api/pmcs_sbs_bulk_fault_delete_mobile.md`, insert the same pointer immediately after its title heading.

- [ ] **Step 2: Write the new consolidated contract doc**

Create `docs/api/pmcs_sbs_inspections_mobile.md`:

```markdown
# PMCS SBS Inspection History API

This document covers the PMCS SBS API after the introduction of inspection
history. It supersedes `pmcs_sbs_faults_guide_manual_mobile.md` and
`pmcs_sbs_bulk_fault_delete_mobile.md`.

## Summary

Faults are no longer a single overwritable state per `(equipment_id,
guide_manual, section_id, item_index)`. Instead, each PMCS performed on a
vehicle is its own **inspection** (`pmcs_sbs_inspections` row), identified by
a client-generated UUID (`pmcs_id`). Faults belong to one inspection.
Equipment can have many inspections over time, including inspections where
no faults were found ("clean" inspections).

Base URL: `/api/v1/auth`

## Inspection Identity

The mobile client generates a UUID (`pmcs_id`) when the user begins a new
PMCS on a vehicle. That id is sent with every fault save during the session
and identifies the inspection going forward. Starting a new PMCS later means
generating a new `pmcs_id` — there is no server-side session boundary.

`guide_manual` is fixed the first time a `pmcs_id` is used and cannot change
afterward. Sending a different `guide_manual` (or reusing a `pmcs_id` under a
different `equipment_id`) for an existing inspection returns `409`.

## Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `PUT` | `/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id` | Create or update inspection metadata. Use this for a clean (zero-fault) completion, or to correct `performed_date`. |
| `GET` | `/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id` | Get one inspection plus its full faults array. |
| `DELETE` | `/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id` | Delete an inspection and all its faults. |
| `GET` | `/pmcs-sbs/equipment/:equipment_id/pmcs` | List inspection history for a vehicle (most recent first). Optional `guide_manual` filter, `limit`/`offset` pagination (default limit 1000, max 1000). |
| `PUT` | `/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults` | Save (create/update) one fault. Creates the inspection implicitly on first use of a new `pmcs_id`. |
| `DELETE` | `/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults` | Delete one fault. |
| `DELETE` | `/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults/bulk` | Delete up to 100 faults from one inspection. |

`:equipment_id` is the selected Shops vehicle id: `shop_vehicle.id`.
`:pmcs_id` is the client-generated inspection UUID.

## Inspection Object

| Field | Type | Notes |
|-------|------|-------|
| `id` | string (UUID) | The `pmcs_id`. |
| `equipment_id` | string | Returned by the server. |
| `guide_manual` | string | Required on create; immutable after. |
| `performed_date` | string | ISO timestamp; required, client-supplied. |
| `created_by` | string, nullable | User id who first created the inspection. |
| `created_at` / `updated_at` | string | ISO timestamps; response only. |
| `faults` | array | Present on the single-inspection `GET`; omitted from the list endpoint. |

## Inspection Summary Object (list endpoint)

| Field | Type | Notes |
|-------|------|-------|
| `id` | string (UUID) | |
| `guide_manual` | string | |
| `performed_date` | string | ISO timestamp. |
| `fault_count` | integer | Number of faults on this inspection. |
| `created_at` | string | ISO timestamp. |

## Fault Object

| Field | Type | Notes |
|-------|------|-------|
| `pmcs_id` | string (UUID) | The inspection this fault belongs to. |
| `section_id` | string | Required for save/delete. |
| `item_index` | integer | Required for save/delete; must be `0` or greater. |
| `item_no` | string | Required for save. |
| `status` | string | Required for save; returned as normalized server value. |
| `fault_text` | string | Required for save. |
| `corrective_action` | string | Optional; blank string is accepted. |
| `created_at` / `updated_at` | string | ISO timestamps; response only. |

Valid saved `status` response values are `x`, `slash`, and `dash`. Accepted
save inputs: `X`/`x` → `x`, `/`/`slash` → `slash`, `-`/`dash` → `dash`.

## Error Responses

| HTTP | Cause |
|------|-------|
| `400` | Invalid `equipment_id`, malformed `pmcs_id`, invalid `guide_manual`, invalid fault fields, or invalid status. |
| `401` | Missing or invalid authentication. |
| `404` | Vehicle not found or not a shop member; or `pmcs_id` not found for this vehicle. |
| `409` | `guide_manual` (or vehicle ownership) mismatch for an existing `pmcs_id`. |
```

- [ ] **Step 3: Add the ADR to `docs/project_notes/decisions.md`**

Append after the last existing ADR entry:

```markdown
### ADR-017: PMCS SBS Inspection History (2026-07-16)

**Context:**
- `pmcs_sbs_faults` keyed faults by `(equipment_id, guide_manual, section_id, item_index)` with no inspection-event dimension — saving a fault always overwrote the prior state for that checklist item, and a clean (zero-fault) inspection left no trace at all
- Users need a historical view of every PMCS performed on a vehicle: the date it was performed and the faults found during it, including clean passes
- A 2026-06-21 design (`docs/OLD/superpowers/plans/2026-06-21-pmcs-sbs-faults-only.md`) deliberately removed a prior `pmcs_sbs_equipment` + `pmcs_sbs_completions` pair of tables to simplify the feature to "faults only" — this ADR reintroduces an inspection-event concept, reversing part of that simplification, because the product requirement now demands it
- The mobile client already autosaves faults one at a time with no start/submit workflow; a full explicit lifecycle (start/finish endpoints, in_progress/completed status) would be a much bigger behavior change than the requirement called for

**Decision:**
- Add `pmcs_sbs_inspections` as the parent of `pmcs_sbs_faults`, FK'd to `shop_vehicle` with `ON DELETE CASCADE`, carrying `guide_manual`, `performed_date`, and `created_by`
- The client generates the inspection id (UUID) and sends it with every fault save in that session — this is what distinguishes a new inspection from a continuation of the last one, not a server-side heuristic like date-bucketing
- Inspection creation is a hybrid of implicit and explicit: saving a fault implicitly creates its parent inspection if one doesn't exist yet (preserving today's autosave contract), and a separate explicit `PUT .../pmcs/:pmcs_id` endpoint exists for the zero-fault clean-completion case
- `performed_date` is client-supplied, not a server write-timestamp, since field techs may inspect offline and sync later
- `guide_manual` is immutable after an inspection's first creation; a mismatched `guide_manual` (or a `pmcs_id` reused under a different `equipment_id`) on a later request is rejected with `ErrInspectionConflict` (409) rather than silently accepted
- No existing `pmcs_sbs_faults` data is migrated — the rows were current-state-only snapshots with no date-performed concept

**Alternatives considered:**
- Full explicit start/finish lifecycle with an in_progress/completed status (rejected: bigger behavior change than the mobile autosave pattern needed; also considered a single atomic submit-everything-at-the-end call, rejected for the same reason)
- One inspection per calendar day, bucketed server-side (rejected: would incorrectly merge two real inspections performed on the same vehicle on the same day)
- Embedding faults as a JSONB array on the inspection row instead of a separate table (rejected: breaks the existing per-fault autosave contract — concurrent shop members editing different faults on the same inspection would race on read-modify-write of the same JSON blob, and per-fault CHECK constraints can't be enforced on individual JSONB array elements)
- Migrating existing fault rows into synthetic "legacy" inspection records (rejected: the data was not judged valuable enough to justify the migration complexity)

**Consequences:**
- Equipment can now have unlimited PMCS inspections over time, each independently listing its own faults, including inspections with zero faults
- The fault API's identity changed from `(equipment_id, guide_manual, section_id, item_index)` to `(pmcs_id, section_id, item_index)` — a breaking change for API consumers, documented in `docs/api/pmcs_sbs_inspections_mobile.md`
- The Flutter mobile client requires a corresponding Drift schema migration (add `pmcs_id` to its local `PmcsSbsFaultsTable` mirror) and client-side generation/tracking of the inspection UUID per session before this can ship end-to-end — tracked as a follow-up in the `miltech` repo, out of scope for this server-side change
- Every write to `pmcs_sbs_faults` now goes through a transaction that also touches `pmcs_sbs_inspections` (to implicitly create the parent row), adding one extra `INSERT ... ON CONFLICT` per fault save compared to the old single-table upsert
```

- [ ] **Step 4: Commit**

```bash
git add docs/api/pmcs_sbs_inspections_mobile.md docs/api/pmcs_sbs_faults_guide_manual_mobile.md docs/api/pmcs_sbs_bulk_fault_delete_mobile.md docs/project_notes/decisions.md
git commit -m "docs(pmcs-sbs): document inspection history API and record ADR-017"
```
