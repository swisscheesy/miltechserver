package completion

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

	router.POST("/shops/:shop_id/equipment-services/:service_id/complete", handler.complete)
}

func (handler *Handler) complete(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	shopID := c.Param("shop_id")
	serviceID := c.Param("service_id")

	if shopID == "" {
		c.JSON(400, gin.H{"message": "shop_id is required"})
		return
	}

	if serviceID == "" {
		c.JSON(400, gin.H{"message": "service_id is required"})
		return
	}

	var req request.CompleteEquipmentServiceRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request", "details": err.Error()})
		return
	}

	completedService, err := handler.service.Complete(user, shopID, serviceID, req)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Equipment service completed successfully",
		Data:    *completedService,
	})
}
