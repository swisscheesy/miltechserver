package calendar

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

	router.GET("/shops/:shop_id/equipment-services/calendar", handler.getCalendar)
}

func (handler *Handler) getCalendar(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	shopID := c.Param("shop_id")
	if shopID == "" {
		response.Error(c, 400, "shop_id is required")
		return
	}

	var req request.GetCalendarServicesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		slog.Info("invalid query parameters", "error", err)
		// Kept as a raw gin.H{} response: flat multi-field body ("message" +
		// "details") that response.Error()'s single message string cannot represent.
		c.JSON(400, gin.H{"message": "invalid query parameters", "details": err.Error()})
		return
	}

	services, err := handler.service.GetCalendarServices(user, shopID, req)
	if err != nil {
		c.Error(err)
		return
	}

	// Kept as a raw StandardResponse literal: response.OK hardcodes an empty
	// Message and has no parameter to carry this success text.
	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Calendar services retrieved successfully",
		Data:    *services,
	})
}
