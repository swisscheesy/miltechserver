package items

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"sort"
)

func (*RepositoryImpl) OwnsLegacyNotificationAudits() {}
func (repo *RepositoryImpl) createLegacyItems(user *bootstrap.User, items []model.ShopNotificationItems) ([]model.ShopNotificationItems, error) {
	if len(items) == 0 {
		return []model.ShopNotificationItems{}, nil
	}
	ordered := append([]model.ShopNotificationItems{}, items...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	createdByID := make(map[string]model.ShopNotificationItems, len(items))
	ctx := context.Background()
	var n model.ShopVehicleNotifications
	var admin string
	err := shared.WithNotificationMutation(ctx, repo.db, func(tx *sql.Tx) error {
		var err error
		n, admin, err = shared.LockNotificationMutation(ctx, tx, user.UserID, ordered[0].NotificationID)
		if err != nil {
			return err
		}
		for _, item := range ordered {
			if item.NotificationID != n.ID || item.ShopID != n.ShopID {
				return shared.ErrShopAccessDenied
			}
		}
		if err := shared.LockNotificationItems(ctx, tx, n.ID); err != nil {
			return err
		}
		for _, item := range ordered {
			var created model.ShopNotificationItems
			var nickname, unitOfMeasure sql.NullString
			err = tx.QueryRowContext(ctx, `INSERT INTO shop_notification_items (id,shop_id,notification_id,niin,nomenclature,quantity,save_time,nickname,unit_of_measure) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id,shop_id,notification_id,niin,nomenclature,quantity,save_time,nickname,unit_of_measure`, item.ID, item.ShopID, item.NotificationID, item.Niin, item.Nomenclature, item.Quantity, item.SaveTime, item.Nickname, item.UnitOfMeasure).Scan(&created.ID, &created.ShopID, &created.NotificationID, &created.Niin, &created.Nomenclature, &created.Quantity, &created.SaveTime, &nickname, &unitOfMeasure)
			if err != nil {
				return err
			}
			created.Nickname = shared.NullStringPtr(nickname)
			created.UnitOfMeasure = shared.NullStringPtr(unitOfMeasure)
			createdByID[item.ID] = created
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	created := make([]model.ShopNotificationItems, len(items))
	for i, item := range items {
		created[i] = createdByID[item.ID]
	}
	fields, err := buildItemAdditionFieldChanges(created)
	if err == nil {
		repo.recordLegacyItemAudit(user, n, admin, "items_added", fields)
	} else {
		slog.Warn("Failed to build notification item audit", "error", err)
	}
	return created, nil
}
func (repo *RepositoryImpl) deleteLegacyItems(user *bootstrap.User, ids []string, requireItem bool) error {
	if len(ids) == 0 {
		return nil
	}
	ordered := append([]string{}, ids...)
	sort.Strings(ordered)
	ctx := context.Background()
	var n model.ShopVehicleNotifications
	var admin string
	var removed []model.ShopNotificationItems
	err := shared.WithNotificationMutation(ctx, repo.db, func(tx *sql.Tx) error {
		removed = nil
		notificationID := ""
		for _, id := range ordered {
			var owner string
			err := tx.QueryRowContext(ctx, `SELECT notification_id FROM shop_notification_items WHERE id=$1`, id).Scan(&owner)
			if errors.Is(err, sql.ErrNoRows) && !requireItem {
				continue
			}
			if err != nil {
				return err
			}
			if notificationID != "" && notificationID != owner {
				return errors.New("cannot delete items from multiple notifications in a single operation")
			}
			notificationID = owner
		}
		if notificationID == "" {
			return nil
		}
		var err error
		n, admin, err = shared.LockNotificationMutation(ctx, tx, user.UserID, notificationID)
		if err != nil {
			return err
		}
		if err := shared.LockNotificationItems(ctx, tx, n.ID); err != nil {
			return err
		}
		previous := ""
		for _, id := range ordered {
			if id == previous {
				continue
			}
			previous = id
			var item model.ShopNotificationItems
			err := tx.QueryRowContext(ctx, `SELECT id,shop_id,notification_id,niin,nomenclature,quantity,save_time FROM shop_notification_items WHERE id=$1`, id).Scan(&item.ID, &item.ShopID, &item.NotificationID, &item.Niin, &item.Nomenclature, &item.Quantity, &item.SaveTime)
			if errors.Is(err, sql.ErrNoRows) && !requireItem {
				continue
			}
			if err != nil {
				return err
			}
			if item.ShopID != n.ShopID || item.NotificationID != n.ID {
				return shared.ErrShopAccessDenied
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM shop_notification_items WHERE id=$1 AND notification_id=$2`, id, n.ID); err != nil {
				return err
			}
			removed = append(removed, item)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(removed) > 0 {
		fields, err := buildItemRemovalFieldChanges(removed)
		if err == nil {
			repo.recordLegacyItemAudit(user, n, admin, "items_removed", fields)
		} else {
			slog.Warn("Failed to build notification item audit", "error", err)
		}
	}
	return nil
}
func (repo *RepositoryImpl) recordLegacyItemAudit(user *bootstrap.User, n model.ShopVehicleNotifications, admin, kind, fields string) {
	change := model.ShopVehicleNotificationChanges{NotificationID: &n.ID, ShopID: n.ShopID, VehicleID: &n.VehicleID, ChangedBy: &user.UserID, ChangeType: kind, FieldChanges: fields, NotificationTitle: &n.Title, NotificationType: &n.Type, VehicleAdmin: &admin}
	if err := repo.CreateNotificationChange(user, change); err != nil {
		slog.Warn("Failed to record notification item change", "error", err, "change_type", kind)
	}
}
