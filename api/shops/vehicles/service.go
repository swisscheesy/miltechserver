package vehicles

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Service interface {
	CreateShopVehicle(ctx context.Context, user *bootstrap.User, vehicle model.ShopVehicle) (*model.ShopVehicle, error)
	GetShopVehicles(ctx context.Context, user *bootstrap.User, shopID string) ([]model.ShopVehicle, error)
	GetShopVehicleByID(ctx context.Context, user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error)
	UpdateShopVehicle(ctx context.Context, user *bootstrap.User, vehicle model.ShopVehicle) error
	AdjustShopVehicleUsage(ctx context.Context, user *bootstrap.User, adjustment UsageAdjustment) (*model.ShopVehicle, error)
	DeleteShopVehicle(ctx context.Context, user *bootstrap.User, vehicleID string) error
}
