package flags

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Service interface {
	Flag(user *bootstrap.User, imageID string, reason string, description string) error
	GetByImage(imageID string) ([]model.MaterialImagesFlags, error)
}
