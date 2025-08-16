package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/google/uuid"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/repository"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

const (
	ContainerName = "material-images"
)

type MaterialImagesServiceImpl struct {
	repo       repository.MaterialImagesRepository
	blobClient *azblob.Client
	env        *bootstrap.Env
}

func NewMaterialImagesServiceImpl(
	repo repository.MaterialImagesRepository,
	blobClient *azblob.Client,
	env *bootstrap.Env,
) MaterialImagesService {
	return &MaterialImagesServiceImpl{
		repo:       repo,
		blobClient: blobClient,
		env:        env,
	}
}

// Helper function to determine if current user can delete an image
func (s *MaterialImagesServiceImpl) canUserDeleteImage(imageUserID string, currentUser *bootstrap.User) bool {
	// If no current user is logged in, can't delete
	if currentUser == nil {
		return false
	}
	
	// If user IDs match, user can delete their own image
	return imageUserID == currentUser.UserID
}

// Image operations

func (s *MaterialImagesServiceImpl) UploadImage(user *bootstrap.User, niin string, imageData []byte, filename string) (*model.MaterialImages, error) {
	// Validate NIIN length
	if len(niin) != 9 {
		return nil, fmt.Errorf("NIIN must be exactly 9 characters")
	}

	// Check rate limit
	canUpload, nextAllowedTime, err := s.repo.CheckUploadLimit(user.UserID, niin)
	if err != nil {
		return nil, fmt.Errorf("failed to check upload limit: %w", err)
	}

	if !canUpload {
		return nil, fmt.Errorf("rate limit exceeded. You can upload another image for this NIIN after %s", nextAllowedTime.Format(time.RFC3339))
	}

	// Generate unique blob name
	imageID := uuid.New()
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".jpg" // Default to jpg if no extension
	}
	blobName := fmt.Sprintf("%s%s", imageID.String(), ext)

	// Upload to Azure Blob Storage using direct client (same pattern as user saves)
	ctx := context.Background()
	_, err = s.blobClient.UploadBuffer(ctx, ContainerName, blobName, imageData, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to blob storage: %w", err)
	}

	// Construct blob URL manually (same pattern as user saves)
	blobURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s",
		s.env.BlobAccountName, ContainerName, blobName)

	// Determine content type from extension
	contentType := "image/jpeg" // Default
	switch ext {
	case ".png":
		contentType = "image/png"
	case ".webp":
		contentType = "image/webp"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	}

	// Create database record
	image := model.MaterialImages{
		ID:               imageID,
		Niin:             niin,
		UserID:           user.UserID,
		BlobName:         blobName,
		BlobURL:          blobURL,
		OriginalFilename: filename,
		FileSizeBytes:    int64(len(imageData)),
		MimeType:         contentType,
		UploadDate:       time.Now(),
		IsActive:         true,
		IsFlagged:        false,
		FlagCount:        0,
		DownvoteCount:    0,
		UpvoteCount:      0,
	}

	createdImage, err := s.repo.CreateImage(user, image)
	if err != nil {
		// Try to clean up blob if database insert fails (same pattern as user saves)
		ctx := context.Background()
		_, _ = s.blobClient.DeleteBlob(ctx, ContainerName, blobName, nil)
		return nil, fmt.Errorf("failed to save image record: %w", err)
	}

	// Update rate limit - this is critical for rate limiting to work
	err = s.repo.UpdateUploadLimit(user.UserID, niin)
	if err != nil {
		// Log the error and fail the upload since rate limiting won't work properly
		fmt.Printf("ERROR: failed to update upload limit: %v\n", err)
		// Try to clean up the created image record since we can't track rate limits
		_ = s.repo.DeleteImage(createdImage.ID.String())
		// Also try to clean up blob
		ctx := context.Background()
		_, _ = s.blobClient.DeleteBlob(ctx, ContainerName, blobName, nil)
		return nil, fmt.Errorf("failed to update upload rate limit: %w", err)
	}

	return createdImage, nil
}

func (s *MaterialImagesServiceImpl) GetImagesByNIIN(niin string, page int, pageSize int, currentUser *bootstrap.User) ([]response.MaterialImageResponse, int64, error) {
	// Calculate offset
	offset := (page - 1) * pageSize

	// Get images with usernames from repository using optimized query
	imagesWithUsers, totalCount, err := s.repo.GetImagesByNIIN(niin, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get images: %w", err)
	}

	// Convert to response format
	responseImages := make([]response.MaterialImageResponse, len(imagesWithUsers))
	for i, imgWithUser := range imagesWithUsers {
		// Download image data from blob storage
		imageData, err := s.downloadImageData(imgWithUser.BlobName)
		if err != nil {
			// Log error but don't fail the entire request - return empty image data
			fmt.Printf("Warning: failed to download image data for %s: %v\n", imgWithUser.BlobName, err)
			imageData = []byte{}
		}

		// Use username from the joined query result
		username := "Unknown"
		if imgWithUser.Username != nil {
			username = *imgWithUser.Username
		}

		responseImages[i] = response.MaterialImageResponse{
			ID:               imgWithUser.ID.String(),
			NIIN:             imgWithUser.Niin,
			UserID:           imgWithUser.UserID,
			Username:         username,
			ImageData:        imageData,
			OriginalFilename: imgWithUser.OriginalFilename,
			FileSizeBytes:    imgWithUser.FileSizeBytes,
			MimeType:         imgWithUser.MimeType,
			UploadDate:       imgWithUser.UploadDate,
			UpvoteCount:      int(imgWithUser.UpvoteCount),
			DownvoteCount:    int(imgWithUser.DownvoteCount),
			NetVotes:         int(*imgWithUser.NetVotes),
			IsFlagged:        imgWithUser.IsFlagged,
			CanDelete:        s.canUserDeleteImage(imgWithUser.UserID, currentUser),
		}
	}

	return responseImages, totalCount, nil
}

func (s *MaterialImagesServiceImpl) GetImagesByUser(userID string, page int, pageSize int, currentUser *bootstrap.User) ([]response.MaterialImageResponse, int64, error) {
	// Calculate offset
	offset := (page - 1) * pageSize

	// Get images with usernames from repository using optimized query
	imagesWithUsers, totalCount, err := s.repo.GetImagesByUser(userID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get images: %w", err)
	}

	// Convert to response format
	responseImages := make([]response.MaterialImageResponse, len(imagesWithUsers))
	for i, imgWithUser := range imagesWithUsers {
		// Download image data from blob storage
		imageData, err := s.downloadImageData(imgWithUser.BlobName)
		if err != nil {
			// Log error but don't fail the entire request - return empty image data
			fmt.Printf("Warning: failed to download image data for %s: %v\n", imgWithUser.BlobName, err)
			imageData = []byte{}
		}

		// Use username from the joined query result
		username := "Unknown"
		if imgWithUser.Username != nil {
			username = *imgWithUser.Username
		}

		responseImages[i] = response.MaterialImageResponse{
			ID:               imgWithUser.ID.String(),
			NIIN:             imgWithUser.Niin,
			UserID:           imgWithUser.UserID,
			Username:         username,
			ImageData:        imageData,
			OriginalFilename: imgWithUser.OriginalFilename,
			FileSizeBytes:    imgWithUser.FileSizeBytes,
			MimeType:         imgWithUser.MimeType,
			UploadDate:       imgWithUser.UploadDate,
			UpvoteCount:      int(imgWithUser.UpvoteCount),
			DownvoteCount:    int(imgWithUser.DownvoteCount),
			NetVotes:         int(*imgWithUser.NetVotes),
			IsFlagged:        imgWithUser.IsFlagged,
			CanDelete:        s.canUserDeleteImage(imgWithUser.UserID, currentUser),
		}
	}

	return responseImages, totalCount, nil
}

func (s *MaterialImagesServiceImpl) GetImageByID(imageID string, currentUser *bootstrap.User) (*response.MaterialImageResponse, error) {
	// Get image from repository
	image, err := s.repo.GetImageByID(imageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get image: %w", err)
	}

	if image == nil {
		return nil, fmt.Errorf("image not found")
	}

	// Download image data from blob storage
	imageData, err := s.downloadImageData(image.BlobName)
	if err != nil {
		return nil, fmt.Errorf("failed to download image data: %w", err)
	}

	// Get username for this user
	username, err := s.repo.GetUsernameByUserID(image.UserID)
	if err != nil {
		// Log error but don't fail the request - use fallback
		fmt.Printf("Warning: failed to get username for user %s: %v\n", image.UserID, err)
		username = "Unknown"
	}

	// Convert to response format
	responseImage := &response.MaterialImageResponse{
		ID:               image.ID.String(),
		NIIN:             image.Niin,
		UserID:           image.UserID,
		Username:         username,
		ImageData:        imageData,
		OriginalFilename: image.OriginalFilename,
		FileSizeBytes:    image.FileSizeBytes,
		MimeType:         image.MimeType,
		UploadDate:       image.UploadDate,
		UpvoteCount:      int(image.UpvoteCount),
		DownvoteCount:    int(image.DownvoteCount),
		NetVotes:         int(*image.NetVotes),
		IsFlagged:        image.IsFlagged,
		CanDelete:        s.canUserDeleteImage(image.UserID, currentUser),
	}

	return responseImage, nil
}

func (s *MaterialImagesServiceImpl) DeleteImage(user *bootstrap.User, imageID string) error {
	// Get image to verify ownership
	image, err := s.repo.GetImageByID(imageID)
	if err != nil {
		return fmt.Errorf("failed to get image: %w", err)
	}

	if image == nil {
		return fmt.Errorf("image not found")
	}

	// Check if user owns the image
	if image.UserID != user.UserID {
		return fmt.Errorf("unauthorized: you can only delete your own images")
	}

	// Soft delete in database
	err = s.repo.DeleteImage(imageID)
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	// Delete from Azure blob storage to free up space
	ctx := context.Background()
	_, err = s.blobClient.DeleteBlob(ctx, ContainerName, image.BlobName, nil)
	if err != nil {
		// Log error but don't fail the entire operation since the database record is already marked as deleted
		fmt.Printf("Warning: failed to delete blob %s from Azure storage: %v\n", image.BlobName, err)
		// The database deletion still succeeded, so the image won't be visible to users
	}

	return nil
}

// Vote operations

func (s *MaterialImagesServiceImpl) VoteOnImage(user *bootstrap.User, imageID string, voteType string) error {
	// Validate vote type
	if voteType != "upvote" && voteType != "downvote" {
		return fmt.Errorf("invalid vote type: %s", voteType)
	}

	// Check if image exists
	image, err := s.repo.GetImageByID(imageID)
	if err != nil {
		return fmt.Errorf("failed to get image: %w", err)
	}

	if image == nil {
		return fmt.Errorf("image not found")
	}

	// Parse image ID to UUID
	imageUUID, err := uuid.Parse(imageID)
	if err != nil {
		return fmt.Errorf("invalid image ID: %w", err)
	}

	// Create or update vote
	vote := model.MaterialImagesVotes{
		ImageID:  imageUUID,
		UserID:   user.UserID,
		VoteType: voteType,
	}

	err = s.repo.UpsertVote(vote)
	if err != nil {
		return fmt.Errorf("failed to save vote: %w", err)
	}

	// Update vote counts
	err = s.repo.UpdateImageVoteCounts(imageID)
	if err != nil {
		return fmt.Errorf("failed to update vote counts: %w", err)
	}

	return nil
}

func (s *MaterialImagesServiceImpl) RemoveVote(user *bootstrap.User, imageID string) error {
	// Delete vote
	err := s.repo.DeleteVote(imageID, user.UserID)
	if err != nil {
		return fmt.Errorf("failed to remove vote: %w", err)
	}

	// Update vote counts
	err = s.repo.UpdateImageVoteCounts(imageID)
	if err != nil {
		return fmt.Errorf("failed to update vote counts: %w", err)
	}

	return nil
}

// Flag operations

func (s *MaterialImagesServiceImpl) FlagImage(user *bootstrap.User, imageID string, reason string, description string) error {
	// Validate reason
	validReasons := map[string]bool{
		"Incorrect Item": true,
		"Inappropriate":  true,
		"Poor Quality":   true,
		"Other":          true,
	}

	if !validReasons[reason] {
		return fmt.Errorf("invalid flag reason: %s", reason)
	}

	// Check if image exists
	image, err := s.repo.GetImageByID(imageID)
	if err != nil {
		return fmt.Errorf("failed to get image: %w", err)
	}

	if image == nil {
		return fmt.Errorf("image not found")
	}

	// Parse image ID to UUID
	imageUUID, err := uuid.Parse(imageID)
	if err != nil {
		return fmt.Errorf("invalid image ID: %w", err)
	}

	// Create flag
	var descPtr *string
	if description != "" {
		descPtr = &description
	}

	flag := model.MaterialImagesFlags{
		ID:          uuid.New(),
		ImageID:     imageUUID,
		UserID:      user.UserID,
		Reason:      reason,
		Description: descPtr,
	}

	err = s.repo.CreateFlag(flag)
	if err != nil {
		// Check if it's a duplicate flag error
		if err.Error() == "pq: duplicate key value violates unique constraint \"unique_user_image_flag\"" {
			return fmt.Errorf("you have already flagged this image")
		}
		return fmt.Errorf("failed to flag image: %w", err)
	}

	return nil
}

func (s *MaterialImagesServiceImpl) GetImageFlags(imageID string) ([]model.MaterialImagesFlags, error) {
	flags, err := s.repo.GetFlagsByImage(imageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get flags: %w", err)
	}

	return flags, nil
}

// downloadImageData downloads image data from blob storage
func (s *MaterialImagesServiceImpl) downloadImageData(blobName string) ([]byte, error) {
	ctx := context.Background()

	// Download the blob
	response, err := s.blobClient.DownloadStream(ctx, ContainerName, blobName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download image from blob storage: %w", err)
	}
	defer response.Body.Close()

	// Read all image data
	imageData, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	return imageData, nil
}
