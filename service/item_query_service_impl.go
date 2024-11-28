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
	item, err := service.ItemQueryServiceRepository.ShortItemSearchNiin(ctx, niin)

	if err != nil {
		return item, err
	} else {
		return item, nil
	}
}

//func (service *ItemQueryServiceImpl) FindShortByPart(ctx context.Context, part string) response.StandardResponse {
//	item, err := service.ItemQueryServiceRepository.ShortItemSearchPart(ctx, part)
//	if err != nil {
//		return response.StandardResponse{Message: err.Error()}
//	}
//	return response.StandardResponse{Data: item}
//}
