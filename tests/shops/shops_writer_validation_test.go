package shops_test

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The snapshots include metadata and operation claims, not only visible rows.
func writerSnapshot(t *testing.T) string {
	t.Helper()
	var snapshot string
	require.NoError(t, testDB.QueryRow(`SELECT json_build_array(
 (SELECT json_agg(x ORDER BY id) FROM shop_lists x),
 (SELECT json_agg(x ORDER BY id) FROM shop_list_items x),
 (SELECT json_agg(x ORDER BY id) FROM shop_vehicle_notifications x),
 (SELECT json_agg(x ORDER BY id) FROM shop_notification_items x),
 (SELECT json_agg(x ORDER BY id) FROM shop_vehicle_notification_changes x),
 (SELECT json_agg(x ORDER BY notification_id,niin) FROM shop_notification_item_metadata x),
 (SELECT json_agg(x ORDER BY operation_id) FROM shop_notification_operations x))::text`).Scan(&snapshot))
	return snapshot
}

func TestEquivalentWriterValidationValidContent(t *testing.T) {
	for _, contract := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy", true: "contract2"}[contract], func(t *testing.T) {
			router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
			list := createList(t, router, "atomic-owner", shopID)
			otherList := createList(t, router, "atomic-owner", shopID)
			notification := createNotification(t, router, "atomic-owner", shopID, vehicleID, "Seed")
			long := strings.Repeat("界", 1000)
			raw := "  FUTURE-UNIT  "
			send := func(method, path string, body any, want int) map[string]any {
				var rec *httptest.ResponseRecorder
				if contract {
					rec = doContractRequest(t, router, method, path, body, "atomic-owner")
				} else {
					rec = doJSONRequest(t, router, method, path, body, "atomic-owner")
				}
				require.Equal(t, want, rec.Code, rec.Body.String())
				return decodeMap(t, decodeStandardResponse(t, rec.Body).Data)
			}
			item := map[string]any{"list_id": list, "notification_id": notification, "niin": long, "nomenclature": long, "quantity": 1, "nickname": "", "unit_of_measure": raw}
			created := send(http.MethodPost, "/api/v1/auth/shops/lists/items", item, 201)
			item["item_id"] = created["id"]
			send(http.MethodPut, "/api/v1/auth/shops/lists/items", item, 200)
			var storedRaw, storedNiin string
			require.NoError(t, testDB.QueryRow(`SELECT niin,unit_of_measure FROM shop_list_items WHERE id=$1`, created["id"]).Scan(&storedNiin, &storedRaw))
			require.Equal(t, long, storedNiin)
			require.Equal(t, raw, storedRaw)
			item["list_id"] = otherList
			// The actual nested list targets remain authoritative for same-Shop batches.
			body := map[string]any{"list_id": list, "items": []any{map[string]any{"list_id": list, "niin": "first", "nomenclature": long, "quantity": 1}, item}}
			var rec *httptest.ResponseRecorder
			if contract {
				rec = doContractRequest(t, router, http.MethodPost, "/api/v1/auth/shops/lists/items/bulk", body, "atomic-owner")
			} else {
				rec = doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/lists/items/bulk", body, "atomic-owner")
			}
			require.Equal(t, 201, rec.Code, rec.Body.String())
			send(http.MethodPost, "/api/v1/auth/shops/notifications/items", item, 201)
			item["niin"] = long + "-bulk"
			body = map[string]any{"notification_id": notification, "items": []any{item}}
			if contract {
				rec = doContractRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items/bulk", body, "atomic-owner")
			} else {
				rec = doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items/bulk", body, "atomic-owner")
			}
			require.Equal(t, 201, rec.Code, rec.Body.String())
			details := map[string]any{"shop_id": shopID, "vehicle_id": vehicleID, "notification_id": notification, "title": long, "description": long, "type": "MW"}
			send(http.MethodPost, "/api/v1/auth/shops/vehicles/notifications", details, 201)
			send(http.MethodPut, "/api/v1/auth/shops/vehicles/notifications", details, 200)
			if contract {
				r := atomicNotificationRequest(shopID, vehicleID)
				r.NotificationID = &notification
				r.Details.Title = long
				r.Details.Description = long
				r.Items = r.Items[:1]
				r.Items[0].Niin = long
				r.Items[0].Nomenclature = long
				r.Items[0].Nickname = itemFieldsPtr("")
				r.Items[0].UnitOfMeasure = &raw
				receipt := atomicReceipt(t, doContract2JSONRequest(t, router, r, "atomic-owner"))
				require.False(t, receipt.Replayed)
				_, _, unit := storedItemFields(t, r.Items[0].ID)
				require.Equal(t, raw, unit.String)
				require.True(t, atomicReceipt(t, doContract2JSONRequest(t, router, r, "atomic-owner")).Replayed)
			}
		})
	}
}

func TestEquivalentWriterValidationDefaultOnlyNoOp(t *testing.T) {
	router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
	r := atomicNotificationRequest(shopID, vehicleID)
	receipt := atomicReceipt(t, doContract2JSONRequest(t, router, r, "atomic-owner"))
	before := atomicRowCount(t, "shop_vehicle_notification_changes", "notification_id=$1 AND field_changes::text LIKE '%items%'", receipt.NotificationID)
	r.OperationID = atomicNotificationRequest(shopID, vehicleID).OperationID
	r.NotificationID = &receipt.NotificationID
	for i := range r.Items {
		r.Items[i].Nickname = itemFieldsPtr("")
		r.Items[i].UnitOfMeasure = itemFieldsPtr("EA")
	}
	atomicReceipt(t, doContract2JSONRequest(t, router, r, "atomic-owner"))
	require.Equal(t, before, atomicRowCount(t, "shop_vehicle_notification_changes", "notification_id=$1 AND field_changes::text LIKE '%items%'", receipt.NotificationID))
	for _, item := range r.Items {
		_, nickname, unit := storedItemFields(t, item.ID)
		require.False(t, nickname.Valid)
		require.False(t, unit.Valid)
	}
}

func TestEquivalentWriterValidation(t *testing.T) {
	for _, contract := range []bool{false, true} {
		label := "legacy"
		if contract {
			label = "contract2"
		}
		t.Run(label, func(t *testing.T) {
			router, shopID, vehicleID := atomicFixture(t, "atomic-owner")
			list := createList(t, router, "atomic-owner", shopID)
			itemID := createListItem(t, router, "atomic-owner", list, "seed", "Seed")
			notification := createNotification(t, router, "atomic-owner", shopID, vehicleID, "Seed")
			other := createNotification(t, router, "atomic-owner", shopID, vehicleID, "Other")
			for _, writer := range []string{"list-single", "list-update", "list-bulk", "notification-single", "notification-bulk", "notification-create", "notification-update", "atomic"} {
				for _, invalid := range []string{"zero", "negative", "blank-niin", "blank-nomenclature", "nickname51", "unit51", "blank-unit", "blank-title", "unsupported-type"} {
					isNotification := writer == "notification-create" || writer == "notification-update"
					if isNotification && invalid != "blank-title" && invalid != "unsupported-type" {
						continue
					}
					if !isNotification && writer != "atomic" && (invalid == "blank-title" || invalid == "unsupported-type") {
						continue
					}
					if writer == "atomic" && !contract {
						continue
					}
					t.Run(writer+"/"+invalid, func(t *testing.T) {
						item := map[string]any{"list_id": list, "notification_id": notification, "item_id": itemID, "niin": "NSN-raw", "nomenclature": "Pump", "quantity": 2}
						switch invalid {
						case "zero":
							item["quantity"] = 0
						case "negative":
							item["quantity"] = -1
						case "blank-niin":
							item["niin"] = " \t"
						case "blank-nomenclature":
							item["nomenclature"] = " \n"
						case "nickname51":
							item["nickname"] = strings.Repeat("界", 51)
						case "unit51":
							item["unit_of_measure"] = strings.Repeat("界", 51)
						case "blank-unit":
							item["unit_of_measure"] = " \t"
						}
						details := map[string]any{"shop_id": shopID, "vehicle_id": vehicleID, "notification_id": notification, "title": "Inspect", "description": "", "type": "PM"}
						if invalid == "blank-title" {
							details["title"] = " \t"
						}
						if invalid == "unsupported-type" {
							details["type"] = "BAD"
						}
						method, path, body := http.MethodPost, "/api/v1/auth/shops/lists/items", any(item)
						switch writer {
						case "list-update":
							method = http.MethodPut
						case "list-bulk":
							path += "/bulk"
							body = map[string]any{"list_id": list, "items": []any{map[string]any{"list_id": list, "niin": "first", "nomenclature": "First", "quantity": 1}, item}}
						case "notification-single":
							path = "/api/v1/auth/shops/notifications/items"
						case "notification-bulk":
							path = "/api/v1/auth/shops/notifications/items/bulk"
							body = map[string]any{"notification_id": notification, "items": []any{map[string]any{"notification_id": notification, "niin": "first", "nomenclature": "First", "quantity": 1}, item}}
						case "notification-create":
							path = "/api/v1/auth/shops/vehicles/notifications"
							body = details
						case "notification-update":
							path = "/api/v1/auth/shops/vehicles/notifications"
							method = http.MethodPut
							body = details
						case "atomic":
							r := atomicNotificationRequest(shopID, vehicleID)
							r.NotificationID = &notification
							r.Details.Title = details["title"].(string)
							r.Details.Type = details["type"].(string)
							raw, err := json.Marshal(item)
							require.NoError(t, err)
							require.NoError(t, json.Unmarshal(raw, &r.Items[1]))
							r.Items[1].ID = atomicNotificationRequest(shopID, vehicleID).Items[0].ID
							path = atomicSavePath
							body = r
						}
						before := writerSnapshot(t)
						var rec *httptest.ResponseRecorder
						if contract {
							rec = doContractRequest(t, router, method, path, body, "atomic-owner")
						} else {
							rec = doJSONRequest(t, router, method, path, body, "atomic-owner")
						}
						require.Equal(t, 400, rec.Code, rec.Body.String())
						if contract {
							require.Equal(t, "invalid", decodeMap(t, json.RawMessage(rec.Body.Bytes()))["code"])
						} else {
							require.Contains(t, rec.Body.String(), "details")
						}
						require.Equal(t, before, writerSnapshot(t))
					})
				}
			}
			for _, nested := range []string{"omitted", "null", "equal", "contradiction"} {
				t.Run("nested-target/"+nested, func(t *testing.T) {
					item := map[string]any{"niin": "nested-" + nested, "nomenclature": "Part", "quantity": 1}
					switch nested {
					case "null":
						item["notification_id"] = nil
					case "equal":
						item["notification_id"] = notification
					case "contradiction":
						item["notification_id"] = other
					}
					before := writerSnapshot(t)
					body := map[string]any{"notification_id": notification, "items": []any{map[string]any{"niin": "first-" + nested, "nomenclature": "First", "quantity": 1}, item}}
					var rec *httptest.ResponseRecorder
					if contract {
						rec = doContractRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items/bulk", body, "atomic-owner")
					} else {
						rec = doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items/bulk", body, "atomic-owner")
					}
					if nested == "contradiction" {
						require.Equal(t, 400, rec.Code, rec.Body.String())
						require.Equal(t, before, writerSnapshot(t))
					} else {
						require.Equal(t, 201, rec.Code, rec.Body.String())
					}
				})
			}
		})
	}
}
