# Shop Vehicle Notification Attached List Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add nullable `attached_shop_list` support to every shop vehicle notification request and response path, including create/update writes and aggregate read responses.

**Architecture:** Treat `attached_shop_list` as an optional relationship from `shop_vehicle_notifications` to `shop_lists(id)`. Direct notification reads can continue using the regenerated Jet model, while manual SQL aggregate reads must explicitly select and scan the column. Update requests need field-presence tracking so omitted fields leave the existing attachment unchanged and explicit `null` clears it.

**Tech Stack:** Go, Gin request binding, Postgres, Jet generated table/model types, existing Shops integration tests.

---

## Current-State Findings

- Database schema in `miltech_ng_test`:
  - `shop_vehicle_notifications.attached_shop_list` is nullable `text`.
  - The column should have `FOREIGN KEY (attached_shop_list) REFERENCES shop_lists(id) ON DELETE SET NULL`.
  - If an environment reports the FK without `ON DELETE SET NULL`, the schema migration has not been applied there yet.
  - There is no index on `attached_shop_list`.
- Generated Jet files are already current:
  - `.gen/miltech_ng/public/model/shop_vehicle_notifications.go` includes `AttachedShopList *string json:"attached_shop_list"`.
  - `.gen/miltech_ng/public/table/shop_vehicle_notifications.go` includes `ShopVehicleNotifications.AttachedShopList` in `AllColumns` and `MutableColumns`.
- Direct notification read paths already select `ShopVehicleNotifications.AllColumns`, so they should emit `attached_shop_list` after writes populate it:
  - `api/shops/vehicles/notifications/repository_impl.go`
  - `api/shops/vehicles/notifications/items/repository_impl.go`
  - `api/shops/vehicles/notifications/changes/repository_impl.go` notification lookup helpers
- Direct write gaps:
  - `CreateVehicleNotificationRequest` does not accept `attached_shop_list`.
  - `UpdateVehicleNotificationRequest` does not accept `attached_shop_list`.
  - `Handler.CreateVehicleNotification` does not copy it into the model.
  - `Handler.UpdateVehicleNotification` cannot distinguish omitted from explicit null.
  - `RepositoryImpl.CreateVehicleNotification` enumerates insert columns and omits `AttachedShopList`.
  - `RepositoryImpl.UpdateVehicleNotification` enumerates update columns and omits `AttachedShopList`.
- Aggregate response gaps:
  - `api/shops/aggregates/repository_impl.go:getVehicleNotifications` uses raw SQL and scans only the old nine notification columns.
  - `api/shops/aggregates/repository_impl.go:getShopSnapshotNotifications` uses raw SQL and scans only the old nine notification columns.
- Security/data-integrity gap:
  - The FK proves the list exists, but it does not prove the list belongs to the same shop as the notification. The service should validate same-shop ownership before writing `attached_shop_list`.
- Delete behavior:
  - With `ON DELETE SET NULL`, deleting a `shop_lists` row should automatically clear `shop_vehicle_notifications.attached_shop_list` for notifications attached to that list.
- Performance note:
  - No new index is required for this feature if the API only writes the FK and returns it with notification rows.
  - Add an index on `shop_vehicle_notifications(attached_shop_list)` only if future endpoints filter by attached list or list deletes become frequent enough for FK checks to matter. Follow ADR-016 and require representative `EXPLAIN (ANALYZE, BUFFERS)` before adding it.

## Request Contract

- Create notification:
  - Accept optional `attached_shop_list`.
  - Omitted or `null` means store `NULL`.
  - Non-empty string means attach to that list after validating the list belongs to the notification shop.
- Update notification:
  - Omitted `attached_shop_list` means leave the existing attachment unchanged.
  - `"attached_shop_list": null` means clear the attachment.
  - `"attached_shop_list": "<list-id>"` means attach to that list after validating same-shop ownership.
  - Empty string should be rejected as an invalid list id instead of relying on a raw FK error.
- Response contract:
  - All notification objects should include `attached_shop_list` with either a string list id or `null`.
  - This includes direct notification endpoints, notification-with-items endpoints, vehicle maintenance snapshots, and shop snapshots.

## Files To Modify

- `api/request/shops_request.go`
  - Add create request field.
  - Add update request field with presence semantics.
- `api/shops/vehicles/notifications/handler.go`
  - Copy create attachment field into `model.ShopVehicleNotifications`.
  - Convert update request into an internal update command that preserves presence.
- `api/shops/vehicles/notifications/service.go`
  - Update the service method signature if an internal command type is introduced.
- `api/shops/vehicles/notifications/service_impl.go`
  - Validate attached list ownership on create and update.
  - Include attachment changes in audit field changes.
- `api/shops/vehicles/notifications/repository.go`
  - Add a list lookup or validation method.
  - Update the notification update method signature if needed.
- `api/shops/vehicles/notifications/repository_impl.go`
  - Insert `AttachedShopList`.
  - Conditionally update `AttachedShopList`.
  - Implement same-shop list lookup/validation.
- `api/shops/aggregates/repository_impl.go`
  - Add `attached_shop_list` to raw notification SELECT and Scan calls.
- `tests/shops/shops_notifications_test.go`
  - Cover create, read, update, clear, and invalid/cross-shop attachment behavior.
- `tests/shops/shops_aggregate_vehicle_maintenance_test.go`
  - Cover `attached_shop_list` in vehicle maintenance snapshot notifications.
- `tests/shops/shops_aggregate_shop_snapshot_test.go`
  - Cover `attached_shop_list` in shop snapshot notifications.
- `tests/shops/helpers_test.go`
  - Add a helper variant for creating notifications with an attached list, or keep existing helper unchanged and create attachments inline in focused tests.
- `docs/api/shops_api_efficiency_mobile.md`
  - Update aggregate notification examples/field tables if they describe notification payload fields.
- `docs/api/shops_api_efficiency_mobile_refactor_guide.md`
  - Update examples/field descriptions if notification payload fields are documented there.

## Files Not To Modify

- `.gen/miltech_ng/public/model/shop_vehicle_notifications.go`
- `.gen/miltech_ng/public/table/shop_vehicle_notifications.go`
- Any other generated Jet file

## Task 1: Add Request Field Semantics

**Files:**
- Modify: `api/request/shops_request.go`

- [ ] **Step 1: Add a nullable string presence type**

Add a small type in `api/request/shops_request.go` or a focused request helper file under `api/request`.

```go
type NullableStringField struct {
	Set   bool
	Value *string
}

func (field *NullableStringField) UnmarshalJSON(data []byte) error {
	field.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		field.Value = nil
		return nil
	}

	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	field.Value = &value
	return nil
}
```

If this is placed in `shops_request.go`, add `bytes` and `encoding/json` to that file's imports.

- [ ] **Step 2: Extend create request**

Add this field to `CreateVehicleNotificationRequest`:

```go
AttachedShopList *string `json:"attached_shop_list"`
```

- [ ] **Step 3: Extend update request**

Add this field to `UpdateVehicleNotificationRequest`:

```go
AttachedShopList NullableStringField `json:"attached_shop_list"`
```

- [ ] **Step 4: Run request package compile check**

Run:

```bash
go test ./api/request
```

Expected: package compiles, or Go reports no test files.

## Task 2: Define an Internal Update Command

**Files:**
- Modify: `api/shops/vehicles/notifications/service.go`
- Modify: `api/shops/vehicles/notifications/service_impl.go`
- Modify: `api/shops/vehicles/notifications/repository.go`
- Modify: `api/shops/vehicles/notifications/repository_impl.go`
- Modify: `api/shops/vehicles/notifications/handler.go`

- [ ] **Step 1: Add a notification update command**

Define a package-local or exported type in `api/shops/vehicles/notifications/service.go`:

```go
type VehicleNotificationUpdate struct {
	Notification model.ShopVehicleNotifications
	AttachedShopListSet bool
	AttachedShopList *string
}
```

- [ ] **Step 2: Change service and repository update signatures**

Change:

```go
UpdateVehicleNotification(user *bootstrap.User, notification model.ShopVehicleNotifications) error
```

to:

```go
UpdateVehicleNotification(user *bootstrap.User, update VehicleNotificationUpdate) error
```

Apply the same signature change to the concrete service and repository implementations.

- [ ] **Step 3: Update handler mapping**

In `Handler.UpdateVehicleNotification`, map the request to the command:

```go
update := VehicleNotificationUpdate{
	Notification: model.ShopVehicleNotifications{
		ID:          req.NotificationID,
		Title:       req.Title,
		Description: req.Description,
		Type:        req.Type,
		Completed:   req.Completed,
	},
	AttachedShopListSet: req.AttachedShopList.Set,
	AttachedShopList:    req.AttachedShopList.Value,
}
```

Because `handler.go` is in the same `notifications` package, use `VehicleNotificationUpdate{...}` without a package qualifier.

- [ ] **Step 4: Preserve current full-field update behavior**

Keep title, description, type, and completed as required full-update fields. Only `attached_shop_list` should behave as an optional update field.

## Task 3: Validate Attached List Ownership

**Files:**
- Modify: `api/shops/vehicles/notifications/repository.go`
- Modify: `api/shops/vehicles/notifications/repository_impl.go`
- Modify: `api/shops/vehicles/notifications/service_impl.go`

- [ ] **Step 1: Add repository lookup method**

Add to the notifications repository interface:

```go
GetShopListByID(user *bootstrap.User, listID string) (*model.ShopLists, error)
```

- [ ] **Step 2: Implement list lookup**

Implement with Jet:

```go
stmt := SELECT(ShopLists.AllColumns).
	FROM(ShopLists).
	WHERE(ShopLists.ID.EQ(String(listID)))
```

Return `errors.New("shop list not found")` for no rows, matching local repository style.

- [ ] **Step 3: Add service validation helper**

Add a helper in `service_impl.go`:

```go
func (service *ServiceImpl) validateAttachedShopList(user *bootstrap.User, shopID string, attachedShopList *string) error {
	if attachedShopList == nil {
		return nil
	}
	if strings.TrimSpace(*attachedShopList) == "" {
		return errors.New("invalid attached_shop_list")
	}

	list, err := service.repo.GetShopListByID(user, *attachedShopList)
	if err != nil {
		return fmt.Errorf("failed to get attached shop list: %w", err)
	}
	if list.ShopID != shopID {
		return errors.New("attached_shop_list does not belong to notification shop")
	}
	return nil
}
```

Add `strings` to imports.

- [ ] **Step 4: Validate on create**

After the vehicle is loaded and `notification.ShopID = vehicle.ShopID`, call:

```go
if err := service.validateAttachedShopList(user, notification.ShopID, notification.AttachedShopList); err != nil {
	return nil, err
}
```

- [ ] **Step 5: Validate on update only when included**

After `currentNotification` is loaded, call:

```go
if update.AttachedShopListSet {
	if err := service.validateAttachedShopList(user, currentNotification.ShopID, update.AttachedShopList); err != nil {
		return err
	}
	update.Notification.AttachedShopList = update.AttachedShopList
} else {
	update.Notification.AttachedShopList = currentNotification.AttachedShopList
}
```

This ensures audit comparison sees the effective new value.

## Task 4: Write Direct Notification Tests First

**Files:**
- Modify: `tests/shops/shops_notifications_test.go`

- [ ] **Step 1: Test create and direct reads include attachment**

Create a shop, vehicle, and list. Create a notification with:

```json
{
  "shop_id": "<shop_id>",
  "vehicle_id": "<vehicle_id>",
  "title": "Attached PM",
  "description": "desc",
  "type": "PM",
  "attached_shop_list": "<list_id>"
}
```

Assert the create response data contains:

```go
require.Equal(t, listID, notificationData["attached_shop_list"])
```

Then assert the same field is present on:

- `GET /api/v1/auth/shops/vehicles/:vehicle_id/notifications`
- `GET /api/v1/auth/shops/vehicles/:vehicle_id/notifications-with-items`
- `GET /api/v1/auth/shops/:shop_id/notifications`
- `GET /api/v1/auth/shops/vehicles/notifications/:notification_id`

- [ ] **Step 2: Test update sets attachment**

Create a notification with no attachment. Update with `"attached_shop_list": "<list_id>"`. Fetch by id and assert the field equals the list id.

- [ ] **Step 3: Test omitted update preserves attachment**

Update the same notification again without `attached_shop_list`. Fetch by id and assert the existing attachment remains.

- [ ] **Step 4: Test explicit null clears attachment**

Update with `"attached_shop_list": null`. Fetch by id and assert:

```go
require.Nil(t, notificationData["attached_shop_list"])
```

- [ ] **Step 5: Test cross-shop list is rejected**

Create a second shop and list. Attempt to attach the second shop's list to the first shop's notification. Assert the request fails and the stored attachment remains unchanged.

The current shared error middleware maps most service errors to 500. If that is still true during implementation, assert non-2xx and document the current behavior. Do not expand this feature into a global error-handler refactor.

- [ ] **Step 6: Run the focused tests and confirm failure before implementation**

Run:

```bash
go test ./tests/shops -run 'TestVehicleNotificationsAndChanges|TestShopAndVehicleChangeLists|TestVehicleNotificationAttachedShopList' -count=1
```

Expected before implementation: new attachment tests fail because request/write paths do not persist the field.

## Task 5: Persist Attachment on Create and Update

**Files:**
- Modify: `api/shops/vehicles/notifications/handler.go`
- Modify: `api/shops/vehicles/notifications/repository_impl.go`

- [ ] **Step 1: Copy create request field into the model**

In `Handler.CreateVehicleNotification`, add:

```go
AttachedShopList: req.AttachedShopList,
```

- [ ] **Step 2: Insert `AttachedShopList`**

In `RepositoryImpl.CreateVehicleNotification`, add `ShopVehicleNotifications.AttachedShopList` to the insert column list.

- [ ] **Step 3: Conditionally update `AttachedShopList`**

In `RepositoryImpl.UpdateVehicleNotification`, build update columns and assignments dynamically enough to include `AttachedShopList` only when `update.AttachedShopListSet` is true.

Use `NULL` assignment when the value is nil. With Jet, use the existing project-compatible nullable expression pattern. If Jet's typed null helper is not obvious, use a small raw SQL update for this method rather than unsafe string concatenation:

```sql
UPDATE shop_vehicle_notifications
SET title = $1,
    description = $2,
    type = $3,
    completed = $4,
    last_updated = $5,
    attached_shop_list = $6
WHERE id = $7
```

Only use this SQL shape for the branch where the field is included. Keep the existing Jet update branch for omitted attachment if that is simpler.

- [ ] **Step 4: Preserve row-count behavior**

Keep the existing `RowsAffected` check and `notification not found` error behavior.

## Task 6: Include Attachment in Aggregate Raw SQL

**Files:**
- Modify: `api/shops/aggregates/repository_impl.go`

- [ ] **Step 1: Update vehicle maintenance notification query**

Change `getVehicleNotifications` from:

```sql
SELECT id, shop_id, vehicle_id, title, description, type, completed, save_time, last_updated
```

to:

```sql
SELECT id, shop_id, vehicle_id, title, description, type, completed, save_time, last_updated, attached_shop_list
```

Add scan target:

```go
&notification.AttachedShopList,
```

- [ ] **Step 2: Update shop snapshot notification query**

Change `getShopSnapshotNotifications` from:

```sql
SELECT n.id, n.shop_id, n.vehicle_id, n.title, n.description, n.type, n.completed, n.save_time, n.last_updated
```

to:

```sql
SELECT n.id, n.shop_id, n.vehicle_id, n.title, n.description, n.type, n.completed, n.save_time, n.last_updated, n.attached_shop_list
```

Add scan target:

```go
&notification.AttachedShopList,
```

## Task 7: Update Audit Field Changes

**Files:**
- Modify: `api/shops/vehicles/notifications/service_impl.go`

- [ ] **Step 1: Include attachment changes in audit JSON**

In `buildFieldChanges`, compare pointer values:

```go
if !sameStringPtr(old.AttachedShopList, new.AttachedShopList) {
	changedFields = append(changedFields, "attached_shop_list")
}
```

Add helper:

```go
func sameStringPtr(left, right *string) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}
```

- [ ] **Step 2: Keep change type behavior unchanged**

Do not alter `determineChangeType`; attachment changes should remain `"update"` unless the completed flag causes `"complete"` or `"reopen"`.

## Task 8: Add Aggregate Tests

**Files:**
- Modify: `tests/shops/shops_aggregate_vehicle_maintenance_test.go`
- Modify: `tests/shops/shops_aggregate_shop_snapshot_test.go`

- [ ] **Step 1: Vehicle maintenance snapshot test**

Create a list, create or update a notification to attach it, call:

```text
GET /api/v1/auth/shops/vehicles/:vehicle_id/maintenance-snapshot
```

Assert:

```go
notifications := payload["notifications"].([]interface{})
notification := notifications[0].(map[string]interface{})["notification"].(map[string]interface{})
require.Equal(t, listID, notification["attached_shop_list"])
```

- [ ] **Step 2: Shop snapshot test**

Create a list, create or update a notification to attach it, call:

```text
GET /api/v1/auth/shops/:shop_id/snapshot
```

Assert the same nested `notification.attached_shop_list` value.

- [ ] **Step 3: Run aggregate tests and confirm failure before implementation**

Run:

```bash
go test ./tests/shops -run 'TestVehicleMaintenanceSnapshot|TestShopSnapshot' -count=1
```

Expected before implementation: new aggregate assertions fail because raw SQL does not scan `attached_shop_list`.

## Task 9: Update API Documentation

**Files:**
- Modify: `docs/api/shops_api_efficiency_mobile.md`
- Modify: `docs/api/shops_api_efficiency_mobile_refactor_guide.md`

- [ ] **Step 1: Search docs for notification payload examples**

Run:

```bash
rg -n '"notification"|attached_shop_list|VehicleNotificationWithItems|notifications' docs/api/shops_api_efficiency_mobile.md docs/api/shops_api_efficiency_mobile_refactor_guide.md
```

- [ ] **Step 2: Add field documentation where notification objects are described**

Use this field description:

```markdown
| `attached_shop_list` | string or null | Optional id of the `shop_lists` row attached to this notification. |
```

- [ ] **Step 3: Add the field to JSON examples that show notification objects**

Use:

```json
"attached_shop_list": null
```

or an example list id where the example also includes matching list data.

## Task 10: Verification

**Files:**
- No file modifications in this task.

- [ ] **Step 1: Verify FK action in the target test database**

Run:

```bash
psql "$TEST_DATABASE_URL" -v ON_ERROR_STOP=1 -X -c "SELECT pg_get_constraintdef(oid) AS definition FROM pg_constraint WHERE conrelid = 'public.shop_vehicle_notifications'::regclass AND conname = 'shop_vehicle_notifications_attached_list_fkey';"
```

Expected:

```text
FOREIGN KEY (attached_shop_list) REFERENCES shop_lists(id) ON DELETE SET NULL
```

If the result does not include `ON DELETE SET NULL`, apply the schema migration before running the integration tests for this feature.

- [ ] **Step 2: Format Go files**

Run:

```bash
gofmt -w api/request/shops_request.go api/shops/vehicles/notifications/handler.go api/shops/vehicles/notifications/service.go api/shops/vehicles/notifications/service_impl.go api/shops/vehicles/notifications/repository.go api/shops/vehicles/notifications/repository_impl.go api/shops/aggregates/repository_impl.go tests/shops/shops_notifications_test.go tests/shops/shops_aggregate_vehicle_maintenance_test.go tests/shops/shops_aggregate_shop_snapshot_test.go
```

- [ ] **Step 3: Run focused Shops tests**

Run:

```bash
go test ./tests/shops -run 'TestVehicleNotificationAttachedShopList|TestVehicleNotificationsAndChanges|TestShopAndVehicleChangeLists|TestVehicleMaintenanceSnapshot|TestShopSnapshot' -count=1
```

Expected: pass.

- [ ] **Step 4: Run full Shops integration tests**

Run:

```bash
go test ./tests/shops -count=1
```

Expected: pass.

- [ ] **Step 5: Run package compile/tests for touched packages**

Run:

```bash
go test ./api/shops/vehicles/notifications ./api/shops/aggregates ./api/request
```

Expected: pass or no test files.

- [ ] **Step 6: Optional full repo check**

Run if time allows:

```bash
go test ./...
```

If unrelated packages fail, capture the failing package and error, then report it as baseline noise only after the focused Shops tests pass.

## Commit Plan

- [ ] Commit after implementation and verification:

```bash
git add api/request/shops_request.go api/shops/vehicles/notifications/handler.go api/shops/vehicles/notifications/service.go api/shops/vehicles/notifications/service_impl.go api/shops/vehicles/notifications/repository.go api/shops/vehicles/notifications/repository_impl.go api/shops/aggregates/repository_impl.go tests/shops/shops_notifications_test.go tests/shops/shops_aggregate_vehicle_maintenance_test.go tests/shops/shops_aggregate_shop_snapshot_test.go docs/api/shops_api_efficiency_mobile.md docs/api/shops_api_efficiency_mobile_refactor_guide.md
git commit -m "feat(shops): attach notifications to shop lists"
```

Do not include generated Jet files in the commit unless they changed before this plan was executed for a separate schema-regeneration reason.

## Schema Prerequisite

The required FK action is `ON DELETE SET NULL`. The application implementation assumes attached notifications are automatically detached when a referenced shop list is deleted. If any local or CI database still reports the old default FK action, update that schema before treating the feature tests as authoritative.
