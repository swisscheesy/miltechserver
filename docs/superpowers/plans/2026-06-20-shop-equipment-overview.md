# Shop Equipment Overview Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `GET /api/v1/auth/shops/equipment/overview`, returning every shop the authenticated user belongs to with compact equipment summaries in one current, unbounded response.

**Architecture:** Extend `api/shops/core` with a context-aware, membership-rooted Jet query that performs one `shops -> shop_members -> shop_vehicle` database call and maps rows into dedicated nested DTOs. The service normalizes empty slices and hides repository details behind a generic error; the handler uses endpoint-scoped gzip and the existing `StandardResponse` envelope. No schema migration, cache, pagination, endpoint deadline, or rate limiter is included.

**Tech Stack:** Go 1.23, Gin 1.10.1, Jet 2.13.0, PostgreSQL, Firebase-authenticated Gin context, `gin-contrib/gzip` 1.1.0, Testify.

**Design reference:** `docs/superpowers/specs/2026-06-20-shop-equipment-overview-design.md`

---

## File map

- Modify `api/response/user_shops_response.go`: add stable response DTOs and Jet aliases.
- Create `api/response/user_shops_response_test.go`: lock JSON names and empty-array behavior.
- Modify `api/shops/core/repository.go`: add the context-aware aggregate contract.
- Modify `api/shops/core/repository_impl.go`: execute the single Jet query.
- Create `api/shops/core/errors.go`: define the generic endpoint error.
- Modify `api/shops/core/service.go` and `service_impl.go`: expose and implement the use case.
- Create `api/shops/core/service_impl_test.go`: verify context, normalization, and errors.
- Modify `api/shops/core/handler.go` and `route.go`: add HTTP handling and endpoint-scoped gzip.
- Create `api/shops/core/handler_test.go`: prove generic `500` behavior at the HTTP boundary.
- Modify `go.mod` and `go.sum`: add gzip 1.1.0 without upgrading Gin.
- Create `tests/shops/shops_equipment_overview_test.go`: repository and endpoint integration coverage.
- Create `tests/shops/shops_equipment_overview_performance_test.go`: opt-in representative validation.
- Modify `docs/project_notes/decisions.md`: record ADR-015.

Do not edit `.gen/` files. Do not create a migration unless a new design review is triggered by measured evidence.

---

### Task 1: Define the response contract

**Files:**
- Modify: `api/response/user_shops_response.go`
- Create: `api/response/user_shops_response_test.go`

- [ ] **Step 1: Write the failing DTO tests**

Create `api/response/user_shops_response_test.go`:

```go
package response

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShopEquipmentOverviewResponseJSON(t *testing.T) {
	details := "Maintenance section"
	result := ShopEquipmentOverviewResponse{
		Shops: []ShopEquipmentOverview{{
			ID: "shop-1", Name: "Alpha Shop", Details: &details, Role: "member",
			EquipmentCount: 1,
			Equipment: []ShopEquipmentSummary{{
				ID: "equipment-1", Admin: "A123", Model: "M1097",
				Serial: "SER-1", Niin: "012345678",
			}},
		}},
	}

	payload, err := json.Marshal(result)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"shops":[{
			"id":"shop-1","name":"Alpha Shop","details":"Maintenance section",
			"role":"member","equipment_count":1,
			"equipment":[{"id":"equipment-1","admin":"A123","model":"M1097","serial":"SER-1","niin":"012345678"}]
		}]
	}`, string(payload))
}

func TestShopEquipmentOverviewResponseEmptyArrays(t *testing.T) {
	result := ShopEquipmentOverviewResponse{Shops: []ShopEquipmentOverview{{
		ID: "shop-1", Name: "Empty Shop", Role: "admin",
		Equipment: []ShopEquipmentSummary{},
	}}}

	payload, err := json.Marshal(result)
	require.NoError(t, err)
	require.Contains(t, string(payload), `"equipment":[]`)
	require.NotContains(t, string(payload), `"equipment":null`)
}
```

- [ ] **Step 2: Prove the tests fail**

Run:

```bash
go test ./api/response -run 'TestShopEquipmentOverviewResponse' -count=1
```

Expected: build failure because the overview DTOs do not exist.

- [ ] **Step 3: Add the DTOs**

Append to `api/response/user_shops_response.go`:

```go
type ShopEquipmentOverviewResponse struct {
	Shops []ShopEquipmentOverview `json:"shops"`
}

type ShopEquipmentOverview struct {
	ID             string                 `sql:"primary_key" alias:"shops.id" json:"id"`
	Name           string                 `alias:"shops.name" json:"name"`
	Details        *string                `alias:"shops.details" json:"details"`
	Role           string                 `alias:"shop_members.role" json:"role"`
	EquipmentCount int                    `json:"equipment_count"`
	Equipment      []ShopEquipmentSummary `alias:"shop_vehicle" json:"equipment"`
}

type ShopEquipmentSummary struct {
	ID     string `sql:"primary_key" alias:"id" json:"id"`
	Admin  string `alias:"admin" json:"admin"`
	Model  string `alias:"model" json:"model"`
	Serial string `alias:"serial" json:"serial"`
	Niin   string `alias:"niin" json:"niin"`
}
```

- [ ] **Step 4: Prove the DTO tests pass**

Run the Step 2 command. Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add api/response/user_shops_response.go api/response/user_shops_response_test.go
git commit -m "feat(shops): define equipment overview response"
```

---

### Task 2: Implement the single-query repository

**Files:**
- Modify: `api/shops/core/repository.go`
- Modify: `api/shops/core/repository_impl.go`
- Create: `tests/shops/shops_equipment_overview_test.go`

- [ ] **Step 1: Write failing repository integration tests**

Create `tests/shops/shops_equipment_overview_test.go`:

```go
package shops_test

import (
	"context"
	"errors"
	"testing"
	"time"

	shopcore "miltechserver/api/shops/core"
	"miltechserver/bootstrap"

	"github.com/stretchr/testify/require"
)

func TestShopEquipmentOverviewRepository(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "overview-user")
	ensureUser(t, testDB, "other-user")
	router := newTestRouter(t)

	newerShopID := createShop(t, router, "overview-user", "Newer Shop")
	olderShopID := createShop(t, router, "overview-user", "Older Shop")
	hiddenShopID := createShop(t, router, "other-user", "Hidden Shop")
	newerEquipmentID := createVehicle(t, router, "overview-user", newerShopID)
	olderEquipmentID := createVehicle(t, router, "overview-user", newerShopID)
	_ = createVehicle(t, router, "other-user", hiddenShopID)

	newerTime := time.Date(2026, time.June, 20, 12, 0, 0, 0, time.UTC)
	olderTime := newerTime.Add(-time.Hour)
	_, err := testDB.Exec(`UPDATE shops SET created_at = $1 WHERE id = $2`, newerTime, newerShopID)
	require.NoError(t, err)
	_, err = testDB.Exec(`UPDATE shops SET created_at = $1 WHERE id = $2`, olderTime, olderShopID)
	require.NoError(t, err)
	_, err = testDB.Exec(`UPDATE shop_vehicle SET admin=$1, model=$2, serial=$3, niin=$4, save_time=$5 WHERE id=$6`,
		"NEW", "M1097", "SER-NEW", "000000001", newerTime, newerEquipmentID)
	require.NoError(t, err)
	_, err = testDB.Exec(`UPDATE shop_vehicle SET admin=$1, model=$2, serial=$3, niin=$4, save_time=$5 WHERE id=$6`,
		"OLD", "M998", "SER-OLD", "000000002", olderTime, olderEquipmentID)
	require.NoError(t, err)

	repository := shopcore.NewRepository(testDB, nil, &bootstrap.Env{})
	shops, err := repository.GetShopEquipmentOverview(context.Background(), &bootstrap.User{UserID: "overview-user"})
	require.NoError(t, err)
	require.Len(t, shops, 2)
	require.Equal(t, newerShopID, shops[0].ID)
	require.Equal(t, olderShopID, shops[1].ID)
	require.Len(t, shops[0].Equipment, 2)
	require.Equal(t, newerEquipmentID, shops[0].Equipment[0].ID)
	require.Equal(t, olderEquipmentID, shops[0].Equipment[1].ID)
	require.Equal(t, "NEW", shops[0].Equipment[0].Admin)
	require.Empty(t, shops[1].Equipment)
	for _, shop := range shops {
		require.NotEqual(t, hiddenShopID, shop.ID)
	}
}

func TestShopEquipmentOverviewRepositoryHonorsCanceledContext(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "overview-user")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repository := shopcore.NewRepository(testDB, nil, &bootstrap.Env{})
	_, err := repository.GetShopEquipmentOverview(ctx, &bootstrap.User{UserID: "overview-user"})
	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled), err.Error())
}
```

- [ ] **Step 2: Prove the repository tests fail**

```bash
go test ./tests/shops -run 'TestShopEquipmentOverviewRepository' -count=1
```

Expected: build failure because the repository method does not exist.

- [ ] **Step 3: Extend the repository interface**

Import `context` in `api/shops/core/repository.go` and add:

```go
GetShopEquipmentOverview(ctx context.Context, user *bootstrap.User) ([]response.ShopEquipmentOverview, error)
```

- [ ] **Step 4: Implement the query**

Add to `api/shops/core/repository_impl.go`:

```go
func (repo *RepositoryImpl) GetShopEquipmentOverview(
	ctx context.Context,
	user *bootstrap.User,
) ([]response.ShopEquipmentOverview, error) {
	queryStarted := time.Now()
	stmt := SELECT(
		Shops.ID, Shops.Name, Shops.Details, ShopMembers.Role,
		ShopVehicle.ID, ShopVehicle.Admin, ShopVehicle.Model,
		ShopVehicle.Serial, ShopVehicle.Niin,
	).FROM(
		Shops.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(Shops.ID)).
			LEFT_JOIN(ShopVehicle, ShopVehicle.ShopID.EQ(Shops.ID)),
	).WHERE(
		ShopMembers.UserID.EQ(String(user.UserID)),
	).ORDER_BY(
		Shops.CreatedAt.DESC(),
		ShopVehicle.SaveTime.DESC(),
		ShopVehicle.ID.DESC(),
	)

	shops := make([]response.ShopEquipmentOverview, 0)
	if err := stmt.QueryContext(ctx, repo.db, &shops); err != nil {
		return nil, fmt.Errorf("failed to query shop equipment overview: %w", err)
	}

	equipmentCount := 0
	for i := range shops {
		equipmentCount += len(shops[i].Equipment)
	}
	slog.Info("Shop equipment overview query completed",
		"user_id", user.UserID,
		"shop_count", len(shops),
		"equipment_count", equipmentCount,
		"duration_ms", time.Since(queryStarted).Milliseconds(),
	)
	return shops, nil
}
```

`repository_impl.go` already imports all required packages. `save_time` is used for ordering but is not selected into the compact DTO.

- [ ] **Step 5: Prove repository and existing core tests pass**

```bash
go test ./tests/shops -run 'TestShopEquipmentOverviewRepository|TestShopCreateGetUpdate|TestUserDataWithShops|TestShopVehicles' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add api/shops/core/repository.go api/shops/core/repository_impl.go tests/shops/shops_equipment_overview_test.go
git commit -m "feat(shops): query equipment overview in one call"
```

---

### Task 3: Add service behavior and generic errors

**Files:**
- Create: `api/shops/core/errors.go`
- Modify: `api/shops/core/service.go`
- Modify: `api/shops/core/service_impl.go`
- Create: `api/shops/core/service_impl_test.go`

- [ ] **Step 1: Write failing service tests**

Create `api/shops/core/service_impl_test.go`:

```go
package core

import (
	"context"
	"errors"
	"testing"

	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/stretchr/testify/require"
)

type overviewRepositoryStub struct {
	Repository
	getOverview func(context.Context, *bootstrap.User) ([]response.ShopEquipmentOverview, error)
}

func (stub overviewRepositoryStub) GetShopEquipmentOverview(ctx context.Context, user *bootstrap.User) ([]response.ShopEquipmentOverview, error) {
	return stub.getOverview(ctx, user)
}

func TestGetShopEquipmentOverview(t *testing.T) {
	requestContext := context.Background()
	repository := overviewRepositoryStub{getOverview: func(ctx context.Context, user *bootstrap.User) ([]response.ShopEquipmentOverview, error) {
		require.Equal(t, requestContext, ctx)
		require.Equal(t, "user-1", user.UserID)
		return []response.ShopEquipmentOverview{
			{ID: "shop-1", Equipment: nil},
			{ID: "shop-2", Equipment: []response.ShopEquipmentSummary{{ID: "equipment-1"}}},
		}, nil
	}}
	service := NewService(repository, nil)

	result, err := service.GetShopEquipmentOverview(requestContext, &bootstrap.User{UserID: "user-1"})
	require.NoError(t, err)
	require.NotNil(t, result.Shops[0].Equipment)
	require.Empty(t, result.Shops[0].Equipment)
	require.Equal(t, 0, result.Shops[0].EquipmentCount)
	require.Equal(t, 1, result.Shops[1].EquipmentCount)
}

func TestGetShopEquipmentOverviewRejectsMissingUser(t *testing.T) {
	service := NewService(overviewRepositoryStub{}, nil)
	result, err := service.GetShopEquipmentOverview(context.Background(), nil)
	require.Nil(t, result)
	require.EqualError(t, err, "unauthorized user")
}

func TestGetShopEquipmentOverviewHidesRepositoryError(t *testing.T) {
	repository := overviewRepositoryStub{getOverview: func(context.Context, *bootstrap.User) ([]response.ShopEquipmentOverview, error) {
		return nil, errors.New("database host and query details")
	}}
	service := NewService(repository, nil)
	result, err := service.GetShopEquipmentOverview(context.Background(), &bootstrap.User{UserID: "user-1"})
	require.Nil(t, result)
	require.ErrorIs(t, err, ErrShopEquipmentOverviewUnavailable)
	require.NotContains(t, err.Error(), "database host")
}
```

- [ ] **Step 2: Prove the tests fail**

```bash
go test ./api/shops/core -run 'TestGetShopEquipmentOverview' -count=1
```

Expected: build failure because the service method and sentinel do not exist.

- [ ] **Step 3: Define the public-safe error**

Create `api/shops/core/errors.go`:

```go
package core

import "errors"

var ErrShopEquipmentOverviewUnavailable = errors.New("failed to retrieve shop equipment overview")
```

- [ ] **Step 4: Extend and implement the service**

Import `context` in `service.go` and add to `ShopService`:

```go
GetShopEquipmentOverview(ctx context.Context, user *bootstrap.User) (*response.ShopEquipmentOverviewResponse, error)
```

Import `context` in `service_impl.go` and add:

```go
func (service *ServiceImpl) GetShopEquipmentOverview(ctx context.Context, user *bootstrap.User) (*response.ShopEquipmentOverviewResponse, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}
	shops, err := service.repo.GetShopEquipmentOverview(ctx, user)
	if err != nil {
		slog.Error("Failed to retrieve shop equipment overview", "error", err, "user_id", user.UserID)
		return nil, ErrShopEquipmentOverviewUnavailable
	}
	if shops == nil {
		shops = make([]response.ShopEquipmentOverview, 0)
	}
	for i := range shops {
		if shops[i].Equipment == nil {
			shops[i].Equipment = make([]response.ShopEquipmentSummary, 0)
		}
		shops[i].EquipmentCount = len(shops[i].Equipment)
	}
	return &response.ShopEquipmentOverviewResponse{Shops: shops}, nil
}
```

- [ ] **Step 5: Prove service tests pass**

Run the Step 2 command. Expected: all three tests PASS.

- [ ] **Step 6: Commit**

```bash
git add api/shops/core/errors.go api/shops/core/service.go api/shops/core/service_impl.go api/shops/core/service_impl_test.go
git commit -m "feat(shops): add equipment overview service"
```

---

### Task 4: Expose the authenticated endpoint with scoped gzip

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Modify: `api/shops/core/handler.go`
- Modify: `api/shops/core/route.go`
- Create: `api/shops/core/handler_test.go`
- Modify: `tests/shops/shops_equipment_overview_test.go`

- [ ] **Step 1: Append failing endpoint tests**

Add `bytes`, `compress/gzip`, `encoding/json`, `io`, `net/http`, and `net/http/httptest` to `tests/shops/shops_equipment_overview_test.go`, then append:

```go
func TestShopEquipmentOverviewEndpoint(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "overview-user")
	ensureUser(t, testDB, "other-user")
	router := newTestRouter(t)

	visibleShopID := createShop(t, router, "overview-user", "Visible Shop")
	emptyShopID := createShop(t, router, "overview-user", "Empty Shop")
	hiddenShopID := createShop(t, router, "other-user", "Hidden Shop")
	visibleEquipmentID := createVehicle(t, router, "overview-user", visibleShopID)
	_ = createVehicle(t, router, "other-user", hiddenShopID)
	_, err := testDB.Exec(`UPDATE shop_vehicle SET admin='A123', model='M1097', serial='SER-1', niin='012345678' WHERE id=$1`, visibleEquipmentID)
	require.NoError(t, err)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/equipment/overview", nil, "overview-user")
	require.Equal(t, http.StatusOK, resp.Code)
	payload := decodeStandardResponse(t, resp.Body)
	data := decodeMap(t, payload.Data)
	shops := data["shops"].([]interface{})
	require.Len(t, shops, 2)

	shopsByID := make(map[string]map[string]interface{}, len(shops))
	for _, rawShop := range shops {
		shop := rawShop.(map[string]interface{})
		shopsByID[shop["id"].(string)] = shop
	}
	require.NotContains(t, shopsByID, hiddenShopID)
	require.Contains(t, shopsByID, visibleShopID)
	require.Contains(t, shopsByID, emptyShopID)
	require.Equal(t, float64(1), shopsByID[visibleShopID]["equipment_count"])
	require.Equal(t, float64(0), shopsByID[emptyShopID]["equipment_count"])
	require.Empty(t, shopsByID[emptyShopID]["equipment"].([]interface{}))

	equipment := shopsByID[visibleShopID]["equipment"].([]interface{})[0].(map[string]interface{})
	require.Equal(t, map[string]interface{}{
		"id": visibleEquipmentID, "admin": "A123", "model": "M1097",
		"serial": "SER-1", "niin": "012345678",
	}, equipment)
}

func TestShopEquipmentOverviewEndpointRequiresAuthentication(t *testing.T) {
	router := newTestRouter(t)
	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/equipment/overview", nil, "")
	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestShopEquipmentOverviewEndpointGzipIsScoped(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "overview-user")
	router := newTestRouter(t)
	_ = createShop(t, router, "overview-user", "Compressed Shop")

	req, err := http.NewRequest(http.MethodGet, "/api/v1/auth/shops/equipment/overview", nil)
	require.NoError(t, err)
	req.Header.Set("X-User-ID", "overview-user")
	req.Header.Set("Accept-Encoding", "gzip")
	compressed := httptest.NewRecorder()
	router.ServeHTTP(compressed, req)
	require.Equal(t, http.StatusOK, compressed.Code)
	require.Equal(t, "gzip", compressed.Header().Get("Content-Encoding"))

	reader, err := gzip.NewReader(bytes.NewReader(compressed.Body.Bytes()))
	require.NoError(t, err)
	decompressed, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	var decoded standardResponse
	require.NoError(t, json.Unmarshal(decompressed, &decoded))
	normalReq, err := http.NewRequest(http.MethodGet, "/api/v1/auth/shops/equipment/overview", nil)
	require.NoError(t, err)
	normalReq.Header.Set("X-User-ID", "overview-user")
	normal := httptest.NewRecorder()
	router.ServeHTTP(normal, normalReq)
	require.Empty(t, normal.Header().Get("Content-Encoding"))
	require.JSONEq(t, normal.Body.String(), string(decompressed))

	shopsReq, err := http.NewRequest(http.MethodGet, "/api/v1/auth/shops", nil)
	require.NoError(t, err)
	shopsReq.Header.Set("X-User-ID", "overview-user")
	shopsReq.Header.Set("Accept-Encoding", "gzip")
	uncompressed := httptest.NewRecorder()
	router.ServeHTTP(uncompressed, shopsReq)
	require.Empty(t, uncompressed.Header().Get("Content-Encoding"))
}
```

Create `api/shops/core/handler_test.go` in the same step:

```go
package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"miltechserver/api/middleware"
	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type overviewServiceStub struct {
	ShopService
	result *response.ShopEquipmentOverviewResponse
	err    error
}

func (stub overviewServiceStub) GetShopEquipmentOverview(context.Context, *bootstrap.User) (*response.ShopEquipmentOverviewResponse, error) {
	return stub.result, stub.err
}

func TestGetShopEquipmentOverviewReturnsGenericServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler)
	router.Use(func(c *gin.Context) {
		c.Set("user", &bootstrap.User{UserID: "user-1"})
		c.Next()
	})
	handler := Handler{service: overviewServiceStub{err: ErrShopEquipmentOverviewUnavailable}}
	router.GET("/shops/equipment/overview", handler.GetShopEquipmentOverview)

	req := httptest.NewRequest(http.MethodGet, "/shops/equipment/overview", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusInternalServerError, resp.Code)
	require.Contains(t, resp.Body.String(), ErrShopEquipmentOverviewUnavailable.Error())
	require.NotContains(t, resp.Body.String(), "database")
}
```

- [ ] **Step 2: Prove endpoint tests fail**

```bash
go test ./tests/shops -run 'TestShopEquipmentOverviewEndpoint' -count=1
```

Expected: `404` for the unregistered endpoint.

- [ ] **Step 3: Add the compatible gzip dependency**

The latest gzip 1.2.6 requires Go 1.25 and Gin 1.12. Pin the compatible version and verify Gin is not upgraded:

```bash
go get github.com/gin-contrib/gzip@v1.1.0
go list -m github.com/gin-contrib/gzip github.com/gin-gonic/gin
```

Expected modules: gzip `v1.1.0`; Gin `v1.10.1`.

- [ ] **Step 4: Add the handler**

Add `net/http` and `time` to `api/shops/core/handler.go`, then add:

```go
func (handler *Handler) GetShopEquipmentOverview(c *gin.Context) {
	startedAt := time.Now()
	ctxUser, ok := c.Get("user")
	user, userOK := ctxUser.(*bootstrap.User)
	if !ok || !userOK || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	overview, err := handler.service.GetShopEquipmentOverview(c.Request.Context(), user)
	if err != nil {
		_ = c.Error(err)
		return
	}

	equipmentCount := 0
	for i := range overview.Shops {
		equipmentCount += overview.Shops[i].EquipmentCount
	}
	c.JSON(http.StatusOK, response.StandardResponse{
		Status: http.StatusOK,
		Message: "Shop equipment overview retrieved successfully",
		Data: overview,
	})
	slog.Info("Shop equipment overview request completed",
		"user_id", user.UserID,
		"shop_count", len(overview.Shops),
		"equipment_count", equipmentCount,
		"duration_ms", time.Since(startedAt).Milliseconds(),
	)
}
```

The service returns only the generic sentinel on repository failure, so existing error middleware cannot expose SQL details for this endpoint.

- [ ] **Step 5: Register endpoint-only gzip**

Update `api/shops/core/route.go`:

```go
import (
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, service ShopService) {
	handler := Handler{service: service}
	router.POST("/shops", handler.CreateShop)
	router.GET("/shops", handler.GetUserShops)
	router.GET("/shops/user-data", handler.GetUserDataWithShops)
	router.GET("/shops/equipment/overview", gzip.Gzip(gzip.DefaultCompression), handler.GetShopEquipmentOverview)
	router.GET("/shops/:shop_id", handler.GetShopByID)
	router.PUT("/shops/:shop_id", handler.UpdateShop)
	router.DELETE("/shops/:shop_id", handler.DeleteShop)
}
```

- [ ] **Step 6: Prove endpoint and Shops tests pass**

```bash
go test ./tests/shops -run 'TestShopEquipmentOverviewEndpoint' -count=1
go test ./tests/shops/... -count=1
```

Expected: PASS. Existing routes remain uncompressed and unchanged.

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum api/shops/core/handler.go api/shops/core/handler_test.go api/shops/core/route.go tests/shops/shops_equipment_overview_test.go
git commit -m "feat(shops): expose equipment overview endpoint"
```

---

### Task 5: Validate representative performance and record ADR-015

**Files:**
- Create: `tests/shops/shops_equipment_overview_performance_test.go`
- Modify: `docs/project_notes/decisions.md`

- [ ] **Step 1: Add the opt-in performance test**

Create `tests/shops/shops_equipment_overview_performance_test.go`:

```go
package shops_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestShopEquipmentOverviewPerformance(t *testing.T) {
	if os.Getenv("SHOP_OVERVIEW_PERF") != "1" {
		t.Skip("set SHOP_OVERVIEW_PERF=1 to run the representative performance test")
	}
	clearShopTables(t, testDB)
	t.Cleanup(func() { clearShopTables(t, testDB) })
	ensureUser(t, testDB, "overview-perf-user")
	_, err := testDB.Exec(`
		INSERT INTO shops (id, name, details, created_by, created_at, updated_at)
		SELECT 'overview-perf-shop-' || n, 'Performance Shop ' || n,
		       'Performance fixture', 'overview-perf-user',
		       now() - (n * interval '1 second'), now()
		FROM generate_series(1, 100) AS n;

		INSERT INTO shop_members (id, shop_id, user_id, role, joined_at)
		SELECT 'overview-perf-member-' || n, 'overview-perf-shop-' || n,
		       'overview-perf-user', 'member', now()
		FROM generate_series(1, 100) AS n;

		INSERT INTO shop_vehicle (
			id, creator_id, niin, admin, model, serial, uoc,
			mileage, hours, comment, save_time, last_updated, shop_id
		)
		SELECT 'overview-perf-equipment-' || n, 'overview-perf-user',
		       lpad(n::text, 9, '0'), 'ADMIN-' || n, 'MODEL-' || (n % 50),
		       'SERIAL-' || n, 'UNK', 0, 0, '',
		       now() - (n * interval '1 millisecond'), now(),
		       'overview-perf-shop-' || (((n - 1) % 100) + 1)
		FROM generate_series(1, 25000) AS n;

		ANALYZE shops;
		ANALYZE shop_members;
		ANALYZE shop_vehicle;
	`)
	require.NoError(t, err)

	planRows, err := testDB.Query(`
		EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
		SELECT s.id, s.name, s.details, sm.role,
		       v.id, v.admin, v.model, v.serial, v.niin
		FROM shops s
		INNER JOIN shop_members sm ON sm.shop_id = s.id
		LEFT JOIN shop_vehicle v ON v.shop_id = s.id
		WHERE sm.user_id = 'overview-perf-user'
		ORDER BY s.created_at DESC, v.save_time DESC, v.id DESC
	`)
	require.NoError(t, err)
	for planRows.Next() {
		var line string
		require.NoError(t, planRows.Scan(&line))
		t.Log(line)
	}
	require.NoError(t, planRows.Err())
	require.NoError(t, planRows.Close())

	router := newTestRouter(t)
	const measuredRequests = 100
	durations := make([]time.Duration, 0, measuredRequests)
	responseSize := 0
	for requestNumber := 0; requestNumber <= measuredRequests; requestNumber++ {
		startedAt := time.Now()
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/equipment/overview", nil, "overview-perf-user")
		duration := time.Since(startedAt)
		require.Equal(t, http.StatusOK, resp.Code)
		if requestNumber == 0 {
			continue
		}
		durations = append(durations, duration)
		responseSize = resp.Body.Len()
	}

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	p50 := durations[49]
	p95 := durations[94]
	compressedDurations := make([]time.Duration, 0, measuredRequests)
	compressedSize := 0
	for requestNumber := 0; requestNumber <= measuredRequests; requestNumber++ {
		compressedReq, requestErr := http.NewRequest(http.MethodGet, "/api/v1/auth/shops/equipment/overview", nil)
		require.NoError(t, requestErr)
		compressedReq.Header.Set("X-User-ID", "overview-perf-user")
		compressedReq.Header.Set("Accept-Encoding", "gzip")
		compressedResp := httptest.NewRecorder()
		startedAt := time.Now()
		router.ServeHTTP(compressedResp, compressedReq)
		duration := time.Since(startedAt)
		require.Equal(t, http.StatusOK, compressedResp.Code)
		require.Equal(t, "gzip", compressedResp.Header().Get("Content-Encoding"))
		if requestNumber == 0 {
			continue
		}
		compressedDurations = append(compressedDurations, duration)
		compressedSize = compressedResp.Body.Len()
	}
	sort.Slice(compressedDurations, func(i, j int) bool { return compressedDurations[i] < compressedDurations[j] })
	compressedP95 := compressedDurations[94]

	t.Logf("warm-cache p50=%s p95=%s compressed_p95=%s uncompressed_bytes=%d compressed_bytes=%d",
		p50, p95, compressedP95, responseSize, compressedSize)
	require.Less(t, p95, time.Second, fmt.Sprintf("p95 target missed: %s", p95))
	require.Less(t, compressedP95, time.Second, fmt.Sprintf("compressed p95 target missed: %s", compressedP95))
}
```

- [ ] **Step 2: Verify normal execution skips it**

```bash
go test ./tests/shops -run 'TestShopEquipmentOverviewPerformance' -count=1 -v
```

Expected: PASS with one SKIP.

- [ ] **Step 3: Run the representative test**

```bash
/usr/bin/time -l env SHOP_OVERVIEW_PERF=1 go test ./tests/shops -run 'TestShopEquipmentOverviewPerformance' -count=1 -v -memprofile=/tmp/shop-equipment-overview.mem
go tool pprof -top -alloc_space /tmp/shop-equipment-overview.mem
go tool pprof -top -inuse_space /tmp/shop-equipment-overview.mem
```

Expected: PASS with p50, p95 below one second, both payload sizes, query plan, allocation profile, in-use profile, and maximum resident set size logged. If it fails, stop and return to design review before adding an index, cache, pagination, or concurrency.

- [ ] **Step 4: Capture and inspect the query plan**

Inspect the `EXPLAIN` output logged by Step 3. The test executes this exact statement before its cleanup runs:

```sql
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT s.id, s.name, s.details, sm.role,
       v.id, v.admin, v.model, v.serial, v.niin
FROM shops s
INNER JOIN shop_members sm ON sm.shop_id = s.id
LEFT JOIN shop_vehicle v ON v.shop_id = s.id
WHERE sm.user_id = 'overview-perf-user'
ORDER BY s.created_at DESC, v.save_time DESC, v.id DESC;
```

Inspect actual versus estimated rows, sort method and memory, buffer reads, and membership/equipment access paths. A sequential scan is not automatically defective when most fixture rows are returned.

- [ ] **Step 5: Append ADR-015**

Append to `docs/project_notes/decisions.md`:

```markdown
### ADR-015: Unbounded Shop Equipment Overview (2026-06-20)

**Context:**
- The overview client needs every shop the authenticated user belongs to and compact equipment identity fields in one current response.
- The accepted representative load is 100 shops and 25,000 equipment records, with warm-cache p95 below one second.
- Existing per-shop vehicle retrieval would create an N+1 query pattern.

**Decision:**
- Add `GET /api/v1/auth/shops/equipment/overview` under `api/shops/core`.
- Use one membership-filtered Jet query with a left join to equipment and request-context cancellation.
- Return compact DTOs, preserve empty shops, and apply endpoint-scoped gzip.
- Keep the response unbounded and uncached, with no endpoint deadline or endpoint-specific rate limiter.
- Keep the existing schema because representative evidence did not justify another index.

**Consequences:**
- The endpoint performs one database round trip and limits rows through membership in the query.
- Memory and bandwidth grow linearly with equipment count; gzip reduces transfer size but not DTO allocation.
- If workloads exceed the accepted bound or p95 target, revisit the transport contract before speculative indexing or caching.
```

If measured evidence contradicts the last decision bullet, stop and amend the design instead of writing a false ADR.

- [ ] **Step 6: Commit**

```bash
git add tests/shops/shops_equipment_overview_performance_test.go docs/project_notes/decisions.md
git commit -m "test(shops): validate equipment overview performance"
```

---

### Task 6: Final verification

**Files:** Verify every file changed in Tasks 1-5.

- [ ] **Step 1: Format modified Go files**

```bash
gofmt -w api/response/user_shops_response.go api/response/user_shops_response_test.go api/shops/core/errors.go api/shops/core/repository.go api/shops/core/repository_impl.go api/shops/core/service.go api/shops/core/service_impl.go api/shops/core/service_impl_test.go api/shops/core/handler.go api/shops/core/handler_test.go api/shops/core/route.go tests/shops/shops_equipment_overview_test.go tests/shops/shops_equipment_overview_performance_test.go
```

Expected: exit 0. Review formatting before staging.

- [ ] **Step 2: Run focused verification**

```bash
go test ./api/response ./api/shops/core ./tests/shops/... -count=1
```

Expected: PASS; the opt-in performance test is skipped.

- [ ] **Step 3: Run the full suite**

```bash
go test ./... -count=1
```

Expected: PASS. If unrelated baseline failures occur, report exact packages and distinguish them from the focused green suite.

- [ ] **Step 4: Verify dependencies and scope**

```bash
go list -m github.com/gin-contrib/gzip github.com/gin-gonic/gin
git diff --check
git status --short
```

Expected: gzip `v1.1.0`, Gin `v1.10.1`, no whitespace errors, no migration or `.gen` changes, and the pre-existing `AGENTS.md` modification excluded from feature commits.

- [ ] **Step 5: Inspect the commit stack**

```bash
git log --oneline -6
```

Expected feature commits newest-first:

```text
test(shops): validate equipment overview performance
feat(shops): expose equipment overview endpoint
feat(shops): add equipment overview service
feat(shops): query equipment overview in one call
feat(shops): define equipment overview response
docs(shops): plan equipment overview endpoint
```

Do not create an empty completion commit.
