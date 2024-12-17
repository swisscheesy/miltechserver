package service

import (
	"miltechserver/api/repository"
)

type ItemDetailedServiceImpl struct {
	ItemDetailedRepository repository.ItemDetailedRepository
}

func NewItemDetailedServiceImpl(itemDetailedServiceRepository repository.ItemDetailedRepository) *ItemDetailedServiceImpl {
	return &ItemDetailedServiceImpl{ItemDetailedRepository: itemDetailedServiceRepository}
}

//func (service *ItemDetailedServiceImpl) FindDetailedItem(ctx *gin.Context, niin string) (model.DetailedItem, error) {
//	amdf, _ := service.ItemDetailedRepository.GetAmdfData(ctx, niin)
//
//	armyPack, _ := service.ItemDetailedRepository.GetArmyPackagingAndFreight(ctx, niin)
//	sarsscat, _ := service.ItemDetailedRepository.GetSarsscat(ctx, niin)
//	identification, _ := service.ItemDetailedRepository.GetIdentification(ctx, niin)
//	management, _ := service.ItemDetailedRepository.GetManagement(ctx, niin)
//	reference, _ := service.ItemDetailedRepository.GetReference(ctx, niin)
//	freight, _ := service.ItemDetailedRepository.GetFreight(ctx, niin)
//	packaging, _ := service.ItemDetailedRepository.GetPackaging(ctx, niin)
//	characteristics, _ := service.ItemDetailedRepository.GetCharacteristics(ctx, niin)
//	disposition, _ := service.ItemDetailedRepository.GetDisposition(ctx, niin)
//
//	return model.DetailedItem{
//		Amdf:                    amdf,
//		ArmyPackagingAndFreight: armyPack,
//		Sarsscat:                sarsscat,
//		Identification:          identification,
//		Management:              management,
//		Reference:               reference,
//		Freight:                 freight,
//		Packaging:               packaging,
//		Characteristics:         characteristics,
//		Disposition:             disposition,
//	}, nil

//}
