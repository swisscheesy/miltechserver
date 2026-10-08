package shops_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"log/slog"
	"miltechserver/api/shops/vehicles/notifications/changes"
	"miltechserver/bootstrap"
	"net/http"
	"strings"
	"testing"
	"time"
)

func auditSnapshots(t *testing.T, notificationID, kind string) (int, map[string]map[string]any) {
	t.Helper()
	rows, err := testDB.Query(`SELECT field_changes FROM shop_vehicle_notification_changes WHERE notification_id=$1 AND change_type=$2`, notificationID, kind)
	require.NoError(t, err)
	defer rows.Close()
	snapshots := map[string]map[string]any{}
	events := 0
	for rows.Next() {
		var raw string
		require.NoError(t, rows.Scan(&raw))
		var p map[string]any
		require.NoError(t, json.Unmarshal([]byte(raw), &p))
		require.Equal(t, []any{"items"}, p["fields_changed"])
		entries := p[kind].([]any)
		require.Equal(t, float64(len(entries)), p["item_count"])
		events++
		for _, v := range entries {
			snapshot := v.(map[string]any)
			id, ok := snapshot["item_id"].(string)
			require.True(t, ok, "stable physical identity missing")
			require.NotContains(t, snapshots, id)
			snapshots[id] = snapshot
		}
	}
	require.NoError(t, rows.Err())
	return events, snapshots
}

func TestLegacyAuditEnrichedRemovalSnapshot(t *testing.T) {
	for _, bulk := range []bool{false, true} {
		name := "single"
		if bulk {
			name = "bulk"
		}
		t.Run(name, func(t *testing.T) {
			r, shop, vehicle := atomicFixture(t, "atomic-owner")
			n := createNotification(t, r, "atomic-owner", shop, vehicle, "Event title")
			inputs := []map[string]any{
				{"notification_id": n, "niin": "first", "nomenclature": "Seal", "quantity": 3, "nickname": "", "unit_of_measure": "DZ"},
				{"notification_id": n, "niin": "raw", "nomenclature": "Raw", "quantity": 4, "nickname": "Raw nickname", "unit_of_measure": " raw-unit "},
				{"notification_id": n, "niin": "null", "nomenclature": "Null", "quantity": 5},
			}
			ids := []string{}
			if bulk {
				rec := doJSONRequest(t, r, http.MethodPost, "/api/v1/auth/shops/notifications/items/bulk", map[string]any{"notification_id": n, "items": inputs}, "atomic-owner")
				require.Equal(t, 201, rec.Code, rec.Body.String())
				var data []map[string]any
				require.NoError(t, json.Unmarshal(decodeStandardResponse(t, rec.Body).Data, &data))
				for _, entry := range data {
					ids = append(ids, entry["id"].(string))
				}
			} else {
				for _, input := range inputs {
					rec := doJSONRequest(t, r, http.MethodPost, "/api/v1/auth/shops/notifications/items", input, "atomic-owner")
					require.Equal(t, 201, rec.Code, rec.Body.String())
					ids = append(ids, decodeMap(t, decodeStandardResponse(t, rec.Body).Data)["id"].(string))
				}
			}
			addedEvents, added := auditSnapshots(t, n, "items_added")
			if bulk {
				assertBatchRemovalCount(t, doJSONRequest(t, r, http.MethodDelete, "/api/v1/auth/shops/notifications/items/bulk", map[string]any{"item_ids": ids}, "atomic-owner"), 3)
			} else {
				for _, id := range ids {
					rec := doJSONRequest(t, r, http.MethodDelete, "/api/v1/auth/shops/notifications/items/"+id, nil, "atomic-owner")
					require.Equal(t, 200, rec.Code, rec.Body.String())
				}
			}
			events, snapshots := auditSnapshots(t, n, "items_removed")
			if bulk {
				require.Equal(t, 1, events)
				require.Equal(t, 1, addedEvents)
			} else {
				require.Equal(t, 3, events)
				require.Equal(t, 3, addedEvents)
			}
			require.Equal(t, added, snapshots)
			require.Len(t, snapshots, 3)
			require.Equal(t, map[string]any{"item_id": ids[0], "niin": "first", "nomenclature": "Seal", "quantity": float64(3), "nickname": "", "unit_of_measure": "DZ"}, snapshots[ids[0]])
			require.Equal(t, " raw-unit ", snapshots[ids[1]]["unit_of_measure"])
			require.Equal(t, "Raw nickname", snapshots[ids[1]]["nickname"])
			require.Contains(t, snapshots[ids[2]], "nickname")
			require.Nil(t, snapshots[ids[2]]["nickname"])
			require.Contains(t, snapshots[ids[2]], "unit_of_measure")
			require.Nil(t, snapshots[ids[2]]["unit_of_measure"])
			require.Zero(t, atomicRowCount(t, "shop_notification_items", "notification_id=$1", n))
		})
	}
}

func TestHistoryPrefersEventLabels(t *testing.T) {
	r, shop, vehicle := atomicFixture(t, "atomic-owner")
	n := createNotification(t, r, "atomic-owner", shop, vehicle, "Original title")
	_, err := testDB.Exec(`UPDATE shop_vehicle_notifications SET title='Current title' WHERE id=$1`, n)
	require.NoError(t, err)
	_, err = testDB.Exec(`UPDATE shop_vehicle SET admin='Current admin' WHERE id=$1`, vehicle)
	require.NoError(t, err)
	_, err = testDB.Exec(`DELETE FROM shop_vehicle_notification_changes WHERE notification_id=$1`, n)
	require.NoError(t, err)
	at := time.Now().UTC()
	ids := []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"}
	for i, id := range ids {
		var title, admin any = "Event title", "Event admin"
		if i == 1 {
			title = nil
			admin = nil
		}
		_, err = testDB.Exec(`INSERT INTO shop_vehicle_notification_changes(id,notification_id,shop_id,vehicle_id,changed_by,changed_at,change_type,field_changes,notification_title,notification_type,vehicle_admin) VALUES($1,$2,$3,$4,'atomic-owner',$5,'update','{}',$6,'M1',$7)`, id, n, shop, vehicle, at, title, admin)
		require.NoError(t, err)
	}
	repo := changes.NewRepository(testDB)
	user := &bootstrap.User{UserID: "atomic-owner"}
	for _, scope := range []string{"notification", "shop", "vehicle"} {
		t.Run(scope, func(t *testing.T) {
			entries, err := repo.GetNotificationChanges(context.Background(), user, n)
			if scope == "shop" {
				entries, err = repo.GetNotificationChangesByShop(context.Background(), user, shop, 500)
			}
			if scope == "vehicle" {
				entries, err = repo.GetNotificationChangesByVehicle(context.Background(), user, vehicle)
			}
			require.NoError(t, err)
			require.Len(t, entries, 2)
			require.Equal(t, ids[0], entries[0].ID)
			require.Equal(t, "Event title", entries[0].NotificationTitle)
			require.Equal(t, "Event admin", *entries[0].VehicleAdmin)
			require.Equal(t, "Current title", entries[1].NotificationTitle)
			require.Equal(t, "Current admin", *entries[1].VehicleAdmin)
		})
	}
}

func TestLegacyAuditFailureKeepsBusinessSuccess(t *testing.T) {
	for _, op := range []string{"single-add", "bulk-add", "single-remove", "bulk-remove", "notification-create", "notification-update", "notification-delete", "vehicle-delete"} {
		t.Run(op, func(t *testing.T) {
			r, shop, vehicle := atomicFixture(t, "atomic-owner")
			n := createNotification(t, r, "atomic-owner", shop, vehicle, "Safe title")
			item := addBatchRemovalItem(t, r, "notification", n, "seed")
			_, err := testDB.Exec(`CREATE OR REPLACE FUNCTION test_infrastructure.fail_legacy_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'private candidate credential sentinel'; END $$`)
			require.NoError(t, err)
			_, err = testDB.Exec(`CREATE TRIGGER fail_legacy_audit BEFORE INSERT ON shop_vehicle_notification_changes FOR EACH ROW EXECUTE FUNCTION test_infrastructure.fail_legacy_audit()`)
			require.NoError(t, err)
			defer func() {
				_, e := testDB.Exec(`DROP TRIGGER fail_legacy_audit ON shop_vehicle_notification_changes`)
				require.NoError(t, e)
			}()
			var out bytes.Buffer
			old := slog.Default()
			slog.SetDefault(slog.New(slog.NewJSONHandler(&out, nil)))
			defer slog.SetDefault(old)
			body := map[string]any{"notification_id": n, "niin": "new", "nomenclature": "New", "quantity": 2}
			path := "/api/v1/auth/shops/notifications/items"
			method := http.MethodPost
			status := 201
			switch op {
			case "vehicle-delete":
				path = "/api/v1/auth/shops/vehicles/" + vehicle
				method = http.MethodDelete
				status = 200
				body = nil
			case "bulk-add":
				path += "/bulk"
				body = map[string]any{"notification_id": n, "items": []any{body}}
			case "single-remove":
				path += "/" + item
				method = http.MethodDelete
				body = nil
				status = 200
			case "bulk-remove":
				path += "/bulk"
				method = http.MethodDelete
				body = map[string]any{"item_ids": []string{item}}
				status = 200
			case "notification-create":
				path = "/api/v1/auth/shops/vehicles/notifications"
				body = map[string]any{"shop_id": shop, "vehicle_id": vehicle, "title": "Created", "type": "M1"}
			case "notification-update":
				path = "/api/v1/auth/shops/vehicles/notifications"
				method = http.MethodPut
				status = 200
				body = map[string]any{"notification_id": n, "title": "Updated", "type": "M1"}
			case "notification-delete":
				path = "/api/v1/auth/shops/vehicles/notifications/" + n
				method = http.MethodDelete
				status = 200
				body = nil
			}
			rec := doJSONRequest(t, r, method, path, body, "atomic-owner")
			require.Equal(t, status, rec.Code, rec.Body.String())
			switch op {
			case "vehicle-delete":
				require.Zero(t, atomicRowCount(t, "shop_vehicle", "id=$1", vehicle))
			case "single-add", "bulk-add":
				require.Equal(t, 1, atomicRowCount(t, "shop_notification_items", "notification_id=$1 AND niin='new'", n))
			case "single-remove", "bulk-remove":
				require.Zero(t, atomicRowCount(t, "shop_notification_items", "id=$1", item))
			case "notification-create":
				require.Equal(t, 1, atomicRowCount(t, "shop_vehicle_notifications", "vehicle_id=$1 AND title='Created'", vehicle))
			case "notification-update":
				require.Equal(t, 1, atomicRowCount(t, "shop_vehicle_notifications", "id=$1 AND title='Updated'", n))
			case "notification-delete":
				require.Zero(t, atomicRowCount(t, "shop_vehicle_notifications", "id=$1", n))
			}
			warnings := 0
			for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
				var event map[string]any
				require.NoError(t, json.Unmarshal([]byte(line), &event))
				if event["level"] == "WARN" {
					warnings++
					require.Equal(t, "legacy_notification_audit_failed", event["msg"])
					require.Equal(t, shop, event["shop_id"])
					require.Equal(t, vehicle, event["vehicle_id"])
					require.Equal(t, "atomic-owner", event["actor_id"])
					if op != "vehicle-delete" {
						require.NotEmpty(t, event["notification_id"])
					}
					require.NotEmpty(t, event["operation"])
					require.Equal(t, "database_failure", event["failure_category"])
					_, e := uuid.Parse(event["correlation_id"].(string))
					require.NoError(t, e)
					require.NotContains(t, event, "error")
				}
			}
			require.Equal(t, 1, warnings)
			require.NotContains(t, out.String(), "private candidate credential sentinel")
		})
	}
}
