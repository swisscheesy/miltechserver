package vehicles

import (
	"context"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type ShopVehicleUsageUpdate struct {
	VehicleID      string
	TrackedMileage *int32
	TrackedHours   *int32
	LastUpdated    time.Time
}

type Repository interface {
	CreateShopVehicle(ctx context.Context, user *bootstrap.User, vehicle model.ShopVehicle) (*model.ShopVehicle, error)
	GetShopVehicles(ctx context.Context, user *bootstrap.User, shopID string) ([]model.ShopVehicle, error)
	GetShopVehicleByID(ctx context.Context, user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error)
	UpdateShopVehicleMetadata(ctx context.Context, user *bootstrap.User, input VehicleUpdateInput) error
	UpdateShopVehicleUsage(ctx context.Context, user *bootstrap.User, update ShopVehicleUsageUpdate) error
	AdjustShopVehicleUsage(ctx context.Context, user *bootstrap.User, adjustment UsageAdjustment) (*model.ShopVehicle, error)
	DeleteShopVehicle(ctx context.Context, user *bootstrap.User, vehicleID string) error
	CreateNotificationChange(ctx context.Context, user *bootstrap.User, change model.ShopVehicleNotificationChanges) error
}
