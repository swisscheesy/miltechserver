package aggregates

import (
	"context"
	"errors"
	"fmt"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
)

const (
	maxVehiclesLimit                         = 200
	maxListsLimit                            = 200
	maxListItemsLimitPerList                 = 200
	maxNotificationsLimit                    = 200
	maxNotificationItemsLimitPerNotification = 100
	maxServicesLimit                         = 200
	maxChangesLimit                          = 200
	maxMessageLimit                          = 100
	maxEquipmentLimitPerShop                 = 250
)

type ServiceImpl struct {
	repo Repository
	auth shared.ShopAuthorization
}

func NewService(repo Repository, auth shared.ShopAuthorization) *ServiceImpl {
	return &ServiceImpl{repo: repo, auth: auth}
}

func normalizeOptionalLimit(value, maxValue int) int {
	if value > maxValue {
		return maxValue
	}
	return value
}

func normalizeListTreeLimits(limits ListTreeLimits) ListTreeLimits {
	return ListTreeLimits{
		ListsLimit:        normalizeOptionalLimit(limits.ListsLimit, maxListsLimit),
		ItemsLimitPerList: normalizeOptionalLimit(limits.ItemsLimitPerList, maxListItemsLimitPerList),
	}
}

func optionalLimitPtr(value int) *int {
	if value == 0 {
		return nil
	}
	return &value
}

func (s *ServiceImpl) requireShopMember(user *bootstrap.User, shopID string) error {
	if err := s.auth.RequireShopMember(user, shopID); err != nil {
		if errors.Is(err, shared.ErrShopAccessDenied) {
			return fmt.Errorf("%w: %w", ErrAccessDenied, err)
		}
		return fmt.Errorf("%w: %w", ErrAggregateUnavailable, err)
	}
	return nil
}

func (s *ServiceImpl) GetListsWithItems(ctx context.Context, user *bootstrap.User, shopID string, limits ListTreeLimits) (*response.ShopListsWithItemsResponse, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
	if err := s.requireShopMember(user, shopID); err != nil {
		return nil, err
	}
	limits = normalizeListTreeLimits(limits)
	lists, err := s.repo.GetListsWithItems(ctx, user, shopID, limits)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAggregateUnavailable, err)
	}
	if lists == nil {
		lists = []response.ShopListWithItems{}
	}
	for i := range lists {
		if lists[i].Items == nil {
			lists[i].Items = []response.ShopListItemWithUsername{}
		}
	}
	itemCount := int64(0)
	for _, list := range lists {
		itemCount += int64(len(list.Items))
	}
	return &response.ShopListsWithItemsResponse{
		Lists: lists,
		Counts: response.ShopListsWithItemsCounts{
			Lists: int64(len(lists)),
			Items: itemCount,
		},
		Limits: response.ShopListsWithItemsLimits{
			Lists:        optionalLimitPtr(limits.ListsLimit),
			ItemsPerList: optionalLimitPtr(limits.ItemsLimitPerList),
		},
	}, nil
}

func (s *ServiceImpl) GetVehicleMaintenanceSnapshot(ctx context.Context, user *bootstrap.User, vehicleID string, limits SnapshotLimits) (*response.VehicleMaintenanceSnapshotResponse, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
	limits.NotificationsLimit = normalizeOptionalLimit(limits.NotificationsLimit, maxNotificationsLimit)
	limits.NotificationItemsLimit = normalizeOptionalLimit(limits.NotificationItemsLimit, maxNotificationItemsLimitPerNotification)
	limits.ServicesLimit = normalizeOptionalLimit(limits.ServicesLimit, maxServicesLimit)
	limits.ChangesLimit = normalizeOptionalLimit(limits.ChangesLimit, maxChangesLimit)
	vehicle, err := s.repo.GetVehicleByIDForMember(ctx, user, vehicleID)
	if err != nil {
		if errors.Is(err, shared.ErrVehicleAccessDenied) {
			return nil, fmt.Errorf("%w: %w", ErrAccessDenied, err)
		}
		return nil, fmt.Errorf("%w: %w", ErrAggregateUnavailable, err)
	}
	if vehicle == nil {
		return nil, fmt.Errorf("%w: vehicle lookup returned nil", ErrAggregateUnavailable)
	}
	notifications, err := s.repo.GetVehicleNotificationsWithItems(ctx, vehicleID, limits)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAggregateUnavailable, err)
	}
	changes, err := s.repo.GetVehicleRecentChanges(ctx, vehicleID, limits.ChangesLimit)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAggregateUnavailable, err)
	}
	services, err := s.repo.GetVehicleServices(ctx, vehicleID, limits.ServicesLimit)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAggregateUnavailable, err)
	}
	if notifications == nil {
		notifications = []response.VehicleNotificationWithItems{}
	}
	if changes == nil {
		changes = []response.NotificationChangeWithUsername{}
	}
	if services == nil {
		services = []response.EquipmentServiceResponse{}
	}
	itemCount := int64(0)
	for _, notification := range notifications {
		itemCount += int64(len(notification.Items))
	}
	return &response.VehicleMaintenanceSnapshotResponse{
		Vehicle:       *vehicle,
		Notifications: notifications,
		RecentChanges: changes,
		Services:      services,
		Counts: response.VehicleMaintenanceSnapshotCounts{
			Notifications:     int64(len(notifications)),
			NotificationItems: itemCount,
			RecentChanges:     int64(len(changes)),
			Services:          int64(len(services)),
		},
		Limits: response.VehicleMaintenanceSnapshotLimits{
			Notifications:                    optionalLimitPtr(limits.NotificationsLimit),
			NotificationItemsPerNotification: optionalLimitPtr(limits.NotificationItemsLimit),
			Services:                         optionalLimitPtr(limits.ServicesLimit),
			RecentChanges:                    optionalLimitPtr(limits.ChangesLimit),
		},
	}, nil
}

func (s *ServiceImpl) GetShopSnapshot(ctx context.Context, user *bootstrap.User, shopID string, options ShopSnapshotOptions) (*response.ShopSnapshotResponse, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
	if err := s.requireShopMember(user, shopID); err != nil {
		return nil, err
	}
	options.VehiclesLimit = normalizeOptionalLimit(options.VehiclesLimit, maxVehiclesLimit)
	options.ListsLimit = normalizeOptionalLimit(options.ListsLimit, maxListsLimit)
	options.ItemsLimitPerList = normalizeOptionalLimit(options.ItemsLimitPerList, maxListItemsLimitPerList)
	options.NotificationsLimit = normalizeOptionalLimit(options.NotificationsLimit, maxNotificationsLimit)
	options.NotificationItemsLimit = normalizeOptionalLimit(options.NotificationItemsLimit, maxNotificationItemsLimitPerNotification)
	options.MessageLimit = normalizeOptionalLimit(options.MessageLimit, maxMessageLimit)
	options.ChangesLimit = normalizeOptionalLimit(options.ChangesLimit, maxChangesLimit)
	options.ServicesLimit = normalizeOptionalLimit(options.ServicesLimit, maxServicesLimit)
	result, err := s.repo.GetShopSnapshot(ctx, user, shopID, options)
	if err != nil {
		if errors.Is(err, shared.ErrShopAccessDenied) {
			return nil, fmt.Errorf("%w: %w", ErrAccessDenied, err)
		}
		return nil, fmt.Errorf("%w: %w", ErrAggregateUnavailable, err)
	}
	normalizeShopSnapshot(result)
	result.Limits = response.ShopSnapshotLimits{
		Vehicles:                         optionalLimitPtr(options.VehiclesLimit),
		Lists:                            optionalLimitPtr(options.ListsLimit),
		ItemsPerList:                     optionalLimitPtr(options.ItemsLimitPerList),
		Notifications:                    optionalLimitPtr(options.NotificationsLimit),
		NotificationItemsPerNotification: optionalLimitPtr(options.NotificationItemsLimit),
		Messages:                         optionalLimitPtr(options.MessageLimit),
		Services:                         optionalLimitPtr(options.ServicesLimit),
		RecentChanges:                    optionalLimitPtr(options.ChangesLimit),
	}
	return result, nil
}

func (s *ServiceImpl) GetBootstrap(ctx context.Context, user *bootstrap.User, options BootstrapOptions) (*response.ShopsBootstrapResponse, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
	options.EquipmentLimitPerShop = normalizeOptionalLimit(options.EquipmentLimitPerShop, maxEquipmentLimitPerShop)
	shops, err := s.repo.GetBootstrap(ctx, user, options)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAggregateUnavailable, err)
	}
	if shops == nil {
		shops = []response.ShopBootstrapSummary{}
	}
	for i := range shops {
		if shops[i].Equipment == nil {
			shops[i].Equipment = []response.ShopEquipmentSummary{}
		}
	}
	return &response.ShopsBootstrapResponse{Shops: shops}, nil
}

func normalizeShopSnapshot(result *response.ShopSnapshotResponse) {
	if result == nil {
		return
	}
	if result.Vehicles == nil {
		result.Vehicles = []model.ShopVehicle{}
	}
	if result.Lists == nil {
		result.Lists = []response.ShopListWithItems{}
	}
	if result.Notifications == nil {
		result.Notifications = []response.VehicleNotificationWithItems{}
	}
	if result.Messages == nil {
		result.Messages = []model.ShopMessages{}
	}
	if result.Services == nil {
		result.Services = []response.EquipmentServiceResponse{}
	}
	if result.RecentChanges == nil {
		result.RecentChanges = []response.NotificationChangeWithUsername{}
	}
}
