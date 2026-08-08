package pmcs_sbs_progress

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"unicode/utf8"

	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	DB *sql.DB
}

type Handler struct {
	service Service
}

const maxInspectionRequestBodyBytes int64 = 8 * 1024 * 1024

func RegisterRoutes(deps Dependencies, group *gin.RouterGroup) {
	repo := NewRepository(deps.DB)
	svc := NewService(repo)
	registerHandlers(group, svc)
}

func registerHandlers(group *gin.RouterGroup, svc Service) {
	handler := Handler{service: svc}

	group.PUT("/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id", handler.upsertInspection)
	group.GET("/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id", handler.getInspection)
	group.DELETE("/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id", handler.deleteInspection)
	group.GET("/pmcs-sbs/equipment/:equipment_id/pmcs", handler.listInspections)
	group.PUT("/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults", handler.upsertFault)
	group.DELETE("/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults", handler.deleteFault)
	group.DELETE("/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults/bulk", handler.deleteFaults)
	group.POST("/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/comments", handler.createComment)
	group.PUT("/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/comments/:comment_id", handler.updateComment)
	group.DELETE("/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/comments/:comment_id", handler.deleteComment)
}

func (handler Handler) upsertInspection(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req InspectionRequest
	if err := decodeInspectionJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	result, err := handler.service.EnsureInspection(user, c.Param("equipment_id"), c.Param("pmcs_id"), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Message: "Inspection saved", Data: result})
}

func (handler Handler) getInspection(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	result, err := handler.service.GetInspection(user, c.Param("equipment_id"), c.Param("pmcs_id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Message: "", Data: result})
}

func (handler Handler) deleteInspection(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	if err := handler.service.DeleteInspection(user, c.Param("equipment_id"), c.Param("pmcs_id")); err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Inspection deleted"})
}

func (handler Handler) listInspections(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req ListInspectionsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid query parameters"})
		return
	}

	result, err := handler.service.ListInspections(user, c.Param("equipment_id"), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Message: "", Data: result})
}

func (handler Handler) upsertFault(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req FaultRequest
	if err := decodeInspectionJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	result, err := handler.service.UpsertFault(user, c.Param("equipment_id"), c.Param("pmcs_id"), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Message: "Fault saved", Data: result})
}

func (handler Handler) deleteFault(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req DeleteFaultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	if err := handler.service.DeleteFault(user, c.Param("equipment_id"), c.Param("pmcs_id"), req); err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Fault deleted"})
}

func (handler Handler) deleteFaults(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req BulkDeleteFaultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	result, err := handler.service.DeleteFaults(user, c.Param("equipment_id"), c.Param("pmcs_id"), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Faults deleted",
		"requested_count": result.RequestedCount,
		"deleted_count":   result.DeletedCount,
	})
}

func (handler Handler) createComment(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	result, err := handler.service.CreateComment(user, c.Param("equipment_id"), c.Param("pmcs_id"), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, response.StandardResponse{Status: http.StatusCreated, Message: "Comment created", Data: result})
}

func (handler Handler) updateComment(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	result, err := handler.service.UpdateComment(user, c.Param("equipment_id"), c.Param("pmcs_id"), c.Param("comment_id"), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Message: "Comment updated", Data: result})
}

func (handler Handler) deleteComment(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	result, err := handler.service.DeleteComment(user, c.Param("equipment_id"), c.Param("pmcs_id"), c.Param("comment_id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Message: "Comment deleted", Data: result})
}

// decodeInspectionJSON validates raw bytes before JSON decoding so malformed
// UTF-8 cannot be replaced by encoding/json and bypass service validation.
func decodeInspectionJSON(c *gin.Context, destination any) error {
	body := http.MaxBytesReader(c.Writer, c.Request.Body, maxInspectionRequestBodyBytes)
	defer body.Close()

	payload, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	if !utf8.Valid(payload) {
		return ErrInvalidRequest
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return ErrInvalidRequest
	}
	return nil
}

func getUser(c *gin.Context) (*bootstrap.User, bool) {
	value, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return nil, false
	}

	user, ok := value.(*bootstrap.User)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return nil, false
	}

	return user, true
}

func respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
	case errors.Is(err, ErrInvalidID),
		errors.Is(err, ErrInvalidPmcsID),
		errors.Is(err, ErrInvalidGuideManual),
		errors.Is(err, ErrInvalidRequest),
		errors.Is(err, ErrInvalidStatus),
		errors.Is(err, ErrInvalidCommentText):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	case errors.Is(err, ErrInspectionConflict):
		c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
	case errors.Is(err, ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"message": err.Error()})
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "pmcs sbs equipment not found"})
	case errors.Is(err, ErrInspectionNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "pmcs sbs inspection not found"})
	case errors.Is(err, ErrCommentNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "pmcs sbs comment not found"})
	default:
		slog.Error("PMCS SBS fault handler failed", "error", err)
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
	}
}
