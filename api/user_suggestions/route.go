package user_suggestions

import (
	"database/sql"
	"errors"
	"net/http"

	"miltechserver/api/middleware"
	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	DB         *sql.DB
	AuthClient *auth.Client
}

type Handler struct {
	service Service
}

func RegisterRoutes(deps Dependencies, publicGroup, authGroup *gin.RouterGroup) {
	repo := NewRepository(deps.DB)
	svc := NewService(repo)
	registerHandlers(publicGroup, authGroup, deps.AuthClient, svc)
}

func registerHandlers(publicGroup, authGroup *gin.RouterGroup, authClient *auth.Client, svc Service) {
	handler := Handler{service: svc}

	// Public route with optional auth so authenticated users get MyVote field
	if authClient != nil {
		publicGroup.GET("/suggestions", middleware.OptionalAuthMiddleware(authClient), handler.listSuggestions)
	} else {
		// Testing path: no real auth client, skip optional auth middleware
		publicGroup.GET("/suggestions", handler.listSuggestions)
	}

	authGroup.POST("/suggestions", handler.createSuggestion)
	authGroup.PUT("/suggestions/:id", handler.updateSuggestion)
	authGroup.DELETE("/suggestions/:id", handler.deleteSuggestion)
	authGroup.POST("/suggestions/:id/vote", handler.vote)
	authGroup.DELETE("/suggestions/:id/vote", handler.removeVote)
}

func (h *Handler) listSuggestions(c *gin.Context) {
	var currentUser *bootstrap.User
	if user, exists := c.Get("user"); exists {
		if u, ok := user.(*bootstrap.User); ok {
			currentUser = u
		}
	}

	suggestions, err := h.service.GetAllSuggestions(currentUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Suggestions retrieved",
		Data:    suggestions,
	})
}

func (h *Handler) createSuggestion(c *gin.Context) {
	currentUser, ok := getUser(c)
	if !ok {
		return
	}

	var req CreateSuggestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	suggestion, err := h.service.CreateSuggestion(currentUser, req.Title, req.Description)
	if err != nil {
		if respondError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	c.JSON(http.StatusCreated, response.StandardResponse{
		Status:  http.StatusCreated,
		Message: "Suggestion created",
		Data:    suggestion,
	})
}

func (h *Handler) updateSuggestion(c *gin.Context) {
	currentUser, ok := getUser(c)
	if !ok {
		return
	}

	suggestionID := c.Param("id")

	var req UpdateSuggestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	suggestion, err := h.service.UpdateSuggestion(currentUser, suggestionID, req.Title, req.Description)
	if err != nil {
		if respondError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Suggestion updated",
		Data:    suggestion,
	})
}

func (h *Handler) deleteSuggestion(c *gin.Context) {
	currentUser, ok := getUser(c)
	if !ok {
		return
	}

	suggestionID := c.Param("id")

	err := h.service.DeleteSuggestion(currentUser, suggestionID)
	if err != nil {
		if respondError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Suggestion deleted",
	})
}

func (h *Handler) vote(c *gin.Context) {
	currentUser, ok := getUser(c)
	if !ok {
		return
	}

	suggestionID := c.Param("id")

	var req VoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	err := h.service.Vote(currentUser, suggestionID, req.Direction)
	if err != nil {
		if respondError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Vote recorded",
	})
}

func (h *Handler) removeVote(c *gin.Context) {
	currentUser, ok := getUser(c)
	if !ok {
		return
	}

	suggestionID := c.Param("id")

	err := h.service.RemoveVote(currentUser, suggestionID)
	if err != nil {
		if respondError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Vote removed",
	})
}

func getUser(c *gin.Context) (*bootstrap.User, bool) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return nil, false
	}

	currentUser, ok := user.(*bootstrap.User)
	if !ok || currentUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return nil, false
	}

	return currentUser, true
}

type errorMapping struct {
	target  error
	status  int
	message string
}

var errorMappings = []errorMapping{
	{target: ErrSuggestionNotFound, status: http.StatusNotFound, message: "suggestion not found"},
	{target: ErrInvalidTitle, status: http.StatusBadRequest, message: "invalid title"},
	{target: ErrInvalidDescription, status: http.StatusBadRequest, message: "invalid description"},
	{target: ErrInvalidDirection, status: http.StatusBadRequest, message: "invalid vote direction"},
	{target: ErrInvalidID, status: http.StatusBadRequest, message: "invalid suggestion ID"},
	{target: ErrUnauthorized, status: http.StatusUnauthorized, message: "unauthorized"},
	{target: ErrForbidden, status: http.StatusForbidden, message: "forbidden"},
}

func respondError(c *gin.Context, err error) bool {
	for _, em := range errorMappings {
		if errors.Is(err, em.target) {
			c.JSON(em.status, gin.H{"message": em.message})
			return true
		}
	}
	return false
}
