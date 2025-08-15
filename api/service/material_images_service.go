package service

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type MaterialImagesService interface {
	// Image operations
	UploadImage(user *bootstrap.User, niin string, imageData []byte, filename string) (*model.MaterialImages, error)
	GetImagesByNIIN(niin string, page int, pageSize int) ([]response.MaterialImageResponse, int64, error)
	GetImagesByUser(userID string, page int, pageSize int) ([]response.MaterialImageResponse, int64, error)
	GetImageByID(imageID string) (*response.MaterialImageResponse, error)
	DeleteImage(user *bootstrap.User, imageID string) error
	
	// Vote operations
	VoteOnImage(user *bootstrap.User, imageID string, voteType string) error
	RemoveVote(user *bootstrap.User, imageID string) error
	
	// Flag operations
	FlagImage(user *bootstrap.User, imageID string, reason string, description string) error
	GetImageFlags(imageID string) ([]model.MaterialImagesFlags, error)
}