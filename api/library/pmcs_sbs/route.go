package pmcs_sbs

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"

	"miltechserver/api/middleware"
	"miltechserver/api/response"
)

// Handler holds the pmcs_sbs service dependency.
type Handler struct {
	service Service
}

// RegisterHandlers wires pmcs_sbs routes into the public router group.
// Called from api/library/route.go.
func RegisterHandlers(publicGroup *gin.RouterGroup, blobClient *azblob.Client) {
	svc := NewService(blobClient)
	registerHandlers(publicGroup, svc)
}

// registerHandlers is the internal wiring function used directly by tests.
func registerHandlers(publicGroup *gin.RouterGroup, svc Service) {
	h := Handler{service: svc}
	publicGroup.GET("/library/pmcs-sbs/folders", h.getFolders)
	publicGroup.GET("/library/pmcs-sbs/:folder/files", h.getFiles)
	// Rate-limited: each IP is allowed a burst of 10 requests, sustained at 2 req/s.
	publicGroup.GET("/library/pmcs-sbs/content", middleware.RateLimiter(), h.getFileContent)
	publicGroup.GET("/library/pmcs-sbs/image", middleware.RateLimiter(), h.getImage)
}

// getFolders returns all top-level folders in the PMCS SBS library.
// GET /library/pmcs-sbs/folders
func (h *Handler) getFolders(c *gin.Context) {
	slog.Info("GetPMCSSBSFolders endpoint called")

	folders, err := h.service.GetFolders(c.Request.Context())
	if err != nil {
		slog.Error("Failed to retrieve PMCS SBS folders", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve PMCS SBS folders",
		})
		return
	}

	slog.Info("Successfully retrieved PMCS SBS folders", "count", folders.Count)
	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: folders})
}

// getFiles returns all JSON files in a specific PMCS SBS folder.
// GET /library/pmcs-sbs/:folder/files
func (h *Handler) getFiles(c *gin.Context) {
	folderName := c.Param("folder")

	slog.Info("GetPMCSSBSFiles endpoint called", "folder", folderName)

	if strings.TrimSpace(folderName) == "" {
		slog.Warn("GetPMCSSBSFiles called with empty folder name")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Folder name is required",
		})
		return
	}

	files, err := h.service.GetFiles(c.Request.Context(), folderName)
	if err != nil {
		slog.Error("Failed to retrieve PMCS SBS files", "error", err, "folder", folderName)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve PMCS SBS files",
		})
		return
	}

	slog.Info("Successfully retrieved PMCS SBS files", "count", files.Count, "folder", folderName)
	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: files})
}

// getFileContent fetches a JSON blob from Azure and returns its raw content.
// GET /library/pmcs-sbs/content?blob_path=pmcs_sbs/hmmwv/file.json
func (h *Handler) getFileContent(c *gin.Context) {
	blobPath := c.Query("blob_path")

	slog.Info("GetPMCSSBSFileContent endpoint called", "blobPath", blobPath)

	if strings.TrimSpace(blobPath) == "" {
		slog.Warn("GetPMCSSBSFileContent called with empty blob_path")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "blob_path query parameter is required",
		})
		return
	}

	content, err := h.service.GetFileContent(c.Request.Context(), blobPath)
	if err != nil {
		switch {
		case errors.Is(err, ErrFileNotFound):
			slog.Warn("PMCS SBS file not found", "blobPath", blobPath, "error", err)
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "File not found",
				"details": "The requested file does not exist or is not accessible",
			})
		case errors.Is(err, ErrEmptyBlobPath), errors.Is(err, ErrInvalidBlobPath), errors.Is(err, ErrInvalidFileType):
			slog.Warn("Invalid blob path for PMCS SBS content", "blobPath", blobPath, "error", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request",
				"details": err.Error(),
			})
		default:
			slog.Error("Failed to retrieve PMCS SBS file content", "error", err, "blobPath", blobPath)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve file content",
			})
		}
		return
	}

	slog.Info("Successfully retrieved PMCS SBS file content", "blobPath", blobPath)
	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: content})
}

// getImage fetches a guide item PNG from Azure and streams its raw bytes.
// GET /library/pmcs-sbs/image?blob_path=pmcs_sbs/hmmwv/file.json&image_name=Before_12
func (h *Handler) getImage(c *gin.Context) {
	blobPath := c.Query("blob_path")
	imageName := c.Query("image_name")

	slog.Info("GetPMCSSBSImage endpoint called", "blobPath", blobPath, "imageName", imageName)

	if strings.TrimSpace(blobPath) == "" {
		slog.Warn("GetPMCSSBSImage called with empty blob_path", "imageName", imageName)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "blob_path query parameter is required",
		})
		return
	}

	if strings.TrimSpace(imageName) == "" {
		slog.Warn("GetPMCSSBSImage called with empty image_name", "blobPath", blobPath)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "image_name query parameter is required",
		})
		return
	}

	image, err := h.service.GetImage(c.Request.Context(), blobPath, imageName)
	if err != nil {
		switch {
		case errors.Is(err, ErrFileNotFound):
			slog.Warn("PMCS SBS image not found", "blobPath", blobPath, "imageName", imageName, "error", err)
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Image not found",
				"details": "The requested image does not exist or is not accessible",
			})
		case errors.Is(err, ErrEmptyBlobPath),
			errors.Is(err, ErrInvalidBlobPath),
			errors.Is(err, ErrInvalidFileType),
			errors.Is(err, ErrEmptyImageName),
			errors.Is(err, ErrInvalidImageName):
			slog.Warn("Invalid request for PMCS SBS image", "blobPath", blobPath, "imageName", imageName, "error", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request",
				"details": err.Error(),
			})
		default:
			slog.Error("Failed to retrieve PMCS SBS image", "error", err, "blobPath", blobPath, "imageName", imageName)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve image",
			})
		}
		return
	}
	defer image.Body.Close()

	slog.Info(
		"Successfully retrieved PMCS SBS image",
		"blobPath", blobPath,
		"imageName", imageName,
		"imageBlobPath", image.BlobPath,
	)

	extraHeaders := map[string]string{
		"Content-Disposition": fmt.Sprintf(`inline; filename="%s"`, image.FileName),
		"Cache-Control":       "public, max-age=86400",
	}

	c.DataFromReader(http.StatusOK, image.ContentLength, image.ContentType, image.Body, extraHeaders)
}
