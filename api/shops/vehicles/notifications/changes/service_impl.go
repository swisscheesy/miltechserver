package changes

import (
	"errors"
	"fmt"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type ServiceImpl struct {
	repo Repository
}

func NewService(repo Repository) *ServiceImpl {
	return &ServiceImpl{repo: repo}
}

func (service *ServiceImpl) GetNotificationChangeHistory(
	user *bootstrap.User,
	notificationID string,
) ([]response.NotificationChangeWithUsername, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	notification, err := service.repo.GetVehicleNotificationByID(user, notificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	isMember, err := service.repo.IsUserMemberOfShop(user, notification.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	changes, err := service.repo.GetNotificationChanges(user, notificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification changes: %w", err)
	}

	return changes, nil
}

func (service *ServiceImpl) GetShopNotificationChanges(
	user *bootstrap.User,
	shopID string,
	limit int,
) ([]response.NotificationChangeWithUsername, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	isMember, err := service.repo.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	changes, err := service.repo.GetNotificationChangesByShop(user, shopID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop notification changes: %w", err)
	}

	return changes, nil
}

func (service *ServiceImpl) GetVehicleNotificationChanges(
	user *bootstrap.User,
	vehicleID string,
) ([]response.NotificationChangeWithUsername, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	vehicle, err := service.repo.GetShopVehicleByID(user, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle: %w", err)
	}

	isMember, err := service.repo.IsUserMemberOfShop(user, vehicle.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	changes, err := service.repo.GetNotificationChangesByVehicle(user, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle notification changes: %w", err)
	}

	return changes, nil
}
