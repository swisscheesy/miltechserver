package messages

import (
	"errors"
	"io"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	ctxUser, _ := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)
	if user == nil || user.UserID == "" {
		response.Error(c, 401, "unauthorized")
		return
	}
	// Install the whole-body bound before any multipart/form access. Keep multipart file
	// buffering bounded; cleanup covers every return.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadRequestBytes)
	defer func() {
		if c.Request.MultipartForm != nil {
			if err := c.Request.MultipartForm.RemoveAll(); err != nil {
				slog.Warn("Multipart cleanup failed")
			}
		}
	}()
	if err := c.Request.ParseMultipartForm(1024 * 1024); err != nil {
		response.Error(c, 400, "invalid or oversized multipart body")
		return
	}
	form := c.Request.MultipartForm
	if len(form.File) != 1 || len(form.File["file"]) != 1 || len(form.Value) > 1 || len(form.Value["shop_id"]) > 1 {
		response.Error(c, 400, "invalid multipart parts")
		return
	}
	for key := range form.Value {
		if key != "shop_id" {
			response.Error(c, 400, "invalid multipart field")
			return
		}
	}
	shopID := c.Query("shop_id")
	if values := form.Value["shop_id"]; len(values) == 1 {
		if len(values[0]) > 36 || (shopID != "" && shopID != values[0]) {
			response.Error(c, 400, "invalid shop_id")
			return
		}
		shopID = values[0]
	}
	if id, err := uuid.Parse(shopID); err != nil || id == uuid.Nil {
		response.Error(c, 400, "invalid shop_id")
		return
	}
	header := form.File["file"][0]
	file, err := header.Open()
	if err != nil {
		response.Error(c, 400, "failed to read uploaded file")
		return
	}
	defer file.Close()
	imageData, err := io.ReadAll(io.LimitReader(file, maxImageSize+1))
	if err != nil || len(imageData) == 0 || len(imageData) > maxImageSize || c.Request.Context().Err() != nil {
		response.Error(c, 400, "invalid image data")
		return
	}
	upload, err := handler.service.UploadMessageImage(c.Request.Context(), user, shopID, imageData, header.Header.Get("Content-Type"))
	if err != nil {
		status := 500
		if errors.Is(err, shared.ErrShopAccessDenied) {
			status = 403
		}
		response.Error(c, status, "failed to upload image")
		return
	}
	c.JSON(200, response.StandardResponse{Status: 200, Message: "Image uploaded successfully", Data: upload})
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
