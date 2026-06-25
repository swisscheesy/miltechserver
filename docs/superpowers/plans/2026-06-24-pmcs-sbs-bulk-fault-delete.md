# PMCS SBS Bulk Fault Delete Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an authenticated PMCS SBS endpoint that deletes up to 100 fault keys for one Shops vehicle and one guide/manual in a single request.

**Architecture:** Keep the feature inside the existing `api/pmcs_sbs_progress` package. Add request/response types, service validation, one handler route, and one repository method that performs the current vehicle membership check once before executing one bulk `DELETE` against `pmcs_sbs_faults`.

**Tech Stack:** Go, Gin, Postgres, Jet, `database/sql`, `testify/require`.

---

## File Structure

- Modify `api/pmcs_sbs_progress/types.go`: add `BulkDeleteFaultRequest`, `BulkDeleteFaultItemRequest`, and `BulkDeleteFaultResponse`.
- Modify `api/pmcs_sbs_progress/service.go`: add `DeleteFaults` to the service interface.
- Modify `api/pmcs_sbs_progress/service_impl.go`: add bulk-delete validation and response assembly.
- Modify `api/pmcs_sbs_progress/service_impl_test.go`: extend `repoStub` and add service validation tests.
- Modify `api/pmcs_sbs_progress/route.go`: register `DELETE /pmcs-sbs/equipment/:equipment_id/faults/bulk` and add the handler.
- Modify `api/pmcs_sbs_progress/route_test.go`: extend `serviceStub` and add handler tests.
- Modify `api/pmcs_sbs_progress/repository.go`: add `DeleteFaults(user, equipmentID, guideManual string, keys []FaultKey) (int64, error)`.
- Modify `api/pmcs_sbs_progress/repository_impl.go`: add the Jet bulk delete implementation.
- Modify `tests/pmcs_sbs_progress/repository_test.go`: add integration coverage for multi-delete, counts, guide isolation, vehicle isolation, and access denial.
- Modify `api/route/route_test.go`: assert route registration.
- Modify `docs/api/pmcs_sbs_faults_guide_manual_mobile.md`: document the new endpoint for mobile.

Do not edit generated files under `.gen/`.

---

### Task 1: Service Contract, Types, And Validation

**Files:**
- Modify: `api/pmcs_sbs_progress/types.go`
- Modify: `api/pmcs_sbs_progress/service.go`
- Modify: `api/pmcs_sbs_progress/repository.go`
- Modify: `api/pmcs_sbs_progress/service_impl.go`
- Test: `api/pmcs_sbs_progress/service_impl_test.go`
- Test support: `api/pmcs_sbs_progress/route_test.go`

- [ ] **Step 1: Extend the repository stub in the service test**

In `api/pmcs_sbs_progress/service_impl_test.go`, update `repoStub`:

```go
type repoStub struct {
	listFaults []model.PmcsSbsFaults
	savedFault *model.PmcsSbsFaults
	deletedCount int64
	err        error

	capturedUser        *bootstrap.User
	capturedEquipmentID string
	capturedGuideManual string
	capturedFault       model.PmcsSbsFaults
	capturedDelete      FaultKey
	capturedBulkKeys    []FaultKey
}
```

Add this method below `DeleteFault`:

```go
func (repo *repoStub) DeleteFaults(user *bootstrap.User, equipmentID string, guideManual string, keys []FaultKey) (int64, error) {
	repo.capturedUser = user
	repo.capturedEquipmentID = equipmentID
	repo.capturedGuideManual = guideManual
	repo.capturedBulkKeys = keys
	return repo.deletedCount, repo.err
}
```

- [ ] **Step 2: Add failing service tests for bulk delete validation and mapping**

Append these tests to `api/pmcs_sbs_progress/service_impl_test.go`:

```go
func TestDeleteFaultsPassesValidatedKeysAndCounts(t *testing.T) {
	stub := &repoStub{deletedCount: 1}
	svc := NewService(stub)

	resp, err := svc.DeleteFaults(requireUser(), " vehicle-1 ", BulkDeleteFaultRequest{
		GuideManual: " pmcs_sbs/hmmwv/file.json ",
		Faults: []BulkDeleteFaultItemRequest{
			{SectionID: " before ", ItemIndex: 0},
			{SectionID: " after ", ItemIndex: 2},
		},
	})

	require.NoError(t, err)
	require.Equal(t, "vehicle-1", stub.capturedEquipmentID)
	require.Equal(t, "pmcs_sbs/hmmwv/file.json", stub.capturedGuideManual)
	require.Equal(t, []FaultKey{
		{EquipmentID: "vehicle-1", GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "before", ItemIndex: 0},
		{EquipmentID: "vehicle-1", GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "after", ItemIndex: 2},
	}, stub.capturedBulkKeys)
	require.Equal(t, 2, resp.RequestedCount)
	require.Equal(t, 1, resp.DeletedCount)
}

func TestDeleteFaultsRequiresAuth(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.DeleteFaults(nil, "vehicle-1", BulkDeleteFaultRequest{
		GuideManual: "pmcs_sbs/hmmwv/file.json",
		Faults:      []BulkDeleteFaultItemRequest{{SectionID: "before", ItemIndex: 0}},
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
		req         BulkDeleteFaultRequest
		want        error
	}{
		{
			name:        "blank equipment",
			equipmentID: " ",
			req:         BulkDeleteFaultRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", Faults: validFaults},
			want:        ErrInvalidID,
		},
		{
			name:        "blank guide manual",
			equipmentID: "vehicle-1",
			req:         BulkDeleteFaultRequest{GuideManual: " ", Faults: validFaults},
			want:        ErrInvalidGuideManual,
		},
		{
			name:        "invalid guide manual",
			equipmentID: "vehicle-1",
			req:         BulkDeleteFaultRequest{GuideManual: "pmcs_sbs/hmmwv/../file.json", Faults: validFaults},
			want:        ErrInvalidGuideManual,
		},
		{
			name:        "empty faults",
			equipmentID: "vehicle-1",
			req:         BulkDeleteFaultRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", Faults: []BulkDeleteFaultItemRequest{}},
			want:        ErrInvalidRequest,
		},
		{
			name:        "too many faults",
			equipmentID: "vehicle-1",
			req:         BulkDeleteFaultRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", Faults: tooManyFaults},
			want:        ErrInvalidRequest,
		},
		{
			name:        "blank section",
			equipmentID: "vehicle-1",
			req:         BulkDeleteFaultRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", Faults: []BulkDeleteFaultItemRequest{{SectionID: " ", ItemIndex: 0}}},
			want:        ErrInvalidRequest,
		},
		{
			name:        "negative item index",
			equipmentID: "vehicle-1",
			req:         BulkDeleteFaultRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", Faults: []BulkDeleteFaultItemRequest{{SectionID: "before", ItemIndex: -1}}},
			want:        ErrInvalidRequest,
		},
		{
			name:        "duplicate key",
			equipmentID: "vehicle-1",
			req: BulkDeleteFaultRequest{
				GuideManual: "pmcs_sbs/hmmwv/file.json",
				Faults: []BulkDeleteFaultItemRequest{
					{SectionID: " before ", ItemIndex: 0},
					{SectionID: "before", ItemIndex: 0},
				},
			},
			want: ErrInvalidRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.validateBulkDeleteFaultRequest(tc.equipmentID, tc.req)
			requireServiceError(t, err, tc.want)
		})
	}
}
```

- [ ] **Step 3: Run service tests and verify they fail**

Run:

```sh
go test ./api/pmcs_sbs_progress -run 'TestDeleteFaults|TestValidateBulkDeleteFaultRequest' -count=1
```

Expected: compile failure because `BulkDeleteFaultRequest`, `BulkDeleteFaultItemRequest`, `DeleteFaults`, `validateBulkDeleteFaultRequest`, and `maxBulkDeleteFaults` do not exist yet.

- [ ] **Step 4: Add bulk request/response types**

In `api/pmcs_sbs_progress/types.go`, add these types after `DeleteFaultRequest`:

```go
type BulkDeleteFaultRequest struct {
	GuideManual string                       `json:"guide_manual"`
	Faults      []BulkDeleteFaultItemRequest `json:"faults"`
}

type BulkDeleteFaultItemRequest struct {
	SectionID string `json:"section_id"`
	ItemIndex int32  `json:"item_index"`
}
```

Add this response type after `FaultListResponse`:

```go
type BulkDeleteFaultResponse struct {
	RequestedCount int `json:"requested_count"`
	DeletedCount   int `json:"deleted_count"`
}
```

- [ ] **Step 5: Extend service and repository interfaces**

In `api/pmcs_sbs_progress/service.go`, replace the interface with:

```go
type Service interface {
	ListFaults(user *bootstrap.User, equipmentID string, guideManual string) (*FaultListResponse, error)
	UpsertFault(user *bootstrap.User, equipmentID string, req FaultRequest) (*FaultResponse, error)
	DeleteFault(user *bootstrap.User, equipmentID string, req DeleteFaultRequest) error
	DeleteFaults(user *bootstrap.User, equipmentID string, req BulkDeleteFaultRequest) (*BulkDeleteFaultResponse, error)
}
```

In `api/pmcs_sbs_progress/repository.go`, replace the interface with:

```go
type Repository interface {
	ListFaults(user *bootstrap.User, equipmentID string, guideManual string) ([]model.PmcsSbsFaults, error)
	UpsertFault(user *bootstrap.User, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error)
	DeleteFault(user *bootstrap.User, key FaultKey) error
	DeleteFaults(user *bootstrap.User, equipmentID string, guideManual string, keys []FaultKey) (int64, error)
}
```

In `api/pmcs_sbs_progress/route_test.go`, add this field to `serviceStub` so the package test binary still compiles after the service interface changes:

```go
bulkDeleteResp *BulkDeleteFaultResponse
```

Add this method after `DeleteFault`:

```go
func (s *serviceStub) DeleteFaults(user *bootstrap.User, equipmentID string, req BulkDeleteFaultRequest) (*BulkDeleteFaultResponse, error) {
	s.capturedUser = user
	s.capturedRequest = struct {
		equipmentID string
		req         BulkDeleteFaultRequest
	}{equipmentID: equipmentID, req: req}
	return s.bulkDeleteResp, s.err
}
```

- [ ] **Step 6: Add service implementation and validation**

In `api/pmcs_sbs_progress/service_impl.go`, add this constant near the top of the file after `NewService`:

```go
const maxBulkDeleteFaults = 100
```

Add this method after `DeleteFault`:

```go
func (service *ServiceImpl) DeleteFaults(user *bootstrap.User, equipmentID string, req BulkDeleteFaultRequest) (*BulkDeleteFaultResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	trimmedEquipmentID, guideManual, keys, err := service.validateBulkDeleteFaultRequest(equipmentID, req)
	if err != nil {
		return nil, err
	}
	deletedCount, err := service.repository.DeleteFaults(user, trimmedEquipmentID, guideManual, keys)
	if err != nil {
		return nil, err
	}
	return &BulkDeleteFaultResponse{
		RequestedCount: len(keys),
		DeletedCount:   int(deletedCount),
	}, nil
}
```

Add this helper after `validateDeleteFaultRequest`:

```go
func (service *ServiceImpl) validateBulkDeleteFaultRequest(equipmentID string, req BulkDeleteFaultRequest) (string, string, []FaultKey, error) {
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return "", "", nil, err
	}

	guideManual, err := validateGuideManual(req.GuideManual)
	if err != nil {
		return "", "", nil, err
	}
	if len(req.Faults) == 0 || len(req.Faults) > maxBulkDeleteFaults {
		return "", "", nil, ErrInvalidRequest
	}

	keys := make([]FaultKey, 0, len(req.Faults))
	seen := make(map[string]struct{}, len(req.Faults))
	for _, fault := range req.Faults {
		sectionID := strings.TrimSpace(fault.SectionID)
		if sectionID == "" || fault.ItemIndex < 0 {
			return "", "", nil, ErrInvalidRequest
		}
		duplicateKey := fmt.Sprintf("%s\x00%d", sectionID, fault.ItemIndex)
		if _, exists := seen[duplicateKey]; exists {
			return "", "", nil, ErrInvalidRequest
		}
		seen[duplicateKey] = struct{}{}
		keys = append(keys, FaultKey{
			EquipmentID: trimmedEquipmentID,
			GuideManual: guideManual,
			SectionID:   sectionID,
			ItemIndex:   fault.ItemIndex,
		})
	}
	return trimmedEquipmentID, guideManual, keys, nil
}
```

Update imports in `service_impl.go` to include `fmt`:

```go
import (
	"fmt"
	"path"
	"strings"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)
```

- [ ] **Step 7: Add a temporary repository method stub so service tests compile**

In `api/pmcs_sbs_progress/repository_impl.go`, add this method after `DeleteFault`:

```go
func (repo *RepositoryImpl) DeleteFaults(user *bootstrap.User, equipmentID string, guideManual string, keys []FaultKey) (int64, error) {
	return 0, errors.New("bulk delete pmcs sbs faults not implemented")
}
```

This is temporary. Task 3 replaces it with the real query.

- [ ] **Step 8: Run service tests and verify they pass**

Run:

```sh
go test ./api/pmcs_sbs_progress -run 'TestDeleteFaults|TestValidateBulkDeleteFaultRequest' -count=1
```

Expected: PASS.

- [ ] **Step 9: Commit service contract and validation**

Run:

```sh
git add api/pmcs_sbs_progress/types.go api/pmcs_sbs_progress/service.go api/pmcs_sbs_progress/repository.go api/pmcs_sbs_progress/service_impl.go api/pmcs_sbs_progress/service_impl_test.go api/pmcs_sbs_progress/route_test.go api/pmcs_sbs_progress/repository_impl.go
git commit -m "feat(pmcs-sbs): validate bulk fault deletes"
```

---

### Task 2: Handler And Route Registration

**Files:**
- Modify: `api/pmcs_sbs_progress/route.go`
- Test: `api/pmcs_sbs_progress/route_test.go`
- Test: `api/route/route_test.go`

- [ ] **Step 1: Add failing handler tests**

Append these tests to `api/pmcs_sbs_progress/route_test.go`:

```go
func TestBulkDeleteFaultsSuccess(t *testing.T) {
	stub := &serviceStub{bulkDeleteResp: &BulkDeleteFaultResponse{RequestedCount: 2, DeletedCount: 1}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodDelete, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/faults/bulk", BulkDeleteFaultRequest{
		GuideManual: "pmcs_sbs/hmmwv/file.json",
		Faults: []BulkDeleteFaultItemRequest{
			{SectionID: "before", ItemIndex: 0},
			{SectionID: "after", ItemIndex: 2},
		},
	}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	captured, ok := stub.capturedRequest.(struct {
		equipmentID string
		req         BulkDeleteFaultRequest
	})
	require.True(t, ok)
	require.Equal(t, "vehicle-1", captured.equipmentID)
	require.Equal(t, "pmcs_sbs/hmmwv/file.json", captured.req.GuideManual)
	require.Len(t, captured.req.Faults, 2)

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

func TestBulkDeleteInvalidJSONReturnsBadRequest(t *testing.T) {
	router := newRouteTestRouter(&serviceStub{})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/faults/bulk", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-1")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
	require.JSONEq(t, `{"message":"invalid request body"}`, resp.Body.String())
}
```

In `api/route/route_test.go`, add this assertion inside `TestSetupRegistersPmcsSbsFaultRoutesUnderAuth`:

```go
requireRouteRegistered(t, router, http.MethodDelete, "/api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults/bulk")
```

- [ ] **Step 2: Run handler and route tests and verify they fail**

Run:

```sh
go test ./api/pmcs_sbs_progress ./api/route -run 'TestBulkDelete|TestSetupRegistersPmcsSbsFaultRoutesUnderAuth' -count=1
```

Expected: FAIL because the bulk route and handler are not registered yet.

- [ ] **Step 3: Register the route and add the handler**

In `api/pmcs_sbs_progress/route.go`, update `registerHandlers`:

```go
func registerHandlers(group *gin.RouterGroup, svc Service) {
	handler := Handler{service: svc}

	group.GET("/pmcs-sbs/equipment/:equipment_id/faults", handler.listFaults)
	group.PUT("/pmcs-sbs/equipment/:equipment_id/faults", handler.upsertFault)
	group.DELETE("/pmcs-sbs/equipment/:equipment_id/faults", handler.deleteFault)
	group.DELETE("/pmcs-sbs/equipment/:equipment_id/faults/bulk", handler.deleteFaults)
}
```

Add this handler after `deleteFault`:

```go
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

	result, err := handler.service.DeleteFaults(user, c.Param("equipment_id"), req)
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
```

- [ ] **Step 4: Run handler and route tests and verify they pass**

Run:

```sh
go test ./api/pmcs_sbs_progress ./api/route -run 'TestBulkDelete|TestSetupRegistersPmcsSbsFaultRoutesUnderAuth' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit handler and route registration**

Run:

```sh
git add api/pmcs_sbs_progress/route.go api/pmcs_sbs_progress/route_test.go api/route/route_test.go
git commit -m "feat(pmcs-sbs): add bulk fault delete route"
```

---

### Task 3: Repository Bulk Delete Query

**Files:**
- Modify: `api/pmcs_sbs_progress/repository_impl.go`
- Test: `tests/pmcs_sbs_progress/repository_test.go`

- [ ] **Step 1: Add failing repository integration tests**

Append these tests to `tests/pmcs_sbs_progress/repository_test.go`:

```go
func TestRepositoryDeleteFaultsDeletesMultipleAndReportsCount(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-bulk-delete")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "A8")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	first := sampleFault(vehicleID)
	first.SectionID = "before"
	first.ItemIndex = 0
	first.ItemNo = "1"
	_, err := repo.UpsertFault(user, first)
	require.NoError(t, err)

	second := sampleFault(vehicleID)
	second.SectionID = "during"
	second.ItemIndex = 3
	second.ItemNo = "4"
	second.FaultText = "during leak"
	_, err = repo.UpsertFault(user, second)
	require.NoError(t, err)

	third := sampleFault(vehicleID)
	third.SectionID = "after"
	third.ItemIndex = 1
	third.ItemNo = "2"
	third.FaultText = "after leak"
	_, err = repo.UpsertFault(user, third)
	require.NoError(t, err)

	deletedCount, err := repo.DeleteFaults(user, vehicleID, "pmcs_sbs/hmmwv/file.json", []pmcs_sbs_progress.FaultKey{
		{EquipmentID: vehicleID, GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "before", ItemIndex: 0},
		{EquipmentID: vehicleID, GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "during", ItemIndex: 3},
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), deletedCount)

	list, err := repo.ListFaults(user, vehicleID, "pmcs_sbs/hmmwv/file.json")
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "after", list[0].SectionID)
}

func TestRepositoryDeleteFaultsIsIdempotentAndReportsExistingRowsOnly(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-bulk-idempotent")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "A9")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	_, err := repo.UpsertFault(user, sampleFault(vehicleID))
	require.NoError(t, err)

	deletedCount, err := repo.DeleteFaults(user, vehicleID, "pmcs_sbs/hmmwv/file.json", []pmcs_sbs_progress.FaultKey{
		{EquipmentID: vehicleID, GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "before", ItemIndex: 0},
		{EquipmentID: vehicleID, GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "missing", ItemIndex: 99},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), deletedCount)

	deletedCount, err = repo.DeleteFaults(user, vehicleID, "pmcs_sbs/hmmwv/file.json", []pmcs_sbs_progress.FaultKey{
		{EquipmentID: vehicleID, GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "before", ItemIndex: 0},
		{EquipmentID: vehicleID, GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "missing", ItemIndex: 99},
	})
	require.NoError(t, err)
	require.Equal(t, int64(0), deletedCount)
}

func TestRepositoryDeleteFaultsPreservesOtherManualsAndVehicles(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-bulk-scope")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "A10")
	otherVehicleID := createShopVehicle(t, testDB, shopID, user, "A11")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	firstManualFault := sampleFault(vehicleID)
	firstManualFault.GuideManual = "pmcs_sbs/hmmwv/first.json"
	_, err := repo.UpsertFault(user, firstManualFault)
	require.NoError(t, err)

	secondManualFault := sampleFault(vehicleID)
	secondManualFault.GuideManual = "pmcs_sbs/hmmwv/second.json"
	secondManualFault.FaultText = "second manual leak"
	_, err = repo.UpsertFault(user, secondManualFault)
	require.NoError(t, err)

	otherVehicleFault := sampleFault(otherVehicleID)
	otherVehicleFault.GuideManual = "pmcs_sbs/hmmwv/first.json"
	otherVehicleFault.FaultText = "other vehicle leak"
	_, err = repo.UpsertFault(user, otherVehicleFault)
	require.NoError(t, err)

	deletedCount, err := repo.DeleteFaults(user, vehicleID, "pmcs_sbs/hmmwv/first.json", []pmcs_sbs_progress.FaultKey{
		{EquipmentID: vehicleID, GuideManual: "pmcs_sbs/hmmwv/first.json", SectionID: "before", ItemIndex: 0},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), deletedCount)

	firstList, err := repo.ListFaults(user, vehicleID, "pmcs_sbs/hmmwv/first.json")
	require.NoError(t, err)
	require.Empty(t, firstList)

	secondList, err := repo.ListFaults(user, vehicleID, "pmcs_sbs/hmmwv/second.json")
	require.NoError(t, err)
	require.Len(t, secondList, 1)
	require.Equal(t, "second manual leak", secondList[0].FaultText)

	otherVehicleList, err := repo.ListFaults(user, otherVehicleID, "pmcs_sbs/hmmwv/first.json")
	require.NoError(t, err)
	require.Len(t, otherVehicleList, 1)
	require.Equal(t, "other vehicle leak", otherVehicleList[0].FaultText)
}

func TestRepositoryDeleteFaultsDeniesNonMemberAndMissingVehicle(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	owner := testUser("pmcs-fault-bulk-owner")
	other := testUser("pmcs-fault-bulk-other")
	ensureUser(t, testDB, owner)
	ensureUser(t, testDB, other)
	shopID := createShopWithMember(t, testDB, owner, "admin")
	vehicleID := createShopVehicle(t, testDB, shopID, owner, "A12")
	repo := pmcs_sbs_progress.NewRepository(testDB)
	keys := []pmcs_sbs_progress.FaultKey{{EquipmentID: vehicleID, GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "before", ItemIndex: 0}}

	_, err := repo.DeleteFaults(other, vehicleID, "pmcs_sbs/hmmwv/file.json", keys)
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)

	_, err = repo.DeleteFaults(owner, "missing-vehicle", "pmcs_sbs/hmmwv/file.json", []pmcs_sbs_progress.FaultKey{{EquipmentID: "missing-vehicle", GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "before", ItemIndex: 0}})
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)
}
```

- [ ] **Step 2: Run repository tests and verify they fail**

Run:

```sh
go test ./tests/pmcs_sbs_progress -run 'TestRepositoryDeleteFaults' -count=1
```

Expected: FAIL because `DeleteFaults` still returns the temporary not-implemented error.

- [ ] **Step 3: Replace the temporary repository stub with the Jet delete**

In `api/pmcs_sbs_progress/repository_impl.go`, replace the temporary `DeleteFaults` method with:

```go
func (repo *RepositoryImpl) DeleteFaults(user *bootstrap.User, equipmentID string, guideManual string, keys []FaultKey) (int64, error) {
	if err := repo.requireVehicleAccess(user, equipmentID); err != nil {
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
			PmcsSbsFaults.EquipmentID.EQ(String(equipmentID)).
				AND(PmcsSbsFaults.GuideManual.EQ(String(guideManual))).
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
```

`Expression` is already available through the existing dot import:

```go
. "github.com/go-jet/jet/v2/postgres"
```

- [ ] **Step 4: Run repository tests and verify they pass**

Run:

```sh
go test ./tests/pmcs_sbs_progress -run 'TestRepositoryDeleteFaults' -count=1
```

Expected: PASS.

- [ ] **Step 5: Run package tests touched by Tasks 1-3**

Run:

```sh
go test ./api/pmcs_sbs_progress ./tests/pmcs_sbs_progress ./api/route -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit repository implementation**

Run:

```sh
git add api/pmcs_sbs_progress/repository_impl.go tests/pmcs_sbs_progress/repository_test.go
git commit -m "feat(pmcs-sbs): delete multiple faults in one query"
```

---

### Task 4: Mobile API Documentation

**Files:**
- Modify: `docs/api/pmcs_sbs_faults_guide_manual_mobile.md`

- [ ] **Step 1: Update the endpoint table**

In `docs/api/pmcs_sbs_faults_guide_manual_mobile.md`, update the "Current Fault Endpoints" table to include:

```markdown
| `DELETE` | `/pmcs-sbs/equipment/:equipment_id/faults/bulk` | Delete up to 100 faults for one Shops vehicle and one PMCS SBS guide/manual. |
```

- [ ] **Step 2: Add the bulk delete section after "Delete Fault"**

Insert this section after the existing single-delete documentation:

````markdown
## Bulk Delete Faults

Request:

`DELETE /api/v1/auth/pmcs-sbs/equipment/550e8400-e29b-41d4-a716-446655440000/faults/bulk`

```json
{
  "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
  "faults": [
    {
      "section_id": "before",
      "item_index": 0
    },
    {
      "section_id": "before",
      "item_index": 1
    }
  ]
}
```

Success response:

```json
{
  "message": "Faults deleted",
  "requested_count": 2,
  "deleted_count": 1
}
```

Rules:

- `guide_manual` is required once per request.
- `faults` must contain `1` to `100` entries.
- Each fault key requires `section_id` and `item_index`.
- Duplicate `(section_id, item_index)` entries are rejected.
- Missing fault keys do not fail the request when the vehicle is accessible. They are included in `requested_count` but not `deleted_count`.
````

- [ ] **Step 3: Update the error table**

In the "Error Responses" table, add these rows:

```markdown
| Empty `faults` on bulk delete | `400` | `{"message":"invalid request"}` |
| More than 100 faults on bulk delete | `400` | `{"message":"invalid request"}` |
| Duplicate bulk delete fault keys | `400` | `{"message":"invalid request"}` |
```

- [ ] **Step 4: Review the documentation diff**

Run:

```sh
git diff -- docs/api/pmcs_sbs_faults_guide_manual_mobile.md
```

Expected: diff documents only the new bulk delete endpoint and does not alter the existing single-delete contract.

- [ ] **Step 5: Commit documentation**

Run:

```sh
git add docs/api/pmcs_sbs_faults_guide_manual_mobile.md
git commit -m "docs(pmcs-sbs): document bulk fault delete"
```

---

### Task 5: Final Verification

**Files:**
- Verify: `api/pmcs_sbs_progress`
- Verify: `tests/pmcs_sbs_progress`
- Verify: `api/route`
- Verify: `docs/api/pmcs_sbs_faults_guide_manual_mobile.md`

- [ ] **Step 1: Run focused PMCS SBS verification**

Run:

```sh
go test ./api/pmcs_sbs_progress ./tests/pmcs_sbs_progress ./api/route -count=1
```

Expected: PASS.

- [ ] **Step 2: Run PMCS SBS library verification if route setup changed outside the fault package**

Run this only if implementation touched route setup outside `api/pmcs_sbs_progress/route.go` and `api/route/route_test.go`:

```sh
go test ./api/library/pmcs_sbs -count=1
```

Expected: PASS.

- [ ] **Step 3: Inspect final git state**

Run:

```sh
git status --short
git log --oneline -5
```

Expected:

- `git status --short` shows no unstaged or uncommitted implementation changes.
- Recent commits include:
  - `feat(pmcs-sbs): validate bulk fault deletes`
  - `feat(pmcs-sbs): add bulk fault delete route`
  - `feat(pmcs-sbs): delete multiple faults in one query`
  - `docs(pmcs-sbs): document bulk fault delete`

- [ ] **Step 4: Summarize verification**

Report:

- New endpoint: `DELETE /api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults/bulk`
- Request max: `100` fault keys
- Scope: one `guide_manual` per request
- Response counts: `requested_count`, `deleted_count`
- Verification commands and pass/fail status
- Any unrelated baseline failures, with file/package names if present
