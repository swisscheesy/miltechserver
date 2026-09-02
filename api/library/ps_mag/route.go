package ps_mag

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"

	"miltechserver/api/analytics"
	"miltechserver/api/middleware"
	"miltechserver/api/response"
)

// Handler holds the ps_mag service dependency.
type Handler struct {
	service Service
}

// RegisterHandlers wires ps_mag routes into the public router group.
// Called from api/library/route.go.
func RegisterHandlers(publicGroup *gin.RouterGroup, blobClient *azblob.Client, db *sql.DB, analyticsService analytics.Service) {
	svc := NewService(blobClient, db, analyticsService)
	registerHandlers(publicGroup, svc)
}

// registerHandlers is the internal wiring function used directly by tests.
func registerHandlers(publicGroup *gin.RouterGroup, svc Service) {
	handler := Handler{service: svc}
	publicGroup.GET("/library/ps-mag/issues", handler.listIssues)
	publicGroup.GET("/library/ps-mag/search", handler.searchSummaries)
	// Rate-limited: each IP is allowed a burst of 10 requests, sustained at 2 req/s.
	publicGroup.GET("/library/ps-mag/download", middleware.RateLimiter(), handler.generateDownloadURL)
}

// parsePage reads the "page" query parameter, defaulting to 1 if absent,
// and validates it is a positive integer. On invalid input it writes
// ps_mag's own {"error": "Invalid request", "details": ...} response and
// returns (0, false); callers must return immediately when ok is false.
//
// This intentionally duplicates pagination.ParsePage's validation logic
// rather than calling it directly: ps_mag's error response body differs
// from ParsePage's built-in {"error": "Invalid page number"} body, and
// ParsePage writes its response as a side effect, so calling it here would
// silently change ps_mag's response shape.
func parsePage(c *gin.Context) (int, bool) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": ErrInvalidPage.Error(),
		})
		return 0, false
	}
	return page, true
}

// listIssues returns a paginated list of PS Magazine issues.
// GET /library/ps-mag/issues?page=1&order=asc&year=1994&issue=495
func (h *Handler) listIssues(c *gin.Context) {
	order := c.DefaultQuery("order", "asc")

	page, ok := parsePage(c)
	if !ok {
		return
	}

	if o := strings.ToLower(order); o != "asc" && o != "desc" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": ErrInvalidOrder.Error(),
		})
		return
	}

	var year *int
	if yearStr := c.Query("year"); yearStr != "" {
		y, err := strconv.Atoi(yearStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request",
				"details": "year must be a valid integer",
			})
			return
		}
		year = &y
	}

	var issueNumber *int
	if issueStr := c.Query("issue"); issueStr != "" {
		i, err := strconv.Atoi(issueStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request",
				"details": "issue must be a valid integer",
			})
			return
		}
		issueNumber = &i
	}

	slog.Info("ListPSMagIssues endpoint called",
		"page", page, "order", order, "year", year, "issueNumber", issueNumber)

	result, err := h.service.ListIssues(c.Request.Context(), page, order, year, issueNumber)
	if err != nil {
		slog.Error("Failed to list PS Magazine issues", "error", err)
		response.Error(c, http.StatusInternalServerError, "Failed to list issues")
		return
	}

	slog.Info("Successfully listed PS Magazine issues",
		"count", result.Count, "totalCount", result.TotalCount, "page", result.Page)

	c.Header("Cache-Control", "public, max-age=300")
	response.OK(c, result)
}

// searchSummaries returns PS Magazine issues whose summary contains the query phrase.
// Only the lines from each summary that contain the phrase are returned.
// GET /library/ps-mag/search?q=phrase&page=1
func (h *Handler) searchSummaries(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if len(q) < 3 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": ErrQueryTooShort.Error(),
		})
		return
	}

	page, ok := parsePage(c)
	if !ok {
		return
	}

	slog.Info("SearchPSMagSummaries endpoint called", "query", q, "page", page)

	result, err := h.service.SearchSummaries(q, page)
	if err != nil {
		slog.Error("Failed to search PS Magazine summaries", "error", err)
		response.Error(c, http.StatusInternalServerError, "Failed to search summaries")
		return
	}

	slog.Info("Successfully searched PS Magazine summaries",
		"query", q, "totalCount", result.TotalCount, "page", result.Page)

	response.OK(c, result)
}

// generateDownloadURL returns a time-limited SAS URL for downloading a PS Magazine issue.
// GET /library/ps-mag/download?blob_path=ps-mag/PS_Magazine_Issue_495_February_1994.pdf
func (h *Handler) generateDownloadURL(c *gin.Context) {
	blobPath := c.Query("blob_path")

	slog.Info("GeneratePSMagDownloadURL endpoint called", "blobPath", blobPath)

	result, err := h.service.GenerateDownloadURL(c.Request.Context(), blobPath)
	if err != nil {
		switch {
		case errors.Is(err, ErrIssueNotFound):
			slog.Warn("PS Magazine issue not found", "blobPath", blobPath, "error", err)
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Issue not found",
				"details": "The requested issue does not exist or is not accessible",
			})
		case errors.Is(err, ErrEmptyBlobPath), errors.Is(err, ErrInvalidBlobPath), errors.Is(err, ErrInvalidFileType):
			slog.Warn("Invalid blob path for PS Magazine download", "blobPath", blobPath, "error", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request",
				"details": err.Error(),
			})
		default:
			slog.Error("Failed to generate PS Magazine download URL", "error", err, "blobPath", blobPath)
			response.Error(c, http.StatusInternalServerError, "Failed to generate download URL")
		}
		return
	}

	slog.Info("Successfully generated PS Magazine download URL",
		"blobPath", blobPath, "expiresAt", result.ExpiresAt)

	response.OK(c, result)
}
