package service

import (
	"context"
	"miltechserver/api/repository"
	"miltechserver/model/details"
)

type ItemDetailedServiceImpl struct {
	ItemDetailedServiceRepository repository.ItemDetailedRepository
}

func NewItemDetailedServiceImpl(itemDetailedServiceRepository repository.ItemDetailedRepository) *ItemDetailedServiceImpl {
	return &ItemDetailedServiceImpl{ItemDetailedServiceRepository: itemDetailedServiceRepository}
}

func (service *ItemDetailedServiceImpl) GetAmdfData(ctx context.Context, niin string) (details.Amdf, error) {
	return service.ItemDetailedServiceRepository.GetAmdfData(ctx, niin)
}

func (service *ItemDetailedServiceImpl) GetArmyPackagingAndFreight(ctx context.Context, niin string) (details.ArmyPackagingAndFreight, error) {
	return service.ItemDetailedServiceRepository.GetArmyPackagingAndFreight(ctx, niin)
}

func (service *ItemDetailedServiceImpl) GetSarsscat(ctx context.Context, niin string) (details.Sarsscat, error) {
	return service.ItemDetailedServiceRepository.GetSarsscat(ctx, niin)
}

func (service *ItemDetailedServiceImpl) GetIdentification(ctx context.Context, niin string) (details.Identification, error) {
	return service.ItemDetailedServiceRepository.GetIdentification(ctx, niin)
}

func (service *ItemDetailedServiceImpl) GetManagement(ctx context.Context, niin string) (details.Management, error) {
	return service.ItemDetailedServiceRepository.GetManagement(ctx, niin)
}

func (service *ItemDetailedServiceImpl) GetReference(ctx context.Context, niin string) (details.Reference, error) {
	return service.ItemDetailedServiceRepository.GetReference(ctx, niin)
}

func (service *ItemDetailedServiceImpl) GetFreight(ctx context.Context, niin string) (details.Freight, error) {
	return service.ItemDetailedServiceRepository.GetFreight(ctx, niin)
}

func (service *ItemDetailedServiceImpl) GetPackaging(ctx context.Context, niin string) (details.Packaging, error) {
	return service.ItemDetailedServiceRepository.GetPackaging(ctx, niin)
}

func (service *ItemDetailedServiceImpl) GetCharacteristics(ctx context.Context, niin string) (details.Characteristics, error) {
	return service.ItemDetailedServiceRepository.GetCharacteristics(ctx, niin)
}

func (service *ItemDetailedServiceImpl) GetDisposition(ctx context.Context, niin string) (details.Disposition, error) {
	return service.ItemDetailedServiceRepository.GetDisposition(ctx, niin)
}
