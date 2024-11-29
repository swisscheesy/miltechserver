package service

import (
	"context"
	"miltechserver/api/repository"
	"miltechserver/model"
)

type ItemQueryServiceImpl struct {
	ItemQueryServiceRepository repository.ItemQueryRepository
}

func NewItemQueryServiceImpl(itemQueryServiceRepository repository.ItemQueryRepository) *ItemQueryServiceImpl {
	return &ItemQueryServiceImpl{ItemQueryServiceRepository: itemQueryServiceRepository}
}

func (service *ItemQueryServiceImpl) FindShortByNiin(ctx context.Context, niin string) (model.ShortItem, error) {
	return service.ItemQueryServiceRepository.ShortItemSearchNiin(ctx, niin)

}

func (service *ItemQueryServiceImpl) FindShortByPart(ctx context.Context, part string) ([]model.ShortItem, error) {
	return service.ItemQueryServiceRepository.ShortItemSearchPart(ctx, part)
}
