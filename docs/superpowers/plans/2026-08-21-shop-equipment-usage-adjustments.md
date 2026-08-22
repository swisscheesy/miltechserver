# Shop Equipment Usage Adjustments Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let every current Shop member atomically add or subtract mileage and/or hours, including subtraction to exactly zero, through a backward-compatible server endpoint and one global Add/Subtract control in Flutter.

**Architecture:** Add a dedicated membership-authorized PATCH endpoint whose one operation applies to both optional positive magnitudes and whose repository locks, calculates, validates, updates, and returns the authoritative equipment row in one short PostgreSQL transaction. Keep the legacy absolute PUT contract unchanged for valid callers, then move Flutter to the new persisted-response contract and preserve `null` versus explicit zero in its domain and Drift schema. Correct the still-unreleased Drift schema v6 in place and extend `from5To6`; do not create schema v7.

**Tech Stack:** Go 1.23, Gin 1.10, go-jet v2.13, `database/sql`, PostgreSQL, Testify; Flutter 3.35+, Dart 3.2+, flutter_bloc, Dio/DioService, Drift/SQLite, json_annotation, mocktail, flutter_test.

**Spec:** `docs/superpowers/specs/2026-08-21-shop-equipment-usage-adjustments-design.md`

## Repository and dependency map

- Server repository: `/Users/swisscheese/projects/miltechserver`, primary branch `main`.
- Mobile repository: `/Users/swisscheese/projects/miltech`, primary branch `master`.
- Server Task 1 establishes the new route and persisted response.
- Server Tasks 2-3 establish bounds, concurrency, legacy hardening, and database invariants.
- Task 4 publishes the client contract and completes the server gate.
- Mobile Task 5 fixes zero semantics in schema v6 before new response handling is added.
- Mobile Tasks 6-8 add transport, state/local persistence, and the global UI control.
- Task 9 is the cross-repository release gate. The server must be deployable before the client can ship.

## Global Constraints

- Read the approved spec before editing. Its contract and acceptance criteria are normative.
- Before implementation, create isolated server and mobile worktrees with `superpowers:using-git-worktrees`; do not reuse either dirty primary checkout.
- Record `rtk git status --short --branch`, `rtk git rev-parse HEAD`, and `rtk git worktree list` in each repository before editing.
- Preserve the existing user-owned server change in `bootstrap/env.go` and mobile change in `lib/_data/services/dio_service.dart`; neither file is part of this feature.
- Preserve `PUT /api/v1/auth/shops/vehicles` for valid existing clients. Never reinterpret its tracked fields as deltas.
- The new route is exactly `PATCH /api/v1/auth/shops/vehicles/:vehicle_id/usage`.
- The request fields are exactly `operation`, `mileage_adjustment`, and `hours_adjustment`.
- One operation applies to both supplied fields. Missing or blank operation defaults to `add`.
- Accept only unsigned integer magnitudes from 0 through 10,000, with at least one value greater than zero.
- Permit a resulting usage of zero. Reject a result below zero or above `2147483647` without changing either field.
- All current Shop members may adjust usage. Metadata, delete, and other permissions do not change.
- Return the persisted `model.ShopVehicle` in `response.StandardResponse.Data`; HTTP status alone is not success.
- Keep PostgreSQL row-lock transactions short and free of authorization queries, logging I/O, and external calls.
- Keep Flutter `schemaVersion => 6`. Modify fresh-v6 schema and `from5To6`; do not add v7, `from6To7`, or `schema_v7.dart`.
- Preserve historical Drift snapshots v1-v5. Regenerate current v6 and generated helpers; never hand-edit generated Drift or json_serializable output.
- Authenticated Shops remain server-authoritative. Do not restore a logged-in local write, optimistic success, legacy PUT fallback, or automatic retry.
- Logged-out visible Shops retain local-only behavior and use the same add/subtract semantics.
- Do not add usage audit history, idempotency persistence, dependencies, or unrelated refactors.
- Use TDD: add a focused failing test, observe the expected failure, add the minimum implementation, rerun focused tests, then commit.
- Use one narrow Conventional Commit per task after its focused verification passes.
- No push, merge, deployment, production migration, app release, or physical-device claim is authorized by executing this plan.

---

### Task 1: Add the persisted-response usage adjustment endpoint

**Repository:** `/Users/swisscheese/projects/miltechserver`

**Files:**
- Modify: `api/request/shops_request.go`
- Create: `api/shops/vehicles/usage.go`
- Create: `api/shops/vehicles/errors.go`
- Modify: `api/shops/vehicles/route.go`
- Modify: `api/shops/vehicles/handler.go`
- Modify: `api/shops/vehicles/service.go`
- Modify: `api/shops/vehicles/service_impl.go`
- Modify: `api/shops/vehicles/repository.go`
- Modify: `api/shops/vehicles/repository_impl.go`
- Modify: `tests/shops/shops_vehicles_test.go`

**Interfaces:**
- Produces: `request.AdjustShopVehicleUsageRequest` with `Operation string`, `MileageAdjustment *int32`, and `HoursAdjustment *int32`.
- Produces: `UsageAdjustmentOperation`, constants `UsageOperationAdd` and `UsageOperationSubtract`, and `UsageAdjustment`.
- Produces: `Service.AdjustShopVehicleUsage(context.Context, *bootstrap.User, UsageAdjustment) (*model.ShopVehicle, error)`.
- Produces: `Repository.AdjustShopVehicleUsage(context.Context, UsageAdjustment) (*model.ShopVehicle, error)`.
- Produces: `PATCH /shops/vehicles/:vehicle_id/usage` under the existing `/api/v1/auth` group.
- Returns: `response.StandardResponse{Status: 200, Message: "Equipment usage adjusted successfully", Data: *updatedVehicle}`.

- [ ] **Step 1: Establish the server baseline in an isolated worktree**

Run:

```bash
rtk git status --short --branch
rtk git rev-parse HEAD
rtk git worktree list
rtk go test ./tests/shops -run 'TestVehicleCRUD|TestVehicleUsageUpdateRequiresShopMembership' -count=1
```

Expected: the primary checkout's unrelated `bootstrap/env.go` modification is absent from the task worktree, the merged member-usage tests pass, and the worktree starts from server `main`. Record any nonzero baseline exactly.

- [ ] **Step 2: Write failing endpoint integration tests**

Extend `tests/shops/shops_vehicles_test.go` with a setup helper that creates an owner, member, outsider, Shop, membership, and equipment with base mileage 100 and hours 50. Add these two tests:

```go
func TestAdjustVehicleUsageAddsAndReturnsPersistedVehicle(t *testing.T) {
	fixture := newVehicleUsageFixture(t, 100, 50)

	resp := doJSONRequest(
		t,
		fixture.router,
		http.MethodPatch,
		"/api/v1/auth/shops/vehicles/"+fixture.vehicleID+"/usage",
		map[string]interface{}{
			"operation":            "add",
			"mileage_adjustment": 10,
			"hours_adjustment":   5,
		},
		fixture.memberID,
	)

	require.Equal(t, http.StatusOK, resp.Code)
	standard := decodeStandardResponse(t, resp.Body)
	require.Equal(t, http.StatusOK, standard.Status)
	require.Equal(t, "Equipment usage adjusted successfully", standard.Message)
	vehicle := decodeMap(t, standard.Data)
	require.Equal(t, fixture.vehicleID, vehicle["id"])
	require.Equal(t, float64(110), vehicle["tracked_mileage"])
	require.Equal(t, float64(55), vehicle["tracked_hours"])
}

func TestAdjustVehicleUsageSubtractsForOrdinaryMember(t *testing.T) {
	fixture := newVehicleUsageFixture(t, 100, 50)
	seedTrackedUsage(t, testDB, fixture.vehicleID, 120, 70)

	resp := doJSONRequest(
		t,
		fixture.router,
		http.MethodPatch,
		"/api/v1/auth/shops/vehicles/"+fixture.vehicleID+"/usage",
		map[string]interface{}{
			"operation":            "subtract",
			"mileage_adjustment": 20,
			"hours_adjustment":   15,
		},
		fixture.memberID,
	)

	require.Equal(t, http.StatusOK, resp.Code)
	vehicle := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	require.Equal(t, float64(100), vehicle["tracked_mileage"])
	require.Equal(t, float64(55), vehicle["tracked_hours"])
}
```

Implement `newVehicleUsageFixture` with the exact user IDs `owner`, `member`,
and `outsider`, using the existing `ensureUser`, `createShop`,
`createInviteCode`, and join-request helpers. Implement `seedTrackedUsage` with
one parameterized test-only SQL update.

- [ ] **Step 3: Run the tests and verify the route is absent**

Run:

```bash
rtk go test ./tests/shops -run 'TestAdjustVehicleUsageAddsAndReturnsPersistedVehicle|TestAdjustVehicleUsageSubtractsForOrdinaryMember' -count=1
```

Expected: FAIL with HTTP 404 because the PATCH route is not registered.

- [ ] **Step 4: Add the request and internal usage types**

Add to `api/request/shops_request.go`:

```go
type AdjustShopVehicleUsageRequest struct {
	Operation            string `json:"operation"`
	MileageAdjustment *int32 `json:"mileage_adjustment"`
	HoursAdjustment   *int32 `json:"hours_adjustment"`
}
```

Create `api/shops/vehicles/usage.go`:

```go
package vehicles

import "time"

type UsageAdjustmentOperation string

const (
	UsageOperationAdd      UsageAdjustmentOperation = "add"
	UsageOperationSubtract UsageAdjustmentOperation = "subtract"
	maximumUsageAdjustment int32                    = 10_000
)

type UsageAdjustment struct {
	VehicleID          string
	Operation          UsageAdjustmentOperation
	MileageAdjustment *int32
	HoursAdjustment   *int32
	LastUpdated        time.Time
}
```

Create `api/shops/vehicles/errors.go` with sentinels used by `errors.Is`:

```go
package vehicles

import "errors"

var (
	ErrInvalidUsageAdjustment = errors.New("invalid usage adjustment")
	ErrUsageOutOfRange        = errors.New("usage adjustment would move tracked usage outside the supported range")
)
```

- [ ] **Step 5: Add route, strict decoding, response, and error mapping**

Register:

```go
router.PATCH("/shops/vehicles/:vehicle_id/usage", handler.AdjustShopVehicleUsage)
```

In `handler.go`, use `json.Decoder.DisallowUnknownFields`, require exactly one JSON value, and map expected errors without `c.Error`:

```go
func (handler *Handler) AdjustShopVehicleUsage(c *gin.Context) {
	user, ok := authenticatedUser(c)
	if !ok {
		return
	}

	vehicleID := c.Param("vehicle_id")
	if vehicleID == "" {
		writeVehicleError(c, fmt.Errorf("%w: vehicle_id is required", ErrInvalidUsageAdjustment))
		return
	}

	var req request.AdjustShopVehicleUsageRequest
	if err := decodeStrictJSON(c.Request.Body, &req); err != nil {
		writeVehicleError(c, fmt.Errorf("%w: malformed request", ErrInvalidUsageAdjustment))
		return
	}

	updated, err := handler.service.AdjustShopVehicleUsage(
		c.Request.Context(),
		user,
		UsageAdjustment{
			VehicleID:          vehicleID,
			Operation:          UsageAdjustmentOperation(req.Operation),
			MileageAdjustment: req.MileageAdjustment,
			HoursAdjustment:   req.HoursAdjustment,
		},
	)
	if err != nil {
		writeVehicleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Equipment usage adjusted successfully",
		Data:    *updated,
	})
}
```

`writeVehicleError` maps `ErrInvalidUsageAdjustment` to 400, `shared.ErrShopAccessDenied` to 403, `shared.ErrVehicleNotFound` to 404, `ErrUsageOutOfRange` to 409, and logs then returns `response.InternalErrorResponseMessage()` for 500. `decodeStrictJSON` rejects unknown fields and trailing JSON values. `authenticatedUser` must reject missing, nil, and wrong-type context values with 401.

- [ ] **Step 6: Extend service and repository interfaces**

Add the exact context-aware signatures:

```go
AdjustShopVehicleUsage(
	ctx context.Context,
	user *bootstrap.User,
	adjustment UsageAdjustment,
) (*model.ShopVehicle, error)
```

The service implementation must:

1. reject a nil user;
2. normalize blank operation to `UsageOperationAdd`;
3. validate the operation and magnitudes;
4. load the equipment and map a missing row to `shared.ErrVehicleNotFound`;
5. call `service.auth.RequireShopMember(user, currentVehicle.ShopID)`;
6. set `LastUpdated = time.Now().UTC()`; and
7. call the repository.

Use this validation shape:

```go
func normalizeUsageAdjustment(adjustment UsageAdjustment) (UsageAdjustment, error) {
	adjustment.Operation = UsageAdjustmentOperation(
		strings.TrimSpace(string(adjustment.Operation)),
	)
	if adjustment.Operation == "" {
		adjustment.Operation = UsageOperationAdd
	}
	if adjustment.Operation != UsageOperationAdd && adjustment.Operation != UsageOperationSubtract {
		return UsageAdjustment{}, fmt.Errorf("%w: operation must be add or subtract", ErrInvalidUsageAdjustment)
	}

	values := []*int32{adjustment.MileageAdjustment, adjustment.HoursAdjustment}
	hasPositive := false
	for _, value := range values {
		if value == nil {
			continue
		}
		if *value < 0 || *value > maximumUsageAdjustment {
			return UsageAdjustment{}, fmt.Errorf("%w: adjustments must be between 0 and 10000", ErrInvalidUsageAdjustment)
		}
		hasPositive = hasPositive || *value > 0
	}
	if !hasPositive {
		return UsageAdjustment{}, fmt.Errorf("%w: at least one adjustment must be greater than zero", ErrInvalidUsageAdjustment)
	}
	return adjustment, nil
}
```

- [ ] **Step 7: Implement the short atomic Jet transaction**

Use `BeginTx(ctx, nil)`, Jet
`SELECT(ShopVehicle.AllColumns).FOR(UPDATE())`, `QueryContext`, and Jet
`UPDATE().RETURNING(ShopVehicle.AllColumns)`. Compute with `int64` before
converting to `int32`:

```go
func applyUsageAdjustment(current int32, magnitude int32, operation UsageAdjustmentOperation) (int32, error) {
	result := int64(current)
	if operation == UsageOperationSubtract {
		result -= int64(magnitude)
	} else {
		result += int64(magnitude)
	}
	if result < 0 || result > math.MaxInt32 {
		return 0, ErrUsageOutOfRange
	}
	return int32(result), nil
}
```

For each supplied field, choose current tracked usage when its pointer is non-nil and otherwise choose the base field. If either calculation fails, return before issuing UPDATE so both fields remain unchanged. Build SET clauses only for supplied fields, always include `LastUpdated`, query the returned row, commit, and return it. Treat both `sql.ErrNoRows` and `qrm.ErrNoRows` as `shared.ErrVehicleNotFound`.

- [ ] **Step 8: Run focused tests and commit**

Run:

```bash
rtk gofmt -w api/request/shops_request.go api/shops/vehicles tests/shops/shops_vehicles_test.go
rtk go test ./tests/shops -run 'TestAdjustVehicleUsageAddsAndReturnsPersistedVehicle|TestAdjustVehicleUsageSubtractsForOrdinaryMember' -count=1
rtk go test ./api/shops/vehicles -count=1
rtk git diff --check
```

Expected: both integration tests pass, the handler package compiles, and the response includes persisted tracked values.

Commit:

```bash
rtk git add api/request/shops_request.go api/shops/vehicles tests/shops/shops_vehicles_test.go
rtk git commit -m "feat(shops): add equipment usage adjustment endpoint"
```

---

### Task 2: Harden validation, authorization, zero-floor, and concurrency

**Repository:** `/Users/swisscheese/projects/miltechserver`

**Files:**
- Modify: `api/shops/vehicles/handler.go`
- Modify: `api/shops/vehicles/service_impl.go`
- Modify: `api/shops/vehicles/repository_impl.go`
- Modify: `tests/shops/shops_vehicles_test.go`

**Interfaces:**
- Preserves: Task 1 route, request, response, and method signatures.
- Produces: exact 400/401/403/404/409 behavior from the spec.
- Produces: cumulative concurrent adjustments without lost updates.
- Preserves: valid legacy PUT behavior while rejecting negative legacy absolute tracked totals.

- [ ] **Step 1: Add table-driven request and floor tests**

Add table cases using a fresh fixture per subtest:

```go
cases := []struct {
	name       string
	body       map[string]interface{}
	userID     string
	wantStatus int
}{
	{name: "missing operation defaults add", body: map[string]interface{}{"mileage_adjustment": 1}, userID: "member", wantStatus: http.StatusOK},
	{name: "blank operation defaults add", body: map[string]interface{}{"operation": "  ", "hours_adjustment": 1}, userID: "member", wantStatus: http.StatusOK},
	{name: "hours only", body: map[string]interface{}{"operation": "subtract", "hours_adjustment": 1}, userID: "member", wantStatus: http.StatusOK},
	{name: "both omitted", body: map[string]interface{}{}, userID: "member", wantStatus: http.StatusBadRequest},
	{name: "both zero", body: map[string]interface{}{"mileage_adjustment": 0, "hours_adjustment": 0}, userID: "member", wantStatus: http.StatusBadRequest},
	{name: "negative magnitude", body: map[string]interface{}{"mileage_adjustment": -1}, userID: "member", wantStatus: http.StatusBadRequest},
	{name: "magnitude too large", body: map[string]interface{}{"hours_adjustment": 10001}, userID: "member", wantStatus: http.StatusBadRequest},
	{name: "unknown operation", body: map[string]interface{}{"operation": "negative", "hours_adjustment": 1}, userID: "member", wantStatus: http.StatusBadRequest},
	{name: "outsider", body: map[string]interface{}{"mileage_adjustment": 1}, userID: "outsider", wantStatus: http.StatusForbidden},
}
```

Add separate tests for no authenticated context (401), a nonexistent UUID (404), an unknown JSON field such as `comment` (400), and trailing JSON data (400). For the unknown-field case, build the request manually because `doJSONRequest` always emits one valid JSON value.

Add an exact-zero atomicity test starting at tracked mileage 10 and hours 10:

```go
body := map[string]interface{}{
	"operation":            "subtract",
	"mileage_adjustment": 10,
	"hours_adjustment":   10,
}
```

Assert 200 and both returned values equal zero. Then request subtraction of one more mile, assert 409, GET the equipment, and assert both values are still zero.

- [ ] **Step 2: Run the edge tests and record failures**

Run:

```bash
rtk go test ./tests/shops -run 'TestAdjustVehicleUsageValidation|TestAdjustVehicleUsageAllowsExactZero|TestAdjustVehicleUsageRejectsBelowZeroAtomically|TestAdjustVehicleUsageRejectsUnknownFields' -count=1
```

Expected: any incomplete error mapping, strict decoding, zero handling, or atomic rollback behavior fails with its observed status/value mismatch.

- [ ] **Step 3: Add the concurrent lost-update regression test**

Seed tracked mileage to 100, then send ten concurrent add-one requests as the same member. Do not call `require` from goroutines; send status codes and errors through buffered channels:

```go
const adjustmentCount = 10
statuses := make(chan int, adjustmentCount)
requestErrors := make(chan error, adjustmentCount)
var waitGroup sync.WaitGroup

for index := 0; index < adjustmentCount; index++ {
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		payload, err := json.Marshal(map[string]interface{}{
			"operation":           "add",
			"mileage_adjustment": 1,
		})
		if err != nil {
			requestErrors <- err
			return
		}
		req, err := http.NewRequest(
			http.MethodPatch,
			"/api/v1/auth/shops/vehicles/"+fixture.vehicleID+"/usage",
			bytes.NewReader(payload),
		)
		if err != nil {
			requestErrors <- err
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", fixture.memberID)
		recorder := httptest.NewRecorder()
		fixture.router.ServeHTTP(recorder, req)
		statuses <- recorder.Code
	}()
}

waitGroup.Wait()
close(statuses)
close(requestErrors)
require.Empty(t, requestErrors)
for status := range statuses {
	require.Equal(t, http.StatusOK, status)
}
```

GET the equipment afterward and assert `tracked_mileage == 110`.

- [ ] **Step 4: Run the concurrency test with the race detector**

Run:

```bash
rtk go test -race ./tests/shops -run TestAdjustVehicleUsageConcurrentAddsAreCumulative -count=1
```

Expected: PASS with final mileage 110 and no race report. If it fails below 110, the repository is reading outside the locked transaction or writing stale absolute values.

- [ ] **Step 5: Add legacy negative-value regression tests**

Use the existing PUT route twice: first with a valid positive tracked update and then with `tracked_mileage: -1`. Assert the valid request remains 200, the negative request is 400, and an authoritative GET retains the positive value.

In `UpdateShopVehicle`, validate pointers before creator/admin branching:

```go
func validateAbsoluteTrackedUsage(vehicle model.ShopVehicle) error {
	if vehicle.TrackedMileage != nil && *vehicle.TrackedMileage < 0 {
		return fmt.Errorf("%w: tracked_mileage cannot be negative", ErrInvalidUsageAdjustment)
	}
	if vehicle.TrackedHours != nil && *vehicle.TrackedHours < 0 {
		return fmt.Errorf("%w: tracked_hours cannot be negative", ErrInvalidUsageAdjustment)
	}
	return nil
}
```

Update only the legacy PUT handler's error branch so `ErrInvalidUsageAdjustment` uses `writeVehicleError`; preserve the existing `c.Error` path for its unrelated legacy errors.

- [ ] **Step 6: Verify Task 2 and commit**

Run:

```bash
rtk gofmt -w api/shops/vehicles tests/shops/shops_vehicles_test.go
rtk go test ./tests/shops -run 'TestAdjustVehicleUsage|TestLegacyVehicleUpdateRejectsNegativeTrackedUsage' -count=1
rtk go test -race ./tests/shops -run TestAdjustVehicleUsageConcurrentAddsAreCumulative -count=1
rtk git diff --check
```

Expected: all adjustment, authorization, validation, atomicity, concurrency, and legacy-negative tests pass.

Commit:

```bash
rtk git add api/shops/vehicles tests/shops/shops_vehicles_test.go
rtk git commit -m "fix(shops): enforce equipment usage bounds"
```

---

### Task 3: Add PostgreSQL nonnegative tracked-usage constraints

**Repository:** `/Users/swisscheese/projects/miltechserver`

**Files:**
- Create: `migrations/015_add_shop_vehicle_usage_constraints.sql`
- Create: `migrations/015_rollback_shop_vehicle_usage_constraints.sql`
- Create: `tests/shops/shops_vehicle_usage_schema_test.go`

**Interfaces:**
- Produces: `shop_vehicle_tracked_mileage_nonnegative` CHECK constraint.
- Produces: `shop_vehicle_tracked_hours_nonnegative` CHECK constraint.
- Preserves: nullable tracked columns and all generated Jet types.

- [ ] **Step 1: Write the failing schema test**

Create `tests/shops/shops_vehicle_usage_schema_test.go`:

```go
func TestShopVehicleTrackedUsageConstraints(t *testing.T) {
	for _, constraintName := range []string{
		"shop_vehicle_tracked_mileage_nonnegative",
		"shop_vehicle_tracked_hours_nonnegative",
	} {
		var validated bool
		err := testDB.QueryRow(`
			SELECT convalidated
			FROM pg_catalog.pg_constraint
			WHERE conrelid = 'public.shop_vehicle'::regclass
			  AND conname = $1`, constraintName).Scan(&validated)
		require.NoError(t, err)
		require.True(t, validated)
	}
}
```

Run:

```bash
rtk go test ./tests/shops -run TestShopVehicleTrackedUsageConstraints -count=1
```

Expected: FAIL with `sql: no rows in result set` before migration 015 is applied to `miltech_ng_test`.

- [ ] **Step 2: Create the forward migration**

Create `migrations/015_add_shop_vehicle_usage_constraints.sql` exactly as:

```sql
BEGIN;

SET LOCAL lock_timeout = '5s';

ALTER TABLE public.shop_vehicle
  ADD CONSTRAINT shop_vehicle_tracked_mileage_nonnegative
  CHECK (tracked_mileage IS NULL OR tracked_mileage >= 0) NOT VALID,
  ADD CONSTRAINT shop_vehicle_tracked_hours_nonnegative
  CHECK (tracked_hours IS NULL OR tracked_hours >= 0) NOT VALID;

ALTER TABLE public.shop_vehicle
  VALIDATE CONSTRAINT shop_vehicle_tracked_mileage_nonnegative;

ALTER TABLE public.shop_vehicle
  VALIDATE CONSTRAINT shop_vehicle_tracked_hours_nonnegative;

COMMIT;
```

- [ ] **Step 3: Create the exact rollback**

Create `migrations/015_rollback_shop_vehicle_usage_constraints.sql` exactly as:

```sql
BEGIN;

SET LOCAL lock_timeout = '5s';

ALTER TABLE public.shop_vehicle
  DROP CONSTRAINT shop_vehicle_tracked_mileage_nonnegative,
  DROP CONSTRAINT shop_vehicle_tracked_hours_nonnegative;

COMMIT;
```

- [ ] **Step 4: Audit and rehearse only on the test database**

Confirm the connection target before changing it:

```bash
rtk psql "$TEST_DATABASE_URL" -X -v ON_ERROR_STOP=1 -c 'SELECT current_database();'
rtk psql "$TEST_DATABASE_URL" -X -v ON_ERROR_STOP=1 -c 'SELECT COUNT(*) AS negative_rows FROM public.shop_vehicle WHERE tracked_mileage < 0 OR tracked_hours < 0;'
```

Expected: database is `miltech_ng_test` and `negative_rows` is 0. If either assertion is false, stop without applying the migration.

Apply, test, roll back, test absence, and reapply:

```bash
rtk psql "$TEST_DATABASE_URL" -X -v ON_ERROR_STOP=1 -f migrations/015_add_shop_vehicle_usage_constraints.sql
rtk go test ./tests/shops -run TestShopVehicleTrackedUsageConstraints -count=1
rtk psql "$TEST_DATABASE_URL" -X -v ON_ERROR_STOP=1 -f migrations/015_rollback_shop_vehicle_usage_constraints.sql
rtk psql "$TEST_DATABASE_URL" -X -v ON_ERROR_STOP=1 -c "SELECT COUNT(*) FROM pg_catalog.pg_constraint WHERE conrelid = 'public.shop_vehicle'::regclass AND conname IN ('shop_vehicle_tracked_mileage_nonnegative', 'shop_vehicle_tracked_hours_nonnegative');"
rtk psql "$TEST_DATABASE_URL" -X -v ON_ERROR_STOP=1 -f migrations/015_add_shop_vehicle_usage_constraints.sql
```

Expected: schema test passes after forward migration; count is 0 after rollback; final reapply succeeds. Do not run Jet generation because only CHECK constraints changed.

- [ ] **Step 5: Verify constraint enforcement and commit**

Add a test transaction that attempts `tracked_mileage = -1`, asserts a
`*pq.Error` with the mileage constraint name, and rolls the transaction back:

```go
func TestShopVehicleTrackedUsageConstraintsRejectNegativeValue(t *testing.T) {
	fixture := newVehicleUsageFixture(t, 100, 50)
	tx, err := testDB.Begin()
	require.NoError(t, err)
	defer tx.Rollback()

	_, err = tx.Exec(
		`UPDATE public.shop_vehicle SET tracked_mileage = -1 WHERE id = $1`,
		fixture.vehicleID,
	)
	require.Error(t, err)
	var postgresError *pq.Error
	require.ErrorAs(t, err, &postgresError)
	require.Equal(
		t,
		"shop_vehicle_tracked_mileage_nonnegative",
		postgresError.Constraint,
	)
}
```

Run:

```bash
rtk go test ./tests/shops -run 'TestShopVehicleTrackedUsageConstraints|TestShopVehicleTrackedUsageConstraintsRejectNegativeValue' -count=1
rtk git diff --check
```

Expected: both schema and enforcement tests pass.

Commit:

```bash
rtk git add migrations/015_add_shop_vehicle_usage_constraints.sql migrations/015_rollback_shop_vehicle_usage_constraints.sql tests/shops/shops_vehicle_usage_schema_test.go
rtk git commit -m "feat(database): constrain shop vehicle usage"
```

---

### Task 4: Publish the client API contract and complete the server gate

**Repository:** `/Users/swisscheese/projects/miltechserver`

**Files:**
- Create: `docs/api/shop_vehicle_usage_adjustments.md`

**Interfaces:**
- Documents: the Task 1 endpoint, JSON body, default-add behavior, one-operation rule, persisted response, errors, compatibility, retry policy, and deployment order.
- Produces: the normative client handoff consumed by mobile Tasks 6-8.

- [ ] **Step 1: Write the API document from the implemented contract**

The document must contain these sections in this order:

```markdown
# Shop Vehicle Usage Adjustments API

## Endpoint
## Authentication and authorization
## Request structure
## Add examples
## Subtract examples
## Success response
## Validation and error responses
## Backward compatibility
## Retry and reconciliation guidance
## Deployment order
```

Include these request examples:

```json
{
  "mileage_adjustment": 25
}
```

```json
{
  "operation": "subtract",
  "mileage_adjustment": 10,
  "hours_adjustment": 5
}
```

State explicitly that omission means add, one operation applies to both fields,
values are magnitudes rather than signed deltas, and subtraction to zero is
valid. Include the full persisted Shop vehicle success envelope and one example
for every documented HTTP status. State that clients must not automatically
retry or fall back to legacy PUT after an ambiguous failure.

- [ ] **Step 2: Verify documentation against executable tests**

Run:

```bash
rtk rg -n 'PATCH /api/v1/auth/shops/vehicles/:vehicle_id/usage|mileage_adjustment|hours_adjustment|operation|defaults to add|409|persisted' docs/api/shop_vehicle_usage_adjustments.md
rtk go test ./tests/shops -run 'TestAdjustVehicleUsage|TestLegacyVehicleUpdateRejectsNegativeTrackedUsage|TestShopVehicleTrackedUsageConstraints' -count=1
rtk go test ./tests/shops -count=1
rtk git diff --check
```

Expected: the document contains every contract term and the full Shop package passes.

- [ ] **Step 3: Commit the client handoff**

```bash
rtk git add docs/api/shop_vehicle_usage_adjustments.md
rtk git commit -m "docs(shops): document equipment usage adjustments"
```

- [ ] **Step 4: Run the server release gate**

Run:

```bash
rtk go test -race ./tests/shops -count=1
rtk go test -p 1 ./... -count=1
rtk git status --short --branch
rtk git log -4 --oneline
```

Expected: race-enabled Shops tests pass. The full suite must be reported by its actual exit code. The previously observed `tests/user_pmcs/TestPerformanceScenarios/approved_index_plans` PostgreSQL planner assertion may remain an unrelated baseline; if it appears, record its exact output and do not call the full suite green. Do not begin mobile Task 6 until the focused server contract, race test, and Shop suite are green.

---

### Task 5: Preserve null versus explicit zero in Flutter schema v6

**Repository:** `/Users/swisscheese/projects/miltech`

**Files:**
- Modify: `lib/_data/models/local_db/shops/shop_vehicle.dart`
- Modify: `lib/_data/remote_models/shops/shop_vehicle_rm.dart`
- Modify: `lib/_data/database/tables.dart`
- Modify: `lib/_data/database/database.dart`
- Generate: `lib/_data/database/database.g.dart`
- Generate: `lib/_data/database/database.steps.dart`
- Generate: `test/drift/database/generated/schema.json`
- Generate: `test/drift/database/generated/schema_v6.dart`
- Create: `test/models/shops/shop_vehicle_usage_test.dart`
- Modify: `test/drift/database/migration_test.dart`

**Interfaces:**
- Produces: nullable `ShopVehicle.trackedMileage` and `trackedHours`.
- Produces: `effectiveMileage => trackedMileage ?? mileage` and equivalent hours behavior.
- Preserves: `MilDb.schemaVersion == 6`.
- Produces: fresh-v6 nullable columns and v5-to-v6 conversion of zero sentinels to null.

- [ ] **Step 1: Establish the mobile baseline and recheck the release fact**

Run:

```bash
rtk git status --short --branch
rtk git rev-parse HEAD
rtk git worktree list
rtk rg -n 'int get schemaVersion => 6|from5To6' lib/_data/database/database.dart
rtk flutter test test/bloc/shops/shop_create_edit_authority_test.dart test/drift/database/migration_test.dart
```

Expected: schema version is 6, focused tests pass, and the task worktree does not contain the primary checkout's `dio_service.dart` modification. If v6 has shipped before execution starts, stop this task and return the plan for a v7 amendment.

- [ ] **Step 2: Write failing null-versus-zero domain tests**

Create `test/models/shops/shop_vehicle_usage_test.dart` with two remote models sharing base mileage 100 and hours 50:

```dart
test('null tracked values fall back to base usage', () {
  final vehicle = vehicleRM(trackedMileage: null, trackedHours: null)
      .toDomainModel();

  expect(vehicle.trackedMileage, isNull);
  expect(vehicle.trackedHours, isNull);
  expect(vehicle.effectiveMileage, 100);
  expect(vehicle.effectiveHours, 50);
});

test('explicit tracked zero remains zero', () {
  final vehicle = vehicleRM(trackedMileage: 0, trackedHours: 0)
      .toDomainModel();

  expect(vehicle.trackedMileage, 0);
  expect(vehicle.trackedHours, 0);
  expect(vehicle.effectiveMileage, 0);
  expect(vehicle.effectiveHours, 0);
});
```

The `vehicleRM` fixture supplies every required `ShopVehicleRM` constructor field with fixed IDs and UTC timestamps.

- [ ] **Step 3: Write failing fresh-v6 and v5-to-v6 Drift tests**

Add a fresh database test that reads `PRAGMA table_info(shop_vehicles)` and asserts `notnull == 0` for both tracked columns, then inserts an equipment row with both values null and reads it back.

Add a v5 migration test:

```dart
test('v5 to v6 converts tracked zero sentinels to null', () async {
  final schema = await verifier.schemaAt(5);
  final db = MilDb.forTesting(schema.newConnection());
  await db.customStatement(
    'INSERT INTO shops '
    '(id, user_id, name, details, created_by, created_at, updated_at) '
    'VALUES (?, ?, ?, ?, ?, ?, ?)',
    ['shop-1', '', 'Shop', '', '', userPmcsFixtureTimestamp, userPmcsFixtureTimestamp],
  );
  await db.customStatement(
    'INSERT INTO shop_vehicles '
    '(id, creator_id, niin, admin, model, serial, uoc, mileage, hours, '
    'comment, save_time, last_updated, shop_id, tracked_mileage, tracked_hours) '
    'VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)',
    [
      'vehicle-1', '', '', 'A123', '', '', 'UNK', 100, 50, '',
      userPmcsFixtureTimestamp, userPmcsFixtureTimestamp, 'shop-1', 0, 0,
    ],
  );

  await verifier.migrateAndValidate(db, 6);
  final vehicle = await (db.select(db.shopVehicleTable)
        ..where((row) => row.id.equals('vehicle-1')))
      .getSingle();
  expect(vehicle.trackedMileage, isNull);
  expect(vehicle.trackedHours, isNull);
  expect(db.schemaVersion, 6);
  await db.close();
});
```

- [ ] **Step 4: Run tests and verify current zero semantics fail**

Run:

```bash
rtk flutter test test/models/shops/shop_vehicle_usage_test.dart test/drift/database/migration_test.dart
```

Expected: FAIL because the domain and Drift columns are non-nullable and explicit zero falls back to base usage.

- [ ] **Step 5: Change the domain and remote mapping**

In `ShopVehicle`, make both fields `int?`, keep constructor parameters required, preserve nullable values in `fromTable` and `toCompanion`, and change the getters:

```dart
final int? trackedMileage;
final int? trackedHours;

int get effectiveMileage => trackedMileage ?? mileage;
int get effectiveHours => trackedHours ?? hours;
```

Keep `copyWith` capable of replacing values with non-null integers; no production flow needs to clear a non-null domain value through `copyWith`. In `ShopVehicleRM.toDomainModel`, assign `trackedMileage` and `trackedHours` directly without `?? 0`.

- [ ] **Step 6: Correct the unshipped v6 table and migration in place**

Change the live table definition:

```dart
IntColumn get trackedMileage => integer().nullable()();
IntColumn get trackedHours => integer().nullable()();
```

Keep `schemaVersion => 6`. At the start of `from5To6`, define `final database = m.database as MilDb;` and rebuild `shop_vehicles` with these transformers before creating new v6 tables:

```dart
await m.alterTable(
  TableMigration(
    database.shopVehicleTable,
    columnTransformer: {
      database.shopVehicleTable.trackedMileage:
          const CustomExpression<int?>('NULLIF(tracked_mileage, 0)'),
      database.shopVehicleTable.trackedHours:
          const CustomExpression<int?>('NULLIF(tracked_hours, 0)'),
    },
  ),
);
```

Do not edit `from2To3`, historical schema snapshots v1-v5, or add a new migration version.

- [ ] **Step 7: Regenerate v6 artifacts from source**

Run in this order:

```bash
rtk dart run build_runner build --delete-conflicting-outputs
rtk dart run drift_dev schema dump lib/_data/database/database.dart test/drift/database/generated/schema.json
rtk dart run drift_dev schema generate test/drift/database/generated/schema.json test/drift/database/generated/
rtk dart run drift_dev schema steps test/drift/database/generated/ lib/_data/database/database.steps.dart
rtk dart format lib/_data/models/local_db/shops/shop_vehicle.dart lib/_data/remote_models/shops/shop_vehicle_rm.dart lib/_data/database/tables.dart lib/_data/database/database.dart test/models/shops/shop_vehicle_usage_test.dart test/drift/database/migration_test.dart
```

Expected: generation completes; `database.g.dart`, `database.steps.dart`, `schema.json`, and `schema_v6.dart` reflect nullable tracked columns; no `schema_v7.dart` exists; ignored `shop_vehicle_rm.g.dart` need not be force-added because its JSON field types were already nullable.

- [ ] **Step 8: Verify and commit the v6 correction**

Run:

```bash
rtk flutter test test/models/shops/shop_vehicle_usage_test.dart test/drift/database/migration_test.dart test/drift/database/user_pmcs_schema_test.dart
rtk dart analyze lib/_data/models/local_db/shops/shop_vehicle.dart lib/_data/remote_models/shops/shop_vehicle_rm.dart lib/_data/database/tables.dart lib/_data/database/database.dart
rtk rg -n 'schemaVersion => 7|from6To7|Schema7' lib test/drift/database/generated
rtk git diff --check
```

Expected: focused tests and analysis pass; the final search returns no matches.

Commit:

```bash
rtk git add lib/_data/models/local_db/shops/shop_vehicle.dart lib/_data/remote_models/shops/shop_vehicle_rm.dart lib/_data/database/tables.dart lib/_data/database/database.dart lib/_data/database/database.g.dart lib/_data/database/database.steps.dart test/models/shops/shop_vehicle_usage_test.dart test/drift/database/migration_test.dart test/drift/database/generated/schema.json test/drift/database/generated/schema_v6.dart
rtk git commit -m "fix(shops): preserve explicit zero equipment usage"
```

---

### Task 6: Add Flutter usage-adjustment transport and authoritative response validation

**Repository:** `/Users/swisscheese/projects/miltech`

**Files:**
- Create: `lib/_data/models/shops/shop_vehicle_usage_operation.dart`
- Modify: `lib/_data/api/shop_url_builder.dart`
- Modify: `lib/_data/api/shop_api.dart`
- Modify: `lib/_data/repository/remote_shop_repository.dart`
- Create: `test/api/shop_api_vehicle_usage_test.dart`

**Interfaces:**
- Produces: `ShopVehicleUsageOperation { add, subtract }` with wire values and `apply`.
- Produces: `ShopUrlBuilder.buildAdjustShopVehicleUsageUrl(String vehicleId)`.
- Produces: `ShopApi.adjustShopVehicleUsage(String vehicleId, Map<String, dynamic> adjustmentData) -> Future<ShopVehicleRM>`.
- Produces: `RemoteShopRepository.adjustShopVehicleUsage({required String vehicleId, required ShopVehicleUsageOperation operation, int? mileageAdjustment, int? hoursAdjustment}) -> Future<ShopVehicle>`.
- Consumes: server Task 4 client contract.

- [ ] **Step 1: Write failing URL, payload, and response tests**

Create `test/api/shop_api_vehicle_usage_test.dart` using the existing `MockDioService` pattern:

```dart
test('builds the vehicle usage adjustment path', () {
  expect(
    const ShopUrlBuilder().buildAdjustShopVehicleUsageUrl('vehicle-1'),
    '/auth/shops/vehicles/vehicle-1/usage',
  );
});

test('patches usage and parses the persisted vehicle', () async {
  when(
    () => dio.patchForData(
      any(),
      data: any(named: 'data'),
    ),
  ).thenAnswer((_) async => persistedVehicleJson(
        trackedMileage: 90,
        trackedHours: 45,
      ));

  final result = await api.adjustShopVehicleUsage(
    'vehicle-1',
    {
      'operation': 'subtract',
      'mileage_adjustment': 10,
      'hours_adjustment': 5,
    },
  );

  expect(result.id, 'vehicle-1');
  expect(result.trackedMileage, 90);
  expect(result.trackedHours, 45);
  verify(
    () => dio.patchForData(
      '/auth/shops/vehicles/vehicle-1/usage',
      data: {
        'operation': 'subtract',
        'mileage_adjustment': 10,
        'hours_adjustment': 5,
      },
    ),
  ).called(1);
});
```

Add repository tests for mileage-only omission and response-ID mismatch. The mismatch test returns `id: vehicle-2` and expects `StateError`.

- [ ] **Step 2: Run tests and verify missing interfaces**

Run:

```bash
rtk flutter test test/api/shop_api_vehicle_usage_test.dart
```

Expected: FAIL because the operation enum, URL method, and API method do not exist.

- [ ] **Step 3: Add the shared operation enum**

Create:

```dart
enum ShopVehicleUsageOperation {
  add('add'),
  subtract('subtract');

  const ShopVehicleUsageOperation(this.wireValue);

  final String wireValue;

  int apply(int current, int magnitude) => switch (this) {
    ShopVehicleUsageOperation.add => current + magnitude,
    ShopVehicleUsageOperation.subtract => current - magnitude,
  };
}
```

- [ ] **Step 4: Implement URL, API, and repository methods**

Add the URL:

```dart
String buildAdjustShopVehicleUsageUrl(String vehicleId) =>
    '/auth/shops/vehicles/$vehicleId/usage';
```

The API method must use `patchForData`, require a map payload from the unwrapped `data`, and decode it:

```dart
Future<ShopVehicleRM> adjustShopVehicleUsage(
  String vehicleId,
  Map<String, dynamic> adjustmentData,
) async {
  final url = _urlBuilder.buildAdjustShopVehicleUsageUrl(vehicleId);
  final responseData = await _dioService.patchForData(
    url,
    data: adjustmentData,
  );
  if (responseData is! Map<String, dynamic>) {
    throw const FormatException('Invalid shop vehicle usage response');
  }
  return ShopVehicleRM.fromJson(responseData);
}
```

The repository method always sends the explicit UI operation and omits unused fields:

```dart
Future<ShopVehicle> adjustShopVehicleUsage({
  required String vehicleId,
  required ShopVehicleUsageOperation operation,
  int? mileageAdjustment,
  int? hoursAdjustment,
}) async {
  final remote = await shopApi.adjustShopVehicleUsage(
    vehicleId,
    {
      'operation': operation.wireValue,
      if (mileageAdjustment != null)
        'mileage_adjustment': mileageAdjustment,
      if (hoursAdjustment != null) 'hours_adjustment': hoursAdjustment,
    },
  );
  if (remote.id != vehicleId) {
    throw StateError('Usage response vehicle did not match request');
  }
  return remote.toDomainModel();
}
```

Do not call `updateShopVehicle`, calculate absolute remote totals, write Drift, or retry inside this method.

- [ ] **Step 5: Verify and commit transport**

Run:

```bash
rtk dart format lib/_data/models/shops/shop_vehicle_usage_operation.dart lib/_data/api/shop_url_builder.dart lib/_data/api/shop_api.dart lib/_data/repository/remote_shop_repository.dart test/api/shop_api_vehicle_usage_test.dart
rtk flutter test test/api/shop_api_vehicle_usage_test.dart
rtk dart analyze lib/_data/models/shops/shop_vehicle_usage_operation.dart lib/_data/api/shop_url_builder.dart lib/_data/api/shop_api.dart lib/_data/repository/remote_shop_repository.dart
rtk git diff --check
```

Expected: URL/payload/parse/ID-validation tests pass and analysis is clean.

Commit:

```bash
rtk git add lib/_data/models/shops/shop_vehicle_usage_operation.dart lib/_data/api/shop_url_builder.dart lib/_data/api/shop_api.dart lib/_data/repository/remote_shop_repository.dart test/api/shop_api_vehicle_usage_test.dart
rtk git commit -m "feat(shops): add equipment usage adjustment client"
```

---

### Task 7: Apply one operation through local persistence and Cubit state

**Repository:** `/Users/swisscheese/projects/miltech`

**Files:**
- Modify: `lib/_data/repository/local_shop_repository.dart`
- Modify: `lib/_bloc/shops/cubits/update_miles_hours_cubit.dart`
- Modify: `lib/_bloc/shops/states/update_miles_hours_state.dart`
- Create: `test/repository/local_shop_repository_usage_test.dart`
- Modify: `test/bloc/shops/shop_create_edit_authority_test.dart`

**Interfaces:**
- Produces: `LocalShopRepository.adjustVehicleUsage({required String vehicleId, required ShopVehicleUsageOperation operation, int? mileageAdjustment, int? hoursAdjustment}) -> Future<ShopVehicle?>`.
- Produces: `UpdateMilesHoursCubit.updateTrackedValues({required operation, mileageAdjustment, hoursAdjustment})`.
- Produces: success state containing the persisted remote or local `ShopVehicle`.
- Removes from this flow: logged-in calls to legacy `updateShopVehicle` and client-calculated absolute totals.

- [ ] **Step 1: Write failing local repository tests**

Create a memory Drift database with a visible local Shop and equipment. Test:

```dart
final subtracted = await repository.adjustVehicleUsage(
  vehicleId: 'vehicle-1',
  operation: ShopVehicleUsageOperation.subtract,
  mileageAdjustment: 100,
  hoursAdjustment: 50,
);

expect(subtracted?.trackedMileage, 0);
expect(subtracted?.trackedHours, 0);
expect(subtracted?.effectiveMileage, 0);
expect(subtracted?.effectiveHours, 0);
```

Add a second test starting at mileage 10/hours 20, subtracting mileage 11 and
hours 5. Expect `null` and assert an authoritative reread still returns 10 and
20. Add mileage-only and add-both tests. Add a hidden
`pmcs-sbs-local-shop` test expecting `null` and no mutation.

- [ ] **Step 2: Write failing Cubit authority and operation tests**

Replace the two existing usage authority tests with these contracts:

```dart
test('logged-in subtract calls adjustment endpoint and stores response', () async {
  final initial = shopVehicleFixture();
  final persisted = initial.copyWith(trackedMileage: 90, trackedHours: 20);
  stubLoggedIn(auth);
  when(
    () => remote.adjustShopVehicleUsage(
      vehicleId: initial.id,
      operation: ShopVehicleUsageOperation.subtract,
      mileageAdjustment: 10,
      hoursAdjustment: 5,
    ),
  ).thenAnswer((_) async => persisted);

  final cubit = UpdateMilesHoursCubit(
    localRepository: local,
    remoteRepository: remote,
    authRepository: auth,
    vehicle: initial,
  );
  final success = await cubit.updateTrackedValues(
    operation: ShopVehicleUsageOperation.subtract,
    mileageAdjustment: 10,
    hoursAdjustment: 5,
  );

  expect(success, isTrue);
  expect(cubit.state.vehicle, persisted);
  verifyNever(() => remote.updateShopVehicle(
        vehicleId: any(named: 'vehicleId'),
        admin: any(named: 'admin'),
        mileage: any(named: 'mileage'),
        hours: any(named: 'hours'),
        trackedMileage: any(named: 'trackedMileage'),
        trackedHours: any(named: 'trackedHours'),
      ));
  verifyNever(() => local.updateVehicleTrackedValues(any(), any(), any()));
});
```

Add logged-out subtraction-to-zero, below-zero prevalidation, one-field, and remote failure tests. The remote failure test verifies no local adjustment and no legacy update are attempted.

- [ ] **Step 3: Run focused tests and verify old add-only interfaces fail**

Run:

```bash
rtk flutter test test/repository/local_shop_repository_usage_test.dart test/bloc/shops/shop_create_edit_authority_test.dart
```

Expected: FAIL because `adjustVehicleUsage`, `adjustShopVehicleUsage`, and the operation parameter are absent.

- [ ] **Step 4: Implement local transactional adjustment**

Use this signature:

```dart
Future<ShopVehicle?> adjustVehicleUsage({
  required String vehicleId,
  required ShopVehicleUsageOperation operation,
  int? mileageAdjustment,
  int? hoursAdjustment,
})
```

Inside `milDb.transaction`, load the visible non-hidden equipment, calculate supplied values with `operation.apply(current.effectiveMileage, magnitude)`, reject the entire operation if either result is negative, and update only supplied tracked fields plus `lastUpdated`:

```dart
final rows = await (milDb.update(milDb.shopVehicleTable)
      ..where((row) => row.id.equals(vehicleId)))
    .write(
      ShopVehicleTableCompanion(
        trackedMileage: mileageAdjustment == null
            ? const Value.absent()
            : Value(nextMileage),
        trackedHours: hoursAdjustment == null
            ? const Value.absent()
            : Value(nextHours),
        lastUpdated: Value(DateTime.now().toUtc()),
      ),
    );
if (rows != 1) {
  return null;
}
final persisted = await (milDb.select(milDb.shopVehicleTable)
      ..where((row) => row.id.equals(vehicleId)))
    .getSingle();
return ShopVehicle.fromTable(persisted);
```

Remove the old `updateVehicleTrackedValues` method only after all call sites have moved. Do not update the full row with `replace`, because the usage operation owns only tracked fields and `lastUpdated`.

- [ ] **Step 5: Update the Cubit**

Use the exact public signature:

```dart
Future<bool> updateTrackedValues({
  required ShopVehicleUsageOperation operation,
  int? mileageAdjustment,
  int? hoursAdjustment,
})
```

Validate at least one positive magnitude, the 10,000 limit, and client-known below-zero subtraction before setting loading. For authenticated Shops, call only `remoteRepository.adjustShopVehicleUsage`; for logged-out Shops, call only `localRepository.adjustVehicleUsage`. Emit success with the returned vehicle:

```dart
emit(
  state.copyWith(
    status: UpdateMilesHoursStatus.success,
    vehicle: persistedVehicle,
  ),
);
```

On any remote failure, keep the dialog open and emit:

```text
Could not confirm the usage update. Refresh the equipment before trying again.
```

Do not retry, invoke the legacy PUT, or write locally after a logged-in failure.

- [ ] **Step 6: Verify and commit state/persistence**

Run:

```bash
rtk dart format lib/_data/repository/local_shop_repository.dart lib/_bloc/shops/cubits/update_miles_hours_cubit.dart lib/_bloc/shops/states/update_miles_hours_state.dart test/repository/local_shop_repository_usage_test.dart test/bloc/shops/shop_create_edit_authority_test.dart
rtk flutter test test/repository/local_shop_repository_usage_test.dart test/bloc/shops/shop_create_edit_authority_test.dart
rtk dart analyze lib/_data/repository/local_shop_repository.dart lib/_bloc/shops/cubits/update_miles_hours_cubit.dart lib/_bloc/shops/states/update_miles_hours_state.dart
rtk git diff --check
```

Expected: local atomicity, zero, authority, remote-response, and no-fallback tests pass.

Commit:

```bash
rtk git add lib/_data/repository/local_shop_repository.dart lib/_bloc/shops/cubits/update_miles_hours_cubit.dart lib/_bloc/shops/states/update_miles_hours_state.dart test/repository/local_shop_repository_usage_test.dart test/bloc/shops/shop_create_edit_authority_test.dart
rtk git commit -m "feat(shops): apply equipment usage operations"
```

---

### Task 8: Add one global Add/Subtract GUI control and result preview

**Repository:** `/Users/swisscheese/projects/miltech`

**Files:**
- Modify: `lib/_bloc/shops/widgets/update_miles_hours_dialog.dart`
- Create: `test/widgets/shops/update_miles_hours_dialog_test.dart`

**Interfaces:**
- Consumes: `ShopVehicleUsageOperation` and the Task 7 Cubit signature.
- Produces: one single-select Add/Subtract control applying to both fields.
- Produces: operation-aware labels, current effective usage, live preview, and floor-zero validation.

- [ ] **Step 1: Build the widget-test harness and write failing default/switch tests**

Pump `MaterialApp` under repository providers for mock `AuthenticationRepository`, `LocalShopRepository`, and `RemoteShopRepository`. Open the dialog through `UpdateMilesHoursDialog.show` from a `Builder` context. Add tests asserting:

```dart
expect(find.text('Add'), findsOneWidget);
expect(find.text('Subtract'), findsOneWidget);
expect(find.text('Miles to Add'), findsOneWidget);
expect(find.text('Hours to Add'), findsOneWidget);
```

Tap `Subtract` and assert both labels change:

```dart
await tester.tap(find.text('Subtract'));
await tester.pump();
expect(find.text('Miles to Subtract'), findsOneWidget);
expect(find.text('Hours to Subtract'), findsOneWidget);
```

Verify the `SegmentedButton<ShopVehicleUsageOperation>` has exactly one selected value and defaults to `{ShopVehicleUsageOperation.add}`.

- [ ] **Step 2: Add failing preview, zero, and below-zero tests**

With current effective mileage 100 and hours 50:

1. Select Subtract, enter mileage 100 and hours 50, assert previews `100 → 0 miles` and `50 → 0 hours`, and assert Update is enabled.
2. Change mileage to 101, assert the inline message `Mileage cannot be reduced below 0` and Update is disabled.
3. Enter only hours 10, assert mileage has no preview, hours preview is `50 → 40 hours`, and Update is enabled.
4. Enter a minus sign and assert the field remains digits-only.
5. Submit a valid subtraction and verify the Cubit-facing repository mock receives one `subtract` operation governing both magnitudes.

- [ ] **Step 3: Run widget tests and verify the old add-only dialog fails**

Run:

```bash
rtk flutter test test/widgets/shops/update_miles_hours_dialog_test.dart
```

Expected: FAIL because the selector, subtract labels, and preview do not exist.

- [ ] **Step 4: Add the single global selector**

Keep `_operation` in the dialog state and initialize it to add:

```dart
ShopVehicleUsageOperation _operation = ShopVehicleUsageOperation.add;
```

Add one control before the inputs:

```dart
SegmentedButton<ShopVehicleUsageOperation>(
  segments: const [
    ButtonSegment(
      value: ShopVehicleUsageOperation.add,
      label: Text('Add'),
      icon: Icon(Icons.add),
    ),
    ButtonSegment(
      value: ShopVehicleUsageOperation.subtract,
      label: Text('Subtract'),
      icon: Icon(Icons.remove),
    ),
  ],
  selected: {_operation},
  onSelectionChanged: isLoading
      ? null
      : (selection) {
          setState(() {
            _operation = selection.single;
          });
          _formKey.currentState?.validate();
        },
)
```

This is the only operation selector; do not add per-field direction controls.

- [ ] **Step 5: Make validation, copy, preview, and submit operation-aware**

Use `widget.vehicle.effectiveMileage` and `effectiveHours` in current-value copy. Rename parsed variables to `mileageAdjustment` and `hoursAdjustment`. Dynamic labels use `Add` or `Subtract` from `_operation`.

For preview and validation, calculate only supplied values:

```dart
int? _result(int current, String text) {
  final adjustment = _parseValue(text);
  if (adjustment == null) {
    return null;
  }
  return _operation.apply(current, adjustment);
}
```

Allow zero results and reject only negative results. Submit exactly one operation:

```dart
final success = await cubit.updateTrackedValues(
  operation: _operation,
  mileageAdjustment: _parseValue(_mileageController.text),
  hoursAdjustment: _parseValue(_hoursController.text),
);
```

Keep `FilteringTextInputFormatter.digitsOnly`, the 10,000 per-field limit, cancellation behavior, and loading disablement. Change success copy to `Usage added and synced to server`, `Usage subtracted and synced to server`, or the corresponding local wording.

- [ ] **Step 6: Verify accessibility-relevant widget behavior and commit**

Run:

```bash
rtk dart format lib/_bloc/shops/widgets/update_miles_hours_dialog.dart test/widgets/shops/update_miles_hours_dialog_test.dart
rtk flutter test test/widgets/shops/update_miles_hours_dialog_test.dart test/bloc/shops/shop_create_edit_authority_test.dart
rtk flutter analyze lib/_bloc/shops/widgets/update_miles_hours_dialog.dart
rtk git diff --check
```

Expected: default Add, global switch, preview, one-field, exact-zero, below-zero, digits-only, submit, loading, and failure-stays-open tests pass.

Commit:

```bash
rtk git add lib/_bloc/shops/widgets/update_miles_hours_dialog.dart test/widgets/shops/update_miles_hours_dialog_test.dart
rtk git commit -m "feat(shops): add usage subtraction control"
```

---

### Task 9: Run cross-repository compatibility and release verification

**Repositories:** `/Users/swisscheese/projects/miltechserver` and `/Users/swisscheese/projects/miltech`

**Files:**
- Verify only; any required correction receives its own focused test and narrow commit in the repository that owns it.

**Interfaces:**
- Verifies: old-client/new-server compatibility, new-client/new-server behavior, v6-only Drift history, and deployment ordering.
- Produces: exact test counts, exit codes, commits, HEADs, and worktree status for handoff.

- [ ] **Step 1: Verify server from a clean task worktree**

Run:

```bash
rtk go test ./api/shops/vehicles -count=1
rtk go test ./tests/shops -count=1
rtk go test -race ./tests/shops -count=1
rtk go test -p 1 ./... -count=1
rtk git diff --check
rtk git status --short --branch
rtk git rev-parse HEAD
```

Expected: focused, Shop, and race runs pass. Report the full suite's real exit code and exact unrelated planner failure if the known baseline recurs.

- [ ] **Step 2: Verify Flutter generation, focused behavior, and full health**

Run:

```bash
rtk dart run build_runner build --delete-conflicting-outputs
rtk flutter test test/models/shops/shop_vehicle_usage_test.dart test/api/shop_api_vehicle_usage_test.dart test/repository/local_shop_repository_usage_test.dart test/bloc/shops/shop_create_edit_authority_test.dart test/widgets/shops/update_miles_hours_dialog_test.dart test/drift/database/migration_test.dart test/drift/database/user_pmcs_schema_test.dart
rtk flutter analyze
rtk flutter test --concurrency=1
rtk git diff --check
rtk git status --short --branch
rtk git rev-parse HEAD
```

Expected: generation has no unexplained churn, focused tests pass, analysis exit code is zero, and the full test suite is reported by exact exit code and counts.

- [ ] **Step 3: Inspect the compatibility matrix**

Verify with tests and diffs:

```text
Old client + new server: valid legacy PUT remains 200 with its existing body.
New client + new server: PATCH returns the persisted vehicle and exact zero remains zero.
New client + old server: PATCH fails; no legacy PUT or local authenticated write follows.
Concurrent new clients: adjustments accumulate without lost updates.
Flutter persistence: schemaVersion is 6; fresh v6 and v5-to-v6 both preserve null/zero semantics.
```

Search for prohibited fallback and v7 artifacts:

```bash
rtk rg -n 'schemaVersion => 7|from6To7|Schema7|schema_v7' lib test
rtk rg -n 'updateShopVehicle\(' lib/_bloc/shops/cubits/update_miles_hours_cubit.dart
rtk rg -n 'updateVehicleTrackedValues\(' lib test
```

Expected: all three searches return no matches in the migrated usage flow; unrelated legitimate full-equipment `updateShopVehicle` calls elsewhere remain untouched.

- [ ] **Step 4: Prepare the deployment handoff without deploying**

Record:

1. migration 015 forward and rollback rehearsal results on `miltech_ng_test`;
2. server task commits and final server HEAD;
3. mobile task commits and final mobile HEAD;
4. focused/full/race/analyze commands with exit codes and counts;
5. known unrelated failures;
6. clean task-worktree state and preserved primary-checkout changes; and
7. required deployment order: migration 015, server release, then Flutter release.

Do not push, merge, deploy, migrate production, or delete task worktrees until the user explicitly chooses the integration action.
