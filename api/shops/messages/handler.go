package messages

import (
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

// Shop Message Operations

// CreateShopMessage creates a new message in the shop chat
func (handler *Handler) CreateShopMessage(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	var req request.CreateShopMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		response.Error(c, 400, "invalid request")
		return
	}

	message := model.ShopMessages{
		ShopID:   req.ShopID,
		Message:  req.Message,
		ParentID: req.ParentID,
	}

	service := handler.service
	createdMessage, err := service.CreateShopMessage(c.Request.Context(), user, message)
	if err != nil {
		c.Error(err)
		return
	}

	// Kept as a raw StandardResponse literal: response.OK hardcodes an empty
	// Message and has no parameter to carry this success text.
	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "Message created successfully",
		Data:    *createdMessage,
	})
}

// GetShopMessages returns all messages for a shop
func (handler *Handler) GetShopMessages(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	shopID := c.Param("shop_id")
	if shopID == "" {
		response.Error(c, 400, "shop_id is required")
		return
	}

	service := handler.service
	messages, err := service.GetShopMessages(c.Request.Context(), user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, messages)
}

// GetShopMessagesPaginated returns paginated messages for a shop
func (handler *Handler) GetShopMessagesPaginated(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	shopID := c.Param("shop_id")
	if shopID == "" {
		response.Error(c, 400, "shop_id is required")
		return
	}

	var req request.GetShopMessagesPaginatedRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		slog.Info("invalid query parameters", "error", err)
		response.Error(c, 400, "invalid query parameters")
		return
	}

	// Set defaults if not provided
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 20
	}
	if req.BeforeID != nil && req.AfterID != nil {
		response.Error(c, 400, "before_id and after_id cannot be used together")
		return
	}

	service := handler.service
	paginatedMessages, err := service.GetShopMessagesPaginated(c.Request.Context(), user, shopID, req)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, *paginatedMessages)
}

// UpdateShopMessage updates an existing shop message
func (handler *Handler) UpdateShopMessage(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	var req request.UpdateShopMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		response.Error(c, 400, "invalid request")
		return
	}

	message := model.ShopMessages{
		ID:      req.MessageID,
		Message: req.Message,
	}

	service := handler.service
	err := service.UpdateShopMessage(c.Request.Context(), user, message)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, gin.H{"message": "Message updated successfully"})
}

// DeleteShopMessage deletes a shop message
func (handler *Handler) DeleteShopMessage(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	messageID := c.Param("message_id")
	if messageID == "" {
		response.Error(c, 400, "message_id is required")
		return
	}

	service := handler.service
	err := service.DeleteShopMessage(c.Request.Context(), user, messageID)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, gin.H{"message": "Message deleted successfully"})
}

// UploadMessageImage handles image upload for shop messages
func (handler *Handler) UploadMessageImage(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	// Get shop_id from query parameter or form data
	shopID := c.Query("shop_id")
	if shopID == "" {
		shopID = c.PostForm("shop_id")
	}
	if shopID == "" {
		response.Error(c, 400, "shop_id is required")
		return
	}

	// Get the uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		slog.Error("Error getting uploaded file", "error", err)
		response.Error(c, 400, "failed to get uploaded file")
		return
	}
	defer file.Close()

	// Check file size before reading
	if header.Size > 5*1024*1024 { // 5MB
		response.Error(c, 400, "file size exceeds maximum allowed size of 5MB")
		return
	}

	// Read file data
	imageData := make([]byte, header.Size)
	_, err = file.Read(imageData)
	if err != nil {
		slog.Error("Error reading file data", "error", err)
		response.Error(c, 500, "failed to read file data")
		return
	}

	// Get content type from header
	contentType := header.Header.Get("Content-Type")

	// Upload to blob storage
	service := handler.service
	messageID, fileExtension, imageURL, err := service.UploadMessageImage(c.Request.Context(), user, shopID, imageData, contentType)
	if err != nil {
		slog.Error("Error uploading image to blob storage", "error", err)
		response.Error(c, 500, fmt.Sprintf("failed to upload image: %v", err))
		return
	}

	// Kept as a raw StandardResponse literal: response.OK hardcodes an empty
	// Message and has no parameter to carry this success text. The nested
	// gin.H{} here is the Data field's payload shape, not an envelope.
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
func (handler *Handler) DeleteMessageImage(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	messageID := c.Param("message_id")
	if messageID == "" {
		response.Error(c, 400, "message_id is required")
		return
	}

	shopID := c.Query("shop_id")
	if shopID == "" {
		response.Error(c, 400, "shop_id is required")
		return
	}

	service := handler.service
	err := service.DeleteMessageImage(c.Request.Context(), user, shopID, messageID)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, gin.H{"message": "Image deleted successfully"})
}
