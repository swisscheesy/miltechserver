package controller

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/api/service"
	"miltechserver/bootstrap"
)

type MaterialImagesController struct {
	MaterialImagesService service.MaterialImagesService
}

func NewMaterialImagesController(materialImagesService service.MaterialImagesService) *MaterialImagesController {
	return &MaterialImagesController{
		MaterialImagesService: materialImagesService,
	}
}

// UploadImage handles image upload for a specific NIIN
func (controller *MaterialImagesController) UploadImage(c *gin.Context) {
	// Get current user from context
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	currentUser := user.(*bootstrap.User)

	// Get NIIN from form data
	niin := c.PostForm("niin")
	if niin == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "NIIN is required"})
		return
	}

	// Validate NIIN length
	if len(niin) != 9 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "NIIN must be exactly 9 characters"})
		return
	}

	// Get file from form - using "file" field name to match frontend
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get uploaded file"})
		return
	}
	defer file.Close()

	// Read file data into byte array
	imageData := make([]byte, header.Size)
	_, err = file.Read(imageData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file data"})
		return
	}

	// Upload image
	image, err := controller.MaterialImagesService.UploadImage(currentUser, niin, imageData, header.Filename)
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
			Username:         currentUser.Username, // Use current user's username
			ImageData:        imageData,            // Return the same image data we uploaded
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

// GetImagesByNIIN retrieves images for a specific NIIN with pagination
func (controller *MaterialImagesController) GetImagesByNIIN(c *gin.Context) {
	niin := c.Param("niin")
	if len(niin) != 9 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid NIIN format"})
		return
	}

	// Parse pagination parameters
	var req request.GetImagesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid query parameters: %v", err)})
		return
	}

	// Get current user from context (may be nil if not authenticated)
	var currentUser *bootstrap.User
	if user := c.ShouldBindJSON(&currentUser); user == nil {
		// User not logged in
	}

	// Get images
	images, totalCount, err := controller.MaterialImagesService.GetImagesByNIIN(niin, req.Page, req.PageSize, currentUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve images"})
		return
	}

	// Calculate total pages
	totalPages := int((totalCount + int64(req.PageSize) - 1) / int64(req.PageSize))

	c.JSON(http.StatusOK, response.PaginatedImagesResponse{
		Images:     images,
		TotalCount: totalCount,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	})
}

// GetImagesByUser retrieves images uploaded by a specific user with pagination
func (controller *MaterialImagesController) GetImagesByUser(c *gin.Context) {
	userID := c.Param("user_id")

	// Parse pagination parameters
	var req request.GetImagesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid query parameters: %v", err)})
		return
	}

	// Get current user from context (may be nil if not authenticated)
	var currentUser *bootstrap.User
	if user, exists := c.Get("user"); exists {
		currentUser = user.(*bootstrap.User)
	}

	// Get images
	images, totalCount, err := controller.MaterialImagesService.GetImagesByUser(userID, req.Page, req.PageSize, currentUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve images"})
		return
	}

	// Calculate total pages
	totalPages := int((totalCount + int64(req.PageSize) - 1) / int64(req.PageSize))

	c.JSON(http.StatusOK, response.PaginatedImagesResponse{
		Images:     images,
		TotalCount: totalCount,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	})
}

// GetImageByID retrieves a specific image by ID
func (controller *MaterialImagesController) GetImageByID(c *gin.Context) {
	imageID := c.Param("image_id")

	// Get current user from context (may be nil if not authenticated)
	var currentUser *bootstrap.User
	if user, exists := c.Get("user"); exists {
		currentUser = user.(*bootstrap.User)
	}

	// Get image
	image, err := controller.MaterialImagesService.GetImageByID(imageID, currentUser)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	c.JSON(http.StatusOK, image)
}

// DeleteImage allows a user to delete their own image
func (controller *MaterialImagesController) DeleteImage(c *gin.Context) {
	// Get current user from context
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	currentUser := user.(*bootstrap.User)

	imageID := c.Param("image_id")

	// Delete image
	err := controller.MaterialImagesService.DeleteImage(currentUser, imageID)
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

// VoteOnImage allows a user to vote on an image
func (controller *MaterialImagesController) VoteOnImage(c *gin.Context) {
	// Get current user from context
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	currentUser := user.(*bootstrap.User)

	imageID := c.Param("image_id")

	// Parse request body
	var req request.VoteImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	// Vote on image
	err := controller.MaterialImagesService.VoteOnImage(currentUser, imageID, req.VoteType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get updated image to return current vote counts
	updatedImage, err := controller.MaterialImagesService.GetImageByID(imageID, currentUser)
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

// RemoveVote allows a user to remove their vote from an image
func (controller *MaterialImagesController) RemoveVote(c *gin.Context) {
	// Get current user from context
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	currentUser := user.(*bootstrap.User)

	imageID := c.Param("image_id")

	// Remove vote
	err := controller.MaterialImagesService.RemoveVote(currentUser, imageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove vote"})
		return
	}

	// Get updated image to return current vote counts
	updatedImage, err := controller.MaterialImagesService.GetImageByID(imageID, currentUser)
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

// FlagImage allows a user to flag an image for review
func (controller *MaterialImagesController) FlagImage(c *gin.Context) {
	// Get current user from context
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	currentUser := user.(*bootstrap.User)

	imageID := c.Param("image_id")

	// Parse request body
	var req request.FlagImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	// Flag image
	err := controller.MaterialImagesService.FlagImage(currentUser, imageID, req.Reason, req.Description)
	if err != nil {
		if err.Error() == "you have already flagged this image" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get updated image to return current flag status
	updatedImage, err := controller.MaterialImagesService.GetImageByID(imageID, currentUser)
	var flagCount int
	var isFlagged bool
	if err == nil {
		// Convert to get flag count (this is a simplification)
		flagCount = 1 // We'd need to add flag count to the response
		isFlagged = updatedImage.IsFlagged
	}

	c.JSON(http.StatusOK, response.ImageFlagResponse{
		Success:   true,
		Message:   "Image flagged successfully",
		FlagCount: flagCount,
		IsFlagged: isFlagged,
	})
}

// GetImageFlags retrieves flags for a specific image (admin only)
func (controller *MaterialImagesController) GetImageFlags(c *gin.Context) {
	// TODO: Add admin authorization check
	imageID := c.Param("image_id")

	// Get flags
	flags, err := controller.MaterialImagesService.GetImageFlags(imageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve flags"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"flags": flags})
}
