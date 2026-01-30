package flags

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

	authRouter.POST("/material-images/:image_id/flag", handler.flag)
	authRouter.GET("/material-images/:image_id/flags", handler.getFlags)
}

func (h *Handler) flag(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	imageID := c.Param("image_id")

	var req request.FlagImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	err = h.service.Flag(user, imageID, req.Reason, req.Description)
	if err != nil {
		if err.Error() == "you have already flagged this image" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedImage, err := h.imagesService.GetByID(imageID, user)
	var flagCount int
	var isFlagged bool
	if err == nil {
		flagCount = 1
		isFlagged = updatedImage.IsFlagged
	}

	c.JSON(http.StatusOK, response.ImageFlagResponse{
		Success:   true,
		Message:   "Image flagged successfully",
		FlagCount: flagCount,
		IsFlagged: isFlagged,
	})
}

func (h *Handler) getFlags(c *gin.Context) {
	imageID := c.Param("image_id")

	flags, err := h.service.GetByImage(imageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve flags"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"flags": flags})
}
