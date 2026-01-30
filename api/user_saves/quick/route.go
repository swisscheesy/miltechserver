package quick

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

	router.GET("/user/saves/quick_items", handler.getQuickSaveItemsByUser)
	router.PUT("/user/saves/quick_items/add", handler.upsertQuickSaveItemByUser)
	router.PUT("/user/saves/quick_items/addlist", handler.upsertQuickSaveItemListByUser)
	router.DELETE("/user/saves/quick_items", handler.deleteQuickSaveItemByUser)
	router.DELETE("/user/saves/quick_items/all", handler.deleteAllQuickSaveItemsByUser)
}

func (handler *Handler) getQuickSaveItemsByUser(c *gin.Context) {
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

func (handler *Handler) upsertQuickSaveItemByUser(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
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

	err = handler.service.Upsert(user, quick)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(200)
}

func (handler *Handler) deleteQuickSaveItemByUser(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var quick model.UserItemsQuick
	if err := c.BindJSON(&quick); err != nil {
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err = handler.service.Delete(user, quick)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(200)
}

func (handler *Handler) deleteAllQuickSaveItemsByUser(c *gin.Context) {
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

func (handler *Handler) upsertQuickSaveItemListByUser(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var quickItems []model.UserItemsQuick
	if err := c.BindJSON(&quickItems); err != nil {
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err = handler.service.UpsertBatch(user, quickItems)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(200)
}
