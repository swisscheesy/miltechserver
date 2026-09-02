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
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	result, err := handler.service.GetByUser(user)
	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
		return
	}

	response.OK(c, result)
}

func (handler *Handler) upsertItemCategoryByUser(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	var itemCategory model.UserItemCategory
	if err := c.BindJSON(&itemCategory); err != nil {
		response.Error(c, 400, "invalid request")
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
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	var itemCategory model.UserItemCategory
	if err := c.BindJSON(&itemCategory); err != nil {
		response.Error(c, 400, "invalid request")
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
		response.Error(c, 401, "unauthorized")
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
