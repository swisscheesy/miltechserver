package repository

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
	"time"
)

// MaterialImageWithUser combines image data with username
type MaterialImageWithUser struct {
	model.MaterialImages
	Username *string
}

type MaterialImagesRepository interface {
	// Image operations
	CreateImage(user *bootstrap.User, image model.MaterialImages) (*model.MaterialImages, error)
	GetImageByID(imageID string) (*model.MaterialImages, error)
	GetImagesByNIIN(niin string, limit int, offset int) ([]MaterialImageWithUser, int64, error)
	GetImagesByUser(userID string, limit int, offset int) ([]MaterialImageWithUser, int64, error)
	UpdateImageFlags(imageID string, flagCount int, isFlagged bool) error
	DeleteImage(imageID string) error
	
	// Vote operations
	UpsertVote(vote model.MaterialImagesVotes) error
	DeleteVote(imageID string, userID string) error
	GetUserVoteForImage(imageID string, userID string) (*model.MaterialImagesVotes, error)
	UpdateImageVoteCounts(imageID string) error
	
	// Flag operations
	CreateFlag(flag model.MaterialImagesFlags) error
	GetFlagsByImage(imageID string) ([]model.MaterialImagesFlags, error)
	
	// Rate limiting
	CheckUploadLimit(userID string, niin string) (bool, *time.Time, error)
	UpdateUploadLimit(userID string, niin string) error
	CleanupOldLimits(olderThan time.Time) error
	
	// User lookup
	GetUsernameByUserID(userID string) (string, error)
}