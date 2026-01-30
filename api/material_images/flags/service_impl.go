package flags

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

func (s *ServiceImpl) Flag(user *bootstrap.User, imageID string, reason string, description string) error {
	validReasons := map[string]bool{
		"Incorrect Item": true,
		"Inappropriate":  true,
		"Poor Quality":   true,
		"Other":          true,
	}

	if !validReasons[reason] {
		return fmt.Errorf("invalid flag reason: %s", reason)
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

	var descPtr *string
	if description != "" {
		descPtr = &description
	}

	flag := model.MaterialImagesFlags{
		ID:          uuid.New(),
		ImageID:     imageUUID,
		UserID:      user.UserID,
		Reason:      reason,
		Description: descPtr,
	}

	err = s.repo.Create(flag)
	if err != nil {
		if err.Error() == "pq: duplicate key value violates unique constraint \"unique_user_image_flag\"" {
			return fmt.Errorf("you have already flagged this image")
		}
		return fmt.Errorf("failed to flag image: %w", err)
	}

	flagCount, err := s.repo.CountByImage(imageID)
	if err != nil {
		return fmt.Errorf("failed to get flags: %w", err)
	}

	isFlagged := flagCount >= 1
	err = s.imagesRepo.UpdateFlags(imageID, flagCount, isFlagged)
	if err != nil {
		return fmt.Errorf("failed to update image flag status: %w", err)
	}

	return nil
}

func (s *ServiceImpl) GetByImage(imageID string) ([]model.MaterialImagesFlags, error) {
	flags, err := s.repo.GetByImage(imageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get flags: %w", err)
	}

	return flags, nil
}
