package votes

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"miltechserver/api/material_images/images"
	"miltechserver/api/material_images/shared"
	"miltechserver/api/request"
	"miltechserver/api/response"
)

type Handler struct {
	service       Service
	imagesService images.Service
}

func RegisterRoutes(authRouter *gin.RouterGroup, service Service, imagesService images.Service) {
	handler := Handler{service: service, imagesService: imagesService}

	authRouter.POST("/material-images/:image_id/vote", handler.vote)
	authRouter.DELETE("/material-images/:image_id/vote", handler.removeVote)
}

func (h *Handler) vote(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	imageID := c.Param("image_id")

	var req request.VoteImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	err = h.service.Vote(user, imageID, req.VoteType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedImage, err := h.imagesService.GetByID(imageID, user)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Vote recorded successfully"})
		return
	}

	c.JSON(http.StatusOK, response.ImageVoteResponse{
		Success:       true,
		Message:       "Vote recorded successfully",
		UpvoteCount:   updatedImage.UpvoteCount,
		DownvoteCount: updatedImage.DownvoteCount,
		NetVotes:      updatedImage.NetVotes,
	})
}

func (h *Handler) removeVote(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	imageID := c.Param("image_id")

	err = h.service.RemoveVote(user, imageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove vote"})
		return
	}

	updatedImage, err := h.imagesService.GetByID(imageID, user)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Vote removed successfully"})
		return
	}

	c.JSON(http.StatusOK, response.ImageVoteResponse{
		Success:       true,
		Message:       "Vote removed successfully",
		UpvoteCount:   updatedImage.UpvoteCount,
		DownvoteCount: updatedImage.DownvoteCount,
		NetVotes:      updatedImage.NetVotes,
	})
}
