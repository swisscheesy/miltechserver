package repository

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

type ShopsRepositoryImpl struct {
	Db *sql.DB
}

func NewShopsRepositoryImpl(db *sql.DB) *ShopsRepositoryImpl {
	return &ShopsRepositoryImpl{Db: db}
}

// Shop Operations
func (repo *ShopsRepositoryImpl) CreateShop(user *bootstrap.User, shop model.Shops) (*model.Shops, error) {
	stmt := Shops.INSERT(
		Shops.ID,
		Shops.Name,
		Shops.Details,
		Shops.PasswordHash,
		Shops.CreatedBy,
		Shops.CreatedAt,
		Shops.UpdatedAt,
	).MODEL(shop).RETURNING(Shops.AllColumns)

	var createdShop model.Shops
	err := stmt.Query(repo.Db, &createdShop)
	if err != nil {
		return nil, fmt.Errorf("failed to create shop: %w", err)
	}

	slog.Info("Shop created in database", "shop_id", shop.ID, "created_by", user.UserID)
	return &createdShop, nil
}

func (repo *ShopsRepositoryImpl) DeleteShop(user *bootstrap.User, shopID string) error {
	// Delete shop will cascade delete related records (members, messages, vehicles, etc.)
	stmt := Shops.DELETE().WHERE(
		Shops.ID.EQ(String(shopID)).
			AND(Shops.CreatedBy.EQ(String(user.UserID))),
	)

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to delete shop: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("shop not found or user not authorized to delete")
	}

	slog.Info("Shop deleted from database", "shop_id", shopID, "deleted_by", user.UserID)
	return nil
}

func (repo *ShopsRepositoryImpl) GetShopsByUser(user *bootstrap.User) ([]model.Shops, error) {
	stmt := SELECT(Shops.AllColumns).
		FROM(
			Shops.
				INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(Shops.ID)),
		).
		WHERE(ShopMembers.UserID.EQ(String(user.UserID))).
		ORDER_BY(Shops.CreatedAt.DESC())

	var shops []model.Shops
	err := stmt.Query(repo.Db, &shops)
	if err != nil {
		return nil, fmt.Errorf("failed to get shops for user: %w", err)
	}

	slog.Info("Shops retrieved for user", "user_id", user.UserID, "count", len(shops))
	return shops, nil
}

func (repo *ShopsRepositoryImpl) GetShopByID(user *bootstrap.User, shopID string) (*model.Shops, error) {
	stmt := SELECT(Shops.AllColumns).
		FROM(Shops).
		WHERE(Shops.ID.EQ(String(shopID)))

	var shop model.Shops
	err := stmt.Query(repo.Db, &shop)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop: %w", err)
	}

	return &shop, nil
}

func (repo *ShopsRepositoryImpl) IsUserShopAdmin(user *bootstrap.User, shopID string) (bool, error) {
	stmt := SELECT(COUNT(ShopMembers.ID)).
		FROM(ShopMembers).
		WHERE(
			ShopMembers.ShopID.EQ(String(shopID)).
				AND(ShopMembers.UserID.EQ(String(user.UserID))).
				AND(ShopMembers.Role.EQ(String("admin"))),
		)

	var count int64
	err := stmt.Query(repo.Db, &count)
	if err != nil {
		return false, fmt.Errorf("failed to check admin status: %w", err)
	}

	return count > 0, nil
}

// Shop Member Operations
func (repo *ShopsRepositoryImpl) AddMemberToShop(user *bootstrap.User, shopID string, role string) error {
	member := model.ShopMembers{
		ID:     fmt.Sprintf("%s_%s", shopID, user.UserID), // Simple composite ID
		ShopID: shopID,
		UserID: user.UserID,
		Role:   role,
	}

	stmt := ShopMembers.INSERT(
		ShopMembers.ID,
		ShopMembers.ShopID,
		ShopMembers.UserID,
		ShopMembers.Role,
		ShopMembers.JoinedAt,
	).MODEL(member).
		ON_CONFLICT(ShopMembers.ShopID, ShopMembers.UserID).
		DO_UPDATE(SET(ShopMembers.Role.SET(String(role))))

	_, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to add member to shop: %w", err)
	}

	slog.Info("Member added to shop", "shop_id", shopID, "user_id", user.UserID, "role", role)
	return nil
}

func (repo *ShopsRepositoryImpl) RemoveMemberFromShop(user *bootstrap.User, shopID string, targetUserID string) error {
	stmt := ShopMembers.DELETE().
		WHERE(
			ShopMembers.ShopID.EQ(String(shopID)).
				AND(ShopMembers.UserID.EQ(String(targetUserID))),
		)

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to remove member from shop: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("member not found in shop")
	}

	slog.Info("Member removed from shop", "shop_id", shopID, "removed_user_id", targetUserID, "removed_by", user.UserID)
	return nil
}

func (repo *ShopsRepositoryImpl) GetShopMembers(user *bootstrap.User, shopID string) ([]model.ShopMembers, error) {
	stmt := SELECT(ShopMembers.AllColumns).
		FROM(ShopMembers).
		WHERE(ShopMembers.ShopID.EQ(String(shopID))).
		ORDER_BY(ShopMembers.JoinedAt.ASC())

	var members []model.ShopMembers
	err := stmt.Query(repo.Db, &members)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop members: %w", err)
	}

	return members, nil
}

func (repo *ShopsRepositoryImpl) IsUserMemberOfShop(user *bootstrap.User, shopID string) (bool, error) {
	stmt := SELECT(COUNT(ShopMembers.ID)).
		FROM(ShopMembers).
		WHERE(
			ShopMembers.ShopID.EQ(String(shopID)).
				AND(ShopMembers.UserID.EQ(String(user.UserID))),
		)

	var count int64
	err := stmt.Query(repo.Db, &count)
	if err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}

	return count > 0, nil
}

// Shop Invite Code Operations
func (repo *ShopsRepositoryImpl) CreateInviteCode(user *bootstrap.User, inviteCode model.ShopInviteCodes) (*model.ShopInviteCodes, error) {
	stmt := ShopInviteCodes.INSERT(
		ShopInviteCodes.ID,
		ShopInviteCodes.ShopID,
		ShopInviteCodes.Code,
		ShopInviteCodes.CreatedBy,
		ShopInviteCodes.ExpiresAt,
		ShopInviteCodes.MaxUses,
		ShopInviteCodes.CurrentUses,
		ShopInviteCodes.IsActive,
		ShopInviteCodes.CreatedAt,
	).MODEL(inviteCode).RETURNING(ShopInviteCodes.AllColumns)

	var createdCode model.ShopInviteCodes
	err := stmt.Query(repo.Db, &createdCode)
	if err != nil {
		return nil, fmt.Errorf("failed to create invite code: %w", err)
	}

	return &createdCode, nil
}

func (repo *ShopsRepositoryImpl) GetInviteCodeByCode(code string) (*model.ShopInviteCodes, error) {
	stmt := SELECT(ShopInviteCodes.AllColumns).
		FROM(ShopInviteCodes).
		WHERE(ShopInviteCodes.Code.EQ(String(code)))

	var inviteCode model.ShopInviteCodes
	err := stmt.Query(repo.Db, &inviteCode)
	if err != nil {
		return nil, fmt.Errorf("invite code not found: %w", err)
	}

	return &inviteCode, nil
}

func (repo *ShopsRepositoryImpl) GetInviteCodesByShop(user *bootstrap.User, shopID string) ([]model.ShopInviteCodes, error) {
	stmt := SELECT(ShopInviteCodes.AllColumns).
		FROM(ShopInviteCodes).
		WHERE(ShopInviteCodes.ShopID.EQ(String(shopID))).
		ORDER_BY(ShopInviteCodes.CreatedAt.DESC())

	var codes []model.ShopInviteCodes
	err := stmt.Query(repo.Db, &codes)
	if err != nil {
		return nil, fmt.Errorf("failed to get invite codes: %w", err)
	}

	return codes, nil
}

func (repo *ShopsRepositoryImpl) IncrementInviteCodeUsage(codeID string) error {
	stmt := ShopInviteCodes.UPDATE(
		ShopInviteCodes.CurrentUses,
	).SET(
		ShopInviteCodes.CurrentUses.SET(ShopInviteCodes.CurrentUses.ADD(Int32(1))),
	).WHERE(ShopInviteCodes.ID.EQ(String(codeID)))

	_, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to increment invite code usage: %w", err)
	}

	return nil
}

func (repo *ShopsRepositoryImpl) DeactivateInviteCode(user *bootstrap.User, codeID string) error {
	stmt := ShopInviteCodes.UPDATE(
		ShopInviteCodes.IsActive,
	).SET(
		ShopInviteCodes.IsActive.SET(Bool(false)),
	).WHERE(ShopInviteCodes.ID.EQ(String(codeID)))

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to deactivate invite code: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("invite code not found")
	}

	return nil
}

// Shop Message Operations
func (repo *ShopsRepositoryImpl) CreateShopMessage(user *bootstrap.User, message model.ShopMessages) (*model.ShopMessages, error) {
	stmt := ShopMessages.INSERT(
		ShopMessages.ID,
		ShopMessages.ShopID,
		ShopMessages.UserID,
		ShopMessages.Message,
		ShopMessages.CreatedAt,
		ShopMessages.UpdatedAt,
		ShopMessages.IsEdited,
	).MODEL(message).RETURNING(ShopMessages.AllColumns)

	var createdMessage model.ShopMessages
	err := stmt.Query(repo.Db, &createdMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to create shop message: %w", err)
	}

	return &createdMessage, nil
}

func (repo *ShopsRepositoryImpl) GetShopMessages(user *bootstrap.User, shopID string) ([]model.ShopMessages, error) {
	stmt := SELECT(ShopMessages.AllColumns).
		FROM(ShopMessages).
		WHERE(ShopMessages.ShopID.EQ(String(shopID))).
		ORDER_BY(ShopMessages.CreatedAt.ASC())

	var messages []model.ShopMessages
	err := stmt.Query(repo.Db, &messages)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop messages: %w", err)
	}

	return messages, nil
}

func (repo *ShopsRepositoryImpl) UpdateShopMessage(user *bootstrap.User, message model.ShopMessages) error {
	stmt := ShopMessages.UPDATE(
		ShopMessages.Message,
		ShopMessages.UpdatedAt,
		ShopMessages.IsEdited,
	).SET(
		ShopMessages.Message.SET(String(message.Message)),
		ShopMessages.UpdatedAt.SET(TimestampzT(*message.UpdatedAt)),
		ShopMessages.IsEdited.SET(Bool(*message.IsEdited)),
	).WHERE(
		ShopMessages.ID.EQ(String(message.ID)).
			AND(ShopMessages.UserID.EQ(String(user.UserID))),
	)

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to update shop message: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("message not found or user not authorized to update")
	}

	return nil
}

func (repo *ShopsRepositoryImpl) DeleteShopMessage(user *bootstrap.User, messageID string) error {
	stmt := ShopMessages.DELETE().
		WHERE(
			ShopMessages.ID.EQ(String(messageID)).
				AND(ShopMessages.UserID.EQ(String(user.UserID))),
		)

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to delete shop message: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("message not found or user not authorized to delete")
	}

	return nil
}

// Shop Vehicle Operations
func (repo *ShopsRepositoryImpl) CreateShopVehicle(user *bootstrap.User, vehicle model.ShopVehicle) (*model.ShopVehicle, error) {
	stmt := ShopVehicle.INSERT(
		ShopVehicle.ID,
		ShopVehicle.CreatorID,
		ShopVehicle.Niin,
		ShopVehicle.Admin,
		ShopVehicle.Model,
		ShopVehicle.Serial,
		ShopVehicle.Uoc,
		ShopVehicle.Mileage,
		ShopVehicle.Hours,
		ShopVehicle.Comment,
		ShopVehicle.SaveTime,
		ShopVehicle.LastUpdated,
		ShopVehicle.ShopID,
	).MODEL(vehicle).RETURNING(ShopVehicle.AllColumns)

	var createdVehicle model.ShopVehicle
	err := stmt.Query(repo.Db, &createdVehicle)
	if err != nil {
		return nil, fmt.Errorf("failed to create shop vehicle: %w", err)
	}

	return &createdVehicle, nil
}

func (repo *ShopsRepositoryImpl) GetShopVehicles(user *bootstrap.User, shopID string) ([]model.ShopVehicle, error) {
	stmt := SELECT(ShopVehicle.AllColumns).
		FROM(ShopVehicle).
		WHERE(ShopVehicle.ShopID.EQ(String(shopID))).
		ORDER_BY(ShopVehicle.SaveTime.DESC())

	var vehicles []model.ShopVehicle
	err := stmt.Query(repo.Db, &vehicles)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop vehicles: %w", err)
	}

	return vehicles, nil
}

func (repo *ShopsRepositoryImpl) GetShopVehicleByID(user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error) {
	stmt := SELECT(ShopVehicle.AllColumns).
		FROM(ShopVehicle).
		WHERE(ShopVehicle.ID.EQ(String(vehicleID)))

	var vehicle model.ShopVehicle
	err := stmt.Query(repo.Db, &vehicle)
	if err != nil {
		return nil, fmt.Errorf("shop vehicle not found: %w", err)
	}

	return &vehicle, nil
}

func (repo *ShopsRepositoryImpl) UpdateShopVehicle(user *bootstrap.User, vehicle model.ShopVehicle) error {
	stmt := ShopVehicle.UPDATE(
		ShopVehicle.Model,
		ShopVehicle.Serial,
		ShopVehicle.Uoc,
		ShopVehicle.Mileage,
		ShopVehicle.Hours,
		ShopVehicle.Comment,
		ShopVehicle.LastUpdated,
	).SET(
		ShopVehicle.Model.SET(String(vehicle.Model)),
		ShopVehicle.Serial.SET(String(vehicle.Serial)),
		ShopVehicle.Uoc.SET(String(vehicle.Uoc)),
		ShopVehicle.Mileage.SET(Int32(vehicle.Mileage)),
		ShopVehicle.Hours.SET(Int32(vehicle.Hours)),
		ShopVehicle.Comment.SET(String(vehicle.Comment)),
		ShopVehicle.LastUpdated.SET(TimestampzT(vehicle.LastUpdated)),
	).WHERE(ShopVehicle.ID.EQ(String(vehicle.ID)))

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to update shop vehicle: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("vehicle not found")
	}

	return nil
}

func (repo *ShopsRepositoryImpl) DeleteShopVehicle(user *bootstrap.User, vehicleID string) error {
	stmt := ShopVehicle.DELETE().
		WHERE(ShopVehicle.ID.EQ(String(vehicleID)))

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to delete shop vehicle: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("vehicle not found")
	}

	return nil
}

// Shop Vehicle Notification Operations
func (repo *ShopsRepositoryImpl) CreateVehicleNotification(user *bootstrap.User, notification model.ShopVehicleNotifications) (*model.ShopVehicleNotifications, error) {
	stmt := ShopVehicleNotifications.INSERT(
		ShopVehicleNotifications.ID,
		ShopVehicleNotifications.ShopID,
		ShopVehicleNotifications.VehicleID,
		ShopVehicleNotifications.Title,
		ShopVehicleNotifications.Description,
		ShopVehicleNotifications.Type,
		ShopVehicleNotifications.Completed,
		ShopVehicleNotifications.SaveTime,
		ShopVehicleNotifications.LastUpdated,
	).MODEL(notification).RETURNING(ShopVehicleNotifications.AllColumns)

	var createdNotification model.ShopVehicleNotifications
	err := stmt.Query(repo.Db, &createdNotification)
	if err != nil {
		return nil, fmt.Errorf("failed to create vehicle notification: %w", err)
	}

	return &createdNotification, nil
}

func (repo *ShopsRepositoryImpl) GetVehicleNotifications(user *bootstrap.User, vehicleID string) ([]model.ShopVehicleNotifications, error) {
	stmt := SELECT(ShopVehicleNotifications.AllColumns).
		FROM(ShopVehicleNotifications).
		WHERE(ShopVehicleNotifications.VehicleID.EQ(String(vehicleID))).
		ORDER_BY(ShopVehicleNotifications.SaveTime.DESC())

	var notifications []model.ShopVehicleNotifications
	err := stmt.Query(repo.Db, &notifications)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle notifications: %w", err)
	}

	return notifications, nil
}

func (repo *ShopsRepositoryImpl) GetVehicleNotificationByID(user *bootstrap.User, notificationID string) (*model.ShopVehicleNotifications, error) {
	stmt := SELECT(ShopVehicleNotifications.AllColumns).
		FROM(ShopVehicleNotifications).
		WHERE(ShopVehicleNotifications.ID.EQ(String(notificationID)))

	var notification model.ShopVehicleNotifications
	err := stmt.Query(repo.Db, &notification)
	if err != nil {
		return nil, fmt.Errorf("vehicle notification not found: %w", err)
	}

	return &notification, nil
}

func (repo *ShopsRepositoryImpl) UpdateVehicleNotification(user *bootstrap.User, notification model.ShopVehicleNotifications) error {
	stmt := ShopVehicleNotifications.UPDATE(
		ShopVehicleNotifications.Title,
		ShopVehicleNotifications.Description,
		ShopVehicleNotifications.Type,
		ShopVehicleNotifications.Completed,
		ShopVehicleNotifications.LastUpdated,
	).SET(
		ShopVehicleNotifications.Title.SET(String(notification.Title)),
		ShopVehicleNotifications.Description.SET(String(notification.Description)),
		ShopVehicleNotifications.Type.SET(String(notification.Type)),
		ShopVehicleNotifications.Completed.SET(Bool(notification.Completed)),
		ShopVehicleNotifications.LastUpdated.SET(TimestampzT(notification.LastUpdated)),
	).WHERE(ShopVehicleNotifications.ID.EQ(String(notification.ID)))

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to update vehicle notification: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("notification not found")
	}

	return nil
}

func (repo *ShopsRepositoryImpl) DeleteVehicleNotification(user *bootstrap.User, notificationID string) error {
	stmt := ShopVehicleNotifications.DELETE().
		WHERE(ShopVehicleNotifications.ID.EQ(String(notificationID)))

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to delete vehicle notification: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("notification not found")
	}

	return nil
}

// Shop Notification Item Operations
func (repo *ShopsRepositoryImpl) CreateNotificationItem(user *bootstrap.User, item model.ShopNotificationItems) (*model.ShopNotificationItems, error) {
	stmt := ShopNotificationItems.INSERT(
		ShopNotificationItems.ID,
		ShopNotificationItems.ShopID,
		ShopNotificationItems.NotificationID,
		ShopNotificationItems.Niin,
		ShopNotificationItems.Nomenclature,
		ShopNotificationItems.Quantity,
		ShopNotificationItems.SaveTime,
	).MODEL(item).RETURNING(ShopNotificationItems.AllColumns)

	var createdItem model.ShopNotificationItems
	err := stmt.Query(repo.Db, &createdItem)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification item: %w", err)
	}

	return &createdItem, nil
}

func (repo *ShopsRepositoryImpl) GetNotificationItems(user *bootstrap.User, notificationID string) ([]model.ShopNotificationItems, error) {
	stmt := SELECT(ShopNotificationItems.AllColumns).
		FROM(ShopNotificationItems).
		WHERE(ShopNotificationItems.NotificationID.EQ(String(notificationID))).
		ORDER_BY(ShopNotificationItems.SaveTime.ASC())

	var items []model.ShopNotificationItems
	err := stmt.Query(repo.Db, &items)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification items: %w", err)
	}

	return items, nil
}

func (repo *ShopsRepositoryImpl) CreateNotificationItemList(user *bootstrap.User, items []model.ShopNotificationItems) error {
	if len(items) == 0 {
		return nil
	}

	stmt := ShopNotificationItems.INSERT(
		ShopNotificationItems.ID,
		ShopNotificationItems.ShopID,
		ShopNotificationItems.NotificationID,
		ShopNotificationItems.Niin,
		ShopNotificationItems.Nomenclature,
		ShopNotificationItems.Quantity,
		ShopNotificationItems.SaveTime,
	).MODELS(items)

	_, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to create notification items: %w", err)
	}

	return nil
}

func (repo *ShopsRepositoryImpl) DeleteNotificationItem(user *bootstrap.User, itemID string) error {
	stmt := ShopNotificationItems.DELETE().
		WHERE(ShopNotificationItems.ID.EQ(String(itemID)))

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to delete notification item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("notification item not found")
	}

	return nil
}

func (repo *ShopsRepositoryImpl) DeleteNotificationItemList(user *bootstrap.User, itemIDs []string) error {
	if len(itemIDs) == 0 {
		return nil
	}

	// Convert string slice to expressions for the IN clause
	var expressions []Expression
	for _, id := range itemIDs {
		expressions = append(expressions, String(id))
	}

	stmt := ShopNotificationItems.DELETE().
		WHERE(ShopNotificationItems.ID.IN(expressions...))

	_, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to delete notification items: %w", err)
	}

	return nil
}
