package images

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Service interface {
	Upload(user *bootstrap.User, niin string, imageData []byte, filename string) (*model.MaterialImages, error)
	GetByNIIN(niin string, page int, pageSize int, currentUser *bootstrap.User) ([]response.MaterialImageResponse, int64, error)
	GetByUser(userID string, page int, pageSize int, currentUser *bootstrap.User) ([]response.MaterialImageResponse, int64, error)
	GetByID(imageID string, currentUser *bootstrap.User) (*response.MaterialImageResponse, error)
	Delete(user *bootstrap.User, imageID string) error
}
