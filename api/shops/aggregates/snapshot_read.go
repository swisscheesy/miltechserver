package aggregates

import (
	"context"
	"database/sql"
	"fmt"

	"miltechserver/api/response"
	shareddb "miltechserver/api/shared/db"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
)

// Membership and every component describe one committed version. Each later
// request starts a fresh snapshot and evaluates membership again.
func (repo *RepositoryImpl) withSnapshot(ctx context.Context, fn func(*sql.Tx) error) error {
	return shareddb.WithTxOptions(ctx, repo.db, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	}, fn)
}

func (repo *RepositoryImpl) GetListsWithItems(ctx context.Context, user *bootstrap.User, shopID string, limits ListTreeLimits) ([]response.ShopListWithItems, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
	var result []response.ShopListWithItems
	err := repo.withSnapshot(ctx, func(tx *sql.Tx) error {
		var isMember bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM shop_members WHERE shop_id=$1 AND user_id=$2)`, shopID, user.UserID).Scan(&isMember); err != nil {
			return fmt.Errorf("check list snapshot membership: %w", err)
		}
		if !isMember {
			return shared.ErrShopAccessDenied
		}
		var err error
		result, err = repo.getListsWithItems(ctx, tx, user, shopID, limits)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (repo *RepositoryImpl) GetShopSnapshot(ctx context.Context, user *bootstrap.User, shopID string, options ShopSnapshotOptions) (*response.ShopSnapshotResponse, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
	var result *response.ShopSnapshotResponse
	err := repo.withSnapshot(ctx, func(tx *sql.Tx) error {
		var err error
		result, err = repo.getShopSnapshot(ctx, tx, user, shopID, options)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (repo *RepositoryImpl) GetBootstrap(ctx context.Context, user *bootstrap.User, options BootstrapOptions) ([]response.ShopBootstrapSummary, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
	var result []response.ShopBootstrapSummary
	err := repo.withSnapshot(ctx, func(tx *sql.Tx) error {
		var err error
		result, err = repo.getBootstrap(ctx, tx, user, options)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (repo *RepositoryImpl) GetEquipmentPmcsHistory(ctx context.Context, user *bootstrap.User) ([]response.EquipmentWithPmcsHistory, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
	var result []response.EquipmentWithPmcsHistory
	err := repo.withSnapshot(ctx, func(tx *sql.Tx) error {
		var err error
		result, err = repo.getEquipmentPmcsHistory(ctx, tx, user)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (repo *RepositoryImpl) GetVehicleMaintenanceSnapshot(ctx context.Context, user *bootstrap.User, vehicleID string, limits SnapshotLimits) (*response.VehicleMaintenanceSnapshotResponse, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
	var result *response.VehicleMaintenanceSnapshotResponse
	err := repo.withSnapshot(ctx, func(tx *sql.Tx) error {
		var err error
		result, err = repo.getVehicleMaintenanceSnapshot(ctx, tx, user, vehicleID, limits)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (repo *RepositoryImpl) getVehicleMaintenanceSnapshot(ctx context.Context, tx *sql.Tx, user *bootstrap.User, vehicleID string, limits SnapshotLimits) (*response.VehicleMaintenanceSnapshotResponse, error) {
	vehicle, err := repo.getVehicleByIDForMember(ctx, tx, user, vehicleID)
	if err != nil {
		return nil, err
	}

	notifications, err := repo.getVehicleNotificationsWithItems(ctx, tx, vehicleID, limits)
	if err != nil {
		return nil, err
	}
	changes, err := repo.getVehicleRecentChanges(ctx, tx, vehicleID, limits.ChangesLimit)
	if err != nil {
		return nil, err
	}
	services, err := repo.getVehicleServices(ctx, tx, vehicleID, limits.ServicesLimit)
	if err != nil {
		return nil, err
	}
	if notifications == nil {
		notifications = []response.VehicleNotificationWithItems{}
	}
	if changes == nil {
		changes = []response.NotificationChangeWithUsername{}
	}
	if services == nil {
		services = []response.EquipmentServiceResponse{}
	}
	itemCount := int64(0)
	for _, notification := range notifications {
		itemCount += int64(len(notification.Items))
	}
	return &response.VehicleMaintenanceSnapshotResponse{
		Vehicle:       *vehicle,
		Notifications: notifications,
		RecentChanges: changes,
		Services:      services,
		Counts: response.VehicleMaintenanceSnapshotCounts{
			Notifications:     int64(len(notifications)),
			NotificationItems: itemCount,
			RecentChanges:     int64(len(changes)),
			Services:          int64(len(services)),
		},
		Limits: response.VehicleMaintenanceSnapshotLimits{
			Notifications:                    optionalLimitPtr(limits.NotificationsLimit),
			NotificationItemsPerNotification: optionalLimitPtr(limits.NotificationItemsLimit),
			Services:                         optionalLimitPtr(limits.ServicesLimit),
			RecentChanges:                    optionalLimitPtr(limits.ChangesLimit),
		},
	}, nil
}
