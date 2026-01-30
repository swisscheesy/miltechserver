package notification_items

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/bootstrap"

	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) GetByUserID(user *bootstrap.User) ([]model.UserNotificationItems, error) {
	var items []model.UserNotificationItems

	if user == nil {
		return nil, errors.New("valid user not found")
	}

	stmt := SELECT(UserNotificationItems.AllColumns).
		FROM(UserNotificationItems).
		WHERE(UserNotificationItems.UserID.EQ(String(user.UserID)))

	err := stmt.Query(repo.db, &items)
	if err != nil {
		return nil, fmt.Errorf("error retrieving notification items for user %s: %w", user.UserID, err)
	}

	slog.Info("notification items retrieved for user", "user_id", user.UserID, "count", len(items))
	return items, nil
}

func (repo *RepositoryImpl) GetByNotificationID(user *bootstrap.User, notificationID string) ([]model.UserNotificationItems, error) {
	var items []model.UserNotificationItems

	if user == nil {
		return nil, errors.New("valid user not found")
	}

	stmt := SELECT(UserNotificationItems.AllColumns).
		FROM(UserNotificationItems).
		WHERE(UserNotificationItems.UserID.EQ(String(user.UserID)).
			AND(UserNotificationItems.NotificationID.EQ(String(notificationID))))

	err := stmt.Query(repo.db, &items)
	if err != nil {
		return nil, fmt.Errorf("error retrieving items for notification %s: %w", notificationID, err)
	}

	slog.Info("notification items retrieved", "user_id", user.UserID, "notification_id", notificationID, "count", len(items))
	return items, nil
}

func (repo *RepositoryImpl) GetByID(user *bootstrap.User, itemID string) (*model.UserNotificationItems, error) {
	var item model.UserNotificationItems

	if user == nil {
		return nil, errors.New("valid user not found")
	}

	stmt := SELECT(UserNotificationItems.AllColumns).
		FROM(UserNotificationItems).
		WHERE(UserNotificationItems.UserID.EQ(String(user.UserID)).
			AND(UserNotificationItems.ID.EQ(String(itemID))))

	err := stmt.Query(repo.db, &item)
	if err != nil {
		return nil, fmt.Errorf("notification item not found for user %s: %w", user.UserID, err)
	}

	slog.Info("notification item retrieved", "user_id", user.UserID, "item_id", itemID)
	return &item, nil
}

func (repo *RepositoryImpl) Upsert(user *bootstrap.User, item model.UserNotificationItems) error {
	if user == nil {
		return errors.New("valid user not found")
	}

	stmt := UserNotificationItems.INSERT(
		UserNotificationItems.ID, UserNotificationItems.UserID, UserNotificationItems.NotificationID,
		UserNotificationItems.Niin, UserNotificationItems.Nomenclature, UserNotificationItems.Quantity,
		UserNotificationItems.SaveTime).
		MODEL(item).
		ON_CONFLICT(UserNotificationItems.ID).
		DO_UPDATE(
			SET(
				UserNotificationItems.NotificationID.SET(String(item.NotificationID)),
				UserNotificationItems.Niin.SET(String(item.Niin)),
				UserNotificationItems.Nomenclature.SET(String(item.Nomenclature)),
				UserNotificationItems.Quantity.SET(Int32(item.Quantity))).
				WHERE(UserNotificationItems.ID.EQ(String(item.ID)))).
		RETURNING(UserNotificationItems.AllColumns)

	err := stmt.Query(repo.db, &item)
	if err != nil {
		return fmt.Errorf("error saving notification item: %w", err)
	}

	slog.Info("notification item saved", "user_id", user.UserID, "item_id", item.ID)
	return nil
}

func (repo *RepositoryImpl) UpsertBatch(user *bootstrap.User, items []model.UserNotificationItems) error {
	if user == nil {
		return errors.New("valid user not found")
	}

	var failedItems []string
	for _, item := range items {
		stmt := UserNotificationItems.INSERT(
			UserNotificationItems.ID, UserNotificationItems.UserID, UserNotificationItems.NotificationID,
			UserNotificationItems.Niin, UserNotificationItems.Nomenclature, UserNotificationItems.Quantity,
			UserNotificationItems.SaveTime).
			MODEL(item).
			ON_CONFLICT(UserNotificationItems.ID).
			DO_UPDATE(
				SET(
					UserNotificationItems.NotificationID.SET(String(item.NotificationID)),
					UserNotificationItems.Niin.SET(String(item.Niin)),
					UserNotificationItems.Nomenclature.SET(String(item.Nomenclature)),
					UserNotificationItems.Quantity.SET(Int32(item.Quantity))).
					WHERE(UserNotificationItems.ID.EQ(String(item.ID))))

		_, err := stmt.Exec(repo.db)
		if err != nil {
			failedItems = append(failedItems, item.ID)
		}
	}

	if len(failedItems) > 0 {
		return fmt.Errorf("failed to save following notification items: %s", failedItems)
	}

	slog.Info("notification item list saved", "user_id", user.UserID, "count", len(items))
	return nil
}

func (repo *RepositoryImpl) Delete(user *bootstrap.User, itemID string) error {
	if user == nil {
		return errors.New("valid user not found")
	}

	stmt := UserNotificationItems.DELETE().
		WHERE(UserNotificationItems.UserID.EQ(String(user.UserID)).
			AND(UserNotificationItems.ID.EQ(String(itemID))))

	_, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("error deleting notification item: %w", err)
	}

	slog.Info("notification item deleted", "user_id", user.UserID, "item_id", itemID)
	return nil
}

func (repo *RepositoryImpl) DeleteAllByNotification(user *bootstrap.User, notificationID string) error {
	if user == nil {
		return errors.New("valid user not found")
	}

	stmt := UserNotificationItems.DELETE().
		WHERE(UserNotificationItems.UserID.EQ(String(user.UserID)).
			AND(UserNotificationItems.NotificationID.EQ(String(notificationID))))

	_, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("error deleting items for notification: %w", err)
	}

	slog.Info("all notification items deleted", "user_id", user.UserID, "notification_id", notificationID)
	return nil
}
