package images

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type ImageWithUser struct {
	model.MaterialImages
	Username *string
}

type Repository interface {
	Create(user *bootstrap.User, image model.MaterialImages) (*model.MaterialImages, error)
	GetByID(imageID string) (*model.MaterialImages, error)
	GetByNIIN(niin string, limit int, offset int) ([]ImageWithUser, int64, error)
	GetByUser(userID string, limit int, offset int) ([]ImageWithUser, int64, error)
	UpdateFlags(imageID string, flagCount int, isFlagged bool) error
	Delete(imageID string) error
	GetUsernameByUserID(userID string) (string, error)
}
