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
		// NOTE: intentionally NOT migrated to response.Error() here.
		// response.Error() always emits {status, data, message}, but
		// route_test.go (TestRouteRejectsUnknownFieldsAndTrailingJSON,
		// TestRouteRejectsInvalidRawUTF8BeforeDecoding) asserts this body is
		// the bare {"message": "invalid request body"} with no data/status
		// keys. Confirmed by running those tests with the migration applied
		// (both fail on JSONEq). Left as a raw gin.H literal per the Task
		// 6/7 precedent for call sites the shared helper cannot express.
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	result, err := handler.service.EnsureInspection(user, c.Param("equipment_id"), c.Param("pmcs_id"), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	// NOTE: intentionally NOT migrated to response.OK() here. response.OK()
	// has no parameter for a success message, and this handler's body has a
	// non-empty "Inspection saved" Message alongside Data. Wrapping in
	// response.OK() would silently drop that message text (Message would
	// default to ""), which is an unintended content change even though no
	// test currently asserts on it. Left as a raw StandardResponse literal
	// per the Task 6/7 precedent for call sites response.OK() cannot express.
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

	response.OK(c, result)
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

	// NOTE: intentionally NOT migrated to response.OK() here. response.OK()
	// has no parameter for a success message, and this body is a bare
	// {"message": "Inspection deleted"} with no Data payload. Left as a raw
	// gin.H literal per the Task 6/7 precedent for call sites response.OK()
	// cannot express.
	c.JSON(http.StatusOK, gin.H{"message": "Inspection deleted"})
}

func (handler Handler) listInspections(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req ListInspectionsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		// NOTE: intentionally NOT migrated to response.Error() here, for the
		// same reason as upsertInspection's decode-error branch above: the
		// bare {"message": ...} shape is preserved to match the sibling
		// bad-request bodies this file's handler tests assert on.
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid query parameters"})
		return
	}

	result, err := handler.service.ListInspections(user, c.Param("equipment_id"), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	response.OK(c, result)
}

func (handler Handler) upsertFault(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req FaultRequest
	if err := decodeInspectionJSON(c, &req); err != nil {
		// NOTE: intentionally NOT migrated to response.Error() here, for the
		// same reason as upsertInspection's decode-error branch above.
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	result, err := handler.service.UpsertFault(user, c.Param("equipment_id"), c.Param("pmcs_id"), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	// NOTE: intentionally NOT migrated to response.OK() here. response.OK()
	// has no parameter for a success message, and this handler's body has a
	// non-empty "Fault saved" Message alongside Data. Left as a raw
	// StandardResponse literal per the Task 6/7 precedent for call sites
	// response.OK() cannot express.
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Message: "Fault saved", Data: result})
}

func (handler Handler) deleteFault(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req DeleteFaultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// NOTE: intentionally NOT migrated to response.Error() here, for the
		// same reason as upsertInspection's decode-error branch above.
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	if err := handler.service.DeleteFault(user, c.Param("equipment_id"), c.Param("pmcs_id"), req); err != nil {
		respondServiceError(c, err)
		return
	}

	// NOTE: intentionally NOT migrated to response.OK() here. response.OK()
	// has no parameter for a success message, and this body is a bare
	// {"message": "Fault deleted"} with no Data payload. Left as a raw
	// gin.H literal per the Task 6/7 precedent for call sites response.OK()
	// cannot express.
	c.JSON(http.StatusOK, gin.H{"message": "Fault deleted"})
}

func (handler Handler) deleteFaults(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req BulkDeleteFaultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// NOTE: intentionally NOT migrated to response.Error() here, for the
		// same reason as upsertInspection's decode-error branch above.
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	result, err := handler.service.DeleteFaults(user, c.Param("equipment_id"), c.Param("pmcs_id"), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	// NOTE: intentionally NOT migrated to response.OK() here. This body is
	// flat — {message, requested_count, deleted_count} all at the top level.
	// response.OK() would nest everything under a new "data" key (a real
	// shape change) and also has no parameter for a success message; the
	// count fields don't fit the message-only response.Error() shape either.
	// Left as a raw gin.H literal per the Task 6/7 precedent for call sites
	// neither helper can express.
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
		// NOTE: intentionally NOT migrated to response.Error() here, for the
		// same reason as upsertInspection's decode-error branch above.
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	result, err := handler.service.CreateComment(user, c.Param("equipment_id"), c.Param("pmcs_id"), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	// NOTE: intentionally NOT migrated to response.OK() here. response.OK()
	// hardcodes status 200, but this handler responds 201 Created, and it
	// also has no parameter for a success message ("Comment created" would
	// be silently dropped). Left as a raw StandardResponse literal per the
	// Task 6/7 precedent for call sites response.OK() cannot express.
	c.JSON(http.StatusCreated, response.StandardResponse{Status: http.StatusCreated, Message: "Comment created", Data: result})
}

func (handler Handler) updateComment(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// NOTE: intentionally NOT migrated to response.Error() here, for the
		// same reason as upsertInspection's decode-error branch above.
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	result, err := handler.service.UpdateComment(user, c.Param("equipment_id"), c.Param("pmcs_id"), c.Param("comment_id"), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	// NOTE: intentionally NOT migrated to response.OK() here. response.OK()
	// has no parameter for a success message, and this handler's body has a
	// non-empty "Comment updated" Message alongside Data. Left as a raw
	// StandardResponse literal per the Task 6/7 precedent for call sites
	// response.OK() cannot express.
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

	// NOTE: intentionally NOT migrated to response.OK() here. response.OK()
	// has no parameter for a success message, and this handler's body has a
	// non-empty "Comment deleted" Message alongside Data. Left as a raw
	// StandardResponse literal per the Task 6/7 precedent for call sites
	// response.OK() cannot express.
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
		// NOTE: intentionally NOT migrated to response.Error() here, for the
		// same reason as upsertInspection's decode-error branch above: the
		// bare {"message": "unauthorized"} shape matches this file's other
		// hand-authored error bodies and no test forces the richer envelope.
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
	// NOTE: none of this switch's branches were migrated to response.Error().
	// response.Error() always emits {status, data, message}, but
	// route_test.go's TestRouteSourceValidationKeepsBadRequestEnvelope
	// asserts the ErrInvalidRequest branch's body is exactly
	// {"message": "invalid request"} with require.NotContains(t, body,
	// "data") — confirmed failing with the migration applied. The other
	// branches in this switch use the same bare {"message": ...} shape, so
	// they are kept raw too for a single consistent envelope across this
	// function, per the Task 6/7 precedent for call sites the shared helper
	// cannot express.
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
