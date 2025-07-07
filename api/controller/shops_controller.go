package controller

import (
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/api/service"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

type ShopsController struct {
	ShopsService service.ShopsService
}

func NewShopsController(shopsService service.ShopsService) *ShopsController {
	return &ShopsController{ShopsService: shopsService}
}

// Shop Operations

// CreateShop handles shop creation
func (controller *ShopsController) CreateShop(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.CreateShopRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	shop := model.Shops{
		Name:         req.Name,
		Details:      req.Details,
		PasswordHash: req.PasswordHash,
	}

	createdShop, err := controller.ShopsService.CreateShop(user, shop)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "Shop created successfully",
		Data:    *createdShop,
	})
}

// DeleteShop handles shop deletion
func (controller *ShopsController) DeleteShop(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	shopID := c.Param("shop_id")
	if shopID == "" {
		c.JSON(400, gin.H{"message": "shop_id is required"})
		return
	}

	err := controller.ShopsService.DeleteShop(user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Shop deleted successfully"})
}

// GetUserShops returns all shops for the authenticated user
func (controller *ShopsController) GetUserShops(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	shops, err := controller.ShopsService.GetShopsByUser(user)
	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    shops,
	})
}

// GetUserDataWithShops returns user data along with all shops they are a part of
func (controller *ShopsController) GetUserDataWithShops(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	userShopsData, err := controller.ShopsService.GetUserDataWithShops(user)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "User data and shops retrieved successfully",
		Data:    *userShopsData,
	})
}

// GetShopByID returns a specific shop by ID
func (controller *ShopsController) GetShopByID(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	shopID := c.Param("shop_id")
	if shopID == "" {
		c.JSON(400, gin.H{"message": "shop_id is required"})
		return
	}

	shop, err := controller.ShopsService.GetShopByID(user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    *shop,
	})
}

// Shop Member Operations

// JoinShopViaInviteCode allows a user to join a shop using an invite code
func (controller *ShopsController) JoinShopViaInviteCode(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.JoinShopRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.ShopsService.JoinShopViaInviteCode(user, req.InviteCode)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Successfully joined shop"})
}

// LeaveShop allows a user to leave a shop
func (controller *ShopsController) LeaveShop(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	shopID := c.Param("shop_id")
	if shopID == "" {
		c.JSON(400, gin.H{"message": "shop_id is required"})
		return
	}

	err := controller.ShopsService.LeaveShop(user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Successfully left shop"})
}

// RemoveMemberFromShop allows admins to remove members from a shop
func (controller *ShopsController) RemoveMemberFromShop(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.RemoveMemberRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.ShopsService.RemoveMemberFromShop(user, req.ShopID, req.TargetUserID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Member removed successfully"})
}

// GetShopMembers returns all members of a shop
func (controller *ShopsController) GetShopMembers(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	shopID := c.Param("shop_id")
	if shopID == "" {
		c.JSON(400, gin.H{"message": "shop_id is required"})
		return
	}

	members, err := controller.ShopsService.GetShopMembers(user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    members,
	})
}

// Shop Invite Code Operations

// GenerateInviteCode creates a new invite code for a shop
func (controller *ShopsController) GenerateInviteCode(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.GenerateInviteCodeRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	code, err := controller.ShopsService.GenerateInviteCode(user, req.ShopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "Invite code generated successfully",
		Data:    *code,
	})
}

// GetInviteCodesByShop returns all invite codes for a shop
func (controller *ShopsController) GetInviteCodesByShop(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	shopID := c.Param("shop_id")
	if shopID == "" {
		c.JSON(400, gin.H{"message": "shop_id is required"})
		return
	}

	codes, err := controller.ShopsService.GetInviteCodesByShop(user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    codes,
	})
}

// DeactivateInviteCode deactivates an invite code
func (controller *ShopsController) DeactivateInviteCode(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	codeID := c.Param("code_id")
	if codeID == "" {
		c.JSON(400, gin.H{"message": "code_id is required"})
		return
	}

	err := controller.ShopsService.DeactivateInviteCode(user, codeID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Invite code deactivated successfully"})
}

// Shop Message Operations

// CreateShopMessage creates a new message in the shop chat
func (controller *ShopsController) CreateShopMessage(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.CreateShopMessageRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	message := model.ShopMessages{
		ShopID:  req.ShopID,
		Message: req.Message,
	}

	createdMessage, err := controller.ShopsService.CreateShopMessage(user, message)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "Message created successfully",
		Data:    *createdMessage,
	})
}

// GetShopMessages returns all messages for a shop
func (controller *ShopsController) GetShopMessages(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	shopID := c.Param("shop_id")
	if shopID == "" {
		c.JSON(400, gin.H{"message": "shop_id is required"})
		return
	}

	messages, err := controller.ShopsService.GetShopMessages(user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    messages,
	})
}

// UpdateShopMessage updates an existing shop message
func (controller *ShopsController) UpdateShopMessage(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.UpdateShopMessageRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	message := model.ShopMessages{
		ID:      req.MessageID,
		Message: req.Message,
	}

	err := controller.ShopsService.UpdateShopMessage(user, message)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Message updated successfully"})
}

// DeleteShopMessage deletes a shop message
func (controller *ShopsController) DeleteShopMessage(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	messageID := c.Param("message_id")
	if messageID == "" {
		c.JSON(400, gin.H{"message": "message_id is required"})
		return
	}

	err := controller.ShopsService.DeleteShopMessage(user, messageID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Message deleted successfully"})
}

// Shop Vehicle Operations

// CreateShopVehicle creates a new vehicle for a shop
func (controller *ShopsController) CreateShopVehicle(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.CreateShopVehicleRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	vehicle := model.ShopVehicle{
		ShopID:  req.ShopID,
		Niin:    req.Niin,
		Model:   req.Model,
		Serial:  req.Serial,
		Uoc:     req.Uoc,
		Mileage: req.Mileage,
		Hours:   req.Hours,
		Comment: req.Comment,
	}

	createdVehicle, err := controller.ShopsService.CreateShopVehicle(user, vehicle)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "Vehicle created successfully",
		Data:    *createdVehicle,
	})
}

// GetShopVehicles returns all vehicles for a shop
func (controller *ShopsController) GetShopVehicles(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	shopID := c.Param("shop_id")
	if shopID == "" {
		c.JSON(400, gin.H{"message": "shop_id is required"})
		return
	}

	vehicles, err := controller.ShopsService.GetShopVehicles(user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    vehicles,
	})
}

// GetShopVehicleByID returns a specific vehicle by ID
func (controller *ShopsController) GetShopVehicleByID(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	vehicleID := c.Param("vehicle_id")
	if vehicleID == "" {
		c.JSON(400, gin.H{"message": "vehicle_id is required"})
		return
	}

	vehicle, err := controller.ShopsService.GetShopVehicleByID(user, vehicleID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    *vehicle,
	})
}

// UpdateShopVehicle updates an existing shop vehicle
func (controller *ShopsController) UpdateShopVehicle(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.UpdateShopVehicleRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	vehicle := model.ShopVehicle{
		ID:      req.VehicleID,
		Model:   req.Model,
		Serial:  req.Serial,
		Uoc:     req.Uoc,
		Mileage: req.Mileage,
		Hours:   req.Hours,
		Comment: req.Comment,
	}

	err := controller.ShopsService.UpdateShopVehicle(user, vehicle)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Vehicle updated successfully"})
}

// DeleteShopVehicle deletes a shop vehicle
func (controller *ShopsController) DeleteShopVehicle(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	vehicleID := c.Param("vehicle_id")
	if vehicleID == "" {
		c.JSON(400, gin.H{"message": "vehicle_id is required"})
		return
	}

	err := controller.ShopsService.DeleteShopVehicle(user, vehicleID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Vehicle deleted successfully"})
}

// Shop Vehicle Notification Operations

// CreateVehicleNotification creates a new notification for a vehicle
func (controller *ShopsController) CreateVehicleNotification(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.CreateVehicleNotificationRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	notification := model.ShopVehicleNotifications{
		VehicleID:   req.VehicleID,
		Title:       req.Title,
		Description: req.Description,
		Type:        req.Type,
		Completed:   false,
	}

	createdNotification, err := controller.ShopsService.CreateVehicleNotification(user, notification)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "Notification created successfully",
		Data:    *createdNotification,
	})
}

// GetVehicleNotifications returns all notifications for a vehicle
func (controller *ShopsController) GetVehicleNotifications(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	vehicleID := c.Param("vehicle_id")
	if vehicleID == "" {
		c.JSON(400, gin.H{"message": "vehicle_id is required"})
		return
	}

	notifications, err := controller.ShopsService.GetVehicleNotifications(user, vehicleID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    notifications,
	})
}

// GetVehicleNotificationByID returns a specific notification by ID
func (controller *ShopsController) GetVehicleNotificationByID(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	notificationID := c.Param("notification_id")
	if notificationID == "" {
		c.JSON(400, gin.H{"message": "notification_id is required"})
		return
	}

	notification, err := controller.ShopsService.GetVehicleNotificationByID(user, notificationID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    *notification,
	})
}

// UpdateVehicleNotification updates an existing vehicle notification
func (controller *ShopsController) UpdateVehicleNotification(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.UpdateVehicleNotificationRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	notification := model.ShopVehicleNotifications{
		ID:          req.NotificationID,
		Title:       req.Title,
		Description: req.Description,
		Type:        req.Type,
		Completed:   req.Completed,
	}

	err := controller.ShopsService.UpdateVehicleNotification(user, notification)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Notification updated successfully"})
}

// DeleteVehicleNotification deletes a vehicle notification
func (controller *ShopsController) DeleteVehicleNotification(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	notificationID := c.Param("notification_id")
	if notificationID == "" {
		c.JSON(400, gin.H{"message": "notification_id is required"})
		return
	}

	err := controller.ShopsService.DeleteVehicleNotification(user, notificationID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Notification deleted successfully"})
}

// Shop Notification Item Operations

// AddNotificationItem adds an item to a vehicle notification
func (controller *ShopsController) AddNotificationItem(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.AddNotificationItemRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	item := model.ShopNotificationItems{
		NotificationID: req.NotificationID,
		Niin:           req.Niin,
		Nomenclature:   req.Nomenclature,
		Quantity:       req.Quantity,
	}

	createdItem, err := controller.ShopsService.AddNotificationItem(user, item)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "Item added successfully",
		Data:    *createdItem,
	})
}

// GetNotificationItems returns all items for a notification
func (controller *ShopsController) GetNotificationItems(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	notificationID := c.Param("notification_id")
	if notificationID == "" {
		c.JSON(400, gin.H{"message": "notification_id is required"})
		return
	}

	items, err := controller.ShopsService.GetNotificationItems(user, notificationID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    items,
	})
}

// AddNotificationItemList adds multiple items to a vehicle notification
func (controller *ShopsController) AddNotificationItemList(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.AddNotificationItemListRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	var items []model.ShopNotificationItems
	for _, reqItem := range req.Items {
		item := model.ShopNotificationItems{
			NotificationID: reqItem.NotificationID,
			Niin:           reqItem.Niin,
			Nomenclature:   reqItem.Nomenclature,
			Quantity:       reqItem.Quantity,
		}
		items = append(items, item)
	}

	err := controller.ShopsService.AddNotificationItemList(user, items)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(201, gin.H{"message": "Items added successfully", "count": len(items)})
}

// RemoveNotificationItem removes an item from a vehicle notification
func (controller *ShopsController) RemoveNotificationItem(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	itemID := c.Param("item_id")
	if itemID == "" {
		c.JSON(400, gin.H{"message": "item_id is required"})
		return
	}

	err := controller.ShopsService.RemoveNotificationItem(user, itemID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Item removed successfully"})
}

// RemoveNotificationItemList removes multiple items from vehicle notifications
func (controller *ShopsController) RemoveNotificationItemList(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.RemoveNotificationItemListRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.ShopsService.RemoveNotificationItemList(user, req.ItemIDs)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Items removed successfully", "count": len(req.ItemIDs)})
}
