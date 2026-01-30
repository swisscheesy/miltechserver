package votes

import (
	"fmt"

	"github.com/google/uuid"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/material_images/images"
	"miltechserver/bootstrap"
)

type ServiceImpl struct {
	repo       Repository
	imagesRepo images.Repository
}

func NewService(repo Repository, imagesRepo images.Repository) Service {
	return &ServiceImpl{
		repo:       repo,
		imagesRepo: imagesRepo,
	}
}

func (s *ServiceImpl) Vote(user *bootstrap.User, imageID string, voteType string) error {
	if voteType != "upvote" && voteType != "downvote" {
		return fmt.Errorf("invalid vote type: %s", voteType)
	}

	image, err := s.imagesRepo.GetByID(imageID)
	if err != nil {
		return fmt.Errorf("failed to get image: %w", err)
	}

	if image == nil {
		return fmt.Errorf("image not found")
	}

	imageUUID, err := uuid.Parse(imageID)
	if err != nil {
		return fmt.Errorf("invalid image ID: %w", err)
	}

	vote := model.MaterialImagesVotes{
		ImageID:  imageUUID,
		UserID:   user.UserID,
		VoteType: voteType,
	}

	err = s.repo.Upsert(vote)
	if err != nil {
		return fmt.Errorf("failed to save vote: %w", err)
	}

	err = s.repo.UpdateImageCounts(imageID)
	if err != nil {
		return fmt.Errorf("failed to update vote counts: %w", err)
	}

	return nil
}

func (s *ServiceImpl) RemoveVote(user *bootstrap.User, imageID string) error {
	err := s.repo.Delete(imageID, user.UserID)
	if err != nil {
		return fmt.Errorf("failed to remove vote: %w", err)
	}

	err = s.repo.UpdateImageCounts(imageID)
	if err != nil {
		return fmt.Errorf("failed to update vote counts: %w", err)
	}

	return nil
}
