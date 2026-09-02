package vehicles

import (
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/api/user_vehicles/shared"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	handler := Handler{service: service}

	router.GET("/user/vehicles", handler.getByUser)
	router.GET("/user/vehicles/:vehicleId", handler.getByID)
	router.PUT("/user/vehicles", handler.upsert)
	router.DELETE("/user/vehicles/:vehicleId", handler.delete)
	router.DELETE("/user/vehicles", handler.deleteAll)
}

func (handler *Handler) getByUser(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	result, err := handler.service.GetByUser(c.Request.Context(), user)
	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
		return
	}

	response.OK(c, result)
}

func (handler *Handler) getByID(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	vehicleID := c.Param("vehicleId")
	if vehicleID == "" {
		response.Error(c, 400, "vehicle ID is required")
		return
	}

	result, err := handler.service.GetByID(c.Request.Context(), user, vehicleID)
	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
		return
	}

	response.OK(c, result)
}

func (handler *Handler) upsert(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	var vehicle model.UserVehicle
	if err := c.BindJSON(&vehicle); err != nil {
		slog.Info("invalid request", "error", err)
		response.Error(c, 400, "invalid request")
		return
	}

	err = handler.service.Upsert(c.Request.Context(), user, vehicle)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(200)
}

func (handler *Handler) delete(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	vehicleID := c.Param("vehicleId")
	if vehicleID == "" {
		response.Error(c, 400, "vehicle ID is required")
		return
	}

	err = handler.service.Delete(c.Request.Context(), user, vehicleID)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(200)
}

func (handler *Handler) deleteAll(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	err = handler.service.DeleteAll(c.Request.Context(), user)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(200)
}
