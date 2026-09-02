package images

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Service interface {
	Upload(ctx context.Context, user *bootstrap.User, niin string, imageData []byte, filename string) (*model.MaterialImages, error)
	GetByNIIN(ctx context.Context, niin string, page int, pageSize int, currentUser *bootstrap.User) ([]response.MaterialImageResponse, int64, error)
	GetByUser(ctx context.Context, userID string, page int, pageSize int, currentUser *bootstrap.User) ([]response.MaterialImageResponse, int64, error)
	GetByID(ctx context.Context, imageID string, currentUser *bootstrap.User) (*response.MaterialImageResponse, error)
	Delete(ctx context.Context, user *bootstrap.User, imageID string) error
}
