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
	defaultVehiclesLimit                         = 50
	defaultListsLimit                            = 50
	defaultListItemsLimitPerList                 = 50
	defaultNotificationsLimit                    = 50
	defaultNotificationItemsLimitPerNotification = 25
	defaultServicesLimit                         = 50
	defaultChangesLimit                          = 50
	defaultMessageLimit                          = 20
	defaultEquipmentLimitPerShop                 = 50
	maxVehiclesLimit                             = 200
	maxListsLimit                                = 200
	maxListItemsLimitPerList                     = 200
	maxNotificationsLimit                        = 200
	maxNotificationItemsLimitPerNotification     = 100
	maxServicesLimit                             = 200
	maxChangesLimit                              = 200
	maxMessageLimit                              = 100
	maxEquipmentLimitPerShop                     = 250
)

type ServiceImpl struct {
	repo Repository
	auth shared.ShopAuthorization
}

func NewService(repo Repository, auth shared.ShopAuthorization) *ServiceImpl {
	return &ServiceImpl{repo: repo, auth: auth}
}

func normalizeLimit(value, defaultValue, maxValue int) int {
	if value <= 0 {
		return defaultValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func normalizeListTreeLimits(limits ListTreeLimits) ListTreeLimits {
	return ListTreeLimits{
		ListsLimit:        normalizeLimit(limits.ListsLimit, defaultListsLimit, maxListsLimit),
		ItemsLimitPerList: normalizeLimit(limits.ItemsLimitPerList, defaultListItemsLimitPerList, maxListItemsLimitPerList),
	}
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
			Lists:        limits.ListsLimit,
			ItemsPerList: limits.ItemsLimitPerList,
		},
	}, nil
}

func (s *ServiceImpl) GetVehicleMaintenanceSnapshot(ctx context.Context, user *bootstrap.User, vehicleID string, limits SnapshotLimits) (*response.VehicleMaintenanceSnapshotResponse, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
	limits.NotificationsLimit = normalizeLimit(limits.NotificationsLimit, defaultNotificationsLimit, maxNotificationsLimit)
	limits.NotificationItemsLimit = normalizeLimit(limits.NotificationItemsLimit, defaultNotificationItemsLimitPerNotification, maxNotificationItemsLimitPerNotification)
	limits.ServicesLimit = normalizeLimit(limits.ServicesLimit, defaultServicesLimit, maxServicesLimit)
	limits.ChangesLimit = normalizeLimit(limits.ChangesLimit, defaultChangesLimit, maxChangesLimit)
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
			Notifications:                    limits.NotificationsLimit,
			NotificationItemsPerNotification: limits.NotificationItemsLimit,
			Services:                         limits.ServicesLimit,
			RecentChanges:                    limits.ChangesLimit,
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
	options.VehiclesLimit = normalizeLimit(options.VehiclesLimit, defaultVehiclesLimit, maxVehiclesLimit)
	options.ListsLimit = normalizeLimit(options.ListsLimit, defaultListsLimit, maxListsLimit)
	options.ItemsLimitPerList = normalizeLimit(options.ItemsLimitPerList, defaultListItemsLimitPerList, maxListItemsLimitPerList)
	options.NotificationsLimit = normalizeLimit(options.NotificationsLimit, defaultNotificationsLimit, maxNotificationsLimit)
	options.NotificationItemsLimit = normalizeLimit(options.NotificationItemsLimit, defaultNotificationItemsLimitPerNotification, maxNotificationItemsLimitPerNotification)
	options.MessageLimit = normalizeLimit(options.MessageLimit, defaultMessageLimit, maxMessageLimit)
	options.ChangesLimit = normalizeLimit(options.ChangesLimit, defaultChangesLimit, maxChangesLimit)
	options.ServicesLimit = normalizeLimit(options.ServicesLimit, defaultServicesLimit, maxServicesLimit)
	result, err := s.repo.GetShopSnapshot(ctx, user, shopID, options)
	if err != nil {
		if errors.Is(err, shared.ErrShopAccessDenied) {
			return nil, fmt.Errorf("%w: %w", ErrAccessDenied, err)
		}
		return nil, fmt.Errorf("%w: %w", ErrAggregateUnavailable, err)
	}
	normalizeShopSnapshot(result)
	result.Limits = response.ShopSnapshotLimits{
		Vehicles:                         options.VehiclesLimit,
		Lists:                            options.ListsLimit,
		ItemsPerList:                     options.ItemsLimitPerList,
		Notifications:                    options.NotificationsLimit,
		NotificationItemsPerNotification: options.NotificationItemsLimit,
		Messages:                         options.MessageLimit,
		Services:                         options.ServicesLimit,
		RecentChanges:                    options.ChangesLimit,
	}
	return result, nil
}

func (s *ServiceImpl) GetBootstrap(ctx context.Context, user *bootstrap.User, options BootstrapOptions) (*response.ShopsBootstrapResponse, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
	options.EquipmentLimitPerShop = normalizeLimit(options.EquipmentLimitPerShop, defaultEquipmentLimitPerShop, maxEquipmentLimitPerShop)
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
