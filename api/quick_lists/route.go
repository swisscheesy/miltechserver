package quick_lists

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

	router.GET("/quick-lists/clothing", handler.queryQuickListClothing)
	router.GET("/quick-lists/wheels", handler.queryQuickListWheels)
	router.GET("/quick-lists/batteries", handler.queryQuickListBatteries)
}

func (handler *Handler) queryQuickListClothing(c *gin.Context) {
	clothingData, err := handler.service.GetQuickListClothing()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "",
		Data:    clothingData,
	})
}

func (handler *Handler) queryQuickListWheels(c *gin.Context) {
	wheelsData, err := handler.service.GetQuickListWheels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "",
		Data:    wheelsData,
	})
}

func (handler *Handler) queryQuickListBatteries(c *gin.Context) {
	batteriesData, err := handler.service.GetQuickListBatteries()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "",
		Data:    batteriesData,
	})
}
