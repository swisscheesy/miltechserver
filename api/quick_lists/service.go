package quick_lists

import "context"

type Service interface {
	GetQuickListClothing(ctx context.Context) (QuickListsClothingResponse, error)
	GetQuickListWheels(ctx context.Context) (QuickListsWheelsResponse, error)
	GetQuickListBatteries(ctx context.Context) (QuickListsBatteryResponse, error)
}
