package shared

import (
	"context"
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

func (auth *Authorization) RequireShopMember(ctx context.Context, user *bootstrap.User, shopID string) error {
	isMember, err := auth.shopAuth.IsUserMemberOfShop(ctx, user, shopID)
	if err != nil {
		return fmt.Errorf("failed to verify shop membership: %w", err)
	}
	if !isMember {
		return ErrAccessDenied
	}
	return nil
}

func (auth *Authorization) GetShopIDForEquipment(ctx context.Context, user *bootstrap.User, equipmentID string) (string, error) {
	stmt := SELECT(ShopVehicle.ShopID).FROM(
		ShopVehicle.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(ShopVehicle.ShopID)),
	).WHERE(
		ShopVehicle.ID.EQ(String(equipmentID)).
			AND(ShopMembers.UserID.EQ(String(user.UserID))),
	)

	var result shopIDResult
	err := stmt.QueryContext(ctx, auth.db, &result)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrEquipmentNotFound, err)
	}

	return result.ShopID, nil
}

func (auth *Authorization) GetShopIDForList(ctx context.Context, user *bootstrap.User, listID string) (string, error) {
	stmt := SELECT(ShopLists.ShopID).FROM(
		ShopLists.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(ShopLists.ShopID)),
	).WHERE(
		ShopLists.ID.EQ(String(listID)).
			AND(ShopMembers.UserID.EQ(String(user.UserID))),
	)

	var result listShopIDResult
	err := stmt.QueryContext(ctx, auth.db, &result)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrListNotFound, err)
	}

	return result.ShopID, nil
}

func (auth *Authorization) RequireServiceAccessByID(ctx context.Context, user *bootstrap.User, serviceID string) (string, error) {
	stmt := SELECT(EquipmentServices.ShopID).FROM(EquipmentServices).WHERE(
		EquipmentServices.ID.EQ(String(serviceID)),
	)

	var result serviceShopIDResult
	err := stmt.QueryContext(ctx, auth.db, &result)
	if err != nil {
		if shopsShared.ErrorsIsNoRows(err) {
			return "", &shopsShared.Failure{Code: "service_not_found", PublicMessage: ErrServiceNotFound.Error(), Status: 404, Cause: err}
		}
		return "", fmt.Errorf("failed to verify service access: %w", err)
	}

	if err := auth.RequireShopMember(ctx, user, result.ShopID); err != nil {
		return "", err
	}

	return result.ShopID, nil
}

func (auth *Authorization) CanUserModifyService(ctx context.Context, user *bootstrap.User, shopID, serviceID string) (bool, error) {
	actualShop, err := auth.RequireServiceAccessByID(ctx, user, serviceID)
	if err != nil {
		return false, err
	}
	if actualShop != shopID {
		return false, ErrAccessDenied
	}
	isAdmin, err := auth.shopAuth.IsUserShopAdmin(ctx, user, actualShop)
	if err != nil {
		return false, err
	}
	if isAdmin {
		return true, nil
	}

	return auth.isServiceOwner(ctx, user, serviceID)
}

func (auth *Authorization) CanUserDeleteService(ctx context.Context, user *bootstrap.User, shopID, serviceID string) (bool, error) {
	return auth.CanUserModifyService(ctx, user, shopID, serviceID)
}

func (auth *Authorization) isServiceOwner(ctx context.Context, user *bootstrap.User, serviceID string) (bool, error) {
	stmt := SELECT(COUNT(STAR)).FROM(EquipmentServices).WHERE(
		EquipmentServices.ID.EQ(String(serviceID)).
			AND(EquipmentServices.CreatedBy.EQ(String(user.UserID))),
	)

	var result countResult
	err := stmt.QueryContext(ctx, auth.db, &result)
	if err != nil {
		return false, fmt.Errorf("failed to validate service ownership: %w", err)
	}

	return result.Count > 0, nil
}

// LockServiceMutation binds authority to the persisted Shop and holds it until
// commit. Lists are locked before the service to match list cleanup writers.
func LockServiceMutation(ctx context.Context, tx *sql.Tx, user *bootstrap.User, shopID, serviceID string, newListIDs ...string) error {
	if user == nil {
		return ErrUnauthorizedUser
	}
	var actualShop, oldList, creator string
	if err := tx.QueryRowContext(ctx, `SELECT shop_id FROM equipment_services WHERE id=$1`, serviceID).Scan(&actualShop); err != nil {
		return serviceAuthorityError(err)
	}
	if actualShop != shopID {
		return ErrAccessDenied
	}
	admin, _, err := shopsShared.LockShopMutation(ctx, tx, actualShop, user.UserID)
	if err != nil {
		return err
	}
	// Refresh references after the Shop lock: a preceding writer may have changed them.
	if err := tx.QueryRowContext(ctx, `SELECT list_id,created_by FROM equipment_services WHERE id=$1 AND shop_id=$2`, serviceID, shopID).Scan(&oldList, &creator); err != nil {
		return serviceAuthorityError(err)
	}
	listIDs := append([]string{oldList}, newListIDs...)
	if err := shopsShared.LockReferencedLists(ctx, tx, shopID, listIDs...); err != nil {
		return err
	}
	var lockedList string
	if err := tx.QueryRowContext(ctx, `SELECT shop_id,list_id,created_by FROM equipment_services WHERE id=$1 FOR UPDATE`, serviceID).Scan(&actualShop, &lockedList, &creator); err != nil {
		return serviceAuthorityError(err)
	}
	if actualShop != shopID || lockedList != oldList || (creator != user.UserID && !admin) {
		return ErrAccessDenied
	}
	return nil
}

func serviceAuthorityError(err error) error {
	if shopsShared.ErrorsIsNoRows(err) {
		return &shopsShared.Failure{Code: "service_not_found", PublicMessage: ErrServiceNotFound.Error(), Status: 404, Cause: err}
	}
	return &shopsShared.Failure{Code: "internal_error", PublicMessage: "Unable to verify service", Status: 500, Cause: err}
}
