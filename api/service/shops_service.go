package service

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type ShopsService interface {
	// Shop Operations
	CreateShop(user *bootstrap.User, shop model.Shops) (*model.Shops, error)
	DeleteShop(user *bootstrap.User, shopID string) error
	GetShopsByUser(user *bootstrap.User) ([]model.Shops, error)
	GetShopByID(user *bootstrap.User, shopID string) (*model.Shops, error)

	// Shop Member Operations
	JoinShopViaInviteCode(user *bootstrap.User, inviteCode string) error
	LeaveShop(user *bootstrap.User, shopID string) error
	RemoveMemberFromShop(user *bootstrap.User, shopID string, targetUserID string) error
	GetShopMembers(user *bootstrap.User, shopID string) ([]model.ShopMembers, error)

	// Shop Invite Code Operations
	GenerateInviteCode(user *bootstrap.User, shopID string) (*model.ShopInviteCodes, error)
	GetInviteCodesByShop(user *bootstrap.User, shopID string) ([]model.ShopInviteCodes, error)
	DeactivateInviteCode(user *bootstrap.User, codeID string) error

	// Shop Message Operations
	CreateShopMessage(user *bootstrap.User, message model.ShopMessages) (*model.ShopMessages, error)
	GetShopMessages(user *bootstrap.User, shopID string) ([]model.ShopMessages, error)
	UpdateShopMessage(user *bootstrap.User, message model.ShopMessages) error
	DeleteShopMessage(user *bootstrap.User, messageID string) error

	// Shop Vehicle Operations
	CreateShopVehicle(user *bootstrap.User, vehicle model.ShopVehicle) (*model.ShopVehicle, error)
	GetShopVehicles(user *bootstrap.User, shopID string) ([]model.ShopVehicle, error)
	GetShopVehicleByID(user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error)
	UpdateShopVehicle(user *bootstrap.User, vehicle model.ShopVehicle) error
	DeleteShopVehicle(user *bootstrap.User, vehicleID string) error

	// Shop Vehicle Notification Operations
	CreateVehicleNotification(user *bootstrap.User, notification model.ShopVehicleNotifications) (*model.ShopVehicleNotifications, error)
	GetVehicleNotifications(user *bootstrap.User, vehicleID string) ([]model.ShopVehicleNotifications, error)
	GetVehicleNotificationByID(user *bootstrap.User, notificationID string) (*model.ShopVehicleNotifications, error)
	UpdateVehicleNotification(user *bootstrap.User, notification model.ShopVehicleNotifications) error
	DeleteVehicleNotification(user *bootstrap.User, notificationID string) error

	// Shop Notification Item Operations
	AddNotificationItem(user *bootstrap.User, item model.ShopNotificationItems) (*model.ShopNotificationItems, error)
	GetNotificationItems(user *bootstrap.User, notificationID string) ([]model.ShopNotificationItems, error)
	AddNotificationItemList(user *bootstrap.User, items []model.ShopNotificationItems) error
	RemoveNotificationItem(user *bootstrap.User, itemID string) error
	RemoveNotificationItemList(user *bootstrap.User, itemIDs []string) error
}
