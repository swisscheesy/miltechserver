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
	defaultServicesLimit         = 50
	defaultChangesLimit          = 50
	maxServicesLimit             = 200
	maxChangesLimit              = 200
	defaultMessageLimit          = 20
	maxMessageLimit              = 100
	defaultEquipmentLimitPerShop = 50
	maxEquipmentLimitPerShop     = 250
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

func (s *ServiceImpl) requireShopMember(user *bootstrap.User, shopID string) error {
	if err := s.auth.RequireShopMember(user, shopID); err != nil {
		if errors.Is(err, shared.ErrShopAccessDenied) {
			return fmt.Errorf("%w: %w", ErrAccessDenied, err)
		}
		return fmt.Errorf("%w: %w", ErrAggregateUnavailable, err)
	}
	return nil
}

func (s *ServiceImpl) GetListsWithItems(ctx context.Context, user *bootstrap.User, shopID string) (*response.ShopListsWithItemsResponse, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
	if err := s.requireShopMember(user, shopID); err != nil {
		return nil, err
	}
	lists, err := s.repo.GetListsWithItems(ctx, user, shopID)
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
	return &response.ShopListsWithItemsResponse{Lists: lists}, nil
}

func (s *ServiceImpl) GetVehicleMaintenanceSnapshot(ctx context.Context, user *bootstrap.User, vehicleID string, limits SnapshotLimits) (*response.VehicleMaintenanceSnapshotResponse, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
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
	notifications, err := s.repo.GetVehicleNotificationsWithItems(ctx, vehicleID)
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
		Counts: response.ShopAggregateCounts{
			Notifications:     int64(len(notifications)),
			NotificationItems: itemCount,
			RecentChanges:     int64(len(changes)),
			Services:          int64(len(services)),
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
	options.MessageLimit = normalizeLimit(options.MessageLimit, defaultMessageLimit, maxMessageLimit)
	options.ChangesLimit = normalizeLimit(options.ChangesLimit, defaultChangesLimit, maxChangesLimit)
	options.ServicesLimit = normalizeLimit(options.ServicesLimit, defaultServicesLimit, maxServicesLimit)
	result, err := s.repo.GetShopSnapshot(ctx, user, shopID, options)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAggregateUnavailable, err)
	}
	normalizeShopSnapshot(result)
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
