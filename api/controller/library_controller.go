package controller

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"miltechserver/api/service"
)

type LibraryController struct {
	LibraryService service.LibraryService
}

func NewLibraryController(libraryService service.LibraryService) *LibraryController {
	return &LibraryController{
		LibraryService: libraryService,
	}
}

// GetPMCSVehicles returns a list of all available PMCS vehicle folders
// GET /api/v1/auth/library/pmcs/vehicles
func (controller *LibraryController) GetPMCSVehicles(c *gin.Context) {
	slog.Info("GetPMCSVehicles endpoint called")

	vehicles, err := controller.LibraryService.GetPMCSVehicles()
	if err != nil {
		slog.Error("Failed to retrieve PMCS vehicles", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve PMCS vehicles",
			"details": err.Error(),
		})
		return
	}

	slog.Info("Successfully retrieved PMCS vehicles", "count", vehicles.Count)
	c.JSON(http.StatusOK, vehicles)
}
