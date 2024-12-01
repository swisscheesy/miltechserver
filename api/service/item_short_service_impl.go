package service

import (
	"context"
	"miltechserver/api/repository"
	"miltechserver/model"
)

type ItemShortServiceImpl struct {
	ItemQueryRepository repository.ItemQueryRepository
}

func NewItemQueryServiceImpl(itemQueryRepository repository.ItemQueryRepository) *ItemShortServiceImpl {
	return &ItemShortServiceImpl{ItemQueryRepository: itemQueryRepository}
}

func (service *ItemShortServiceImpl) FindShortByNiin(ctx context.Context, niin string) (model.ShortItem, error) {
	return service.ItemQueryRepository.ShortItemSearchNiin(ctx, niin)

}

func (service *ItemShortServiceImpl) FindShortByPart(ctx context.Context, part string) ([]model.ShortItem, error) {
	return service.ItemQueryRepository.ShortItemSearchPart(ctx, part)
}
