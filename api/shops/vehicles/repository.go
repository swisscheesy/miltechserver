package vehicles

import (
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
	CreateShopVehicle(user *bootstrap.User, vehicle model.ShopVehicle) (*model.ShopVehicle, error)
	GetShopVehicles(user *bootstrap.User, shopID string) ([]model.ShopVehicle, error)
	GetShopVehicleByID(user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error)
	UpdateShopVehicle(user *bootstrap.User, vehicle model.ShopVehicle) error
	UpdateShopVehicleUsage(user *bootstrap.User, update ShopVehicleUsageUpdate) error
	DeleteShopVehicle(user *bootstrap.User, vehicleID string) error
	CreateNotificationChange(user *bootstrap.User, change model.ShopVehicleNotificationChanges) error
}
