package pol_products

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

	"miltechserver/api/response"
)

type Dependencies struct {
	DB *sql.DB
}

type Handler struct {
	service Service
}

func RegisterRoutes(deps Dependencies, router *gin.RouterGroup) {
	repo := NewRepository(deps.DB)
	svc := NewService(repo)
	registerHandlers(router, svc)
}

func registerHandlers(router *gin.RouterGroup, svc Service) {
	handler := Handler{service: svc}
	router.GET("/pol-products", handler.getPolProducts)
}

func (handler *Handler) getPolProducts(c *gin.Context) {
	data, err := handler.service.GetPolProducts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "",
		Data:    data,
	})
}
