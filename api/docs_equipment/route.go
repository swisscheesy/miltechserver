package docs_equipment

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"

	"miltechserver/api/middleware"
	"miltechserver/api/response"
)

// Dependencies holds external resources needed by this package.
type Dependencies struct {
	DB         *sql.DB
	BlobClient *azblob.Client
}

// Handler holds the service dependency.
type Handler struct {
	service Service
}

// RegisterRoutes wires docs_equipment routes into the public router group.
func RegisterRoutes(deps Dependencies, router *gin.RouterGroup) {
	repo := NewRepository(deps.DB)
	svc := NewService(repo, deps.BlobClient)
	registerHandlers(router, svc)
}

// registerHandlers is the internal wiring function used directly by tests.
func registerHandlers(router *gin.RouterGroup, svc Service) {
	handler := Handler{service: svc}

	// Data endpoints
	router.GET("/equipment-details", handler.getAllPaginated)
	router.GET("/equipment-details/families", handler.getFamilies)
	router.GET("/equipment-details/family/:family", handler.getByFamily)
	router.GET("/equipment-details/search", handler.search)

	// Image endpoints
	router.GET("/equipment-details/images/families", handler.listImageFamilies)
	router.GET("/equipment-details/images/family/:family", handler.listFamilyImages)
	router.GET("/equipment-details/images/download", middleware.RateLimiter(), handler.generateImageDownloadURL)
}

func (h *Handler) getAllPaginated(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
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

	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: data})
}

func (h *Handler) getFamilies(c *gin.Context) {
	data, err := h.service.GetFamilies()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: data})
}

func (h *Handler) getByFamily(c *gin.Context) {
	family := c.Param("family")
	if strings.TrimSpace(family) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Family parameter is required"})
		return
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}

	data, err := h.service.GetByFamilyPaginated(family, page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: data})
}

func (h *Handler) search(c *gin.Context) {
	q := c.Query("q")
	if strings.TrimSpace(q) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query (q) is required"})
		return
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}

	data, err := h.service.SearchPaginated(q, page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: data})
}

func (h *Handler) listImageFamilies(c *gin.Context) {
	data, err := h.service.ListImageFamilies()
	if err != nil {
		slog.Error("Failed to list image families", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list image families",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: data})
}

func (h *Handler) listFamilyImages(c *gin.Context) {
	family := c.Param("family")
	if strings.TrimSpace(family) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Family parameter is required"})
		return
	}

	data, err := h.service.ListFamilyImages(family)
	if err != nil {
		slog.Error("Failed to list family images", "error", err, "family", family)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list images",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: data})
}

func (h *Handler) generateImageDownloadURL(c *gin.Context) {
	blobPath := c.Query("blob_path")

	result, err := h.service.GenerateImageDownloadURL(c.Request.Context(), blobPath)
	if err != nil {
		switch {
		case errors.Is(err, ErrImageNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Image not found",
				"details": "The requested image does not exist",
			})
		case errors.Is(err, ErrEmptyBlobPath), errors.Is(err, ErrInvalidBlobPath), errors.Is(err, ErrInvalidFileType):
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request",
				"details": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to generate download URL",
				"details": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: result})
}
