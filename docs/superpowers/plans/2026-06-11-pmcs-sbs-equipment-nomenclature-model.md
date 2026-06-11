# PMCS SBS Equipment Nomenclature Model Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire the new `pmcs_sbs_equipment.nomenclature` and `pmcs_sbs_equipment.model` columns through PMCS SBS equipment save, sync, list, get, and response paths.

**Architecture:** The generated Jet model and table definitions already include `Nomenclature` and `Model`, so this is an API/service/repository mapping refactor. Treat both fields as optional user-facing equipment metadata like `serial` and `uic`: accept omitted values as empty strings, trim whitespace on writes, persist them on insert/update, and return them on every equipment response shape.

**Tech Stack:** Go, Gin, Postgres, Jet generated model/table types, Firebase-authenticated PMCS SBS progress API.

---

## Current State Summary

- Generated model support exists:
  - `.gen/miltech_ng/public/model/pmcs_sbs_equipment.go` has `Nomenclature string` and `Model string`.
  - `.gen/miltech_ng/public/table/pmcs_sbs_equipment.go` has `PmcsSbsEquipment.Nomenclature`, `PmcsSbsEquipment.Model`, and includes them in `AllColumns`.
- Read queries already select `PmcsSbsEquipment.AllColumns`, so the repository reads the new columns.
- Responses currently drop the values because `EquipmentResponse` and `mapEquipment` omit `nomenclature` and `model`.
- Single-equipment writes currently drop request values because `EquipmentRequest`, `validatedEquipment`, and `UpsertEquipment` omit these fields.
- Sync equipment writes currently drop request values because `SyncEquipmentRequest` and `buildSyncChangeSet` omit these fields.
- Repository upsert currently omits the columns from explicit `INSERT` and `ON CONFLICT DO UPDATE`.
- Route handlers use `ShouldBindJSON` into request structs, so handler logic does not need custom parsing changes once the request structs are expanded.

## Files To Modify

- Modify: `api/pmcs_sbs_progress/types.go`
  - Add `Nomenclature` and `Model` to `EquipmentRequest`, `SyncEquipmentRequest`, and `EquipmentResponse`.
- Modify: `api/pmcs_sbs_progress/service_impl.go`
  - Add fields to `validatedEquipment`.
  - Trim the new fields in `validateEquipmentRequest`.
  - Set the new fields in `UpsertEquipment`.
  - Pass the new fields through `buildSyncChangeSet`.
  - Include the new fields in `mapEquipment`.
- Modify: `api/pmcs_sbs_progress/repository_impl.go`
  - Add `PmcsSbsEquipment.Nomenclature` and `PmcsSbsEquipment.Model` to the explicit insert column list.
  - Add both fields to the `ON CONFLICT DO UPDATE SET` list.
- Modify: `api/pmcs_sbs_progress/service_impl_test.go`
  - Cover trimming, model construction, sync change-set mapping, and response mapping.
- Modify: `api/pmcs_sbs_progress/route_test.go`
  - Cover request JSON binding and response JSON for the new fields.
- Modify: `tests/pmcs_sbs_progress/helpers_test.go`
  - Include sample `Nomenclature` and `Model` values in `sampleEquipment`.
- Modify: `tests/pmcs_sbs_progress/repository_test.go`
  - Verify insert, list/get readback, and update persistence for the new columns.
- Modify: `docs/api/pmcs-sbs-progress-sync.md`
  - Document the two fields in save and response examples.
- Modify: `docs/api/pmcs-sbs-progress-sync-mobile.md`
  - Document the two fields in the resource model, save request, list/get response examples, and sync request examples.

## Task 1: Service DTO And Mapping Tests

**Files:**
- Modify: `api/pmcs_sbs_progress/service_impl_test.go`

- [ ] **Step 1: Extend `TestValidateEquipmentRequest` expectations before production changes**

Update the request in `TestValidateEquipmentRequest`:

```go
req, err := svc.validateEquipmentRequest("550e8400-e29b-41d4-a716-446655440000", EquipmentRequest{
	EquipmentManual: " pmcs_sbs/hmmwv/basic.json ",
	Admin:           " A12 ",
	Serial:          " SER ",
	Nomenclature:    " Truck, Utility ",
	Model:           " M1152A1 ",
	Uic:             " UIC ",
})
```

Add assertions:

```go
require.Equal(t, "Truck, Utility", req.Nomenclature)
require.Equal(t, "M1152A1", req.Model)
```

- [ ] **Step 2: Extend `TestBuildSyncChangeSet` input and assertions**

Update the `SyncEquipmentRequest` in `TestBuildSyncChangeSet`:

```go
UpsertEquipment: []SyncEquipmentRequest{{
	ID:              "550e8400-e29b-41d4-a716-446655440000",
	EquipmentManual: "pmcs_sbs/hmmwv/basic.json",
	Admin:           "A12",
	Serial:          " SER ",
	Nomenclature:    " Truck, Utility ",
	Model:           " M1152A1 ",
	Uic:             " UIC ",
}},
```

Add assertions after the existing `UserUID` assertion:

```go
require.Equal(t, "SER", changeSet.UpsertEquipment[0].Serial)
require.Equal(t, "Truck, Utility", changeSet.UpsertEquipment[0].Nomenclature)
require.Equal(t, "M1152A1", changeSet.UpsertEquipment[0].Model)
require.Equal(t, "UIC", changeSet.UpsertEquipment[0].Uic)
```

- [ ] **Step 3: Add a focused response mapper test**

Add this test near the other equipment tests:

```go
func TestMapEquipmentIncludesMetadata(t *testing.T) {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	now := time.Now().UTC()

	resp := mapEquipment(model.PmcsSbsEquipment{
		ID:              id,
		EquipmentManual: "pmcs_sbs/hmmwv/basic.json",
		Admin:           "A12",
		Serial:          "SER",
		Nomenclature:    "Truck, Utility",
		Model:           "M1152A1",
		Uic:             "UIC",
		CreatedAt:       now,
		UpdatedAt:       now,
	})

	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", resp.ID)
	require.Equal(t, "Truck, Utility", resp.Nomenclature)
	require.Equal(t, "M1152A1", resp.Model)
}
```

Ensure `service_impl_test.go` imports `github.com/google/uuid` and `time` if not already present.

- [ ] **Step 4: Run service tests and confirm failure**

Run:

```bash
go test ./api/pmcs_sbs_progress -run 'TestValidateEquipmentRequest|TestBuildSyncChangeSet|TestMapEquipmentIncludesMetadata' -count=1
```

Expected before implementation: compile failures for missing `Nomenclature` and `Model` fields on request/response/validated structs.

## Task 2: Add API DTO Fields And Service Mapping

**Files:**
- Modify: `api/pmcs_sbs_progress/types.go`
- Modify: `api/pmcs_sbs_progress/service_impl.go`

- [ ] **Step 1: Add fields to equipment request/response DTOs**

In `api/pmcs_sbs_progress/types.go`, update `EquipmentRequest`:

```go
type EquipmentRequest struct {
	EquipmentManual string `json:"equipment_manual"`
	Admin           string `json:"admin"`
	Serial          string `json:"serial"`
	Nomenclature    string `json:"nomenclature"`
	Model           string `json:"model"`
	Uic             string `json:"uic"`
}
```

Update `SyncEquipmentRequest`:

```go
type SyncEquipmentRequest struct {
	ID              string `json:"id"`
	EquipmentManual string `json:"equipment_manual"`
	Admin           string `json:"admin"`
	Serial          string `json:"serial"`
	Nomenclature    string `json:"nomenclature"`
	Model           string `json:"model"`
	Uic             string `json:"uic"`
}
```

Update `EquipmentResponse`:

```go
type EquipmentResponse struct {
	ID              string    `json:"id"`
	EquipmentManual string    `json:"equipment_manual"`
	Admin           string    `json:"admin"`
	Serial          string    `json:"serial"`
	Nomenclature    string    `json:"nomenclature"`
	Model           string    `json:"model"`
	Uic             string    `json:"uic"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
```

- [ ] **Step 2: Add fields to `validatedEquipment`**

In `api/pmcs_sbs_progress/service_impl.go`, update the struct:

```go
type validatedEquipment struct {
	ID              uuid.UUID
	EquipmentManual string
	Admin           string
	Serial          string
	Nomenclature    string
	Model           string
	Uic             string
}
```

- [ ] **Step 3: Trim the fields in `validateEquipmentRequest`**

Update the return value in `validateEquipmentRequest`:

```go
return validatedEquipment{
	ID:              id,
	EquipmentManual: equipmentManual,
	Admin:           admin,
	Serial:          strings.TrimSpace(req.Serial),
	Nomenclature:    strings.TrimSpace(req.Nomenclature),
	Model:           strings.TrimSpace(req.Model),
	Uic:             strings.TrimSpace(req.Uic),
}, nil
```

Do not require `nomenclature` or `model`; old clients must continue to save equipment without sending them.

- [ ] **Step 4: Set fields in single-equipment upsert model**

In `UpsertEquipment`, update the `model.PmcsSbsEquipment` literal:

```go
row := model.PmcsSbsEquipment{
	ID:              validated.ID,
	UserUID:         user.UserID,
	EquipmentManual: validated.EquipmentManual,
	Admin:           validated.Admin,
	Serial:          validated.Serial,
	Nomenclature:    validated.Nomenclature,
	Model:           validated.Model,
	Uic:             validated.Uic,
	CreatedAt:       now,
	UpdatedAt:       now,
}
```

- [ ] **Step 5: Pass fields through sync change-set construction**

In `buildSyncChangeSet`, update the temporary `EquipmentRequest`:

```go
validated, err := service.validateEquipmentRequest(equipment.ID, EquipmentRequest{
	EquipmentManual: equipment.EquipmentManual,
	Admin:           equipment.Admin,
	Serial:          equipment.Serial,
	Nomenclature:    equipment.Nomenclature,
	Model:           equipment.Model,
	Uic:             equipment.Uic,
})
```

Update the `model.PmcsSbsEquipment` literal:

```go
changeSet.UpsertEquipment = append(changeSet.UpsertEquipment, model.PmcsSbsEquipment{
	ID:              validated.ID,
	UserUID:         user.UserID,
	EquipmentManual: validated.EquipmentManual,
	Admin:           validated.Admin,
	Serial:          validated.Serial,
	Nomenclature:    validated.Nomenclature,
	Model:           validated.Model,
	Uic:             validated.Uic,
	CreatedAt:       now,
	UpdatedAt:       now,
})
```

- [ ] **Step 6: Include fields in response mapper**

Update `mapEquipment`:

```go
func mapEquipment(row model.PmcsSbsEquipment) EquipmentResponse {
	return EquipmentResponse{
		ID:              row.ID.String(),
		EquipmentManual: row.EquipmentManual,
		Admin:           row.Admin,
		Serial:          row.Serial,
		Nomenclature:    row.Nomenclature,
		Model:           row.Model,
		Uic:             row.Uic,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}
```

- [ ] **Step 7: Run service tests**

Run:

```bash
go test ./api/pmcs_sbs_progress -run 'TestValidateEquipmentRequest|TestBuildSyncChangeSet|TestMapEquipmentIncludesMetadata' -count=1
```

Expected: PASS.

## Task 3: Repository Write Path Tests And Implementation

**Files:**
- Modify: `tests/pmcs_sbs_progress/helpers_test.go`
- Modify: `tests/pmcs_sbs_progress/repository_test.go`
- Modify: `api/pmcs_sbs_progress/repository_impl.go`

- [ ] **Step 1: Update sample equipment**

In `tests/pmcs_sbs_progress/helpers_test.go`, update `sampleEquipment`:

```go
func sampleEquipment(user *bootstrap.User) model.PmcsSbsEquipment {
	now := time.Now().UTC()
	return model.PmcsSbsEquipment{
		ID:              uuid.New(),
		UserUID:         user.UserID,
		EquipmentManual: "pmcs_sbs/hmmwv/basic.json",
		Admin:           "A12",
		Serial:          "SER-1",
		Nomenclature:    "Truck, Utility",
		Model:           "M1152A1",
		Uic:             "UIC",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}
```

- [ ] **Step 2: Add insert/read assertions to `TestRepositoryEquipmentLifecycle`**

After `require.Equal(t, user.UserID, saved.UserUID)`, add:

```go
require.Equal(t, "Truck, Utility", saved.Nomenclature)
require.Equal(t, "M1152A1", saved.Model)
```

After `require.Len(t, list, 1)`, add:

```go
require.Equal(t, "Truck, Utility", list[0].Nomenclature)
require.Equal(t, "M1152A1", list[0].Model)
```

After `require.Equal(t, equipment.ID, aggregate.Equipment.ID)`, add:

```go
require.Equal(t, "Truck, Utility", aggregate.Equipment.Nomenclature)
require.Equal(t, "M1152A1", aggregate.Equipment.Model)
```

- [ ] **Step 3: Add an update test for conflict path**

Add this test after `TestRepositoryEquipmentLifecycle`:

```go
func TestRepositoryEquipmentUpsertUpdatesMetadata(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-equipment-metadata")
	ensureUser(t, testDB, user)
	repo := pmcs_sbs_progress.NewRepository(testDB)
	equipment := sampleEquipment(user)

	_, err := repo.UpsertEquipment(user, equipment)
	require.NoError(t, err)

	equipment.Nomenclature = "Carrier, Personnel"
	equipment.Model = "M1165A1"
	updated, err := repo.UpsertEquipment(user, equipment)
	require.NoError(t, err)
	require.Equal(t, "Carrier, Personnel", updated.Nomenclature)
	require.Equal(t, "M1165A1", updated.Model)

	aggregate, err := repo.GetEquipmentAggregate(user, equipment.ID.String())
	require.NoError(t, err)
	require.Equal(t, "Carrier, Personnel", aggregate.Equipment.Nomenclature)
	require.Equal(t, "M1165A1", aggregate.Equipment.Model)
}
```

- [ ] **Step 4: Run repository tests and confirm failure**

Run:

```bash
go test ./tests/pmcs_sbs_progress -run 'TestRepositoryEquipmentLifecycle|TestRepositoryEquipmentUpsertUpdatesMetadata' -count=1
```

Expected before repository implementation: assertions fail because returned values are empty or unchanged.

- [ ] **Step 5: Add columns to repository insert/update**

In `api/pmcs_sbs_progress/repository_impl.go`, update the explicit `INSERT` columns in `upsertEquipmentWithExecutor`:

```go
stmt := PmcsSbsEquipment.INSERT(
	PmcsSbsEquipment.ID,
	PmcsSbsEquipment.UserUID,
	PmcsSbsEquipment.EquipmentManual,
	PmcsSbsEquipment.Admin,
	PmcsSbsEquipment.Serial,
	PmcsSbsEquipment.Nomenclature,
	PmcsSbsEquipment.Model,
	PmcsSbsEquipment.Uic,
	PmcsSbsEquipment.CreatedAt,
	PmcsSbsEquipment.UpdatedAt,
).MODEL(equipment).
```

Update the `DO_UPDATE SET` block:

```go
SET(
	PmcsSbsEquipment.EquipmentManual.SET(String(equipment.EquipmentManual)),
	PmcsSbsEquipment.Admin.SET(String(equipment.Admin)),
	PmcsSbsEquipment.Serial.SET(String(equipment.Serial)),
	PmcsSbsEquipment.Nomenclature.SET(String(equipment.Nomenclature)),
	PmcsSbsEquipment.Model.SET(String(equipment.Model)),
	PmcsSbsEquipment.Uic.SET(String(equipment.Uic)),
	PmcsSbsEquipment.UpdatedAt.SET(TimestampzT(now)),
).WHERE(PmcsSbsEquipment.UserUID.EQ(String(user.UserID))),
```

- [ ] **Step 6: Run repository tests**

Run:

```bash
go test ./tests/pmcs_sbs_progress -run 'TestRepositoryEquipmentLifecycle|TestRepositoryEquipmentUpsertUpdatesMetadata' -count=1
```

Expected: PASS.

## Task 4: Route Contract Tests

**Files:**
- Modify: `api/pmcs_sbs_progress/route_test.go`

- [ ] **Step 1: Update `TestUpsertEquipmentSuccess` to prove request binding**

In the stub response, include:

```go
Nomenclature:    "Truck, Utility",
Model:           "M1152A1",
```

In the request body, include whitespace to verify the service receives the bound fields:

```go
resp := doRouteJSON(router, http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/550e8400-e29b-41d4-a716-446655440000", EquipmentRequest{
	EquipmentManual: "pmcs_sbs/hmmwv/basic.json",
	Admin:           "A12",
	Serial:          "SER123",
	Nomenclature:    " Truck, Utility ",
	Model:           " M1152A1 ",
	Uic:             "WABC01",
}, routeUser())
```

After the status assertion, verify the captured request:

```go
captured, ok := stub.capturedRequest.(struct {
	equipmentID string
	req         EquipmentRequest
})
require.True(t, ok)
require.Equal(t, " Truck, Utility ", captured.req.Nomenclature)
require.Equal(t, " M1152A1 ", captured.req.Model)
```

- [ ] **Step 2: Add response JSON assertion**

If the helper already exposes the response body as JSON in other route tests, use the same pattern. Otherwise add a local decode:

```go
var body struct {
	Status int               `json:"status"`
	Data   EquipmentResponse `json:"data"`
}
require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
require.Equal(t, "Truck, Utility", body.Data.Nomenclature)
require.Equal(t, "M1152A1", body.Data.Model)
```

Ensure `encoding/json` is imported if needed.

- [ ] **Step 3: Run route test**

Run:

```bash
go test ./api/pmcs_sbs_progress -run TestUpsertEquipmentSuccess -count=1
```

Expected: PASS.

## Task 5: End-To-End Sync Coverage

**Files:**
- Modify: `api/pmcs_sbs_progress/service_impl_test.go`
- Modify: `tests/pmcs_sbs_progress/repository_test.go`

- [ ] **Step 1: Add service sync response coverage if missing**

If no existing test asserts sync response mapping through service `Sync`, add a repository stub field:

```go
type repoStub struct {
	syncResult *SyncResult
}
```

Update stub `Sync`:

```go
func (repo *repoStub) Sync(user *bootstrap.User, changeSet SyncChangeSet) (*SyncResult, error) {
	if repo.syncResult != nil {
		return repo.syncResult, nil
	}
	return &SyncResult{}, nil
}
```

Add test:

```go
func TestSyncResponseIncludesEquipmentMetadata(t *testing.T) {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	now := time.Now().UTC()
	svc := NewService(&repoStub{syncResult: &SyncResult{
		Equipment: []EquipmentAggregate{{
			Equipment: model.PmcsSbsEquipment{
				ID:              id,
				EquipmentManual: "pmcs_sbs/hmmwv/basic.json",
				Admin:           "A12",
				Serial:          "SER",
				Nomenclature:    "Truck, Utility",
				Model:           "M1152A1",
				Uic:             "UIC",
				CreatedAt:       now,
				UpdatedAt:       now,
			},
		}},
	}})

	resp, err := svc.Sync(requireUser(), SyncRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Equipment, 1)
	require.Equal(t, "Truck, Utility", resp.Equipment[0].Equipment.Nomenclature)
	require.Equal(t, "M1152A1", resp.Equipment[0].Equipment.Model)
}
```

- [ ] **Step 2: Extend repository sync integration assertion**

In `TestRepositorySyncAppliesChangeSet`, after `require.Len(t, result.Equipment, 1)`, add:

```go
require.Equal(t, "Truck, Utility", result.Equipment[0].Equipment.Nomenclature)
require.Equal(t, "M1152A1", result.Equipment[0].Equipment.Model)
```

- [ ] **Step 3: Run focused sync tests**

Run:

```bash
go test ./api/pmcs_sbs_progress ./tests/pmcs_sbs_progress -run 'TestSyncResponseIncludesEquipmentMetadata|TestRepositorySyncAppliesChangeSet' -count=1
```

Expected: PASS.

## Task 6: API Documentation

**Files:**
- Modify: `docs/api/pmcs-sbs-progress-sync.md`
- Modify: `docs/api/pmcs-sbs-progress-sync-mobile.md`

- [ ] **Step 1: Update rules/resource model**

In both docs, state:

```markdown
- `nomenclature` and `model` are optional equipment metadata fields. Blank or omitted values are accepted and returned as empty strings.
```

In `docs/api/pmcs-sbs-progress-sync-mobile.md`, add rows to the Equipment resource table:

```markdown
| `nomenclature` | string | No | Equipment nomenclature. Blank is accepted. |
| `model` | string | No | Equipment model. Blank is accepted. |
```

- [ ] **Step 2: Update save request examples**

Use this request body in both save-equipment sections:

```json
{
  "equipment_manual": "pmcs_sbs/hmmwv/basic.json",
  "admin": "A12",
  "serial": "SER123",
  "nomenclature": "Truck, Utility",
  "model": "M1152A1",
  "uic": "WABC01"
}
```

- [ ] **Step 3: Update equipment response examples**

Where an equipment object is shown in list, get, save, and sync responses, include:

```json
"nomenclature": "Truck, Utility",
"model": "M1152A1",
```

Place the fields after `serial` and before `uic` to match the Go response struct.

## Task 7: Final Verification And Commit

**Files:**
- Verify all modified files.

- [ ] **Step 1: Format Go files**

Run:

```bash
gofmt -w api/pmcs_sbs_progress/types.go api/pmcs_sbs_progress/service_impl.go api/pmcs_sbs_progress/service_impl_test.go api/pmcs_sbs_progress/route_test.go api/pmcs_sbs_progress/repository_impl.go tests/pmcs_sbs_progress/helpers_test.go tests/pmcs_sbs_progress/repository_test.go
```

- [ ] **Step 2: Run focused package tests**

Run:

```bash
go test ./api/pmcs_sbs_progress ./tests/pmcs_sbs_progress -count=1
```

Expected: PASS.

- [ ] **Step 3: Run broader PMCS SBS tests if database is available**

Run:

```bash
go test ./tests/pmcs_sbs_progress -count=1
```

Expected: PASS. If this fails because the local test database does not have current PMCS SBS tables or constraints, capture the exact database error and do not report the refactor as fully verified.

- [ ] **Step 4: Inspect diff**

Run:

```bash
git diff -- api/pmcs_sbs_progress tests/pmcs_sbs_progress docs/api/pmcs-sbs-progress-sync.md docs/api/pmcs-sbs-progress-sync-mobile.md
```

Check that:

- `nomenclature` and `model` are present in both request structs.
- `nomenclature` and `model` are present in equipment response JSON.
- `validateEquipmentRequest` trims both fields but does not require them.
- `UpsertEquipment` and `buildSyncChangeSet` populate `model.PmcsSbsEquipment`.
- Repository insert and update both include the two columns.
- API docs show the new fields in request and response examples.

- [ ] **Step 5: Commit**

Run:

```bash
git add api/pmcs_sbs_progress tests/pmcs_sbs_progress docs/api/pmcs-sbs-progress-sync.md docs/api/pmcs-sbs-progress-sync-mobile.md
git commit -m "fix(pmcs-sbs): persist equipment metadata fields"
```

## Risk Notes

- Backward compatibility: keep `nomenclature` and `model` optional. Requiring them would break existing clients that only send `equipment_manual`, `admin`, `serial`, and `uic`.
- Database/codegen drift: this plan assumes the current generated Jet files are accurate. They already include the columns, so do not manually edit `.gen` files.
- Response compatibility: adding JSON fields is additive. Existing clients should ignore unknown fields.
- Persistence risk: because the repository uses an explicit column list, service-only changes are insufficient. Both insert and conflict-update paths must be updated.
