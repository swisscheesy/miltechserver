package aggregates

import (
	"context"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Repository interface {
	GetListsWithItems(ctx context.Context, user *bootstrap.User, shopID string, limits ListTreeLimits) ([]response.ShopListWithItems, error)
	GetVehicleByIDForMember(ctx context.Context, user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error)
	GetVehicleNotificationsWithItems(ctx context.Context, vehicleID string, limits SnapshotLimits) ([]response.VehicleNotificationWithItems, error)
	GetVehicleRecentChanges(ctx context.Context, vehicleID string, limit int) ([]response.NotificationChangeWithUsername, error)
	GetVehicleServices(ctx context.Context, vehicleID string, limit int) ([]response.EquipmentServiceResponse, error)
	GetShopSnapshot(ctx context.Context, user *bootstrap.User, shopID string, options ShopSnapshotOptions) (*response.ShopSnapshotResponse, error)
	GetBootstrap(ctx context.Context, user *bootstrap.User, options BootstrapOptions) ([]response.ShopBootstrapSummary, error)
	GetEquipmentPmcsHistory(ctx context.Context, user *bootstrap.User) ([]response.EquipmentWithPmcsHistory, error)
}
