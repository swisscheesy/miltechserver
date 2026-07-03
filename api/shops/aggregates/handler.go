package aggregates

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

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

func (handler Handler) getShopSnapshot(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	options, err := parseShopSnapshotOptions(c)
	if err != nil {
		writeAggregateError(c, err)
		return
	}

	result, err := handler.service.GetShopSnapshot(c.Request.Context(), user, c.Param("shop_id"), options)
	if err != nil {
		writeAggregateError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Shop snapshot retrieved successfully",
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

func parseShopSnapshotOptions(c *gin.Context) (ShopSnapshotOptions, error) {
	includes, err := parseShopSnapshotIncludes(c)
	if err != nil {
		return ShopSnapshotOptions{}, err
	}
	messageLimit, err := parseOptionalIntQuery(c, "message_limit")
	if err != nil {
		return ShopSnapshotOptions{}, err
	}
	changesLimit, err := parseOptionalIntQuery(c, "changes_limit")
	if err != nil {
		return ShopSnapshotOptions{}, err
	}
	servicesLimit, err := parseOptionalIntQuery(c, "services_limit")
	if err != nil {
		return ShopSnapshotOptions{}, err
	}
	return ShopSnapshotOptions{
		Includes:      includes,
		MessageLimit:  messageLimit,
		ChangesLimit:  changesLimit,
		ServicesLimit: servicesLimit,
	}, nil
}

func parseShopSnapshotIncludes(c *gin.Context) (map[string]bool, error) {
	raw, exists := c.GetQuery("include")
	if !exists {
		return map[string]bool{
			"vehicles":      true,
			"lists":         true,
			"notifications": true,
			"services":      true,
		}, nil
	}

	allowed := map[string]bool{
		"vehicles":      true,
		"lists":         true,
		"notifications": true,
		"messages":      true,
		"services":      true,
		"changes":       true,
	}
	includes := make(map[string]bool)
	for _, value := range strings.Split(raw, ",") {
		include := strings.TrimSpace(value)
		if !allowed[include] {
			return nil, ErrInvalidInclude
		}
		includes[include] = true
	}
	return includes, nil
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
