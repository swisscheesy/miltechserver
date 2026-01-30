package votes

import "miltechserver/.gen/miltech_ng/public/model"

type Repository interface {
	Upsert(vote model.MaterialImagesVotes) error
	Delete(imageID string, userID string) error
	GetUserVote(imageID string, userID string) (*model.MaterialImagesVotes, error)
	UpdateImageCounts(imageID string) error
}
