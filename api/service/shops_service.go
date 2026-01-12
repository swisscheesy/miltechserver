package service

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type ShopsService interface {
	// Shop Operations
	CreateShop(user *bootstrap.User, shop model.Shops) (*model.Shops, error)
	UpdateShop(user *bootstrap.User, shop model.Shops) (*model.Shops, error)
	DeleteShop(user *bootstrap.User, shopID string) error
	GetShopsByUser(user *bootstrap.User) ([]model.Shops, error)
	GetShopByID(user *bootstrap.User, shopID string) (*response.ShopDetailResponse, error)
	GetUserDataWithShops(user *bootstrap.User) (*response.UserShopsResponse, error)

	// Shop Member Operations
	JoinShopViaInviteCode(user *bootstrap.User, inviteCode string) error
	LeaveShop(user *bootstrap.User, shopID string) error
	RemoveMemberFromShop(user *bootstrap.User, shopID string, targetUserID string) error
	GetShopMembers(user *bootstrap.User, shopID string) ([]response.ShopMemberWithUsername, error)
	PromoteMemberToAdmin(user *bootstrap.User, shopID string, targetUserID string) error

	// Shop Invite Code Operations
	GenerateInviteCode(user *bootstrap.User, shopID string) (*model.ShopInviteCodes, error)
	GetInviteCodesByShop(user *bootstrap.User, shopID string) ([]model.ShopInviteCodes, error)
	DeactivateInviteCode(user *bootstrap.User, codeID string) error
	DeleteInviteCode(user *bootstrap.User, codeID string) error

	// Shop Message Operations
	CreateShopMessage(user *bootstrap.User, message model.ShopMessages) (*model.ShopMessages, error)
	GetShopMessages(user *bootstrap.User, shopID string) ([]model.ShopMessages, error)
	GetShopMessagesPaginated(user *bootstrap.User, shopID string, page int, limit int) (*response.PaginatedShopMessagesResponse, error)
	UpdateShopMessage(user *bootstrap.User, message model.ShopMessages) error
	DeleteShopMessage(user *bootstrap.User, messageID string) error
	UploadMessageImage(user *bootstrap.User, shopID string, imageData []byte, contentType string) (string, string, string, error)
	DeleteMessageImage(user *bootstrap.User, shopID string, messageID string) error

	// Shop Vehicle Operations
	CreateShopVehicle(user *bootstrap.User, vehicle model.ShopVehicle) (*model.ShopVehicle, error)
	GetShopVehicles(user *bootstrap.User, shopID string) ([]model.ShopVehicle, error)
	GetShopVehicleByID(user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error)
	UpdateShopVehicle(user *bootstrap.User, vehicle model.ShopVehicle) error
	DeleteShopVehicle(user *bootstrap.User, vehicleID string) error

	// Shop Vehicle Notification Operations
	CreateVehicleNotification(user *bootstrap.User, notification model.ShopVehicleNotifications) (*model.ShopVehicleNotifications, error)
	GetVehicleNotifications(user *bootstrap.User, vehicleID string) ([]model.ShopVehicleNotifications, error)
	GetVehicleNotificationsWithItems(user *bootstrap.User, vehicleID string) ([]response.VehicleNotificationWithItems, error)
	GetShopNotifications(user *bootstrap.User, shopID string) ([]model.ShopVehicleNotifications, error)
	GetVehicleNotificationByID(user *bootstrap.User, notificationID string) (*model.ShopVehicleNotifications, error)
	UpdateVehicleNotification(user *bootstrap.User, notification model.ShopVehicleNotifications) error
	DeleteVehicleNotification(user *bootstrap.User, notificationID string) error

	// Shop Notification Item Operations
	AddNotificationItem(user *bootstrap.User, item model.ShopNotificationItems) (*model.ShopNotificationItems, error)
	GetNotificationItems(user *bootstrap.User, notificationID string) ([]model.ShopNotificationItems, error)
	GetShopNotificationItems(user *bootstrap.User, shopID string) ([]model.ShopNotificationItems, error)
	AddNotificationItemList(user *bootstrap.User, items []model.ShopNotificationItems) ([]model.ShopNotificationItems, error)
	RemoveNotificationItem(user *bootstrap.User, itemID string) error
	RemoveNotificationItemList(user *bootstrap.User, itemIDs []string) error

	// Shop List Operations
	CreateShopList(user *bootstrap.User, list model.ShopLists) (*response.ShopListWithUsername, error)
	GetShopLists(user *bootstrap.User, shopID string) ([]response.ShopListWithUsername, error)
	GetShopListByID(user *bootstrap.User, listID string) (*response.ShopListWithUsername, error)
	UpdateShopList(user *bootstrap.User, list model.ShopLists) error
	DeleteShopList(user *bootstrap.User, listID string) error

	// Shop List Item Operations
	AddListItem(user *bootstrap.User, item model.ShopListItems) (*response.ShopListItemWithUsername, error)
	GetListItems(user *bootstrap.User, listID string) ([]response.ShopListItemWithUsername, error)
	UpdateListItem(user *bootstrap.User, item model.ShopListItems) error
	RemoveListItem(user *bootstrap.User, itemID string) error
	AddListItemBatch(user *bootstrap.User, items []model.ShopListItems) ([]response.ShopListItemWithUsername, error)
	RemoveListItemBatch(user *bootstrap.User, itemIDs []string) error

	// Shop Settings Operations
	GetShopAdminOnlyListsSetting(user *bootstrap.User, shopID string) (bool, error)
	UpdateShopAdminOnlyListsSetting(user *bootstrap.User, shopID string, adminOnlyLists bool) error
	IsUserShopAdmin(user *bootstrap.User, shopID string) (bool, error)

	// Unified Shop Settings Operations
	GetShopSettings(user *bootstrap.User, shopID string) (*request.ShopSettings, error)
	UpdateShopSettings(user *bootstrap.User, shopID string, updates request.UpdateShopSettingsRequest) (*request.ShopSettings, error)

	// Notification Change Tracking (Audit Trail) Operations
	GetNotificationChangeHistory(user *bootstrap.User, notificationID string) ([]response.NotificationChangeWithUsername, error)
	GetShopNotificationChanges(user *bootstrap.User, shopID string, limit int) ([]response.NotificationChangeWithUsername, error)
	GetVehicleNotificationChanges(user *bootstrap.User, vehicleID string) ([]response.NotificationChangeWithUsername, error)
}
