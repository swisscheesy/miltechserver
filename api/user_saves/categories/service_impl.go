package categories

import (
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/user_saves/categories/items"
	"miltechserver/api/user_saves/images"
	"miltechserver/api/user_saves/shared"
	"miltechserver/bootstrap"
)

type ServiceImpl struct {
	repo       Repository
	itemsRepo  items.Repository
	imagesRepo images.Repository
}

func NewService(repo Repository, itemsRepo items.Repository, imagesRepo images.Repository) *ServiceImpl {
	return &ServiceImpl{repo: repo, itemsRepo: itemsRepo, imagesRepo: imagesRepo}
}

func (service *ServiceImpl) GetByUser(user *bootstrap.User) ([]model.UserItemCategory, error) {
	if user == nil {
		return nil, shared.ErrUserNotFound
	}

	return service.repo.GetByUser(user)
}

func (service *ServiceImpl) Upsert(user *bootstrap.User, category model.UserItemCategory) error {
	if user == nil {
		return shared.ErrUserNotFound
	}

	return service.repo.Upsert(user, category)
}

func (service *ServiceImpl) Delete(user *bootstrap.User, category model.UserItemCategory) error {
	if user == nil {
		return shared.ErrUserNotFound
	}

	categorizedItems, err := service.itemsRepo.GetByCategory(user, category)
	if err != nil {
		slog.Error("Failed to retrieve categorized items with images", "error", err, "user_id", user.UserID, "category_id", category.ID)
	} else {
		for _, item := range categorizedItems {
			if item.Image == nil || *item.Image == "" {
				continue
			}
			err := service.imagesRepo.Delete(user, item.ID, "categorized")
			if err != nil {
				slog.Error("Failed to delete categorized item images from blob storage", "error", err, "user_id", user.UserID, "category_id", category.ID)
			}
		}
	}

	if category.Image != nil && *category.Image != "" {
		err := service.imagesRepo.Delete(user, category.ID, "category")
		if err != nil {
			slog.Error("Failed to delete category image from blob storage", "error", err, "user_id", user.UserID, "category_id", category.ID)
		}
	}

	err = service.repo.Delete(user, category)
	if err != nil {
		return err
	}

	slog.Info("item category, categorized items and associated images deleted", "user_id", user.UserID, "category_uuid", category.ID)
	return nil
}

func (service *ServiceImpl) DeleteAll(user *bootstrap.User) error {
	if user == nil {
		return shared.ErrUserNotFound
	}

	categorizedItems, err := service.itemsRepo.GetByUser(user)
	if err != nil {
		slog.Error("Failed to retrieve categorized items with images", "error", err, "user_id", user.UserID)
	}

	categories, err := service.repo.GetByUser(user)
	if err != nil {
		slog.Error("Failed to retrieve categories with images", "error", err, "user_id", user.UserID)
	}

	for _, item := range categorizedItems {
		if item.Image == nil || *item.Image == "" {
			continue
		}

		err := service.imagesRepo.Delete(user, item.ID, "categorized")
		if err != nil {
			slog.Error("Failed to delete images from blob storage", "error", err, "user_id", user.UserID)
		}
	}

	for _, category := range categories {
		if category.Image == nil || *category.Image == "" {
			continue
		}

		err := service.imagesRepo.Delete(user, category.ID, "category")
		if err != nil {
			slog.Error("Failed to delete images from blob storage", "error", err, "user_id", user.UserID)
		}
	}

	err = service.repo.DeleteAll(user)
	if err != nil {
		return err
	}

	slog.Info("all item categories, categorized items and associated images deleted", "user_id", user.UserID)
	return nil
}
