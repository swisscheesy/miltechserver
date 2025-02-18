package service

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/repository"
)

type ItemShortServiceImpl struct {
	ItemQueryRepository repository.ItemQueryRepository
}

func NewItemQueryServiceImpl(itemQueryRepository repository.ItemQueryRepository) *ItemShortServiceImpl {
	return &ItemShortServiceImpl{ItemQueryRepository: itemQueryRepository}
}

func (service *ItemShortServiceImpl) FindShortByNiin(niin string) (model.NiinLookup, error) {
	val, err := service.ItemQueryRepository.ShortItemSearchNiin(niin)
	if err != nil {
		return model.NiinLookup{}, err
	} else {
		return val, nil
	}
}

func (service *ItemShortServiceImpl) FindShortByPart(part string) ([]model.NiinLookup, error) {
	return service.ItemQueryRepository.ShortItemSearchPart(part)
}
