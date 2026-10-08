package aggregates

import (
	"context"

	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Repository interface {
	GetVehicleMaintenanceSnapshot(ctx context.Context, user *bootstrap.User, vehicleID string, limits SnapshotLimits) (*response.VehicleMaintenanceSnapshotResponse, error)
	GetListsWithItems(ctx context.Context, user *bootstrap.User, shopID string, limits ListTreeLimits) ([]response.ShopListWithItems, error)
	GetShopSnapshot(ctx context.Context, user *bootstrap.User, shopID string, options ShopSnapshotOptions) (*response.ShopSnapshotResponse, error)
	GetBootstrap(ctx context.Context, user *bootstrap.User, options BootstrapOptions) ([]response.ShopBootstrapSummary, error)
	GetEquipmentPmcsHistory(ctx context.Context, user *bootstrap.User) ([]response.EquipmentWithPmcsHistory, error)
}
