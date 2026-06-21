# PMCS SBS Faults-Only Server Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor authenticated PMCS SBS server persistence to faults-only operations backed by `shop_vehicle`, removing stale equipment, completion, and sync behavior.

**Architecture:** Keep the existing `api/pmcs_sbs_progress` package name to limit churn, but reduce its public surface to fault list/save/delete. Every operation validates authenticated shop membership through `shop_vehicle -> shop_members` before reading or mutating `pmcs_sbs_faults`. The public PMCS SBS library API and Shops feature behavior remain unchanged.

**Tech Stack:** Go, Gin, PostgreSQL, Jet generated models/tables, Firebase-authenticated `bootstrap.User`, `testify/require`.

---

## File Structure

- Modify `api/pmcs_sbs_progress/types.go`: remove equipment, completion, and sync request/response types; keep fault request/response/list response types.
- Modify `api/pmcs_sbs_progress/service.go`: expose only `ListFaults`, `UpsertFault`, and `DeleteFault`.
- Modify `api/pmcs_sbs_progress/repository.go`: expose only fault operations and small key/result types.
- Modify `api/pmcs_sbs_progress/errors.go`: remove stale blob/sync errors; keep auth, invalid request/status, and not found.
- Modify `api/pmcs_sbs_progress/service_impl.go`: remove equipment/completion/sync code; validate text `shop_vehicle.id` values without UUID parsing.
- Modify `api/pmcs_sbs_progress/repository_impl.go`: remove equipment/completion/sync code; authorize through `shop_vehicle` and `shop_members`; use string `PmcsSbsFaults.EquipmentID`.
- Modify `api/pmcs_sbs_progress/route.go`: register only fault list/save/delete routes.
- Modify `api/pmcs_sbs_progress/service_impl_test.go`: replace stale tests with fault validation/service tests.
- Modify `api/pmcs_sbs_progress/route_test.go`: replace stale route tests with fault-only handler tests.
- Modify `tests/pmcs_sbs_progress/helpers_test.go`: create users, shops, shop members, shop vehicles, and sample PMCS SBS faults.
- Modify `tests/pmcs_sbs_progress/repository_test.go`: replace stale repository tests with fault-only integration tests.
- Modify `api/route/route_test.go`: assert only the surviving PMCS SBS fault routes are registered.
- Modify `docs/api/pmcs-sbs-progress-sync.md`: rewrite as a faults-only PMCS SBS API doc.
- Modify `docs/api/pmcs-sbs-progress-sync-mobile.md`: rewrite mobile contract around list/save/delete faults only.

## Task 1: Fault-Only Contracts And Service Validation

**Files:**
- Modify: `api/pmcs_sbs_progress/types.go`
- Modify: `api/pmcs_sbs_progress/service.go`
- Modify: `api/pmcs_sbs_progress/repository.go`
- Modify: `api/pmcs_sbs_progress/errors.go`
- Modify: `api/pmcs_sbs_progress/service_impl.go`
- Test: `api/pmcs_sbs_progress/service_impl_test.go`

- [ ] **Step 1: Replace service tests with failing fault-only validation tests**

Replace `api/pmcs_sbs_progress/service_impl_test.go` with:

```go
package pmcs_sbs_progress

import (
	"errors"
	"testing"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"

	"github.com/stretchr/testify/require"
)

type repoStub struct {
	listFaults []model.PmcsSbsFaults
	savedFault *model.PmcsSbsFaults
	err        error

	capturedUser        *bootstrap.User
	capturedEquipmentID string
	capturedFault       model.PmcsSbsFaults
	capturedDelete      FaultKey
}

func (repo *repoStub) ListFaults(user *bootstrap.User, equipmentID string) ([]model.PmcsSbsFaults, error) {
	repo.capturedUser = user
	repo.capturedEquipmentID = equipmentID
	return repo.listFaults, repo.err
}

func (repo *repoStub) UpsertFault(user *bootstrap.User, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error) {
	repo.capturedUser = user
	repo.capturedFault = fault
	if repo.savedFault != nil {
		return repo.savedFault, repo.err
	}
	return &fault, repo.err
}

func (repo *repoStub) DeleteFault(user *bootstrap.User, key FaultKey) error {
	repo.capturedUser = user
	repo.capturedDelete = key
	return repo.err
}

func requireUser() *bootstrap.User {
	return &bootstrap.User{UserID: "user-1", Email: "user-1@example.com", Username: "user-1"}
}

func requireServiceError(t *testing.T, err error, target error) {
	t.Helper()
	require.Error(t, err)
	require.Truef(t, errors.Is(err, target), "expected %v, got %v", target, err)
}

func TestListFaultsRequiresAuth(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.ListFaults(nil, "vehicle-1")

	requireServiceError(t, err, ErrUnauthorized)
}

func TestListFaultsRejectsBlankEquipmentID(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.ListFaults(requireUser(), " ")

	requireServiceError(t, err, ErrInvalidID)
}

func TestListFaultsMapsRows(t *testing.T) {
	now := time.Now().UTC()
	stub := &repoStub{listFaults: []model.PmcsSbsFaults{{
		EquipmentID:      "vehicle-1",
		SectionID:        "before",
		ItemIndex:        0,
		ItemNo:           "1",
		Status:           "x",
		FaultText:        "leak",
		CorrectiveAction: "tightened",
		CreatedAt:        now,
		UpdatedAt:        now,
	}}}
	svc := NewService(stub)

	resp, err := svc.ListFaults(requireUser(), " vehicle-1 ")

	require.NoError(t, err)
	require.Equal(t, "vehicle-1", stub.capturedEquipmentID)
	require.Len(t, resp.Faults, 1)
	require.Equal(t, "vehicle-1", resp.Faults[0].EquipmentID)
	require.Equal(t, "before", resp.Faults[0].SectionID)
	require.Equal(t, "x", resp.Faults[0].Status)
	require.Equal(t, 1, resp.Count)
}

func TestValidateFaultRequest(t *testing.T) {
	svc := NewService(&repoStub{})

	fault, err := svc.validateFaultRequest(" vehicle-1 ", FaultRequest{
		SectionID:        " before ",
		ItemIndex:        0,
		ItemNo:           " 1 ",
		Status:           " X ",
		FaultText:        " leak ",
		CorrectiveAction: " tightened ",
	})

	require.NoError(t, err)
	require.Equal(t, "vehicle-1", fault.EquipmentID)
	require.Equal(t, "before", fault.SectionID)
	require.Equal(t, int32(0), fault.ItemIndex)
	require.Equal(t, "1", fault.ItemNo)
	require.Equal(t, "x", fault.Status)
	require.Equal(t, "leak", fault.FaultText)
	require.Equal(t, "tightened", fault.CorrectiveAction)
	require.False(t, fault.CreatedAt.IsZero())
	require.False(t, fault.UpdatedAt.IsZero())
}

func TestValidateFaultRequestAcceptsAllowedStatuses(t *testing.T) {
	svc := NewService(&repoStub{})
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
			fault, err := svc.validateFaultRequest("vehicle-1", FaultRequest{
				SectionID: "before",
				ItemIndex: 0,
				ItemNo:    "1",
				Status:    tc.input,
				FaultText: "leak",
			})
			require.NoError(t, err)
			require.Equal(t, tc.want, fault.Status)
		})
	}
}

func TestValidateFaultRequestRejectsInvalidValues(t *testing.T) {
	svc := NewService(&repoStub{})

	cases := []struct {
		name        string
		equipmentID string
		req         FaultRequest
		want        error
	}{
		{
			name:        "blank equipment",
			equipmentID: " ",
			req:         FaultRequest{SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "X", FaultText: "leak"},
			want:        ErrInvalidID,
		},
		{
			name:        "blank section",
			equipmentID: "vehicle-1",
			req:         FaultRequest{SectionID: " ", ItemIndex: 0, ItemNo: "1", Status: "X", FaultText: "leak"},
			want:        ErrInvalidRequest,
		},
		{
			name:        "negative item index",
			equipmentID: "vehicle-1",
			req:         FaultRequest{SectionID: "before", ItemIndex: -1, ItemNo: "1", Status: "X", FaultText: "leak"},
			want:        ErrInvalidRequest,
		},
		{
			name:        "blank item no",
			equipmentID: "vehicle-1",
			req:         FaultRequest{SectionID: "before", ItemIndex: 0, ItemNo: " ", Status: "X", FaultText: "leak"},
			want:        ErrInvalidRequest,
		},
		{
			name:        "invalid status",
			equipmentID: "vehicle-1",
			req:         FaultRequest{SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "BAD", FaultText: "leak"},
			want:        ErrInvalidStatus,
		},
		{
			name:        "blank fault text",
			equipmentID: "vehicle-1",
			req:         FaultRequest{SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "X", FaultText: " "},
			want:        ErrInvalidRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.validateFaultRequest(tc.equipmentID, tc.req)
			requireServiceError(t, err, tc.want)
		})
	}
}

func TestValidateDeleteFaultRequest(t *testing.T) {
	svc := NewService(&repoStub{})

	key, err := svc.validateDeleteFaultRequest(" vehicle-1 ", DeleteFaultRequest{
		SectionID: " before ",
		ItemIndex: 0,
	})

	require.NoError(t, err)
	require.Equal(t, "vehicle-1", key.EquipmentID)
	require.Equal(t, "before", key.SectionID)
	require.Equal(t, int32(0), key.ItemIndex)
}

func TestValidateDeleteFaultRequestRejectsInvalidValues(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.validateDeleteFaultRequest(" ", DeleteFaultRequest{SectionID: "before", ItemIndex: 0})
	requireServiceError(t, err, ErrInvalidID)

	_, err = svc.validateDeleteFaultRequest("vehicle-1", DeleteFaultRequest{SectionID: " ", ItemIndex: 0})
	requireServiceError(t, err, ErrInvalidRequest)

	_, err = svc.validateDeleteFaultRequest("vehicle-1", DeleteFaultRequest{SectionID: "before", ItemIndex: -1})
	requireServiceError(t, err, ErrInvalidRequest)
}

func TestUpsertFaultReturnsMappedResponse(t *testing.T) {
	now := time.Now().UTC()
	stub := &repoStub{savedFault: &model.PmcsSbsFaults{
		EquipmentID:      "vehicle-1",
		SectionID:        "before",
		ItemIndex:        0,
		ItemNo:           "1",
		Status:           "x",
		FaultText:        "leak",
		CorrectiveAction: "",
		CreatedAt:        now,
		UpdatedAt:        now,
	}}
	svc := NewService(stub)

	resp, err := svc.UpsertFault(requireUser(), "vehicle-1", FaultRequest{
		SectionID: "before",
		ItemIndex: 0,
		ItemNo:    "1",
		Status:    "X",
		FaultText: "leak",
	})

	require.NoError(t, err)
	require.Equal(t, "vehicle-1", stub.capturedFault.EquipmentID)
	require.Equal(t, "x", stub.capturedFault.Status)
	require.Equal(t, "vehicle-1", resp.EquipmentID)
	require.Equal(t, "x", resp.Status)
}

func TestDeleteFaultPassesValidatedKey(t *testing.T) {
	stub := &repoStub{}
	svc := NewService(stub)

	err := svc.DeleteFault(requireUser(), " vehicle-1 ", DeleteFaultRequest{SectionID: " before ", ItemIndex: 0})

	require.NoError(t, err)
	require.Equal(t, "vehicle-1", stub.capturedDelete.EquipmentID)
	require.Equal(t, "before", stub.capturedDelete.SectionID)
	require.Equal(t, int32(0), stub.capturedDelete.ItemIndex)
}
```

- [ ] **Step 2: Run service tests to verify they fail**

Run:

```bash
go test ./api/pmcs_sbs_progress -run 'Test(ListFaults|ValidateFault|ValidateDeleteFault|UpsertFault|DeleteFault)' -count=1
```

Expected: FAIL because `Repository`, `Service`, and implementation still include stale equipment/completion/sync signatures, and `model.PmcsSbsEquipment` / `model.PmcsSbsCompletions` no longer exist.

- [ ] **Step 3: Replace request/response types with faults-only types**

Replace `api/pmcs_sbs_progress/types.go` with:

```go
package pmcs_sbs_progress

import "time"

type FaultRequest struct {
	SectionID        string `json:"section_id"`
	ItemIndex        int32  `json:"item_index"`
	ItemNo           string `json:"item_no"`
	Status           string `json:"status"`
	FaultText        string `json:"fault_text"`
	CorrectiveAction string `json:"corrective_action"`
}

type DeleteFaultRequest struct {
	SectionID string `json:"section_id"`
	ItemIndex int32  `json:"item_index"`
}

type FaultResponse struct {
	EquipmentID      string    `json:"equipment_id"`
	SectionID        string    `json:"section_id"`
	ItemIndex        int32     `json:"item_index"`
	ItemNo           string    `json:"item_no"`
	Status           string    `json:"status"`
	FaultText        string    `json:"fault_text"`
	CorrectiveAction string    `json:"corrective_action"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type FaultListResponse struct {
	Faults []FaultResponse `json:"faults"`
	Count  int             `json:"count"`
}
```

- [ ] **Step 4: Replace service and repository interfaces**

Replace `api/pmcs_sbs_progress/service.go` with:

```go
package pmcs_sbs_progress

import "miltechserver/bootstrap"

type Service interface {
	ListFaults(user *bootstrap.User, equipmentID string) (*FaultListResponse, error)
	UpsertFault(user *bootstrap.User, equipmentID string, req FaultRequest) (*FaultResponse, error)
	DeleteFault(user *bootstrap.User, equipmentID string, req DeleteFaultRequest) error
}
```

Replace `api/pmcs_sbs_progress/repository.go` with:

```go
package pmcs_sbs_progress

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Repository interface {
	ListFaults(user *bootstrap.User, equipmentID string) ([]model.PmcsSbsFaults, error)
	UpsertFault(user *bootstrap.User, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error)
	DeleteFault(user *bootstrap.User, key FaultKey) error
}

type FaultKey struct {
	EquipmentID string
	SectionID   string
	ItemIndex   int32
}
```

- [ ] **Step 5: Remove stale errors**

Replace `api/pmcs_sbs_progress/errors.go` with:

```go
package pmcs_sbs_progress

import "errors"

var (
	ErrInvalidID      = errors.New("invalid id")
	ErrInvalidRequest = errors.New("invalid request")
	ErrInvalidStatus  = errors.New("invalid fault status")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrNotFound       = errors.New("pmcs sbs equipment not found")
)
```

- [ ] **Step 6: Replace service implementation with faults-only implementation**

Replace `api/pmcs_sbs_progress/service_impl.go` with:

```go
package pmcs_sbs_progress

import (
	"strings"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type ServiceImpl struct {
	repository Repository
}

func NewService(repository Repository) *ServiceImpl {
	return &ServiceImpl{repository: repository}
}

func (service *ServiceImpl) ListFaults(user *bootstrap.User, equipmentID string) (*FaultListResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return nil, err
	}

	rows, err := service.repository.ListFaults(user, trimmedEquipmentID)
	if err != nil {
		return nil, err
	}

	faults := make([]FaultResponse, 0, len(rows))
	for _, row := range rows {
		faults = append(faults, mapFault(row))
	}
	return &FaultListResponse{Faults: faults, Count: len(faults)}, nil
}

func (service *ServiceImpl) UpsertFault(user *bootstrap.User, equipmentID string, req FaultRequest) (*FaultResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	row, err := service.validateFaultRequest(equipmentID, req)
	if err != nil {
		return nil, err
	}
	saved, err := service.repository.UpsertFault(user, row)
	if err != nil {
		return nil, err
	}
	resp := mapFault(*saved)
	return &resp, nil
}

func (service *ServiceImpl) DeleteFault(user *bootstrap.User, equipmentID string, req DeleteFaultRequest) error {
	if !hasAuthenticatedUser(user) {
		return ErrUnauthorized
	}
	key, err := service.validateDeleteFaultRequest(equipmentID, req)
	if err != nil {
		return err
	}
	return service.repository.DeleteFault(user, key)
}

func (service *ServiceImpl) validateFaultRequest(equipmentID string, req FaultRequest) (model.PmcsSbsFaults, error) {
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return model.PmcsSbsFaults{}, err
	}

	sectionID := strings.TrimSpace(req.SectionID)
	itemNo := strings.TrimSpace(req.ItemNo)
	status, validStatus := normalizeFaultStatus(req.Status)
	faultText := strings.TrimSpace(req.FaultText)
	if sectionID == "" || itemNo == "" || req.ItemIndex < 0 || faultText == "" {
		return model.PmcsSbsFaults{}, ErrInvalidRequest
	}
	if !validStatus {
		return model.PmcsSbsFaults{}, ErrInvalidStatus
	}

	now := time.Now().UTC()
	return model.PmcsSbsFaults{
		EquipmentID:      trimmedEquipmentID,
		SectionID:        sectionID,
		ItemIndex:        req.ItemIndex,
		ItemNo:           itemNo,
		Status:           status,
		FaultText:        faultText,
		CorrectiveAction: strings.TrimSpace(req.CorrectiveAction),
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

func (service *ServiceImpl) validateDeleteFaultRequest(equipmentID string, req DeleteFaultRequest) (FaultKey, error) {
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return FaultKey{}, err
	}

	sectionID := strings.TrimSpace(req.SectionID)
	if sectionID == "" || req.ItemIndex < 0 {
		return FaultKey{}, ErrInvalidRequest
	}
	return FaultKey{EquipmentID: trimmedEquipmentID, SectionID: sectionID, ItemIndex: req.ItemIndex}, nil
}

func validateEquipmentID(equipmentID string) (string, error) {
	trimmedEquipmentID := strings.TrimSpace(equipmentID)
	if trimmedEquipmentID == "" {
		return "", ErrInvalidID
	}
	return trimmedEquipmentID, nil
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
		EquipmentID:      row.EquipmentID,
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
```

- [ ] **Step 7: Run service tests**

Run:

```bash
go test ./api/pmcs_sbs_progress -run 'Test(ListFaults|ValidateFault|ValidateDeleteFault|UpsertFault|DeleteFault)' -count=1
```

Expected: PASS for service tests after repository/route compile errors are removed in later tasks. If the package still fails to compile because repository or route files reference stale types, continue to Task 2 before re-running the full package.

- [ ] **Step 8: Commit Task 1**

```bash
git add api/pmcs_sbs_progress/types.go api/pmcs_sbs_progress/service.go api/pmcs_sbs_progress/repository.go api/pmcs_sbs_progress/errors.go api/pmcs_sbs_progress/service_impl.go api/pmcs_sbs_progress/service_impl_test.go
git commit -m "refactor(pmcs-sbs): reduce service contract to faults"
```

## Task 2: Fault-Only Routes

**Files:**
- Modify: `api/pmcs_sbs_progress/route.go`
- Test: `api/pmcs_sbs_progress/route_test.go`
- Test: `api/route/route_test.go`

- [ ] **Step 1: Replace route tests with failing fault-only handler tests**

Replace `api/pmcs_sbs_progress/route_test.go` with:

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
	"github.com/stretchr/testify/require"
)

type serviceStub struct {
	listResp        *FaultListResponse
	faultResp       *FaultResponse
	err             error
	capturedUser    *bootstrap.User
	capturedRequest interface{}
}

func (s *serviceStub) ListFaults(user *bootstrap.User, equipmentID string) (*FaultListResponse, error) {
	s.capturedUser = user
	s.capturedRequest = equipmentID
	return s.listResp, s.err
}

func (s *serviceStub) UpsertFault(user *bootstrap.User, equipmentID string, req FaultRequest) (*FaultResponse, error) {
	s.capturedUser = user
	s.capturedRequest = struct {
		equipmentID string
		req         FaultRequest
	}{equipmentID: equipmentID, req: req}
	return s.faultResp, s.err
}

func (s *serviceStub) DeleteFault(user *bootstrap.User, equipmentID string, req DeleteFaultRequest) error {
	s.capturedUser = user
	s.capturedRequest = struct {
		equipmentID string
		req         DeleteFaultRequest
	}{equipmentID: equipmentID, req: req}
	return s.err
}

func TestHandlersRequireAuth(t *testing.T) {
	router := newRouteTestRouter(&serviceStub{})

	resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/faults", nil, nil)

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

			resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/faults", nil, routeUser())

			require.Equal(t, http.StatusUnauthorized, resp.Code)
		})
	}
}

func TestListFaultsSuccess(t *testing.T) {
	stub := &serviceStub{listResp: &FaultListResponse{Faults: []FaultResponse{}, Count: 0}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/faults", nil, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "user-1", stub.capturedUser.UserID)
	require.Equal(t, "vehicle-1", stub.capturedRequest)
}

func TestUpsertFaultSuccess(t *testing.T) {
	now := time.Now().UTC()
	stub := &serviceStub{faultResp: &FaultResponse{
		EquipmentID:      "vehicle-1",
		SectionID:        "before",
		ItemIndex:        0,
		ItemNo:           "1",
		Status:           "x",
		FaultText:        "leak",
		CorrectiveAction: "",
		CreatedAt:        now,
		UpdatedAt:        now,
	}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/faults", FaultRequest{
		SectionID: "before",
		ItemIndex: 0,
		ItemNo:    "1",
		Status:    "X",
		FaultText: "leak",
	}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	captured, ok := stub.capturedRequest.(struct {
		equipmentID string
		req         FaultRequest
	})
	require.True(t, ok)
	require.Equal(t, "vehicle-1", captured.equipmentID)
	require.Equal(t, "X", captured.req.Status)

	var body struct {
		Status int           `json:"status"`
		Data   FaultResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Equal(t, "x", body.Data.Status)
}

func TestInvalidJSONReturnsBadRequest(t *testing.T) {
	router := newRouteTestRouter(&serviceStub{})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/faults", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-1")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestDeleteFaultSuccess(t *testing.T) {
	stub := &serviceStub{}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodDelete, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/faults", DeleteFaultRequest{
		SectionID: "before",
		ItemIndex: 0,
	}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	captured, ok := stub.capturedRequest.(struct {
		equipmentID string
		req         DeleteFaultRequest
	})
	require.True(t, ok)
	require.Equal(t, "vehicle-1", captured.equipmentID)
	require.Equal(t, "before", captured.req.SectionID)
}

func TestServiceErrorMapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{name: "unauthorized", err: ErrUnauthorized, want: http.StatusUnauthorized},
		{name: "invalid id", err: ErrInvalidID, want: http.StatusBadRequest},
		{name: "invalid request", err: ErrInvalidRequest, want: http.StatusBadRequest},
		{name: "invalid status", err: ErrInvalidStatus, want: http.StatusBadRequest},
		{name: "not found", err: ErrNotFound, want: http.StatusNotFound},
		{name: "internal", err: errors.New("boom"), want: http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := newRouteTestRouter(&serviceStub{err: tc.err})
			resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/faults", nil, routeUser())
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

- [ ] **Step 2: Update route registration test**

Replace `TestSetupRegistersPmcsSbsProgressRoutesUnderAuth` in `api/route/route_test.go` with:

```go
func TestSetupRegistersPmcsSbsFaultRoutesUnderAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	Setup(nil, router, nil, nil, nil)

	requireRouteRegistered(t, router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults")
	requireRouteRegistered(t, router, http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults")
	requireRouteRegistered(t, router, http.MethodDelete, "/api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults")
}
```

- [ ] **Step 3: Run route tests to verify they fail**

Run:

```bash
go test ./api/pmcs_sbs_progress -run 'Test(Handlers|ListFaults|UpsertFault|InvalidJSON|DeleteFault|ServiceError)' -count=1
go test ./api/route -run TestSetupRegistersPmcsSbsFaultRoutesUnderAuth -count=1
```

Expected: FAIL because route registration still exposes equipment, completion, and sync routes and lacks `GET /faults`.

- [ ] **Step 4: Replace route implementation with fault-only handlers**

Replace `api/pmcs_sbs_progress/route.go` with:

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

	group.GET("/pmcs-sbs/equipment/:equipment_id/faults", handler.listFaults)
	group.PUT("/pmcs-sbs/equipment/:equipment_id/faults", handler.upsertFault)
	group.DELETE("/pmcs-sbs/equipment/:equipment_id/faults", handler.deleteFault)
}

func (handler Handler) listFaults(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	result, err := handler.service.ListFaults(user, c.Param("equipment_id"))
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

	result, err := handler.service.UpsertFault(user, c.Param("equipment_id"), req)
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

	if err := handler.service.DeleteFault(user, c.Param("equipment_id"), req); err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Fault deleted"})
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
		errors.Is(err, ErrInvalidRequest),
		errors.Is(err, ErrInvalidStatus):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "pmcs sbs equipment not found"})
	default:
		slog.Error("PMCS SBS fault handler failed", "error", err)
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
	}
}
```

- [ ] **Step 5: Run route tests**

Run:

```bash
go test ./api/pmcs_sbs_progress -run 'Test(Handlers|ListFaults|UpsertFault|InvalidJSON|DeleteFault|ServiceError)' -count=1
go test ./api/route -run TestSetupRegistersPmcsSbsFaultRoutesUnderAuth -count=1
```

Expected: PASS for route tests once repository stale compile errors are removed in Task 3. If package compile still fails due to repository references, continue to Task 3 and re-run.

- [ ] **Step 6: Commit Task 2**

```bash
git add api/pmcs_sbs_progress/route.go api/pmcs_sbs_progress/route_test.go api/route/route_test.go
git commit -m "refactor(pmcs-sbs): expose fault-only routes"
```

## Task 3: Fault Repository And Authorization

**Files:**
- Modify: `api/pmcs_sbs_progress/repository_impl.go`
- Test: `tests/pmcs_sbs_progress/helpers_test.go`
- Test: `tests/pmcs_sbs_progress/repository_test.go`

- [ ] **Step 1: Replace integration helpers**

Replace `tests/pmcs_sbs_progress/helpers_test.go` with:

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
		`INSERT INTO shops (id, name, details, creator_id, created_at, last_updated, admin_only_lists)
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

func sampleFault(equipmentID string) model.PmcsSbsFaults {
	now := time.Now().UTC()
	return model.PmcsSbsFaults{
		EquipmentID:      equipmentID,
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

- [ ] **Step 2: Replace repository integration tests**

Replace `tests/pmcs_sbs_progress/repository_test.go` with:

```go
package pmcs_sbs_progress_test

import (
	"testing"
	"time"

	"miltechserver/api/pmcs_sbs_progress"

	"github.com/stretchr/testify/require"
)

func TestRepositoryMemberCanListSaveAndDeleteFaults(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-member")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "A1")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	saved, err := repo.UpsertFault(user, sampleFault(vehicleID))
	require.NoError(t, err)
	require.Equal(t, vehicleID, saved.EquipmentID)
	require.Equal(t, "leak", saved.FaultText)

	list, err := repo.ListFaults(user, vehicleID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "before", list[0].SectionID)

	err = repo.DeleteFault(user, pmcs_sbs_progress.FaultKey{
		EquipmentID: vehicleID,
		SectionID:   "before",
		ItemIndex:   0,
	})
	require.NoError(t, err)

	list, err = repo.ListFaults(user, vehicleID)
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestRepositoryNonMemberCannotAccessFaults(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	owner := testUser("pmcs-fault-owner")
	other := testUser("pmcs-fault-other")
	ensureUser(t, testDB, owner)
	ensureUser(t, testDB, other)
	shopID := createShopWithMember(t, testDB, owner, "admin")
	vehicleID := createShopVehicle(t, testDB, shopID, owner, "A2")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	_, err := repo.ListFaults(other, vehicleID)
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)

	_, err = repo.UpsertFault(other, sampleFault(vehicleID))
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)

	err = repo.DeleteFault(other, pmcs_sbs_progress.FaultKey{EquipmentID: vehicleID, SectionID: "before", ItemIndex: 0})
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)
}

func TestRepositoryMissingVehicleReturnsNotFound(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-missing")
	ensureUser(t, testDB, user)
	repo := pmcs_sbs_progress.NewRepository(testDB)

	_, err := repo.ListFaults(user, "missing-vehicle")
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)

	_, err = repo.UpsertFault(user, sampleFault("missing-vehicle"))
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)
}

func TestRepositoryAnyShopMemberCanManageFaults(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	owner := testUser("pmcs-fault-shop-owner")
	member := testUser("pmcs-fault-shop-member")
	ensureUser(t, testDB, owner)
	ensureUser(t, testDB, member)
	shopID := createShopWithMember(t, testDB, owner, "admin")
	addShopMember(t, testDB, shopID, member, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, owner, "A3")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	_, err := repo.UpsertFault(member, sampleFault(vehicleID))
	require.NoError(t, err)

	list, err := repo.ListFaults(member, vehicleID)
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestRepositoryFaultUpsertPreservesCreatedAt(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-update")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "A4")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	first, err := repo.UpsertFault(user, sampleFault(vehicleID))
	require.NoError(t, err)
	time.Sleep(time.Millisecond)

	updatedFault := sampleFault(vehicleID)
	updatedFault.FaultText = "updated leak"
	updatedFault.CorrectiveAction = "tightened"
	second, err := repo.UpsertFault(user, updatedFault)
	require.NoError(t, err)

	require.Equal(t, first.CreatedAt, second.CreatedAt)
	require.True(t, second.UpdatedAt.After(first.UpdatedAt) || second.UpdatedAt.Equal(first.UpdatedAt))
	require.Equal(t, "updated leak", second.FaultText)
	require.Equal(t, "tightened", second.CorrectiveAction)
}

func TestRepositoryDeleteFaultIsIdempotentForAccessibleVehicle(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-delete-missing")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "A5")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	err := repo.DeleteFault(user, pmcs_sbs_progress.FaultKey{EquipmentID: vehicleID, SectionID: "before", ItemIndex: 0})

	require.NoError(t, err)
}

func TestRepositoryVehicleDeleteCascadesFaults(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-cascade")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "A6")
	repo := pmcs_sbs_progress.NewRepository(testDB)
	_, err := repo.UpsertFault(user, sampleFault(vehicleID))
	require.NoError(t, err)

	_, err = testDB.Exec(`DELETE FROM shop_vehicle WHERE id=$1`, vehicleID)
	require.NoError(t, err)

	var count int
	err = testDB.QueryRow(`SELECT COUNT(*) FROM pmcs_sbs_faults WHERE equipment_id=$1`, vehicleID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}
```

- [ ] **Step 3: Run repository tests to verify they fail**

Run:

```bash
go test ./tests/pmcs_sbs_progress -run TestRepository -count=1
```

Expected: FAIL because repository still references removed equipment/completion tables and UUID equipment IDs.

- [ ] **Step 4: Replace repository implementation**

Replace `api/pmcs_sbs_progress/repository_impl.go` with:

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
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) ListFaults(user *bootstrap.User, equipmentID string) ([]model.PmcsSbsFaults, error) {
	if err := repo.requireVehicleAccess(user, equipmentID); err != nil {
		return nil, err
	}

	var rows []model.PmcsSbsFaults
	stmt := SELECT(PmcsSbsFaults.AllColumns).
		FROM(PmcsSbsFaults).
		WHERE(PmcsSbsFaults.EquipmentID.EQ(String(equipmentID))).
		ORDER_BY(
			PmcsSbsFaults.SectionID.ASC(),
			PmcsSbsFaults.ItemIndex.ASC(),
		)

	if err := stmt.Query(repo.db, &rows); err != nil {
		return nil, fmt.Errorf("list pmcs sbs faults: %w", err)
	}
	return rows, nil
}

func (repo *RepositoryImpl) UpsertFault(user *bootstrap.User, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error) {
	if err := repo.requireVehicleAccess(user, fault.EquipmentID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if fault.CreatedAt.IsZero() {
		fault.CreatedAt = now
	}
	fault.UpdatedAt = now

	stmt := PmcsSbsFaults.INSERT(
		PmcsSbsFaults.EquipmentID,
		PmcsSbsFaults.SectionID,
		PmcsSbsFaults.ItemIndex,
		PmcsSbsFaults.ItemNo,
		PmcsSbsFaults.Status,
		PmcsSbsFaults.FaultText,
		PmcsSbsFaults.CorrectiveAction,
		PmcsSbsFaults.CreatedAt,
		PmcsSbsFaults.UpdatedAt,
	).VALUES(
		String(fault.EquipmentID),
		String(fault.SectionID),
		Int32(fault.ItemIndex),
		String(fault.ItemNo),
		String(fault.Status),
		String(fault.FaultText),
		String(fault.CorrectiveAction),
		TimestampzT(fault.CreatedAt),
		TimestampzT(now),
	).ON_CONFLICT(
		PmcsSbsFaults.EquipmentID,
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
	if err := stmt.Query(repo.db, &saved); err != nil {
		return nil, fmt.Errorf("upsert pmcs sbs fault: %w", err)
	}
	return &saved, nil
}

func (repo *RepositoryImpl) DeleteFault(user *bootstrap.User, key FaultKey) error {
	if err := repo.requireVehicleAccess(user, key.EquipmentID); err != nil {
		return err
	}

	if _, err := PmcsSbsFaults.DELETE().
		WHERE(
			PmcsSbsFaults.EquipmentID.EQ(String(key.EquipmentID)).
				AND(PmcsSbsFaults.SectionID.EQ(String(key.SectionID))).
				AND(PmcsSbsFaults.ItemIndex.EQ(Int32(key.ItemIndex))),
		).
		Exec(repo.db); err != nil {
		return fmt.Errorf("delete pmcs sbs fault: %w", err)
	}
	return nil
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
```

- [ ] **Step 5: Run repository tests**

Run:

```bash
go test ./tests/pmcs_sbs_progress -run TestRepository -count=1
```

Expected: PASS.

- [ ] **Step 6: Run PMCS SBS package tests**

Run:

```bash
go test ./api/pmcs_sbs_progress -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit Task 3**

```bash
git add api/pmcs_sbs_progress/repository_impl.go tests/pmcs_sbs_progress/helpers_test.go tests/pmcs_sbs_progress/repository_test.go
git commit -m "refactor(pmcs-sbs): authorize faults through shop vehicles"
```

## Task 4: Remove Stale API Documentation

**Files:**
- Modify: `docs/api/pmcs-sbs-progress-sync.md`
- Modify: `docs/api/pmcs-sbs-progress-sync-mobile.md`

- [ ] **Step 1: Replace concise API doc**

Replace `docs/api/pmcs-sbs-progress-sync.md` with:

```markdown
# PMCS SBS Faults API

Base URL: `/api/v1/auth`

The server no longer stores PMCS SBS guide progress, completions, or PMCS-owned equipment. PMCS SBS tracking remains client-side. The authenticated server API stores only PMCS SBS faults for existing `shop_vehicle` equipment.

The public PMCS SBS library API remains separate and continues to serve guide JSON from Azure Blob Storage.

## Authorization

All endpoints require Firebase authentication. The authenticated user must be a member of the shop that owns the target `shop_vehicle`.

Missing vehicles and vehicles in shops the user cannot access both return:

```json
{"message":"pmcs sbs equipment not found"}
```

## List Faults

`GET /pmcs-sbs/equipment/:equipment_id/faults`

Returns all PMCS SBS faults for the shop vehicle.

## Save Fault

`PUT /pmcs-sbs/equipment/:equipment_id/faults`

```json
{
  "section_id": "before",
  "item_index": 0,
  "item_no": "1",
  "status": "X",
  "fault_text": "Oil leak observed",
  "corrective_action": ""
}
```

Accepted status inputs are `X`, `x`, `/`, `slash`, `-`, and `dash`. Responses normalize to `x`, `slash`, or `dash`.

## Delete Fault

`DELETE /pmcs-sbs/equipment/:equipment_id/faults`

```json
{
  "section_id": "before",
  "item_index": 0
}
```

Deletes are idempotent after the user has access to the parent vehicle.
```

- [ ] **Step 2: Replace mobile API contract**

Replace `docs/api/pmcs-sbs-progress-sync-mobile.md` with:

```markdown
# PMCS SBS Faults Mobile API Contract

## API Summary

Base URL: `https://<host>/api/v1/auth`

Authentication: Firebase ID token in `Authorization: Bearer <token>`.

Content type: request bodies are JSON and should use `Content-Type: application/json`.

The server stores PMCS SBS faults only. PMCS SBS guide progress and completed-step tracking are client-side only. Equipment is owned by Shops and lives in `shop_vehicle`.

## Resource Model

### Fault

Faults are keyed by:

```text
(equipment_id, section_id, item_index)
```

`equipment_id` is the `shop_vehicle.id`.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `equipment_id` | string | Response only | `shop_vehicle.id`. In requests this comes from `:equipment_id`. |
| `section_id` | string | Yes | PMCS section id. Blank values are rejected. |
| `item_index` | integer | Yes | Zero-based item index in the section. Must be `0` or greater. |
| `item_no` | string | Save only | Display item number from the source PMCS item. Required for save requests. |
| `status` | string | Save only | Accepted input values are `X`, `x`, `/`, `slash`, `-`, and `dash`. Responses use `x`, `slash`, or `dash`. |
| `fault_text` | string | Save only | User-entered deficiency text. Blank values are rejected. |
| `corrective_action` | string | No | User-entered corrective action. Blank is accepted. |
| `created_at` | RFC 3339 timestamp | Response only | Server timestamp from initial insert. Preserved on update. |
| `updated_at` | RFC 3339 timestamp | Response only | Server timestamp from latest accepted write. |

## Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/pmcs-sbs/equipment/:equipment_id/faults` | List all PMCS SBS faults for a shop vehicle. |
| `PUT` | `/pmcs-sbs/equipment/:equipment_id/faults` | Create or update one PMCS SBS fault. |
| `DELETE` | `/pmcs-sbs/equipment/:equipment_id/faults` | Delete one PMCS SBS fault. |

## Common Errors

| Condition | HTTP status | Response body |
|-----------|-------------|---------------|
| Missing authorization header | `401` | `{"message":"No Authorization header found"}` |
| Authenticated user missing from handler context | `401` | `{"message":"unauthorized"}` |
| Invalid JSON body | `400` | `{"message":"invalid request body"}` |
| Blank equipment id | `400` | `{"message":"invalid id"}` |
| Invalid required fields | `400` | `{"message":"invalid request"}` |
| Invalid fault status | `400` | `{"message":"invalid fault status"}` |
| Missing or unauthorized vehicle | `404` | `{"message":"pmcs sbs equipment not found"}` |
| Unexpected server error | `500` | `{"status":500,"data":null,"message":"internal Server Error"}` |

## List Faults

`GET /pmcs-sbs/equipment/:equipment_id/faults`

Success response:

```json
{
  "status": 200,
  "data": {
    "faults": [
      {
        "equipment_id": "550e8400-e29b-41d4-a716-446655440000",
        "section_id": "before",
        "item_index": 0,
        "item_no": "1",
        "status": "x",
        "fault_text": "Oil leak observed",
        "corrective_action": "",
        "created_at": "2026-06-21T18:44:12.123456Z",
        "updated_at": "2026-06-21T19:12:03.654321Z"
      }
    ],
    "count": 1
  },
  "message": ""
}
```

An accessible vehicle with no faults returns:

```json
{
  "status": 200,
  "data": {
    "faults": [],
    "count": 0
  },
  "message": ""
}
```

## Save Fault

`PUT /pmcs-sbs/equipment/:equipment_id/faults`

Request:

```json
{
  "section_id": "before",
  "item_index": 0,
  "item_no": "1",
  "status": "X",
  "fault_text": "Oil leak observed",
  "corrective_action": ""
}
```

Success response:

```json
{
  "status": 200,
  "data": {
    "equipment_id": "550e8400-e29b-41d4-a716-446655440000",
    "section_id": "before",
    "item_index": 0,
    "item_no": "1",
    "status": "x",
    "fault_text": "Oil leak observed",
    "corrective_action": "",
    "created_at": "2026-06-21T18:44:12.123456Z",
    "updated_at": "2026-06-21T19:12:03.654321Z"
  },
  "message": "Fault saved"
}
```

## Delete Fault

`DELETE /pmcs-sbs/equipment/:equipment_id/faults`

Request:

```json
{
  "section_id": "before",
  "item_index": 0
}
```

Success response:

```json
{
  "message": "Fault deleted"
}
```

## Removed Server Behavior

The server no longer exposes PMCS SBS equipment create/update/delete/list endpoints, completion endpoints, or `POST /pmcs-sbs/sync`.
```

- [ ] **Step 3: Run stale docs search**

Run:

```bash
rg "upsert_equipment|delete_equipment_ids|upsert_completions|delete_completions|equipment_manual|pmcs_sbs_completions|pmcs_sbs_equipment" docs/api/pmcs-sbs-progress-sync.md docs/api/pmcs-sbs-progress-sync-mobile.md
```

Expected: no matches.

- [ ] **Step 4: Commit Task 4**

```bash
git add docs/api/pmcs-sbs-progress-sync.md docs/api/pmcs-sbs-progress-sync-mobile.md
git commit -m "docs(pmcs-sbs): document faults-only API"
```

## Task 5: Final Stale Reference Cleanup And Verification

**Files:**
- Inspect: `api/pmcs_sbs_progress/*`
- Inspect: `tests/pmcs_sbs_progress/*`
- Inspect: `docs/api/pmcs-sbs-progress-sync.md`
- Inspect: `docs/api/pmcs-sbs-progress-sync-mobile.md`

- [ ] **Step 1: Run stale reference guard**

Run:

```bash
rg "PmcsSbsEquipment|PmcsSbsCompletions|pmcs_sbs_equipment|pmcs_sbs_completions" api/pmcs_sbs_progress tests/pmcs_sbs_progress docs/api/pmcs-sbs-progress-sync.md docs/api/pmcs-sbs-progress-sync-mobile.md
```

Expected: no matches.

- [ ] **Step 2: Run route surface guard**

Run:

```bash
rg "pmcs-sbs/equipment\"|completions|pmcs-sbs/sync|listEquipment|upsertEquipment|deleteEquipment|sync\\(" api/pmcs_sbs_progress api/route/route_test.go
```

Expected: no matches except the three exact fault routes containing `/pmcs-sbs/equipment/:equipment_id/faults`.

- [ ] **Step 3: Run focused tests**

Run:

```bash
go test ./api/pmcs_sbs_progress -count=1
go test ./tests/pmcs_sbs_progress -count=1
go test ./api/route -count=1
```

Expected: PASS.

- [ ] **Step 4: Run broader PMCS SBS route/library smoke tests**

Run:

```bash
go test ./api/library/pmcs_sbs ./api/route -count=1
```

Expected: PASS. This confirms the public PMCS SBS library API still compiles and route setup remains healthy.

- [ ] **Step 5: Run optional full-suite health check**

Run:

```bash
go test ./... -count=1
```

Expected: PASS, or clearly document unrelated baseline failures separately from the PMCS SBS faults-only refactor.

- [ ] **Step 6: Commit final cleanup if any files changed after Task 4**

Only run this commit if stale-reference cleanup required additional edits:

```bash
git add api/pmcs_sbs_progress tests/pmcs_sbs_progress api/route/route_test.go docs/api/pmcs-sbs-progress-sync.md docs/api/pmcs-sbs-progress-sync-mobile.md
git commit -m "test(pmcs-sbs): verify faults-only contract"
```

## Final Handoff Notes

- Do not edit generated Jet files by hand.
- Do not touch Shops behavior beyond inserting test fixtures into `shop_vehicle` and `shop_members`.
- Leave unrelated dirty files alone, especially `docs/api/shop_equipment_overview_mobile.md` if it is still modified.
- The accepted route wording intentionally keeps `equipment` in the URL even though `equipment_id` now means `shop_vehicle.id`.
