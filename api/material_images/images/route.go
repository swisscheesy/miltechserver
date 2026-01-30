package images

import (
	"fmt"
	"net/http"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"

	"miltechserver/api/material_images/shared"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Handler struct {
	service Service
}

func RegisterRoutes(publicRouter *gin.RouterGroup, authRouter *gin.RouterGroup, service Service, authClient *auth.Client) {
	_ = authClient
	handler := Handler{service: service}

	publicRouter.GET("/material-images/niin/:niin", handler.getByNIIN)
	publicRouter.GET("/material-images/:image_id", handler.getByID)

	authRouter.POST("/material-images/upload", handler.upload)
	authRouter.DELETE("/material-images/:image_id", handler.delete)
	authRouter.GET("/material-images/user/:user_id", handler.getByUser)
}

func (h *Handler) upload(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	niin := c.PostForm("niin")
	if niin == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "NIIN is required"})
		return
	}

	if len(niin) != 9 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "NIIN must be exactly 9 characters"})
		return
	}

	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get uploaded file"})
		return
	}
	defer file.Close()

	imageData := make([]byte, header.Size)
	_, err = file.Read(imageData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file data"})
		return
	}

	image, err := h.service.Upload(user, niin, imageData, header.Filename)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response.ImageUploadResponse{
		Success: true,
		Message: "Image uploaded successfully",
		Image: &response.MaterialImageResponse{
			ID:               image.ID.String(),
			NIIN:             image.Niin,
			UserID:           image.UserID,
			Username:         user.Username,
			ImageData:        imageData,
			OriginalFilename: image.OriginalFilename,
			FileSizeBytes:    image.FileSizeBytes,
			MimeType:         image.MimeType,
			UploadDate:       image.UploadDate,
			UpvoteCount:      int(image.UpvoteCount),
			DownvoteCount:    int(image.DownvoteCount),
			NetVotes:         int(*image.NetVotes),
			IsFlagged:        image.IsFlagged,
			CanDelete:        true,
		},
	})
}

func (h *Handler) getByNIIN(c *gin.Context) {
	niin := c.Param("niin")
	if len(niin) != 9 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid NIIN format"})
		return
	}

	var req request.GetImagesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid query parameters: %v", err)})
		return
	}

	var currentUser *bootstrap.User
	currentUser = shared.GetOptionalUserFromContext(c)

	images, totalCount, err := h.service.GetByNIIN(niin, req.Page, req.PageSize, currentUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve images"})
		return
	}

	totalPages := int((totalCount + int64(req.PageSize) - 1) / int64(req.PageSize))

	c.JSON(http.StatusOK, response.PaginatedImagesResponse{
		Images:     images,
		TotalCount: totalCount,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	})
}

func (h *Handler) getByUser(c *gin.Context) {
	userID := c.Param("user_id")

	var req request.GetImagesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid query parameters: %v", err)})
		return
	}

	var currentUser *bootstrap.User
	if user, exists := c.Get("user"); exists {
		currentUser = user.(*bootstrap.User)
	}

	images, totalCount, err := h.service.GetByUser(userID, req.Page, req.PageSize, currentUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve images"})
		return
	}

	totalPages := int((totalCount + int64(req.PageSize) - 1) / int64(req.PageSize))

	c.JSON(http.StatusOK, response.PaginatedImagesResponse{
		Images:     images,
		TotalCount: totalCount,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	})
}

func (h *Handler) getByID(c *gin.Context) {
	imageID := c.Param("image_id")

	var currentUser *bootstrap.User
	if user, exists := c.Get("user"); exists {
		currentUser = user.(*bootstrap.User)
	}

	image, err := h.service.GetByID(imageID, currentUser)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	c.JSON(http.StatusOK, image)
}

func (h *Handler) delete(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	imageID := c.Param("image_id")

	err = h.service.Delete(user, imageID)
	if err != nil {
		if err.Error() == "unauthorized: you can only delete your own images" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete image"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Image deleted successfully"})
}
