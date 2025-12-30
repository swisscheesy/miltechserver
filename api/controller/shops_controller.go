package controller

import (
	"fmt"
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
		Name:    req.Name,
		Details: req.Details,
	}

	// Set admin_only_lists if provided, otherwise defaults to false in database
	if req.AdminOnlyLists != nil {
		shop.AdminOnlyLists = *req.AdminOnlyLists
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

// UpdateShop handles shop updates
func (controller *ShopsController) UpdateShop(c *gin.Context) {
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

	var req request.UpdateShopRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	shop := model.Shops{
		ID:      shopID,
		Name:    req.Name,
		Details: req.Details,
	}

	updatedShop, err := controller.ShopsService.UpdateShop(user, shop)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Shop updated successfully",
		Data:    *updatedShop,
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

// PromoteMemberToAdmin allows admins to promote members to admin role
func (controller *ShopsController) PromoteMemberToAdmin(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.PromoteMemberRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.ShopsService.PromoteMemberToAdmin(user, req.ShopID, req.TargetUserID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Member promoted to admin successfully"})
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

// DeleteInviteCode permanently deletes an invite code
func (controller *ShopsController) DeleteInviteCode(c *gin.Context) {
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

	err := controller.ShopsService.DeleteInviteCode(user, codeID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Invite code deleted successfully"})
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

// GetShopMessagesPaginated returns paginated messages for a shop
func (controller *ShopsController) GetShopMessagesPaginated(c *gin.Context) {
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

	var req request.GetShopMessagesPaginatedRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		slog.Info("invalid query parameters", "error", err)
		c.JSON(400, gin.H{"message": "invalid query parameters"})
		return
	}

	// Set defaults if not provided
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 20
	}

	paginatedMessages, err := controller.ShopsService.GetShopMessagesPaginated(user, shopID, req.Page, req.Limit)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    *paginatedMessages,
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

// UploadMessageImage handles image upload for shop messages
func (controller *ShopsController) UploadMessageImage(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	// Get shop_id from query parameter or form data
	shopID := c.Query("shop_id")
	if shopID == "" {
		shopID = c.PostForm("shop_id")
	}
	if shopID == "" {
		c.JSON(400, gin.H{"message": "shop_id is required"})
		return
	}

	// Get the uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		slog.Error("Error getting uploaded file", "error", err)
		c.JSON(400, gin.H{"message": "failed to get uploaded file"})
		return
	}
	defer file.Close()

	// Check file size before reading
	if header.Size > 5*1024*1024 { // 5MB
		c.JSON(400, gin.H{"message": "file size exceeds maximum allowed size of 5MB"})
		return
	}

	// Read file data
	imageData := make([]byte, header.Size)
	_, err = file.Read(imageData)
	if err != nil {
		slog.Error("Error reading file data", "error", err)
		c.JSON(500, gin.H{"message": "failed to read file data"})
		return
	}

	// Get content type from header
	contentType := header.Header.Get("Content-Type")

	// Upload to blob storage
	messageID, fileExtension, imageURL, err := controller.ShopsService.UploadMessageImage(user, shopID, imageData, contentType)
	if err != nil {
		slog.Error("Error uploading image to blob storage", "error", err)
		c.JSON(500, gin.H{"message": fmt.Sprintf("failed to upload image: %v", err)})
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Image uploaded successfully",
		Data: gin.H{
			"message_id":     messageID,
			"shop_id":        shopID,
			"image_url":      imageURL,
			"file_extension": fileExtension,
		},
	})
}

// DeleteMessageImage handles deletion of orphaned message images
func (controller *ShopsController) DeleteMessageImage(c *gin.Context) {
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

	shopID := c.Query("shop_id")
	if shopID == "" {
		c.JSON(400, gin.H{"message": "shop_id is required"})
		return
	}

	err := controller.ShopsService.DeleteMessageImage(user, shopID, messageID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Image deleted successfully"})
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
		Admin:   req.Admin,
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
		ID:             req.VehicleID,
		Admin:          req.Admin,
		Niin:           req.Niin,
		Model:          req.Model,
		Serial:         req.Serial,
		Uoc:            req.Uoc,
		Mileage:        req.Mileage,
		Hours:          req.Hours,
		Comment:        req.Comment,
		TrackedMileage: req.TrackedMileage,
		TrackedHours:   req.TrackedHours,
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
		ShopID:      req.ShopID,
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

// GetVehicleNotificationsWithItems returns all notifications for a vehicle with their items
func (controller *ShopsController) GetVehicleNotificationsWithItems(c *gin.Context) {
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

	notificationsWithItems, err := controller.ShopsService.GetVehicleNotificationsWithItems(user, vehicleID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    notificationsWithItems,
	})
}

// GetShopNotifications returns all notifications for a shop
func (controller *ShopsController) GetShopNotifications(c *gin.Context) {
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

	notifications, err := controller.ShopsService.GetShopNotifications(user, shopID)
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

// GetShopNotificationItems returns all notification items for a shop
func (controller *ShopsController) GetShopNotificationItems(c *gin.Context) {
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

	items, err := controller.ShopsService.GetShopNotificationItems(user, shopID)
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

	createdItems, err := controller.ShopsService.AddNotificationItemList(user, items)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "Items added successfully",
		Data:    createdItems,
	})
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

// Shop List Operations

// CreateShopList creates a new list for a shop
func (controller *ShopsController) CreateShopList(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.CreateShopListRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	list := model.ShopLists{
		ShopID:      req.ShopID,
		Description: req.Description,
	}

	createdList, err := controller.ShopsService.CreateShopList(user, list)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "List created successfully",
		Data:    *createdList,
	})
}

// GetShopLists returns all lists for a shop with creator usernames
func (controller *ShopsController) GetShopLists(c *gin.Context) {
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

	lists, err := controller.ShopsService.GetShopLists(user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    lists,
	})
}

// GetShopListByID returns a specific list by ID
func (controller *ShopsController) GetShopListByID(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	listID := c.Param("list_id")
	if listID == "" {
		c.JSON(400, gin.H{"message": "list_id is required"})
		return
	}

	list, err := controller.ShopsService.GetShopListByID(user, listID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    *list,
	})
}

// UpdateShopList updates an existing shop list
func (controller *ShopsController) UpdateShopList(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.UpdateShopListRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	list := model.ShopLists{
		ID:          req.ListID,
		Description: req.Description,
	}

	err := controller.ShopsService.UpdateShopList(user, list)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "List updated successfully"})
}

// DeleteShopList deletes a shop list
func (controller *ShopsController) DeleteShopList(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.DeleteShopListRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.ShopsService.DeleteShopList(user, req.ListID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "List deleted successfully"})
}

// Shop List Item Operations

// AddListItem adds an item to a shop list
func (controller *ShopsController) AddListItem(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.AddListItemRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	item := model.ShopListItems{
		ListID:        req.ListID,
		Niin:          req.Niin,
		Nomenclature:  req.Nomenclature,
		Quantity:      req.Quantity,
		Nickname:      req.Nickname,
		UnitOfMeasure: req.UnitOfMeasure,
	}

	createdItem, err := controller.ShopsService.AddListItem(user, item)
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

// GetListItems returns all items for a list with added by usernames
func (controller *ShopsController) GetListItems(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	listID := c.Param("list_id")
	if listID == "" {
		c.JSON(400, gin.H{"message": "list_id is required"})
		return
	}

	items, err := controller.ShopsService.GetListItems(user, listID)
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

// UpdateListItem updates an existing list item
func (controller *ShopsController) UpdateListItem(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.UpdateListItemRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	item := model.ShopListItems{
		ID:            req.ItemID,
		Niin:          req.Niin,
		Nomenclature:  req.Nomenclature,
		Quantity:      req.Quantity,
		Nickname:      req.Nickname,
		UnitOfMeasure: req.UnitOfMeasure,
	}

	err := controller.ShopsService.UpdateListItem(user, item)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Item updated successfully"})
}

// RemoveListItem removes an item from a list
func (controller *ShopsController) RemoveListItem(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.RemoveListItemRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.ShopsService.RemoveListItem(user, req.ItemID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Item removed successfully"})
}

// AddListItemBatch adds multiple items to a list
func (controller *ShopsController) AddListItemBatch(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.AddListItemBatchRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	var items []model.ShopListItems
	for _, reqItem := range req.Items {
		item := model.ShopListItems{
			ListID:        reqItem.ListID,
			Niin:          reqItem.Niin,
			Nomenclature:  reqItem.Nomenclature,
			Quantity:      reqItem.Quantity,
			Nickname:      reqItem.Nickname,
			UnitOfMeasure: reqItem.UnitOfMeasure,
		}
		items = append(items, item)
	}

	createdItems, err := controller.ShopsService.AddListItemBatch(user, items)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "Items added successfully",
		Data:    createdItems,
	})
}

// RemoveListItemBatch removes multiple items from lists
func (controller *ShopsController) RemoveListItemBatch(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.RemoveListItemBatchRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.ShopsService.RemoveListItemBatch(user, req.ItemIDs)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Items removed successfully", "count": len(req.ItemIDs)})
}

// Notification Change Tracking (Audit Trail) Operations

// GetNotificationChangeHistory returns the complete change history for a notification
func (controller *ShopsController) GetNotificationChangeHistory(c *gin.Context) {
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

	changes, err := controller.ShopsService.GetNotificationChangeHistory(user, notificationID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    changes,
	})
}

// GetShopNotificationChanges returns recent notification changes for all notifications in a shop
func (controller *ShopsController) GetShopNotificationChanges(c *gin.Context) {
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

	// Get optional limit parameter (default 100, max 500)
	limit := 100
	if limitParam := c.Query("limit"); limitParam != "" {
		var parsedLimit int
		if _, err := fmt.Sscanf(limitParam, "%d", &parsedLimit); err == nil {
			limit = parsedLimit
		}
	}

	changes, err := controller.ShopsService.GetShopNotificationChanges(user, shopID, limit)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    changes,
	})
}

// GetVehicleNotificationChanges returns all notification changes for a specific vehicle
func (controller *ShopsController) GetVehicleNotificationChanges(c *gin.Context) {
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

	changes, err := controller.ShopsService.GetVehicleNotificationChanges(user, vehicleID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    changes,
	})
}

// Shop Settings Operations

// GetShopAdminOnlyListsSetting returns the admin_only_lists setting for a shop
func (controller *ShopsController) GetShopAdminOnlyListsSetting(c *gin.Context) {
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

	adminOnlyLists, err := controller.ShopsService.GetShopAdminOnlyListsSetting(user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Shop admin_only_lists setting retrieved successfully",
		Data: gin.H{
			"shop_id":         shopID,
			"admin_only_lists": adminOnlyLists,
		},
	})
}

// UpdateShopAdminOnlyListsSetting updates the admin_only_lists setting for a shop (admin only)
func (controller *ShopsController) UpdateShopAdminOnlyListsSetting(c *gin.Context) {
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

	var req request.UpdateAdminOnlyListsRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.ShopsService.UpdateShopAdminOnlyListsSetting(user, shopID, req.AdminOnlyLists)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Shop admin_only_lists setting updated successfully",
		Data: gin.H{
			"shop_id":         shopID,
			"admin_only_lists": req.AdminOnlyLists,
		},
	})
}

// CheckUserIsShopAdmin checks if the current user is an admin for the specified shop
func (controller *ShopsController) CheckUserIsShopAdmin(c *gin.Context) {
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

	isAdmin, err := controller.ShopsService.IsUserShopAdmin(user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Admin status checked successfully",
		Data: gin.H{
			"shop_id":  shopID,
			"user_id":  user.UserID,
			"is_admin": isAdmin,
		},
	})
}
