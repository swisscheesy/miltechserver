package controller

import (
	"miltechserver/api/response"
	"miltechserver/api/service"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type EICController struct {
	EICService service.EICService
}

// NewEICController creates a new instance of EICController.
// \param eicService - the service to handle EIC operations.
// \return a pointer to the newly created EICController.
func NewEICController(eicService service.EICService) *EICController {
	return &EICController{EICService: eicService}
}

// LookupByNIIN handles the request to lookup EIC records by NIIN.
// \param c - the Gin context for the request.
func (controller *EICController) LookupByNIIN(c *gin.Context) {
	niin := c.Param("niin")

	if strings.TrimSpace(niin) == "" {
		c.JSON(400, gin.H{"error": "NIIN parameter is required"})
		return
	}

	consolidatedData, err := controller.EICService.LookupByNIIN(niin)

	if err != nil {
		if strings.Contains(err.Error(), "no EIC items found") {
			c.JSON(404, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(500, response.InternalErrorResponseMessage())
		}
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data: response.EICSearchResponse{
				Count: len(consolidatedData),
				Items: consolidatedData,
			},
		})
	}
}

// LookupByLIN handles the request to lookup EIC records by LIN.
// \param c - the Gin context for the request.
func (controller *EICController) LookupByLIN(c *gin.Context) {
	lin := c.Param("lin")

	if strings.TrimSpace(lin) == "" {
		c.JSON(400, gin.H{"error": "LIN parameter is required"})
		return
	}

	consolidatedData, err := controller.EICService.LookupByLIN(lin)

	if err != nil {
		if strings.Contains(err.Error(), "no EIC items found") {
			c.JSON(404, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(500, response.InternalErrorResponseMessage())
		}
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data: response.EICSearchResponse{
				Count: len(consolidatedData),
				Items: consolidatedData,
			},
		})
	}
}

// LookupByFSCPaginated handles the request to lookup EIC records by FSC with pagination.
// \param c - the Gin context for the request.
func (controller *EICController) LookupByFSCPaginated(c *gin.Context) {
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

	eicData, err := controller.EICService.LookupByFSCPaginated(fsc, page)

	if err != nil {
		if strings.Contains(err.Error(), "no EIC items found") {
			c.JSON(404, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(500, response.InternalErrorResponseMessage())
		}
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    eicData,
		})
	}
}

// LookupAllPaginated handles the request to lookup all EIC records with optional search and pagination.
// \param c - the Gin context for the request.
func (controller *EICController) LookupAllPaginated(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	search := c.Query("search")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(400, gin.H{"error": "Invalid page number"})
		return
	}

	eicData, err := controller.EICService.LookupAllPaginated(page, search)

	if err != nil {
		if strings.Contains(err.Error(), "no EIC items found") {
			c.JSON(404, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(500, response.InternalErrorResponseMessage())
		}
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    eicData,
		})
	}
}
