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
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	shopID := c.Param("shop_id")
	if shopID == "" {
		c.JSON(400, gin.H{"message": "shop_id is required"})
		return
	}

	var req request.CreateEquipmentServiceRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request", "details": err.Error()})
		return
	}

	createdService, err := handler.service.Create(user, req)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "Equipment service created successfully",
		Data:    *createdService,
	})
}

func (handler *Handler) getByID(c *gin.Context) {
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

	service, err := handler.service.GetByID(user, shopID, serviceID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    *service,
	})
}

func (handler *Handler) update(c *gin.Context) {
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

	var req request.UpdateEquipmentServiceRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request", "details": err.Error()})
		return
	}

	req.ServiceID = serviceID

	updatedService, err := handler.service.Update(user, shopID, req)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Equipment service updated successfully",
		Data:    *updatedService,
	})
}

func (handler *Handler) delete(c *gin.Context) {
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

	err = handler.service.Delete(user, shopID, serviceID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Equipment service deleted successfully"})
}
