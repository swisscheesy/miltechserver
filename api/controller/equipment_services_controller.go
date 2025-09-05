package controller

import (
	"log/slog"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/api/service"
	"miltechserver/bootstrap"
	"time"

	"github.com/gin-gonic/gin"
)

type EquipmentServicesController struct {
	EquipmentServicesService service.EquipmentServicesService
}

func NewEquipmentServicesController(equipmentServicesService service.EquipmentServicesService) *EquipmentServicesController {
	return &EquipmentServicesController{EquipmentServicesService: equipmentServicesService}
}

// CreateEquipmentService creates a new equipment service
func (controller *EquipmentServicesController) CreateEquipmentService(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
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

	createdService, err := controller.EquipmentServicesService.CreateEquipmentService(user, req)
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

// GetEquipmentServices retrieves equipment services with filtering and pagination
func (controller *EquipmentServicesController) GetEquipmentServices(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
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

	services, err := controller.EquipmentServicesService.GetEquipmentServices(user, shopID, req)
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

// GetEquipmentServiceByID retrieves a specific equipment service by ID
func (controller *EquipmentServicesController) GetEquipmentServiceByID(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
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

	service, err := controller.EquipmentServicesService.GetEquipmentServiceByID(user, shopID, serviceID)
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

// UpdateEquipmentService updates an existing equipment service
func (controller *EquipmentServicesController) UpdateEquipmentService(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
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

	// Set the service ID from URL parameter
	req.ServiceID = serviceID

	updatedService, err := controller.EquipmentServicesService.UpdateEquipmentService(user, shopID, req)
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

// DeleteEquipmentService deletes an equipment service
func (controller *EquipmentServicesController) DeleteEquipmentService(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
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

	err := controller.EquipmentServicesService.DeleteEquipmentService(user, shopID, serviceID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Equipment service deleted successfully"})
}

// GetServicesByEquipment retrieves services for a specific equipment
func (controller *EquipmentServicesController) GetServicesByEquipment(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
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

	// Parse date filters if provided
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

	services, err := controller.EquipmentServicesService.GetServicesByEquipment(user, equipmentID, req.Limit, req.Offset, startDate, endDate)
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

// GetServicesInDateRange retrieves services within a specific date range for calendar view
func (controller *EquipmentServicesController) GetServicesInDateRange(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	shopID := c.Param("shop_id")
	if shopID == "" {
		c.JSON(400, gin.H{"message": "shop_id is required"})
		return
	}

	var req request.GetCalendarServicesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		slog.Info("invalid query parameters", "error", err)
		c.JSON(400, gin.H{"message": "invalid query parameters", "details": err.Error()})
		return
	}

	services, err := controller.EquipmentServicesService.GetServicesInDateRange(user, shopID, req)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Calendar services retrieved successfully",
		Data:    *services,
	})
}

// GetOverdueServices retrieves services that are overdue
func (controller *EquipmentServicesController) GetOverdueServices(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
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

	services, err := controller.EquipmentServicesService.GetOverdueServices(user, shopID, req)
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

// GetServicesDueSoon retrieves services that are due soon
func (controller *EquipmentServicesController) GetServicesDueSoon(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
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

	services, err := controller.EquipmentServicesService.GetServicesDueSoon(user, shopID, req)
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

// CompleteEquipmentService marks a service as completed
func (controller *EquipmentServicesController) CompleteEquipmentService(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
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

	completedService, err := controller.EquipmentServicesService.CompleteEquipmentService(user, shopID, serviceID, req)
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
