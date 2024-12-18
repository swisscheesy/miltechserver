package service

import (
	"context"
	"miltechserver/model"
)

type ItemDetailedService interface {
	FindDetailedItem(ctx context.Context, niin string) (model.DetailedItem, error)

	//GetAmdfData(ctx context.Context, niin string) (details.Amdf, error)
	//GetArmyPackagingAndFreight(ctx context.Context, niin string) (details.ArmyPackagingAndFreight, error)
	//GetSarsscat(ctx context.Context, niin string) (details.Sarsscat, error)
	//GetIdentification(ctx context.Context, niin string) (details.Identification, error)
	//GetManagement(ctx context.Context, niin string) (details.Management, error)
	//GetReference(ctx context.Context, niin string) (details.Reference, error)
	//GetFreight(ctx context.Context, niin string) (details.Freight, error)
	//GetPackaging(ctx context.Context, niin string) (details.Packaging, error)
	//GetCharacteristics(ctx context.Context, niin string) (details.Characteristics, error)
	//GetDisposition(ctx context.Context, niin string) (details.Disposition, error)
}
