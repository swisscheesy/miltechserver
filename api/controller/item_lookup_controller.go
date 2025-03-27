package controller

import (
	"github.com/gin-gonic/gin"
	"miltechserver/api/response"
	"miltechserver/api/service"
	"strconv"
	"strings"
)

type ItemLookupController struct {
	ItemLookupService service.ItemLookupService
}

// NewItemLookupController creates a new instance of ItemLookupController.
// \param itemLookupService - the service to handle item lookup operations.
// \return a pointer to the newly created ItemLookupController.
func NewItemLookupController(itemLookupService service.ItemLookupService) *ItemLookupController {
	return &ItemLookupController{ItemLookupService: itemLookupService}
}

// LookupLINByPage handles the request to lookup LIN by page.
// \param c - the Gin context for the request.
func (controller *ItemLookupController) LookupLINByPage(c *gin.Context) {
	pageStr := c.Query("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid page number"})
		return
	}

	linData, err := controller.ItemLookupService.LookupLINByPage(page)

	if err != nil {
		if strings.Contains(err.Error(), "no item") {
			c.JSON(404, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(500, response.InternalErrorResponseMessage())
		}

	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    linData,
		})
	}

}

// LookupLINByNIIN handles the request to lookup LIN by NIIN.
// \param c - the Gin context for the request.
func (controller *ItemLookupController) LookupLINByNIIN(c *gin.Context) {
	niin := c.Param("niin")

	linData, err := controller.ItemLookupService.LookupLINByNIIN(niin)

	if err != nil {
		if strings.Contains(err.Error(), "no item") {
			c.JSON(404, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(500, response.InternalErrorResponseMessage())
		}
	} else {
		retCount := len(linData)

		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data: response.LinSearchResponse{
				Count:      retCount,
				Lins:       linData,
				TotalPages: 1,
				Page:       1,
				IsLastPage: true,
			},
		})
	}

}

// LookupNIINByLIN handles the request to lookup NIIN by LIN.
// \param c - the Gin context for the request.
func (controller *ItemLookupController) LookupNIINByLIN(c *gin.Context) {
	lin := c.Param("lin")

	niinData, err := controller.ItemLookupService.LookupNIINByLIN(lin)

	if err != nil {
		if strings.Contains(err.Error(), "no item") {
			c.JSON(404, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(500, response.InternalErrorResponseMessage())
		}
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data: response.LinSearchResponse{
				Count:      len(niinData),
				Lins:       niinData,
				TotalPages: 1,
				Page:       1,
				IsLastPage: true,
			},
		})
	}

}

// LookupUOCByPage handles the request to lookup UOC by page.
// \param c - the Gin context for the request.
func (controller *ItemLookupController) LookupUOCByPage(c *gin.Context) {
	pageStr := c.Query("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid page number"})
		return
	} else {
		uocData, err := controller.ItemLookupService.LookupUOCByPage(page)

		if err != nil {
			if strings.Contains(err.Error(), "no item") {
				c.JSON(404, response.NoItemFoundResponseMessage())
			} else {
				c.JSON(500, response.InternalErrorResponseMessage())
			}
		} else {
			c.JSON(200, response.StandardResponse{
				Status:  200,
				Message: "",
				Data:    uocData,
			})
		}

	}

}

// LookupSpecificUOC handles the request to look up a specific UOC.
// \param c - the Gin context for the request.
func (controller *ItemLookupController) LookupSpecificUOC(c *gin.Context) {
	uoc := c.Param("uoc")

	uocData, err := controller.ItemLookupService.LookupSpecificUOC(uoc)

	if err != nil {
		if strings.Contains(err.Error(), "no item") {
			c.JSON(404, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(500, response.InternalErrorResponseMessage())
		}
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data: response.UOCPLookupResponse{
				Count: len(uocData),
				UOCs:  uocData,
			},
		})
	}

}

// LookupUOCByModel handles the request to lookup UOC by model.
// \param c - the Gin context for the request.
func (controller *ItemLookupController) LookupUOCByModel(c *gin.Context) {
	model := c.Param("model")

	uocData, err := controller.ItemLookupService.LookupUOCByModel(model)

	if err != nil {
		if strings.Contains(err.Error(), "no item") {
			c.JSON(404, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(500, response.InternalErrorResponseMessage())
		}
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data: response.UOCPLookupResponse{
				Count: len(uocData),
				UOCs:  uocData,
			},
		})
	}

}
