package repository

import (
	"miltechserver/api/response"
)

type ItemQuickListsRepository interface {
	GetQuickListClothing() (response.QuickListsClothingResponse, error)
	GetQuickListWheels() (response.QuickListsWheelsResponse, error)
	GetQuickListBatteries() (response.QuickListsBatteryResponse, error)
}
