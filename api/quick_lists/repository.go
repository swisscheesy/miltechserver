package quick_lists

type Repository interface {
	GetQuickListClothing() (QuickListsClothingResponse, error)
	GetQuickListWheels() (QuickListsWheelsResponse, error)
	GetQuickListBatteries() (QuickListsBatteryResponse, error)
}
