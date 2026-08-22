package vehicles

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Service interface {
	CreateShopVehicle(user *bootstrap.User, vehicle model.ShopVehicle) (*model.ShopVehicle, error)
	GetShopVehicles(user *bootstrap.User, shopID string) ([]model.ShopVehicle, error)
	GetShopVehicleByID(user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error)
	UpdateShopVehicle(user *bootstrap.User, vehicle model.ShopVehicle) error
	AdjustShopVehicleUsage(ctx context.Context, user *bootstrap.User, adjustment UsageAdjustment) (*model.ShopVehicle, error)
	DeleteShopVehicle(user *bootstrap.User, vehicleID string) error
}
