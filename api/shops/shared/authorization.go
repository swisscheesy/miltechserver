package shared

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/bootstrap"
	"sort"

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

	isMember, err := auth.IsUserMemberOfShop(user, vehicle.ShopID)
	if err != nil || !isMember {
		return false, err
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

	isMember, err := auth.IsUserMemberOfShop(user, list.ShopID)
	if err != nil || !isMember {
		return false, err
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

func CanManageListItems(member, admin, adminOnly bool) bool {
	return member && (admin || !adminOnly)
}

func CanDeleteList(member, admin, adminOnly, creator bool) bool {
	return member && (admin || (!adminOnly && creator))
}

// authorizationError keeps driver details out of Error(), which legacy HTTP
// middleware publishes verbatim, while preserving the cause for diagnostics.
type authorizationError struct {
	public error
	cause  error
}

func (err *authorizationError) Error() string        { return err.public.Error() }
func (err *authorizationError) Unwrap() error        { return err.cause }
func (err *authorizationError) Is(target error) bool { return errors.Is(err.public, target) }

func authorizationQueryError(operation string, missing error, cause error) error {
	public := missing
	if !errors.Is(cause, sql.ErrNoRows) {
		public = errors.New("failed to verify shop authorization")
		slog.Error("Shop authorization query failed", "operation", operation, "error", cause)
	}
	return &authorizationError{public: public, cause: cause}
}

// RequireShopMember holds current membership and role until the caller commits.
func RequireShopMember(ctx context.Context, tx *sql.Tx, shopID, userID string) (bool, error) {
	var role string
	err := tx.QueryRowContext(ctx, `SELECT role FROM shop_members WHERE shop_id=$1 AND user_id=$2 FOR UPDATE`, shopID, userID).Scan(&role)
	if err != nil {
		return false, authorizationQueryError("lock shop membership", ErrShopAccessDenied, err)
	}
	return role == "admin", nil
}

// LockShopMutation orders settings before membership locks for all participating
// writers, including role changes, so authorization stays valid until commit.
func LockShopMutation(ctx context.Context, tx *sql.Tx, shopID, userID string) (admin, adminOnly bool, err error) {
	err = tx.QueryRowContext(ctx, `SELECT admin_only_lists FROM shops WHERE id=$1 FOR UPDATE`, shopID).Scan(&adminOnly)
	if err != nil {
		return false, false, authorizationQueryError("lock shop settings", ErrShopNotFound, err)
	}
	admin, err = RequireShopMember(ctx, tx, shopID, userID)
	return
}

// AuthorizeListMutation resolves ownership from persisted IDs, rejects an entire
// cross-Shop batch, then locks and rechecks each resource before any write.
func AuthorizeListMutation(ctx context.Context, tx *sql.Tx, userID, shopID string, listIDs, itemIDs []string, deletingList bool) error {
	listIDs = append([]string(nil), listIDs...)
	itemIDs = append([]string(nil), itemIDs...)
	sort.Strings(itemIDs)
	itemLists := make(map[string]string, len(itemIDs))
	for _, id := range itemIDs {
		var listID string
		if err := tx.QueryRowContext(ctx, `SELECT list_id FROM shop_list_items WHERE id=$1`, id).Scan(&listID); err != nil {
			return authorizationQueryError("resolve or lock list item", errors.New("list item not found"), err)
		}
		itemLists[id] = listID
		listIDs = append(listIDs, listID)
	}
	sort.Strings(listIDs)
	owners := make(map[string]string, len(listIDs))
	for _, id := range listIDs {
		if _, exists := owners[id]; exists {
			continue
		}
		var owner string
		if err := tx.QueryRowContext(ctx, `SELECT shop_id FROM shop_lists WHERE id=$1`, id).Scan(&owner); err != nil {
			return authorizationQueryError("resolve or lock list", ErrListNotFound, err)
		}
		if shopID == "" {
			shopID = owner
		}
		if owner != shopID {
			return ErrListAccessDenied
		}
		owners[id] = owner
	}
	admin, only, err := LockShopMutation(ctx, tx, shopID, userID)
	if err != nil {
		return err
	}
	if !CanManageListItems(true, admin, only) {
		return ErrListAccessDenied
	}
	for _, id := range listIDs {
		var owner, creator string
		if err := tx.QueryRowContext(ctx, `SELECT shop_id,created_by FROM shop_lists WHERE id=$1 FOR UPDATE`, id).Scan(&owner, &creator); err != nil {
			return authorizationQueryError("resolve or lock list", ErrListNotFound, err)
		}
		if owner != shopID {
			return ErrListAccessDenied
		}
		if deletingList && !CanDeleteList(true, admin, only, creator == userID) {
			return ErrListAccessDenied
		}
	}
	for _, id := range itemIDs {
		var listID string
		if err := tx.QueryRowContext(ctx, `SELECT list_id FROM shop_list_items WHERE id=$1 FOR UPDATE`, id).Scan(&listID); err != nil {
			return authorizationQueryError("resolve or lock list item", errors.New("list item not found"), err)
		}
		if listID != itemLists[id] {
			return ErrListAccessDenied
		}
	}
	return nil
}

// AuthorizeOwnedMutation preserves author-only edits and author/admin deletes.
// Resource names are chosen by server code, never interpolated from a request.
func AuthorizeOwnedMutation(ctx context.Context, tx *sql.Tx, userID, resource, id string, allowAdmin bool) error {
	var query string
	switch resource {
	case "message":
		query = `SELECT shop_id,user_id FROM shop_messages WHERE id=$1`
	case "vehicle":
		query = `SELECT shop_id,creator_id FROM shop_vehicle WHERE id=$1`
	default:
		return errors.New("unknown shop resource")
	}
	var shopID, author string
	if err := tx.QueryRowContext(ctx, query, id).Scan(&shopID, &author); err != nil {
		return authorizationQueryError("resolve or lock "+resource, errors.New("resource not found"), err)
	}
	admin, _, err := LockShopMutation(ctx, tx, shopID, userID)
	if err != nil {
		return err
	}
	var lockedShop string
	if err := tx.QueryRowContext(ctx, query+" FOR UPDATE", id).Scan(&lockedShop, &author); err != nil {
		return authorizationQueryError("resolve or lock "+resource, errors.New("resource not found"), err)
	}
	if lockedShop != shopID || (author != userID && !(allowAdmin && admin)) {
		return ErrShopAccessDenied
	}
	return nil
}
