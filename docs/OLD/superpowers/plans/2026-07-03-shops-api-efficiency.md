# Shops API Efficiency Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Improve Shops API efficiency with backward-compatible query fixes and additive aggregate endpoints that reduce mobile/client round trips without removing or reshaping existing endpoints.

**Architecture:** Keep existing endpoints and response contracts intact. First fix query-shape problems that affect current endpoints, then add new aggregate read endpoints under the authenticated `/api/v1/auth/shops` surface using dedicated DTOs, membership-rooted queries, bounded high-cardinality sections, request-context cancellation, and endpoint-scoped gzip where payloads can become large.

**Tech Stack:** Go, Gin, PostgreSQL, Jet, Firebase-authenticated Gin context, `gin-contrib/gzip`, Testify, existing `response.StandardResponse` envelope.

**Source analysis:** Based on the live Shops API analysis from 2026-07-03, including `api/shops`, `api/equipment_services`, live test-database index inspection, Gin routing/middleware docs, and Azure PostgreSQL `EXPLAIN ANALYZE` guidance.

---

## Non-Negotiable Compatibility Rules

- Do not remove any existing route.
- Do not change existing successful response envelopes, field names, status codes, or legacy unbounded behavior unless a task explicitly says the change is a bug fix that preserves shape.
- New endpoints must be additive.
- Existing endpoints may receive internal query fixes, added context propagation, logging, or deduplication only when the external contract remains the same.
- Do not hand-edit generated Jet files under `.gen/`.
- Do not add database migrations unless representative `EXPLAIN (ANALYZE, BUFFERS)` evidence proves an index is needed and the migration is called out in a separate review.
- Keep high-cardinality aggregate sections bounded by default; do not create another unbounded "everything in a shop" endpoint for messages, changes, or service history.
- Prefer dedicated DTOs for aggregate endpoints. Where an existing response DTO already exposes a stable generated-model shape, reuse it only behind aggregate-owned top-level DTOs and do not add fields to existing endpoint responses.

---

## File Map

### Existing files to modify

- `api/shops/route.go`: wire new aggregate package routes before `core.RegisterRoutes` so static aggregate paths such as `/shops/bootstrap` are registered before the existing `/shops/:shop_id` route.
- `api/shops/core/repository_impl.go`: rewrite `GetShopsWithStatsForUser` to aggregate only the authenticated user's shops.
- `api/equipment_services/queries/repository_impl.go`: add authenticated-user membership predicates to current joins.
- `api/equipment_services/calendar/repository_impl.go`: add authenticated-user membership predicate to current join.
- `api/equipment_services/status/repository_impl.go`: add authenticated-user membership predicates to current joins.
- `api/response/user_shops_response.go`: add stable DTOs for list tree, shop snapshot, vehicle maintenance snapshot, and bootstrap responses.
- `docs/api/shop_equipment_overview_mobile.md`: link related new aggregate docs without changing existing contract wording.
- `docs/project_notes/decisions.md`: add a short ADR after endpoints are implemented and verified.

### New files to create

- `tests/equipment_services/equipment_services_query_regression_test.go`: regression coverage for duplicate rows/counts in existing equipment-service endpoints.
- `tests/shops/shops_user_data_stats_test.go`: regression coverage for user-data stats query behavior.
- `api/shops/aggregates/repository.go`: aggregate repository interface.
- `api/shops/aggregates/repository_impl.go`: set-based aggregate queries.
- `api/shops/aggregates/service.go`: aggregate service interface.
- `api/shops/aggregates/service_impl.go`: authorization, normalization, limits, and generic error handling.
- `api/shops/aggregates/handler.go`: HTTP handlers for new additive endpoints.
- `api/shops/aggregates/route.go`: route registration for new additive endpoints.
- `api/shops/aggregates/errors.go`: package-local typed errors.
- `api/shops/aggregates/repository_impl_test.go`: focused repository tests where pure integration tests are useful.
- `api/shops/aggregates/service_impl_test.go`: limit/default/normalization tests.
- `api/shops/aggregates/handler_test.go`: HTTP boundary tests.
- `tests/shops/shops_aggregate_lists_test.go`: integration tests for list tree endpoint.
- `tests/shops/shops_aggregate_vehicle_maintenance_test.go`: integration tests for vehicle maintenance snapshot endpoint.
- `tests/shops/shops_aggregate_shop_snapshot_test.go`: integration tests for shop snapshot endpoint.
- `tests/shops/shops_aggregate_bootstrap_test.go`: integration tests for bootstrap endpoint.
- `tests/shops/shops_aggregate_performance_test.go`: opt-in representative aggregate performance tests.
- `docs/api/shops_api_efficiency_mobile.md`: mobile-facing contract document for new aggregate endpoints.

---

## Route Contracts To Add

All new routes are authenticated under `/api/v1/auth` and return `response.StandardResponse`.

### `GET /shops/:shop_id/lists-with-items`

Purpose: replace the client pattern of calling `GET /shops/:shop_id/lists` and then `GET /shops/lists/:list_id/items` once per list.

Query parameters: none.

Response data:

```json
{
  "lists": [
    {
      "id": "list-id",
      "shop_id": "shop-id",
      "created_by": "user-id",
      "created_by_username": "user name",
      "description": "Parts list",
      "created_at": "2026-07-03T12:00:00Z",
      "updated_at": "2026-07-03T12:00:00Z",
      "items": [
        {
          "id": "item-id",
          "list_id": "list-id",
          "niin": "012345678",
          "nomenclature": "Item",
          "quantity": 1,
          "added_by": "user-id",
          "added_by_username": "user name",
          "created_at": "2026-07-03T12:00:00Z",
          "updated_at": "2026-07-03T12:00:00Z",
          "nickname": null,
          "unit_of_measure": "ea"
        }
      ]
    }
  ]
}
```

### `GET /shops/vehicles/:vehicle_id/maintenance-snapshot`

Purpose: load the vehicle maintenance screen in one request.

Query parameters:

- `services_limit`: default `50`, min `1`, max `200`.
- `changes_limit`: default `50`, min `1`, max `200`.

Response data:

```json
{
  "vehicle": { "id": "vehicle-id", "shop_id": "shop-id", "admin": "A123" },
  "notifications": [
    {
      "notification": { "id": "notification-id", "vehicle_id": "vehicle-id" },
      "items": []
    }
  ],
  "recent_changes": [],
  "services": [],
  "counts": {
    "notifications": 1,
    "notification_items": 0,
    "recent_changes": 0,
    "services": 0
  }
}
```

The `vehicle`, `notification`, item, change, and service objects may initially reuse existing response model shapes to keep behavior familiar, but the top-level snapshot DTO must be owned by `api/response`.

### `GET /shops/:shop_id/snapshot`

Purpose: load a shop detail dashboard without calling shop detail, settings, vehicles, lists, notification summaries, and message preview separately.

Query parameters:

- `include`: comma-separated optional sections. Supported values: `vehicles`, `lists`, `notifications`, `messages`, `services`, `changes`.
- `message_limit`: default `20`, max `100`.
- `changes_limit`: default `50`, max `200`.
- `services_limit`: default `50`, max `200`.

Default include set: `vehicles,lists,notifications,services`.

Response data:

```json
{
  "shop": {
    "id": "shop-id",
    "name": "Alpha Shop",
    "details": "Maintenance section",
    "role": "admin",
    "is_admin": true,
    "settings": { "admin_only_lists": false },
    "counts": {
      "members": 2,
      "vehicles": 4,
      "lists": 1,
      "messages": 12,
      "notifications": 3,
      "open_services": 2
    }
  },
  "vehicles": [],
  "lists": [],
  "notifications": [],
  "messages": [],
  "services": [],
  "recent_changes": []
}
```

### `GET /shops/bootstrap`

Purpose: load the first Shops app screen across all shops without per-shop setup calls.

Query parameters:

- `equipment_limit_per_shop`: default `50`, max `250`.

Response data:

```json
{
  "shops": [
    {
      "id": "shop-id",
      "name": "Alpha Shop",
      "details": "Maintenance section",
      "role": "admin",
      "is_admin": true,
      "settings": { "admin_only_lists": false },
      "counts": {
        "members": 2,
        "vehicles": 4,
        "lists": 1,
        "messages": 12,
        "notifications": 3,
        "open_services": 2
      },
      "equipment": []
    }
  ]
}
```

This does not replace `GET /shops/equipment/overview`. The existing overview remains the unbounded compact equipment endpoint documented in `docs/api/shop_equipment_overview_mobile.md`.

---

## Task 1: Lock Current Equipment-Service Query Behavior

**Files:**
- Create: `tests/equipment_services/equipment_services_query_regression_test.go`
- Modify: `api/equipment_services/queries/repository_impl.go`
- Modify: `api/equipment_services/calendar/repository_impl.go`
- Modify: `api/equipment_services/status/repository_impl.go`

- [ ] **Step 1: Write regression tests that expose duplicate rows and inflated counts**

Create `tests/equipment_services/equipment_services_query_regression_test.go`:

```go
package equipment_services_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEquipmentServicesQueriesDoNotDuplicateForMultipleShopMembers(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "owner")
	ensureUser(t, testDB, "member")
	router := newTestRouter(t)

	shopID := createShop(t, router, "owner", "Multi Member Services")
	_, inviteCode := createInviteCode(t, router, "owner", shopID)
	joinResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/join", map[string]any{"invite_code": inviteCode}, "member")
	require.Equal(t, http.StatusOK, joinResp.Code)

	equipmentID := createVehicle(t, router, "owner", shopID)
	serviceDate := time.Now().AddDate(0, 0, 3)
	createEquipmentService(t, router, "owner", shopID, equipmentID, "", "One service", &serviceDate, false)

	listResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/equipment-services", nil, "owner")
	require.Equal(t, http.StatusOK, listResp.Code)
	listPayload := decodeMap(t, decodeStandardResponse(t, listResp.Body).Data)
	require.Equal(t, float64(1), listPayload["total_count"])
	require.Len(t, listPayload["services"].([]interface{}), 1)

	equipmentResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/equipment/"+equipmentID+"/services", nil, "owner")
	require.Equal(t, http.StatusOK, equipmentResp.Code)
	equipmentPayload := decodeMap(t, decodeStandardResponse(t, equipmentResp.Body).Data)
	require.Equal(t, float64(1), equipmentPayload["total_count"])
	require.Len(t, equipmentPayload["services"].([]interface{}), 1)
}

func TestEquipmentServicesCalendarAndStatusDoNotDuplicateForMultipleShopMembers(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "owner")
	ensureUser(t, testDB, "member")
	router := newTestRouter(t)

	shopID := createShop(t, router, "owner", "Multi Member Calendar")
	_, inviteCode := createInviteCode(t, router, "owner", shopID)
	joinResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/join", map[string]any{"invite_code": inviteCode}, "member")
	require.Equal(t, http.StatusOK, joinResp.Code)

	equipmentID := createVehicle(t, router, "owner", shopID)
	overdueDate := time.Now().AddDate(0, 0, -2)
	dueSoonDate := time.Now().AddDate(0, 0, 3)
	createEquipmentService(t, router, "owner", shopID, equipmentID, "", "Overdue", &overdueDate, false)
	createEquipmentService(t, router, "owner", shopID, equipmentID, "", "Due soon", &dueSoonDate, false)

	start := time.Now().AddDate(0, 0, -7).Format(time.RFC3339)
	end := time.Now().AddDate(0, 0, 7).Format(time.RFC3339)
	calendarResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/equipment-services/calendar?start_date="+start+"&end_date="+end, nil, "owner")
	require.Equal(t, http.StatusOK, calendarResp.Code)
	calendarPayload := decodeMap(t, decodeStandardResponse(t, calendarResp.Body).Data)
	require.Equal(t, float64(2), calendarPayload["total_count"])
	require.Len(t, calendarPayload["services"].([]interface{}), 2)

	overdueResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/equipment-services/overdue", nil, "owner")
	require.Equal(t, http.StatusOK, overdueResp.Code)
	overduePayload := decodeMap(t, decodeStandardResponse(t, overdueResp.Body).Data)
	require.Equal(t, float64(1), overduePayload["total_count"])
	require.Len(t, overduePayload["overdue_services"].([]interface{}), 1)

	dueSoonResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/equipment-services/due-soon", nil, "owner")
	require.Equal(t, http.StatusOK, dueSoonResp.Code)
	dueSoonPayload := decodeMap(t, decodeStandardResponse(t, dueSoonResp.Body).Data)
	require.Equal(t, float64(1), dueSoonPayload["total_count"])
	require.Len(t, dueSoonPayload["due_soon_services"].([]interface{}), 1)
}
```

- [ ] **Step 2: Run tests and confirm the current failure**

Run:

```bash
go test ./tests/equipment_services -run 'TestEquipmentServices.*DoNotDuplicate' -count=1
```

Expected before the fix: at least one assertion fails because joins against `shop_members` multiply rows/counts for multi-member shops.

- [ ] **Step 3: Fix `GetByShop` and `GetByEquipment` joins**

In `api/equipment_services/queries/repository_impl.go`, add the authenticated user predicate to both `ShopMembers` joins:

```go
EquipmentServices.
	INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(EquipmentServices.ShopID)),
```

becomes:

```go
EquipmentServices.
	INNER_JOIN(ShopMembers,
		ShopMembers.ShopID.EQ(EquipmentServices.ShopID).
			AND(ShopMembers.UserID.EQ(String(user.UserID))),
	),
```

Apply that change to both the count and data statements in `GetByShop` and `GetByEquipment`.

- [ ] **Step 4: Fix calendar join**

In `api/equipment_services/calendar/repository_impl.go`, change the join to:

```go
EquipmentServices.
	INNER_JOIN(ShopMembers,
		ShopMembers.ShopID.EQ(EquipmentServices.ShopID).
			AND(ShopMembers.UserID.EQ(String(user.UserID))),
	),
```

- [ ] **Step 5: Fix overdue and due-soon joins**

In `api/equipment_services/status/repository_impl.go`, change both joins to:

```go
EquipmentServices.
	INNER_JOIN(ShopMembers,
		ShopMembers.ShopID.EQ(EquipmentServices.ShopID).
			AND(ShopMembers.UserID.EQ(String(user.UserID))),
	),
```

- [ ] **Step 6: Verify focused equipment-service suite**

Run:

```bash
go test ./tests/equipment_services -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit Task 1**

```bash
git add tests/equipment_services/equipment_services_query_regression_test.go api/equipment_services/queries/repository_impl.go api/equipment_services/calendar/repository_impl.go api/equipment_services/status/repository_impl.go
git commit -m "fix(equipment-services): prevent duplicate member joins"
```

---

## Task 2: Rewrite User Shop Stats Around User Membership

**Files:**
- Create: `tests/shops/shops_user_data_stats_test.go`
- Modify: `api/shops/core/repository_impl.go`

- [ ] **Step 1: Write regression tests for current `/shops/user-data` contract**

Create `tests/shops/shops_user_data_stats_test.go`:

```go
package shops_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserDataWithShopsStatsRemainScopedToAuthenticatedUser(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "user-2")
	router := newTestRouter(t)

	visibleShopID := createShop(t, router, "user-1", "Visible Stats")
	hiddenShopID := createShop(t, router, "user-2", "Hidden Stats")
	_ = createVehicle(t, router, "user-1", visibleShopID)
	_ = createVehicle(t, router, "user-2", hiddenShopID)
	_ = createMessage(t, router, "user-1", visibleShopID, "visible")
	_ = createMessage(t, router, "user-2", hiddenShopID, "hidden")

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/user-data", nil, "user-1")
	require.Equal(t, http.StatusOK, resp.Code)
	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	shops := payload["shops"].([]interface{})
	require.Len(t, shops, 1)

	shopWithStats := shops[0].(map[string]interface{})
	shop := shopWithStats["shop"].(map[string]interface{})
	require.Equal(t, visibleShopID, shop["id"])
	require.Equal(t, float64(1), shopWithStats["vehicle_count"])
	require.Equal(t, true, shopWithStats["is_admin"])
	require.Equal(t, false, shopWithStats["is_lists_admin_only"])
}
```

- [ ] **Step 2: Run the contract test**

Run:

```bash
go test ./tests/shops -run TestUserDataWithShopsStatsRemainScopedToAuthenticatedUser -count=1
```

Expected before the rewrite: PASS. This test protects the legacy response shape while the query is rewritten.

- [ ] **Step 3: Rewrite the SQL query**

In `api/shops/core/repository_impl.go`, replace the SQL inside `GetShopsWithStatsForUser` with a membership-rooted query:

```sql
WITH user_shops AS (
	SELECT
		s.id,
		s.name,
		s.details,
		s.created_by,
		s.created_at,
		s.updated_at,
		s.admin_only_lists,
		sm.role
	FROM shops s
	INNER JOIN shop_members sm ON s.id = sm.shop_id
	WHERE sm.user_id = $1
),
member_stats AS (
	SELECT sm.shop_id, COUNT(*) AS member_count
	FROM shop_members sm
	INNER JOIN user_shops us ON us.id = sm.shop_id
	GROUP BY sm.shop_id
),
vehicle_stats AS (
	SELECT sv.shop_id, COUNT(*) AS vehicle_count
	FROM shop_vehicle sv
	INNER JOIN user_shops us ON us.id = sv.shop_id
	GROUP BY sv.shop_id
)
SELECT
	us.id,
	us.name,
	us.details,
	us.created_by,
	us.created_at,
	us.updated_at,
	us.admin_only_lists,
	COALESCE(member_stats.member_count, 0) AS member_count,
	COALESCE(vehicle_stats.vehicle_count, 0) AS vehicle_count,
	(us.role = 'admin') AS is_admin
FROM user_shops us
LEFT JOIN member_stats ON us.id = member_stats.shop_id
LEFT JOIN vehicle_stats ON us.id = vehicle_stats.shop_id
ORDER BY us.created_at DESC
```

Keep the existing scan target and `response.ShopWithStats` mapping so `GET /shops/user-data` remains backward compatible.

- [ ] **Step 4: Run Shops tests that cover core and overview behavior**

Run:

```bash
go test ./tests/shops -run 'TestUserDataWithShopsStatsRemainScopedToAuthenticatedUser|TestShopCoreLifecycle|TestShopEquipmentOverview' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit Task 2**

```bash
git add tests/shops/shops_user_data_stats_test.go api/shops/core/repository_impl.go
git commit -m "perf(shops): scope user shop stats aggregation"
```

---

## Task 3: Add Aggregate DTOs

**Files:**
- Modify: `api/response/user_shops_response.go`
- Create or modify: `api/response/user_shops_response_test.go`

- [ ] **Step 1: Add JSON contract tests for new DTOs**

Add these tests to `api/response/user_shops_response_test.go`:

```go
func TestShopAggregateResponseDTOsEncodeEmptyArrays(t *testing.T) {
	result := ShopListsWithItemsResponse{
		Lists: []ShopListWithItems{{
			ShopListWithUsername: ShopListWithUsername{
				ID: "list-1", ShopID: "shop-1", CreatedBy: "user-1", Description: "Parts",
			},
			Items: []ShopListItemWithUsername{},
		}},
	}

	payload, err := json.Marshal(result)
	require.NoError(t, err)
	require.Contains(t, string(payload), `"lists":[`)
	require.Contains(t, string(payload), `"items":[]`)
}

func TestShopBootstrapResponseDTOsEncodeCounts(t *testing.T) {
	result := ShopsBootstrapResponse{
		Shops: []ShopBootstrapSummary{{
			ID: "shop-1", Name: "Alpha", Role: "admin", IsAdmin: true,
			Settings: ShopAggregateSettings{AdminOnlyLists: true},
			Counts: ShopAggregateCounts{Members: 2, Vehicles: 3},
			Equipment: []ShopEquipmentSummary{},
		}},
	}

	payload, err := json.Marshal(result)
	require.NoError(t, err)
	require.Contains(t, string(payload), `"admin_only_lists":true`)
	require.Contains(t, string(payload), `"members":2`)
	require.Contains(t, string(payload), `"equipment":[]`)
}
```

- [ ] **Step 2: Run tests and confirm failure**

Run:

```bash
go test ./api/response -run 'TestShopAggregateResponseDTOs|TestShopBootstrapResponseDTOs' -count=1
```

Expected: compile failure because DTOs do not exist.

- [ ] **Step 3: Add DTOs**

Append these DTOs to `api/response/user_shops_response.go`:

```go
type ShopListsWithItemsResponse struct {
	Lists []ShopListWithItems `json:"lists"`
}

type ShopListWithItems struct {
	ShopListWithUsername
	Items []ShopListItemWithUsername `json:"items"`
}

type ShopAggregateSettings struct {
	AdminOnlyLists bool `json:"admin_only_lists"`
}

type ShopAggregateCounts struct {
	Members           int64 `json:"members"`
	Vehicles          int64 `json:"vehicles"`
	Lists             int64 `json:"lists"`
	Messages          int64 `json:"messages"`
	Notifications     int64 `json:"notifications"`
	NotificationItems int64 `json:"notification_items,omitempty"`
	OpenServices      int64 `json:"open_services"`
	Services          int64 `json:"services,omitempty"`
	RecentChanges     int64 `json:"recent_changes,omitempty"`
}

type ShopSnapshotResponse struct {
	Shop          ShopSnapshotSummary            `json:"shop"`
	Vehicles      []model.ShopVehicle            `json:"vehicles"`
	Lists         []ShopListWithItems            `json:"lists"`
	Notifications []VehicleNotificationWithItems `json:"notifications"`
	Messages      []model.ShopMessages           `json:"messages"`
	Services      []EquipmentServiceResponse     `json:"services"`
	RecentChanges []NotificationChangeWithUsername `json:"recent_changes"`
}

type ShopSnapshotSummary struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Details  *string `json:"details"`
	Role     string  `json:"role"`
	IsAdmin  bool    `json:"is_admin"`
	Settings ShopAggregateSettings `json:"settings"`
	Counts   ShopAggregateCounts   `json:"counts"`
}

type VehicleMaintenanceSnapshotResponse struct {
	Vehicle       model.ShopVehicle             `json:"vehicle"`
	Notifications []VehicleNotificationWithItems `json:"notifications"`
	RecentChanges []NotificationChangeWithUsername `json:"recent_changes"`
	Services      []EquipmentServiceResponse    `json:"services"`
	Counts        ShopAggregateCounts           `json:"counts"`
}

type ShopsBootstrapResponse struct {
	Shops []ShopBootstrapSummary `json:"shops"`
}

type ShopBootstrapSummary struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Details   *string                `json:"details"`
	Role      string                 `json:"role"`
	IsAdmin   bool                   `json:"is_admin"`
	Settings  ShopAggregateSettings  `json:"settings"`
	Counts    ShopAggregateCounts     `json:"counts"`
	Equipment []ShopEquipmentSummary  `json:"equipment"`
}
```

After adding this code, run `gofmt`. If the formatter changes alignment, keep the formatted version.

- [ ] **Step 4: Run DTO tests**

Run:

```bash
go test ./api/response -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit Task 3**

```bash
git add api/response/user_shops_response.go api/response/user_shops_response_test.go
git commit -m "feat(shops): add aggregate response contracts"
```

---

## Task 4: Add Aggregate Package Skeleton And Routing

**Files:**
- Create: `api/shops/aggregates/errors.go`
- Create: `api/shops/aggregates/repository.go`
- Create: `api/shops/aggregates/service.go`
- Create: `api/shops/aggregates/repository_impl.go`
- Create: `api/shops/aggregates/service_impl.go`
- Create: `api/shops/aggregates/handler.go`
- Create: `api/shops/aggregates/route.go`
- Modify: `api/shops/route.go`
- Create: `api/shops/aggregates/handler_test.go`

- [ ] **Step 1: Add handler tests for route registration and auth**

Create `api/shops/aggregates/handler_test.go`:

```go
package aggregates

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type serviceStub struct {
	listsResp *response.ShopListsWithItemsResponse
	err       error
}

func (s serviceStub) GetListsWithItems(context.Context, *bootstrap.User, string) (*response.ShopListsWithItemsResponse, error) {
	return s.listsResp, s.err
}
func (s serviceStub) GetVehicleMaintenanceSnapshot(context.Context, *bootstrap.User, string, SnapshotLimits) (*response.VehicleMaintenanceSnapshotResponse, error) {
	return nil, errors.New("unexpected vehicle snapshot call")
}
func (s serviceStub) GetShopSnapshot(context.Context, *bootstrap.User, string, ShopSnapshotOptions) (*response.ShopSnapshotResponse, error) {
	return nil, errors.New("unexpected shop snapshot call")
}
func (s serviceStub) GetBootstrap(context.Context, *bootstrap.User, BootstrapOptions) (*response.ShopsBootstrapResponse, error) {
	return nil, errors.New("unexpected bootstrap call")
}

func TestGetListsWithItemsRequiresUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1/auth"), serviceStub{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/shops/shop-1/lists-with-items", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusUnauthorized, resp.Code)
}
```

- [ ] **Step 2: Run tests and confirm skeleton failure**

Run:

```bash
go test ./api/shops/aggregates -count=1
```

Expected before adding the package: `stat .../api/shops/aggregates: directory not found` or a compile failure for missing symbols.

- [ ] **Step 3: Add package errors**

Create `api/shops/aggregates/errors.go`:

```go
package aggregates

import "errors"

var (
	ErrUnauthorized        = errors.New("unauthorized")
	ErrAccessDenied       = errors.New("access denied")
	ErrInvalidLimit       = errors.New("invalid limit")
	ErrInvalidInclude     = errors.New("invalid include")
	ErrAggregateUnavailable = errors.New("failed to retrieve shops aggregate")
)
```

- [ ] **Step 4: Add interfaces and option types**

Create `api/shops/aggregates/service.go`:

```go
package aggregates

import (
	"context"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type SnapshotLimits struct {
	ServicesLimit int
	ChangesLimit  int
}

type ShopSnapshotOptions struct {
	Includes      map[string]bool
	MessageLimit  int
	ChangesLimit  int
	ServicesLimit int
}

type BootstrapOptions struct {
	EquipmentLimitPerShop int
	IncludeEmptyEquipment bool
}

type Service interface {
	GetListsWithItems(ctx context.Context, user *bootstrap.User, shopID string) (*response.ShopListsWithItemsResponse, error)
	GetVehicleMaintenanceSnapshot(ctx context.Context, user *bootstrap.User, vehicleID string, limits SnapshotLimits) (*response.VehicleMaintenanceSnapshotResponse, error)
	GetShopSnapshot(ctx context.Context, user *bootstrap.User, shopID string, options ShopSnapshotOptions) (*response.ShopSnapshotResponse, error)
	GetBootstrap(ctx context.Context, user *bootstrap.User, options BootstrapOptions) (*response.ShopsBootstrapResponse, error)
}
```

Create `api/shops/aggregates/repository.go`:

```go
package aggregates

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Repository interface {
	GetListsWithItems(ctx context.Context, user *bootstrap.User, shopID string) ([]response.ShopListWithItems, error)
	GetVehicleByIDForMember(ctx context.Context, user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error)
	GetVehicleNotificationsWithItems(ctx context.Context, vehicleID string) ([]response.VehicleNotificationWithItems, error)
	GetVehicleRecentChanges(ctx context.Context, vehicleID string, limit int) ([]response.NotificationChangeWithUsername, error)
	GetVehicleServices(ctx context.Context, vehicleID string, limit int) ([]response.EquipmentServiceResponse, error)
	GetShopSnapshot(ctx context.Context, user *bootstrap.User, shopID string, options ShopSnapshotOptions) (*response.ShopSnapshotResponse, error)
	GetBootstrap(ctx context.Context, user *bootstrap.User, options BootstrapOptions) ([]response.ShopBootstrapSummary, error)
}
```

- [ ] **Step 5: Add minimal service implementation**

Create `api/shops/aggregates/service_impl.go` with constructor and limit normalization:

```go
package aggregates

import (
	"context"
	"fmt"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
)

const (
	defaultServicesLimit = 50
	defaultChangesLimit = 50
	maxServicesLimit = 200
	maxChangesLimit = 200
	defaultMessageLimit = 20
	maxMessageLimit = 100
	defaultEquipmentLimitPerShop = 50
	maxEquipmentLimitPerShop = 250
)

type ServiceImpl struct {
	repo Repository
	auth shared.ShopAuthorization
}

func NewService(repo Repository, auth shared.ShopAuthorization) *ServiceImpl {
	return &ServiceImpl{repo: repo, auth: auth}
}

func normalizeLimit(value, defaultValue, maxValue int) int {
	if value <= 0 {
		return defaultValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func (s *ServiceImpl) GetListsWithItems(ctx context.Context, user *bootstrap.User, shopID string) (*response.ShopListsWithItemsResponse, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
	if err := s.auth.RequireShopMember(user, shopID); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAccessDenied, err)
	}
	lists, err := s.repo.GetListsWithItems(ctx, user, shopID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAggregateUnavailable, err)
	}
	if lists == nil {
		lists = []response.ShopListWithItems{}
	}
	for i := range lists {
		if lists[i].Items == nil {
			lists[i].Items = []response.ShopListItemWithUsername{}
		}
	}
	return &response.ShopListsWithItemsResponse{Lists: lists}, nil
}
```

Add these concrete service methods so the package compiles. Later endpoint tasks replace the repository behavior with real queries and add success tests before those routes are considered usable:

```go
func (s *ServiceImpl) GetVehicleMaintenanceSnapshot(ctx context.Context, user *bootstrap.User, vehicleID string, limits SnapshotLimits) (*response.VehicleMaintenanceSnapshotResponse, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
	limits.ServicesLimit = normalizeLimit(limits.ServicesLimit, defaultServicesLimit, maxServicesLimit)
	limits.ChangesLimit = normalizeLimit(limits.ChangesLimit, defaultChangesLimit, maxChangesLimit)
	vehicle, err := s.repo.GetVehicleByIDForMember(ctx, user, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAccessDenied, err)
	}
	notifications, err := s.repo.GetVehicleNotificationsWithItems(ctx, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAggregateUnavailable, err)
	}
	changes, err := s.repo.GetVehicleRecentChanges(ctx, vehicleID, limits.ChangesLimit)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAggregateUnavailable, err)
	}
	services, err := s.repo.GetVehicleServices(ctx, vehicleID, limits.ServicesLimit)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAggregateUnavailable, err)
	}
	if notifications == nil {
		notifications = []response.VehicleNotificationWithItems{}
	}
	if changes == nil {
		changes = []response.NotificationChangeWithUsername{}
	}
	if services == nil {
		services = []response.EquipmentServiceResponse{}
	}
	itemCount := int64(0)
	for _, notification := range notifications {
		itemCount += int64(len(notification.Items))
	}
	return &response.VehicleMaintenanceSnapshotResponse{
		Vehicle: *vehicle,
		Notifications: notifications,
		RecentChanges: changes,
		Services: services,
		Counts: response.ShopAggregateCounts{
			Notifications: int64(len(notifications)),
			NotificationItems: itemCount,
			RecentChanges: int64(len(changes)),
			Services: int64(len(services)),
		},
	}, nil
}

func (s *ServiceImpl) GetShopSnapshot(ctx context.Context, user *bootstrap.User, shopID string, options ShopSnapshotOptions) (*response.ShopSnapshotResponse, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
	if err := s.auth.RequireShopMember(user, shopID); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAccessDenied, err)
	}
	options.MessageLimit = normalizeLimit(options.MessageLimit, defaultMessageLimit, maxMessageLimit)
	options.ChangesLimit = normalizeLimit(options.ChangesLimit, defaultChangesLimit, maxChangesLimit)
	options.ServicesLimit = normalizeLimit(options.ServicesLimit, defaultServicesLimit, maxServicesLimit)
	result, err := s.repo.GetShopSnapshot(ctx, user, shopID, options)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAggregateUnavailable, err)
	}
	normalizeShopSnapshot(result)
	return result, nil
}

func (s *ServiceImpl) GetBootstrap(ctx context.Context, user *bootstrap.User, options BootstrapOptions) (*response.ShopsBootstrapResponse, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
	options.EquipmentLimitPerShop = normalizeLimit(options.EquipmentLimitPerShop, defaultEquipmentLimitPerShop, maxEquipmentLimitPerShop)
	shops, err := s.repo.GetBootstrap(ctx, user, options)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAggregateUnavailable, err)
	}
	if shops == nil {
		shops = []response.ShopBootstrapSummary{}
	}
	for i := range shops {
		if shops[i].Equipment == nil {
			shops[i].Equipment = []response.ShopEquipmentSummary{}
		}
	}
	return &response.ShopsBootstrapResponse{Shops: shops}, nil
}

func normalizeShopSnapshot(result *response.ShopSnapshotResponse) {
	if result == nil {
		return
	}
	if result.Vehicles == nil {
		result.Vehicles = []model.ShopVehicle{}
	}
	if result.Lists == nil {
		result.Lists = []response.ShopListWithItems{}
	}
	if result.Notifications == nil {
		result.Notifications = []response.VehicleNotificationWithItems{}
	}
	if result.Messages == nil {
		result.Messages = []model.ShopMessages{}
	}
	if result.Services == nil {
		result.Services = []response.EquipmentServiceResponse{}
	}
	if result.RecentChanges == nil {
		result.RecentChanges = []response.NotificationChangeWithUsername{}
	}
}
```

Add `miltechserver/.gen/miltech_ng/public/model` to the imports in `service_impl.go` for `normalizeShopSnapshot`.

- [ ] **Step 6: Add minimal repository implementation**

Create `api/shops/aggregates/repository_impl.go`:

```go
package aggregates

import (
	"database/sql"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}
```

Add concrete repository methods that return `ErrAggregateUnavailable` until each endpoint task replaces the method body with a real query:

```go
func (repo *RepositoryImpl) GetListsWithItems(context.Context, *bootstrap.User, string) ([]response.ShopListWithItems, error) {
	return nil, ErrAggregateUnavailable
}

func (repo *RepositoryImpl) GetVehicleByIDForMember(context.Context, *bootstrap.User, string) (*model.ShopVehicle, error) {
	return nil, ErrAggregateUnavailable
}

func (repo *RepositoryImpl) GetVehicleNotificationsWithItems(context.Context, string) ([]response.VehicleNotificationWithItems, error) {
	return nil, ErrAggregateUnavailable
}

func (repo *RepositoryImpl) GetVehicleRecentChanges(context.Context, string, int) ([]response.NotificationChangeWithUsername, error) {
	return nil, ErrAggregateUnavailable
}

func (repo *RepositoryImpl) GetVehicleServices(context.Context, string, int) ([]response.EquipmentServiceResponse, error) {
	return nil, ErrAggregateUnavailable
}

func (repo *RepositoryImpl) GetShopSnapshot(context.Context, *bootstrap.User, string, ShopSnapshotOptions) (*response.ShopSnapshotResponse, error) {
	return nil, ErrAggregateUnavailable
}

func (repo *RepositoryImpl) GetBootstrap(context.Context, *bootstrap.User, BootstrapOptions) ([]response.ShopBootstrapSummary, error) {
	return nil, ErrAggregateUnavailable
}
```

Add imports for `context`, `.gen/.../model`, `api/response`, and `bootstrap` in `repository_impl.go`.

- [ ] **Step 7: Add handlers and route registration**

Create `api/shops/aggregates/handler.go` with `getUser`, `getListsWithItems`, and error mapping:

```go
package aggregates

import (
	"errors"
	"net/http"

	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func getUser(c *gin.Context) (*bootstrap.User, bool) {
	ctxUser, ok := c.Get("user")
	user, userOK := ctxUser.(*bootstrap.User)
	return user, ok && userOK && user != nil
}

func (handler Handler) getListsWithItems(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	result, err := handler.service.GetListsWithItems(c.Request.Context(), user, c.Param("shop_id"))
	if err != nil {
		writeAggregateError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{
		Status: http.StatusOK,
		Message: "Shop lists with items retrieved successfully",
		Data: result,
	})
}

func writeAggregateError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
	case errors.Is(err, ErrAccessDenied):
		c.JSON(http.StatusForbidden, gin.H{"message": "access denied"})
	case errors.Is(err, ErrInvalidLimit), errors.Is(err, ErrInvalidInclude):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, response.StandardResponse{
			Status: http.StatusInternalServerError,
			Message: ErrAggregateUnavailable.Error(),
			Data: nil,
		})
	}
}
```

Create `api/shops/aggregates/route.go`:

```go
package aggregates

import (
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	handler := Handler{service: service}
	router.GET("/shops/:shop_id/lists-with-items", gzip.Gzip(gzip.DefaultCompression), handler.getListsWithItems)
}
```

- [ ] **Step 8: Wire package into Shops route registration**

In `api/shops/route.go`, import `miltechserver/api/shops/aggregates`, create `aggregatesRepository := aggregates.NewRepository(deps.DB)`, create `aggregatesService := aggregates.NewService(aggregatesRepository, authorization)`, and call `aggregates.RegisterRoutes(router, aggregatesService)` before `core.RegisterRoutes(router, coreService)`.

The relevant registration block should be ordered like this:

```go
aggregates.RegisterRoutes(router, aggregatesService)
core.RegisterRoutes(router, coreService)
settings.RegisterRoutes(router, settingsService)
```

This order keeps `/shops/bootstrap` and other static aggregate paths from being swallowed by the existing `GET /shops/:shop_id` route.

- [ ] **Step 9: Verify package tests**

Run:

```bash
go test ./api/shops/aggregates -count=1
```

Expected: PASS.

- [ ] **Step 10: Commit Task 4**

```bash
git add api/shops/aggregates api/shops/route.go
git commit -m "feat(shops): scaffold aggregate endpoints"
```

---

## Task 5: Implement `GET /shops/:shop_id/lists-with-items`

**Files:**
- Modify: `api/shops/aggregates/repository_impl.go`
- Modify: `api/shops/aggregates/service_impl.go`
- Modify: `api/shops/aggregates/handler.go`
- Modify: `api/shops/aggregates/handler_test.go`
- Create: `tests/shops/shops_aggregate_lists_test.go`

- [ ] **Step 1: Add integration tests**

Create `tests/shops/shops_aggregate_lists_test.go`:

```go
package shops_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListsWithItemsAggregate(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "other-user")
	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Aggregate Lists")
	hiddenShopID := createShop(t, router, "other-user", "Hidden Lists")
	listID := createList(t, router, "user-1", shopID)
	hiddenListID := createList(t, router, "other-user", hiddenShopID)
	createListItem(t, router, "user-1", listID, "111111111", "Visible Item")
	createListItem(t, router, "other-user", hiddenListID, "222222222", "Hidden Item")

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/lists-with-items", nil, "user-1")
	require.Equal(t, http.StatusOK, resp.Code)

	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	lists := payload["lists"].([]interface{})
	require.Len(t, lists, 1)
	list := lists[0].(map[string]interface{})
	require.Equal(t, listID, list["id"])
	items := list["items"].([]interface{})
	require.Len(t, items, 1)
	require.Equal(t, "111111111", items[0].(map[string]interface{})["niin"])
}

func TestListsWithItemsAggregateRejectsNonMember(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "user-2")
	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Private Lists")
	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/lists-with-items", nil, "user-2")
	require.NotEqual(t, http.StatusOK, resp.Code)
}
```

Add these helpers to `tests/shops/helpers_test.go` if they are not already present:

```go
func createList(t *testing.T, router *gin.Engine, userID string, shopID string) string {
	t.Helper()
	body := map[string]interface{}{
		"shop_id": shopID,
		"description": "Test list",
	}
	resp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/lists", body, userID)
	require.Equal(t, http.StatusCreated, resp.Code)
	data := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	listID, ok := data["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, listID)
	return listID
}

func createListItem(t *testing.T, router *gin.Engine, userID string, listID string, niin string, nomenclature string) string {
	t.Helper()
	body := map[string]interface{}{
		"list_id": listID,
		"niin": niin,
		"nomenclature": nomenclature,
		"quantity": 1,
		"unit_of_measure": "ea",
	}
	resp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/lists/items", body, userID)
	require.Equal(t, http.StatusCreated, resp.Code)
	data := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	itemID, ok := data["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, itemID)
	return itemID
}
```

- [ ] **Step 2: Run failing integration tests**

Run:

```bash
go test ./tests/shops -run 'TestListsWithItemsAggregate' -count=1
```

Expected before implementation: server error or route failure.

- [ ] **Step 3: Implement repository query**

In `api/shops/aggregates/repository_impl.go`, implement `GetListsWithItems` with a single set-based query:

```sql
SELECT
	l.id, l.shop_id, l.created_by, creator.username, l.description, l.created_at, l.updated_at,
	i.id, i.list_id, i.niin, i.nomenclature, i.quantity, i.added_by, added.username,
	i.created_at, i.updated_at, i.nickname, i.unit_of_measure
FROM shop_lists l
INNER JOIN shop_members sm ON sm.shop_id = l.shop_id AND sm.user_id = $2
LEFT JOIN users creator ON creator.uid = l.created_by
LEFT JOIN shop_list_items i ON i.list_id = l.id
LEFT JOIN users added ON added.uid = i.added_by
WHERE l.shop_id = $1
ORDER BY l.created_at DESC, i.created_at ASC, i.id ASC
```

Map rows into `[]response.ShopListWithItems`, preserving empty lists with `items: []`.

- [ ] **Step 4: Complete handler success mapping**

In `api/shops/aggregates/handler.go`, return:

```go
c.JSON(http.StatusOK, response.StandardResponse{
	Status: http.StatusOK,
	Message: "Shop lists with items retrieved successfully",
	Data: result,
})
```

Keep `401` for missing user and generic error mapping consistent with existing Shops handlers.

- [ ] **Step 5: Verify tests**

Run:

```bash
go test ./api/shops/aggregates ./tests/shops -run 'TestListsWithItems|TestListsWithItemsAggregate' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit Task 5**

```bash
git add api/shops/aggregates tests/shops/shops_aggregate_lists_test.go tests/shops/helpers_test.go
git commit -m "feat(shops): add lists with items aggregate"
```

---

## Task 6: Implement `GET /shops/vehicles/:vehicle_id/maintenance-snapshot`

**Files:**
- Modify: `api/shops/aggregates/route.go`
- Modify: `api/shops/aggregates/handler.go`
- Modify: `api/shops/aggregates/service_impl.go`
- Modify: `api/shops/aggregates/repository_impl.go`
- Create: `tests/shops/shops_aggregate_vehicle_maintenance_test.go`

- [ ] **Step 1: Add integration tests**

Create `tests/shops/shops_aggregate_vehicle_maintenance_test.go`:

```go
package shops_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestVehicleMaintenanceSnapshot(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Maintenance Snapshot")
	vehicleID := createVehicle(t, router, "user-1", shopID)
	notificationID := createNotification(t, router, "user-1", shopID, vehicleID, "PM")
	itemBody := map[string]any{"notification_id": notificationID, "niin": "123456789", "nomenclature": "Filter", "quantity": 1}
	addItemResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items", itemBody, "user-1")
	require.Equal(t, http.StatusCreated, addItemResp.Code)

	serviceDate := time.Now().AddDate(0, 0, 5)
	createEquipmentService(t, router, "user-1", shopID, vehicleID, "", "Scheduled service", &serviceDate, false)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/"+vehicleID+"/maintenance-snapshot", nil, "user-1")
	require.Equal(t, http.StatusOK, resp.Code)
	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	require.NotNil(t, payload["vehicle"])
	require.Len(t, payload["notifications"].([]interface{}), 1)
	require.Len(t, payload["services"].([]interface{}), 1)
	counts := payload["counts"].(map[string]interface{})
	require.Equal(t, float64(1), counts["notifications"])
	require.Equal(t, float64(1), counts["notification_items"])
	require.Equal(t, float64(1), counts["services"])
}
```

- [ ] **Step 2: Register route**

In `api/shops/aggregates/route.go`, add:

```go
router.GET("/shops/vehicles/:vehicle_id/maintenance-snapshot", gzip.Gzip(gzip.DefaultCompression), handler.getVehicleMaintenanceSnapshot)
```

- [ ] **Step 3: Implement query methods**

Implement:

- `GetVehicleByIDForMember`: join `shop_vehicle` to `shop_members` with `shop_members.user_id = $2`.
- `GetVehicleNotificationsWithItems`: reuse the two-query pattern from `api/shops/vehicles/notifications/repository_impl.go`, but accept `ctx`.
- `GetVehicleRecentChanges`: query `shop_vehicle_notification_changes` by `vehicle_id`, ordered by `changed_at DESC`, limited by normalized `changes_limit`.
- `GetVehicleServices`: query `equipment_services` by `equipment_id`, ordered by `service_date DESC NULLS LAST, created_at DESC`, limited by normalized `services_limit`, and map to `response.EquipmentServiceResponse`.

- [ ] **Step 4: Implement service method**

In `service_impl.go`, normalize limits, load the vehicle first for authorization, then load snapshot sections. Sequential queries are acceptable for the first implementation because each query is set-based and bounded. If later profiling shows latency, parallelize independent reads with `errgroup.WithContext`.

- [ ] **Step 5: Implement handler**

Parse `services_limit` and `changes_limit` as integers. Invalid non-integer values return `400`. Missing values use service defaults.

- [ ] **Step 6: Verify focused tests**

Run:

```bash
go test ./tests/shops -run TestVehicleMaintenanceSnapshot -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit Task 6**

```bash
git add api/shops/aggregates tests/shops/shops_aggregate_vehicle_maintenance_test.go
git commit -m "feat(shops): add vehicle maintenance snapshot"
```

---

## Task 7: Implement `GET /shops/:shop_id/snapshot`

**Files:**
- Modify: `api/shops/aggregates/route.go`
- Modify: `api/shops/aggregates/handler.go`
- Modify: `api/shops/aggregates/service_impl.go`
- Modify: `api/shops/aggregates/repository_impl.go`
- Create: `tests/shops/shops_aggregate_shop_snapshot_test.go`

- [ ] **Step 1: Add integration tests**

Create `tests/shops/shops_aggregate_shop_snapshot_test.go`:

```go
package shops_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestShopSnapshotDefaultIncludes(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Snapshot Shop")
	vehicleID := createVehicle(t, router, "user-1", shopID)
	listID := createList(t, router, "user-1", shopID)
	createListItem(t, router, "user-1", listID, "123456789", "Part")
	notificationID := createNotification(t, router, "user-1", shopID, vehicleID, "PM")
	require.NotEmpty(t, notificationID)
	serviceDate := time.Now().AddDate(0, 0, 2)
	createEquipmentService(t, router, "user-1", shopID, vehicleID, "", "Service", &serviceDate, false)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/snapshot", nil, "user-1")
	require.Equal(t, http.StatusOK, resp.Code)
	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	shop := payload["shop"].(map[string]interface{})
	require.Equal(t, shopID, shop["id"])
	require.Len(t, payload["vehicles"].([]interface{}), 1)
	require.Len(t, payload["lists"].([]interface{}), 1)
	require.Len(t, payload["notifications"].([]interface{}), 1)
	require.Len(t, payload["services"].([]interface{}), 1)
}

func TestShopSnapshotIncludeMessages(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Message Snapshot")
	_ = createMessage(t, router, "user-1", shopID, "hello")

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/snapshot?include=messages&message_limit=1", nil, "user-1")
	require.Equal(t, http.StatusOK, resp.Code)
	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	require.Len(t, payload["messages"].([]interface{}), 1)
	require.Empty(t, payload["vehicles"].([]interface{}))
}
```

- [ ] **Step 2: Register route**

In `api/shops/aggregates/route.go`, add:

```go
router.GET("/shops/:shop_id/snapshot", gzip.Gzip(gzip.DefaultCompression), handler.getShopSnapshot)
```

Register this route before any generic `"/shops/:shop_id"` aggregate route if such a route is ever added. Gin route matching must keep `/snapshot` distinct from existing core `GET /shops/:shop_id`.

- [ ] **Step 3: Implement include parsing**

In `handler.go`, parse `include`. If missing, set `vehicles`, `lists`, `notifications`, and `services` to true. If present, accept only:

```text
vehicles,lists,notifications,messages,services,changes
```

Unknown include values return `400` with `"invalid include"`.

- [ ] **Step 4: Implement repository method**

Use membership-rooted queries and bounded sections:

- Shop summary and counts from one query rooted in `shop_members.user_id`.
- Vehicles query ordered by `shop_vehicle.save_time DESC`.
- Lists with items by reusing the repository method from Task 5.
- Notifications with items by shop with a batched item lookup.
- Messages query ordered by `created_at DESC`, limited by `message_limit`.
- Services query ordered by `service_date ASC NULLS LAST, created_at DESC`, limited by `services_limit`.
- Recent changes query ordered by `changed_at DESC`, limited by `changes_limit`.

All sections must return empty arrays when omitted or empty. Omitted sections due to `include` are empty arrays, not `null`.

- [ ] **Step 5: Verify focused tests**

Run:

```bash
go test ./tests/shops -run 'TestShopSnapshot' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit Task 7**

```bash
git add api/shops/aggregates tests/shops/shops_aggregate_shop_snapshot_test.go
git commit -m "feat(shops): add shop snapshot endpoint"
```

---

## Task 8: Implement `GET /shops/bootstrap`

**Files:**
- Modify: `api/shops/aggregates/route.go`
- Modify: `api/shops/aggregates/handler.go`
- Modify: `api/shops/aggregates/service_impl.go`
- Modify: `api/shops/aggregates/repository_impl.go`
- Create: `tests/shops/shops_aggregate_bootstrap_test.go`

- [ ] **Step 1: Add integration tests**

Create `tests/shops/shops_aggregate_bootstrap_test.go`:

```go
package shops_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShopsBootstrapReturnsOnlyAuthenticatedUserShops(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "user-2")
	router := newTestRouter(t)

	visibleShopID := createShop(t, router, "user-1", "Visible Bootstrap")
	hiddenShopID := createShop(t, router, "user-2", "Hidden Bootstrap")
	visibleEquipmentID := createVehicle(t, router, "user-1", visibleShopID)
	_ = createVehicle(t, router, "user-2", hiddenShopID)
	_, err := testDB.Exec(`UPDATE shop_vehicle SET admin='A1', model='M1', serial='S1', niin='000000001' WHERE id=$1`, visibleEquipmentID)
	require.NoError(t, err)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/bootstrap", nil, "user-1")
	require.Equal(t, http.StatusOK, resp.Code)
	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	shops := payload["shops"].([]interface{})
	require.Len(t, shops, 1)
	shop := shops[0].(map[string]interface{})
	require.Equal(t, visibleShopID, shop["id"])
	require.Len(t, shop["equipment"].([]interface{}), 1)
}
```

- [ ] **Step 2: Register route**

In `api/shops/aggregates/route.go`, add:

```go
router.GET("/shops/bootstrap", gzip.Gzip(gzip.DefaultCompression), handler.getBootstrap)
```

Keep this route registered before `core.RegisterRoutes` generic `/shops/:shop_id` is considered for future route changes.

The required route order remains:

```go
aggregates.RegisterRoutes(router, aggregatesService)
core.RegisterRoutes(router, coreService)
```

- [ ] **Step 3: Implement repository query**

Use a membership-rooted query for shops, counts, role, settings, and equipment summaries. To avoid unbounded equipment in bootstrap, use one of these two approaches:

1. Preferred first implementation: query user shops and counts, then query equipment for all returned shop IDs with `ROW_NUMBER() OVER (PARTITION BY shop_id ORDER BY save_time DESC, id DESC) <= $equipmentLimitPerShop`, then group in Go.
2. Alternative if Jet becomes awkward: use raw SQL with bound parameters and scan into small row structs.

Do not reuse `GET /shops/equipment/overview` internally because bootstrap has per-shop equipment limits and counts.

- [ ] **Step 4: Implement handler options**

Parse:

- `equipment_limit_per_shop`: default `50`, max `250`.
- Always return `equipment: []` for shops with no returned equipment rows. Do not add a query option for omitting empty arrays because it would create another response variant without reducing query cost.

- [ ] **Step 5: Verify focused tests**

Run:

```bash
go test ./tests/shops -run TestShopsBootstrapReturnsOnlyAuthenticatedUserShops -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit Task 8**

```bash
git add api/shops/aggregates tests/shops/shops_aggregate_bootstrap_test.go
git commit -m "feat(shops): add bootstrap aggregate endpoint"
```

---

## Task 9: Add Compatibility And Route-Conflict Guards

**Files:**
- Create: `tests/shops/shops_aggregate_compatibility_test.go`
- Modify: `api/shops/aggregates/route.go` only if route ordering tests expose a conflict.
- Modify: `api/shops/route.go` only if aggregate route registration is not already before `core.RegisterRoutes`.

- [ ] **Step 1: Add route compatibility tests**

Create `tests/shops/shops_aggregate_compatibility_test.go`:

```go
package shops_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAggregateRoutesDoNotBreakExistingShopRoutes(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Compatibility Shop")

	legacyShopsResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops", nil, "user-1")
	require.Equal(t, http.StatusOK, legacyShopsResp.Code)

	legacyShopDetailResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID, nil, "user-1")
	require.Equal(t, http.StatusOK, legacyShopDetailResp.Code)

	legacyUserDataResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/user-data", nil, "user-1")
	require.Equal(t, http.StatusOK, legacyUserDataResp.Code)

	legacyOverviewResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/equipment/overview", nil, "user-1")
	require.Equal(t, http.StatusOK, legacyOverviewResp.Code)

	bootstrapResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/bootstrap", nil, "user-1")
	require.Equal(t, http.StatusOK, bootstrapResp.Code)

	snapshotResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/snapshot", nil, "user-1")
	require.Equal(t, http.StatusOK, snapshotResp.Code)
}

func TestAggregateRoutesRequireAuthentication(t *testing.T) {
	router := newTestRouter(t)

	paths := []string{
		"/api/v1/auth/shops/bootstrap",
		"/api/v1/auth/shops/shop-1/lists-with-items",
		"/api/v1/auth/shops/shop-1/snapshot",
		"/api/v1/auth/shops/vehicles/vehicle-1/maintenance-snapshot",
	}

	for _, path := range paths {
		resp := doJSONRequest(t, router, http.MethodGet, path, nil, "")
		require.Equal(t, http.StatusUnauthorized, resp.Code, path)
	}
}
```

- [ ] **Step 2: Fix route order if the compatibility test fails**

If `GET /api/v1/auth/shops/bootstrap` is handled by the legacy `GET /shops/:shop_id` route, ensure `api/shops/route.go` registers aggregate routes before core routes:

```go
aggregates.RegisterRoutes(router, aggregatesService)
core.RegisterRoutes(router, coreService)
```

- [ ] **Step 3: Run focused aggregate compatibility tests**

Run:

```bash
go test ./tests/shops -run 'TestAggregateRoutes' -count=1
```

Expected: PASS.

- [ ] **Step 4: Run legacy Shops smoke tests**

Run:

```bash
go test ./tests/shops -run 'TestShopCoreLifecycle|TestListsAndItems|TestVehicleNotificationsAndChanges|TestShopSettings|TestShopEquipmentOverviewEndpoint' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit Task 9**

```bash
git add tests/shops/shops_aggregate_compatibility_test.go api/shops/aggregates/route.go api/shops/route.go
git commit -m "test(shops): guard aggregate route compatibility"
```

---

## Task 10: Performance Validation And Evidence

**Files:**
- Create: `tests/shops/shops_aggregate_performance_test.go`
- Modify: `docs/api/shops_api_efficiency_mobile.md`

- [ ] **Step 1: Add opt-in representative performance test**

Create `tests/shops/shops_aggregate_performance_test.go` guarded by `SHOP_AGGREGATE_PERF=1`. The fixture should create:

- one user;
- 25 shops;
- 250 equipment rows;
- 250 lists;
- 2,500 list items;
- 250 notifications;
- 500 notification items;
- 500 equipment services.

The test must log:

- `EXPLAIN (ANALYZE, BUFFERS)` for bootstrap core query;
- p50 and p95 for `/shops/bootstrap`;
- p50 and p95 for `/shops/:shop_id/snapshot`;
- uncompressed and gzip response sizes.

- [ ] **Step 2: Run normal tests without opt-in perf**

Run:

```bash
go test ./tests/shops -run 'TestShopsBootstrap|TestShopSnapshot|TestVehicleMaintenanceSnapshot|TestListsWithItemsAggregate' -count=1
```

Expected: PASS.

- [ ] **Step 3: Run opt-in perf test only when a representative database is acceptable**

Run:

```bash
SHOP_AGGREGATE_PERF=1 go test ./tests/shops -run TestShopsAggregatePerformance -count=1 -v
```

Expected: PASS and logs include p50, p95, sizes, and `EXPLAIN` output. If the test misses a performance target, do not add indexes inside this task; capture the plan output and open a separate schema-index design.

- [ ] **Step 4: Commit Task 10**

```bash
git add tests/shops/shops_aggregate_performance_test.go
git commit -m "test(shops): add aggregate performance validation"
```

---

## Task 11: Mobile API Documentation And ADR

**Files:**
- Create: `docs/api/shops_api_efficiency_mobile.md`
- Modify: `docs/api/shop_equipment_overview_mobile.md`
- Modify: `docs/project_notes/decisions.md`

- [ ] **Step 1: Write mobile-facing docs**

Create `docs/api/shops_api_efficiency_mobile.md` with these sections:

- API summary.
- Authentication.
- Backward compatibility guarantee.
- `GET /shops/:shop_id/lists-with-items`.
- `GET /shops/vehicles/:vehicle_id/maintenance-snapshot`.
- `GET /shops/:shop_id/snapshot`.
- `GET /shops/bootstrap`.
- Error responses.
- Compression notes.
- Client migration guidance.
- Performance notes.

Use JSON examples only. Do not include mobile code snippets.

- [ ] **Step 2: Cross-link existing equipment overview docs**

In `docs/api/shop_equipment_overview_mobile.md`, add a short "Related endpoints" paragraph that points to `docs/api/shops_api_efficiency_mobile.md` and clearly states that `GET /shops/equipment/overview` remains unchanged.

- [ ] **Step 3: Add ADR**

Append to `docs/project_notes/decisions.md`:

```markdown
### ADR-016: Add Backward-Compatible Shops Aggregate Read Endpoints (2026-07-03)

**Context:**
- Existing Shops clients can load complex screens by chaining multiple narrow endpoints.
- The existing `GET /shops/equipment/overview` aggregate proved the set-based additive endpoint pattern.
- Some current query shapes needed internal fixes without response-contract changes.

**Decision:**
- Keep all legacy endpoints active and backward compatible.
- Add aggregate read endpoints for list trees, vehicle maintenance snapshots, shop snapshots, and Shops bootstrap.
- Keep high-cardinality sections bounded and gzip aggregate payloads when requested.
- Require representative `EXPLAIN (ANALYZE, BUFFERS)` evidence before adding new indexes.

**Consequences:**
- New clients can reduce round trips substantially.
- Old clients continue using existing endpoints.
- Aggregate endpoint DTOs must be maintained separately from generated Jet models to avoid accidental response growth.
```

- [ ] **Step 4: Validate docs formatting**

Run:

```bash
rg --pcre2 -n '^```(?!json|text|bash$)' docs/api/shops_api_efficiency_mobile.md docs/api/shop_equipment_overview_mobile.md
```

Expected: no output.

- [ ] **Step 5: Commit Task 11**

```bash
git add docs/api/shops_api_efficiency_mobile.md docs/api/shop_equipment_overview_mobile.md docs/project_notes/decisions.md
git commit -m "docs(shops): document aggregate API endpoints"
```

---

## Final Verification

- [ ] **Step 1: Run focused Shops and equipment-service integration tests**

```bash
go test ./tests/shops ./tests/equipment_services -count=1
```

Expected: PASS.

- [ ] **Step 2: Run touched package tests**

```bash
go test ./api/shops/... ./api/equipment_services/... ./api/response -count=1
```

Expected: PASS.

- [ ] **Step 3: Run broader Go tests if time allows**

```bash
go test ./... -count=1
```

Expected: PASS, or document unrelated baseline failures with exact packages and error snippets.

- [ ] **Step 4: Review compatibility**

Manually confirm:

- existing route paths remain registered;
- existing response DTOs were not renamed;
- existing mobile docs still describe `GET /shops/equipment/overview` unchanged;
- new endpoints are additive;
- no generated Jet files changed;
- no migration was added without measured evidence.

---

## Recommended Execution Order

1. Task 1: Fix equipment-service duplicate joins.
2. Task 2: Rewrite user shop stats query.
3. Task 3: Add DTOs.
4. Task 4: Add aggregate skeleton and routing.
5. Task 5: Add lists-with-items endpoint.
6. Task 6: Add vehicle maintenance snapshot.
7. Task 7: Add shop snapshot.
8. Task 8: Add bootstrap endpoint.
9. Task 9: Add route compatibility guards.
10. Task 10: Add performance evidence.
11. Task 11: Document mobile contracts and ADR.

If scope needs to be reduced, implement Tasks 1, 2, 5, 6, 10, and 11 first. Those deliver the most immediate efficiency gain with the least broad contract surface.
