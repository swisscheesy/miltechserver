package serialized

import (
	"context"
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

func (service *ServiceImpl) GetByUser(ctx context.Context, user *bootstrap.User) ([]model.UserItemsSerialized, error) {
	if user == nil {
		return nil, shared.ErrUserNotFound
	}

	return service.repo.GetByUser(user)
}

func (service *ServiceImpl) Upsert(ctx context.Context, user *bootstrap.User, item model.UserItemsSerialized) error {
	if user == nil {
		return shared.ErrUserNotFound
	}

	return service.repo.Upsert(user, item)
}

func (service *ServiceImpl) UpsertBatch(ctx context.Context, user *bootstrap.User, items []model.UserItemsSerialized) error {
	if user == nil {
		return shared.ErrUserNotFound
	}

	return service.repo.UpsertBatch(user, items)
}

func (service *ServiceImpl) Delete(ctx context.Context, user *bootstrap.User, item model.UserItemsSerialized) error {
	if user == nil {
		return shared.ErrUserNotFound
	}

	if item.Image != nil && *item.Image != "" {
		err := service.imagesRepo.Delete(user, item.ID, "serialized")
		if err != nil {
			slog.Error("Failed to delete image from blob storage", "error", err, "user_id", user.UserID, "item_id", item.ID)
		}
	}

	return service.repo.Delete(user, item)
}

func (service *ServiceImpl) DeleteAll(ctx context.Context, user *bootstrap.User) error {
	if user == nil {
		return shared.ErrUserNotFound
	}

	items, err := service.repo.GetByUser(user)
	if err != nil {
		slog.Error("Failed to retrieve serialized items with images", "error", err, "user_id", user.UserID)
	} else {
		for _, item := range items {
			if item.Image == nil || *item.Image == "" {
				continue
			}

			err := service.imagesRepo.Delete(user, item.ID, "serialized")
			if err != nil {
				slog.Error("Failed to delete image from blob storage", "error", err, "user_id", user.UserID, "item_id", item.ID)
			}
		}
	}

	return service.repo.DeleteAll(user)
}
