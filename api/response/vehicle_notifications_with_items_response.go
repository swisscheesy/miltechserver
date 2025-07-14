package response

import (
	"miltechserver/.gen/miltech_ng/public/model"
)

type VehicleNotificationWithItems struct {
	Notification model.ShopVehicleNotifications `json:"notification"`
	Items        []model.ShopNotificationItems  `json:"items"`
}