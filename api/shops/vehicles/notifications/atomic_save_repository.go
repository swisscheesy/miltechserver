package notifications

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"sort"
	"time"
)

func (repo *RepositoryImpl) SaveAtomic(ctx context.Context, userID string, r request.NotificationSaveRequest) (response.NotificationSaveReceipt, error) {
	var receipt response.NotificationSaveReceipt
	if userID == "" {
		return receipt, &shared.Failure{Code: "unauthorized", PublicMessage: "Unauthorized", Status: 401}
	}
	if err := ValidateNotificationSave(r); err != nil {
		return receipt, err
	}
	fingerprint, err := FingerprintNotificationSave(r)
	if err != nil {
		return receipt, err
	}
	// Namespace plus an unambiguous JSON tuple produces the same create target on retry.
	identity, _ := json.Marshal([]string{userID, r.OperationID})
	target := uuid.NewSHA1(uuid.NameSpaceOID, identity).String()
	if r.NotificationID != nil {
		target = *r.NotificationID
	}
	err = shared.WithNotificationMutation(ctx, repo.db, func(tx *sql.Tx) error {
		var err error
		receipt, err = repo.saveAtomicTx(ctx, tx, userID, r, target, fingerprint)
		return err
	})
	if err == nil {
		return receipt, nil
	}
	// Never publish a receipt after a failed/uncertain commit. The caller retries
	// the identical operation ID to resolve a lost commit acknowledgement.
	return response.NotificationSaveReceipt{}, shared.ClassifyFailure(err)
}

func (repo *RepositoryImpl) saveAtomicTx(ctx context.Context, tx *sql.Tx, userID string, r request.NotificationSaveRequest, target string, fingerprint [32]byte) (response.NotificationSaveReceipt, error) {
	receipt := response.NotificationSaveReceipt{OperationID: r.OperationID, NotificationID: target, CommittedAt: time.Now().UTC()}
	result, err := tx.ExecContext(ctx, `INSERT INTO shop_notification_operations (user_id,operation_id,fingerprint,notification_id,committed_at) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (user_id,operation_id) DO NOTHING`, userID, r.OperationID, fingerprint[:], target, receipt.CommittedAt)
	if err != nil {
		return receipt, err
	}
	claimed, err := result.RowsAffected()
	if err != nil {
		return receipt, err
	}
	if claimed == 0 {
		var stored []byte
		err = tx.QueryRowContext(ctx, `SELECT fingerprint,notification_id,committed_at FROM shop_notification_operations WHERE user_id=$1 AND operation_id=$2 FOR UPDATE`, userID, r.OperationID).Scan(&stored, &receipt.NotificationID, &receipt.CommittedAt)
		if err != nil {
			return receipt, err
		}
		if !bytes.Equal(stored, fingerprint[:]) {
			return receipt, &shared.Failure{Code: "operation_payload_conflict", PublicMessage: "operation payload conflicts with previous request", Status: 409}
		}
		// A deleted Shop has no membership left to check. Ownership and fingerprint
		// still constrain this receipt, and replay never runs resource writes.
		if _, _, err := shared.LockShopMutation(ctx, tx, r.ShopID, userID); err != nil && !errors.Is(err, shared.ErrShopNotFound) {
			return receipt, err
		}
		receipt.Replayed = true
		return receipt, nil
	}
	if r.NotificationID != nil {
		var shopID, vehicleID string
		if err := tx.QueryRowContext(ctx, `SELECT shop_id,vehicle_id FROM shop_vehicle_notifications WHERE id=$1`, target).Scan(&shopID, &vehicleID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return receipt, shared.ErrNotificationNotFound
			}
			return receipt, err
		}
		if shopID != r.ShopID || vehicleID != r.VehicleID {
			return receipt, shared.ErrShopAccessDenied
		}
	}
	if _, _, err := shared.LockShopMutation(ctx, tx, r.ShopID, userID); err != nil {
		return receipt, err
	}
	var old model.ShopVehicleNotifications
	var admin string
	if r.NotificationID != nil {
		lists := []string{}
		if r.Attachment.ListID != nil {
			lists = append(lists, *r.Attachment.ListID)
		}
		old, admin, err = shared.LockNotificationMutation(ctx, tx, userID, target, lists...)
		if err != nil {
			return receipt, err
		}
		if old.ShopID != r.ShopID || old.VehicleID != r.VehicleID {
			return receipt, shared.ErrShopAccessDenied
		}
	} else {
		if r.Attachment.ListID != nil {
			if err := shared.LockReferencedLists(ctx, tx, r.ShopID, *r.Attachment.ListID); err != nil {
				return receipt, err
			}
		}
		admin, err = shared.LockNotificationVehicle(ctx, tx, r.ShopID, r.VehicleID)
		if err != nil {
			return receipt, err
		}
	}
	next := model.ShopVehicleNotifications{ID: target, ShopID: r.ShopID, VehicleID: r.VehicleID, Title: r.Details.Title, Description: r.Details.Description, Type: r.Details.Type, Completed: r.Details.IsCompleted, SaveTime: receipt.CommittedAt, LastUpdated: receipt.CommittedAt, AttachedShopList: old.AttachedShopList}
	switch r.Attachment.Intent {
	case "attach":
		next.AttachedShopList = r.Attachment.ListID
	case "remove":
		next.AttachedShopList = nil
	}
	if r.NotificationID == nil {
		_, err = tx.ExecContext(ctx, `INSERT INTO shop_vehicle_notifications (id,shop_id,vehicle_id,title,description,type,completed,attached_shop_list,save_time,last_updated) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)`, target, r.ShopID, r.VehicleID, next.Title, next.Description, next.Type, next.Completed, next.AttachedShopList, receipt.CommittedAt)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE shop_vehicle_notifications SET title=$1,description=$2,type=$3,completed=$4,attached_shop_list=$5,last_updated=$6 WHERE id=$7`, next.Title, next.Description, next.Type, next.Completed, next.AttachedShopList, receipt.CommittedAt, target)
	}
	if err != nil {
		return receipt, err
	}
	changes, err := replaceDirectItems(ctx, tx, r, target, receipt.CommittedAt)
	if err != nil {
		return receipt, err
	}
	details, err := buildFieldChanges(&old, &next)
	if err != nil {
		return receipt, err
	}
	kind := determineChangeType(&old, &next)
	if r.NotificationID == nil {
		kind = "create"
		details = `{"fields_changed":["created"]}`
	}
	if err := insertAtomicAudit(ctx, tx, userID, next, admin, kind, details); err != nil {
		return receipt, err
	}
	for _, change := range changes {
		if err := insertAtomicAudit(ctx, tx, userID, next, admin, change.kind, change.fields); err != nil {
			return receipt, err
		}
	}
	// Return the persisted timestamp so the first receipt and later replays
	// agree even when the database normalizes timestamp precision.
	if err := tx.QueryRowContext(ctx, `UPDATE shop_notification_operations SET committed_at=$1 WHERE user_id=$2 AND operation_id=$3 RETURNING committed_at`, time.Now().UTC(), userID, r.OperationID).Scan(&receipt.CommittedAt); err != nil {
		return receipt, &shared.Failure{Code: "internal_error", PublicMessage: "Unable to complete notification save", Status: 500, Cause: err}
	}
	// The claim, writes and mandatory audits become durable in a single commit.
	return receipt, nil
}

type atomicItemChange struct{ kind, fields string }

func replaceDirectItems(ctx context.Context, tx *sql.Tx, r request.NotificationSaveRequest, target string, now time.Time) ([]atomicItemChange, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id,niin,nomenclature,quantity FROM shop_notification_items WHERE notification_id=$1 ORDER BY id FOR UPDATE`, target)
	if err != nil {
		return nil, err
	}
	current := map[string]request.NotificationSaveItem{}
	for rows.Next() {
		var item request.NotificationSaveItem
		if err := rows.Scan(&item.ID, &item.Niin, &item.Nomenclature, &item.Quantity); err != nil {
			rows.Close()
			return nil, err
		}
		current[item.ID] = item
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	desired := append([]request.NotificationSaveItem{}, r.Items...)
	sort.Slice(desired, func(i, j int) bool { return desired[i].ID < desired[j].ID })
	added, updated, removed := []request.NotificationSaveItem{}, []request.NotificationSaveItem{}, []request.NotificationSaveItem{}
	for _, item := range desired {
		old, exists := current[item.ID]
		if exists {
			delete(current, item.ID)
			if old == item {
				continue
			}
			_, err = tx.ExecContext(ctx, `UPDATE shop_notification_items SET niin=$1,nomenclature=$2,quantity=$3 WHERE id=$4 AND notification_id=$5`, item.Niin, item.Nomenclature, item.Quantity, item.ID, target)
			updated = append(updated, item)
		} else {
			// A UUID already owned by another notification must never be reassigned.
			var owner string
			lookup := tx.QueryRowContext(ctx, `SELECT notification_id FROM shop_notification_items WHERE id=$1`, item.ID).Scan(&owner)
			if lookup == nil {
				return nil, invalidSave()
			}
			if !errors.Is(lookup, sql.ErrNoRows) {
				return nil, lookup
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO shop_notification_items (id,shop_id,notification_id,niin,nomenclature,quantity,save_time) VALUES ($1,$2,$3,$4,$5,$6,$7)`, item.ID, r.ShopID, target, item.Niin, item.Nomenclature, item.Quantity, now)
			added = append(added, item)
		}
		if err != nil {
			return nil, err
		}
	}
	for _, item := range current {
		removed = append(removed, item)
	}
	sort.Slice(removed, func(i, j int) bool { return removed[i].ID < removed[j].ID })
	for _, item := range removed {
		if _, err := tx.ExecContext(ctx, `DELETE FROM shop_notification_items WHERE id=$1 AND notification_id=$2`, item.ID, target); err != nil {
			return nil, err
		}
	}
	changes := []atomicItemChange{}
	for _, group := range []struct {
		kind  string
		items []request.NotificationSaveItem
	}{{"items_added", added}, {"items_updated", updated}, {"items_removed", removed}} {
		if len(group.items) == 0 {
			continue
		}
		data, err := json.Marshal(map[string]any{"fields_changed": []string{"items"}, "item_count": len(group.items), group.kind: group.items})
		if err != nil {
			return nil, err
		}
		kind := group.kind
		if kind == "items_updated" {
			kind = "update"
		}
		changes = append(changes, atomicItemChange{kind, string(data)})
	}
	return changes, nil
}
func insertAtomicAudit(ctx context.Context, tx *sql.Tx, userID string, n model.ShopVehicleNotifications, admin, kind, fields string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO shop_vehicle_notification_changes (notification_id,shop_id,vehicle_id,changed_by,change_type,field_changes,notification_title,notification_type,vehicle_admin) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, n.ID, n.ShopID, n.VehicleID, userID, kind, fields, n.Title, n.Type, admin)
	return err
}
