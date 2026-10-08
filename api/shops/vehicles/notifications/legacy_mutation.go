package notifications

import (
	"context"
	"database/sql"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/shops/shared"
	notificationitems "miltechserver/api/shops/vehicles/notifications/items"
	"miltechserver/bootstrap"
)

// Production repositories capture audit facts while locked, then emit after commit.
// Legacy service fakes retain their existing audit contract.
func (*RepositoryImpl) OwnsLegacyNotificationAudits() {}

func (repo *RepositoryImpl) createLegacyNotification(ctx context.Context, user *bootstrap.User, n model.ShopVehicleNotifications) (*model.ShopVehicleNotifications, error) {
	if err := shared.ValidateNotificationFields(n.Title, n.Type); err != nil {
		return nil, err
	}

	ctx = notificationitems.WithAuditCorrelation(ctx)
	var admin string
	err := shared.WithNotificationMutation(ctx, repo.db, func(tx *sql.Tx) error {
		if _, _, err := shared.LockShopMutation(ctx, tx, n.ShopID, user.UserID); err != nil {
			return err
		}
		if n.AttachedShopList != nil {
			if err := shared.LockReferencedLists(ctx, tx, n.ShopID, *n.AttachedShopList); err != nil {
				return err
			}
		}
		var err error
		admin, err = shared.LockNotificationVehicle(ctx, tx, n.ShopID, n.VehicleID)
		if err != nil {
			return err
		}
		return tx.QueryRowContext(ctx, `INSERT INTO shop_vehicle_notifications (id,shop_id,vehicle_id,title,description,type,completed,attached_shop_list,save_time,last_updated) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id,shop_id,vehicle_id,title,description,type,completed,attached_shop_list,save_time,last_updated`, n.ID, n.ShopID, n.VehicleID, n.Title, n.Description, n.Type, n.Completed, n.AttachedShopList, n.SaveTime, n.LastUpdated).Scan(&n.ID, &n.ShopID, &n.VehicleID, &n.Title, &n.Description, &n.Type, &n.Completed, &n.AttachedShopList, &n.SaveTime, &n.LastUpdated)
	})
	if err != nil {
		return nil, err
	}
	repo.recordLegacyAudit(ctx, user, n, admin, "create", `{"fields_changed":["created"]}`, false)
	return &n, nil
}
func (repo *RepositoryImpl) updateLegacyNotification(ctx context.Context, user *bootstrap.User, u VehicleNotificationUpdate) error {
	if err := shared.ValidateNotificationFields(u.Notification.Title, u.Notification.Type); err != nil {
		return err
	}

	ctx = notificationitems.WithAuditCorrelation(ctx)
	var next model.ShopVehicleNotifications
	var admin, fields, kind string
	err := shared.WithNotificationMutation(ctx, repo.db, func(tx *sql.Tx) error {
		lists := []string{}
		if u.AttachedShopListSet && u.AttachedShopList != nil {
			lists = append(lists, *u.AttachedShopList)
		}
		old, a, err := shared.LockNotificationMutation(ctx, tx, user.UserID, u.Notification.ID, lists...)
		if err != nil {
			return err
		}
		admin = a
		next = u.Notification
		next.ShopID = old.ShopID
		next.VehicleID = old.VehicleID
		next.SaveTime = old.SaveTime
		next.AttachedShopList = old.AttachedShopList
		if u.AttachedShopListSet {
			next.AttachedShopList = u.AttachedShopList
		}
		fields, err = buildFieldChanges(&old, &next)
		if err != nil {
			return err
		}
		kind = determineChangeType(&old, &next)
		_, err = tx.ExecContext(ctx, `UPDATE shop_vehicle_notifications SET title=$1,description=$2,type=$3,completed=$4,attached_shop_list=$5,last_updated=$6 WHERE id=$7`, next.Title, next.Description, next.Type, next.Completed, next.AttachedShopList, next.LastUpdated, next.ID)
		return err
	})
	if err != nil {
		return err
	}
	repo.recordLegacyAudit(ctx, user, next, admin, kind, fields, false)
	return nil
}
func (repo *RepositoryImpl) deleteLegacyNotification(ctx context.Context, user *bootstrap.User, id string) error {
	ctx = notificationitems.WithAuditCorrelation(ctx)
	var n model.ShopVehicleNotifications
	var admin string
	err := shared.WithNotificationMutation(ctx, repo.db, func(tx *sql.Tx) error {
		var err error
		n, admin, err = shared.LockNotificationMutation(ctx, tx, user.UserID, id)
		if err != nil {
			return err
		}
		if err := shared.LockNotificationItems(ctx, tx, id); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM shop_vehicle_notifications WHERE id=$1`, id)
		return err
	})
	if err != nil {
		return err
	}
	repo.recordLegacyAudit(ctx, user, n, admin, "delete", `{"fields_changed":["deleted"]}`, true)
	return nil
}
func (repo *RepositoryImpl) recordLegacyAudit(ctx context.Context, user *bootstrap.User, n model.ShopVehicleNotifications, admin, kind, fields string, deleted bool) {
	var id *string = &n.ID
	if deleted {
		id = nil
	}
	change := model.ShopVehicleNotificationChanges{NotificationID: id, ShopID: n.ShopID, VehicleID: &n.VehicleID, ChangedBy: &user.UserID, ChangeType: kind, FieldChanges: fields, NotificationTitle: &n.Title, NotificationType: &n.Type, VehicleAdmin: &admin}
	if err := repo.CreateNotificationChange(ctx, user, change); err != nil {
		notificationitems.WarnLegacyAuditFailure(ctx, kind, n.ShopID, n.VehicleID, n.ID, user.UserID, err)
	}
}
