package queries

import (
	"log/slog"
	"time"

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

	router.GET("/shops/:shop_id/equipment-services", handler.getByShop)
	router.GET("/shops/:shop_id/equipment/:equipment_id/services", handler.getByEquipment)
}

func (handler *Handler) getByShop(c *gin.Context) {
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

	var req request.GetEquipmentServicesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		slog.Info("invalid query parameters", "error", err)
		c.JSON(400, gin.H{"message": "invalid query parameters", "details": err.Error()})
		return
	}

	services, err := handler.service.GetByShop(user, shopID, req)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Services retrieved successfully",
		Data:    *services,
	})
}

func (handler *Handler) getByEquipment(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	equipmentID := c.Param("equipment_id")
	if equipmentID == "" {
		c.JSON(400, gin.H{"message": "equipment_id is required"})
		return
	}

	var req request.GetEquipmentServicesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		slog.Info("invalid query parameters", "error", err)
		c.JSON(400, gin.H{"message": "invalid query parameters", "details": err.Error()})
		return
	}

	var startDate, endDate *time.Time
	if req.StartDate != nil {
		parsed, err := time.Parse(time.RFC3339, *req.StartDate)
		if err != nil {
			c.JSON(400, gin.H{"message": "invalid start_date format", "details": err.Error()})
			return
		}
		startDate = &parsed
	}

	if req.EndDate != nil {
		parsed, err := time.Parse(time.RFC3339, *req.EndDate)
		if err != nil {
			c.JSON(400, gin.H{"message": "invalid end_date format", "details": err.Error()})
			return
		}
		endDate = &parsed
	}

	services, err := handler.service.GetByEquipment(user, equipmentID, req.Limit, req.Offset, startDate, endDate)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Services retrieved successfully",
		Data:    *services,
	})
}
