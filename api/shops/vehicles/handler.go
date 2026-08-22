package vehicles

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func authenticatedUser(c *gin.Context) (*bootstrap.User, bool) {
	value, exists := c.Get("user")
	if !exists {
		writeUsageErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return nil, false
	}

	user, ok := value.(*bootstrap.User)
	if !ok || user == nil {
		writeUsageErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return nil, false
	}

	return user, true
}

func decodeStrictJSON(body io.Reader, destination interface{}) error {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}

	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain exactly one JSON value")
		}
		return err
	}

	return nil
}

func writeVehicleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidUsageAdjustment):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	case errors.Is(err, shared.ErrShopAccessDenied):
		c.JSON(http.StatusForbidden, gin.H{"message": err.Error()})
	case errors.Is(err, shared.ErrVehicleNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
	case errors.Is(err, ErrUsageOutOfRange):
		c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
	default:
		slog.Error("Shop vehicle usage adjustment failed", "error", err)
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
	}
}

func writeUsageAdjustmentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidUsageAdjustment):
		writeUsageErrorResponse(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, shared.ErrShopAccessDenied):
		writeUsageErrorResponse(c, http.StatusForbidden, "shop access denied")
	case errors.Is(err, shared.ErrVehicleNotFound):
		writeUsageErrorResponse(c, http.StatusNotFound, "shop vehicle not found")
	case errors.Is(err, ErrUsageOutOfRange):
		writeUsageErrorResponse(c, http.StatusConflict, ErrUsageOutOfRange.Error())
	default:
		slog.Error("Shop vehicle usage adjustment failed", "error", err)
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
	}
}

func writeUsageErrorResponse(c *gin.Context, status int, message string) {
	c.JSON(status, response.StandardResponse{
		Status:  status,
		Message: message,
		Data:    nil,
	})
}

// Shop Vehicle Operations

// CreateShopVehicle creates a new vehicle for a shop
func (handler *Handler) CreateShopVehicle(c *gin.Context) {
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

	service := handler.service
	createdVehicle, err := service.CreateShopVehicle(user, vehicle)
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
func (handler *Handler) GetShopVehicles(c *gin.Context) {
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

	service := handler.service
	vehicles, err := service.GetShopVehicles(user, shopID)
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
func (handler *Handler) GetShopVehicleByID(c *gin.Context) {
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

	service := handler.service
	vehicle, err := service.GetShopVehicleByID(user, vehicleID)
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
func (handler *Handler) UpdateShopVehicle(c *gin.Context) {
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

	service := handler.service
	err := service.UpdateShopVehicle(user, vehicle)
	if err != nil {
		if errors.Is(err, ErrInvalidUsageAdjustment) {
			writeVehicleError(c, err)
			return
		}
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Vehicle updated successfully"})
}

func (handler *Handler) AdjustShopVehicleUsage(c *gin.Context) {
	user, ok := authenticatedUser(c)
	if !ok {
		return
	}

	vehicleID := c.Param("vehicle_id")
	if vehicleID == "" {
		writeUsageAdjustmentError(c, fmt.Errorf("%w: vehicle_id is required", ErrInvalidUsageAdjustment))
		return
	}

	var req request.AdjustShopVehicleUsageRequest
	if err := decodeStrictJSON(c.Request.Body, &req); err != nil {
		writeUsageAdjustmentError(c, fmt.Errorf("%w: malformed request", ErrInvalidUsageAdjustment))
		return
	}

	updated, err := handler.service.AdjustShopVehicleUsage(
		c.Request.Context(),
		user,
		UsageAdjustment{
			VehicleID:         vehicleID,
			Operation:         UsageAdjustmentOperation(req.Operation),
			MileageAdjustment: req.MileageAdjustment,
			HoursAdjustment:   req.HoursAdjustment,
		},
	)
	if err != nil {
		writeUsageAdjustmentError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Equipment usage adjusted successfully",
		Data:    *updated,
	})
}

// DeleteShopVehicle deletes a shop vehicle
func (handler *Handler) DeleteShopVehicle(c *gin.Context) {
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

	service := handler.service
	err := service.DeleteShopVehicle(user, vehicleID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Vehicle deleted successfully"})
}
