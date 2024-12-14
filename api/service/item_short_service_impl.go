package service

import (
	"github.com/gin-gonic/gin"
	"miltechserver/api/repository"
	"miltechserver/model"
)

type ItemShortServiceImpl struct {
	ItemQueryRepository repository.ItemQueryRepository
}

func NewItemQueryServiceImpl(itemQueryRepository repository.ItemQueryRepository) *ItemShortServiceImpl {
	return &ItemShortServiceImpl{ItemQueryRepository: itemQueryRepository}
}

func (service *ItemShortServiceImpl) FindShortByNiin(ctx *gin.Context, niin string) (model.ShortItem, error) {
	return service.ItemQueryRepository.ShortItemSearchNiin(ctx, niin)

}

func (service *ItemShortServiceImpl) FindShortByPart(ctx *gin.Context, part string) ([]model.ShortItem, error) {
	return service.ItemQueryRepository.ShortItemSearchPart(ctx, part)
}
