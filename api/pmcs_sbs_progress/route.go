package pmcs_sbs_progress

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

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

func RegisterRoutes(deps Dependencies, group *gin.RouterGroup) {
	repo := NewRepository(deps.DB)
	svc := NewService(repo)
	registerHandlers(group, svc)
}

func registerHandlers(group *gin.RouterGroup, svc Service) {
	handler := Handler{service: svc}

	group.GET("/pmcs-sbs/equipment", handler.listEquipment)
	group.GET("/pmcs-sbs/equipment/:equipment_id", handler.getEquipment)
	group.PUT("/pmcs-sbs/equipment/:equipment_id", handler.upsertEquipment)
	group.DELETE("/pmcs-sbs/equipment/:equipment_id", handler.deleteEquipment)
	group.PUT("/pmcs-sbs/equipment/:equipment_id/completions", handler.upsertCompletion)
	group.PATCH("/pmcs-sbs/equipment/:equipment_id/completions/batch", handler.batchCompletions)
	group.DELETE("/pmcs-sbs/equipment/:equipment_id/completions", handler.deleteCompletion)
	group.PUT("/pmcs-sbs/equipment/:equipment_id/faults", handler.upsertFault)
	group.DELETE("/pmcs-sbs/equipment/:equipment_id/faults", handler.deleteFault)
	group.POST("/pmcs-sbs/sync", handler.sync)
}

func (handler Handler) listEquipment(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	result, err := handler.service.ListEquipment(user)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Message: "", Data: result})
}

func (handler Handler) getEquipment(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	result, err := handler.service.GetEquipment(user, c.Param("equipment_id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Message: "", Data: result})
}

func (handler Handler) upsertEquipment(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req EquipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	result, err := handler.service.UpsertEquipment(user, c.Param("equipment_id"), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Message: "Equipment saved", Data: result})
}

func (handler Handler) deleteEquipment(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	if err := handler.service.DeleteEquipment(user, c.Param("equipment_id")); err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Equipment deleted"})
}

func (handler Handler) upsertCompletion(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req CompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	result, err := handler.service.UpsertCompletion(user, c.Param("equipment_id"), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Message: "Completion saved", Data: result})
}

func (handler Handler) batchCompletions(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req BatchCompletionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	result, err := handler.service.BatchCompletions(user, c.Param("equipment_id"), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Message: "Completions synced", Data: result})
}

func (handler Handler) deleteCompletion(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req DeleteCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	if err := handler.service.DeleteCompletion(user, c.Param("equipment_id"), req); err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Completion deleted"})
}

func (handler Handler) upsertFault(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req FaultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	result, err := handler.service.UpsertFault(user, c.Param("equipment_id"), req)
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

	if err := handler.service.DeleteFault(user, c.Param("equipment_id"), req); err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Fault deleted"})
}

func (handler Handler) sync(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		return
	}

	var req SyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	result, err := handler.service.Sync(user, req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Message: "Sync complete", Data: result})
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
		errors.Is(err, ErrInvalidRequest),
		errors.Is(err, ErrInvalidBlobPath),
		errors.Is(err, ErrInvalidStatus),
		errors.Is(err, ErrInvalidSyncRequest):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "pmcs sbs equipment not found"})
	default:
		slog.Error("PMCS SBS progress handler failed", "error", err)
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
	}
}
