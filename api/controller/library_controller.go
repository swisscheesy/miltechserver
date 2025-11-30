package controller

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"miltechserver/api/response"
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
// GET /api/v1/library/pmcs/vehicles
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
	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: vehicles})
}

// GetPMCSDocuments returns a list of all PDF documents for a specific vehicle
// GET /api/v1/library/pmcs/:vehicle/documents
func (controller *LibraryController) GetPMCSDocuments(c *gin.Context) {
	vehicleName := c.Param("vehicle")

	slog.Info("GetPMCSDocuments endpoint called", "vehicle", vehicleName)

	// Validate vehicle name
	if vehicleName == "" {
		slog.Warn("GetPMCSDocuments called with empty vehicle name")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Vehicle name is required",
		})
		return
	}

	documents, err := controller.LibraryService.GetPMCSDocuments(vehicleName)
	if err != nil {
		slog.Error("Failed to retrieve PMCS documents",
			"error", err,
			"vehicle", vehicleName)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve PMCS documents",
			"details": err.Error(),
		})
		return
	}

	slog.Info("Successfully retrieved PMCS documents",
		"count", documents.Count,
		"vehicle", vehicleName)

	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: documents})
}

// GenerateDownloadURL returns a time-limited SAS URL for downloading a document
// GET /api/v1/library/download?blob_path=pmcs/TRACK/file.pdf
func (controller *LibraryController) GenerateDownloadURL(c *gin.Context) {
	blobPath := c.Query("blob_path")

	slog.Info("GenerateDownloadURL endpoint called", "blobPath", blobPath)

	// Validate blob path parameter
	if blobPath == "" {
		slog.Warn("GenerateDownloadURL called with empty blob_path")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "blob_path query parameter is required",
		})
		return
	}

	downloadURLResp, err := controller.LibraryService.GenerateDownloadURL(blobPath)
	if err != nil {
		// Check if it's a not found error
		if strings.Contains(err.Error(), "document not found") {
			slog.Warn("Document not found for download",
				"blobPath", blobPath,
				"error", err)
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Document not found",
				"details": "The requested document does not exist or is not accessible",
			})
			return
		}

		// Check if it's a validation error
		if strings.Contains(err.Error(), "invalid") {
			slog.Warn("Invalid blob path for download",
				"blobPath", blobPath,
				"error", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request",
				"details": err.Error(),
			})
			return
		}

		// Generic server error
		slog.Error("Failed to generate download URL",
			"error", err,
			"blobPath", blobPath)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate download URL",
			"details": err.Error(),
		})
		return
	}

	slog.Info("Successfully generated download URL",
		"blobPath", blobPath,
		"expiresAt", downloadURLResp.ExpiresAt)

	c.JSON(http.StatusOK, downloadURLResp)
}
