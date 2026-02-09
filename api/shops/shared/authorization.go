package shared

import (
	"database/sql"
	"errors"
	"fmt"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/bootstrap"

	. "github.com/go-jet/jet/v2/postgres"
)

type ShopAuthorization interface {
	IsUserMemberOfShop(user *bootstrap.User, shopID string) (bool, error)
	IsUserShopAdmin(user *bootstrap.User, shopID string) (bool, error)
	GetUserRoleInShop(user *bootstrap.User, shopID string) (string, error)

	CanUserModifyVehicle(user *bootstrap.User, vehicleID string) (bool, error)
	CanUserModifyList(user *bootstrap.User, listID string) (bool, error)
	CanUserModifyNotification(user *bootstrap.User, notificationID string) (bool, error)

	RequireShopMember(user *bootstrap.User, shopID string) error
	RequireShopAdmin(user *bootstrap.User, shopID string) error
}

type AuthorizationAware interface {
	WithAuthorization(auth ShopAuthorization) AuthorizationAware
}

type ShopAuthorizationImpl struct {
	db *sql.DB
}

func NewShopAuthorization(db *sql.DB) *ShopAuthorizationImpl {
	return &ShopAuthorizationImpl{db: db}
}

func (auth *ShopAuthorizationImpl) IsUserMemberOfShop(user *bootstrap.User, shopID string) (bool, error) {
	stmt := SELECT(Int(1).AS("exists")).
		FROM(ShopMembers).
		WHERE(
			ShopMembers.ShopID.EQ(String(shopID)).
				AND(ShopMembers.UserID.EQ(String(user.UserID))),
		).
		LIMIT(1)

	var result []struct {
		Exists int `sql:"exists"`
	}
	err := stmt.Query(auth.db, &result)
	if err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}

	return len(result) > 0, nil
}

func (auth *ShopAuthorizationImpl) IsUserShopAdmin(user *bootstrap.User, shopID string) (bool, error) {
	stmt := SELECT(Int(1).AS("exists")).
		FROM(ShopMembers).
		WHERE(
			ShopMembers.ShopID.EQ(String(shopID)).
				AND(ShopMembers.UserID.EQ(String(user.UserID))).
				AND(ShopMembers.Role.EQ(String("admin"))),
		).
		LIMIT(1)

	var result []struct {
		Exists int `sql:"exists"`
	}
	err := stmt.Query(auth.db, &result)
	if err != nil {
		return false, fmt.Errorf("failed to check admin status: %w", err)
	}

	return len(result) > 0, nil
}

func (auth *ShopAuthorizationImpl) GetUserRoleInShop(user *bootstrap.User, shopID string) (string, error) {
	stmt := SELECT(ShopMembers.Role).
		FROM(ShopMembers).
		WHERE(
			ShopMembers.ShopID.EQ(String(shopID)).
				AND(ShopMembers.UserID.EQ(String(user.UserID))),
		)

	var role string
	err := stmt.Query(auth.db, &role)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("user is not a member of this shop")
		}
		return "", fmt.Errorf("failed to get user role: %w", err)
	}

	return role, nil
}

func (auth *ShopAuthorizationImpl) CanUserModifyVehicle(user *bootstrap.User, vehicleID string) (bool, error) {
	stmt := SELECT(
		ShopVehicle.ID,
		ShopVehicle.ShopID,
		ShopVehicle.CreatorID,
	).FROM(ShopVehicle).
		WHERE(ShopVehicle.ID.EQ(String(vehicleID)))

	var vehicle model.ShopVehicle
	err := stmt.Query(auth.db, &vehicle)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, ErrVehicleNotFound
		}
		return false, fmt.Errorf("failed to get vehicle: %w", err)
	}

	isCreator := vehicle.CreatorID == user.UserID
	if isCreator {
		return true, nil
	}

	return auth.IsUserShopAdmin(user, vehicle.ShopID)
}

func (auth *ShopAuthorizationImpl) CanUserModifyList(user *bootstrap.User, listID string) (bool, error) {
	stmt := SELECT(
		ShopLists.ID,
		ShopLists.ShopID,
	).FROM(ShopLists).
		WHERE(ShopLists.ID.EQ(String(listID)))

	var list model.ShopLists
	err := stmt.Query(auth.db, &list)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, ErrListNotFound
		}
		return false, fmt.Errorf("failed to get list: %w", err)
	}

	adminOnlyLists, err := auth.getShopAdminOnlyListsSetting(list.ShopID)
	if err != nil {
		return false, err
	}

	if !adminOnlyLists {
		return true, nil
	}

	return auth.IsUserShopAdmin(user, list.ShopID)
}

func (auth *ShopAuthorizationImpl) CanUserModifyNotification(user *bootstrap.User, notificationID string) (bool, error) {
	stmt := SELECT(
		ShopVehicleNotifications.ID,
		ShopVehicleNotifications.ShopID,
	).FROM(ShopVehicleNotifications).
		WHERE(ShopVehicleNotifications.ID.EQ(String(notificationID)))

	var notification model.ShopVehicleNotifications
	err := stmt.Query(auth.db, &notification)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, ErrNotificationNotFound
		}
		return false, fmt.Errorf("failed to get notification: %w", err)
	}

	return auth.IsUserMemberOfShop(user, notification.ShopID)
}

func (auth *ShopAuthorizationImpl) RequireShopMember(user *bootstrap.User, shopID string) error {
	isMember, err := auth.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrShopAccessDenied
	}
	return nil
}

func (auth *ShopAuthorizationImpl) RequireShopAdmin(user *bootstrap.User, shopID string) error {
	isAdmin, err := auth.IsUserShopAdmin(user, shopID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return ErrShopAdminRequired
	}
	return nil
}

func (auth *ShopAuthorizationImpl) getShopAdminOnlyListsSetting(shopID string) (bool, error) {
	stmt := SELECT(Shops.AdminOnlyLists).
		FROM(Shops).
		WHERE(Shops.ID.EQ(String(shopID)))

	var result struct {
		AdminOnlyLists bool
	}
	err := stmt.Query(auth.db, &result)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, ErrShopNotFound
		}
		return false, fmt.Errorf("failed to get admin_only_lists setting: %w", err)
	}

	return result.AdminOnlyLists, nil
}
