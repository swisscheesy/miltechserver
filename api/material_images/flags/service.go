package flags

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Service interface {
	Flag(ctx context.Context, user *bootstrap.User, imageID string, reason string, description string) error
	GetByImage(ctx context.Context, imageID string) ([]model.MaterialImagesFlags, error)
}
