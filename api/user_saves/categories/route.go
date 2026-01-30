package categories

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

	router.GET("/user/saves/item_category", handler.getItemCategoriesByUser)
	router.PUT("/user/saves/item_category", handler.upsertItemCategoryByUser)
	router.DELETE("/user/saves/item_category", handler.deleteItemCategory)
	router.DELETE("/user/saves/item_category/all", handler.deleteAllItemCategories)
}

func (handler *Handler) getItemCategoriesByUser(c *gin.Context) {
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

func (handler *Handler) upsertItemCategoryByUser(c *gin.Context) {
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

	err = handler.service.Upsert(user, itemCategory)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(200)
}

func (handler *Handler) deleteItemCategory(c *gin.Context) {
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

	err = handler.service.Delete(user, itemCategory)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(200)
}

func (handler *Handler) deleteAllItemCategories(c *gin.Context) {
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
