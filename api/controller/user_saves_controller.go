package controller

import (
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
		slog.Info("Unauthorized request %s")
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
		slog.Info("Unauthorized request %s")
		return
	}

	var quick model.UserItemsQuick

	if err := c.BindJSON(&quick); err != nil {
		slog.Info("invalid request %s", err)
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
		slog.Info("Unauthorized request %s")
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
		slog.Info("Unauthorized request %s")
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
		slog.Info("Unauthorized request %s")
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
		slog.Info("Unauthorized request %s")
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
		slog.Info("Unauthorized request %s")
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
		slog.Info("Unauthorized request %s")
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
		slog.Info("Unauthorized request %s")
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
		slog.Info("Unauthorized request %s")
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
		slog.Info("Unauthorized request %s")
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
		slog.Info("Unauthorized request %s")
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
		slog.Info("Unauthorized request %s")
		return
	}

	err := controller.UserSavesService.DeleteItemCategoryByUuid(user, itemCategory)

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
		slog.Info("Unauthorized request %s")
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
		slog.Info("Unauthorized request %s")
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
		slog.Info("Unauthorized request %s")
		return
	}

	err := controller.UserSavesService.DeleteCategorizedItemByCategoryId(user, categorizedItem)

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
		slog.Info("Unauthorized request %s")
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
		slog.Info("Unauthorized request %s")
		return
	}

	err := controller.UserSavesService.UpsertCategorizedItemListByUser(user, categorizedItems)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}
