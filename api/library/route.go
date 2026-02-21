package library

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"

	"miltechserver/api/analytics"
	"miltechserver/api/library/ps_mag"
	"miltechserver/api/middleware"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Dependencies struct {
	DB         *sql.DB
	BlobClient *azblob.Client
	Env        *bootstrap.Env
	Analytics  analytics.Service
}

type Handler struct {
	service Service
}

func RegisterRoutes(deps Dependencies, publicGroup, authGroup *gin.RouterGroup) {
	svc := NewService(deps.BlobClient, deps.Env, deps.Analytics)
	registerHandlers(publicGroup, authGroup, svc)
	ps_mag.RegisterHandlers(publicGroup, deps.BlobClient)
}

func registerHandlers(publicGroup, authGroup *gin.RouterGroup, svc Service) {
	handler := Handler{service: svc}

	publicGroup.GET("/library/pmcs/vehicles", handler.getPMCSVehicles)
	publicGroup.GET("/library/pmcs/:vehicle/documents", handler.getPMCSDocuments)
	// Rate-limited: each IP is allowed a burst of 10 requests, sustained at 2 req/s.
	publicGroup.GET("/library/download", middleware.RateLimiter(), handler.generateDownloadURL)

	// Future public routes:
	// publicGroup.GET("/library/bii/categories", handler.getBIICategories)
	// publicGroup.GET("/library/bii/:category/documents", handler.getBIIDocuments)

	// Future authenticated routes (downloads, favorites, etc.):
	// authGroup.POST("/library/favorites", handler.addFavorite)
	// authGroup.DELETE("/library/favorites/:document_path", handler.removeFavorite)
	// authGroup.GET("/library/favorites", handler.getUserFavorites)
	// authGroup.GET("/library/download/:path", handler.generateDownloadURL)
	_ = authGroup
}

// getPMCSVehicles returns a list of all available PMCS vehicle folders.
// GET /api/v1/library/pmcs/vehicles
func (handler *Handler) getPMCSVehicles(c *gin.Context) {
	slog.Info("GetPMCSVehicles endpoint called")

	vehicles, err := handler.service.GetPMCSVehicles()
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

// getPMCSDocuments returns a list of all PDF documents for a specific vehicle.
// GET /api/v1/library/pmcs/:vehicle/documents
func (handler *Handler) getPMCSDocuments(c *gin.Context) {
	vehicleName := c.Param("vehicle")

	slog.Info("GetPMCSDocuments endpoint called", "vehicle", vehicleName)

	if strings.TrimSpace(vehicleName) == "" {
		slog.Warn("GetPMCSDocuments called with empty vehicle name")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Vehicle name is required",
		})
		return
	}

	documents, err := handler.service.GetPMCSDocuments(vehicleName)
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

// generateDownloadURL returns a time-limited SAS URL for downloading a document.
// GET /api/v1/library/download?blob_path=pmcs/TRACK/file.pdf
func (handler *Handler) generateDownloadURL(c *gin.Context) {
	blobPath := c.Query("blob_path")

	slog.Info("GenerateDownloadURL endpoint called", "blobPath", blobPath)

	if strings.TrimSpace(blobPath) == "" {
		slog.Warn("GenerateDownloadURL called with empty blob_path")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "blob_path query parameter is required",
		})
		return
	}

	downloadURLResp, err := handler.service.GenerateDownloadURL(c.Request.Context(), blobPath)
	if err != nil {
		switch {
		case errors.Is(err, ErrDocumentNotFound):
			slog.Warn("Document not found for download",
				"blobPath", blobPath,
				"error", err)
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Document not found",
				"details": "The requested document does not exist or is not accessible",
			})
		case errors.Is(err, ErrInvalidBlobPath), errors.Is(err, ErrInvalidFileType), errors.Is(err, ErrEmptyBlobPath):
			slog.Warn("Invalid blob path for download",
				"blobPath", blobPath,
				"error", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request",
				"details": err.Error(),
			})
		default:
			slog.Error("Failed to generate download URL",
				"error", err,
				"blobPath", blobPath)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to generate download URL",
				"details": err.Error(),
			})
		}
		return
	}

	slog.Info("Successfully generated download URL",
		"blobPath", blobPath,
		"expiresAt", downloadURLResp.ExpiresAt)

	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: downloadURLResp})
}
