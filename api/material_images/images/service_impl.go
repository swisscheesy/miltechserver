package images

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/google/uuid"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/material_images/ratelimit"
	"miltechserver/api/material_images/shared"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type VoteRepository interface {
	GetUserVote(imageID string, userID string) (*model.MaterialImagesVotes, error)
}

// ServiceImpl implements image operations.
type ServiceImpl struct {
	repo          Repository
	rateLimitRepo ratelimit.Repository
	blobStorage   *shared.BlobStorage
	env           *bootstrap.Env
	voteRepo      VoteRepository
}

func NewService(repo Repository, rateLimitRepo ratelimit.Repository, voteRepo VoteRepository, blobClient *azblob.Client, env *bootstrap.Env) Service {
	return &ServiceImpl{
		repo:          repo,
		rateLimitRepo: rateLimitRepo,
		blobStorage:   shared.NewBlobStorage(blobClient),
		env:           env,
		voteRepo:      voteRepo,
	}
}

func (s *ServiceImpl) canUserDeleteImage(imageUserID string, currentUser *bootstrap.User) bool {
	if currentUser == nil {
		return false
	}

	return imageUserID == currentUser.UserID
}

func (s *ServiceImpl) Upload(user *bootstrap.User, niin string, imageData []byte, filename string) (*model.MaterialImages, error) {
	if len(niin) != 9 {
		return nil, fmt.Errorf("NIIN must be exactly 9 characters")
	}

	canUpload, nextAllowedTime, err := s.rateLimitRepo.CheckLimit(user.UserID, niin)
	if err != nil {
		return nil, fmt.Errorf("failed to check upload limit: %w", err)
	}

	if !canUpload {
		return nil, fmt.Errorf("rate limit exceeded. You can upload another image for this NIIN after %s", nextAllowedTime.Format(time.RFC3339))
	}

	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".jpg"
	}

	imageID := uuid.New()
	blobName := fmt.Sprintf("%s%s", imageID.String(), ext)
	err = s.blobStorage.Upload(blobName, imageData)
	if err != nil {
		return nil, err
	}

	blobURL := s.blobStorage.GetURL(blobName, s.env.BlobAccountName)

	contentType := "image/jpeg"
	switch ext {
	case ".png":
		contentType = "image/png"
	case ".webp":
		contentType = "image/webp"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	}

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

	createdImage, err := s.repo.Create(user, image)
	if err != nil {
		_ = s.blobStorage.Delete(blobName)
		return nil, fmt.Errorf("failed to save image record: %w", err)
	}

	err = s.rateLimitRepo.UpdateLimit(user.UserID, niin)
	if err != nil {
		fmt.Printf("ERROR: failed to update upload limit: %v\n", err)
		_ = s.repo.Delete(createdImage.ID.String())
		_ = s.blobStorage.Delete(blobName)
		return nil, fmt.Errorf("failed to update upload rate limit: %w", err)
	}

	return createdImage, nil
}

func (s *ServiceImpl) GetByNIIN(niin string, page int, pageSize int, currentUser *bootstrap.User) ([]response.MaterialImageResponse, int64, error) {
	offset := (page - 1) * pageSize

	imagesWithUsers, totalCount, err := s.repo.GetByNIIN(niin, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get images: %w", err)
	}

	responseImages := make([]response.MaterialImageResponse, len(imagesWithUsers))
	for i, imgWithUser := range imagesWithUsers {
		imageData, err := s.blobStorage.Download(imgWithUser.BlobName)
		if err != nil {
			fmt.Printf("Warning: failed to download image data for %s: %v\n", imgWithUser.BlobName, err)
			imageData = []byte{}
		}

		username := "Unknown"
		if imgWithUser.Username != nil {
			username = *imgWithUser.Username
		}

		userVote := s.getUserVote(imgWithUser.ID.String(), currentUser)

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
			UserVote:         userVote,
			CanDelete:        s.canUserDeleteImage(imgWithUser.UserID, currentUser),
		}
	}

	return responseImages, totalCount, nil
}

func (s *ServiceImpl) GetByUser(userID string, page int, pageSize int, currentUser *bootstrap.User) ([]response.MaterialImageResponse, int64, error) {
	offset := (page - 1) * pageSize

	imagesWithUsers, totalCount, err := s.repo.GetByUser(userID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get images: %w", err)
	}

	responseImages := make([]response.MaterialImageResponse, len(imagesWithUsers))
	for i, imgWithUser := range imagesWithUsers {
		imageData, err := s.blobStorage.Download(imgWithUser.BlobName)
		if err != nil {
			fmt.Printf("Warning: failed to download image data for %s: %v\n", imgWithUser.BlobName, err)
			imageData = []byte{}
		}

		username := "Unknown"
		if imgWithUser.Username != nil {
			username = *imgWithUser.Username
		}

		userVote := s.getUserVote(imgWithUser.ID.String(), currentUser)

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
			UserVote:         userVote,
			CanDelete:        s.canUserDeleteImage(imgWithUser.UserID, currentUser),
		}
	}

	return responseImages, totalCount, nil
}

func (s *ServiceImpl) GetByID(imageID string, currentUser *bootstrap.User) (*response.MaterialImageResponse, error) {
	image, err := s.repo.GetByID(imageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get image: %w", err)
	}

	if image == nil {
		return nil, fmt.Errorf("image not found")
	}

	imageData, err := s.blobStorage.Download(image.BlobName)
	if err != nil {
		return nil, fmt.Errorf("failed to download image data: %w", err)
	}

	username, err := s.repo.GetUsernameByUserID(image.UserID)
	if err != nil {
		fmt.Printf("Warning: failed to get username for user %s: %v\n", image.UserID, err)
		username = "Unknown"
	}

	userVote := s.getUserVote(image.ID.String(), currentUser)

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
		UserVote:         userVote,
		CanDelete:        s.canUserDeleteImage(image.UserID, currentUser),
	}

	return responseImage, nil
}

func (s *ServiceImpl) Delete(user *bootstrap.User, imageID string) error {
	image, err := s.repo.GetByID(imageID)
	if err != nil {
		return fmt.Errorf("failed to get image: %w", err)
	}

	if image == nil {
		return fmt.Errorf("image not found")
	}

	if image.UserID != user.UserID {
		return fmt.Errorf("unauthorized: you can only delete your own images")
	}

	err = s.repo.Delete(imageID)
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	err = s.blobStorage.Delete(image.BlobName)
	if err != nil {
		fmt.Printf("Warning: failed to delete blob %s from Azure storage: %v\n", image.BlobName, err)
	}

	return nil
}

func (s *ServiceImpl) getUserVote(imageID string, currentUser *bootstrap.User) *string {
	if currentUser == nil || s.voteRepo == nil {
		return nil
	}

	vote, err := s.voteRepo.GetUserVote(imageID, currentUser.UserID)
	if err != nil {
		fmt.Printf("Warning: failed to get user vote for image %s: %v\n", imageID, err)
		return nil
	}
	if vote == nil {
		return nil
	}

	return &vote.VoteType
}
