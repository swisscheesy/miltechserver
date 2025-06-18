package controller

import (
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/api/service"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

type UserSavesController struct {
	UserSavesService service.UserSavesService
}

func NewUserSavesController(userSavesService service.UserSavesService) *UserSavesController {
	return &UserSavesController{UserSavesService: userSavesService}
}

func (controller *UserSavesController) GetQuickSaveItemsByUser(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	result, err := controller.UserSavesService.GetQuickSaveItemsByUser(user)

	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    result,
		})
	}
}

func (controller *UserSavesController) UpsertQuickSaveItemByUser(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var quick model.UserItemsQuick

	if err := c.BindJSON(&quick); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.UserSavesService.UpsertQuickSaveItemByUser(user, quick)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserSavesController) DeleteQuickSaveItemByUser(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var quick model.UserItemsQuick
	if err := c.BindJSON(&quick); err != nil {
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.UserSavesService.DeleteQuickSaveItemByUser(user, quick)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserSavesController) DeleteAllQuickSaveItemsByUser(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	err := controller.UserSavesService.DeleteAllQuickSaveItemsByUser(user)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserSavesController) UpsertQuickSaveItemListByUser(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var quickItems []model.UserItemsQuick
	if err := c.BindJSON(&quickItems); err != nil {
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.UserSavesService.UpsertQuickSaveItemListByUser(user, quickItems)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserSavesController) GetSerializedItemsByUser(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	result, err := controller.UserSavesService.GetSerializedItemsByUser(user)

	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    result,
		})
	}
}

func (controller *UserSavesController) UpsertSerializedSaveItemByUser(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var serializedItem model.UserItemsSerialized
	if err := c.BindJSON(&serializedItem); err != nil {
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.UserSavesService.UpsertSerializedSaveItemByUser(user, serializedItem)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserSavesController) DeleteSerializedSaveItemByUser(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var serializedItem model.UserItemsSerialized
	if err := c.BindJSON(&serializedItem); err != nil {
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.UserSavesService.DeleteSerializedSaveItemByUser(user, serializedItem)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserSavesController) DeleteAllSerializedItemsByUser(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	err := controller.UserSavesService.DeleteAllSerializedItemsByUser(user)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserSavesController) UpsertSerializedSaveItemListByUser(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var serializedItems []model.UserItemsSerialized
	if err := c.BindJSON(&serializedItems); err != nil {
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.UserSavesService.UpsertSerializedSaveItemListByUser(user, serializedItems)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserSavesController) GetItemCategoriesByUser(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	result, err := controller.UserSavesService.GetItemCategoriesByUser(user)

	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    result,
		})
	}
}

func (controller *UserSavesController) UpsertItemCategoryByUser(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var itemCategory model.UserItemCategory
	if err := c.BindJSON(&itemCategory); err != nil {
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.UserSavesService.UpsertItemCategoryByUser(user, itemCategory)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserSavesController) DeleteItemCategory(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	var itemCategory model.UserItemCategory
	if err := c.BindJSON(&itemCategory); err != nil {
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	err := controller.UserSavesService.DeleteItemCategory(user, itemCategory)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserSavesController) DeleteAllItemCategories(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	err := controller.UserSavesService.DeleteAllItemCategories(user)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserSavesController) GetCategorizedItemsByCategory(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	var itemCategory model.UserItemCategory
	if err := c.BindJSON(&itemCategory); err != nil {
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	result, err := controller.UserSavesService.GetCategorizedItemsByCategory(user, itemCategory)

	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    result,
		})
	}
}

func (controller *UserSavesController) GetCategorizedItemsByUser(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	result, err := controller.UserSavesService.GetCategorizedItemsByUser(user)

	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    result,
		})
	}
}

func (controller *UserSavesController) DeleteCategorizedItemByCategoryId(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	var categorizedItem model.UserItemsCategorized
	if err := c.BindJSON(&categorizedItem); err != nil {
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	err := controller.UserSavesService.DeleteCategorizedItemByCategoryId(user, categorizedItem)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserSavesController) DeleteAllCategorizedItems(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	err := controller.UserSavesService.DeleteAllCategorizedItems(user)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserSavesController) UpsertCategorizedItemByUser(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	var categorizedItem model.UserItemsCategorized
	if err := c.BindJSON(&categorizedItem); err != nil {
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	err := controller.UserSavesService.UpsertCategorizedItemByUser(user, categorizedItem)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserSavesController) UpsertCategorizedItemListByUser(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	var categorizedItems []model.UserItemsCategorized
	if err := c.BindJSON(&categorizedItems); err != nil {
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	err := controller.UserSavesService.UpsertCategorizedItemListByUser(user, categorizedItems)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

// UploadItemImage handles image upload for items
// Example endpoint: POST /api/v1/auth/user/saves/items/image/upload/:table_type
func (controller *UserSavesController) UploadItemImage(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	// Get the table type from URL parameters
	tableType := c.Param("table_type")
	if tableType == "" {
		c.JSON(400, gin.H{"message": "table_type is required"})
		return
	}

	// Get the item ID from query parameters or form data
	itemID := c.Query("item_id")
	if itemID == "" {
		c.JSON(400, gin.H{"message": "item_id is required"})
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

	// Read file data
	imageData := make([]byte, header.Size)
	_, err = file.Read(imageData)
	if err != nil {
		slog.Error("Error reading file data", "error", err)
		c.JSON(500, gin.H{"message": "failed to read file data"})
		return
	}

	// Upload to blob storage
	blobURL, err := controller.UserSavesService.UploadItemImage(user, itemID, tableType, imageData)
	if err != nil {
		slog.Error("Error uploading image to blob storage", "error", err)
		c.JSON(500, gin.H{"message": "failed to upload image"})
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Image uploaded successfully",
		Data:    gin.H{"item_id": itemID, "table_type": tableType, "blob_url": blobURL},
	})
}

// DeleteItemImage handles image deletion for items
// Example endpoint: DELETE /api/v1/auth/user/saves/items/image/:table_type?item_id=123
func (controller *UserSavesController) DeleteItemImage(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	// Get the table type from URL parameters
	tableType := c.Param("table_type")
	if tableType == "" {
		c.JSON(400, gin.H{"message": "table_type is required"})
		return
	}

	// Get the item ID from query parameters
	itemID := c.Query("item_id")
	if itemID == "" {
		c.JSON(400, gin.H{"message": "item_id is required"})
		return
	}

	// Delete from blob storage
	err := controller.UserSavesService.DeleteItemImage(user, itemID, tableType)
	if err != nil {
		slog.Error("Error deleting image from blob storage", "error", err)
		c.JSON(500, gin.H{"message": "failed to delete image"})
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Image deleted successfully",
		Data:    gin.H{"item_id": itemID, "table_type": tableType},
	})
}

// GetItemImage handles image retrieval for items
// Example endpoint: GET /api/v1/auth/user/saves/items/image/:table_type?item_id=123
func (controller *UserSavesController) GetItemImage(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	// Get the table type from URL parameters
	tableType := c.Param("table_type")
	if tableType == "" {
		c.JSON(400, gin.H{"message": "table_type is required"})
		return
	}

	// Get the item ID from query parameters
	itemID := c.Query("item_id")
	if itemID == "" {
		c.JSON(400, gin.H{"message": "item_id is required"})
		return
	}

	// Get image from blob storage
	imageData, contentType, err := controller.UserSavesService.GetItemImage(user, itemID, tableType)
	if err != nil {
		slog.Error("Error retrieving image from blob storage", "error", err, "table_type", tableType, "item_id", itemID)
		c.JSON(404, gin.H{"message": "image not found"})
		return
	}

	// Set appropriate headers and return the image
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", fmt.Sprintf("%d", len(imageData)))
	c.Header("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	c.Data(200, contentType, imageData)
}
