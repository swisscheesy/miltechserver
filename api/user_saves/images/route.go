package images

import (
	"fmt"
	"log/slog"
	"miltechserver/api/response"
	"miltechserver/api/user_saves/shared"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	handler := Handler{service: service}

	router.POST("/user/saves/items/image/upload/:table_type", handler.uploadItemImage)
	router.DELETE("/user/saves/items/image/:table_type", handler.deleteItemImage)
	router.GET("/user/saves/items/image/:table_type", handler.getItemImage)
}

// Example endpoint: POST /api/v1/auth/user/saves/items/image/upload/:table_type
func (handler *Handler) uploadItemImage(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	tableType := c.Param("table_type")
	if tableType == "" {
		response.Error(c, 400, "table_type is required")
		return
	}

	itemID := c.Query("item_id")
	if itemID == "" {
		response.Error(c, 400, "item_id is required")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		slog.Error("Error getting uploaded file", "error", err)
		response.Error(c, 400, "failed to get uploaded file")
		return
	}
	defer file.Close()

	imageData := make([]byte, header.Size)
	_, err = file.Read(imageData)
	if err != nil {
		slog.Error("Error reading file data", "error", err)
		response.Error(c, 500, "failed to read file data")
		return
	}

	blobURL, err := handler.service.Upload(c.Request.Context(), user, itemID, tableType, imageData)
	if err != nil {
		slog.Error("Error uploading image to blob storage", "error", err)
		response.Error(c, 500, "failed to upload image")
		return
	}

	// Kept as a raw StandardResponse literal: response.OK hardcodes an empty
	// Message and has no parameter to carry this success text.
	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Image uploaded successfully",
		Data:    gin.H{"item_id": itemID, "table_type": tableType, "blob_url": blobURL},
	})
}

// Example endpoint: DELETE /api/v1/auth/user/saves/items/image/:table_type?item_id=123
func (handler *Handler) deleteItemImage(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	tableType := c.Param("table_type")
	if tableType == "" {
		response.Error(c, 400, "table_type is required")
		return
	}

	itemID := c.Query("item_id")
	if itemID == "" {
		response.Error(c, 400, "item_id is required")
		return
	}

	err = handler.service.Delete(c.Request.Context(), user, itemID, tableType)
	if err != nil {
		slog.Error("Error deleting image from blob storage", "error", err)
		response.Error(c, 500, "failed to delete image")
		return
	}

	// Kept as a raw StandardResponse literal: response.OK hardcodes an empty
	// Message and has no parameter to carry this success text.
	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Image deleted successfully",
		Data:    gin.H{"item_id": itemID, "table_type": tableType},
	})
}

// Example endpoint: GET /api/v1/auth/user/saves/items/image/:table_type?item_id=123
func (handler *Handler) getItemImage(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	tableType := c.Param("table_type")
	if tableType == "" {
		response.Error(c, 400, "table_type is required")
		return
	}

	itemID := c.Query("item_id")
	if itemID == "" {
		response.Error(c, 400, "item_id is required")
		return
	}

	imageData, contentType, err := handler.service.Get(c.Request.Context(), user, itemID, tableType)
	if err != nil {
		slog.Error("Error retrieving image from blob storage", "error", err, "table_type", tableType, "item_id", itemID)
		response.Error(c, 404, "image not found")
		return
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Length", fmt.Sprintf("%d", len(imageData)))
	c.Header("Cache-Control", "public, max-age=3600")
	c.Data(200, contentType, imageData)
}
