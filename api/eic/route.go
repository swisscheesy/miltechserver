package eic

import (
	"database/sql"
	"errors"
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

func RegisterHandlers(router *gin.RouterGroup, service Service) {
	handler := Handler{service: service}

	router.GET("/eic/niin/:niin", handler.lookupByNIIN)
	router.GET("/eic/lin/:lin", handler.lookupByLIN)
	router.GET("/eic/fsc/:fsc", handler.lookupByFSCPaginated)
	router.GET("/eic/items", handler.lookupAllPaginated)
}

func (handler *Handler) lookupByNIIN(c *gin.Context) {
	niin := c.Param("niin")

	if strings.TrimSpace(niin) == "" {
		c.JSON(400, gin.H{"error": "NIIN parameter is required"})
		return
	}

	consolidatedData, err := handler.service.LookupByNIIN(niin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(404, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(500, response.InternalErrorResponseMessage())
		}
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data: response.EICSearchResponse{
			Count: len(consolidatedData),
			Items: consolidatedData,
		},
	})
}

func (handler *Handler) lookupByLIN(c *gin.Context) {
	lin := c.Param("lin")

	if strings.TrimSpace(lin) == "" {
		c.JSON(400, gin.H{"error": "LIN parameter is required"})
		return
	}

	consolidatedData, err := handler.service.LookupByLIN(lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(404, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(500, response.InternalErrorResponseMessage())
		}
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data: response.EICSearchResponse{
			Count: len(consolidatedData),
			Items: consolidatedData,
		},
	})
}

func (handler *Handler) lookupByFSCPaginated(c *gin.Context) {
	fsc := c.Param("fsc")
	pageStr := c.DefaultQuery("page", "1")

	if strings.TrimSpace(fsc) == "" {
		c.JSON(400, gin.H{"error": "FSC parameter is required"})
		return
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(400, gin.H{"error": "Invalid page number"})
		return
	}

	eicData, err := handler.service.LookupByFSCPaginated(fsc, page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(404, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(500, response.InternalErrorResponseMessage())
		}
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    eicData,
	})
}

func (handler *Handler) lookupAllPaginated(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	search := c.Query("search")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(400, gin.H{"error": "Invalid page number"})
		return
	}

	eicData, err := handler.service.LookupAllPaginated(page, search)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(404, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(500, response.InternalErrorResponseMessage())
		}
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    eicData,
	})
}
