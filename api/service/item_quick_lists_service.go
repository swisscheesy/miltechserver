package service

import (
	"miltechserver/api/response"
)

type ItemQuickListsService interface {
	GetQuickListClothing() (response.QuickListsClothingResponse, error)
	GetQuickListWheels() (response.QuickListsWheelsResponse, error)
	GetQuickListBatteries() (response.QuickListsBatteryResponse, error)
}
