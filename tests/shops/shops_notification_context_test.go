package shops_test

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/shops/vehicles/notifications"
	"miltechserver/api/shops/vehicles/notifications/changes"
	"miltechserver/bootstrap"
	"miltechserver/tests/testutil"
	"os"
	"testing"
	"time"
)

func TestNotificationServicePhysicalPoolCancellation(t *testing.T) {
	r, shop, vehicle := atomicFixture(t, "atomic-owner")
	n := createNotification(t, r, "atomic-owner", shop, vehicle, "Cancellation")
	db, err := testutil.OpenDisposableTestDB(os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_DATABASE_MARKER"))
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)
	user := &bootstrap.User{UserID: "atomic-owner"}
	service := notifications.NewService(notifications.NewRepository(db), nil)
	history := changes.NewService(changes.NewRepository(db))
	cases := []struct {
		name string
		call func(context.Context) error
	}{
		{"notification", func(ctx context.Context) error {
			_, err := service.GetVehicleNotificationByID(ctx, user, n)
			return err
		}},
		{"history", func(ctx context.Context) error {
			_, err := history.GetNotificationChangeHistory(ctx, user, n)
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conn, err := db.Conn(context.Background())
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 75*time.Millisecond)
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- tc.call(ctx) }()
			var result error
			returned := false
			select {
			case result = <-done:
				returned = true
			case <-time.After(250 * time.Millisecond):
			}
			require.NoError(t, conn.Close())
			if !returned {
				select {
				case result = <-done:
				case <-time.After(time.Second):
					t.Fatal("pool call did not return")
				}
			}
			require.True(t, returned, "request remained blocked after its deadline until connection release")
			require.ErrorIs(t, result, context.DeadlineExceeded)
		})
	}
}

func TestNotificationRepositoryPhysicalCancellation(t *testing.T) {
	r, shop, vehicle := atomicFixture(t, "atomic-owner")
	n := createNotification(t, r, "atomic-owner", shop, vehicle, "Cancellation")
	list := createList(t, r, "atomic-owner", shop)
	db, err := testutil.OpenDisposableTestDB(os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_DATABASE_MARKER"))
	require.NoError(t, err)
	defer db.Close()
	repo, history := notifications.NewRepository(db), changes.NewRepository(db)
	user := &bootstrap.User{UserID: "atomic-owner"}
	notification := model.ShopVehicleNotifications{ID: uuid.NewString(), ShopID: shop, VehicleID: vehicle, Title: "Created", Type: "M1", SaveTime: time.Now(), LastUpdated: time.Now()}
	update := notifications.VehicleNotificationUpdate{Notification: notification}
	update.Notification.ID = n
	update.AttachedShopListSet = true
	update.AttachedShopList = &list
	change := model.ShopVehicleNotificationChanges{NotificationID: &n, ShopID: shop, VehicleID: &vehicle, ChangedBy: &user.UserID, ChangeType: "update", FieldChanges: "{}"}
	cases := []struct {
		name  string
		write bool
		call  func(context.Context) error
	}{
		{"CreateVehicleNotification", true, func(ctx context.Context) error {
			_, e := repo.CreateVehicleNotification(ctx, user, notification)
			return e
		}},
		{"GetVehicleNotifications", false, func(ctx context.Context) error { _, e := repo.GetVehicleNotifications(ctx, user, vehicle); return e }},
		{"GetVehicleNotificationsWithItems", false, func(ctx context.Context) error {
			_, e := repo.GetVehicleNotificationsWithItems(ctx, user, vehicle)
			return e
		}},
		{"GetItemsByNotificationIDs", false, func(ctx context.Context) error { _, e := repo.GetItemsByNotificationIDs(ctx, []string{n}); return e }},
		{"GetShopNotifications", false, func(ctx context.Context) error { _, e := repo.GetShopNotifications(ctx, user, shop); return e }},
		{"GetVehicleNotificationByID", false, func(ctx context.Context) error { _, e := repo.GetVehicleNotificationByID(ctx, user, n); return e }},
		{"UpdateVehicleNotification", true, func(ctx context.Context) error { return repo.UpdateVehicleNotification(ctx, user, update) }},
		{"DeleteVehicleNotification", true, func(ctx context.Context) error { return repo.DeleteVehicleNotification(ctx, user, n) }},
		{"CreateNotificationChange", false, func(ctx context.Context) error { return repo.CreateNotificationChange(ctx, user, change) }},
		{"GetShopVehicleByID", false, func(ctx context.Context) error { _, e := repo.GetShopVehicleByID(ctx, user, vehicle); return e }},
		{"GetShopListByID", false, func(ctx context.Context) error { _, e := repo.GetShopListByID(ctx, user, list); return e }},
		{"IsUserMemberOfShop", false, func(ctx context.Context) error { _, e := repo.IsUserMemberOfShop(ctx, user, shop); return e }},
		{"GetNotificationItems", false, func(ctx context.Context) error { _, e := repo.GetNotificationItems(ctx, user, n); return e }},
		{"history/GetNotificationChanges", false, func(ctx context.Context) error { _, e := history.GetNotificationChanges(ctx, user, n); return e }},
		{"history/GetNotificationChangesByShop", false, func(ctx context.Context) error {
			_, e := history.GetNotificationChangesByShop(ctx, user, shop, 500)
			return e
		}},
		{"history/GetNotificationChangesByVehicle", false, func(ctx context.Context) error {
			_, e := history.GetNotificationChangesByVehicle(ctx, user, vehicle)
			return e
		}},
		{"history/GetVehicleNotificationByID", false, func(ctx context.Context) error { _, e := history.GetVehicleNotificationByID(ctx, user, n); return e }},
		{"history/GetShopVehicleByID", false, func(ctx context.Context) error { _, e := history.GetShopVehicleByID(ctx, user, vehicle); return e }},
		{"history/IsUserMemberOfShop", false, func(ctx context.Context) error { _, e := history.IsUserMemberOfShop(ctx, user, shop); return e }},
	}
	db.SetMaxOpenConns(1)
	for _, tc := range cases {
		t.Run("pool/"+tc.name, func(t *testing.T) {
			before := writerSnapshot(t)
			conn, e := db.Conn(context.Background())
			require.NoError(t, e)
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- tc.call(ctx) }()
			var result error
			returned := false
			select {
			case result = <-done:
				returned = true
			case <-time.After(200 * time.Millisecond):
			}
			require.NoError(t, conn.Close())
			if !returned {
				select {
				case result = <-done:
				case <-time.After(time.Second):
					t.Fatal("call did not return")
				}
			}
			require.True(t, returned)
			require.ErrorIs(t, result, context.DeadlineExceeded)
			require.Equal(t, before, writerSnapshot(t))
		})
	}
	db.SetMaxOpenConns(4)
	for _, target := range []string{"shop", "vehicle", "notification"} {
		for _, tc := range cases {
			if !tc.write || (target == "notification" && tc.name == "CreateVehicleNotification") {
				continue
			}
			t.Run(target+"-lock/"+tc.name, func(t *testing.T) {
				before := writerSnapshot(t)
				tx, e := testDB.BeginTx(context.Background(), nil)
				require.NoError(t, e)
				query, id := `SELECT id FROM shops WHERE id=$1 FOR UPDATE`, shop
				if target == "vehicle" {
					query, id = `SELECT id FROM shop_vehicle WHERE id=$1 FOR UPDATE`, vehicle
				}
				if target == "notification" {
					query, id = `SELECT id FROM shop_vehicle_notifications WHERE id=$1 FOR UPDATE`, n
				}
				_, e = tx.Exec(query, id)
				require.NoError(t, e)
				ctx, cancel := context.WithTimeout(context.Background(), 75*time.Millisecond)
				defer cancel()
				started := time.Now()
				e = tc.call(ctx)
				require.NoError(t, tx.Rollback())
				require.Error(t, e)
				require.Error(t, ctx.Err())
				require.Less(t, time.Since(started), time.Second)
				require.Equal(t, before, writerSnapshot(t))
			})
		}
	}
}
