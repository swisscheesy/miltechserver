# Equipment + PMCS History Aggregate Endpoint Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `GET /api/v1/auth/shops/equipment-pmcs-history`, returning every vehicle a user has access to across all their shops, each with its PMCS SBS inspection history nested underneath, in one call.

**Architecture:** A new method on the existing `api/shops/aggregates` package's repository runs two batched jet queries (all accessible equipment, then all PMCS inspections for those equipment ids with fault counts) and merges them in Go. The service/handler/route layers follow this package's existing `GetBootstrap` pattern exactly — no request options, no validation, thin pass-through.

**Tech Stack:** Go, Gin, go-jet v2.13.0, PostgreSQL, `github.com/google/uuid`, `testify/require`.

## Global Constraints

- Spec: `docs/superpowers/specs/2026-07-17-equipment-pmcs-history-endpoint-design.md` — every task below implements a section of it; read it first if anything here is ambiguous.
- No query parameters, no pagination, no per-vehicle inspection cap — every accessible vehicle and its complete inspection history is always returned.
- Equipment with zero PMCS history is still included, with `historical_pmcs: []` (never `null`).
- PMCS detail is summary-only (`id`, `guide_manual`, `performed_date`, `fault_count`, `created_at`) — no nested fault line-items.
- `historical_pmcs` is ordered most-recent-`performed_date`-first within each vehicle.
- Access control is entirely the join itself (`shop_members.user_id = <caller>`) — no separate per-vehicle authorization check, matching `GetShopEquipmentOverview`'s existing model.
- Follow existing project conventions: go-jet dot-imported from `miltechserver/.gen/miltech_ng/public/table`, `model` package imported normally, `bootstrap.User` for the authenticated caller, `response.StandardResponse{Status, Data, Message}` for success bodies, `gin.H{"message": ...}` for error bodies, `writeAggregateError` for this package's error mapping.
- **Task boundary note:** Task 1 adds the new method to the concrete `*RepositoryImpl` type only — it deliberately does NOT touch the `Repository` interface in `repository.go`. This keeps `go build ./...` and every existing test passing throughout Task 1, since Go permits calling a method that exists on a concrete type but not yet on the interface it implements. Task 2 adds the interface declaration (and updates the hand-written test stub that implements `Repository`) at the same time it adds the service/handler/route layers that actually need to call the method through the interface.

---

### Task 1: Response Types and Repository Implementation

**Files:**
- Modify: `api/response/user_shops_response.go` (append new types after line 214)
- Modify: `api/shops/aggregates/repository_impl.go` (append new method after line 1149; add imports)
- Modify: `tests/shops/helpers_test.go` (append two new helpers after line 310; add `uuid` import)
- Create: `tests/shops/shops_equipment_pmcs_history_test.go`

**Interfaces:**
- Consumes: `model.PmcsSbsInspections`, jet tables `ShopVehicle`, `ShopMembers`, `PmcsSbsInspections`, `PmcsSbsFaults` (all pre-existing, from `miltechserver/.gen/miltech_ng/public/table`).
- Produces: `response.EquipmentPmcsHistoryResponse{Equipment []EquipmentWithPmcsHistory, Count int}`, `response.EquipmentWithPmcsHistory{ShopEquipmentSummary, ShopID string, HistoricalPmcs []PmcsHistorySummary}`, `response.PmcsHistorySummary{ID uuid.UUID, GuideManual string, PerformedDate time.Time, FaultCount int, CreatedAt time.Time}`, and `(*RepositoryImpl).GetEquipmentPmcsHistory(ctx context.Context, user *bootstrap.User) ([]response.EquipmentWithPmcsHistory, error)` — Task 2 is written directly against these names and signatures.

- [ ] **Step 1: Add the new response types**

Open `api/response/user_shops_response.go`. Add `"github.com/google/uuid"` to the import block so it reads:

```go
import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
	"time"

	"github.com/google/uuid"
)
```

Append at the end of the file (after `ShopBootstrapSummary`, currently ending at line 214):

```go

type EquipmentPmcsHistoryResponse struct {
	Equipment []EquipmentWithPmcsHistory `json:"equipment"`
	Count     int                        `json:"count"`
}

type EquipmentWithPmcsHistory struct {
	ShopEquipmentSummary
	ShopID         string               `json:"shop_id"`
	HistoricalPmcs []PmcsHistorySummary `json:"historical_pmcs"`
}

type PmcsHistorySummary struct {
	ID            uuid.UUID `json:"id"`
	GuideManual   string    `json:"guide_manual"`
	PerformedDate time.Time `json:"performed_date"`
	FaultCount    int       `json:"fault_count"`
	CreatedAt     time.Time `json:"created_at"`
}
```

- [ ] **Step 2: Add test helpers for seeding PMCS data directly**

Open `tests/shops/helpers_test.go`. Add `"github.com/google/uuid"` to the import block so it reads:

```go
import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"miltechserver/api/middleware"
	"miltechserver/api/shops"
	"miltechserver/bootstrap"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)
```

Append at the end of the file (after `createMessage`, currently ending at line 310):

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

func createPmcsFault(t *testing.T, db *sql.DB, pmcsID string, sectionID string, itemIndex int) {
	t.Helper()

	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO pmcs_sbs_faults (pmcs_id, section_id, item_index, item_no, status, fault_text, corrective_action, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)`,
		pmcsID, sectionID, itemIndex, "1", "x", "test fault", "", now,
	)
	require.NoError(t, err)
}
```

`pmcs_sbs_inspections`/`pmcs_sbs_faults` are not listed in `clearShopTables`'s `TRUNCATE` statement, but that statement already includes `shop_vehicle ... CASCADE`, and `pmcs_sbs_inspections.equipment_id` has an `ON DELETE CASCADE` foreign key to `shop_vehicle.id` (with `pmcs_sbs_faults.pmcs_id` cascading from `pmcs_sbs_inspections.id` in turn) — so `TRUNCATE ... CASCADE` on `shop_vehicle` already clears both tables. No change to `clearShopTables` itself is needed.

- [ ] **Step 3: Write the failing repository test**

Create `tests/shops/shops_equipment_pmcs_history_test.go`:

```go
package shops_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"miltechserver/api/shops/aggregates"
	"miltechserver/bootstrap"

	"github.com/stretchr/testify/require"
)

func TestGetEquipmentPmcsHistoryRepository(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "history-user")
	ensureUser(t, testDB, "other-user")
	router := newTestRouter(t)

	shopID := createShop(t, router, "history-user", "History Shop")
	hiddenShopID := createShop(t, router, "other-user", "Hidden Shop")
	vehicleWithHistoryID := createVehicle(t, router, "history-user", shopID)
	vehicleWithoutHistoryID := createVehicle(t, router, "history-user", shopID)
	_ = createVehicle(t, router, "other-user", hiddenShopID)

	newerTime := time.Date(2026, time.July, 16, 14, 30, 0, 0, time.UTC)
	olderTime := newerTime.Add(-7 * 24 * time.Hour)
	newerInspectionID := createPmcsInspection(t, testDB, vehicleWithHistoryID, "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json", newerTime)
	createPmcsFault(t, testDB, newerInspectionID, "before", 0)
	createPmcsFault(t, testDB, newerInspectionID, "during", 1)
	olderInspectionID := createPmcsInspection(t, testDB, vehicleWithHistoryID, "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json", olderTime)

	repository := aggregates.NewRepository(testDB)
	equipment, err := repository.GetEquipmentPmcsHistory(context.Background(), &bootstrap.User{UserID: "history-user"})

	require.NoError(t, err)
	require.Len(t, equipment, 2)

	byID := make(map[string]int, len(equipment))
	for i, e := range equipment {
		byID[e.ID] = i
	}
	require.Contains(t, byID, vehicleWithHistoryID)
	require.Contains(t, byID, vehicleWithoutHistoryID)

	withHistory := equipment[byID[vehicleWithHistoryID]]
	require.Equal(t, shopID, withHistory.ShopID)
	require.Len(t, withHistory.HistoricalPmcs, 2)
	require.Equal(t, newerInspectionID, withHistory.HistoricalPmcs[0].ID.String())
	require.Equal(t, 2, withHistory.HistoricalPmcs[0].FaultCount)
	require.Equal(t, olderInspectionID, withHistory.HistoricalPmcs[1].ID.String())
	require.Equal(t, 0, withHistory.HistoricalPmcs[1].FaultCount)

	withoutHistory := equipment[byID[vehicleWithoutHistoryID]]
	require.Empty(t, withoutHistory.HistoricalPmcs)

	for _, e := range equipment {
		require.NotEqual(t, hiddenShopID, e.ShopID)
	}
}

func TestGetEquipmentPmcsHistoryRepositoryReturnsEmptyForUserWithNoShops(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "lonely-user")

	repository := aggregates.NewRepository(testDB)
	equipment, err := repository.GetEquipmentPmcsHistory(context.Background(), &bootstrap.User{UserID: "lonely-user"})

	require.NoError(t, err)
	require.Empty(t, equipment)
}

func TestGetEquipmentPmcsHistoryRepositoryHonorsCanceledContext(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "history-user")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repository := aggregates.NewRepository(testDB)
	_, err := repository.GetEquipmentPmcsHistory(ctx, &bootstrap.User{UserID: "history-user"})

	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled), err.Error())
}
```

- [ ] **Step 4: Run the test to confirm it fails to compile**

```bash
go test ./tests/shops/... -run TestGetEquipmentPmcsHistoryRepository -v
```

Expected: FAIL — build error, `aggregates.NewRepository(testDB)` (a `*RepositoryImpl`) has no method `GetEquipmentPmcsHistory`.

- [ ] **Step 5: Implement the repository method**

Open `api/shops/aggregates/repository_impl.go`. Update the import block (currently lines 1-14) to add the jet dot-imports and `uuid`:

```go
package aggregates

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
)
```

Append at the end of the file (after the last method, currently ending at line 1149):

```go

func (repo *RepositoryImpl) GetEquipmentPmcsHistory(ctx context.Context, user *bootstrap.User) ([]response.EquipmentWithPmcsHistory, error) {
	type equipmentRow struct {
		ID     string `sql:"id"`
		ShopID string `sql:"shop_id"`
		Admin  string `sql:"admin"`
		Model  string `sql:"model"`
		Serial string `sql:"serial"`
		Niin   string `sql:"niin"`
	}

	equipmentStmt := SELECT(
		ShopVehicle.ID.AS("id"),
		ShopVehicle.ShopID.AS("shop_id"),
		ShopVehicle.Admin.AS("admin"),
		ShopVehicle.Model.AS("model"),
		ShopVehicle.Serial.AS("serial"),
		ShopVehicle.Niin.AS("niin"),
	).FROM(
		ShopVehicle.INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(ShopVehicle.ShopID)),
	).WHERE(
		ShopMembers.UserID.EQ(String(user.UserID)),
	).ORDER_BY(
		ShopVehicle.SaveTime.DESC(),
		ShopVehicle.ID.DESC(),
	)

	var equipmentRows []equipmentRow
	if err := equipmentStmt.QueryContext(ctx, repo.db, &equipmentRows); err != nil {
		return nil, fmt.Errorf("failed to query equipment for pmcs history: %w", err)
	}
	if len(equipmentRows) == 0 {
		return []response.EquipmentWithPmcsHistory{}, nil
	}

	equipmentIDs := make([]Expression, 0, len(equipmentRows))
	for _, row := range equipmentRows {
		equipmentIDs = append(equipmentIDs, String(row.ID))
	}

	var inspections []model.PmcsSbsInspections
	inspectionsStmt := SELECT(PmcsSbsInspections.AllColumns).
		FROM(PmcsSbsInspections).
		WHERE(PmcsSbsInspections.EquipmentID.IN(equipmentIDs...)).
		ORDER_BY(PmcsSbsInspections.EquipmentID.ASC(), PmcsSbsInspections.PerformedDate.DESC())

	if err := inspectionsStmt.QueryContext(ctx, repo.db, &inspections); err != nil {
		return nil, fmt.Errorf("failed to query pmcs inspections for equipment history: %w", err)
	}

	faultCountByInspectionID := make(map[uuid.UUID]int)
	if len(inspections) > 0 {
		inspectionIDs := make([]Expression, 0, len(inspections))
		for _, inspection := range inspections {
			inspectionIDs = append(inspectionIDs, UUID(inspection.ID))
		}

		var counts []struct {
			PmcsID uuid.UUID `sql:"pmcs_id"`
			Total  int32     `sql:"total"`
		}
		countStmt := SELECT(
			PmcsSbsFaults.PmcsID.AS("pmcs_id"),
			COUNT(PmcsSbsFaults.PmcsID).AS("total"),
		).FROM(PmcsSbsFaults).
			WHERE(PmcsSbsFaults.PmcsID.IN(inspectionIDs...)).
			GROUP_BY(PmcsSbsFaults.PmcsID)

		if err := countStmt.QueryContext(ctx, repo.db, &counts); err != nil {
			return nil, fmt.Errorf("failed to count pmcs faults for equipment history: %w", err)
		}
		for _, count := range counts {
			faultCountByInspectionID[count.PmcsID] = int(count.Total)
		}
	}

	historyByEquipmentID := make(map[string][]response.PmcsHistorySummary, len(equipmentRows))
	for _, inspection := range inspections {
		historyByEquipmentID[inspection.EquipmentID] = append(historyByEquipmentID[inspection.EquipmentID], response.PmcsHistorySummary{
			ID:            inspection.ID,
			GuideManual:   inspection.GuideManual,
			PerformedDate: inspection.PerformedDate,
			FaultCount:    faultCountByInspectionID[inspection.ID],
			CreatedAt:     inspection.CreatedAt,
		})
	}

	equipment := make([]response.EquipmentWithPmcsHistory, 0, len(equipmentRows))
	for _, row := range equipmentRows {
		history := historyByEquipmentID[row.ID]
		if history == nil {
			history = []response.PmcsHistorySummary{}
		}
		equipment = append(equipment, response.EquipmentWithPmcsHistory{
			ShopEquipmentSummary: response.ShopEquipmentSummary{
				ID:     row.ID,
				Admin:  row.Admin,
				Model:  row.Model,
				Serial: row.Serial,
				Niin:   row.Niin,
			},
			ShopID:         row.ShopID,
			HistoricalPmcs: history,
		})
	}
	return equipment, nil
}
```

Two queries total regardless of how many vehicles or inspections exist: one for equipment, one batched `IN (...)` query for inspections (skipped entirely if there are zero vehicles), plus one batched `IN (...)` query for fault counts (skipped if there are zero inspections) — never a per-row query. The explicit `.AS(...)` on every selected column, paired with matching `sql:"..."` tags on the destination structs, mirrors the exact pattern `pmcs_sbs_progress.RepositoryImpl.ListInspections` already uses for its fault-count query — that pattern exists because go-jet's default column naming can be table-qualified (e.g. `shop_vehicle.id`) rather than bare (`id`), which silently zero-values a destination field expecting the bare name if the aliasing is skipped. `model.PmcsSbsInspections` is scanned directly via `.AllColumns` (no custom aliasing needed) because that struct is the exact go-jet-generated type for that exact table — the same safe pattern `ListInspections` uses for its own inspections query.

- [ ] **Step 6: Run the test to confirm it passes**

```bash
go test ./tests/shops/... -run TestGetEquipmentPmcsHistoryRepository -v
```

Expected: PASS — all three tests from Step 3.

- [ ] **Step 7: Confirm the whole module still builds and no other tests broke**

```bash
go build ./...
go vet ./...
go test ./tests/shops/... -v
```

Expected: all succeed. Unlike the PMCS SBS inspection-history plan's Task 1→2 boundary, there is no expected breakage here — `repository.go`'s `Repository` interface was not touched, so nothing else depends on this new method yet.

- [ ] **Step 8: Commit**

```bash
git add api/response/user_shops_response.go api/shops/aggregates/repository_impl.go tests/shops/helpers_test.go tests/shops/shops_equipment_pmcs_history_test.go
git commit -m "feat(shops): add equipment pmcs history repository query"
```

---

### Task 2: Repository Interface, Service, Handler, and Route

**Files:**
- Modify: `api/shops/aggregates/repository.go`
- Modify: `api/shops/aggregates/service.go`
- Modify: `api/shops/aggregates/service_impl.go`
- Modify: `api/shops/aggregates/service_impl_test.go`
- Modify: `api/shops/aggregates/handler.go`
- Modify: `api/shops/aggregates/handler_test.go`
- Modify: `api/shops/aggregates/route.go`
- Modify: `tests/shops/shops_equipment_pmcs_history_test.go` (append HTTP-level tests)

**Interfaces:**
- Consumes: `(*RepositoryImpl).GetEquipmentPmcsHistory`, `response.EquipmentPmcsHistoryResponse`, `response.EquipmentWithPmcsHistory`, `response.PmcsHistorySummary` from Task 1.
- Produces: `GET /api/v1/auth/shops/equipment-pmcs-history`, registered and reachable end-to-end.

- [ ] **Step 1: Add the interface method and stub, and write failing service tests**

Open `api/shops/aggregates/repository.go`. Add one line to the `Repository` interface (after `GetBootstrap`, currently the last line before the closing brace at line 19):

```go
type Repository interface {
	GetListsWithItems(ctx context.Context, user *bootstrap.User, shopID string, limits ListTreeLimits) ([]response.ShopListWithItems, error)
	GetVehicleByIDForMember(ctx context.Context, user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error)
	GetVehicleNotificationsWithItems(ctx context.Context, vehicleID string, limits SnapshotLimits) ([]response.VehicleNotificationWithItems, error)
	GetVehicleRecentChanges(ctx context.Context, vehicleID string, limit int) ([]response.NotificationChangeWithUsername, error)
	GetVehicleServices(ctx context.Context, vehicleID string, limit int) ([]response.EquipmentServiceResponse, error)
	GetShopSnapshot(ctx context.Context, user *bootstrap.User, shopID string, options ShopSnapshotOptions) (*response.ShopSnapshotResponse, error)
	GetBootstrap(ctx context.Context, user *bootstrap.User, options BootstrapOptions) ([]response.ShopBootstrapSummary, error)
	GetEquipmentPmcsHistory(ctx context.Context, user *bootstrap.User) ([]response.EquipmentWithPmcsHistory, error)
}
```

Open `api/shops/aggregates/service_impl_test.go`. Add two fields to `repositoryStubForService` (currently lines 52-60):

```go
type repositoryStubForService struct {
	listsResp            []response.ShopListWithItems
	listsErr             error
	vehicleResp          *model.ShopVehicle
	vehicleErr           error
	notifications        []response.VehicleNotificationWithItems
	changes              []response.NotificationChangeWithUsername
	services             []response.EquipmentServiceResponse
	equipmentHistoryResp []response.EquipmentWithPmcsHistory
	equipmentHistoryErr  error
}
```

Add the stub method (after `GetBootstrap`, currently lines 86-88):

```go
func (r repositoryStubForService) GetEquipmentPmcsHistory(context.Context, *bootstrap.User) ([]response.EquipmentWithPmcsHistory, error) {
	return r.equipmentHistoryResp, r.equipmentHistoryErr
}
```

Append these failing tests at the end of the file:

```go

func TestGetEquipmentPmcsHistoryRequiresAuth(t *testing.T) {
	service := NewService(repositoryStubForService{}, authStubForService{})

	_, err := service.GetEquipmentPmcsHistory(context.Background(), nil)

	require.ErrorIs(t, err, ErrUnauthorized)
}

func TestGetEquipmentPmcsHistoryReturnsRepositoryResult(t *testing.T) {
	expected := []response.EquipmentWithPmcsHistory{
		{
			ShopEquipmentSummary: response.ShopEquipmentSummary{ID: "vehicle-1", Admin: "A1", Model: "M1", Serial: "S1", Niin: "N1"},
			ShopID:               "shop-1",
			HistoricalPmcs:       []response.PmcsHistorySummary{{GuideManual: "pmcs_sbs/hmmwv/file.json"}},
		},
	}
	service := NewService(repositoryStubForService{equipmentHistoryResp: expected}, authStubForService{})

	result, err := service.GetEquipmentPmcsHistory(context.Background(), &bootstrap.User{UserID: "user-1"})

	require.NoError(t, err)
	require.Equal(t, 1, result.Count)
	require.Equal(t, expected, result.Equipment)
}

func TestGetEquipmentPmcsHistoryNormalizesNilSlices(t *testing.T) {
	service := NewService(repositoryStubForService{equipmentHistoryResp: []response.EquipmentWithPmcsHistory{
		{ShopEquipmentSummary: response.ShopEquipmentSummary{ID: "vehicle-1"}, ShopID: "shop-1", HistoricalPmcs: nil},
	}}, authStubForService{})

	result, err := service.GetEquipmentPmcsHistory(context.Background(), &bootstrap.User{UserID: "user-1"})

	require.NoError(t, err)
	require.NotNil(t, result.Equipment[0].HistoricalPmcs)
	require.Empty(t, result.Equipment[0].HistoricalPmcs)
}

func TestGetEquipmentPmcsHistoryWrapsRepositoryError(t *testing.T) {
	service := NewService(repositoryStubForService{equipmentHistoryErr: errors.New("db exploded")}, authStubForService{})

	_, err := service.GetEquipmentPmcsHistory(context.Background(), &bootstrap.User{UserID: "user-1"})

	require.ErrorIs(t, err, ErrAggregateUnavailable)
}
```

- [ ] **Step 2: Run the service tests to confirm they fail to compile**

```bash
go test ./api/shops/aggregates/... -run TestGetEquipmentPmcsHistory -v
```

Expected: FAIL — build error, `Service` has no method `GetEquipmentPmcsHistory` yet.

- [ ] **Step 3: Add the method to the Service interface and implement it**

Open `api/shops/aggregates/service.go`. Add one line to the `Service` interface (after `GetBootstrap`, currently the last line before the closing brace at line 44):

```go
type Service interface {
	GetListsWithItems(ctx context.Context, user *bootstrap.User, shopID string, limits ListTreeLimits) (*response.ShopListsWithItemsResponse, error)
	GetVehicleMaintenanceSnapshot(ctx context.Context, user *bootstrap.User, vehicleID string, limits SnapshotLimits) (*response.VehicleMaintenanceSnapshotResponse, error)
	GetShopSnapshot(ctx context.Context, user *bootstrap.User, shopID string, options ShopSnapshotOptions) (*response.ShopSnapshotResponse, error)
	GetBootstrap(ctx context.Context, user *bootstrap.User, options BootstrapOptions) (*response.ShopsBootstrapResponse, error)
	GetEquipmentPmcsHistory(ctx context.Context, user *bootstrap.User) (*response.EquipmentPmcsHistoryResponse, error)
}
```

Open `api/shops/aggregates/service_impl.go`. Append this method after `GetBootstrap` (currently ending at line 220):

```go

func (s *ServiceImpl) GetEquipmentPmcsHistory(ctx context.Context, user *bootstrap.User) (*response.EquipmentPmcsHistoryResponse, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
	equipment, err := s.repo.GetEquipmentPmcsHistory(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAggregateUnavailable, err)
	}
	if equipment == nil {
		equipment = []response.EquipmentWithPmcsHistory{}
	}
	for i := range equipment {
		if equipment[i].HistoricalPmcs == nil {
			equipment[i].HistoricalPmcs = []response.PmcsHistorySummary{}
		}
	}
	return &response.EquipmentPmcsHistoryResponse{Equipment: equipment, Count: len(equipment)}, nil
}
```

- [ ] **Step 4: Run the service tests again**

```bash
go test ./api/shops/aggregates/... -run TestGetEquipmentPmcsHistory -v
```

Expected: PASS — all four tests from Step 1.

- [ ] **Step 5: Add the stub method to the handler test, and write failing handler tests**

Open `api/shops/aggregates/handler_test.go`. Add two fields to `serviceStub` (currently lines 18-21):

```go
type serviceStub struct {
	listsResp            *response.ShopListsWithItemsResponse
	equipmentHistoryResp *response.EquipmentPmcsHistoryResponse
	equipmentHistoryErr  error
	err                  error
}
```

Add the stub method (after `GetBootstrap`, currently lines 32-34):

```go
func (s serviceStub) GetEquipmentPmcsHistory(context.Context, *bootstrap.User) (*response.EquipmentPmcsHistoryResponse, error) {
	return s.equipmentHistoryResp, s.equipmentHistoryErr
}
```

Append these failing tests at the end of the file:

```go

func TestEquipmentPmcsHistoryRequiresUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1/auth"), serviceStub{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/shops/equipment-pmcs-history", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestEquipmentPmcsHistoryReturnsStandardResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", &bootstrap.User{UserID: "user-1"})
		c.Next()
	})
	RegisterRoutes(router.Group("/api/v1/auth"), serviceStub{
		equipmentHistoryResp: &response.EquipmentPmcsHistoryResponse{
			Equipment: []response.EquipmentWithPmcsHistory{},
			Count:     0,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/shops/equipment-pmcs-history", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)

	var payload struct {
		Status int `json:"status"`
		Data   struct {
			Equipment []response.EquipmentWithPmcsHistory `json:"equipment"`
			Count     int                                 `json:"count"`
		} `json:"data"`
		Message string `json:"message"`
	}
	err := json.Unmarshal(resp.Body.Bytes(), &payload)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, payload.Status)
	require.Equal(t, "Equipment PMCS history retrieved successfully", payload.Message)
	require.Empty(t, payload.Data.Equipment)
}
```

- [ ] **Step 6: Run the handler tests to confirm they fail**

```bash
go test ./api/shops/aggregates/... -run TestEquipmentPmcsHistory -v
```

Expected: FAIL — `TestEquipmentPmcsHistoryRequiresUser` gets a 404 (route not registered) instead of 401; `TestEquipmentPmcsHistoryReturnsStandardResponse` also 404s.

- [ ] **Step 7: Implement the handler and register the route**

Open `api/shops/aggregates/handler.go`. Append this handler after `getBootstrap` (currently ending at line 123):

```go

func (handler Handler) getEquipmentPmcsHistory(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	result, err := handler.service.GetEquipmentPmcsHistory(c.Request.Context(), user)
	if err != nil {
		writeAggregateError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Equipment PMCS history retrieved successfully",
		Data:    result,
	})
}
```

Open `api/shops/aggregates/route.go`. Add one line to `RegisterRoutes` (after the `maintenance-snapshot` line, currently line 13):

```go
func RegisterRoutes(router *gin.RouterGroup, service Service) {
	handler := Handler{service: service}
	router.GET("/shops/bootstrap", gzip.Gzip(gzip.DefaultCompression), handler.getBootstrap)
	router.GET("/shops/:shop_id/snapshot", gzip.Gzip(gzip.DefaultCompression), handler.getShopSnapshot)
	router.GET("/shops/:shop_id/lists-with-items", gzip.Gzip(gzip.DefaultCompression), handler.getListsWithItems)
	router.GET("/shops/vehicles/:vehicle_id/maintenance-snapshot", gzip.Gzip(gzip.DefaultCompression), handler.getVehicleMaintenanceSnapshot)
	router.GET("/shops/equipment-pmcs-history", gzip.Gzip(gzip.DefaultCompression), handler.getEquipmentPmcsHistory)
}
```

- [ ] **Step 8: Run the handler tests again**

```bash
go test ./api/shops/aggregates/... -v
```

Expected: PASS — every test in the package, including the two new ones and all pre-existing ones (confirms the two stub additions didn't break any other test in this file).

- [ ] **Step 9: Write and run the end-to-end HTTP test**

Append to `tests/shops/shops_equipment_pmcs_history_test.go` (the file Task 1 created):

```go

func TestEquipmentPmcsHistoryEndpoint(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "history-user")
	ensureUser(t, testDB, "other-user")
	router := newTestRouter(t)

	shopID := createShop(t, router, "history-user", "History Shop")
	hiddenShopID := createShop(t, router, "other-user", "Hidden Shop")
	vehicleID := createVehicle(t, router, "history-user", shopID)
	_ = createVehicle(t, router, "other-user", hiddenShopID)
	_, err := testDB.Exec(`UPDATE shop_vehicle SET admin='A1', model='M1152A1', serial='SER-1', niin='000000001' WHERE id=$1`, vehicleID)
	require.NoError(t, err)

	performedDate := time.Date(2026, time.July, 16, 14, 30, 0, 0, time.UTC)
	inspectionID := createPmcsInspection(t, testDB, vehicleID, "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json", performedDate)
	createPmcsFault(t, testDB, inspectionID, "before", 0)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/equipment-pmcs-history", nil, "history-user")
	require.Equal(t, http.StatusOK, resp.Code)

	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	equipment := payload["equipment"].([]interface{})
	require.Len(t, equipment, 1)

	entry := equipment[0].(map[string]interface{})
	require.Equal(t, vehicleID, entry["id"])
	require.Equal(t, shopID, entry["shop_id"])
	require.Equal(t, "A1", entry["admin"])

	history := entry["historical_pmcs"].([]interface{})
	require.Len(t, history, 1)
	inspection := history[0].(map[string]interface{})
	require.Equal(t, inspectionID, inspection["id"])
	require.Equal(t, "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json", inspection["guide_manual"])
	require.Equal(t, float64(1), inspection["fault_count"])
}

func TestEquipmentPmcsHistoryEndpointRequiresAuthentication(t *testing.T) {
	router := newTestRouter(t)
	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/equipment-pmcs-history", nil, "")
	require.Equal(t, http.StatusUnauthorized, resp.Code)
}
```

Add `"net/http"` and `"github.com/stretchr/testify/require"` to this file's imports if not already present from Task 1 (Task 1's version already imports `"context"`, `"errors"`, `"testing"`, `"time"`, `"miltechserver/api/shops/aggregates"`, `"miltechserver/bootstrap"`, `"github.com/stretchr/testify/require"` — add `"net/http"` alongside them).

Run:

```bash
go test ./tests/shops/... -run TestEquipmentPmcsHistoryEndpoint -v
```

Expected: PASS — both new tests.

- [ ] **Step 10: Run the full test suite for both affected packages**

```bash
go build ./...
go vet ./...
go test ./api/shops/aggregates/... ./tests/shops/... -v
```

Expected: `go build ./...` succeeds, all tests pass, output pristine (no stray warnings).

- [ ] **Step 11: Commit**

```bash
git add api/shops/aggregates/repository.go api/shops/aggregates/service.go api/shops/aggregates/service_impl.go api/shops/aggregates/service_impl_test.go api/shops/aggregates/handler.go api/shops/aggregates/handler_test.go api/shops/aggregates/route.go tests/shops/shops_equipment_pmcs_history_test.go
git commit -m "feat(shops): add equipment pmcs history endpoint"
```

---

### Task 3: Mobile Documentation

**Files:**
- Modify: `docs/api/pmcs_sbs_inspection_history_mobile_changes.md`

**Interfaces:**
- None — documentation only. Can run any time after Task 2 merges.

- [ ] **Step 1: Add a new section documenting the endpoint**

Open `docs/api/pmcs_sbs_inspection_history_mobile_changes.md`. Insert a new section after the "Full Endpoint Reference" section's endpoint 7 (`DELETE .../faults/bulk`) and before "## Object Reference" (currently the file transitions from endpoint 7 straight into `## Object Reference` after the closing `---`):

```markdown
### 8. `GET /shops/equipment-pmcs-history`

Not part of the PMCS SBS route group (it lives under `/shops`, alongside the other Shops aggregate endpoints), but returns PMCS inspection history batched across every vehicle the user can access, so it's documented here for convenience.

Returns every vehicle the authenticated user has access to, across every shop they belong to, each with its PMCS inspection history (summary form — same fields as the list endpoint above, no nested faults) nested underneath. Equipment with no PMCS history yet is included with an empty `historical_pmcs` array. There are no query parameters, no pagination, and no per-vehicle cap — every accessible vehicle and its complete inspection history is always returned in one response.

Success response (`200`):

```json
{
  "status": 200,
  "data": {
    "equipment": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "shop_id": "b2f1c1b4-1111-4a2b-9c3d-000000000001",
        "admin": "B1",
        "model": "M1152A1",
        "serial": "SER-B1",
        "niin": "",
        "historical_pmcs": [
          {
            "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
            "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
            "performed_date": "2026-07-16T14:30:00Z",
            "fault_count": 3,
            "created_at": "2026-07-16T14:31:02.123456Z"
          },
          {
            "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
            "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
            "performed_date": "2026-07-09T09:00:00Z",
            "fault_count": 0,
            "created_at": "2026-07-09T09:05:44.001122Z"
          }
        ]
      },
      {
        "id": "9d8e7f6a-2222-4b3c-8d4e-000000000002",
        "shop_id": "b2f1c1b4-1111-4a2b-9c3d-000000000001",
        "admin": "B2",
        "model": "M998",
        "serial": "SER-B2",
        "niin": "",
        "historical_pmcs": []
      }
    ],
    "count": 2
  },
  "message": ""
}
```

`historical_pmcs` is ordered most-recent-`performed_date`-first within each vehicle, same as the inspection list endpoint. `count` is the number of equipment entries in this response (there is no pagination, so this is always the caller's full accessible equipment count). `shop_id` identifies which shop each vehicle belongs to — the equipment list itself is flat, not grouped by shop.

To see full fault detail (not just a count) for one specific inspection, call `GET /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id` (endpoint 2 above) with that inspection's `id`.

Error responses: `401` (`{"message":"unauthorized"}`) for missing/invalid authentication; `500` for an unexpected server error. There is no `400`/`404`/`409` case for this endpoint — it takes no parameters and its equipment list is inherently scoped to what the caller can access.
```

- [ ] **Step 2: Commit**

```bash
git add docs/api/pmcs_sbs_inspection_history_mobile_changes.md
git commit -m "docs(shops): document equipment pmcs history endpoint"
```
