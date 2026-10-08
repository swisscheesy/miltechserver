package shops_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/shops/members"
	"miltechserver/api/shops/members/invites"
	"miltechserver/api/shops/shared"
	"miltechserver/api/shops/vehicles/notifications"
	notificationitems "miltechserver/api/shops/vehicles/notifications/items"
	"miltechserver/bootstrap"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoggingInviteAndJoin(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "logging-owner")
	ensureUser(t, testDB, "logging-user")
	shop := createShop(t, newTestRouter(t), "logging-owner", "logging-shop")
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	inviteRepo := invites.NewRepository(testDB)
	auth := shared.NewShopAuthorization(testDB)
	code, err := invites.NewService(inviteRepo, auth).GenerateInviteCode(context.Background(), &bootstrap.User{UserID: "logging-owner"}, shop)
	require.NoError(t, err)
	require.NotEmpty(t, code.Code)
	service := members.NewService(members.NewRepository(testDB, nil, nil), inviteRepo, auth)
	user := &bootstrap.User{UserID: "logging-user"}
	require.NoError(t, service.JoinShopViaInviteCode(context.Background(), user, code.Code))
	require.Error(t, service.JoinShopViaInviteCode(context.Background(), user, code.Code))
	require.NotContains(t, output.String(), code.Code, "invite secret leaked")
	joined := 0
	for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
		var event map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &event))
		if event["msg"] == "shop_joined" {
			joined++
			require.Equal(t, shop, event["shop_id"])
			require.Equal(t, "success", event["outcome"])
		}
	}
	require.Equal(t, 1, joined, "only the committed admission emits success")
	require.Contains(t, output.String(), "invite_generated")
	var count int
	require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_members WHERE shop_id=$1 AND user_id='logging-user'`, shop).Scan(&count))
	require.Equal(t, 1, count)
}

func TestLegacyAuditCanceledDeliveryKeepsCommittedRows(t *testing.T) {
	for _, kind := range []string{"notification", "item"} {
		t.Run(kind, func(t *testing.T) {
			r, shop, vehicle := atomicFixture(t, "atomic-owner")
			n := createNotification(t, r, "atomic-owner", shop, vehicle, "Existing")
			tx, err := testDB.BeginTx(context.Background(), nil)
			require.NoError(t, err)
			_, err = tx.Exec(`LOCK TABLE shop_vehicle_notification_changes IN ACCESS EXCLUSIVE MODE`)
			require.NoError(t, err)
			defer tx.Rollback()
			var output bytes.Buffer
			old := slog.Default()
			slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
			defer slog.SetDefault(old)
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			ctx = notificationitems.WithAuditCorrelation(ctx)
			user := &bootstrap.User{UserID: "atomic-owner"}
			id := uuid.NewString()
			if kind == "notification" {
				_, err = notifications.NewRepository(testDB).CreateVehicleNotification(ctx, user, model.ShopVehicleNotifications{ID: id, ShopID: shop, VehicleID: vehicle, Title: "Committed", Type: "M1", SaveTime: time.Now(), LastUpdated: time.Now()})
			} else {
				_, err = notificationitems.NewRepository(testDB).CreateNotificationItem(ctx, user, model.ShopNotificationItems{ID: id, ShopID: shop, NotificationID: n, Niin: "audit-cancel", Nomenclature: "Committed", Quantity: 1, SaveTime: time.Now()})
			}
			require.NoError(t, err, "business commit survives later canceled audit")
			require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
			require.NoError(t, tx.Rollback())
			table := "shop_vehicle_notifications"
			if kind == "item" {
				table = "shop_notification_items"
			}
			require.Equal(t, 1, atomicRowCount(t, table, "id=$1", id))
			var event map[string]any
			require.NoError(t, json.Unmarshal(bytes.TrimSpace(output.Bytes()), &event))
			require.Equal(t, "legacy_notification_audit_failed", event["msg"])
			require.Equal(t, "request_deadline", event["failure_category"])
			firstID := event["correlation_id"]
			_, err = uuid.Parse(firstID.(string))
			require.NoError(t, err)
			require.NotContains(t, event, "error")
			output.Reset()
			notificationitems.WarnLegacyAuditFailure(notificationitems.WithAuditCorrelation(ctx), "probe", shop, vehicle, n, user.UserID, context.DeadlineExceeded)
			require.NoError(t, json.Unmarshal(bytes.TrimSpace(output.Bytes()), &event))
			require.Equal(t, firstID, event["correlation_id"], "nested helpers retain the same server correlation ID")
		})
	}
}
