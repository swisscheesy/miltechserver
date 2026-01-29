package facade

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type ServiceImpl struct {
	CoreService                CoreService
	SettingsService            SettingsService
	MembersService             MembersService
	InviteService              InviteService
	ListsService               ListsService
	ListItemsService           ListItemsService
	MessagesService            MessagesService
	VehiclesService            VehiclesService
	NotificationsService       NotificationsService
	NotificationItemsService   NotificationItemsService
	NotificationChangesService NotificationChangesService
}

func NewService(
	coreService CoreService,
	settingsService SettingsService,
	membersService MembersService,
	inviteService InviteService,
	listsService ListsService,
	listItemsService ListItemsService,
	messagesService MessagesService,
	vehiclesService VehiclesService,
	notificationsService NotificationsService,
	notificationItemsService NotificationItemsService,
	notificationChangesService NotificationChangesService,
) *ServiceImpl {
	return &ServiceImpl{
		CoreService:                coreService,
		SettingsService:            settingsService,
		MembersService:             membersService,
		InviteService:              inviteService,
		ListsService:               listsService,
		ListItemsService:           listItemsService,
		MessagesService:            messagesService,
		VehiclesService:            vehiclesService,
		NotificationsService:       notificationsService,
		NotificationItemsService:   notificationItemsService,
		NotificationChangesService: notificationChangesService,
	}
}

// Shop Operations
func (service *ServiceImpl) CreateShop(user *bootstrap.User, shop model.Shops) (*model.Shops, error) {
	return service.CoreService.CreateShop(user, shop)
}

func (service *ServiceImpl) UpdateShop(user *bootstrap.User, shop model.Shops) (*model.Shops, error) {
	return service.CoreService.UpdateShop(user, shop)
}

func (service *ServiceImpl) DeleteShop(user *bootstrap.User, shopID string) error {
	return service.CoreService.DeleteShop(user, shopID)
}

func (service *ServiceImpl) GetShopsByUser(user *bootstrap.User) ([]model.Shops, error) {
	return service.CoreService.GetShopsByUser(user)
}

func (service *ServiceImpl) GetShopByID(user *bootstrap.User, shopID string) (*response.ShopDetailResponse, error) {
	return service.CoreService.GetShopByID(user, shopID)
}

func (service *ServiceImpl) GetUserDataWithShops(user *bootstrap.User) (*response.UserShopsResponse, error) {
	return service.CoreService.GetUserDataWithShops(user)
}

// Shop Member Operations
func (service *ServiceImpl) JoinShopViaInviteCode(user *bootstrap.User, inviteCode string) error {
	return service.MembersService.JoinShopViaInviteCode(user, inviteCode)
}

func (service *ServiceImpl) LeaveShop(user *bootstrap.User, shopID string) error {
	return service.MembersService.LeaveShop(user, shopID)
}

func (service *ServiceImpl) RemoveMemberFromShop(user *bootstrap.User, shopID string, targetUserID string) error {
	return service.MembersService.RemoveMemberFromShop(user, shopID, targetUserID)
}

func (service *ServiceImpl) GetShopMembers(user *bootstrap.User, shopID string) ([]response.ShopMemberWithUsername, error) {
	return service.MembersService.GetShopMembers(user, shopID)
}

func (service *ServiceImpl) PromoteMemberToAdmin(user *bootstrap.User, shopID string, targetUserID string) error {
	return service.MembersService.PromoteMemberToAdmin(user, shopID, targetUserID)
}

// Shop Invite Code Operations
func (service *ServiceImpl) GenerateInviteCode(user *bootstrap.User, shopID string) (*model.ShopInviteCodes, error) {
	return service.InviteService.GenerateInviteCode(user, shopID)
}

func (service *ServiceImpl) GetInviteCodesByShop(user *bootstrap.User, shopID string) ([]model.ShopInviteCodes, error) {
	return service.InviteService.GetInviteCodesByShop(user, shopID)
}

func (service *ServiceImpl) DeactivateInviteCode(user *bootstrap.User, codeID string) error {
	return service.InviteService.DeactivateInviteCode(user, codeID)
}

func (service *ServiceImpl) DeleteInviteCode(user *bootstrap.User, codeID string) error {
	return service.InviteService.DeleteInviteCode(user, codeID)
}

// Shop Message Operations
func (service *ServiceImpl) CreateShopMessage(user *bootstrap.User, message model.ShopMessages) (*model.ShopMessages, error) {
	return service.MessagesService.CreateShopMessage(user, message)
}

func (service *ServiceImpl) GetShopMessages(user *bootstrap.User, shopID string) ([]model.ShopMessages, error) {
	return service.MessagesService.GetShopMessages(user, shopID)
}

func (service *ServiceImpl) GetShopMessagesPaginated(user *bootstrap.User, shopID string, page int, limit int) (*response.PaginatedShopMessagesResponse, error) {
	return service.MessagesService.GetShopMessagesPaginated(user, shopID, page, limit)
}

func (service *ServiceImpl) UpdateShopMessage(user *bootstrap.User, message model.ShopMessages) error {
	return service.MessagesService.UpdateShopMessage(user, message)
}

func (service *ServiceImpl) DeleteShopMessage(user *bootstrap.User, messageID string) error {
	return service.MessagesService.DeleteShopMessage(user, messageID)
}

// UploadMessageImage uploads an image for a shop message to Azure Blob Storage
// Returns: messageID, fileExtension, imageURL, error
func (service *ServiceImpl) UploadMessageImage(user *bootstrap.User, shopID string, imageData []byte, contentType string) (string, string, string, error) {
	return service.MessagesService.UploadMessageImage(user, shopID, imageData, contentType)
}

// DeleteMessageImage deletes an orphaned message image from Azure Blob Storage
// This is called when upload succeeds but message creation fails or is cancelled
func (service *ServiceImpl) DeleteMessageImage(user *bootstrap.User, shopID string, messageID string) error {
	return service.MessagesService.DeleteMessageImage(user, shopID, messageID)
}

// Shop Vehicle Operations
func (service *ServiceImpl) CreateShopVehicle(user *bootstrap.User, vehicle model.ShopVehicle) (*model.ShopVehicle, error) {
	return service.VehiclesService.CreateShopVehicle(user, vehicle)
}

func (service *ServiceImpl) GetShopVehicles(user *bootstrap.User, shopID string) ([]model.ShopVehicle, error) {
	return service.VehiclesService.GetShopVehicles(user, shopID)
}

func (service *ServiceImpl) GetShopVehicleByID(user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error) {
	return service.VehiclesService.GetShopVehicleByID(user, vehicleID)
}

func (service *ServiceImpl) UpdateShopVehicle(user *bootstrap.User, vehicle model.ShopVehicle) error {
	return service.VehiclesService.UpdateShopVehicle(user, vehicle)
}

func (service *ServiceImpl) DeleteShopVehicle(user *bootstrap.User, vehicleID string) error {
	return service.VehiclesService.DeleteShopVehicle(user, vehicleID)
}

// Shop Vehicle Notification Operations
func (service *ServiceImpl) CreateVehicleNotification(user *bootstrap.User, notification model.ShopVehicleNotifications) (*model.ShopVehicleNotifications, error) {
	return service.NotificationsService.CreateVehicleNotification(user, notification)
}

func (service *ServiceImpl) GetVehicleNotifications(user *bootstrap.User, vehicleID string) ([]model.ShopVehicleNotifications, error) {
	return service.NotificationsService.GetVehicleNotifications(user, vehicleID)
}

func (service *ServiceImpl) GetVehicleNotificationsWithItems(user *bootstrap.User, vehicleID string) ([]response.VehicleNotificationWithItems, error) {
	return service.NotificationsService.GetVehicleNotificationsWithItems(user, vehicleID)
}

func (service *ServiceImpl) GetShopNotifications(user *bootstrap.User, shopID string) ([]model.ShopVehicleNotifications, error) {
	return service.NotificationsService.GetShopNotifications(user, shopID)
}

func (service *ServiceImpl) GetVehicleNotificationByID(user *bootstrap.User, notificationID string) (*model.ShopVehicleNotifications, error) {
	return service.NotificationsService.GetVehicleNotificationByID(user, notificationID)
}

func (service *ServiceImpl) UpdateVehicleNotification(user *bootstrap.User, notification model.ShopVehicleNotifications) error {
	return service.NotificationsService.UpdateVehicleNotification(user, notification)
}

func (service *ServiceImpl) DeleteVehicleNotification(user *bootstrap.User, notificationID string) error {
	return service.NotificationsService.DeleteVehicleNotification(user, notificationID)
}

// Shop Notification Item Operations
func (service *ServiceImpl) AddNotificationItem(user *bootstrap.User, item model.ShopNotificationItems) (*model.ShopNotificationItems, error) {
	return service.NotificationItemsService.AddNotificationItem(user, item)
}

func (service *ServiceImpl) GetNotificationItems(user *bootstrap.User, notificationID string) ([]model.ShopNotificationItems, error) {
	return service.NotificationItemsService.GetNotificationItems(user, notificationID)
}

func (service *ServiceImpl) GetShopNotificationItems(user *bootstrap.User, shopID string) ([]model.ShopNotificationItems, error) {
	return service.NotificationItemsService.GetShopNotificationItems(user, shopID)
}

func (service *ServiceImpl) AddNotificationItemList(user *bootstrap.User, items []model.ShopNotificationItems) ([]model.ShopNotificationItems, error) {
	return service.NotificationItemsService.AddNotificationItemList(user, items)
}

func (service *ServiceImpl) RemoveNotificationItem(user *bootstrap.User, itemID string) error {
	return service.NotificationItemsService.RemoveNotificationItem(user, itemID)
}

func (service *ServiceImpl) RemoveNotificationItemList(user *bootstrap.User, itemIDs []string) error {
	return service.NotificationItemsService.RemoveNotificationItemList(user, itemIDs)
}

// Shop List Operations
func (service *ServiceImpl) CreateShopList(user *bootstrap.User, list model.ShopLists) (*response.ShopListWithUsername, error) {
	return service.ListsService.CreateShopList(user, list)
}

func (service *ServiceImpl) GetShopLists(user *bootstrap.User, shopID string) ([]response.ShopListWithUsername, error) {
	return service.ListsService.GetShopLists(user, shopID)
}

func (service *ServiceImpl) GetShopListByID(user *bootstrap.User, listID string) (*response.ShopListWithUsername, error) {
	return service.ListsService.GetShopListByID(user, listID)
}

func (service *ServiceImpl) UpdateShopList(user *bootstrap.User, list model.ShopLists) error {
	return service.ListsService.UpdateShopList(user, list)
}

func (service *ServiceImpl) DeleteShopList(user *bootstrap.User, listID string) error {
	return service.ListsService.DeleteShopList(user, listID)
}

// Shop List Item Operations
func (service *ServiceImpl) AddListItem(user *bootstrap.User, item model.ShopListItems) (*response.ShopListItemWithUsername, error) {
	return service.ListItemsService.AddListItem(user, item)
}

func (service *ServiceImpl) GetListItems(user *bootstrap.User, listID string) ([]response.ShopListItemWithUsername, error) {
	return service.ListItemsService.GetListItems(user, listID)
}

func (service *ServiceImpl) UpdateListItem(user *bootstrap.User, item model.ShopListItems) error {
	return service.ListItemsService.UpdateListItem(user, item)
}

func (service *ServiceImpl) RemoveListItem(user *bootstrap.User, itemID string) error {
	return service.ListItemsService.RemoveListItem(user, itemID)
}

func (service *ServiceImpl) AddListItemBatch(user *bootstrap.User, items []model.ShopListItems) ([]response.ShopListItemWithUsername, error) {
	return service.ListItemsService.AddListItemBatch(user, items)
}

func (service *ServiceImpl) RemoveListItemBatch(user *bootstrap.User, itemIDs []string) error {
	return service.ListItemsService.RemoveListItemBatch(user, itemIDs)
}

// Shop Settings Operations
func (service *ServiceImpl) GetShopAdminOnlyListsSetting(user *bootstrap.User, shopID string) (bool, error) {
	return service.SettingsService.GetShopAdminOnlyListsSetting(user, shopID)
}

func (service *ServiceImpl) UpdateShopAdminOnlyListsSetting(user *bootstrap.User, shopID string, adminOnlyLists bool) error {
	return service.SettingsService.UpdateShopAdminOnlyListsSetting(user, shopID, adminOnlyLists)
}

func (service *ServiceImpl) IsUserShopAdmin(user *bootstrap.User, shopID string) (bool, error) {
	return service.SettingsService.IsUserShopAdmin(user, shopID)
}

// Unified Shop Settings Operations
func (service *ServiceImpl) GetShopSettings(user *bootstrap.User, shopID string) (*request.ShopSettings, error) {
	return service.SettingsService.GetShopSettings(user, shopID)
}

func (service *ServiceImpl) UpdateShopSettings(user *bootstrap.User, shopID string, updates request.UpdateShopSettingsRequest) (*request.ShopSettings, error) {
	return service.SettingsService.UpdateShopSettings(user, shopID, updates)
}

// Notification Change Tracking (Audit Trail) Operations
func (service *ServiceImpl) GetNotificationChangeHistory(user *bootstrap.User, notificationID string) ([]response.NotificationChangeWithUsername, error) {
	return service.NotificationChangesService.GetNotificationChangeHistory(user, notificationID)
}

func (service *ServiceImpl) GetShopNotificationChanges(user *bootstrap.User, shopID string, limit int) ([]response.NotificationChangeWithUsername, error) {
	return service.NotificationChangesService.GetShopNotificationChanges(user, shopID, limit)
}

func (service *ServiceImpl) GetVehicleNotificationChanges(user *bootstrap.User, vehicleID string) ([]response.NotificationChangeWithUsername, error) {
	return service.NotificationChangesService.GetVehicleNotificationChanges(user, vehicleID)
}
