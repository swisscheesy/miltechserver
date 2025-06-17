package service

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/repository"
	"miltechserver/bootstrap"
)

type UserVehicleServiceImpl struct {
	UserVehicleRepository repository.UserVehicleRepository
}

func NewUserVehicleServiceImpl(userVehicleRepository repository.UserVehicleRepository) *UserVehicleServiceImpl {
	return &UserVehicleServiceImpl{UserVehicleRepository: userVehicleRepository}
}

// User Vehicle Operations

// GetUserVehiclesByUser returns all vehicles for a user
func (service *UserVehicleServiceImpl) GetUserVehiclesByUser(user *bootstrap.User) ([]model.UserVehicle, error) {
	vehicles, err := service.UserVehicleRepository.GetUserVehiclesByUserId(user)
	if vehicles == nil {
		return []model.UserVehicle{}, nil
	}
	return vehicles, err
}

// GetUserVehicleById returns a specific vehicle by ID for a user
func (service *UserVehicleServiceImpl) GetUserVehicleById(user *bootstrap.User, vehicleId string) (*model.UserVehicle, error) {
	return service.UserVehicleRepository.GetUserVehicleById(user, vehicleId)
}

// UpsertUserVehicle creates or updates a vehicle for a user
func (service *UserVehicleServiceImpl) UpsertUserVehicle(user *bootstrap.User, vehicle model.UserVehicle) error {
	return service.UserVehicleRepository.UpsertUserVehicle(user, vehicle)
}

// DeleteUserVehicle deletes a specific vehicle for a user
func (service *UserVehicleServiceImpl) DeleteUserVehicle(user *bootstrap.User, vehicleId string) error {
	return service.UserVehicleRepository.DeleteUserVehicle(user, vehicleId)
}

// DeleteAllUserVehicles deletes all vehicles for a user
func (service *UserVehicleServiceImpl) DeleteAllUserVehicles(user *bootstrap.User) error {
	return service.UserVehicleRepository.DeleteAllUserVehicles(user)
}

// User Vehicle Notifications Operations

// GetVehicleNotificationsByUser returns all vehicle notifications for a user
func (service *UserVehicleServiceImpl) GetVehicleNotificationsByUser(user *bootstrap.User) ([]model.UserVehicleNotifications, error) {
	notifications, err := service.UserVehicleRepository.GetVehicleNotificationsByUserId(user)
	if notifications == nil {
		return []model.UserVehicleNotifications{}, nil
	}
	return notifications, err
}

// GetVehicleNotificationsByVehicle returns all notifications for a specific vehicle
func (service *UserVehicleServiceImpl) GetVehicleNotificationsByVehicle(user *bootstrap.User, vehicleId string) ([]model.UserVehicleNotifications, error) {
	notifications, err := service.UserVehicleRepository.GetVehicleNotificationsByVehicleId(user, vehicleId)
	if notifications == nil {
		return []model.UserVehicleNotifications{}, nil
	}
	return notifications, err
}

// GetVehicleNotificationById returns a specific notification by ID for a user
func (service *UserVehicleServiceImpl) GetVehicleNotificationById(user *bootstrap.User, notificationId string) (*model.UserVehicleNotifications, error) {
	return service.UserVehicleRepository.GetVehicleNotificationById(user, notificationId)
}

// UpsertVehicleNotification creates or updates a vehicle notification for a user
func (service *UserVehicleServiceImpl) UpsertVehicleNotification(user *bootstrap.User, notification model.UserVehicleNotifications) error {
	return service.UserVehicleRepository.UpsertVehicleNotification(user, notification)
}

// DeleteVehicleNotification deletes a specific notification for a user
func (service *UserVehicleServiceImpl) DeleteVehicleNotification(user *bootstrap.User, notificationId string) error {
	return service.UserVehicleRepository.DeleteVehicleNotification(user, notificationId)
}

// DeleteAllVehicleNotificationsByVehicle deletes all notifications for a specific vehicle
func (service *UserVehicleServiceImpl) DeleteAllVehicleNotificationsByVehicle(user *bootstrap.User, vehicleId string) error {
	return service.UserVehicleRepository.DeleteAllVehicleNotificationsByVehicle(user, vehicleId)
}

// User Vehicle Comments Operations

// GetVehicleCommentsByUser returns all vehicle comments for a user
func (service *UserVehicleServiceImpl) GetVehicleCommentsByUser(user *bootstrap.User) ([]model.UserVehicleComments, error) {
	comments, err := service.UserVehicleRepository.GetVehicleCommentsByUserId(user)
	if comments == nil {
		return []model.UserVehicleComments{}, nil
	}
	return comments, err
}

// GetVehicleCommentsByVehicle returns all comments for a specific vehicle
func (service *UserVehicleServiceImpl) GetVehicleCommentsByVehicle(user *bootstrap.User, vehicleId string) ([]model.UserVehicleComments, error) {
	comments, err := service.UserVehicleRepository.GetVehicleCommentsByVehicleId(user, vehicleId)
	if comments == nil {
		return []model.UserVehicleComments{}, nil
	}
	return comments, err
}

// GetVehicleCommentsByNotification returns all comments for a specific notification
func (service *UserVehicleServiceImpl) GetVehicleCommentsByNotification(user *bootstrap.User, notificationId string) ([]model.UserVehicleComments, error) {
	comments, err := service.UserVehicleRepository.GetVehicleCommentsByNotificationId(user, notificationId)
	if comments == nil {
		return []model.UserVehicleComments{}, nil
	}
	return comments, err
}

// GetVehicleCommentById returns a specific comment by ID for a user
func (service *UserVehicleServiceImpl) GetVehicleCommentById(user *bootstrap.User, commentId string) (*model.UserVehicleComments, error) {
	return service.UserVehicleRepository.GetVehicleCommentById(user, commentId)
}

// UpsertVehicleComment creates or updates a vehicle comment for a user
func (service *UserVehicleServiceImpl) UpsertVehicleComment(user *bootstrap.User, comment model.UserVehicleComments) error {
	return service.UserVehicleRepository.UpsertVehicleComment(user, comment)
}

// DeleteVehicleComment deletes a specific comment for a user
func (service *UserVehicleServiceImpl) DeleteVehicleComment(user *bootstrap.User, commentId string) error {
	return service.UserVehicleRepository.DeleteVehicleComment(user, commentId)
}

// DeleteAllVehicleCommentsByVehicle deletes all comments for a specific vehicle
func (service *UserVehicleServiceImpl) DeleteAllVehicleCommentsByVehicle(user *bootstrap.User, vehicleId string) error {
	return service.UserVehicleRepository.DeleteAllVehicleCommentsByVehicle(user, vehicleId)
}

// User Notification Items Operations

// GetNotificationItemsByUser returns all notification items for a user
func (service *UserVehicleServiceImpl) GetNotificationItemsByUser(user *bootstrap.User) ([]model.UserNotificationItems, error) {
	items, err := service.UserVehicleRepository.GetNotificationItemsByUserId(user)
	if items == nil {
		return []model.UserNotificationItems{}, nil
	}
	return items, err
}

// GetNotificationItemsByNotification returns all items for a specific notification
func (service *UserVehicleServiceImpl) GetNotificationItemsByNotification(user *bootstrap.User, notificationId string) ([]model.UserNotificationItems, error) {
	items, err := service.UserVehicleRepository.GetNotificationItemsByNotificationId(user, notificationId)
	if items == nil {
		return []model.UserNotificationItems{}, nil
	}
	return items, err
}

// GetNotificationItemById returns a specific notification item by ID for a user
func (service *UserVehicleServiceImpl) GetNotificationItemById(user *bootstrap.User, itemId string) (*model.UserNotificationItems, error) {
	return service.UserVehicleRepository.GetNotificationItemById(user, itemId)
}

// UpsertNotificationItem creates or updates a notification item for a user
func (service *UserVehicleServiceImpl) UpsertNotificationItem(user *bootstrap.User, item model.UserNotificationItems) error {
	return service.UserVehicleRepository.UpsertNotificationItem(user, item)
}

// UpsertNotificationItemList creates or updates a list of notification items for a user
func (service *UserVehicleServiceImpl) UpsertNotificationItemList(user *bootstrap.User, items []model.UserNotificationItems) error {
	return service.UserVehicleRepository.UpsertNotificationItemList(user, items)
}

// DeleteNotificationItem deletes a specific notification item for a user
func (service *UserVehicleServiceImpl) DeleteNotificationItem(user *bootstrap.User, itemId string) error {
	return service.UserVehicleRepository.DeleteNotificationItem(user, itemId)
}

// DeleteAllNotificationItemsByNotification deletes all items for a specific notification
func (service *UserVehicleServiceImpl) DeleteAllNotificationItemsByNotification(user *bootstrap.User, notificationId string) error {
	return service.UserVehicleRepository.DeleteAllNotificationItemsByNotification(user, notificationId)
}
