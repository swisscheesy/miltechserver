package shared

import (
	"context"
	"database/sql"
	"errors"
	"github.com/lib/pq"
	"math/rand/v2"
	"miltechserver/.gen/miltech_ng/public/model"
	txdb "miltechserver/api/shared/db"
	"time"
)

// ErrNotificationLockChanged requires a fresh transaction, never a late list lock.
var ErrNotificationLockChanged = errors.New("notification attachment changed while acquiring locks")

// LockNotificationMutation follows Shop -> member -> lists -> vehicle -> notification.
// The initial reads only identify locks; authorization and state are rechecked under locks.
func LockNotificationMutation(ctx context.Context, tx *sql.Tx, userID, notificationID string, newLists ...string) (model.ShopVehicleNotifications, string, error) {
	var n model.ShopVehicleNotifications
	var shopID, vehicleID string
	if err := tx.QueryRowContext(ctx, `SELECT shop_id,vehicle_id FROM shop_vehicle_notifications WHERE id=$1`, notificationID).Scan(&shopID, &vehicleID); err != nil {
		return n, "", authorizationQueryError("resolve notification", ErrNotificationNotFound, err)
	}
	if _, _, err := LockShopMutation(ctx, tx, shopID, userID); err != nil {
		return n, "", err
	}
	var oldList sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT attached_shop_list FROM shop_vehicle_notifications WHERE id=$1 AND shop_id=$2`, notificationID, shopID).Scan(&oldList); err != nil {
		return n, "", authorizationQueryError("resolve notification attachment", ErrNotificationNotFound, err)
	}
	lists := append([]string{oldList.String}, newLists...)
	if err := LockReferencedLists(ctx, tx, shopID, lists...); err != nil {
		return n, "", err
	}
	admin, err := LockNotificationVehicle(ctx, tx, shopID, vehicleID)
	if err != nil {
		return n, "", err
	}
	var list sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT id,shop_id,vehicle_id,title,description,type,completed,attached_shop_list,save_time,last_updated FROM shop_vehicle_notifications WHERE id=$1 FOR UPDATE`, notificationID).Scan(&n.ID, &n.ShopID, &n.VehicleID, &n.Title, &n.Description, &n.Type, &n.Completed, &list, &n.SaveTime, &n.LastUpdated)
	if err != nil {
		return n, "", authorizationQueryError("lock notification", ErrNotificationNotFound, err)
	}
	if n.ShopID != shopID || n.VehicleID != vehicleID {
		return n, "", ErrShopAccessDenied
	}
	if list != oldList {
		return n, "", ErrNotificationLockChanged
	}
	if list.Valid {
		n.AttachedShopList = &list.String
	}
	return n, admin, nil
}

func LockNotificationVehicle(ctx context.Context, tx *sql.Tx, shopID, vehicleID string) (string, error) {
	var owner, admin string
	if err := tx.QueryRowContext(ctx, `SELECT shop_id,admin FROM shop_vehicle WHERE id=$1 FOR UPDATE`, vehicleID).Scan(&owner, &admin); err != nil {
		return "", authorizationQueryError("lock notification vehicle", ErrVehicleNotFound, err)
	}
	if owner != shopID {
		return "", ErrShopAccessDenied
	}
	return admin, nil
}

// WithNotificationMutation retries only known rolled-back database failures.
// An ambiguous commit is returned; legacy requests have no operation identity.
func WithNotificationMutation(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		err = txdb.WithTxContext(ctx, db, fn)
		if err == nil {
			return nil
		}
		var pg *pq.Error
		retry := errors.Is(err, ErrNotificationLockChanged) || (errors.As(err, &pg) && (pg.Code == "40P01" || pg.Code == "40001"))
		if !retry || attempt == 2 {
			return err
		}
		timer := time.NewTimer(time.Duration(25+rand.IntN(76)) * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return err
}
func LockNotificationItems(ctx context.Context, tx *sql.Tx, id string) error {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM shop_notification_items WHERE notification_id=$1 ORDER BY id FOR UPDATE`, id)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var itemID string
		if err := rows.Scan(&itemID); err != nil {
			return err
		}
	}
	return rows.Err()
}
