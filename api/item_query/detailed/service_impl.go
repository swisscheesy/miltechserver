package detailed

import (
	"context"

	"miltechserver/api/response"
)

type ServiceImpl struct {
	repo  Repository
	cache *Cache
}

func NewService(repo Repository) *ServiceImpl {
	return &ServiceImpl{
		repo:  repo,
		cache: NewCache(24 * 60 * 60), // 24 hour TTL in seconds
	}
}

func (service *ServiceImpl) FindDetailedItem(ctx context.Context, niin string) (response.DetailedResponse, error) {
	// Check cache first
	if cached, ok := service.cache.Get(niin); ok {
		return cached, nil
	}

	// Cache miss - fetch from database
	data, err := service.repo.GetDetailedItemData(ctx, niin)
	if err != nil {
		return response.DetailedResponse{}, err
	}

	// Store in cache
	service.cache.Set(niin, data)
	return data, nil
}
