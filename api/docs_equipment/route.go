package docs_equipment

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"

	"miltechserver/api/middleware"
	"miltechserver/api/response"
	"miltechserver/api/shared/pagination"
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
	router.GET("/equipment-details/images/family/:family/urls", middleware.RateLimiter(), handler.getFamilyImageURLs)
	router.GET("/equipment-details/images/download", middleware.RateLimiter(), handler.generateImageDownloadURL)
}

func (h *Handler) getAllPaginated(c *gin.Context) {
	page, ok := pagination.ParsePage(c)
	if !ok {
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

	response.OK(c, data)
}

func (h *Handler) getFamilies(c *gin.Context) {
	data, err := h.service.GetFamilies()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	response.OK(c, data)
}

func (h *Handler) getByFamily(c *gin.Context) {
	family := c.Param("family")
	if strings.TrimSpace(family) == "" {
		response.Error(c, http.StatusBadRequest, "Family parameter is required")
		return
	}

	page, ok := pagination.ParsePage(c)
	if !ok {
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

	response.OK(c, data)
}

func (h *Handler) search(c *gin.Context) {
	q := c.Query("q")
	if strings.TrimSpace(q) == "" {
		response.Error(c, http.StatusBadRequest, "Search query (q) is required")
		return
	}

	page, ok := pagination.ParsePage(c)
	if !ok {
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

	response.OK(c, data)
}

func (h *Handler) listImageFamilies(c *gin.Context) {
	data, err := h.service.ListImageFamilies()
	if err != nil {
		slog.Error("Failed to list image families", "error", err)
		// Kept as a raw gin.H{} response: this is a flat multi-field body
		// ("error" + "details"), which response.Error() cannot represent
		// (it only carries a single message string).
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list image families",
			"details": err.Error(),
		})
		return
	}

	response.OK(c, data)
}

func (h *Handler) listFamilyImages(c *gin.Context) {
	family := c.Param("family")
	if strings.TrimSpace(family) == "" {
		response.Error(c, http.StatusBadRequest, "Family parameter is required")
		return
	}

	data, err := h.service.ListFamilyImages(family)
	if err != nil {
		slog.Error("Failed to list family images", "error", err, "family", family)
		// Kept as a raw gin.H{} response: flat multi-field body ("error" +
		// "details") that response.Error()'s single message string cannot represent.
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list images",
			"details": err.Error(),
		})
		return
	}

	response.OK(c, data)
}

func (h *Handler) getFamilyImageURLs(c *gin.Context) {
	family := c.Param("family")
	if strings.TrimSpace(family) == "" {
		response.Error(c, http.StatusBadRequest, "Family parameter is required")
		return
	}

	data, err := h.service.GetFamilyImageURLs(c.Request.Context(), family)
	if err != nil {
		if errors.Is(err, ErrEmptyParam) {
			response.Error(c, http.StatusBadRequest, "Family parameter is required")
		} else {
			slog.Error("Failed to get family image URLs", "error", err, "family", family)
			// Kept as a raw gin.H{} response: flat multi-field body ("error" +
			// "details") that response.Error()'s single message string cannot represent.
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to generate image URLs",
				"details": err.Error(),
			})
		}
		return
	}

	response.OK(c, data)
}

func (h *Handler) generateImageDownloadURL(c *gin.Context) {
	blobPath := c.Query("blob_path")

	result, err := h.service.GenerateImageDownloadURL(c.Request.Context(), blobPath)
	if err != nil {
		switch {
		case errors.Is(err, ErrImageNotFound):
			// Kept as a raw gin.H{} response: flat multi-field body ("error" +
			// "details") that response.Error()'s single message string cannot represent.
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

	response.OK(c, result)
}
