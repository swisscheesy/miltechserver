package shops_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/shops/shared"
	notificationitems "miltechserver/api/shops/vehicles/notifications/items"
	"miltechserver/bootstrap"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetadataDeleteReadd(t *testing.T) {
	router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
	notificationID := createNotification(t, router, "atomic-owner", shopID, vehicleID, "Retention")
	body := map[string]any{"notification_id": notificationID, "niin": "123456789", "nomenclature": "Hub", "quantity": 9, "nickname": "Front hub", "unit_of_measure": "KT"}
	added := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items", body, "atomic-owner")
	require.Equal(t, http.StatusCreated, added.Code, added.Body.String())
	removedID := decodeMap(t, decodeStandardResponse(t, added.Body).Data)["id"].(string)
	removed := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/notifications/items/"+removedID, nil, "atomic-owner")
	require.Equal(t, http.StatusOK, removed.Code, removed.Body.String())
	delete(body, "nickname")
	delete(body, "unit_of_measure")
	body["quantity"] = 3
	added = doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items", body, "atomic-owner")
	require.Equal(t, http.StatusCreated, added.Code, added.Body.String())
	replacement := decodeMap(t, decodeStandardResponse(t, added.Body).Data)
	require.NotEqual(t, removedID, replacement["id"])
	require.EqualValues(t, 3, replacement["quantity"])
	require.Equal(t, "Front hub", replacement["nickname"])
	require.Equal(t, "KT", replacement["unit_of_measure"])
	require.Equal(t, 0, atomicRowCount(t, "shop_notification_items", "id=$1", removedID))
}

func TestMetadataReplayAfterLaterEdit(t *testing.T) {
	router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
	original := atomicNotificationRequest(shopID, vehicleID)
	original.Items = original.Items[:1]
	original.Items[0].Nickname = itemFieldsPtr("Front hub")
	original.Items[0].UnitOfMeasure = itemFieldsPtr("KT")
	receipt := atomicReceipt(t, doContract2JSONRequest(t, router, original, "atomic-owner"))
	edit := original
	edit.Items = append([]request.NotificationSaveItem{}, original.Items...)
	edit.NotificationID = &receipt.NotificationID
	edit.OperationID = uuid.NewString()
	edit.Items[0].Nickname = itemFieldsPtr("")
	edit.Items[0].UnitOfMeasure = itemFieldsPtr(" raw-code ")
	atomicReceipt(t, doContract2JSONRequest(t, router, edit, "atomic-owner"))
	replay := atomicReceipt(t, doContract2JSONRequest(t, router, original, "atomic-owner"))
	require.True(t, replay.Replayed)
	require.Equal(t, receipt.CommittedAt, replay.CommittedAt)
	_, nickname, unit := storedItemFields(t, original.Items[0].ID)
	require.Equal(t, "", nickname.String)
	require.Equal(t, " raw-code ", unit.String)
	remove := edit
	remove.OperationID = uuid.NewString()
	remove.Items = []request.NotificationSaveItem{}
	atomicReceipt(t, doContract2JSONRequest(t, router, remove, "atomic-owner"))
	readd := edit
	readd.OperationID = uuid.NewString()
	readd.Items = []request.NotificationSaveItem{{ID: uuid.NewString(), Niin: original.Items[0].Niin, Nomenclature: "Hub", Quantity: 3}}
	atomicReceipt(t, doContract2JSONRequest(t, router, readd, "atomic-owner"))
	q, nickname, unit := storedItemFields(t, readd.Items[0].ID)
	require.EqualValues(t, 3, q)
	require.True(t, nickname.Valid)
	require.Empty(t, nickname.String)
	require.Equal(t, " raw-code ", unit.String)
}

func TestMetadataAmbiguitySurvivesDeletion(t *testing.T) {
	router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
	original := atomicNotificationRequest(shopID, vehicleID)
	original.Items = original.Items[:1]
	original.Items[0].Nickname = itemFieldsPtr("Front hub")
	original.Items[0].UnitOfMeasure = itemFieldsPtr("KT")
	receipt := atomicReceipt(t, doContract2JSONRequest(t, router, original, "atomic-owner"))
	// The deployed unique key includes shop_id; historical corrupt ownership can
	// therefore contain two physical rows for the same notification/exact NIIN.
	conflictID := uuid.NewString()
	_, err := testDB.Exec(`INSERT INTO shop_notification_items(id,shop_id,notification_id,niin,nomenclature,quantity,nickname,unit_of_measure) VALUES($1,'historical-other',$2,$3,'Other',17,'Other hub','EA')`, conflictID, receipt.NotificationID, original.Items[0].Niin)
	require.NoError(t, err)
	body := map[string]any{"notification_id": receipt.NotificationID, "niin": original.Items[0].Niin, "nomenclature": "Hub", "quantity": 3}
	failed := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items", body, "atomic-owner")
	require.Equal(t, http.StatusInternalServerError, failed.Code, failed.Body.String())
	require.Contains(t, failed.Body.String(), "metadata requires explicit resolution")
	var state string
	require.NoError(t, testDB.QueryRow(`SELECT state FROM shop_notification_item_metadata WHERE notification_id=$1 AND niin=$2`, receipt.NotificationID, original.Items[0].Niin).Scan(&state))
	require.Equal(t, "ambiguous", state)
	_, err = testDB.Exec(`DELETE FROM shop_notification_items WHERE id=$1`, conflictID)
	require.NoError(t, err)
	removed := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/notifications/items/"+original.Items[0].ID, nil, "atomic-owner")
	require.Equal(t, http.StatusOK, removed.Code, removed.Body.String())
	require.NoError(t, testDB.QueryRow(`SELECT state FROM shop_notification_item_metadata WHERE notification_id=$1 AND niin=$2`, receipt.NotificationID, original.Items[0].Niin).Scan(&state))
	require.Equal(t, "ambiguous", state)
	failed = doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items", body, "atomic-owner")
	require.Equal(t, http.StatusInternalServerError, failed.Code, failed.Body.String())
	require.Contains(t, failed.Body.String(), "metadata requires explicit resolution")
	body["nickname"] = ""
	body["unit_of_measure"] = "DZ"
	added := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items", body, "atomic-owner")
	require.Equal(t, http.StatusCreated, added.Code, added.Body.String())
	require.NoError(t, testDB.QueryRow(`SELECT state FROM shop_notification_item_metadata WHERE notification_id=$1 AND niin=$2`, receipt.NotificationID, original.Items[0].Niin).Scan(&state))
	require.Equal(t, "resolved", state)
}

func TestMetadataNotificationIsolation(t *testing.T) {
	router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
	original := atomicNotificationRequest(shopID, vehicleID)
	original.Items = original.Items[:1]
	original.Items[0].Nickname = itemFieldsPtr("Front hub")
	original.Items[0].UnitOfMeasure = itemFieldsPtr("KT")
	receipt := atomicReceipt(t, doContract2JSONRequest(t, router, original, "atomic-owner"))
	other := createNotification(t, router, "atomic-owner", shopID, vehicleID, "Other")
	for _, tc := range []struct{ notification, niin string }{{other, original.Items[0].Niin}, {receipt.NotificationID, " " + original.Items[0].Niin}} {
		body := map[string]any{"notification_id": tc.notification, "niin": tc.niin, "nomenclature": "Hub", "quantity": 3}
		added := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items", body, "atomic-owner")
		require.Equal(t, http.StatusCreated, added.Code, added.Body.String())
		row := decodeMap(t, decodeStandardResponse(t, added.Body).Data)
		require.Nil(t, row["nickname"])
		require.Nil(t, row["unit_of_measure"])
	}
}

// observeMetadataConflict deliberately separates rollback from evidence delivery
// so tests can place a real committed interleaving at that boundary.
func observeMetadataConflict(t *testing.T, notificationID, niin string) error {
	t.Helper()
	err := shared.WithNotificationMutation(context.Background(), testDB, func(tx *sql.Tx) error {
		ctx := notificationitems.MetadataMutationContext(context.Background())
		_, _, err := shared.LockNotificationMutation(ctx, tx, "atomic-owner", notificationID)
		if err != nil {
			return err
		}
		_, err = notificationitems.ResolveRetainedMetadata(ctx, tx, notificationitems.MetadataIntent{ItemID: uuid.NewString(), NotificationID: notificationID, Niin: niin})
		return err
	})
	require.Error(t, err)
	require.Equal(t, 409, shared.ClassifyFailure(err).Status)
	return err
}

func TestMetadataEvidenceVersionAndAuthorityFence(t *testing.T) {
	for _, mode := range []string{"later-resolution", "removed-membership", "cancelled-request"} {
		t.Run(mode, func(t *testing.T) {
			router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
			original := atomicNotificationRequest(shopID, vehicleID)
			original.Items = original.Items[:1]
			original.Items[0].Nickname = itemFieldsPtr("Front hub")
			original.Items[0].UnitOfMeasure = itemFieldsPtr("KT")
			receipt := atomicReceipt(t, doContract2JSONRequest(t, router, original, "atomic-owner"))
			conflictID := uuid.NewString()
			_, err := testDB.Exec(`INSERT INTO shop_notification_items(id,shop_id,notification_id,niin,nomenclature,quantity,nickname,unit_of_measure) VALUES($1,'historical-other',$2,$3,'Other',17,'Other hub','EA')`, conflictID, receipt.NotificationID, original.Items[0].Niin)
			require.NoError(t, err)
			observed := observeMetadataConflict(t, receipt.NotificationID, original.Items[0].Niin)
			ctx := context.Background()
			switch mode {
			case "later-resolution":
				_, err = testDB.Exec(`DELETE FROM shop_notification_items WHERE id=$1`, conflictID)
				require.NoError(t, err)
				edit := original
				edit.OperationID = uuid.NewString()
				edit.NotificationID = &receipt.NotificationID
				edit.Items[0].Nickname = itemFieldsPtr("Accepted newer")
				edit.Items[0].UnitOfMeasure = itemFieldsPtr("DZ")
				atomicReceipt(t, doContract2JSONRequest(t, router, edit, "atomic-owner"))
			case "removed-membership":
				_, err = testDB.Exec(`DELETE FROM shop_members WHERE shop_id=$1 AND user_id='atomic-owner'`, shopID)
				require.NoError(t, err)
			case "cancelled-request":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			var before string
			require.NoError(t, testDB.QueryRow(`SELECT row_to_json(m)::text FROM shop_notification_item_metadata m WHERE notification_id=$1`, receipt.NotificationID).Scan(&before))
			notificationitems.PersistMetadataAmbiguity(ctx, testDB, "atomic-owner", observed)
			var after string
			require.NoError(t, testDB.QueryRow(`SELECT row_to_json(m)::text FROM shop_notification_item_metadata m WHERE notification_id=$1`, receipt.NotificationID).Scan(&after))
			require.Equal(t, before, after)
		})
	}
}

func TestMetadataRejectedProspectiveConflictDoesNotPoisonEvidence(t *testing.T) {
	router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
	original := atomicNotificationRequest(shopID, vehicleID)
	original.Items = original.Items[:1]
	original.Items[0].Nickname = itemFieldsPtr("Front hub")
	original.Items[0].UnitOfMeasure = itemFieldsPtr("KT")
	receipt := atomicReceipt(t, doContract2JSONRequest(t, router, original, "atomic-owner"))
	phantomID := uuid.NewString()
	var before string
	require.NoError(t, testDB.QueryRow(`SELECT row_to_json(m)::text FROM shop_notification_item_metadata m WHERE notification_id=$1`, receipt.NotificationID).Scan(&before))
	err := shared.WithNotificationMutation(context.Background(), testDB, func(tx *sql.Tx) error {
		ctx := notificationitems.MetadataMutationContext(context.Background())
		_, _, err := shared.LockNotificationMutation(ctx, tx, "atomic-owner", receipt.NotificationID)
		if err != nil {
			return err
		}
		err = notificationitems.RetainItemMetadata(ctx, tx, model.ShopNotificationItems{ID: original.Items[0].ID, NotificationID: receipt.NotificationID, Niin: original.Items[0].Niin, Nickname: original.Items[0].Nickname, UnitOfMeasure: original.Items[0].UnitOfMeasure})
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO shop_notification_items(id,shop_id,notification_id,niin,nomenclature,quantity,nickname,unit_of_measure) VALUES($1,'prospective-other',$2,$3,'Other',17,'Rejected edit','EA')`, phantomID, receipt.NotificationID, original.Items[0].Niin)
		if err != nil {
			return err
		}
		_, err = notificationitems.ResolveRetainedMetadata(ctx, tx, notificationitems.MetadataIntent{ItemID: uuid.NewString(), NotificationID: receipt.NotificationID, Niin: original.Items[0].Niin})
		return err
	})
	require.Error(t, err)
	require.Equal(t, 409, shared.ClassifyFailure(err).Status)
	notificationitems.PersistMetadataAmbiguity(context.Background(), testDB, "atomic-owner", err)
	var after string
	require.NoError(t, testDB.QueryRow(`SELECT row_to_json(m)::text FROM shop_notification_item_metadata m WHERE notification_id=$1`, receipt.NotificationID).Scan(&after))
	require.Equal(t, before, after)
	require.NotContains(t, after, phantomID)
	require.NotContains(t, after, "Rejected edit")
	require.Equal(t, 0, atomicRowCount(t, "shop_notification_items", "id=$1", phantomID))
}

func TestMetadataAtomicConflictRollsBackReceiptAndWrites(t *testing.T) {
	router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
	original := atomicNotificationRequest(shopID, vehicleID)
	original.Items = original.Items[:1]
	original.Items[0].Nickname = itemFieldsPtr("Front hub")
	original.Items[0].UnitOfMeasure = itemFieldsPtr("KT")
	receipt := atomicReceipt(t, doContract2JSONRequest(t, router, original, "atomic-owner"))
	conflictID := uuid.NewString()
	_, err := testDB.Exec(`INSERT INTO shop_notification_items(id,shop_id,notification_id,niin,nomenclature,quantity,nickname,unit_of_measure) VALUES($1,'historical-other',$2,$3,'Other',17,'Other hub','EA')`, conflictID, receipt.NotificationID, original.Items[0].Niin)
	require.NoError(t, err)
	edit := original
	edit.OperationID = uuid.NewString()
	edit.NotificationID = &receipt.NotificationID
	edit.Details.Title = "Rejected title"
	edit.Items = []request.NotificationSaveItem{{ID: uuid.NewString(), Niin: original.Items[0].Niin, Nomenclature: "Hub", Quantity: 3}}
	audits := atomicRowCount(t, "shop_vehicle_notification_changes", "notification_id=$1", receipt.NotificationID)
	failed := doContract2JSONRequest(t, router, edit, "atomic-owner")
	require.Equal(t, http.StatusConflict, failed.Code, failed.Body.String())
	require.Contains(t, failed.Body.String(), "notification_item_metadata_conflict")
	require.Equal(t, 0, atomicRowCount(t, "shop_notification_operations", "operation_id=$1", edit.OperationID))
	require.Equal(t, audits, atomicRowCount(t, "shop_vehicle_notification_changes", "notification_id=$1", receipt.NotificationID))
	require.Equal(t, 2, atomicRowCount(t, "shop_notification_items", "notification_id=$1", receipt.NotificationID))
	var state, candidates, title string
	require.NoError(t, testDB.QueryRow(`SELECT state,candidates::text FROM shop_notification_item_metadata WHERE notification_id=$1`, receipt.NotificationID).Scan(&state, &candidates))
	require.Equal(t, "ambiguous", state)
	require.Contains(t, candidates, conflictID)
	require.NotContains(t, candidates, edit.Items[0].ID)
	require.NoError(t, testDB.QueryRow(`SELECT title FROM shop_vehicle_notifications WHERE id=$1`, receipt.NotificationID).Scan(&title))
	require.Equal(t, original.Details.Title, title)
	// Same legacy route with negotiated v2 returns the typed conflict envelope.
	body, _ := json.Marshal(map[string]any{"notification_id": receipt.NotificationID, "niin": original.Items[0].Niin, "nomenclature": "Hub", "quantity": 3})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/shops/notifications/items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "atomic-owner")
	req.Header.Set(shared.ContractHeader, "2")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
}

func TestMetadataRawNullAndDefaultRetention(t *testing.T) {
	for _, tc := range []struct {
		name           string
		nickname, unit *string
	}{{"null", nil, nil}, {"default", itemFieldsPtr(""), itemFieldsPtr("EA")}, {"unknown", itemFieldsPtr(""), itemFieldsPtr(" raw-code ")}} {
		t.Run(tc.name, func(t *testing.T) {
			router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
			n := createNotification(t, router, "atomic-owner", shopID, vehicleID, "Raw")
			repo := notificationitems.NewRepository(testDB)
			ctx := context.Background()
			user := &bootstrap.User{UserID: "atomic-owner"}
			first, err := repo.CreateNotificationItem(ctx, user, model.ShopNotificationItems{ID: uuid.NewString(), ShopID: shopID, NotificationID: n, Niin: "raw", Nomenclature: "Hub", Quantity: 99, Nickname: tc.nickname, UnitOfMeasure: tc.unit})
			require.NoError(t, err)
			require.NoError(t, repo.DeleteNotificationItem(ctx, user, first.ID))
			next, err := repo.CreateNotificationItem(ctx, user, model.ShopNotificationItems{ID: uuid.NewString(), ShopID: shopID, NotificationID: n, Niin: "raw", Nomenclature: "Hub", Quantity: 3})
			require.NoError(t, err)
			require.Equal(t, tc.nickname, next.Nickname)
			require.Equal(t, tc.unit, next.UnitOfMeasure)
			require.EqualValues(t, 3, next.Quantity)
			require.NotEqual(t, first.ID, next.ID)
			_, err = testDB.Exec(`DELETE FROM shop_vehicle_notifications WHERE id=$1`, n)
			require.NoError(t, err)
			require.Equal(t, 0, atomicRowCount(t, "shop_notification_item_metadata", "notification_id=$1", n))
		})
	}
}

func TestMetadataSamePhysicalRowNeverBorrowsSibling(t *testing.T) {
	router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
	original := atomicNotificationRequest(shopID, vehicleID)
	original.Items = original.Items[:1]
	receipt := atomicReceipt(t, doContract2JSONRequest(t, router, original, "atomic-owner"))
	sibling := uuid.NewString()
	_, err := testDB.Exec(`INSERT INTO shop_notification_items(id,shop_id,notification_id,niin,nomenclature,quantity,nickname,unit_of_measure) VALUES($1,'historical-other',$2,$3,'Other',17,'Sibling only','KT')`, sibling, receipt.NotificationID, original.Items[0].Niin)
	require.NoError(t, err)
	edit := original
	edit.OperationID = uuid.NewString()
	edit.NotificationID = &receipt.NotificationID
	edit.Items = append([]request.NotificationSaveItem{}, original.Items...)
	edit.Items[0].Quantity = 3
	edit.Items = append(edit.Items, request.NotificationSaveItem{ID: sibling, Niin: original.Items[0].Niin, Nomenclature: "Other", Quantity: 17})
	atomicReceipt(t, doContract2JSONRequest(t, router, edit, "atomic-owner"))
	quantity, nickname, unit := storedItemFields(t, original.Items[0].ID)
	require.EqualValues(t, 3, quantity)
	require.False(t, nickname.Valid)
	require.False(t, unit.Valid)
	_, nickname, unit = storedItemFields(t, sibling)
	require.Equal(t, "Sibling only", nickname.String)
	require.Equal(t, "KT", unit.String)
	var state string
	require.NoError(t, testDB.QueryRow(`SELECT state FROM shop_notification_item_metadata WHERE notification_id=$1`, receipt.NotificationID).Scan(&state))
	require.Equal(t, "ambiguous", state)
}

func TestMetadataAlternatingBulkAndAtomic(t *testing.T) {
	router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
	n := createNotification(t, router, "atomic-owner", shopID, vehicleID, "Alternating")
	repo := notificationitems.NewRepository(testDB)
	ctx := context.Background()
	user := &bootstrap.User{UserID: "atomic-owner"}
	created, err := repo.CreateNotificationItemList(ctx, user, []model.ShopNotificationItems{{ID: uuid.NewString(), ShopID: shopID, NotificationID: n, Niin: "bulk", Nomenclature: "Hub", Quantity: 99, Nickname: itemFieldsPtr("Front hub"), UnitOfMeasure: itemFieldsPtr("KT")}})
	require.NoError(t, err)
	_, err = repo.DeleteNotificationItemList(ctx, user, []string{created[0].ID})
	require.NoError(t, err)
	atomic := atomicNotificationRequest(shopID, vehicleID)
	atomic.NotificationID = &n
	atomic.Items = []request.NotificationSaveItem{{ID: uuid.NewString(), Niin: "bulk", Nomenclature: "Hub", Quantity: 3}}
	atomicReceipt(t, doContract2JSONRequest(t, router, atomic, "atomic-owner"))
	q, nickname, unit := storedItemFields(t, atomic.Items[0].ID)
	require.EqualValues(t, 3, q)
	require.Equal(t, "Front hub", nickname.String)
	require.Equal(t, "KT", unit.String)
	edit := atomic
	edit.OperationID = uuid.NewString()
	edit.Items = append([]request.NotificationSaveItem{}, atomic.Items...)
	edit.Items[0].Nickname = itemFieldsPtr("")
	edit.Items[0].UnitOfMeasure = itemFieldsPtr("DZ")
	atomicReceipt(t, doContract2JSONRequest(t, router, edit, "atomic-owner"))
	_, err = repo.DeleteNotificationItemList(ctx, user, []string{atomic.Items[0].ID})
	require.NoError(t, err)
	next, err := repo.CreateNotificationItemList(ctx, user, []model.ShopNotificationItems{{ID: uuid.NewString(), ShopID: shopID, NotificationID: n, Niin: "bulk", Nomenclature: "Hub", Quantity: 3}})
	require.NoError(t, err)
	require.Equal(t, "", *next[0].Nickname)
	require.Equal(t, "DZ", *next[0].UnitOfMeasure)
	require.EqualValues(t, 3, next[0].Quantity)
	require.NotEqual(t, created[0].ID, next[0].ID)
	require.NotEqual(t, atomic.Items[0].ID, next[0].ID)
}

func TestMetadataItemRepositoryCancellation(t *testing.T) {
	_, shopID, vehicleID := atomicFixture(t, "atomic-owner")
	repo := notificationitems.NewRepository(testDB)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	user := &bootstrap.User{UserID: "atomic-owner"}
	cases := []struct {
		name string
		call func() error
	}{
		{"items", func() error { _, err := repo.GetNotificationItems(ctx, user, "missing"); return err }},
		{"shop-items", func() error { _, err := repo.GetShopNotificationItems(ctx, user, shopID); return err }},
		{"item", func() error { _, err := repo.GetNotificationItemByID(ctx, user, "missing"); return err }},
		{"items-by-id", func() error { _, err := repo.GetNotificationItemsByIDs(ctx, user, []string{"missing"}); return err }},
		{"notification", func() error { _, err := repo.GetVehicleNotificationByID(ctx, user, "missing"); return err }},
		{"vehicle", func() error { _, err := repo.GetShopVehicleByID(ctx, user, vehicleID); return err }},
		{"membership", func() error { _, err := repo.IsUserMemberOfShop(ctx, user, shopID); return err }},
		{"audit", func() error { return repo.CreateNotificationChange(ctx, user, model.ShopVehicleNotificationChanges{}) }},
		{"create", func() error {
			_, err := repo.CreateNotificationItem(ctx, user, model.ShopNotificationItems{ID: uuid.NewString()})
			return err
		}},
		{"create-bulk", func() error {
			_, err := repo.CreateNotificationItemList(ctx, user, []model.ShopNotificationItems{{ID: uuid.NewString()}})
			return err
		}},
		{"delete", func() error { return repo.DeleteNotificationItem(ctx, user, "missing") }},
		{"delete-bulk", func() error { _, err := repo.DeleteNotificationItemList(ctx, user, []string{"missing"}); return err }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { require.True(t, errors.Is(tc.call(), context.Canceled)) })
	}
}

func TestMetadataDelayedEvidenceSurvivesOrdinaryWrites(t *testing.T) {
	for _, mode := range []string{"delete", "quantity-only"} {
		t.Run(mode, func(t *testing.T) {
			router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
			original := atomicNotificationRequest(shopID, vehicleID)
			original.Items = original.Items[:1]
			original.Items[0].Nickname = itemFieldsPtr("Front hub")
			original.Items[0].UnitOfMeasure = itemFieldsPtr("KT")
			receipt := atomicReceipt(t, doContract2JSONRequest(t, router, original, "atomic-owner"))
			conflictID := uuid.NewString()
			_, err := testDB.Exec(`INSERT INTO shop_notification_items(id,shop_id,notification_id,niin,nomenclature,quantity,nickname,unit_of_measure) VALUES($1,'historical-other',$2,$3,'Other',17,'Other hub','EA')`, conflictID, receipt.NotificationID, original.Items[0].Niin)
			require.NoError(t, err)
			auditsBefore := atomicRowCount(t, "shop_vehicle_notification_changes", "notification_id=$1", receipt.NotificationID)
			operationsBefore := atomicRowCount(t, "shop_notification_operations", "notification_id=$1", receipt.NotificationID)
			var observedVersion int64
			require.NoError(t, testDB.QueryRow(`SELECT version FROM shop_notification_item_metadata WHERE notification_id=$1`, receipt.NotificationID).Scan(&observedVersion))
			observed := observeMetadataConflict(t, receipt.NotificationID, original.Items[0].Niin)
			require.Equal(t, 2, atomicRowCount(t, "shop_notification_items", "notification_id=$1", receipt.NotificationID))
			require.Equal(t, auditsBefore, atomicRowCount(t, "shop_vehicle_notification_changes", "notification_id=$1", receipt.NotificationID))
			require.Equal(t, operationsBefore, atomicRowCount(t, "shop_notification_operations", "notification_id=$1", receipt.NotificationID))
			// This committed conflict was observed before rollback. Its later physical
			// disappearance is not an accepted explicit metadata resolution.
			_, err = testDB.Exec(`DELETE FROM shop_notification_items WHERE id=$1`, conflictID)
			require.NoError(t, err)
			switch mode {
			case "delete":
				removed := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/notifications/items/"+original.Items[0].ID, nil, "atomic-owner")
				require.Equal(t, http.StatusOK, removed.Code, removed.Body.String())
			case "quantity-only":
				edit := original
				edit.OperationID = uuid.NewString()
				edit.NotificationID = &receipt.NotificationID
				edit.Items = []request.NotificationSaveItem{{ID: original.Items[0].ID, Niin: original.Items[0].Niin, Nomenclature: original.Items[0].Nomenclature, Quantity: 3}}
				atomicReceipt(t, doContract2JSONRequest(t, router, edit, "atomic-owner"))
				quantity, nickname, unit := storedItemFields(t, original.Items[0].ID)
				require.EqualValues(t, 3, quantity)
				require.Equal(t, "Front hub", nickname.String)
				require.Equal(t, "KT", unit.String)
			}
			auditCount := atomicRowCount(t, "shop_vehicle_notification_changes", "notification_id=$1", receipt.NotificationID)
			operationCount := atomicRowCount(t, "shop_notification_operations", "notification_id=$1", receipt.NotificationID)
			itemCount := atomicRowCount(t, "shop_notification_items", "notification_id=$1", receipt.NotificationID)
			var ordinaryVersion, resolutionVersion int64
			require.NoError(t, testDB.QueryRow(`SELECT version,resolution_version FROM shop_notification_item_metadata WHERE notification_id=$1`, receipt.NotificationID).Scan(&ordinaryVersion, &resolutionVersion))
			require.Greater(t, ordinaryVersion, observedVersion)
			require.LessOrEqual(t, resolutionVersion, observedVersion)
			notificationitems.PersistMetadataAmbiguity(context.Background(), testDB, "atomic-owner", observed)
			var state, candidates string
			require.NoError(t, testDB.QueryRow(`SELECT state,candidates::text FROM shop_notification_item_metadata WHERE notification_id=$1`, receipt.NotificationID).Scan(&state, &candidates))
			require.Equal(t, "ambiguous", state, "ordinary writes must not supersede observed committed conflict")
			require.Contains(t, candidates, conflictID)
			require.Contains(t, candidates, "Other hub")
			require.Contains(t, candidates, original.Items[0].ID)
			require.Equal(t, auditCount, atomicRowCount(t, "shop_vehicle_notification_changes", "notification_id=$1", receipt.NotificationID))
			require.Equal(t, operationCount, atomicRowCount(t, "shop_notification_operations", "notification_id=$1", receipt.NotificationID))
			require.Equal(t, itemCount, atomicRowCount(t, "shop_notification_items", "notification_id=$1", receipt.NotificationID))
			if mode == "quantity-only" {
				removed := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/notifications/items/"+original.Items[0].ID, nil, "atomic-owner")
				require.Equal(t, http.StatusOK, removed.Code, removed.Body.String())
			}
			body := map[string]any{"notification_id": receipt.NotificationID, "niin": original.Items[0].Niin, "nomenclature": "Hub", "quantity": 3}
			rejected := doContractRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items", body, "atomic-owner")
			require.Equal(t, http.StatusConflict, rejected.Code, rejected.Body.String())
			require.Contains(t, rejected.Body.String(), "notification_item_metadata_conflict")
			require.Equal(t, 0, atomicRowCount(t, "shop_notification_items", "notification_id=$1", receipt.NotificationID))
		})
	}
}
