package eic

import (
	"database/sql"
	"errors"
	"strings"

	"miltechserver/api/response"
	"miltechserver/api/shared/pagination"

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
		response.Error(c, 400, "NIIN parameter is required")
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

	response.OK(c, response.EICSearchResponse{
		Count: len(consolidatedData),
		Items: consolidatedData,
	})
}

func (handler *Handler) lookupByLIN(c *gin.Context) {
	lin := c.Param("lin")

	if strings.TrimSpace(lin) == "" {
		response.Error(c, 400, "LIN parameter is required")
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

	response.OK(c, response.EICSearchResponse{
		Count: len(consolidatedData),
		Items: consolidatedData,
	})
}

func (handler *Handler) lookupByFSCPaginated(c *gin.Context) {
	fsc := c.Param("fsc")

	if strings.TrimSpace(fsc) == "" {
		response.Error(c, 400, "FSC parameter is required")
		return
	}

	page, ok := pagination.ParsePage(c)
	if !ok {
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

	response.OK(c, eicData)
}

func (handler *Handler) lookupAllPaginated(c *gin.Context) {
	search := c.Query("search")

	page, ok := pagination.ParsePage(c)
	if !ok {
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

	response.OK(c, eicData)
}
