package repository

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type ShopsRepository interface {
	// Shop Operations
	CreateShop(user *bootstrap.User, shop model.Shops) (*model.Shops, error)
	UpdateShop(user *bootstrap.User, shop model.Shops) (*model.Shops, error)
	DeleteShop(user *bootstrap.User, shopID string) error
	GetShopsByUser(user *bootstrap.User) ([]model.Shops, error)
	GetShopByID(user *bootstrap.User, shopID string) (*model.Shops, error)
	GetShopsWithStatsForUser(user *bootstrap.User) ([]response.ShopWithStats, error)
	IsUserShopAdmin(user *bootstrap.User, shopID string) (bool, error)

	// Shop Member Operations
	AddMemberToShop(user *bootstrap.User, shopID string, role string) error
	RemoveMemberFromShop(user *bootstrap.User, shopID string, targetUserID string) error
	GetShopMembers(user *bootstrap.User, shopID string) ([]response.ShopMemberWithUsername, error)
	GetShopMemberCount(user *bootstrap.User, shopID string) (int64, error)
	IsUserMemberOfShop(user *bootstrap.User, shopID string) (bool, error)
	UpdateMemberRole(user *bootstrap.User, shopID string, targetUserID string, newRole string) error

	// Shop Vehicle Operations
	GetShopVehicleCount(user *bootstrap.User, shopID string) (int64, error)

	// Shop Invite Code Operations
	CreateInviteCode(user *bootstrap.User, inviteCode model.ShopInviteCodes) (*model.ShopInviteCodes, error)
	GetInviteCodeByCode(code string) (*model.ShopInviteCodes, error)
	GetInviteCodeByID(codeID string) (*model.ShopInviteCodes, error)
	GetInviteCodesByShop(user *bootstrap.User, shopID string) ([]model.ShopInviteCodes, error)
	DeactivateInviteCode(user *bootstrap.User, codeID string) error
	DeleteInviteCode(user *bootstrap.User, codeID string) error

	// Shop Message Operations
	CreateShopMessage(user *bootstrap.User, message model.ShopMessages) (*model.ShopMessages, error)
	GetShopMessages(user *bootstrap.User, shopID string) ([]model.ShopMessages, error)
	GetShopMessagesPaginated(user *bootstrap.User, shopID string, offset int, limit int) ([]model.ShopMessages, error)
	GetShopMessagesCount(user *bootstrap.User, shopID string) (int64, error)
	GetShopMessageByID(user *bootstrap.User, messageID string) (*model.ShopMessages, error)
	UpdateShopMessage(user *bootstrap.User, message model.ShopMessages) error
	DeleteShopMessage(user *bootstrap.User, messageID string) error
	UploadMessageImage(user *bootstrap.User, messageID string, shopID string, imageData []byte, contentType string) (string, string, error)
	DeleteMessageImageBlob(user *bootstrap.User, messageID string, shopID string) error
	DeleteBlobByURL(imageURL string) error
	DeleteShopMessageBlobs(shopID string) error

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
	CreateNotificationItem(user *bootstrap.User, item model.ShopNotificationItems) (*model.ShopNotificationItems, error)
	GetNotificationItems(user *bootstrap.User, notificationID string) ([]model.ShopNotificationItems, error)
	GetShopNotificationItems(user *bootstrap.User, shopID string) ([]model.ShopNotificationItems, error)
	CreateNotificationItemList(user *bootstrap.User, items []model.ShopNotificationItems) ([]model.ShopNotificationItems, error)
	DeleteNotificationItem(user *bootstrap.User, itemID string) error
	DeleteNotificationItemList(user *bootstrap.User, itemIDs []string) error

	// Shop List Operations
	CreateShopList(user *bootstrap.User, list model.ShopLists) (*response.ShopListWithUsername, error)
	GetShopLists(user *bootstrap.User, shopID string) ([]response.ShopListWithUsername, error)
	GetShopListByID(user *bootstrap.User, listID string) (*response.ShopListWithUsername, error)
	UpdateShopList(user *bootstrap.User, list model.ShopLists) error
	DeleteShopList(user *bootstrap.User, listID string) error

	// Shop List Item Operations
	AddListItem(user *bootstrap.User, item model.ShopListItems) (*response.ShopListItemWithUsername, error)
	GetListItems(user *bootstrap.User, listID string) ([]response.ShopListItemWithUsername, error)
	GetListItemByID(user *bootstrap.User, itemID string) (*model.ShopListItems, error)
	UpdateListItem(user *bootstrap.User, item model.ShopListItems) error
	RemoveListItem(user *bootstrap.User, itemID string) error
	AddListItemBatch(user *bootstrap.User, items []model.ShopListItems) ([]response.ShopListItemWithUsername, error)
	RemoveListItemBatch(user *bootstrap.User, itemIDs []string) error

	// Helper method for permissions
	GetUserRoleInShop(user *bootstrap.User, shopID string) (string, error)

	// Notification Change Tracking (Audit Trail)
	CreateNotificationChange(user *bootstrap.User, change model.ShopVehicleNotificationChanges) error
	GetNotificationChanges(user *bootstrap.User, notificationID string) ([]response.NotificationChangeWithUsername, error)
	GetNotificationChangesByShop(user *bootstrap.User, shopID string, limit int) ([]response.NotificationChangeWithUsername, error)
	GetNotificationChangesByVehicle(user *bootstrap.User, vehicleID string) ([]response.NotificationChangeWithUsername, error)
}
