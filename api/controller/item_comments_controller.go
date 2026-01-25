package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/api/service"
	"miltechserver/bootstrap"
)

type ItemCommentsController struct {
	ItemCommentsService service.ItemCommentsService
}

func NewItemCommentsController(itemCommentsService service.ItemCommentsService) *ItemCommentsController {
	return &ItemCommentsController{ItemCommentsService: itemCommentsService}
}

func (controller *ItemCommentsController) GetCommentsByNiin(c *gin.Context) {
	niin := c.Param("niin")
	comments, err := controller.ItemCommentsService.GetCommentsByNiin(niin)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidNiin):
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid NIIN"})
		default:
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Comments retrieved",
		Data:    comments,
	})
}

func (controller *ItemCommentsController) CreateComment(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	currentUser := user.(*bootstrap.User)

	niin := c.Param("niin")

	var req request.ItemCommentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	comment, err := controller.ItemCommentsService.CreateComment(currentUser, niin, req.Text, req.ParentID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidNiin):
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid NIIN"})
		case errors.Is(err, service.ErrInvalidText):
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid comment text"})
		case errors.Is(err, service.ErrInvalidParent):
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid parent comment"})
		case errors.Is(err, service.ErrUnauthorized):
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		default:
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}

	c.JSON(http.StatusCreated, response.StandardResponse{
		Status:  http.StatusCreated,
		Message: "Comment created",
		Data:    comment,
	})
}

func (controller *ItemCommentsController) UpdateComment(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	currentUser := user.(*bootstrap.User)

	niin := c.Param("niin")
	commentID := c.Param("comment_id")

	var req request.ItemCommentUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	comment, err := controller.ItemCommentsService.UpdateComment(currentUser, niin, commentID, req.Text)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidNiin):
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid NIIN"})
		case errors.Is(err, service.ErrInvalidText):
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid comment text"})
		case errors.Is(err, service.ErrCommentNotFound):
			c.JSON(http.StatusNotFound, gin.H{"message": "comment not found"})
		case errors.Is(err, service.ErrForbidden):
			c.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
		case errors.Is(err, service.ErrUnauthorized):
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		default:
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Comment updated",
		Data:    comment,
	})
}

func (controller *ItemCommentsController) DeleteComment(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	currentUser := user.(*bootstrap.User)

	niin := c.Param("niin")
	commentID := c.Param("comment_id")

	comment, err := controller.ItemCommentsService.DeleteComment(currentUser, niin, commentID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidNiin):
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid NIIN"})
		case errors.Is(err, service.ErrCommentNotFound):
			c.JSON(http.StatusNotFound, gin.H{"message": "comment not found"})
		case errors.Is(err, service.ErrForbidden):
			c.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
		case errors.Is(err, service.ErrUnauthorized):
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		default:
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Comment deleted",
		Data:    comment,
	})
}

func (controller *ItemCommentsController) FlagComment(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	currentUser := user.(*bootstrap.User)

	niin := c.Param("niin")
	commentID := c.Param("comment_id")

	err := controller.ItemCommentsService.FlagComment(currentUser, niin, commentID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidNiin):
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid NIIN"})
		case errors.Is(err, service.ErrCommentNotFound):
			c.JSON(http.StatusNotFound, gin.H{"message": "comment not found"})
		case errors.Is(err, service.ErrUnauthorized):
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		default:
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Comment flagged",
		Data:    gin.H{"comment_id": commentID},
	})
}
