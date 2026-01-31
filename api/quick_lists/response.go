package quick_lists

import "miltechserver/.gen/miltech_ng/public/model"

type QuickListsClothingResponse struct {
	Clothing []model.QuickListClothing `json:"clothing"`
	Count    int                       `json:"count"`
}

type QuickListsWheelsResponse struct {
	Wheels []model.QuickListWheelTires `json:"wheels"`
	Count  int                         `json:"count"`
}

type QuickListsBatteryResponse struct {
	Batteries []model.QuickListBattery `json:"batteries"`
	Count     int                      `json:"count"`
}
