package aggregates

import (
	"context"

	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type SnapshotLimits struct {
	NotificationsLimit     int
	NotificationItemsLimit int
	ServicesLimit          int
	ChangesLimit           int
}

type ListTreeLimits struct {
	ListsLimit        int
	ItemsLimitPerList int
}

type ShopSnapshotOptions struct {
	Includes               map[string]bool
	VehiclesLimit          int
	ListsLimit             int
	ItemsLimitPerList      int
	NotificationsLimit     int
	NotificationItemsLimit int
	MessageLimit           int
	ChangesLimit           int
	ServicesLimit          int
}

type BootstrapOptions struct {
	EquipmentLimitPerShop int
	IncludeEmptyEquipment bool
}

type Service interface {
	GetListsWithItems(ctx context.Context, user *bootstrap.User, shopID string, limits ListTreeLimits) (*response.ShopListsWithItemsResponse, error)
	GetVehicleMaintenanceSnapshot(ctx context.Context, user *bootstrap.User, vehicleID string, limits SnapshotLimits) (*response.VehicleMaintenanceSnapshotResponse, error)
	GetShopSnapshot(ctx context.Context, user *bootstrap.User, shopID string, options ShopSnapshotOptions) (*response.ShopSnapshotResponse, error)
	GetBootstrap(ctx context.Context, user *bootstrap.User, options BootstrapOptions) (*response.ShopsBootstrapResponse, error)
	GetEquipmentPmcsHistory(ctx context.Context, user *bootstrap.User) (*response.EquipmentPmcsHistoryResponse, error)
}
