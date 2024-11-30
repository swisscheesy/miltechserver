package service

import (
	"context"
	"miltechserver/api/repository"
	"miltechserver/model"
)

type ItemShortServiceImpl struct {
	ItemQueryServiceRepository repository.ItemQueryRepository
}

func NewItemQueryServiceImpl(itemQueryServiceRepository repository.ItemQueryRepository) *ItemShortServiceImpl {
	return &ItemShortServiceImpl{ItemQueryServiceRepository: itemQueryServiceRepository}
}

func (service *ItemShortServiceImpl) FindShortByNiin(ctx context.Context, niin string) (model.ShortItem, error) {
	return service.ItemQueryServiceRepository.ShortItemSearchNiin(ctx, niin)

}

func (service *ItemShortServiceImpl) FindShortByPart(ctx context.Context, part string) ([]model.ShortItem, error) {
	return service.ItemQueryServiceRepository.ShortItemSearchPart(ctx, part)
}

//func (service *ItemShortServiceImpl) FindAmdfData(ctx context.Context, niin string) (db.ArmyMasterDataFileModel, error) {
//	return service.ItemQueryServiceRepository.GetAmdfData(ctx, niin)
//}
