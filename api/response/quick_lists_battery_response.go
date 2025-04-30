package response

import (
	"miltechserver/.gen/miltech_ng/public/model"
)

type QuickListsBatteryResponse struct {
	Batteries []model.QuickListBattery `json:"batteries"`
	Count     int                      `json:"count"`
}
