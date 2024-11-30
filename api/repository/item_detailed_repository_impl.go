package repository

import (
	"context"
	"miltechserver/model/details"
	"miltechserver/prisma/db"
)

type ItemDetailedRepositoryImpl struct {
	Db *db.PrismaClient
}

func NewItemDetailedRepositoryImpl(db *db.PrismaClient) *ItemDetailedRepositoryImpl {
	return &ItemDetailedRepositoryImpl{Db: db}
}

func (repo *ItemDetailedRepositoryImpl) GetAmdfData(ctx context.Context, niin string) (details.Amdf, error) {
	data, err := repo.Db.ArmyMasterDataFile.FindFirst(db.ArmyMasterDataFile.Niin.Equals(niin)).Exec(ctx)
	if err != nil {
		return details.Amdf{}, err
	}
	return details.Amdf{
		// Map fields from data to details.Amdf
	}, nil
}

func (repo *ItemDetailedRepositoryImpl) GetArmyPackagingAndFreight(ctx context.Context, niin string) (details.ArmyPackagingAndFreight, error) {
	data, err := repo.Db.ArmyPackagingAndFreight.FindFirst(db.ArmyPackagingAndFreight.Niin.Equals(niin)).Exec(ctx)
	if err != nil {
		return details.ArmyPackagingAndFreight{}, err
	}
	return details.ArmyPackagingAndFreight{
		// Map fields from data to details.ArmyPackagingAndFreight
	}, nil
}

func (repo *ItemDetailedRepositoryImpl) GetSarsscat(ctx context.Context, niin string) (details.Sarsscat, error) {
	data, err := repo.Db.Sarsscat.FindFirst(db.Sarsscat.Niin.Equals(niin)).Exec(ctx)
	if err != nil {
		return details.Sarsscat{}, err
	}
	return details.Sarsscat{
		// Map fields from data to details.Sarsscat
	}, nil
}

func (repo *ItemDetailedRepositoryImpl) GetIdentification(ctx context.Context, niin string) (details.Identification, error) {
	data, err := repo.Db.Identification.FindFirst(db.Identification.Niin.Equals(niin)).Exec(ctx)
	if err != nil {
		return details.Identification{}, err
	}
	return details.Identification{
		// Map fields from data to details.Identification
	}, nil
}

func (repo *ItemDetailedRepositoryImpl) GetManagement(ctx context.Context, niin string) (details.Management, error) {
	data, err := repo.Db.Management.FindFirst(db.Management.Niin.Equals(niin)).Exec(ctx)
	if err != nil {
		return details.Management{}, err
	}
	return details.Management{
		// Map fields from data to details.Management
	}, nil
}

func (repo *ItemDetailedRepositoryImpl) GetReference(ctx context.Context, niin string) (details.Reference, error) {
	data, err := repo.Db.Reference.FindFirst(db.Reference.Niin.Equals(niin)).Exec(ctx)
	if err != nil {
		return details.Reference{}, err
	}
	return details.Reference{
		// Map fields from data to details.Reference
	}, nil
}

func (repo *ItemDetailedRepositoryImpl) GetFreight(ctx context.Context, niin string) (details.Freight, error) {
	data, err := repo.Db.Freight.FindFirst(db.Freight.Niin.Equals(niin)).Exec(ctx)
	if err != nil {
		return details.Freight{}, err
	}
	return details.Freight{
		// Map fields from data to details.Freight
	}, nil
}

func (repo *ItemDetailedRepositoryImpl) GetPackaging(ctx context.Context, niin string) (details.Packaging, error) {
	data, err := repo.Db.Packaging.FindFirst(db.Packaging.Niin.Equals(niin)).Exec(ctx)
	if err != nil {
		return details.Packaging{}, err
	}
	return details.Packaging{
		// Map fields from data to details.Packaging
	}, nil
}

func (repo *ItemDetailedRepositoryImpl) GetCharacteristics(ctx context.Context, niin string) (details.Characteristics, error) {
	data, err := repo.Db.Characteristics.FindFirst(db.Characteristics.Niin.Equals(niin)).Exec(ctx)
	if err != nil {
		return details.Characteristics{}, err
	}
	return details.Characteristics{
		// Map fields from data to details.Characteristics
	}, nil
}

func (repo *ItemDetailedRepositoryImpl) GetDisposition(ctx context.Context, niin string) (details.Disposition, error) {
	data, err := repo.Db.Disposition.FindFirst(db.Disposition.Niin.Equals(niin)).Exec(ctx)
	if err != nil {
		return details.Disposition{}, err
	}
	return details.Disposition{
		// Map fields from data to details.Disposition
	}, nil
}
