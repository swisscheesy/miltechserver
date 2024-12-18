package service

import (
	"context"
	"miltechserver/api/repository"
	"miltechserver/model"
)

type ItemDetailedServiceImpl struct {
	ItemDetailedRepository repository.ItemDetailedRepository
}

func NewItemDetailedServiceImpl(itemDetailedServiceRepository repository.ItemDetailedRepository) *ItemDetailedServiceImpl {
	return &ItemDetailedServiceImpl{ItemDetailedRepository: itemDetailedServiceRepository}
}

func (service *ItemDetailedServiceImpl) FindDetailedItem(ctx context.Context, niin string) (model.DetailedItem, error) {
	amdf, _ := service.ItemDetailedRepository.GetAmdfData(ctx, niin)

	armypack, _ := service.ItemDetailedRepository.GetArmyPackagingAndFreight(ctx, niin)
	sarsscat, _ := service.ItemDetailedRepository.GetSarsscat(ctx, niin)
	identification, _ := service.ItemDetailedRepository.GetIdentification(ctx, niin)
	management, _ := service.ItemDetailedRepository.GetManagement(ctx, niin)
	reference, _ := service.ItemDetailedRepository.GetReference(ctx, niin)
	freight, _ := service.ItemDetailedRepository.GetFreight(ctx, niin)
	packaging, _ := service.ItemDetailedRepository.GetPackaging(ctx, niin)
	characteristics, _ := service.ItemDetailedRepository.GetCharacteristics(ctx, niin)
	disposition, _ := service.ItemDetailedRepository.GetDisposition(ctx, niin)

	return model.DetailedItem{
		Amdf:                    amdf,
		ArmyPackagingAndFreight: armypack,
		Sarsscat:                sarsscat,
		Identification:          identification,
		Management:              management,
		Reference:               reference,
		Freight:                 freight,
		Packaging:               packaging,
		Characteristics:         characteristics,
		Disposition:             disposition,
	}, nil

}

//func (service *ItemDetailedServiceImpl) GetAmdfData(ctx context.Context, niin string) (details.Amdf, error) {
//	return service.ItemDetailedRepository.GetAmdfData(ctx, niin)
//}
//
//func (service *ItemDetailedServiceImpl) GetArmyPackagingAndFreight(ctx context.Context, niin string) (details.ArmyPackagingAndFreight, error) {
//	return service.ItemDetailedRepository.GetArmyPackagingAndFreight(ctx, niin)
//}
//
//func (service *ItemDetailedServiceImpl) GetSarsscat(ctx context.Context, niin string) (details.Sarsscat, error) {
//	return service.ItemDetailedRepository.GetSarsscat(ctx, niin)
//}
//
//func (service *ItemDetailedServiceImpl) GetIdentification(ctx context.Context, niin string) (details.Identification, error) {
//	return service.ItemDetailedRepository.GetIdentification(ctx, niin)
//}
//
//func (service *ItemDetailedServiceImpl) GetManagement(ctx context.Context, niin string) (details.Management, error) {
//	return service.ItemDetailedRepository.GetManagement(ctx, niin)
//}
//
//func (service *ItemDetailedServiceImpl) GetReference(ctx context.Context, niin string) (details.Reference, error) {
//	return service.ItemDetailedRepository.GetReference(ctx, niin)
//}
//
//func (service *ItemDetailedServiceImpl) GetFreight(ctx context.Context, niin string) (details.Freight, error) {
//	return service.ItemDetailedRepository.GetFreight(ctx, niin)
//}
//
//func (service *ItemDetailedServiceImpl) GetPackaging(ctx context.Context, niin string) (details.Packaging, error) {
//	return service.ItemDetailedRepository.GetPackaging(ctx, niin)
//}
//
//func (service *ItemDetailedServiceImpl) GetCharacteristics(ctx context.Context, niin string) (details.Characteristics, error) {
//	return service.ItemDetailedRepository.GetCharacteristics(ctx, niin)
//}
//
//func (service *ItemDetailedServiceImpl) GetDisposition(ctx context.Context, niin string) (details.Disposition, error) {
//	return service.ItemDetailedRepository.GetDisposition(ctx, niin)
//}
