package short

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"miltechserver/api/item_query/shared"
	"miltechserver/api/response"
)

type Handler struct {
	service Service
}

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	registerHandlers(router, service)
}

func registerHandlers(router *gin.RouterGroup, service Service) {
	handler := Handler{service: service}
	router.GET("/queries/items/initial", handler.findShort)
}

func (handler *Handler) findShort(c *gin.Context) {
	method := c.Query("method")
	value := c.Query("value")

	switch method {
	case "niin":
		result, err := handler.service.FindShortByNiin(value)
		if err != nil {
			if errors.Is(err, shared.ErrNoItemsFound) {
				c.JSON(http.StatusNotFound, response.EmptyResponseMessage())
				return
			}
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
			return
		}
		c.JSON(http.StatusOK, response.StandardResponse{
			Status:  http.StatusOK,
			Message: "",
			Data:    result,
		})
	case "part":
		result, err := handler.service.FindShortByPart(value)
		if err != nil {
			if errors.Is(err, shared.ErrNoItemsFound) {
				c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
				return
			}
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
			return
		}
		c.JSON(http.StatusOK, response.StandardResponse{
			Status:  http.StatusOK,
			Message: "",
			Data:    result,
		})
	}
}
