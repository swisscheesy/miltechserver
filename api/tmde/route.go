package tmde

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"miltechserver/api/response"

	"github.com/gin-gonic/gin"
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
	RegisterHandlers(router, svc)
}

func RegisterHandlers(router *gin.RouterGroup, svc Service) {
	handler := Handler{service: svc}
	router.GET("/tmde/niin/:niin", handler.lookupByNIIN)
	router.GET("/tmde/requirements", handler.listAllPaginated)
}

func (h *Handler) lookupByNIIN(c *gin.Context) {
	niin := c.Param("niin")

	if strings.TrimSpace(niin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "NIIN parameter is required"})
		return
	}

	item, err := h.service.LookupByNIIN(niin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "",
		Data:    item,
	})
}

func (h *Handler) listAllPaginated(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}

	data, err := h.service.GetAllPaginated(page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "",
		Data:    data,
	})
}
