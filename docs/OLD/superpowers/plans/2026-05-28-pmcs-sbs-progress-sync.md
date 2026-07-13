# PMCS SBS Progress Sync Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add authenticated PMCS SBS progress sync so logged-in users can sync equipment, completed steps, and item faults across devices.

**Architecture:** Add a new `api/pmcs_sbs_progress` package registered under `/api/v1/auth`. Keep the existing public `api/library/pmcs_sbs` package unchanged. Use service-level validation and ownership checks, Jet repository methods for Postgres, and one transaction-backed batch sync endpoint for offline replay.

**Tech Stack:** Go 1.23, Gin, Postgres, Jet, Firebase auth context from `bootstrap.User`, generated models/tables in `.gen/miltech_ng/public`.

---

## File Structure

Create:

- `api/pmcs_sbs_progress/errors.go` - sentinel errors used by service and handler response mapping.
- `api/pmcs_sbs_progress/types.go` - API request and response DTOs.
- `api/pmcs_sbs_progress/service.go` - service interface.
- `api/pmcs_sbs_progress/service_impl.go` - validation, ownership-oriented business rules, and sync contradiction checks.
- `api/pmcs_sbs_progress/service_impl_test.go` - focused validation and service behavior tests.
- `api/pmcs_sbs_progress/repository.go` - repository interface.
- `api/pmcs_sbs_progress/repository_impl.go` - Jet/Postgres persistence.
- `api/pmcs_sbs_progress/route.go` - Gin route registration and handlers.
- `api/pmcs_sbs_progress/route_test.go` - handler tests with a service stub.
- `tests/pmcs_sbs_progress/main_test.go` - integration test database setup.
- `tests/pmcs_sbs_progress/helpers_test.go` - test router, request helpers, table cleanup, fixture helpers.
- `tests/pmcs_sbs_progress/repository_test.go` - integration tests for repository behavior and user isolation.
- `docs/api/pmcs-sbs-progress-sync.md` - mobile integration guide for authenticated sync endpoints.

Modify:

- `api/route/route.go` - register `pmcs_sbs_progress.RegisterRoutes` under `authRoutes`.

---

## Shared Types To Implement

Use these names consistently across tasks.

```go
type EquipmentRequest struct {
	EquipmentManual string `json:"equipment_manual"`
	Admin           string `json:"admin"`
	Serial          string `json:"serial"`
	Uic             string `json:"uic"`
}

type CompletionRequest struct {
	SectionID string `json:"section_id"`
	ItemIndex int32  `json:"item_index"`
	ItemNo    string `json:"item_no"`
	StepID    string `json:"step_id"`
}

type DeleteCompletionRequest struct {
	SectionID string `json:"section_id"`
	ItemIndex int32  `json:"item_index"`
	StepID    string `json:"step_id"`
}

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

type SyncRequest struct {
	UpsertEquipment   []SyncEquipmentRequest         `json:"upsert_equipment"`
	DeleteEquipmentIDs []string                       `json:"delete_equipment_ids"`
	UpsertCompletions []SyncCompletionRequest        `json:"upsert_completions"`
	DeleteCompletions []SyncDeleteCompletionRequest  `json:"delete_completions"`
	UpsertFaults      []SyncFaultRequest             `json:"upsert_faults"`
	DeleteFaults      []SyncDeleteFaultRequest       `json:"delete_faults"`
}

type SyncEquipmentRequest struct {
	ID              string `json:"id"`
	EquipmentManual string `json:"equipment_manual"`
	Admin           string `json:"admin"`
	Serial          string `json:"serial"`
	Uic             string `json:"uic"`
}

type SyncCompletionRequest struct {
	EquipmentID string `json:"equipment_id"`
	SectionID   string `json:"section_id"`
	ItemIndex   int32  `json:"item_index"`
	ItemNo      string `json:"item_no"`
	StepID      string `json:"step_id"`
}

type SyncDeleteCompletionRequest struct {
	EquipmentID string `json:"equipment_id"`
	SectionID   string `json:"section_id"`
	ItemIndex   int32  `json:"item_index"`
	StepID      string `json:"step_id"`
}

type SyncFaultRequest struct {
	EquipmentID      string `json:"equipment_id"`
	SectionID        string `json:"section_id"`
	ItemIndex        int32  `json:"item_index"`
	ItemNo           string `json:"item_no"`
	Status           string `json:"status"`
	FaultText        string `json:"fault_text"`
	CorrectiveAction string `json:"corrective_action"`
}

type SyncDeleteFaultRequest struct {
	EquipmentID string `json:"equipment_id"`
	SectionID   string `json:"section_id"`
	ItemIndex   int32  `json:"item_index"`
}

type EquipmentAggregateResponse struct {
	Equipment   EquipmentResponse    `json:"equipment"`
	Completions []CompletionResponse `json:"completions"`
	Faults      []FaultResponse      `json:"faults"`
}

type EquipmentListResponse struct {
	Equipment []EquipmentResponse `json:"equipment"`
	Count     int                 `json:"count"`
}

type SyncResponse struct {
	Equipment          []EquipmentAggregateResponse `json:"equipment"`
	DeletedEquipmentIDs []string                    `json:"deleted_equipment_ids"`
}
```

---

### Task 1: Foundation Types, Errors, And Validation

**Files:**
- Create: `api/pmcs_sbs_progress/errors.go`
- Create: `api/pmcs_sbs_progress/types.go`
- Create: `api/pmcs_sbs_progress/service.go`
- Create: `api/pmcs_sbs_progress/repository.go`
- Create: `api/pmcs_sbs_progress/service_impl.go`
- Create: `api/pmcs_sbs_progress/service_impl_test.go`

- [ ] **Step 1: Create validation tests first**

Create `api/pmcs_sbs_progress/service_impl_test.go` with tests for the field rules from the design.

```go
package pmcs_sbs_progress

import (
	"errors"
	"testing"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"

	"github.com/stretchr/testify/require"
)

type repoStub struct{}

func (repo *repoStub) ListEquipment(user *bootstrap.User) ([]model.PmcsSbsEquipment, error) {
	return nil, nil
}

func (repo *repoStub) GetEquipmentAggregate(user *bootstrap.User, equipmentID string) (*EquipmentAggregate, error) {
	return nil, nil
}

func (repo *repoStub) UpsertEquipment(user *bootstrap.User, equipment model.PmcsSbsEquipment) (*model.PmcsSbsEquipment, error) {
	return &equipment, nil
}

func (repo *repoStub) DeleteEquipment(user *bootstrap.User, equipmentID string) error {
	return nil
}

func (repo *repoStub) UpsertCompletion(user *bootstrap.User, completion model.PmcsSbsCompletions) (*model.PmcsSbsCompletions, error) {
	return &completion, nil
}

func (repo *repoStub) DeleteCompletion(user *bootstrap.User, equipmentID string, sectionID string, itemIndex int32, stepID string) error {
	return nil
}

func (repo *repoStub) UpsertFault(user *bootstrap.User, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error) {
	return &fault, nil
}

func (repo *repoStub) DeleteFault(user *bootstrap.User, equipmentID string, sectionID string, itemIndex int32) error {
	return nil
}

func (repo *repoStub) Sync(user *bootstrap.User, changeSet SyncChangeSet) (*SyncResult, error) {
	return &SyncResult{}, nil
}

func TestValidateEquipmentRequest(t *testing.T) {
	svc := NewService(&repoStub{})

	req, err := svc.validateEquipmentRequest("550e8400-e29b-41d4-a716-446655440000", EquipmentRequest{
		EquipmentManual: " pmcs_sbs/hmmwv/basic.json ",
		Admin:           " A12 ",
		Serial:          " SER ",
		Uic:             " UIC ",
	})

	require.NoError(t, err)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", req.ID.String())
	require.Equal(t, "pmcs_sbs/hmmwv/basic.json", req.EquipmentManual)
	require.Equal(t, "A12", req.Admin)
	require.Equal(t, "SER", req.Serial)
	require.Equal(t, "UIC", req.Uic)
}

func TestValidateEquipmentRequestRejectsInvalidValues(t *testing.T) {
	svc := NewService(&repoStub{})

	cases := []struct {
		name        string
		equipmentID string
		req         EquipmentRequest
		want        error
	}{
		{
			name:        "invalid uuid",
			equipmentID: "not-a-uuid",
			req:         EquipmentRequest{EquipmentManual: "pmcs_sbs/hmmwv/basic.json", Admin: "A12"},
			want:        ErrInvalidID,
		},
		{
			name:        "bad blob prefix",
			equipmentID: "550e8400-e29b-41d4-a716-446655440000",
			req:         EquipmentRequest{EquipmentManual: "pmcs/hmmwv/basic.json", Admin: "A12"},
			want:        ErrInvalidBlobPath,
		},
		{
			name:        "bad blob extension",
			equipmentID: "550e8400-e29b-41d4-a716-446655440000",
			req:         EquipmentRequest{EquipmentManual: "pmcs_sbs/hmmwv/basic.pdf", Admin: "A12"},
			want:        ErrInvalidBlobPath,
		},
		{
			name:        "path traversal",
			equipmentID: "550e8400-e29b-41d4-a716-446655440000",
			req:         EquipmentRequest{EquipmentManual: "pmcs_sbs/../secret.json", Admin: "A12"},
			want:        ErrInvalidBlobPath,
		},
		{
			name:        "missing admin",
			equipmentID: "550e8400-e29b-41d4-a716-446655440000",
			req:         EquipmentRequest{EquipmentManual: "pmcs_sbs/hmmwv/basic.json", Admin: " "},
			want:        ErrInvalidRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.validateEquipmentRequest(tc.equipmentID, tc.req)
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestValidateCompletionRequest(t *testing.T) {
	svc := NewService(&repoStub{})

	completion, err := svc.validateCompletionRequest("550e8400-e29b-41d4-a716-446655440000", CompletionRequest{
		SectionID: " before ",
		ItemIndex: 0,
		ItemNo:    " 1 ",
		StepID:    " 1-a ",
	})

	require.NoError(t, err)
	require.Equal(t, "before", completion.SectionID)
	require.Equal(t, int32(0), completion.ItemIndex)
	require.Equal(t, "1", completion.ItemNo)
	require.Equal(t, "1-a", completion.StepID)
	require.True(t, completion.IsComplete)
}

func TestValidateCompletionRequestRejectsInvalidValues(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.validateCompletionRequest("bad", CompletionRequest{SectionID: "before", ItemIndex: 0, ItemNo: "1", StepID: "1-a"})
	require.ErrorIs(t, err, ErrInvalidID)

	_, err = svc.validateCompletionRequest("550e8400-e29b-41d4-a716-446655440000", CompletionRequest{SectionID: "", ItemIndex: 0, ItemNo: "1", StepID: "1-a"})
	require.ErrorIs(t, err, ErrInvalidRequest)

	_, err = svc.validateCompletionRequest("550e8400-e29b-41d4-a716-446655440000", CompletionRequest{SectionID: "before", ItemIndex: -1, ItemNo: "1", StepID: "1-a"})
	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestValidateFaultRequest(t *testing.T) {
	svc := NewService(&repoStub{})

	fault, err := svc.validateFaultRequest("550e8400-e29b-41d4-a716-446655440000", FaultRequest{
		SectionID:        " before ",
		ItemIndex:        0,
		ItemNo:           " 1 ",
		Status:           "X",
		FaultText:        " leak ",
		CorrectiveAction: " tighten ",
	})

	require.NoError(t, err)
	require.Equal(t, "before", fault.SectionID)
	require.Equal(t, "1", fault.ItemNo)
	require.Equal(t, "X", fault.Status)
	require.Equal(t, "leak", fault.FaultText)
	require.Equal(t, "tighten", fault.CorrectiveAction)
}

func TestValidateFaultRequestRejectsInvalidValues(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.validateFaultRequest("550e8400-e29b-41d4-a716-446655440000", FaultRequest{SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "BAD", FaultText: "leak"})
	require.ErrorIs(t, err, ErrInvalidStatus)

	_, err = svc.validateFaultRequest("550e8400-e29b-41d4-a716-446655440000", FaultRequest{SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "X", FaultText: " "})
	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestValidateSyncRequestRejectsContradictions(t *testing.T) {
	svc := NewService(&repoStub{})

	err := svc.validateSyncRequest(SyncRequest{
		DeleteEquipmentIDs: []string{"550e8400-e29b-41d4-a716-446655440000"},
		UpsertFaults: []SyncFaultRequest{{
			EquipmentID: "550e8400-e29b-41d4-a716-446655440000",
			SectionID:   "before",
			ItemIndex:   0,
			ItemNo:      "1",
			Status:      "X",
			FaultText:   "leak",
		}},
	})
	require.ErrorIs(t, err, ErrInvalidSyncRequest)

	err = svc.validateSyncRequest(SyncRequest{
		UpsertCompletions: []SyncCompletionRequest{{
			EquipmentID: "550e8400-e29b-41d4-a716-446655440000",
			SectionID:   "before",
			ItemIndex:   0,
			ItemNo:      "1",
			StepID:      "1-a",
		}},
		DeleteCompletions: []SyncDeleteCompletionRequest{{
			EquipmentID: "550e8400-e29b-41d4-a716-446655440000",
			SectionID:   "before",
			ItemIndex:   0,
			StepID:      "1-a",
		}},
	})
	require.ErrorIs(t, err, ErrInvalidSyncRequest)
}

func requireUser() *bootstrap.User {
	return &bootstrap.User{UserID: "user-1", Username: "tester", Email: "user-1@example.com"}
}

func requireServiceError(t *testing.T, err error, target error) {
	t.Helper()
	require.True(t, errors.Is(err, target), "expected %v, got %v", target, err)
}
```

- [ ] **Step 2: Run the failing validation tests**

Run:

```bash
go test ./api/pmcs_sbs_progress -run 'TestValidate' -count=1
```

Expected: fail because `api/pmcs_sbs_progress` does not exist yet.

- [ ] **Step 3: Add errors, DTOs, interfaces, and validation implementation**

Create `api/pmcs_sbs_progress/errors.go`.

```go
package pmcs_sbs_progress

import "errors"

var (
	ErrInvalidID          = errors.New("invalid id")
	ErrInvalidRequest     = errors.New("invalid request")
	ErrInvalidBlobPath    = errors.New("invalid equipment manual blob path")
	ErrInvalidStatus      = errors.New("invalid fault status")
	ErrInvalidSyncRequest = errors.New("invalid sync request")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrNotFound           = errors.New("pmcs sbs equipment not found")
)
```

Create `api/pmcs_sbs_progress/types.go` with the shared types from the "Shared Types To Implement" section plus response DTOs.

```go
package pmcs_sbs_progress

import "time"

type EquipmentRequest struct {
	EquipmentManual string `json:"equipment_manual"`
	Admin           string `json:"admin"`
	Serial          string `json:"serial"`
	Uic             string `json:"uic"`
}

type CompletionRequest struct {
	SectionID string `json:"section_id"`
	ItemIndex int32  `json:"item_index"`
	ItemNo    string `json:"item_no"`
	StepID    string `json:"step_id"`
}

type DeleteCompletionRequest struct {
	SectionID string `json:"section_id"`
	ItemIndex int32  `json:"item_index"`
	StepID    string `json:"step_id"`
}

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

type SyncRequest struct {
	UpsertEquipment    []SyncEquipmentRequest        `json:"upsert_equipment"`
	DeleteEquipmentIDs []string                      `json:"delete_equipment_ids"`
	UpsertCompletions  []SyncCompletionRequest       `json:"upsert_completions"`
	DeleteCompletions  []SyncDeleteCompletionRequest `json:"delete_completions"`
	UpsertFaults       []SyncFaultRequest            `json:"upsert_faults"`
	DeleteFaults       []SyncDeleteFaultRequest      `json:"delete_faults"`
}

type SyncEquipmentRequest struct {
	ID              string `json:"id"`
	EquipmentManual string `json:"equipment_manual"`
	Admin           string `json:"admin"`
	Serial          string `json:"serial"`
	Uic             string `json:"uic"`
}

type SyncCompletionRequest struct {
	EquipmentID string `json:"equipment_id"`
	SectionID   string `json:"section_id"`
	ItemIndex   int32  `json:"item_index"`
	ItemNo      string `json:"item_no"`
	StepID      string `json:"step_id"`
}

type SyncDeleteCompletionRequest struct {
	EquipmentID string `json:"equipment_id"`
	SectionID   string `json:"section_id"`
	ItemIndex   int32  `json:"item_index"`
	StepID      string `json:"step_id"`
}

type SyncFaultRequest struct {
	EquipmentID      string `json:"equipment_id"`
	SectionID        string `json:"section_id"`
	ItemIndex        int32  `json:"item_index"`
	ItemNo           string `json:"item_no"`
	Status           string `json:"status"`
	FaultText        string `json:"fault_text"`
	CorrectiveAction string `json:"corrective_action"`
}

type SyncDeleteFaultRequest struct {
	EquipmentID string `json:"equipment_id"`
	SectionID   string `json:"section_id"`
	ItemIndex   int32  `json:"item_index"`
}

type EquipmentResponse struct {
	ID              string    `json:"id"`
	EquipmentManual string    `json:"equipment_manual"`
	Admin           string    `json:"admin"`
	Serial          string    `json:"serial"`
	Uic             string    `json:"uic"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CompletionResponse struct {
	EquipmentID string    `json:"equipment_id"`
	SectionID   string    `json:"section_id"`
	ItemIndex   int32     `json:"item_index"`
	ItemNo      string    `json:"item_no"`
	StepID      string    `json:"step_id"`
	IsComplete  bool      `json:"is_complete"`
	UpdatedAt   time.Time `json:"updated_at"`
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

type EquipmentListResponse struct {
	Equipment []EquipmentResponse `json:"equipment"`
	Count     int                 `json:"count"`
}

type EquipmentAggregateResponse struct {
	Equipment   EquipmentResponse    `json:"equipment"`
	Completions []CompletionResponse `json:"completions"`
	Faults      []FaultResponse      `json:"faults"`
}

type SyncResponse struct {
	Equipment           []EquipmentAggregateResponse `json:"equipment"`
	DeletedEquipmentIDs []string                     `json:"deleted_equipment_ids"`
}
```

Create `api/pmcs_sbs_progress/service.go`.

```go
package pmcs_sbs_progress

import "miltechserver/bootstrap"

type Service interface {
	ListEquipment(user *bootstrap.User) (*EquipmentListResponse, error)
	GetEquipment(user *bootstrap.User, equipmentID string) (*EquipmentAggregateResponse, error)
	UpsertEquipment(user *bootstrap.User, equipmentID string, req EquipmentRequest) (*EquipmentResponse, error)
	DeleteEquipment(user *bootstrap.User, equipmentID string) error
	UpsertCompletion(user *bootstrap.User, equipmentID string, req CompletionRequest) (*CompletionResponse, error)
	DeleteCompletion(user *bootstrap.User, equipmentID string, req DeleteCompletionRequest) error
	UpsertFault(user *bootstrap.User, equipmentID string, req FaultRequest) (*FaultResponse, error)
	DeleteFault(user *bootstrap.User, equipmentID string, req DeleteFaultRequest) error
	Sync(user *bootstrap.User, req SyncRequest) (*SyncResponse, error)
}
```

Create `api/pmcs_sbs_progress/repository.go`.

```go
package pmcs_sbs_progress

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Repository interface {
	ListEquipment(user *bootstrap.User) ([]model.PmcsSbsEquipment, error)
	GetEquipmentAggregate(user *bootstrap.User, equipmentID string) (*EquipmentAggregate, error)
	UpsertEquipment(user *bootstrap.User, equipment model.PmcsSbsEquipment) (*model.PmcsSbsEquipment, error)
	DeleteEquipment(user *bootstrap.User, equipmentID string) error
	UpsertCompletion(user *bootstrap.User, completion model.PmcsSbsCompletions) (*model.PmcsSbsCompletions, error)
	DeleteCompletion(user *bootstrap.User, equipmentID string, sectionID string, itemIndex int32, stepID string) error
	UpsertFault(user *bootstrap.User, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error)
	DeleteFault(user *bootstrap.User, equipmentID string, sectionID string, itemIndex int32) error
	Sync(user *bootstrap.User, changeSet SyncChangeSet) (*SyncResult, error)
}

type EquipmentAggregate struct {
	Equipment   model.PmcsSbsEquipment
	Completions []model.PmcsSbsCompletions
	Faults      []model.PmcsSbsFaults
}

type SyncChangeSet struct {
	UpsertEquipment    []model.PmcsSbsEquipment
	DeleteEquipmentIDs []string
	UpsertCompletions  []model.PmcsSbsCompletions
	DeleteCompletions  []CompletionKey
	UpsertFaults       []model.PmcsSbsFaults
	DeleteFaults       []FaultKey
}

type CompletionKey struct {
	EquipmentID string
	SectionID   string
	ItemIndex   int32
	StepID      string
}

type FaultKey struct {
	EquipmentID string
	SectionID   string
	ItemIndex   int32
}

type SyncResult struct {
	Equipment           []EquipmentAggregate
	DeletedEquipmentIDs []string
}
```

Create `api/pmcs_sbs_progress/service_impl.go`.

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

type validatedEquipment struct {
	ID              uuid.UUID
	EquipmentManual string
	Admin           string
	Serial          string
	Uic             string
}

func (service *ServiceImpl) ListEquipment(user *bootstrap.User) (*EquipmentListResponse, error) {
	if user == nil || strings.TrimSpace(user.UserID) == "" {
		return nil, ErrUnauthorized
	}
	rows, err := service.repository.ListEquipment(user)
	if err != nil {
		return nil, err
	}
	equipment := make([]EquipmentResponse, 0, len(rows))
	for _, row := range rows {
		equipment = append(equipment, mapEquipment(row))
	}
	return &EquipmentListResponse{Equipment: equipment, Count: len(equipment)}, nil
}

func (service *ServiceImpl) GetEquipment(user *bootstrap.User, equipmentID string) (*EquipmentAggregateResponse, error) {
	if user == nil || strings.TrimSpace(user.UserID) == "" {
		return nil, ErrUnauthorized
	}
	if _, err := uuid.Parse(equipmentID); err != nil {
		return nil, ErrInvalidID
	}
	aggregate, err := service.repository.GetEquipmentAggregate(user, equipmentID)
	if err != nil {
		return nil, err
	}
	return mapAggregate(*aggregate), nil
}

func (service *ServiceImpl) UpsertEquipment(user *bootstrap.User, equipmentID string, req EquipmentRequest) (*EquipmentResponse, error) {
	if user == nil || strings.TrimSpace(user.UserID) == "" {
		return nil, ErrUnauthorized
	}
	validated, err := service.validateEquipmentRequest(equipmentID, req)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	row := model.PmcsSbsEquipment{
		ID:              validated.ID,
		UserUID:         user.UserID,
		EquipmentManual: validated.EquipmentManual,
		Admin:           validated.Admin,
		Serial:          validated.Serial,
		Uic:             validated.Uic,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	saved, err := service.repository.UpsertEquipment(user, row)
	if err != nil {
		return nil, err
	}
	resp := mapEquipment(*saved)
	return &resp, nil
}

func (service *ServiceImpl) DeleteEquipment(user *bootstrap.User, equipmentID string) error {
	if user == nil || strings.TrimSpace(user.UserID) == "" {
		return ErrUnauthorized
	}
	if _, err := uuid.Parse(equipmentID); err != nil {
		return ErrInvalidID
	}
	return service.repository.DeleteEquipment(user, equipmentID)
}

func (service *ServiceImpl) UpsertCompletion(user *bootstrap.User, equipmentID string, req CompletionRequest) (*CompletionResponse, error) {
	if user == nil || strings.TrimSpace(user.UserID) == "" {
		return nil, ErrUnauthorized
	}
	row, err := service.validateCompletionRequest(equipmentID, req)
	if err != nil {
		return nil, err
	}
	saved, err := service.repository.UpsertCompletion(user, row)
	if err != nil {
		return nil, err
	}
	resp := mapCompletion(*saved)
	return &resp, nil
}

func (service *ServiceImpl) DeleteCompletion(user *bootstrap.User, equipmentID string, req DeleteCompletionRequest) error {
	if user == nil || strings.TrimSpace(user.UserID) == "" {
		return ErrUnauthorized
	}
	row, err := service.validateDeleteCompletionRequest(equipmentID, req)
	if err != nil {
		return err
	}
	return service.repository.DeleteCompletion(user, row.EquipmentID.String(), row.SectionID, row.ItemIndex, row.StepID)
}

func (service *ServiceImpl) UpsertFault(user *bootstrap.User, equipmentID string, req FaultRequest) (*FaultResponse, error) {
	if user == nil || strings.TrimSpace(user.UserID) == "" {
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
	if user == nil || strings.TrimSpace(user.UserID) == "" {
		return ErrUnauthorized
	}
	key, err := service.validateDeleteFaultRequest(equipmentID, req)
	if err != nil {
		return err
	}
	return service.repository.DeleteFault(user, key.EquipmentID, key.SectionID, key.ItemIndex)
}

func (service *ServiceImpl) Sync(user *bootstrap.User, req SyncRequest) (*SyncResponse, error) {
	if user == nil || strings.TrimSpace(user.UserID) == "" {
		return nil, ErrUnauthorized
	}
	if err := service.validateSyncRequest(req); err != nil {
		return nil, err
	}
	changeSet, err := service.buildSyncChangeSet(user, req)
	if err != nil {
		return nil, err
	}
	result, err := service.repository.Sync(user, changeSet)
	if err != nil {
		return nil, err
	}
	resp := &SyncResponse{DeletedEquipmentIDs: result.DeletedEquipmentIDs}
	for _, aggregate := range result.Equipment {
		resp.Equipment = append(resp.Equipment, *mapAggregate(aggregate))
	}
	return resp, nil
}

func (service *ServiceImpl) validateEquipmentRequest(equipmentID string, req EquipmentRequest) (validatedEquipment, error) {
	id, err := uuid.Parse(strings.TrimSpace(equipmentID))
	if err != nil {
		return validatedEquipment{}, ErrInvalidID
	}
	equipmentManual := strings.TrimSpace(req.EquipmentManual)
	if !isValidEquipmentManual(equipmentManual) {
		return validatedEquipment{}, ErrInvalidBlobPath
	}
	admin := strings.TrimSpace(req.Admin)
	if admin == "" {
		return validatedEquipment{}, ErrInvalidRequest
	}
	return validatedEquipment{
		ID:              id,
		EquipmentManual: equipmentManual,
		Admin:           admin,
		Serial:          strings.TrimSpace(req.Serial),
		Uic:             strings.TrimSpace(req.Uic),
	}, nil
}

func (service *ServiceImpl) validateCompletionRequest(equipmentID string, req CompletionRequest) (model.PmcsSbsCompletions, error) {
	id, err := uuid.Parse(strings.TrimSpace(equipmentID))
	if err != nil {
		return model.PmcsSbsCompletions{}, ErrInvalidID
	}
	sectionID := strings.TrimSpace(req.SectionID)
	itemNo := strings.TrimSpace(req.ItemNo)
	stepID := strings.TrimSpace(req.StepID)
	if sectionID == "" || itemNo == "" || stepID == "" || req.ItemIndex < 0 {
		return model.PmcsSbsCompletions{}, ErrInvalidRequest
	}
	return model.PmcsSbsCompletions{
		EquipmentID: id,
		SectionID:   sectionID,
		ItemIndex:   req.ItemIndex,
		ItemNo:      itemNo,
		StepID:      stepID,
		IsComplete:  true,
		UpdatedAt:   time.Now().UTC(),
	}, nil
}

func (service *ServiceImpl) validateDeleteCompletionRequest(equipmentID string, req DeleteCompletionRequest) (model.PmcsSbsCompletions, error) {
	return service.validateCompletionRequest(equipmentID, CompletionRequest{
		SectionID: req.SectionID,
		ItemIndex: req.ItemIndex,
		ItemNo:    "delete",
		StepID:    req.StepID,
	})
}

func (service *ServiceImpl) validateFaultRequest(equipmentID string, req FaultRequest) (model.PmcsSbsFaults, error) {
	id, err := uuid.Parse(strings.TrimSpace(equipmentID))
	if err != nil {
		return model.PmcsSbsFaults{}, ErrInvalidID
	}
	sectionID := strings.TrimSpace(req.SectionID)
	itemNo := strings.TrimSpace(req.ItemNo)
	status := strings.TrimSpace(req.Status)
	faultText := strings.TrimSpace(req.FaultText)
	if sectionID == "" || itemNo == "" || req.ItemIndex < 0 || faultText == "" {
		return model.PmcsSbsFaults{}, ErrInvalidRequest
	}
	if !isValidFaultStatus(status) {
		return model.PmcsSbsFaults{}, ErrInvalidStatus
	}
	now := time.Now().UTC()
	return model.PmcsSbsFaults{
		EquipmentID:      id,
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
	if _, err := uuid.Parse(strings.TrimSpace(equipmentID)); err != nil {
		return FaultKey{}, ErrInvalidID
	}
	sectionID := strings.TrimSpace(req.SectionID)
	if sectionID == "" || req.ItemIndex < 0 {
		return FaultKey{}, ErrInvalidRequest
	}
	return FaultKey{EquipmentID: strings.TrimSpace(equipmentID), SectionID: sectionID, ItemIndex: req.ItemIndex}, nil
}

func (service *ServiceImpl) validateSyncRequest(req SyncRequest) error {
	deletedEquipment := map[string]struct{}{}
	for _, id := range req.DeleteEquipmentIDs {
		trimmed := strings.TrimSpace(id)
		if _, err := uuid.Parse(trimmed); err != nil {
			return ErrInvalidID
		}
		deletedEquipment[trimmed] = struct{}{}
	}
	completionUpserts := map[string]struct{}{}
	for _, completion := range req.UpsertCompletions {
		key := completionKey(completion.EquipmentID, completion.SectionID, completion.ItemIndex, completion.StepID)
		if _, deleted := deletedEquipment[strings.TrimSpace(completion.EquipmentID)]; deleted {
			return ErrInvalidSyncRequest
		}
		completionUpserts[key] = struct{}{}
	}
	for _, completion := range req.DeleteCompletions {
		key := completionKey(completion.EquipmentID, completion.SectionID, completion.ItemIndex, completion.StepID)
		if _, duplicate := completionUpserts[key]; duplicate {
			return ErrInvalidSyncRequest
		}
	}
	faultUpserts := map[string]struct{}{}
	for _, fault := range req.UpsertFaults {
		key := faultKey(fault.EquipmentID, fault.SectionID, fault.ItemIndex)
		if _, deleted := deletedEquipment[strings.TrimSpace(fault.EquipmentID)]; deleted {
			return ErrInvalidSyncRequest
		}
		faultUpserts[key] = struct{}{}
	}
	for _, fault := range req.DeleteFaults {
		key := faultKey(fault.EquipmentID, fault.SectionID, fault.ItemIndex)
		if _, duplicate := faultUpserts[key]; duplicate {
			return ErrInvalidSyncRequest
		}
	}
	return nil
}

func (service *ServiceImpl) buildSyncChangeSet(user *bootstrap.User, req SyncRequest) (SyncChangeSet, error) {
	changeSet := SyncChangeSet{DeleteEquipmentIDs: req.DeleteEquipmentIDs}
	for _, equipment := range req.UpsertEquipment {
		validated, err := service.validateEquipmentRequest(equipment.ID, EquipmentRequest{
			EquipmentManual: equipment.EquipmentManual,
			Admin:           equipment.Admin,
			Serial:          equipment.Serial,
			Uic:             equipment.Uic,
		})
		if err != nil {
			return SyncChangeSet{}, err
		}
		now := time.Now().UTC()
		changeSet.UpsertEquipment = append(changeSet.UpsertEquipment, model.PmcsSbsEquipment{
			ID:              validated.ID,
			UserUID:         user.UserID,
			EquipmentManual: validated.EquipmentManual,
			Admin:           validated.Admin,
			Serial:          validated.Serial,
			Uic:             validated.Uic,
			CreatedAt:       now,
			UpdatedAt:       now,
		})
	}
	for _, completion := range req.UpsertCompletions {
		row, err := service.validateCompletionRequest(completion.EquipmentID, CompletionRequest{
			SectionID: completion.SectionID,
			ItemIndex: completion.ItemIndex,
			ItemNo:    completion.ItemNo,
			StepID:    completion.StepID,
		})
		if err != nil {
			return SyncChangeSet{}, err
		}
		changeSet.UpsertCompletions = append(changeSet.UpsertCompletions, row)
	}
	for _, completion := range req.DeleteCompletions {
		row, err := service.validateDeleteCompletionRequest(completion.EquipmentID, DeleteCompletionRequest{
			SectionID: completion.SectionID,
			ItemIndex: completion.ItemIndex,
			StepID:    completion.StepID,
		})
		if err != nil {
			return SyncChangeSet{}, err
		}
		changeSet.DeleteCompletions = append(changeSet.DeleteCompletions, CompletionKey{
			EquipmentID: row.EquipmentID.String(),
			SectionID:   row.SectionID,
			ItemIndex:   row.ItemIndex,
			StepID:      row.StepID,
		})
	}
	for _, fault := range req.UpsertFaults {
		row, err := service.validateFaultRequest(fault.EquipmentID, FaultRequest{
			SectionID:        fault.SectionID,
			ItemIndex:        fault.ItemIndex,
			ItemNo:           fault.ItemNo,
			Status:           fault.Status,
			FaultText:        fault.FaultText,
			CorrectiveAction: fault.CorrectiveAction,
		})
		if err != nil {
			return SyncChangeSet{}, err
		}
		changeSet.UpsertFaults = append(changeSet.UpsertFaults, row)
	}
	for _, fault := range req.DeleteFaults {
		key, err := service.validateDeleteFaultRequest(fault.EquipmentID, DeleteFaultRequest{
			SectionID: fault.SectionID,
			ItemIndex: fault.ItemIndex,
		})
		if err != nil {
			return SyncChangeSet{}, err
		}
		changeSet.DeleteFaults = append(changeSet.DeleteFaults, key)
	}
	return changeSet, nil
}

func isValidEquipmentManual(blobPath string) bool {
	cleaned := path.Clean(blobPath)
	return cleaned == blobPath &&
		strings.HasPrefix(cleaned, "pmcs_sbs/") &&
		strings.HasSuffix(strings.ToLower(cleaned), ".json")
}

func isValidFaultStatus(status string) bool {
	return status == "X" || status == "/" || status == "-"
}

func completionKey(equipmentID string, sectionID string, itemIndex int32, stepID string) string {
	return fmt.Sprintf("%s|%s|%d|%s", strings.TrimSpace(equipmentID), strings.TrimSpace(sectionID), itemIndex, strings.TrimSpace(stepID))
}

func faultKey(equipmentID string, sectionID string, itemIndex int32) string {
	return fmt.Sprintf("%s|%s|%d", strings.TrimSpace(equipmentID), strings.TrimSpace(sectionID), itemIndex)
}

func mapEquipment(row model.PmcsSbsEquipment) EquipmentResponse {
	return EquipmentResponse{
		ID:              row.ID.String(),
		EquipmentManual: row.EquipmentManual,
		Admin:           row.Admin,
		Serial:          row.Serial,
		Uic:             row.Uic,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func mapCompletion(row model.PmcsSbsCompletions) CompletionResponse {
	return CompletionResponse{
		EquipmentID: row.EquipmentID.String(),
		SectionID:   row.SectionID,
		ItemIndex:   row.ItemIndex,
		ItemNo:      row.ItemNo,
		StepID:      row.StepID,
		IsComplete:  row.IsComplete,
		UpdatedAt:   row.UpdatedAt,
	}
}

func mapFault(row model.PmcsSbsFaults) FaultResponse {
	return FaultResponse{
		EquipmentID:      row.EquipmentID.String(),
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

func mapAggregate(row EquipmentAggregate) *EquipmentAggregateResponse {
	resp := &EquipmentAggregateResponse{Equipment: mapEquipment(row.Equipment)}
	for _, completion := range row.Completions {
		resp.Completions = append(resp.Completions, mapCompletion(completion))
	}
	for _, fault := range row.Faults {
		resp.Faults = append(resp.Faults, mapFault(fault))
	}
	return resp
}
```

- [ ] **Step 4: Run validation tests**

Run:

```bash
go test ./api/pmcs_sbs_progress -run 'TestValidate' -count=1
```

Expected: pass.

- [ ] **Step 5: Commit foundation**

```bash
git add api/pmcs_sbs_progress/errors.go api/pmcs_sbs_progress/types.go api/pmcs_sbs_progress/service.go api/pmcs_sbs_progress/repository.go api/pmcs_sbs_progress/service_impl.go api/pmcs_sbs_progress/service_impl_test.go
git commit -m "feat(pmcs-sbs): add progress sync foundation"
```

---

### Task 2: Authenticated Handlers

**Files:**
- Create: `api/pmcs_sbs_progress/route.go`
- Create: `api/pmcs_sbs_progress/route_test.go`

- [ ] **Step 1: Write handler tests with a service stub**

Create `api/pmcs_sbs_progress/route_test.go`.

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
	listResp       *EquipmentListResponse
	aggregateResp  *EquipmentAggregateResponse
	equipmentResp  *EquipmentResponse
	completionResp *CompletionResponse
	faultResp      *FaultResponse
	syncResp       *SyncResponse
	err            error
	capturedUser   *bootstrap.User
}

func (s *serviceStub) ListEquipment(user *bootstrap.User) (*EquipmentListResponse, error) {
	s.capturedUser = user
	return s.listResp, s.err
}
func (s *serviceStub) GetEquipment(user *bootstrap.User, _ string) (*EquipmentAggregateResponse, error) {
	s.capturedUser = user
	return s.aggregateResp, s.err
}
func (s *serviceStub) UpsertEquipment(user *bootstrap.User, _ string, _ EquipmentRequest) (*EquipmentResponse, error) {
	s.capturedUser = user
	return s.equipmentResp, s.err
}
func (s *serviceStub) DeleteEquipment(user *bootstrap.User, _ string) error {
	s.capturedUser = user
	return s.err
}
func (s *serviceStub) UpsertCompletion(user *bootstrap.User, _ string, _ CompletionRequest) (*CompletionResponse, error) {
	s.capturedUser = user
	return s.completionResp, s.err
}
func (s *serviceStub) DeleteCompletion(user *bootstrap.User, _ string, _ DeleteCompletionRequest) error {
	s.capturedUser = user
	return s.err
}
func (s *serviceStub) UpsertFault(user *bootstrap.User, _ string, _ FaultRequest) (*FaultResponse, error) {
	s.capturedUser = user
	return s.faultResp, s.err
}
func (s *serviceStub) DeleteFault(user *bootstrap.User, _ string, _ DeleteFaultRequest) error {
	s.capturedUser = user
	return s.err
}
func (s *serviceStub) Sync(user *bootstrap.User, _ SyncRequest) (*SyncResponse, error) {
	s.capturedUser = user
	return s.syncResp, s.err
}

func TestHandlersRequireAuth(t *testing.T) {
	router := newRouteTestRouter(&serviceStub{})
	resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment", nil, nil)
	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestListEquipmentSuccess(t *testing.T) {
	stub := &serviceStub{listResp: &EquipmentListResponse{Equipment: []EquipmentResponse{}, Count: 0}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment", nil, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "user-1", stub.capturedUser.UserID)
}

func TestGetEquipmentMapsNotFound(t *testing.T) {
	router := newRouteTestRouter(&serviceStub{err: ErrNotFound})

	resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/550e8400-e29b-41d4-a716-446655440000", nil, routeUser())

	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestUpsertEquipmentSuccess(t *testing.T) {
	now := time.Now().UTC()
	stub := &serviceStub{equipmentResp: &EquipmentResponse{
		ID: "550e8400-e29b-41d4-a716-446655440000", EquipmentManual: "pmcs_sbs/hmmwv/basic.json", Admin: "A12", CreatedAt: now, UpdatedAt: now,
	}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/550e8400-e29b-41d4-a716-446655440000", EquipmentRequest{
		EquipmentManual: "pmcs_sbs/hmmwv/basic.json",
		Admin:           "A12",
	}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestInvalidJSONReturnsBadRequest(t *testing.T) {
	router := newRouteTestRouter(&serviceStub{})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/550e8400-e29b-41d4-a716-446655440000", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-1")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestUpsertCompletionSuccess(t *testing.T) {
	stub := &serviceStub{completionResp: &CompletionResponse{EquipmentID: "550e8400-e29b-41d4-a716-446655440000", SectionID: "before", ItemIndex: 0, ItemNo: "1", StepID: "1-a", IsComplete: true}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/550e8400-e29b-41d4-a716-446655440000/completions", CompletionRequest{
		SectionID: "before",
		ItemIndex: 0,
		ItemNo:    "1",
		StepID:    "1-a",
	}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestUpsertFaultMapsInvalidStatus(t *testing.T) {
	router := newRouteTestRouter(&serviceStub{err: ErrInvalidStatus})

	resp := doRouteJSON(router, http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/550e8400-e29b-41d4-a716-446655440000/faults", FaultRequest{
		SectionID: "before",
		ItemIndex: 0,
		ItemNo:    "1",
		Status:    "BAD",
		FaultText: "leak",
	}, routeUser())

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestSyncSuccess(t *testing.T) {
	stub := &serviceStub{syncResp: &SyncResponse{DeletedEquipmentIDs: []string{}}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodPost, "/api/v1/auth/pmcs-sbs/sync", SyncRequest{}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestUnexpectedServiceErrorReturnsGeneric500(t *testing.T) {
	router := newRouteTestRouter(&serviceStub{err: errors.New("db exploded")})

	resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment", nil, routeUser())

	require.Equal(t, http.StatusInternalServerError, resp.Code)
	require.NotContains(t, resp.Body.String(), "db exploded")
}

func newRouteTestRouter(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/auth")
	group.Use(func(c *gin.Context) {
		if c.GetHeader("X-User-ID") != "" {
			c.Set("user", routeUser())
		}
		c.Next()
	})
	registerHandlers(group, svc)
	return router
}

func routeUser() *bootstrap.User {
	return &bootstrap.User{UserID: "user-1", Username: "tester", Email: "user-1@example.com"}
}

func doRouteJSON(router *gin.Engine, method string, path string, body interface{}, user *bootstrap.User) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if user != nil {
		req.Header.Set("X-User-ID", user.UserID)
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}
```

- [ ] **Step 2: Run handler tests and verify failure**

Run:

```bash
go test ./api/pmcs_sbs_progress -run 'Test.*Handler|TestListEquipment|TestGetEquipment|TestUpsert|TestSync|TestUnexpected' -count=1
```

Expected: fail because `route.go` does not exist.

- [ ] **Step 3: Implement route handlers**

Create `api/pmcs_sbs_progress/route.go`.

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
	group.GET("/pmcs-sbs/equipment", handler.listEquipment)
	group.GET("/pmcs-sbs/equipment/:equipment_id", handler.getEquipment)
	group.PUT("/pmcs-sbs/equipment/:equipment_id", handler.upsertEquipment)
	group.DELETE("/pmcs-sbs/equipment/:equipment_id", handler.deleteEquipment)
	group.PUT("/pmcs-sbs/equipment/:equipment_id/completions", handler.upsertCompletion)
	group.DELETE("/pmcs-sbs/equipment/:equipment_id/completions", handler.deleteCompletion)
	group.PUT("/pmcs-sbs/equipment/:equipment_id/faults", handler.upsertFault)
	group.DELETE("/pmcs-sbs/equipment/:equipment_id/faults", handler.deleteFault)
	group.POST("/pmcs-sbs/sync", handler.sync)
}

func (handler Handler) listEquipment(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}
	result, err := handler.service.ListEquipment(user)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Message: "", Data: result})
}

func (handler Handler) getEquipment(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}
	result, err := handler.service.GetEquipment(user, c.Param("equipment_id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Message: "", Data: result})
}

func (handler Handler) upsertEquipment(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}
	var req EquipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}
	result, err := handler.service.UpsertEquipment(user, c.Param("equipment_id"), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Message: "Equipment saved", Data: result})
}

func (handler Handler) deleteEquipment(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}
	if err := handler.service.DeleteEquipment(user, c.Param("equipment_id")); err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Equipment deleted"})
}

func (handler Handler) upsertCompletion(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}
	var req CompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}
	result, err := handler.service.UpsertCompletion(user, c.Param("equipment_id"), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Message: "Completion saved", Data: result})
}

func (handler Handler) deleteCompletion(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}
	var req DeleteCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}
	if err := handler.service.DeleteCompletion(user, c.Param("equipment_id"), req); err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Completion deleted"})
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

func (handler Handler) sync(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}
	var req SyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}
	result, err := handler.service.Sync(user, req)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Message: "Sync complete", Data: result})
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
	case errors.Is(err, ErrInvalidID), errors.Is(err, ErrInvalidRequest), errors.Is(err, ErrInvalidBlobPath), errors.Is(err, ErrInvalidStatus), errors.Is(err, ErrInvalidSyncRequest):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "pmcs sbs equipment not found"})
	default:
		slog.Error("PMCS SBS progress handler failed", "error", err)
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
	}
}
```

- [ ] **Step 4: Run handler tests**

Run:

```bash
go test ./api/pmcs_sbs_progress -run 'Test.*Handler|TestListEquipment|TestGetEquipment|TestUpsert|TestSync|TestUnexpected' -count=1
```

Expected: pass.

- [ ] **Step 5: Commit handlers**

```bash
git add api/pmcs_sbs_progress/route.go api/pmcs_sbs_progress/route_test.go
git commit -m "feat(pmcs-sbs): add authenticated progress handlers"
```

---

### Task 3: Repository Equipment Aggregate

**Files:**
- Create: `api/pmcs_sbs_progress/repository_impl.go`
- Create: `tests/pmcs_sbs_progress/main_test.go`
- Create: `tests/pmcs_sbs_progress/helpers_test.go`
- Create: `tests/pmcs_sbs_progress/repository_test.go`

- [ ] **Step 1: Write repository integration test setup**

Create `tests/pmcs_sbs_progress/main_test.go`.

```go
package pmcs_sbs_progress_test

import (
	"database/sql"
	"log"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	var err error
	testDB, err = sql.Open("postgres", "postgres://postgres:potato123@192.168.20.70/miltech_ng_test?sslmode=disable")
	if err != nil {
		log.Fatalf("failed to open test database: %v", err)
	}
	if err := testDB.Ping(); err != nil {
		log.Fatalf("failed to ping test database: %v", err)
	}
	exitCode := m.Run()
	if err := testDB.Close(); err != nil {
		log.Printf("failed to close test database: %v", err)
	}
	os.Exit(exitCode)
}
```

Create `tests/pmcs_sbs_progress/helpers_test.go`.

```go
package pmcs_sbs_progress_test

import (
	"database/sql"
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

func clearPmcsSbsTables(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`TRUNCATE TABLE pmcs_sbs_faults, pmcs_sbs_completions, pmcs_sbs_equipment RESTART IDENTITY CASCADE`)
	require.NoError(t, err)
}

func sampleEquipment(user *bootstrap.User) model.PmcsSbsEquipment {
	now := time.Now().UTC()
	return model.PmcsSbsEquipment{
		ID:              uuid.New(),
		UserUID:         user.UserID,
		EquipmentManual: "pmcs_sbs/hmmwv/basic.json",
		Admin:           "A12",
		Serial:          "SER-1",
		Uic:             "UIC",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func sampleCompletion(equipmentID uuid.UUID) model.PmcsSbsCompletions {
	return model.PmcsSbsCompletions{
		EquipmentID: equipmentID,
		SectionID:   "before",
		ItemIndex:   0,
		ItemNo:      "1",
		StepID:      "1-a",
		IsComplete:  true,
		UpdatedAt:   time.Now().UTC(),
	}
}

func sampleFault(equipmentID uuid.UUID) model.PmcsSbsFaults {
	now := time.Now().UTC()
	return model.PmcsSbsFaults{
		EquipmentID:      equipmentID,
		SectionID:        "before",
		ItemIndex:        0,
		ItemNo:           "1",
		Status:           "X",
		FaultText:        "leak",
		CorrectiveAction: "",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}
```

- [ ] **Step 2: Write equipment repository tests**

Create `tests/pmcs_sbs_progress/repository_test.go`.

```go
package pmcs_sbs_progress_test

import (
	"testing"

	"miltechserver/api/pmcs_sbs_progress"

	"github.com/stretchr/testify/require"
)

func TestRepositoryEquipmentLifecycle(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-equipment-user")
	repo := pmcs_sbs_progress.NewRepository(testDB)
	equipment := sampleEquipment(user)

	saved, err := repo.UpsertEquipment(user, equipment)
	require.NoError(t, err)
	require.Equal(t, equipment.ID, saved.ID)
	require.Equal(t, user.UserID, saved.UserUID)

	list, err := repo.ListEquipment(user)
	require.NoError(t, err)
	require.Len(t, list, 1)

	aggregate, err := repo.GetEquipmentAggregate(user, equipment.ID.String())
	require.NoError(t, err)
	require.Equal(t, equipment.ID, aggregate.Equipment.ID)
	require.Empty(t, aggregate.Completions)
	require.Empty(t, aggregate.Faults)

	require.NoError(t, repo.DeleteEquipment(user, equipment.ID.String()))
	list, err = repo.ListEquipment(user)
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestRepositoryGetEquipmentAggregateIncludesChildren(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-equipment-aggregate")
	repo := pmcs_sbs_progress.NewRepository(testDB)
	equipment := sampleEquipment(user)

	_, err := repo.UpsertEquipment(user, equipment)
	require.NoError(t, err)
	_, err = repo.UpsertCompletion(user, sampleCompletion(equipment.ID))
	require.NoError(t, err)
	_, err = repo.UpsertFault(user, sampleFault(equipment.ID))
	require.NoError(t, err)

	aggregate, err := repo.GetEquipmentAggregate(user, equipment.ID.String())
	require.NoError(t, err)
	require.Len(t, aggregate.Completions, 1)
	require.Len(t, aggregate.Faults, 1)
}

func TestRepositoryEquipmentUserIsolation(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	owner := testUser("pmcs-owner")
	other := testUser("pmcs-other")
	repo := pmcs_sbs_progress.NewRepository(testDB)
	equipment := sampleEquipment(owner)

	_, err := repo.UpsertEquipment(owner, equipment)
	require.NoError(t, err)

	_, err = repo.GetEquipmentAggregate(other, equipment.ID.String())
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)

	err = repo.DeleteEquipment(other, equipment.ID.String())
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)
}
```

- [ ] **Step 3: Run repository tests and verify failure**

Run:

```bash
go test ./tests/pmcs_sbs_progress -run 'TestRepositoryEquipment' -count=1
```

Expected: fail because `NewRepository` and repository methods do not exist.

- [ ] **Step 4: Implement repository equipment and aggregate methods**

Create `api/pmcs_sbs_progress/repository_impl.go`.

```go
package pmcs_sbs_progress

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/bootstrap"

	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) ListEquipment(user *bootstrap.User) ([]model.PmcsSbsEquipment, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
	var rows []model.PmcsSbsEquipment
	stmt := SELECT(PmcsSbsEquipment.AllColumns).
		FROM(PmcsSbsEquipment).
		WHERE(PmcsSbsEquipment.UserUID.EQ(String(user.UserID))).
		ORDER_BY(PmcsSbsEquipment.UpdatedAt.DESC())
	if err := stmt.Query(repo.db, &rows); err != nil {
		return nil, fmt.Errorf("list pmcs sbs equipment: %w", err)
	}
	return rows, nil
}

func (repo *RepositoryImpl) GetEquipmentAggregate(user *bootstrap.User, equipmentID string) (*EquipmentAggregate, error) {
	equipment, err := repo.getEquipmentByID(repo.db, user, equipmentID)
	if err != nil {
		return nil, err
	}
	completions, err := repo.getCompletions(repo.db, equipmentID)
	if err != nil {
		return nil, err
	}
	faults, err := repo.getFaults(repo.db, equipmentID)
	if err != nil {
		return nil, err
	}
	return &EquipmentAggregate{Equipment: *equipment, Completions: completions, Faults: faults}, nil
}

func (repo *RepositoryImpl) UpsertEquipment(user *bootstrap.User, equipment model.PmcsSbsEquipment) (*model.PmcsSbsEquipment, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
	now := time.Now().UTC()
	equipment.UserUID = user.UserID
	equipment.UpdatedAt = now
	if equipment.CreatedAt.IsZero() {
		equipment.CreatedAt = now
	}
	stmt := PmcsSbsEquipment.INSERT(
		PmcsSbsEquipment.ID,
		PmcsSbsEquipment.UserUID,
		PmcsSbsEquipment.EquipmentManual,
		PmcsSbsEquipment.Admin,
		PmcsSbsEquipment.Serial,
		PmcsSbsEquipment.Uic,
		PmcsSbsEquipment.CreatedAt,
		PmcsSbsEquipment.UpdatedAt,
	).MODEL(equipment).
		ON_CONFLICT(PmcsSbsEquipment.ID).
		DO_UPDATE(SET(
			PmcsSbsEquipment.EquipmentManual.SET(String(equipment.EquipmentManual)),
			PmcsSbsEquipment.Admin.SET(String(equipment.Admin)),
			PmcsSbsEquipment.Serial.SET(String(equipment.Serial)),
			PmcsSbsEquipment.Uic.SET(String(equipment.Uic)),
			PmcsSbsEquipment.UpdatedAt.SET(TimestampzT(now)),
		).WHERE(PmcsSbsEquipment.UserUID.EQ(String(user.UserID)))).
		RETURNING(PmcsSbsEquipment.AllColumns)
	var saved model.PmcsSbsEquipment
	if err := stmt.Query(repo.db, &saved); err != nil {
		return nil, fmt.Errorf("upsert pmcs sbs equipment: %w", err)
	}
	return &saved, nil
}

func (repo *RepositoryImpl) DeleteEquipment(user *bootstrap.User, equipmentID string) error {
	if user == nil {
		return ErrUnauthorized
	}
	tx, err := repo.db.Begin()
	if err != nil {
		return fmt.Errorf("begin delete pmcs sbs equipment: %w", err)
	}
	defer tx.Rollback()
	if _, err := repo.getEquipmentByID(tx, user, equipmentID); err != nil {
		return err
	}
	if _, err := PmcsSbsFaults.DELETE().WHERE(PmcsSbsFaults.EquipmentID.EQ(String(equipmentID))).Exec(tx); err != nil {
		return fmt.Errorf("delete pmcs sbs faults: %w", err)
	}
	if _, err := PmcsSbsCompletions.DELETE().WHERE(PmcsSbsCompletions.EquipmentID.EQ(String(equipmentID))).Exec(tx); err != nil {
		return fmt.Errorf("delete pmcs sbs completions: %w", err)
	}
	result, err := PmcsSbsEquipment.DELETE().
		WHERE(PmcsSbsEquipment.ID.EQ(String(equipmentID)).AND(PmcsSbsEquipment.UserUID.EQ(String(user.UserID)))).
		Exec(tx)
	if err != nil {
		return fmt.Errorf("delete pmcs sbs equipment: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete pmcs sbs equipment: %w", err)
	}
	slog.Info("pmcs sbs equipment deleted", "user_id", user.UserID, "equipment_id", equipmentID)
	return nil
}

type queryer interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

func (repo *RepositoryImpl) getEquipmentByID(db interface{}, user *bootstrap.User, equipmentID string) (*model.PmcsSbsEquipment, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
	var row model.PmcsSbsEquipment
	stmt := SELECT(PmcsSbsEquipment.AllColumns).
		FROM(PmcsSbsEquipment).
		WHERE(PmcsSbsEquipment.ID.EQ(String(equipmentID)).AND(PmcsSbsEquipment.UserUID.EQ(String(user.UserID))))
	err := stmt.Query(db, &row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get pmcs sbs equipment: %w", err)
	}
	if row.ID.String() == "00000000-0000-0000-0000-000000000000" {
		return nil, ErrNotFound
	}
	return &row, nil
}

func (repo *RepositoryImpl) getCompletions(db interface{}, equipmentID string) ([]model.PmcsSbsCompletions, error) {
	var rows []model.PmcsSbsCompletions
	stmt := SELECT(PmcsSbsCompletions.AllColumns).
		FROM(PmcsSbsCompletions).
		WHERE(PmcsSbsCompletions.EquipmentID.EQ(String(equipmentID))).
		ORDER_BY(PmcsSbsCompletions.SectionID.ASC(), PmcsSbsCompletions.ItemIndex.ASC(), PmcsSbsCompletions.StepID.ASC())
	if err := stmt.Query(db, &rows); err != nil {
		return nil, fmt.Errorf("get pmcs sbs completions: %w", err)
	}
	return rows, nil
}

func (repo *RepositoryImpl) getFaults(db interface{}, equipmentID string) ([]model.PmcsSbsFaults, error) {
	var rows []model.PmcsSbsFaults
	stmt := SELECT(PmcsSbsFaults.AllColumns).
		FROM(PmcsSbsFaults).
		WHERE(PmcsSbsFaults.EquipmentID.EQ(String(equipmentID))).
		ORDER_BY(PmcsSbsFaults.SectionID.ASC(), PmcsSbsFaults.ItemIndex.ASC())
	if err := stmt.Query(db, &rows); err != nil {
		return nil, fmt.Errorf("get pmcs sbs faults: %w", err)
	}
	return rows, nil
}
```

- [ ] **Step 5: Add compile-only repository methods for later tasks**

Append these methods to `repository_impl.go` so Task 3 compiles. Later tasks replace them.

```go
func (repo *RepositoryImpl) UpsertCompletion(user *bootstrap.User, completion model.PmcsSbsCompletions) (*model.PmcsSbsCompletions, error) {
	return nil, errors.New("completion persistence starts in task 4")
}

func (repo *RepositoryImpl) DeleteCompletion(user *bootstrap.User, equipmentID string, sectionID string, itemIndex int32, stepID string) error {
	return errors.New("completion delete persistence starts in task 4")
}

func (repo *RepositoryImpl) UpsertFault(user *bootstrap.User, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error) {
	return nil, errors.New("fault persistence starts in task 4")
}

func (repo *RepositoryImpl) DeleteFault(user *bootstrap.User, equipmentID string, sectionID string, itemIndex int32) error {
	return errors.New("fault delete persistence starts in task 4")
}

func (repo *RepositoryImpl) Sync(user *bootstrap.User, changeSet SyncChangeSet) (*SyncResult, error) {
	return nil, errors.New("batch sync persistence starts in task 5")
}
```

- [ ] **Step 6: Run equipment repository tests**

Run:

```bash
go test ./tests/pmcs_sbs_progress -run 'TestRepositoryEquipment' -count=1
```

Expected: `TestRepositoryEquipmentLifecycle` and `TestRepositoryEquipmentUserIsolation` pass. `TestRepositoryGetEquipmentAggregateIncludesChildren` is not selected by this command and runs in Task 4 after child persistence exists.

- [ ] **Step 7: Commit repository equipment work**

```bash
git add api/pmcs_sbs_progress/repository_impl.go tests/pmcs_sbs_progress/main_test.go tests/pmcs_sbs_progress/helpers_test.go tests/pmcs_sbs_progress/repository_test.go
git commit -m "feat(pmcs-sbs): persist progress equipment"
```

---

### Task 4: Completion And Fault Persistence

**Files:**
- Modify: `api/pmcs_sbs_progress/repository_impl.go`
- Modify: `tests/pmcs_sbs_progress/repository_test.go`

- [ ] **Step 1: Add completion and fault repository tests**

Append to `tests/pmcs_sbs_progress/repository_test.go`.

```go
func TestRepositoryCompletionLifecycle(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-completion-user")
	repo := pmcs_sbs_progress.NewRepository(testDB)
	equipment := sampleEquipment(user)
	_, err := repo.UpsertEquipment(user, equipment)
	require.NoError(t, err)

	completion := sampleCompletion(equipment.ID)
	saved, err := repo.UpsertCompletion(user, completion)
	require.NoError(t, err)
	require.True(t, saved.IsComplete)

	aggregate, err := repo.GetEquipmentAggregate(user, equipment.ID.String())
	require.NoError(t, err)
	require.Len(t, aggregate.Completions, 1)

	require.NoError(t, repo.DeleteCompletion(user, equipment.ID.String(), completion.SectionID, completion.ItemIndex, completion.StepID))
	aggregate, err = repo.GetEquipmentAggregate(user, equipment.ID.String())
	require.NoError(t, err)
	require.Empty(t, aggregate.Completions)
}

func TestRepositoryFaultLifecycle(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-user")
	repo := pmcs_sbs_progress.NewRepository(testDB)
	equipment := sampleEquipment(user)
	_, err := repo.UpsertEquipment(user, equipment)
	require.NoError(t, err)

	fault := sampleFault(equipment.ID)
	saved, err := repo.UpsertFault(user, fault)
	require.NoError(t, err)
	require.Equal(t, "leak", saved.FaultText)

	fault.FaultText = "updated leak"
	updated, err := repo.UpsertFault(user, fault)
	require.NoError(t, err)
	require.Equal(t, "updated leak", updated.FaultText)
	require.Equal(t, saved.CreatedAt, updated.CreatedAt)

	aggregate, err := repo.GetEquipmentAggregate(user, equipment.ID.String())
	require.NoError(t, err)
	require.Len(t, aggregate.Faults, 1)

	require.NoError(t, repo.DeleteFault(user, equipment.ID.String(), fault.SectionID, fault.ItemIndex))
	aggregate, err = repo.GetEquipmentAggregate(user, equipment.ID.String())
	require.NoError(t, err)
	require.Empty(t, aggregate.Faults)
}

func TestRepositoryChildWritesRequireOwnedEquipment(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	owner := testUser("pmcs-child-owner")
	other := testUser("pmcs-child-other")
	repo := pmcs_sbs_progress.NewRepository(testDB)
	equipment := sampleEquipment(owner)
	_, err := repo.UpsertEquipment(owner, equipment)
	require.NoError(t, err)

	_, err = repo.UpsertCompletion(other, sampleCompletion(equipment.ID))
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)

	_, err = repo.UpsertFault(other, sampleFault(equipment.ID))
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)
}
```

- [ ] **Step 2: Run tests and verify failure**

Run:

```bash
go test ./tests/pmcs_sbs_progress -run 'TestRepositoryCompletion|TestRepositoryFault|TestRepositoryChild|TestRepositoryGetEquipmentAggregateIncludesChildren' -count=1
```

Expected: fail because Task 3 compile-only methods return task-boundary errors.

- [ ] **Step 3: Replace completion and fault repository stubs**

Replace the completion/fault stub methods in `api/pmcs_sbs_progress/repository_impl.go` with:

```go
func (repo *RepositoryImpl) UpsertCompletion(user *bootstrap.User, completion model.PmcsSbsCompletions) (*model.PmcsSbsCompletions, error) {
	if _, err := repo.getEquipmentByID(repo.db, user, completion.EquipmentID.String()); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	completion.IsComplete = true
	completion.UpdatedAt = now
	stmt := PmcsSbsCompletions.INSERT(
		PmcsSbsCompletions.EquipmentID,
		PmcsSbsCompletions.SectionID,
		PmcsSbsCompletions.ItemIndex,
		PmcsSbsCompletions.ItemNo,
		PmcsSbsCompletions.StepID,
		PmcsSbsCompletions.IsComplete,
		PmcsSbsCompletions.UpdatedAt,
	).MODEL(completion).
		ON_CONFLICT(PmcsSbsCompletions.EquipmentID, PmcsSbsCompletions.SectionID, PmcsSbsCompletions.ItemIndex, PmcsSbsCompletions.StepID).
		DO_UPDATE(SET(
			PmcsSbsCompletions.ItemNo.SET(String(completion.ItemNo)),
			PmcsSbsCompletions.IsComplete.SET(Bool(true)),
			PmcsSbsCompletions.UpdatedAt.SET(TimestampzT(now)),
		)).
		RETURNING(PmcsSbsCompletions.AllColumns)
	var saved model.PmcsSbsCompletions
	if err := stmt.Query(repo.db, &saved); err != nil {
		return nil, fmt.Errorf("upsert pmcs sbs completion: %w", err)
	}
	return &saved, nil
}

func (repo *RepositoryImpl) DeleteCompletion(user *bootstrap.User, equipmentID string, sectionID string, itemIndex int32, stepID string) error {
	if _, err := repo.getEquipmentByID(repo.db, user, equipmentID); err != nil {
		return err
	}
	_, err := PmcsSbsCompletions.DELETE().
		WHERE(PmcsSbsCompletions.EquipmentID.EQ(String(equipmentID)).
			AND(PmcsSbsCompletions.SectionID.EQ(String(sectionID))).
			AND(PmcsSbsCompletions.ItemIndex.EQ(Int32(itemIndex))).
			AND(PmcsSbsCompletions.StepID.EQ(String(stepID)))).
		Exec(repo.db)
	if err != nil {
		return fmt.Errorf("delete pmcs sbs completion: %w", err)
	}
	return nil
}

func (repo *RepositoryImpl) UpsertFault(user *bootstrap.User, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error) {
	if _, err := repo.getEquipmentByID(repo.db, user, fault.EquipmentID.String()); err != nil {
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
	).MODEL(fault).
		ON_CONFLICT(PmcsSbsFaults.EquipmentID, PmcsSbsFaults.SectionID, PmcsSbsFaults.ItemIndex).
		DO_UPDATE(SET(
			PmcsSbsFaults.ItemNo.SET(String(fault.ItemNo)),
			PmcsSbsFaults.Status.SET(String(fault.Status)),
			PmcsSbsFaults.FaultText.SET(String(fault.FaultText)),
			PmcsSbsFaults.CorrectiveAction.SET(String(fault.CorrectiveAction)),
			PmcsSbsFaults.UpdatedAt.SET(TimestampzT(now)),
		)).
		RETURNING(PmcsSbsFaults.AllColumns)
	var saved model.PmcsSbsFaults
	if err := stmt.Query(repo.db, &saved); err != nil {
		return nil, fmt.Errorf("upsert pmcs sbs fault: %w", err)
	}
	return &saved, nil
}

func (repo *RepositoryImpl) DeleteFault(user *bootstrap.User, equipmentID string, sectionID string, itemIndex int32) error {
	if _, err := repo.getEquipmentByID(repo.db, user, equipmentID); err != nil {
		return err
	}
	_, err := PmcsSbsFaults.DELETE().
		WHERE(PmcsSbsFaults.EquipmentID.EQ(String(equipmentID)).
			AND(PmcsSbsFaults.SectionID.EQ(String(sectionID))).
			AND(PmcsSbsFaults.ItemIndex.EQ(Int32(itemIndex)))).
		Exec(repo.db)
	if err != nil {
		return fmt.Errorf("delete pmcs sbs fault: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run completion and fault repository tests**

Run:

```bash
go test ./tests/pmcs_sbs_progress -run 'TestRepositoryCompletion|TestRepositoryFault|TestRepositoryChild|TestRepositoryGetEquipmentAggregateIncludesChildren' -count=1
```

Expected: pass.

- [ ] **Step 5: Commit child persistence**

```bash
git add api/pmcs_sbs_progress/repository_impl.go tests/pmcs_sbs_progress/repository_test.go
git commit -m "feat(pmcs-sbs): persist completions and faults"
```

---

### Task 5: Batch Sync Transaction

**Files:**
- Modify: `api/pmcs_sbs_progress/repository_impl.go`
- Modify: `api/pmcs_sbs_progress/service_impl_test.go`
- Modify: `tests/pmcs_sbs_progress/repository_test.go`

- [ ] **Step 1: Add service test for sync change set building**

Append to `api/pmcs_sbs_progress/service_impl_test.go`.

```go
func TestBuildSyncChangeSet(t *testing.T) {
	svc := NewService(&repoStub{})
	user := requireUser()

	changeSet, err := svc.buildSyncChangeSet(user, SyncRequest{
		UpsertEquipment: []SyncEquipmentRequest{{
			ID:              "550e8400-e29b-41d4-a716-446655440000",
			EquipmentManual: "pmcs_sbs/hmmwv/basic.json",
			Admin:           "A12",
		}},
		UpsertCompletions: []SyncCompletionRequest{{
			EquipmentID: "550e8400-e29b-41d4-a716-446655440000",
			SectionID:   "before",
			ItemIndex:   0,
			ItemNo:      "1",
			StepID:      "1-a",
		}},
		UpsertFaults: []SyncFaultRequest{{
			EquipmentID: "550e8400-e29b-41d4-a716-446655440000",
			SectionID:   "before",
			ItemIndex:   0,
			ItemNo:      "1",
			Status:      "X",
			FaultText:   "leak",
		}},
	})

	require.NoError(t, err)
	require.Len(t, changeSet.UpsertEquipment, 1)
	require.Len(t, changeSet.UpsertCompletions, 1)
	require.Len(t, changeSet.UpsertFaults, 1)
	require.Equal(t, user.UserID, changeSet.UpsertEquipment[0].UserUID)
}
```

- [ ] **Step 2: Add repository sync integration tests**

Append to `tests/pmcs_sbs_progress/repository_test.go`.

```go
func TestRepositorySyncAppliesChangeSet(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-sync-user")
	repo := pmcs_sbs_progress.NewRepository(testDB)
	equipment := sampleEquipment(user)
	completion := sampleCompletion(equipment.ID)
	fault := sampleFault(equipment.ID)

	result, err := repo.Sync(user, pmcs_sbs_progress.SyncChangeSet{
		UpsertEquipment:   []model.PmcsSbsEquipment{equipment},
		UpsertCompletions: []model.PmcsSbsCompletions{completion},
		UpsertFaults:      []model.PmcsSbsFaults{fault},
	})

	require.NoError(t, err)
	require.Len(t, result.Equipment, 1)
	require.Len(t, result.Equipment[0].Completions, 1)
	require.Len(t, result.Equipment[0].Faults, 1)
}

func TestRepositorySyncDeletesRows(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-sync-delete")
	repo := pmcs_sbs_progress.NewRepository(testDB)
	equipment := sampleEquipment(user)
	_, err := repo.UpsertEquipment(user, equipment)
	require.NoError(t, err)
	_, err = repo.UpsertCompletion(user, sampleCompletion(equipment.ID))
	require.NoError(t, err)
	_, err = repo.UpsertFault(user, sampleFault(equipment.ID))
	require.NoError(t, err)

	result, err := repo.Sync(user, pmcs_sbs_progress.SyncChangeSet{
		DeleteCompletions: []pmcs_sbs_progress.CompletionKey{{
			EquipmentID: equipment.ID.String(),
			SectionID:   "before",
			ItemIndex:   0,
			StepID:      "1-a",
		}},
		DeleteFaults: []pmcs_sbs_progress.FaultKey{{
			EquipmentID: equipment.ID.String(),
			SectionID:   "before",
			ItemIndex:   0,
		}},
	})

	require.NoError(t, err)
	require.Len(t, result.Equipment, 1)
	require.Empty(t, result.Equipment[0].Completions)
	require.Empty(t, result.Equipment[0].Faults)
}

func TestRepositorySyncDeletesEquipment(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-sync-delete-equipment")
	repo := pmcs_sbs_progress.NewRepository(testDB)
	equipment := sampleEquipment(user)
	_, err := repo.UpsertEquipment(user, equipment)
	require.NoError(t, err)
	_, err = repo.UpsertCompletion(user, sampleCompletion(equipment.ID))
	require.NoError(t, err)

	result, err := repo.Sync(user, pmcs_sbs_progress.SyncChangeSet{
		DeleteEquipmentIDs: []string{equipment.ID.String()},
	})

	require.NoError(t, err)
	require.Empty(t, result.Equipment)
	require.Equal(t, []string{equipment.ID.String()}, result.DeletedEquipmentIDs)

	list, err := repo.ListEquipment(user)
	require.NoError(t, err)
	require.Empty(t, list)
}
```

Add `model` import to `tests/pmcs_sbs_progress/repository_test.go`:

```go
import (
	"testing"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/pmcs_sbs_progress"

	"github.com/stretchr/testify/require"
)
```

- [ ] **Step 3: Run sync tests and verify failure**

Run:

```bash
go test ./api/pmcs_sbs_progress ./tests/pmcs_sbs_progress -run 'TestBuildSyncChangeSet|TestRepositorySync' -count=1
```

Expected: repository sync tests fail because repository `Sync` is still a stub.

- [ ] **Step 4: Implement repository Sync**

Replace the `Sync` stub in `api/pmcs_sbs_progress/repository_impl.go`.

```go
func (repo *RepositoryImpl) Sync(user *bootstrap.User, changeSet SyncChangeSet) (*SyncResult, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
	tx, err := repo.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin pmcs sbs sync: %w", err)
	}
	defer tx.Rollback()

	touched := map[string]struct{}{}
	deleted := map[string]struct{}{}

	for _, equipment := range changeSet.UpsertEquipment {
		equipment.UserUID = user.UserID
		saved, err := repo.upsertEquipmentWithExecutor(tx, user, equipment)
		if err != nil {
			return nil, err
		}
		touched[saved.ID.String()] = struct{}{}
	}

	for _, completion := range changeSet.UpsertCompletions {
		if _, err := repo.getEquipmentByID(tx, user, completion.EquipmentID.String()); err != nil {
			return nil, err
		}
		if _, err := repo.upsertCompletionWithExecutor(tx, completion); err != nil {
			return nil, err
		}
		touched[completion.EquipmentID.String()] = struct{}{}
	}

	for _, key := range changeSet.DeleteCompletions {
		if _, err := repo.getEquipmentByID(tx, user, key.EquipmentID); err != nil {
			return nil, err
		}
		if _, err := PmcsSbsCompletions.DELETE().
			WHERE(PmcsSbsCompletions.EquipmentID.EQ(String(key.EquipmentID)).
				AND(PmcsSbsCompletions.SectionID.EQ(String(key.SectionID))).
				AND(PmcsSbsCompletions.ItemIndex.EQ(Int32(key.ItemIndex))).
				AND(PmcsSbsCompletions.StepID.EQ(String(key.StepID)))).
			Exec(tx); err != nil {
			return nil, fmt.Errorf("sync delete completion: %w", err)
		}
		touched[key.EquipmentID] = struct{}{}
	}

	for _, fault := range changeSet.UpsertFaults {
		if _, err := repo.getEquipmentByID(tx, user, fault.EquipmentID.String()); err != nil {
			return nil, err
		}
		if _, err := repo.upsertFaultWithExecutor(tx, fault); err != nil {
			return nil, err
		}
		touched[fault.EquipmentID.String()] = struct{}{}
	}

	for _, key := range changeSet.DeleteFaults {
		if _, err := repo.getEquipmentByID(tx, user, key.EquipmentID); err != nil {
			return nil, err
		}
		if _, err := PmcsSbsFaults.DELETE().
			WHERE(PmcsSbsFaults.EquipmentID.EQ(String(key.EquipmentID)).
				AND(PmcsSbsFaults.SectionID.EQ(String(key.SectionID))).
				AND(PmcsSbsFaults.ItemIndex.EQ(Int32(key.ItemIndex)))).
			Exec(tx); err != nil {
			return nil, fmt.Errorf("sync delete fault: %w", err)
		}
		touched[key.EquipmentID] = struct{}{}
	}

	for _, equipmentID := range changeSet.DeleteEquipmentIDs {
		if _, err := repo.getEquipmentByID(tx, user, equipmentID); err != nil {
			return nil, err
		}
		if _, err := PmcsSbsFaults.DELETE().WHERE(PmcsSbsFaults.EquipmentID.EQ(String(equipmentID))).Exec(tx); err != nil {
			return nil, fmt.Errorf("sync delete equipment faults: %w", err)
		}
		if _, err := PmcsSbsCompletions.DELETE().WHERE(PmcsSbsCompletions.EquipmentID.EQ(String(equipmentID))).Exec(tx); err != nil {
			return nil, fmt.Errorf("sync delete equipment completions: %w", err)
		}
		if _, err := PmcsSbsEquipment.DELETE().
			WHERE(PmcsSbsEquipment.ID.EQ(String(equipmentID)).AND(PmcsSbsEquipment.UserUID.EQ(String(user.UserID)))).
			Exec(tx); err != nil {
			return nil, fmt.Errorf("sync delete equipment: %w", err)
		}
		delete(touched, equipmentID)
		deleted[equipmentID] = struct{}{}
	}

	result := &SyncResult{}
	for equipmentID := range touched {
		aggregate, err := repo.getEquipmentAggregateWithExecutor(tx, user, equipmentID)
		if err != nil {
			return nil, err
		}
		result.Equipment = append(result.Equipment, *aggregate)
	}
	for equipmentID := range deleted {
		result.DeletedEquipmentIDs = append(result.DeletedEquipmentIDs, equipmentID)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit pmcs sbs sync: %w", err)
	}
	return result, nil
}

func (repo *RepositoryImpl) getEquipmentAggregateWithExecutor(db interface{}, user *bootstrap.User, equipmentID string) (*EquipmentAggregate, error) {
	equipment, err := repo.getEquipmentByID(db, user, equipmentID)
	if err != nil {
		return nil, err
	}
	completions, err := repo.getCompletions(db, equipmentID)
	if err != nil {
		return nil, err
	}
	faults, err := repo.getFaults(db, equipmentID)
	if err != nil {
		return nil, err
	}
	return &EquipmentAggregate{Equipment: *equipment, Completions: completions, Faults: faults}, nil
}

func (repo *RepositoryImpl) upsertEquipmentWithExecutor(db interface{}, user *bootstrap.User, equipment model.PmcsSbsEquipment) (*model.PmcsSbsEquipment, error) {
	now := time.Now().UTC()
	equipment.UserUID = user.UserID
	equipment.UpdatedAt = now
	if equipment.CreatedAt.IsZero() {
		equipment.CreatedAt = now
	}
	stmt := PmcsSbsEquipment.INSERT(
		PmcsSbsEquipment.ID,
		PmcsSbsEquipment.UserUID,
		PmcsSbsEquipment.EquipmentManual,
		PmcsSbsEquipment.Admin,
		PmcsSbsEquipment.Serial,
		PmcsSbsEquipment.Uic,
		PmcsSbsEquipment.CreatedAt,
		PmcsSbsEquipment.UpdatedAt,
	).MODEL(equipment).
		ON_CONFLICT(PmcsSbsEquipment.ID).
		DO_UPDATE(SET(
			PmcsSbsEquipment.EquipmentManual.SET(String(equipment.EquipmentManual)),
			PmcsSbsEquipment.Admin.SET(String(equipment.Admin)),
			PmcsSbsEquipment.Serial.SET(String(equipment.Serial)),
			PmcsSbsEquipment.Uic.SET(String(equipment.Uic)),
			PmcsSbsEquipment.UpdatedAt.SET(TimestampzT(now)),
		).WHERE(PmcsSbsEquipment.UserUID.EQ(String(user.UserID)))).
		RETURNING(PmcsSbsEquipment.AllColumns)
	var saved model.PmcsSbsEquipment
	if err := stmt.Query(db, &saved); err != nil {
		return nil, fmt.Errorf("upsert pmcs sbs equipment: %w", err)
	}
	return &saved, nil
}

func (repo *RepositoryImpl) upsertCompletionWithExecutor(db interface{}, completion model.PmcsSbsCompletions) (*model.PmcsSbsCompletions, error) {
	now := time.Now().UTC()
	completion.IsComplete = true
	completion.UpdatedAt = now
	stmt := PmcsSbsCompletions.INSERT(
		PmcsSbsCompletions.EquipmentID,
		PmcsSbsCompletions.SectionID,
		PmcsSbsCompletions.ItemIndex,
		PmcsSbsCompletions.ItemNo,
		PmcsSbsCompletions.StepID,
		PmcsSbsCompletions.IsComplete,
		PmcsSbsCompletions.UpdatedAt,
	).MODEL(completion).
		ON_CONFLICT(PmcsSbsCompletions.EquipmentID, PmcsSbsCompletions.SectionID, PmcsSbsCompletions.ItemIndex, PmcsSbsCompletions.StepID).
		DO_UPDATE(SET(
			PmcsSbsCompletions.ItemNo.SET(String(completion.ItemNo)),
			PmcsSbsCompletions.IsComplete.SET(Bool(true)),
			PmcsSbsCompletions.UpdatedAt.SET(TimestampzT(now)),
		)).
		RETURNING(PmcsSbsCompletions.AllColumns)
	var saved model.PmcsSbsCompletions
	if err := stmt.Query(db, &saved); err != nil {
		return nil, fmt.Errorf("upsert pmcs sbs completion: %w", err)
	}
	return &saved, nil
}

func (repo *RepositoryImpl) upsertFaultWithExecutor(db interface{}, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error) {
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
	).MODEL(fault).
		ON_CONFLICT(PmcsSbsFaults.EquipmentID, PmcsSbsFaults.SectionID, PmcsSbsFaults.ItemIndex).
		DO_UPDATE(SET(
			PmcsSbsFaults.ItemNo.SET(String(fault.ItemNo)),
			PmcsSbsFaults.Status.SET(String(fault.Status)),
			PmcsSbsFaults.FaultText.SET(String(fault.FaultText)),
			PmcsSbsFaults.CorrectiveAction.SET(String(fault.CorrectiveAction)),
			PmcsSbsFaults.UpdatedAt.SET(TimestampzT(now)),
		)).
		RETURNING(PmcsSbsFaults.AllColumns)
	var saved model.PmcsSbsFaults
	if err := stmt.Query(db, &saved); err != nil {
		return nil, fmt.Errorf("upsert pmcs sbs fault: %w", err)
	}
	return &saved, nil
}
```

Update `UpsertEquipment`, `UpsertCompletion`, and `UpsertFault` to call the helper methods so duplicate Jet statements do not stay in the file.

- [ ] **Step 5: Run sync tests**

Run:

```bash
go test ./api/pmcs_sbs_progress ./tests/pmcs_sbs_progress -run 'TestBuildSyncChangeSet|TestRepositorySync' -count=1
```

Expected: pass.

- [ ] **Step 6: Commit sync**

```bash
git add api/pmcs_sbs_progress/repository_impl.go api/pmcs_sbs_progress/service_impl_test.go tests/pmcs_sbs_progress/repository_test.go
git commit -m "feat(pmcs-sbs): add batch progress sync"
```

---

### Task 6: Route Wiring And API Documentation

**Files:**
- Modify: `api/route/route.go`
- Create: `docs/api/pmcs-sbs-progress-sync.md`
- Modify: `api/pmcs_sbs_progress/route_test.go`

- [ ] **Step 1: Add route registration test**

Append to `api/pmcs_sbs_progress/route_test.go`.

```go
func TestRegisterRoutesUsesAuthGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/auth")
	group.Use(func(c *gin.Context) {
		c.Set("user", routeUser())
		c.Next()
	})
	registerHandlers(group, &serviceStub{listResp: &EquipmentListResponse{Equipment: []EquipmentResponse{}, Count: 0}})

	resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment", nil, routeUser())
	require.Equal(t, http.StatusOK, resp.Code)
}
```

- [ ] **Step 2: Wire package into main router**

Modify `api/route/route.go`.

Add import:

```go
"miltechserver/api/pmcs_sbs_progress"
```

Register inside the authenticated routes block after `equipment_services.RegisterRoutes(...)`:

```go
pmcs_sbs_progress.RegisterRoutes(pmcs_sbs_progress.Dependencies{DB: db}, authRoutes)
```

- [ ] **Step 3: Add API documentation**

Create `docs/api/pmcs-sbs-progress-sync.md`.

```markdown
# PMCS SBS Progress Sync API

Base URL: `https://<host>/api/v1/auth`
Authentication: Firebase bearer token required.
Content-Type: `application/json`

## Overview

This API syncs PMCS SBS equipment, completed steps, and faults for logged-in users. The public PMCS SBS library API still serves document JSON from Azure Blob Storage. This API stores user progress for a selected `equipment_manual` blob path.

## Rules

- Equipment IDs are client-generated UUIDs.
- `equipment_manual` stores the PMCS SBS JSON blob path, for example `pmcs_sbs/hmmwv/basic.json`.
- Completion rows represent completed steps only.
- Fault status must be `X`, `/`, or `-`.
- Deletes are hard deletes.
- Last write wins by server processing time.

## Endpoints

### List Equipment

`GET /pmcs-sbs/equipment`

Returns all PMCS SBS equipment for the authenticated user.

### Get Equipment Progress

`GET /pmcs-sbs/equipment/:equipment_id`

Returns one equipment row with its completions and faults.

### Save Equipment

`PUT /pmcs-sbs/equipment/:equipment_id`

```json
{
  "equipment_manual": "pmcs_sbs/hmmwv/basic.json",
  "admin": "A12",
  "serial": "SER123",
  "uic": "WABC01"
}
```

### Delete Equipment

`DELETE /pmcs-sbs/equipment/:equipment_id`

Deletes equipment and child completion/fault rows.

### Save Completion

`PUT /pmcs-sbs/equipment/:equipment_id/completions`

```json
{
  "section_id": "before",
  "item_index": 0,
  "item_no": "1",
  "step_id": "1-a"
}
```

### Delete Completion

`DELETE /pmcs-sbs/equipment/:equipment_id/completions`

```json
{
  "section_id": "before",
  "item_index": 0,
  "step_id": "1-a"
}
```

### Save Fault

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

### Delete Fault

`DELETE /pmcs-sbs/equipment/:equipment_id/faults`

```json
{
  "section_id": "before",
  "item_index": 0
}
```

### Batch Sync

`POST /pmcs-sbs/sync`

Sends explicit offline replay changes in one request.

```json
{
  "upsert_equipment": [],
  "delete_equipment_ids": [],
  "upsert_completions": [],
  "delete_completions": [],
  "upsert_faults": [],
  "delete_faults": []
}
```
```

- [ ] **Step 4: Run route and package tests**

Run:

```bash
go test ./api/pmcs_sbs_progress -count=1
```

Expected: pass.

- [ ] **Step 5: Commit route wiring and docs**

```bash
git add api/route/route.go api/pmcs_sbs_progress/route_test.go docs/api/pmcs-sbs-progress-sync.md
git commit -m "feat(pmcs-sbs): wire progress sync routes"
```

---

### Task 7: Full Verification And Cleanup

**Files:**
- Inspect: `api/pmcs_sbs_progress/*.go`
- Inspect: `tests/pmcs_sbs_progress/*.go`
- Inspect: `api/route/route.go`
- Inspect: `docs/api/pmcs-sbs-progress-sync.md`

- [ ] **Step 1: Run focused unit tests**

Run:

```bash
go test ./api/pmcs_sbs_progress -count=1
```

Expected: pass.

- [ ] **Step 2: Run PMCS SBS progress integration tests**

Run:

```bash
go test ./tests/pmcs_sbs_progress -count=1
```

Expected: pass.

- [ ] **Step 3: Run related library tests to guard existing PMCS SBS content API**

Run:

```bash
go test ./api/library/pmcs_sbs ./api/library -count=1
```

Expected: pass.

- [ ] **Step 4: Run full Go test suite**

Run:

```bash
go test ./...
```

Expected: pass. If unrelated packages fail because their external test database or services are unavailable, record the exact failing package and error text in the final handoff.

- [ ] **Step 5: Inspect git status**

Run:

```bash
git status --short
```

Expected: only intentional PMCS SBS progress files are changed.

- [ ] **Step 6: Commit final cleanup if changes were needed**

Only run this if Step 4 or Step 5 required fixes after the previous task commits.

```bash
git add api/pmcs_sbs_progress tests/pmcs_sbs_progress api/route/route.go docs/api/pmcs-sbs-progress-sync.md
git commit -m "test(pmcs-sbs): verify progress sync API"
```

---

## Implementation Notes

- Do not modify `api/library/pmcs_sbs`; the public content API remains unchanged.
- Do not add database migrations in this pass; the tables and generated models already exist.
- Keep authentication under `/api/v1/auth`.
- Never trust `user_uid` from request JSON. Always use `bootstrap.User.UserID`.
- Child table access must validate ownership through `pmcs_sbs_equipment`.
- Completion delete should not require `item_no`; the primary key uses `equipment_id`, `section_id`, `item_index`, and `step_id`.
- Fault updates must preserve `created_at`; only `updated_at` changes on conflict.
- Avoid returning raw database errors in HTTP responses.

## Plan Self-Review

- Spec coverage: endpoints, hard deletes, client UUIDs, `equipment_manual` blob path, one fault per item, DA Form status values, completion delete semantics, batch explicit change sets, auth, and tests are all mapped to tasks.
- Scope: one backend subsystem, no schema migration, no frontend work.
- Type consistency: DTO names and service/repository method names are defined once and reused across tasks.
