package shops_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"miltechserver/api/request"
	"miltechserver/api/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const atomicSavePath = "/api/v1/auth/shops/vehicles/notifications/save"

func atomicNotificationRequest(shopID, vehicleID string) request.NotificationSaveRequest {
	return request.NotificationSaveRequest{
		OperationID: uuid.NewString(),
		ShopID:      shopID,
		VehicleID:   vehicleID,
		Details: request.NotificationSaveDetails{
			Title:       "Inspect equipment",
			Description: "Check the pump",
			Type:        "PM",
		},
		Attachment: request.NotificationSaveAttachment{Intent: "keep"},
		Items: []request.NotificationSaveItem{
			{ID: uuid.NewString(), Niin: "1234", Nomenclature: "Pump", Quantity: 2},
			{ID: uuid.NewString(), Niin: "5678", Nomenclature: "Seal", Quantity: 1},
		},
	}
}

func doContract2JSONRequest(t *testing.T, router *gin.Engine, body request.NotificationSaveRequest, userID string) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, atomicSavePath, strings.NewReader(string(payload)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-MilTech-Shops-Contract", "2")
	if userID != "" {
		req.Header.Set("X-User-ID", userID)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func atomicReceipt(t *testing.T, rec *httptest.ResponseRecorder) response.NotificationSaveReceipt {
	t.Helper()
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	envelope := decodeStandardResponse(t, rec.Body)
	require.Equal(t, http.StatusOK, envelope.Status)
	var receipt response.NotificationSaveReceipt
	require.NoError(t, json.Unmarshal(envelope.Data, &receipt))
	return receipt
}

func atomicRowCount(t *testing.T, table, where string, args ...any) int {
	t.Helper()
	var count int
	require.NoError(t, testDB.QueryRow("SELECT count(*) FROM "+table+" WHERE "+where, args...).Scan(&count))
	return count
}

func atomicFixture(t *testing.T, userID string) (*gin.Engine, string, string) {
	t.Helper()
	clearShopTables(t, testDB)
	ensureUser(t, testDB, userID)
	router := newTestRouter(t)
	shopID := createShop(t, router, userID, "Atomic notification")
	vehicleID := createVehicle(t, router, userID, shopID)
	return router, shopID, vehicleID
}

func TestAtomicNotificationCreateCommitsCompleteSet(t *testing.T) {
	router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
	request := atomicNotificationRequest(shopID, vehicleID)

	receipt := atomicReceipt(t, doContract2JSONRequest(t, router, request, "atomic-owner"))
	require.Equal(t, request.OperationID, receipt.OperationID)
	require.NotEmpty(t, receipt.NotificationID)
	require.False(t, receipt.CommittedAt.IsZero())
	require.False(t, receipt.Replayed)
	require.Equal(t, 1, atomicRowCount(t, "shop_vehicle_notifications", "id=$1 AND shop_id=$2 AND vehicle_id=$3 AND title=$4", receipt.NotificationID, shopID, vehicleID, request.Details.Title))
	require.Equal(t, len(request.Items), atomicRowCount(t, "shop_notification_items", "notification_id=$1", receipt.NotificationID))
	for _, item := range request.Items {
		require.Equal(t, 1, atomicRowCount(t, "shop_notification_items", "id=$1 AND notification_id=$2 AND niin=$3 AND nomenclature=$4 AND quantity=$5", item.ID, receipt.NotificationID, item.Niin, item.Nomenclature, item.Quantity))
	}
	require.Equal(t, 2, atomicRowCount(t, "shop_vehicle_notification_changes", "notification_id=$1", receipt.NotificationID))
	require.Equal(t, 1, atomicRowCount(t, "shop_vehicle_notification_changes", "notification_id=$1 AND change_type='create' AND changed_by=$2", receipt.NotificationID, "atomic-owner"))
	require.Equal(t, 1, atomicRowCount(t, "shop_vehicle_notification_changes", "notification_id=$1 AND change_type='items_added' AND changed_by=$2", receipt.NotificationID, "atomic-owner"))
	require.Equal(t, 1, atomicRowCount(t, "shop_notification_operations", "user_id=$1 AND operation_id=$2 AND notification_id=$3 AND committed_at=$4", "atomic-owner", receipt.OperationID, receipt.NotificationID, receipt.CommittedAt))
}

func TestAtomicNotificationConcurrentReplayAndConflict(t *testing.T) {
	router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
	r := atomicNotificationRequest(shopID, vehicleID)
	responses := make([]*httptest.ResponseRecorder, 2)
	start := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(2)
	for i := range responses {
		go func(index int) {
			defer workers.Done()
			<-start
			responses[index] = doContract2JSONRequest(t, router, r, "atomic-owner")
		}(i)
	}
	close(start)
	workers.Wait()
	first := atomicReceipt(t, responses[0])
	second := atomicReceipt(t, responses[1])
	require.Equal(t, first.OperationID, second.OperationID)
	require.Equal(t, first.NotificationID, second.NotificationID)
	require.Equal(t, first.CommittedAt, second.CommittedAt)
	require.NotEqual(t, first.Replayed, second.Replayed)
	require.Equal(t, 1, atomicRowCount(t, "shop_vehicle_notifications", "id=$1", first.NotificationID))
	require.Equal(t, 2, atomicRowCount(t, "shop_notification_items", "notification_id=$1", first.NotificationID))
	require.Equal(t, 2, atomicRowCount(t, "shop_vehicle_notification_changes", "notification_id=$1", first.NotificationID))
	require.Equal(t, 1, atomicRowCount(t, "shop_notification_operations", "user_id=$1 AND operation_id=$2", "atomic-owner", r.OperationID))

	conflicting := r
	conflicting.Details.Title = "Different submitted title"
	rec := doContract2JSONRequest(t, router, conflicting, "atomic-owner")
	require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	require.Equal(t, "operation_payload_conflict", decodeMap(t, json.RawMessage(rec.Body.Bytes()))["code"])
	require.Equal(t, 1, atomicRowCount(t, "shop_vehicle_notifications", "id=$1 AND title=$2", first.NotificationID, r.Details.Title))
	require.Equal(t, 2, atomicRowCount(t, "shop_vehicle_notification_changes", "notification_id=$1", first.NotificationID))
}

func TestAtomicNotificationLostResponseAndLaterChangesReplayReceiptOnly(t *testing.T) {
	router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
	create := atomicNotificationRequest(shopID, vehicleID)
	// The first HTTP response is deliberately discarded after ServeHTTP returns.
	lostResponse := doContract2JSONRequest(t, router, create, "atomic-owner")
	require.Equal(t, http.StatusOK, lostResponse.Code, lostResponse.Body.String())
	replay := atomicReceipt(t, doContract2JSONRequest(t, router, create, "atomic-owner"))
	require.True(t, replay.Replayed)
	require.Equal(t, create.OperationID, replay.OperationID)
	require.Equal(t, 1, atomicRowCount(t, "shop_vehicle_notifications", "id=$1", replay.NotificationID))

	edit := atomicNotificationRequest(shopID, vehicleID)
	edit.NotificationID = &replay.NotificationID
	edit.Details.Title = "Another editor's title"
	edit.Items = []request.NotificationSaveItem{}
	editReceipt := atomicReceipt(t, doContract2JSONRequest(t, router, edit, "atomic-owner"))
	require.False(t, editReceipt.Replayed)
	before := atomicRowCount(t, "shop_vehicle_notification_changes", "notification_id=$1", replay.NotificationID)
	afterEditReplay := atomicReceipt(t, doContract2JSONRequest(t, router, create, "atomic-owner"))
	require.True(t, afterEditReplay.Replayed)
	require.Equal(t, replay.OperationID, afterEditReplay.OperationID)
	require.Equal(t, replay.NotificationID, afterEditReplay.NotificationID)
	require.Equal(t, replay.CommittedAt, afterEditReplay.CommittedAt)
	require.Equal(t, 1, atomicRowCount(t, "shop_vehicle_notifications", "id=$1 AND title=$2", replay.NotificationID, edit.Details.Title))
	require.Zero(t, atomicRowCount(t, "shop_notification_items", "notification_id=$1", replay.NotificationID))
	require.Equal(t, before, atomicRowCount(t, "shop_vehicle_notification_changes", "notification_id=$1", replay.NotificationID))

	deleted := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/vehicles/notifications/"+replay.NotificationID, nil, "atomic-owner")
	require.Equal(t, http.StatusOK, deleted.Code, deleted.Body.String())
	afterDeleteReplay := atomicReceipt(t, doContract2JSONRequest(t, router, create, "atomic-owner"))
	require.True(t, afterDeleteReplay.Replayed)
	require.Equal(t, replay.CommittedAt, afterDeleteReplay.CommittedAt)
	require.Zero(t, atomicRowCount(t, "shop_vehicle_notifications", "id=$1", replay.NotificationID))
	require.Zero(t, atomicRowCount(t, "shop_notification_items", "notification_id=$1", replay.NotificationID))
	require.Equal(t, 2, atomicRowCount(t, "shop_notification_operations", "user_id=$1", "atomic-owner"))
}

func TestAtomicNotificationRemovedMemberCannotSaveOrReplay(t *testing.T) {
	router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
	ensureUser(t, testDB, "atomic-member")
	_, err := testDB.Exec(`INSERT INTO shop_members (id,shop_id,user_id,role,joined_at) VALUES ($1,$2,$3,'member',now())`, uuid.NewString(), shopID, "atomic-member")
	require.NoError(t, err)
	r := atomicNotificationRequest(shopID, vehicleID)
	receipt := atomicReceipt(t, doContract2JSONRequest(t, router, r, "atomic-member"))
	_, err = testDB.Exec(`DELETE FROM shop_members WHERE shop_id=$1 AND user_id=$2`, shopID, "atomic-member")
	require.NoError(t, err)

	fresh := atomicNotificationRequest(shopID, vehicleID)
	for _, submitted := range []request.NotificationSaveRequest{r, fresh} {
		rec := doContract2JSONRequest(t, router, submitted, "atomic-member")
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	}
	require.Equal(t, 1, atomicRowCount(t, "shop_vehicle_notifications", "id=$1", receipt.NotificationID))
	require.Equal(t, 1, atomicRowCount(t, "shop_notification_operations", "user_id=$1", "atomic-member"))
}

func TestAtomicNotificationRequiresAuthenticatedUser(t *testing.T) {
	router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
	r := atomicNotificationRequest(shopID, vehicleID)
	rec := doContract2JSONRequest(t, router, r, "")
	require.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
	require.Zero(t, atomicRowCount(t, "shop_notification_operations", "operation_id=$1", r.OperationID))
	require.Zero(t, atomicRowCount(t, "shop_vehicle_notifications", "shop_id=$1", shopID))
}

func TestAtomicNotificationRejectsWrongShopReferences(t *testing.T) {
	router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
	otherShop := createShop(t, router, "atomic-owner", "Other shop")
	otherVehicle := createVehicle(t, router, "atomic-owner", otherShop)
	otherList := createList(t, router, "atomic-owner", otherShop)
	foreignItem := atomicNotificationRequest(otherShop, otherVehicle)
	foreignReceipt := atomicReceipt(t, doContract2JSONRequest(t, router, foreignItem, "atomic-owner"))
	require.Equal(t, 1, atomicRowCount(t, "shop_notification_items", "id=$1 AND notification_id=$2", foreignItem.Items[0].ID, foreignReceipt.NotificationID))

	for _, tc := range []struct {
		name   string
		mutate func(*request.NotificationSaveRequest)
		status int
	}{
		{"vehicle", func(r *request.NotificationSaveRequest) { r.VehicleID = otherVehicle }, http.StatusForbidden},
		{"list", func(r *request.NotificationSaveRequest) {
			r.Attachment.Intent = "attach"
			r.Attachment.ListID = &otherList
		}, http.StatusForbidden},
		{"item", func(r *request.NotificationSaveRequest) { r.Items[0].ID = foreignItem.Items[0].ID }, http.StatusBadRequest},
		{"edit target", func(r *request.NotificationSaveRequest) { r.NotificationID = &foreignReceipt.NotificationID }, http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := atomicNotificationRequest(shopID, vehicleID)
			tc.mutate(&r)
			rec := doContract2JSONRequest(t, router, r, "atomic-owner")
			require.Equal(t, tc.status, rec.Code, rec.Body.String())
			require.Zero(t, atomicRowCount(t, "shop_notification_operations", "operation_id=$1", r.OperationID))
			require.Zero(t, atomicRowCount(t, "shop_vehicle_notifications", "shop_id=$1", shopID))
		})
	}
	require.Equal(t, 1, atomicRowCount(t, "shop_notification_items", "id=$1 AND notification_id=$2", foreignItem.Items[0].ID, foreignReceipt.NotificationID))
}

func TestAtomicNotificationInjectedWriteFailuresRollBackAllRows(t *testing.T) {
	router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
	_, err := testDB.Exec(`CREATE FUNCTION test_infrastructure.fail_atomic_notification_write()
		RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected notification write failure'; END $$`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, dropErr := testDB.Exec(`DROP FUNCTION test_infrastructure.fail_atomic_notification_write()`)
		require.NoError(t, dropErr)
	})
	for _, tc := range []struct {
		name, event, table string
	}{
		{"items", "BEFORE INSERT", "shop_notification_items"},
		{"audit", "BEFORE INSERT", "shop_vehicle_notification_changes"},
		{"receipt", "BEFORE UPDATE OF committed_at", "shop_notification_operations"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			trigger := "fail_atomic_notification_" + tc.name
			_, err := testDB.Exec("CREATE TRIGGER " + trigger + " " + tc.event + " ON public." + tc.table + " FOR EACH ROW EXECUTE FUNCTION test_infrastructure.fail_atomic_notification_write()")
			require.NoError(t, err)
			t.Cleanup(func() {
				_, dropErr := testDB.Exec("DROP TRIGGER " + trigger + " ON public." + tc.table)
				require.NoError(t, dropErr)
			})
			r := atomicNotificationRequest(shopID, vehicleID)
			rec := doContract2JSONRequest(t, router, r, "atomic-owner")
			require.GreaterOrEqual(t, rec.Code, 500, rec.Body.String())
			require.Less(t, rec.Code, 600, rec.Body.String())
			require.NotContains(t, rec.Body.String(), "injected notification write failure")
			require.Zero(t, atomicRowCount(t, "shop_vehicle_notifications", "shop_id=$1", shopID))
			require.Zero(t, atomicRowCount(t, "shop_notification_items", "shop_id=$1", shopID))
			require.Zero(t, atomicRowCount(t, "shop_vehicle_notification_changes", "shop_id=$1", shopID))
			require.Zero(t, atomicRowCount(t, "shop_notification_operations", "operation_id=$1", r.OperationID))
		})
	}
}
