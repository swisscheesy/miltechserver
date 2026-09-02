package core

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

	router.POST("/shops/:shop_id/equipment-services", handler.create)
	router.GET("/shops/:shop_id/equipment-services/:service_id", handler.getByID)
	router.PUT("/shops/:shop_id/equipment-services/:service_id", handler.update)
	router.DELETE("/shops/:shop_id/equipment-services/:service_id", handler.delete)
}

func (handler *Handler) create(c *gin.Context) {
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

	var req request.CreateEquipmentServiceRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		// Kept as a raw gin.H{} response: flat multi-field body ("message" +
		// "details") that response.Error()'s single message string cannot represent.
		c.JSON(400, gin.H{"message": "invalid request", "details": err.Error()})
		return
	}

	createdService, err := handler.service.Create(c.Request.Context(), user, req)
	if err != nil {
		c.Error(err)
		return
	}

	// Kept as a raw StandardResponse literal: response.OK hardcodes an empty
	// Message and has no parameter to carry this success text.
	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "Equipment service created successfully",
		Data:    *createdService,
	})
}

func (handler *Handler) getByID(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	shopID := c.Param("shop_id")
	serviceID := c.Param("service_id")

	if shopID == "" {
		response.Error(c, 400, "shop_id is required")
		return
	}

	if serviceID == "" {
		response.Error(c, 400, "service_id is required")
		return
	}

	service, err := handler.service.GetByID(c.Request.Context(), user, shopID, serviceID)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, *service)
}

func (handler *Handler) update(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	shopID := c.Param("shop_id")
	serviceID := c.Param("service_id")

	if shopID == "" {
		response.Error(c, 400, "shop_id is required")
		return
	}

	if serviceID == "" {
		response.Error(c, 400, "service_id is required")
		return
	}

	var req request.UpdateEquipmentServiceRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		// Kept as a raw gin.H{} response: flat multi-field body ("message" +
		// "details") that response.Error()'s single message string cannot represent.
		c.JSON(400, gin.H{"message": "invalid request", "details": err.Error()})
		return
	}

	req.ServiceID = serviceID

	updatedService, err := handler.service.Update(c.Request.Context(), user, shopID, req)
	if err != nil {
		c.Error(err)
		return
	}

	// Kept as a raw StandardResponse literal: response.OK hardcodes an empty
	// Message and has no parameter to carry this success text.
	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Equipment service updated successfully",
		Data:    *updatedService,
	})
}

func (handler *Handler) delete(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	shopID := c.Param("shop_id")
	serviceID := c.Param("service_id")

	if shopID == "" {
		response.Error(c, 400, "shop_id is required")
		return
	}

	if serviceID == "" {
		response.Error(c, 400, "service_id is required")
		return
	}

	err = handler.service.Delete(c.Request.Context(), user, shopID, serviceID)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, gin.H{"message": "Equipment service deleted successfully"})
}
