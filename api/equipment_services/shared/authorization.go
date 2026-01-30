package shared

import (
	"database/sql"
	"fmt"

	. "miltechserver/.gen/miltech_ng/public/table"
	shopsShared "miltechserver/api/shops/shared"
	"miltechserver/bootstrap"

	. "github.com/go-jet/jet/v2/postgres"
)

type Authorization struct {
	db       *sql.DB
	shopAuth shopsShared.ShopAuthorization
}

type shopIDResult struct {
	ShopID string `alias:"shop_vehicle.shop_id"`
}

type listShopIDResult struct {
	ShopID string `alias:"shop_lists.shop_id"`
}

type serviceShopIDResult struct {
	ShopID string `alias:"equipment_services.shop_id"`
}

type countResult struct {
	Count int64 `alias:"count"`
}

func NewAuthorization(db *sql.DB, shopAuth shopsShared.ShopAuthorization) *Authorization {
	return &Authorization{db: db, shopAuth: shopAuth}
}

func (auth *Authorization) RequireShopMember(user *bootstrap.User, shopID string) error {
	isMember, err := auth.shopAuth.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return fmt.Errorf("failed to verify shop membership: %w", err)
	}
	if !isMember {
		return ErrAccessDenied
	}
	return nil
}

func (auth *Authorization) GetShopIDForEquipment(user *bootstrap.User, equipmentID string) (string, error) {
	stmt := SELECT(ShopVehicle.ShopID).FROM(
		ShopVehicle.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(ShopVehicle.ShopID)),
	).WHERE(
		ShopVehicle.ID.EQ(String(equipmentID)).
			AND(ShopMembers.UserID.EQ(String(user.UserID))),
	)

	var result shopIDResult
	err := stmt.Query(auth.db, &result)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrEquipmentNotFound, err)
	}

	return result.ShopID, nil
}

func (auth *Authorization) GetShopIDForList(user *bootstrap.User, listID string) (string, error) {
	stmt := SELECT(ShopLists.ShopID).FROM(
		ShopLists.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(ShopLists.ShopID)),
	).WHERE(
		ShopLists.ID.EQ(String(listID)).
			AND(ShopMembers.UserID.EQ(String(user.UserID))),
	)

	var result listShopIDResult
	err := stmt.Query(auth.db, &result)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrListNotFound, err)
	}

	return result.ShopID, nil
}

func (auth *Authorization) RequireServiceAccessByID(user *bootstrap.User, serviceID string) (string, error) {
	stmt := SELECT(EquipmentServices.ShopID).FROM(EquipmentServices).WHERE(
		EquipmentServices.ID.EQ(String(serviceID)),
	)

	var result serviceShopIDResult
	err := stmt.Query(auth.db, &result)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrServiceNotFound, err)
	}

	if err := auth.RequireShopMember(user, result.ShopID); err != nil {
		return "", err
	}

	return result.ShopID, nil
}

func (auth *Authorization) CanUserModifyService(user *bootstrap.User, shopID, serviceID string) (bool, error) {
	isAdmin, err := auth.shopAuth.IsUserShopAdmin(user, shopID)
	if err != nil {
		return false, err
	}
	if isAdmin {
		return true, nil
	}

	return auth.isServiceOwner(user, serviceID)
}

func (auth *Authorization) CanUserDeleteService(user *bootstrap.User, shopID, serviceID string) (bool, error) {
	return auth.CanUserModifyService(user, shopID, serviceID)
}

func (auth *Authorization) isServiceOwner(user *bootstrap.User, serviceID string) (bool, error) {
	stmt := SELECT(COUNT(STAR)).FROM(EquipmentServices).WHERE(
		EquipmentServices.ID.EQ(String(serviceID)).
			AND(EquipmentServices.CreatedBy.EQ(String(user.UserID))),
	)

	var result countResult
	err := stmt.Query(auth.db, &result)
	if err != nil {
		return false, fmt.Errorf("failed to validate service ownership: %w", err)
	}

	return result.Count > 0, nil
}
