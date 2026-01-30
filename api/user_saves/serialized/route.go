package serialized

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

	router.GET("/user/saves/serialized_items", handler.getSerializedItemsByUser)
	router.PUT("/user/saves/serialized_items/add", handler.upsertSerializedSaveItemByUser)
	router.PUT("/user/saves/serialized_items/addlist", handler.upsertSerializedSaveItemListByUser)
	router.DELETE("/user/saves/serialized_items", handler.deleteSerializedSaveItemByUser)
	router.DELETE("/user/saves/serialized_items/all", handler.deleteAllSerializedItemsByUser)
}

func (handler *Handler) getSerializedItemsByUser(c *gin.Context) {
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

func (handler *Handler) upsertSerializedSaveItemByUser(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var serializedItem model.UserItemsSerialized
	if err := c.BindJSON(&serializedItem); err != nil {
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err = handler.service.Upsert(user, serializedItem)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(200)
}

func (handler *Handler) deleteSerializedSaveItemByUser(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var serializedItem model.UserItemsSerialized
	if err := c.BindJSON(&serializedItem); err != nil {
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err = handler.service.Delete(user, serializedItem)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(200)
}

func (handler *Handler) deleteAllSerializedItemsByUser(c *gin.Context) {
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

func (handler *Handler) upsertSerializedSaveItemListByUser(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var serializedItems []model.UserItemsSerialized
	if err := c.BindJSON(&serializedItems); err != nil {
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err = handler.service.UpsertBatch(user, serializedItems)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(200)
}
