package shops_test

import (
	"database/sql"
	"net/http"
	"strings"
	"testing"

	"miltechserver/api/request"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func itemFieldsPtr(value string) *string { return &value }

func storedItemFields(t *testing.T, itemID string) (quantity int32, nickname, unitOfMeasure sql.NullString) {
	t.Helper()
	require.NoError(t, testDB.QueryRow(
		`SELECT quantity, nickname, unit_of_measure FROM shop_notification_items WHERE id=$1`, itemID,
	).Scan(&quantity, &nickname, &unitOfMeasure))
	return quantity, nickname, unitOfMeasure
}

// Released clients never send nickname/unit_of_measure. Their saves must keep
// what a newer client stored, and a newer client must still be able to clear
// a nickname by sending "".
func TestAtomicNotificationItemFieldsSurviveReleasedClientSaves(t *testing.T) {
	router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
	create := atomicNotificationRequest(shopID, vehicleID)
	create.Items[0].Nickname = itemFieldsPtr("Front hub")
	create.Items[0].UnitOfMeasure = itemFieldsPtr("KT")
	receipt := atomicReceipt(t, doContract2JSONRequest(t, router, create, "atomic-owner"))

	quantity, nickname, unit := storedItemFields(t, create.Items[0].ID)
	require.Equal(t, int32(2), quantity)
	require.Equal(t, sql.NullString{String: "Front hub", Valid: true}, nickname)
	require.Equal(t, sql.NullString{String: "KT", Valid: true}, unit)
	_, legacyNickname, legacyUnit := storedItemFields(t, create.Items[1].ID)
	require.False(t, legacyNickname.Valid, "omitted nickname must store NULL")
	require.False(t, legacyUnit.Valid, "omitted unit must store NULL")

	// Marshalling nil pointers with omitempty produces the released body.
	releasedEdit := create
	releasedEdit.OperationID = uuid.NewString()
	releasedEdit.NotificationID = &receipt.NotificationID
	releasedEdit.Items = []request.NotificationSaveItem{
		{ID: create.Items[0].ID, Niin: create.Items[0].Niin, Nomenclature: create.Items[0].Nomenclature, Quantity: 7},
		{ID: create.Items[1].ID, Niin: create.Items[1].Niin, Nomenclature: create.Items[1].Nomenclature, Quantity: create.Items[1].Quantity},
	}
	atomicReceipt(t, doContract2JSONRequest(t, router, releasedEdit, "atomic-owner"))

	quantity, nickname, unit = storedItemFields(t, create.Items[0].ID)
	require.Equal(t, int32(7), quantity)
	require.Equal(t, "Front hub", nickname.String, "released client save erased the nickname")
	require.Equal(t, "KT", unit.String, "released client save erased the unit")

	newClientEdit := releasedEdit
	newClientEdit.OperationID = uuid.NewString()
	newClientEdit.Items = []request.NotificationSaveItem{
		{ID: create.Items[0].ID, Niin: create.Items[0].Niin, Nomenclature: create.Items[0].Nomenclature, Quantity: 7, Nickname: itemFieldsPtr(""), UnitOfMeasure: itemFieldsPtr("DZ")},
		{ID: create.Items[1].ID, Niin: create.Items[1].Niin, Nomenclature: create.Items[1].Nomenclature, Quantity: create.Items[1].Quantity, Nickname: itemFieldsPtr(""), UnitOfMeasure: itemFieldsPtr("EA")},
	}
	atomicReceipt(t, doContract2JSONRequest(t, router, newClientEdit, "atomic-owner"))

	_, nickname, unit = storedItemFields(t, create.Items[0].ID)
	require.Equal(t, sql.NullString{String: "", Valid: true}, nickname)
	require.Equal(t, sql.NullString{String: "DZ", Valid: true}, unit)
	// "" and "EA" match the client defaults for a NULL row, so the legacy
	// item is not rewritten and gets no update audit.
	_, legacyNickname, legacyUnit = storedItemFields(t, create.Items[1].ID)
	require.False(t, legacyNickname.Valid)
	require.False(t, legacyUnit.Valid)
	require.Equal(t, 1, atomicRowCount(t, "shop_vehicle_notification_changes",
		"notification_id=$1 AND change_type='items_updated' AND field_changes::text LIKE '%'||$2||'%'", receipt.NotificationID, create.Items[0].ID))
	require.Equal(t, 0, atomicRowCount(t, "shop_vehicle_notification_changes",
		"notification_id=$1 AND change_type='items_updated' AND field_changes::text LIKE '%'||$2||'%'", receipt.NotificationID, create.Items[1].ID))
}

func TestAtomicNotificationRejectsOversizedItemNickname(t *testing.T) {
	router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
	create := atomicNotificationRequest(shopID, vehicleID)
	create.Items[0].Nickname = itemFieldsPtr(strings.Repeat("a", 51))

	rec := doContract2JSONRequest(t, router, create, "atomic-owner")
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	require.Equal(t, 0, atomicRowCount(t, "shop_notification_items", "id=$1", create.Items[0].ID))
}

func TestLegacyNotificationItemAddStoresOptionalFields(t *testing.T) {
	router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
	notificationID := createNotification(t, router, "atomic-owner", shopID, vehicleID, "Legacy add")

	releasedBody := map[string]any{"notification_id": notificationID, "niin": "123456789", "nomenclature": "Filter", "quantity": 1}
	resp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items", releasedBody, "atomic-owner")
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
	released := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	require.Nil(t, released["nickname"])
	require.Nil(t, released["unit_of_measure"])

	newBody := map[string]any{"notification_id": notificationID, "niin": "987654321", "nomenclature": "Seal", "quantity": 3, "nickname": "Rear seal", "unit_of_measure": "DZ"}
	resp = doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items", newBody, "atomic-owner")
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
	created := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	require.Equal(t, "Rear seal", created["nickname"])
	require.Equal(t, "DZ", created["unit_of_measure"])

	invalidBody := map[string]any{"notification_id": notificationID, "niin": "111111111", "nomenclature": "Bad", "quantity": 1, "unit_of_measure": ""}
	resp = doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items", invalidBody, "atomic-owner")
	require.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
}

func TestMaintenanceSnapshotReturnsNotificationItemFields(t *testing.T) {
	router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
	create := atomicNotificationRequest(shopID, vehicleID)
	create.Items = create.Items[:1]
	create.Items[0].Nickname = itemFieldsPtr("Front hub")
	create.Items[0].UnitOfMeasure = itemFieldsPtr("KT")
	atomicReceipt(t, doContract2JSONRequest(t, router, create, "atomic-owner"))

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/"+vehicleID+"/maintenance-snapshot", nil, "atomic-owner")
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	notifications := payload["notifications"].([]interface{})
	require.Len(t, notifications, 1)
	items := notifications[0].(map[string]interface{})["items"].([]interface{})
	require.Len(t, items, 1)
	item := items[0].(map[string]interface{})
	require.Equal(t, "Front hub", item["nickname"])
	require.Equal(t, "KT", item["unit_of_measure"])
}
