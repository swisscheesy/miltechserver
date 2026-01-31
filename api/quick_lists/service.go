package quick_lists

type Service interface {
	GetQuickListClothing() (QuickListsClothingResponse, error)
	GetQuickListWheels() (QuickListsWheelsResponse, error)
	GetQuickListBatteries() (QuickListsBatteryResponse, error)
}
