package status

import (
	"log/slog"

	"miltechserver/api/equipment_services/shared"
	"miltechserver/api/request"
	"miltechserver/api/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	handler := Handler{service: service}

	router.GET("/shops/:shop_id/equipment-services/overdue", handler.getOverdue)
	router.GET("/shops/:shop_id/equipment-services/due-soon", handler.getDueSoon)
}

func (handler *Handler) getOverdue(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	shopID := c.Param("shop_id")
	if shopID == "" {
		c.JSON(400, gin.H{"message": "shop_id is required"})
		return
	}

	var req request.GetOverdueServicesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		slog.Info("invalid query parameters", "error", err)
		c.JSON(400, gin.H{"message": "invalid query parameters", "details": err.Error()})
		return
	}

	services, err := handler.service.GetOverdue(user, shopID, req)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Overdue services retrieved successfully",
		Data:    *services,
	})
}

func (handler *Handler) getDueSoon(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	shopID := c.Param("shop_id")
	if shopID == "" {
		c.JSON(400, gin.H{"message": "shop_id is required"})
		return
	}

	var req request.GetDueSoonServicesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		slog.Info("invalid query parameters", "error", err)
		c.JSON(400, gin.H{"message": "invalid query parameters", "details": err.Error()})
		return
	}

	services, err := handler.service.GetDueSoon(user, shopID, req)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Due soon services retrieved successfully",
		Data:    *services,
	})
}
