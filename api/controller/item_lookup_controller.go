package controller

import (
	"github.com/gin-gonic/gin"
	response2 "miltechserver/api/response"
	"miltechserver/api/service"
	"strconv"
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

	linData, err := controller.ItemLookupService.LookupLINByPage(c, page)

	c.JSON(200, response2.StandardResponse{
		Status:  200,
		Message: "",
		Data:    linData,
	})
}

// LookupLINByNIIN handles the request to lookup LIN by NIIN.
// \param c - the Gin context for the request.
func (controller *ItemLookupController) LookupLINByNIIN(c *gin.Context) {
	niin := c.Param("niin")

	linData, err := controller.ItemLookupService.LookupLINByNIIN(c, niin)

	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	retCount := len(linData)

	c.JSON(200, response2.StandardResponse{
		Status:  200,
		Message: "",
		Data: response2.LinSearchResponse{
			Count: retCount,
			Lins:  linData,
		},
	})
}

// LookupNIINByLIN handles the request to lookup NIIN by LIN.
// \param c - the Gin context for the request.
func (controller *ItemLookupController) LookupNIINByLIN(c *gin.Context) {
	lin := c.Param("lin")

	niinData, err := controller.ItemLookupService.LookupNIINByLIN(c, lin)

	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, response2.StandardResponse{
		Status:  200,
		Message: "",
		Data: response2.NiinSearchResponse{
			Count: len(niinData),
			Niins: niinData,
		},
	})
}

// LookupUOCByPage handles the request to lookup UOC by page.
// \param c - the Gin context for the request.
func (controller *ItemLookupController) LookupUOCByPage(c *gin.Context) {
	pageStr := c.Query("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid page number"})
		return
	}

	uocData, err := controller.ItemLookupService.LookupUOCByPage(c, page)

	c.JSON(200, response2.StandardResponse{
		Status:  200,
		Message: "",
		Data:    uocData,
	})
}

// LookupSpecificUOC handles the request to lookup a specific UOC.
// \param c - the Gin context for the request.
func (controller *ItemLookupController) LookupSpecificUOC(c *gin.Context) {
	uoc := c.Param("uoc")

	uocData, err := controller.ItemLookupService.LookupSpecificUOC(c, uoc)

	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, response2.StandardResponse{
		Status:  200,
		Message: "",
		Data: response2.UOCPLookupResponse{
			Count: len(uocData),
			UOCs:  uocData,
		},
	})
}

// LookupUOCByModel handles the request to lookup UOC by model.
// \param c - the Gin context for the request.
func (controller *ItemLookupController) LookupUOCByModel(c *gin.Context) {
	model := c.Param("model")

	uocData, err := controller.ItemLookupService.LookupUOCByModel(c, model)

	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, response2.StandardResponse{
		Status:  200,
		Message: "",
		Data: response2.UOCPLookupResponse{
			Count: len(uocData),
			UOCs:  uocData,
		},
	})
}
