package flags

import "miltechserver/.gen/miltech_ng/public/model"

type Repository interface {
	Create(flag model.MaterialImagesFlags) error
	GetByImage(imageID string) ([]model.MaterialImagesFlags, error)
	CountByImage(imageID string) (int, error)
}
