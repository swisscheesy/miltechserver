package controller

import (
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

// Shop Vehicle Operations

// CreateShopVehicle creates a new vehicle for a shop
func (controller *ShopsController) CreateShopVehicle(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.CreateShopVehicleRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	vehicle := model.ShopVehicle{
		ShopID:  req.ShopID,
		Niin:    req.Niin,
		Admin:   req.Admin,
		Model:   req.Model,
		Serial:  req.Serial,
		Uoc:     req.Uoc,
		Mileage: req.Mileage,
		Hours:   req.Hours,
		Comment: req.Comment,
	}

	createdVehicle, err := controller.ShopsService.CreateShopVehicle(user, vehicle)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "Vehicle created successfully",
		Data:    *createdVehicle,
	})
}

// GetShopVehicles returns all vehicles for a shop
func (controller *ShopsController) GetShopVehicles(c *gin.Context) {
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

	vehicles, err := controller.ShopsService.GetShopVehicles(user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    vehicles,
	})
}

// GetShopVehicleByID returns a specific vehicle by ID
func (controller *ShopsController) GetShopVehicleByID(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	vehicleID := c.Param("vehicle_id")
	if vehicleID == "" {
		c.JSON(400, gin.H{"message": "vehicle_id is required"})
		return
	}

	vehicle, err := controller.ShopsService.GetShopVehicleByID(user, vehicleID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    *vehicle,
	})
}

// UpdateShopVehicle updates an existing shop vehicle
func (controller *ShopsController) UpdateShopVehicle(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.UpdateShopVehicleRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	vehicle := model.ShopVehicle{
		ID:             req.VehicleID,
		Admin:          req.Admin,
		Niin:           req.Niin,
		Model:          req.Model,
		Serial:         req.Serial,
		Uoc:            req.Uoc,
		Mileage:        req.Mileage,
		Hours:          req.Hours,
		Comment:        req.Comment,
		TrackedMileage: req.TrackedMileage,
		TrackedHours:   req.TrackedHours,
	}

	err := controller.ShopsService.UpdateShopVehicle(user, vehicle)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Vehicle updated successfully"})
}

// DeleteShopVehicle deletes a shop vehicle
func (controller *ShopsController) DeleteShopVehicle(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	vehicleID := c.Param("vehicle_id")
	if vehicleID == "" {
		c.JSON(400, gin.H{"message": "vehicle_id is required"})
		return
	}

	err := controller.ShopsService.DeleteShopVehicle(user, vehicleID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Vehicle deleted successfully"})
}
