package help

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

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
	router.GET("/queries/items/help", handler.findByCode)
}

func (handler *Handler) findByCode(c *gin.Context) {
	code := c.Query("code")
	result, err := handler.service.FindByCode(code)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCode):
			c.JSON(http.StatusBadRequest, gin.H{"message": "code is required"})
		case errors.Is(err, ErrHelpNotFound):
			c.JSON(http.StatusNotFound, response.EmptyResponseMessage())
		default:
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "",
		Data:    result,
	})
}
