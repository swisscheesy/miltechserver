package aggregates

import (
	"errors"
	"net/http"
	"strconv"

	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func getUser(c *gin.Context) (*bootstrap.User, bool) {
	ctxUser, ok := c.Get("user")
	user, userOK := ctxUser.(*bootstrap.User)
	return user, ok && userOK && user != nil
}

func (handler Handler) getListsWithItems(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	result, err := handler.service.GetListsWithItems(c.Request.Context(), user, c.Param("shop_id"))
	if err != nil {
		writeAggregateError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Shop lists with items retrieved successfully",
		Data:    result,
	})
}

func (handler Handler) getVehicleMaintenanceSnapshot(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	limits, err := parseSnapshotLimits(c)
	if err != nil {
		writeAggregateError(c, err)
		return
	}

	result, err := handler.service.GetVehicleMaintenanceSnapshot(c.Request.Context(), user, c.Param("vehicle_id"), limits)
	if err != nil {
		writeAggregateError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Vehicle maintenance snapshot retrieved successfully",
		Data:    result,
	})
}

func parseSnapshotLimits(c *gin.Context) (SnapshotLimits, error) {
	servicesLimit, err := parseOptionalIntQuery(c, "services_limit")
	if err != nil {
		return SnapshotLimits{}, err
	}
	changesLimit, err := parseOptionalIntQuery(c, "changes_limit")
	if err != nil {
		return SnapshotLimits{}, err
	}
	return SnapshotLimits{
		ServicesLimit: servicesLimit,
		ChangesLimit:  changesLimit,
	}, nil
}

func parseOptionalIntQuery(c *gin.Context, key string) (int, error) {
	raw, exists := c.GetQuery(key)
	if !exists {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, ErrInvalidLimit
	}
	return value, nil
}

func writeAggregateError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
	case errors.Is(err, ErrAccessDenied):
		c.JSON(http.StatusForbidden, gin.H{"message": "access denied"})
	case errors.Is(err, ErrInvalidLimit), errors.Is(err, ErrInvalidInclude):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, response.StandardResponse{
			Status:  http.StatusInternalServerError,
			Message: ErrAggregateUnavailable.Error(),
			Data:    nil,
		})
	}
}
