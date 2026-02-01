package detailed

import (
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
	router.GET("/queries/items/detailed", handler.findDetailed)
}

func (handler *Handler) findDetailed(c *gin.Context) {
	ctx := c.Request.Context()
	niin := c.Query("niin")
	itemData, err := handler.service.FindDetailedItem(ctx, niin)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "",
		Data:    itemData,
	})
}
