package items

import (
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/api/user_saves/shared"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	handler := Handler{service: service}

	router.GET("/user/saves/categorized_items/category", handler.getCategorizedItemsByCategory)
	router.GET("/user/saves/categorized_items", handler.getCategorizedItemsByUser)
	router.PUT("/user/saves/categorized_items/add", handler.upsertCategorizedItemByUser)
	router.PUT("/user/saves/categorized_items/addlist", handler.upsertCategorizedItemListByUser)
	router.DELETE("/user/saves/categorized_items", handler.deleteCategorizedItemByCategoryId)
	router.DELETE("/user/saves/categorized_items/all", handler.deleteAllCategorizedItems)
}

func (handler *Handler) getCategorizedItemsByCategory(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var itemCategory model.UserItemCategory
	if err := c.BindJSON(&itemCategory); err != nil {
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	result, err := handler.service.GetByCategory(user, itemCategory)
	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    result,
	})
}

func (handler *Handler) getCategorizedItemsByUser(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	result, err := handler.service.GetByUser(user)
	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    result,
	})
}

func (handler *Handler) deleteCategorizedItemByCategoryId(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var categorizedItem model.UserItemsCategorized
	if err := c.BindJSON(&categorizedItem); err != nil {
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err = handler.service.Delete(user, categorizedItem)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(200)
}

func (handler *Handler) deleteAllCategorizedItems(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	err = handler.service.DeleteAll(user)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(200)
}

func (handler *Handler) upsertCategorizedItemByUser(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var categorizedItem model.UserItemsCategorized
	if err := c.BindJSON(&categorizedItem); err != nil {
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err = handler.service.Upsert(user, categorizedItem)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(200)
}

func (handler *Handler) upsertCategorizedItemListByUser(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var categorizedItems []model.UserItemsCategorized
	if err := c.BindJSON(&categorizedItems); err != nil {
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err = handler.service.UpsertBatch(user, categorizedItems)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(200)
}
