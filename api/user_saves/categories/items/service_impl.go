package items

import (
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/user_saves/images"
	"miltechserver/api/user_saves/shared"
	"miltechserver/bootstrap"
)

type ServiceImpl struct {
	repo       Repository
	imagesRepo images.Repository
}

func NewService(repo Repository, imagesRepo images.Repository) *ServiceImpl {
	return &ServiceImpl{repo: repo, imagesRepo: imagesRepo}
}

func (service *ServiceImpl) GetByCategory(user *bootstrap.User, category model.UserItemCategory) ([]model.UserItemsCategorized, error) {
	if user == nil {
		return nil, shared.ErrUserNotFound
	}

	return service.repo.GetByCategory(user, category)
}

func (service *ServiceImpl) GetByUser(user *bootstrap.User) ([]model.UserItemsCategorized, error) {
	if user == nil {
		return nil, shared.ErrUserNotFound
	}

	return service.repo.GetByUser(user)
}

func (service *ServiceImpl) Upsert(user *bootstrap.User, item model.UserItemsCategorized) error {
	if user == nil {
		return shared.ErrUserNotFound
	}

	return service.repo.Upsert(user, item)
}

func (service *ServiceImpl) UpsertBatch(user *bootstrap.User, items []model.UserItemsCategorized) error {
	if user == nil {
		return shared.ErrUserNotFound
	}

	return service.repo.UpsertBatch(user, items)
}

func (service *ServiceImpl) Delete(user *bootstrap.User, item model.UserItemsCategorized) error {
	if user == nil {
		return shared.ErrUserNotFound
	}

	if item.Image != nil && *item.Image != "" {
		err := service.imagesRepo.Delete(user, item.ID, "categorized")
		if err != nil {
			slog.Error("Failed to delete image from blob storage", "error", err, "user_id", user.UserID, "item_id", item.ID)
		}
	}

	return service.repo.Delete(user, item)
}

func (service *ServiceImpl) DeleteAll(user *bootstrap.User) error {
	if user == nil {
		return shared.ErrUserNotFound
	}

	items, err := service.repo.GetByUser(user)
	if err != nil {
		slog.Error("Failed to retrieve categorized items with images", "error", err, "user_id", user.UserID)
	} else {
		for _, item := range items {
			if item.Image == nil || *item.Image == "" {
				continue
			}

			err := service.imagesRepo.Delete(user, item.ID, "categorized")
			if err != nil {
				slog.Error("Failed to delete images from blob storage", "error", err, "user_id", user.UserID)
			}
		}
	}

	return service.repo.DeleteAll(user)
}
