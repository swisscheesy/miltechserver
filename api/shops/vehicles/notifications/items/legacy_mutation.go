package items

import (
	"context"
	"database/sql"
	"errors"
	"github.com/lib/pq"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"slices"
	"sort"
)

func (*RepositoryImpl) OwnsLegacyNotificationAudits() {}
func (repo *RepositoryImpl) createLegacyItems(ctx context.Context, user *bootstrap.User, items []model.ShopNotificationItems) ([]model.ShopNotificationItems, error) {
	ctx = WithAuditCorrelation(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for _, item := range items {
		if err := shared.ValidateItemFields(item.Niin, item.Nomenclature, item.Quantity); err != nil {
			return nil, err
		}
		if err := shared.ValidateItemEnrichment(item.Nickname, item.UnitOfMeasure); err != nil {
			return nil, err
		}
	}

	for _, item := range items {
		if item.NotificationID != items[0].NotificationID {
			return nil, &shared.ValidationError{}
		}
	}

	if len(items) == 0 {
		return []model.ShopNotificationItems{}, nil
	}
	ordered := append([]model.ShopNotificationItems{}, items...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	createdByID := make(map[string]model.ShopNotificationItems, len(items))
	var n model.ShopVehicleNotifications
	var admin string
	err := shared.WithNotificationMutation(ctx, repo.db, func(tx *sql.Tx) error {
		ctx := MetadataMutationContext(ctx)
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
			resolved, err := ResolveRetainedMetadata(ctx, tx, MetadataIntent{item.ID, item.NotificationID, item.Niin, item.Nickname, item.UnitOfMeasure})
			if err != nil {
				return err
			}
			item.Nickname, item.UnitOfMeasure = resolved.Nickname, resolved.UnitOfMeasure
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
		PersistMetadataAmbiguity(ctx, repo.db, user.UserID, err)
		return nil, err
	}
	created := make([]model.ShopNotificationItems, len(items))
	for i, item := range items {
		created[i] = createdByID[item.ID]
	}
	fields, err := buildItemAdditionFieldChanges(created)
	if err == nil {
		repo.recordLegacyItemAudit(ctx, user, n, admin, "items_added", fields)
	} else {
		WarnLegacyAuditFailure(ctx, "audit_encoding", n.ShopID, n.VehicleID, n.ID, user.UserID, err)
	}
	return created, nil
}
func (repo *RepositoryImpl) deleteLegacyItems(ctx context.Context, user *bootstrap.User, ids []string, requireItem bool) (int64, error) {
	ctx = WithAuditCorrelation(ctx)
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if user == nil || user.UserID == "" {
		return 0, errors.New("unauthorized user")
	}
	if len(ids) == 0 {
		return 0, nil
	}
	ordered := append([]string{}, ids...)
	sort.Strings(ordered)
	ordered = slices.Compact(ordered)
	var n model.ShopVehicleNotifications
	var admin string
	var removed []model.ShopNotificationItems
	var count int64
	err := shared.WithNotificationMutation(ctx, repo.db, func(tx *sql.Tx) error {
		ctx := MetadataMutationContext(ctx)
		removed = nil
		count = 0
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
		if !requireItem && (errors.Is(err, shared.ErrNotificationNotFound) || errors.Is(err, shared.ErrShopNotFound) || errors.Is(err, shared.ErrVehicleNotFound) || errors.Is(err, shared.ErrShopAccessDenied)) {
			// A concurrent deletion may win while authority changes. Only
			// an empty physical target set is a no-op; any survivor keeps the error.
			var exists bool
			if queryErr := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM shop_notification_items WHERE id=ANY($1))`, pq.Array(ordered)).Scan(&exists); queryErr != nil {
				return queryErr
			}
			if !exists {
				return nil
			}
		}
		if err != nil {
			return err
		}
		if err := shared.LockNotificationItems(ctx, tx, n.ID); err != nil {
			return err
		}
		for _, id := range ordered {
			var item model.ShopNotificationItems
			err := tx.QueryRowContext(ctx, `SELECT id,shop_id,notification_id,niin,nomenclature,quantity,save_time,nickname,unit_of_measure FROM shop_notification_items WHERE id=$1`, id).Scan(&item.ID, &item.ShopID, &item.NotificationID, &item.Niin, &item.Nomenclature, &item.Quantity, &item.SaveTime, &item.Nickname, &item.UnitOfMeasure)
			if errors.Is(err, sql.ErrNoRows) && !requireItem {
				continue
			}
			if err != nil {
				return err
			}
			if item.ShopID != n.ShopID || item.NotificationID != n.ID {
				return shared.ErrShopAccessDenied
			}
			removed = append(removed, item)
		}
		// Validate the complete batch before retaining metadata or deleting rows.
		for _, item := range removed {
			if err := RetainItemMetadata(ctx, tx, item); err != nil {
				return err
			}
			result, err := tx.ExecContext(ctx, `DELETE FROM shop_notification_items WHERE id=$1 AND notification_id=$2`, item.ID, n.ID)
			if err != nil {
				return err
			}
			affected, err := result.RowsAffected()
			if err != nil {
				return err
			}
			count += affected
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	if len(removed) > 0 {
		fields, err := buildItemRemovalFieldChanges(removed)
		if err == nil {
			repo.recordLegacyItemAudit(ctx, user, n, admin, "items_removed", fields)
		} else {
			WarnLegacyAuditFailure(ctx, "audit_encoding", n.ShopID, n.VehicleID, n.ID, user.UserID, err)
		}
	}
	return count, nil
}
func (repo *RepositoryImpl) recordLegacyItemAudit(ctx context.Context, user *bootstrap.User, n model.ShopVehicleNotifications, admin, kind, fields string) {
	change := model.ShopVehicleNotificationChanges{NotificationID: &n.ID, ShopID: n.ShopID, VehicleID: &n.VehicleID, ChangedBy: &user.UserID, ChangeType: kind, FieldChanges: fields, NotificationTitle: &n.Title, NotificationType: &n.Type, VehicleAdmin: &admin}
	if err := repo.CreateNotificationChange(ctx, user, change); err != nil {
		WarnLegacyAuditFailure(ctx, kind, n.ShopID, n.VehicleID, n.ID, user.UserID, err)
	}
}
