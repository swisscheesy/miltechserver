package queries

import (
	"log/slog"
	"time"

	"miltechserver/api/equipment_services/shared"
	"miltechserver/api/request"
	"miltechserver/api/response"
	shopsContract "miltechserver/api/shops/shared"

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
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	shopID := c.Param("shop_id")
	if shopID == "" {
		response.Error(c, 400, "shop_id is required")
		return
	}

	var req request.GetEquipmentServicesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		slog.Info("invalid query parameters", "error", err)
		// Kept as a raw gin.H{} response: flat multi-field body ("message" +
		// "details") that response.Error()'s single message string cannot represent.
		shopsContract.WriteValidationError(c, "invalid query parameters")
		return
	}

	services, err := handler.service.GetByShop(c.Request.Context(), user, shopID, req)
	if err != nil {
		c.Error(err)
		return
	}

	// Kept as a raw StandardResponse literal: response.OK hardcodes an empty
	// Message and has no parameter to carry this success text.
	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Services retrieved successfully",
		Data:    *services,
	})
}

func (handler *Handler) getByEquipment(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	equipmentID := c.Param("equipment_id")
	if equipmentID == "" {
		response.Error(c, 400, "equipment_id is required")
		return
	}

	var req request.GetEquipmentServicesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		slog.Info("invalid query parameters", "error", err)
		// Kept as a raw gin.H{} response: flat multi-field body ("message" +
		// "details") that response.Error()'s single message string cannot represent.
		shopsContract.WriteValidationError(c, "invalid query parameters")
		return
	}

	var startDate, endDate *time.Time
	if req.StartDate != nil {
		parsed, err := time.Parse(time.RFC3339, *req.StartDate)
		if err != nil {
			// Kept as a raw gin.H{} response: flat multi-field body ("message" +
			// "details") that response.Error()'s single message string cannot represent.
			shopsContract.WriteValidationError(c, "invalid start_date format")
			return
		}
		startDate = &parsed
	}

	if req.EndDate != nil {
		parsed, err := time.Parse(time.RFC3339, *req.EndDate)
		if err != nil {
			// Kept as a raw gin.H{} response: flat multi-field body ("message" +
			// "details") that response.Error()'s single message string cannot represent.
			shopsContract.WriteValidationError(c, "invalid end_date format")
			return
		}
		endDate = &parsed
	}

	services, err := handler.service.GetByEquipment(c.Request.Context(), user, equipmentID, req.Limit, req.Offset, startDate, endDate)
	if err != nil {
		c.Error(err)
		return
	}

	// Kept as a raw StandardResponse literal: response.OK hardcodes an empty
	// Message and has no parameter to carry this success text.
	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Services retrieved successfully",
		Data:    *services,
	})
}
