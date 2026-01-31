package item_comments

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Dependencies struct {
	DB *sql.DB
}

type Handler struct {
	service Service
}

func RegisterRoutes(deps Dependencies, publicGroup, authGroup *gin.RouterGroup) {
	repo := NewRepository(deps.DB)
	svc := NewService(repo)
	registerHandlers(publicGroup, authGroup, svc)
}

func registerHandlers(publicGroup, authGroup *gin.RouterGroup, svc Service) {
	handler := Handler{service: svc}

	publicGroup.GET("/items/:niin/comments", handler.getCommentsByNiin)

	authGroup.POST("/items/:niin/comments", handler.createComment)
	authGroup.PUT("/items/:niin/comments/:comment_id", handler.updateComment)
	authGroup.DELETE("/items/:niin/comments/:comment_id", handler.deleteComment)
	authGroup.POST("/items/:niin/comments/:comment_id/flags", handler.flagComment)
}

func (handler *Handler) getCommentsByNiin(c *gin.Context) {
	niin := c.Param("niin")
	comments, err := handler.service.GetCommentsByNiin(niin)
	if err != nil {
		if respondError(c, err, []errorCase{
			{target: ErrInvalidNiin, status: http.StatusBadRequest, message: "invalid NIIN"},
		}) {
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Comments retrieved",
		Data:    comments,
	})
}

func (handler *Handler) createComment(c *gin.Context) {
	currentUser, ok := getUser(c)
	if !ok {
		return
	}

	niin := c.Param("niin")

	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	comment, err := handler.service.CreateComment(currentUser, niin, req.Text, req.ParentID)
	if err != nil {
		if respondError(c, err, []errorCase{
			{target: ErrInvalidNiin, status: http.StatusBadRequest, message: "invalid NIIN"},
			{target: ErrInvalidText, status: http.StatusBadRequest, message: "invalid comment text"},
			{target: ErrInvalidParent, status: http.StatusBadRequest, message: "invalid parent comment"},
			{target: ErrUnauthorized, status: http.StatusUnauthorized, message: "unauthorized"},
		}) {
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	c.JSON(http.StatusCreated, response.StandardResponse{
		Status:  http.StatusCreated,
		Message: "Comment created",
		Data:    comment,
	})
}

func (handler *Handler) updateComment(c *gin.Context) {
	currentUser, ok := getUser(c)
	if !ok {
		return
	}

	niin := c.Param("niin")
	commentID := c.Param("comment_id")

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	comment, err := handler.service.UpdateComment(currentUser, niin, commentID, req.Text)
	if err != nil {
		if respondError(c, err, []errorCase{
			{target: ErrInvalidNiin, status: http.StatusBadRequest, message: "invalid NIIN"},
			{target: ErrInvalidText, status: http.StatusBadRequest, message: "invalid comment text"},
			{target: ErrCommentNotFound, status: http.StatusNotFound, message: "comment not found"},
			{target: ErrForbidden, status: http.StatusForbidden, message: "forbidden"},
			{target: ErrUnauthorized, status: http.StatusUnauthorized, message: "unauthorized"},
		}) {
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Comment updated",
		Data:    comment,
	})
}

func (handler *Handler) deleteComment(c *gin.Context) {
	currentUser, ok := getUser(c)
	if !ok {
		return
	}

	niin := c.Param("niin")
	commentID := c.Param("comment_id")

	comment, err := handler.service.DeleteComment(currentUser, niin, commentID)
	if err != nil {
		if respondError(c, err, []errorCase{
			{target: ErrInvalidNiin, status: http.StatusBadRequest, message: "invalid NIIN"},
			{target: ErrCommentNotFound, status: http.StatusNotFound, message: "comment not found"},
			{target: ErrForbidden, status: http.StatusForbidden, message: "forbidden"},
			{target: ErrUnauthorized, status: http.StatusUnauthorized, message: "unauthorized"},
		}) {
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Comment deleted",
		Data:    comment,
	})
}

func (handler *Handler) flagComment(c *gin.Context) {
	currentUser, ok := getUser(c)
	if !ok {
		return
	}

	niin := c.Param("niin")
	commentID := c.Param("comment_id")

	err := handler.service.FlagComment(currentUser, niin, commentID)
	if err != nil {
		if respondError(c, err, []errorCase{
			{target: ErrInvalidNiin, status: http.StatusBadRequest, message: "invalid NIIN"},
			{target: ErrCommentNotFound, status: http.StatusNotFound, message: "comment not found"},
			{target: ErrUnauthorized, status: http.StatusUnauthorized, message: "unauthorized"},
		}) {
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Comment flagged",
		Data:    gin.H{"comment_id": commentID},
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

type errorCase struct {
	target  error
	status  int
	message string
}

func respondError(c *gin.Context, err error, cases []errorCase) bool {
	for _, entry := range cases {
		if errors.Is(err, entry.target) {
			c.JSON(entry.status, gin.H{"message": entry.message})
			return true
		}
	}

	return false
}
